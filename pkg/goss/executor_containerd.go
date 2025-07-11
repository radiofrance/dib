package goss

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	osExec "os/exec"
	"strings"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/buildkit"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/rootlessutil"
	"github.com/radiofrance/dib/pkg/types"
)

const defaultShell = "/bin/bash"

// ContainerdGossExecutor executes goss tests using containerd via ctr.
type ContainerdGossExecutor struct {
	Shell string
}

// NewContainerdGossExecutor creates a new instance of ContainerdGossExecutor.
func NewContainerdGossExecutor() *ContainerdGossExecutor {
	shell, exists := os.LookupEnv("SHELL")
	if !exists {
		shell = defaultShell
	}

	return &ContainerdGossExecutor{
		Shell: shell,
	}
}

// Execute goss tests on the given image using ctr. goss.yaml file is expected to be present in the given path.
func (e ContainerdGossExecutor) Execute(
	_ context.Context,
	output io.Writer,
	opts types.RunTestOptions,
	args ...string,
) error {
	shell := &exec.ShellExecutor{
		Dir: opts.DockerContextPath,
		Env: append(os.Environ(), fmt.Sprintf("GOSS_OPTS=%s", strings.Join(args, " "))),
	}

	gossBinary, err := gossBinary()
	if err != nil {
		return err
	}

	// Prepare arguments for ctr command
	ctrArgs := []string{
		"run",
		"--cgroup", "user.slice:foo:bar",
		"--runc-systemd-cgroup",
		"--rm",
		"--mount", fmt.Sprintf("type=bind,src=%s,dst=/usr/local/bin/goss,options=rbind:ro", gossBinary),
		"--mount", fmt.Sprintf("type=bind,src=%s,dst=/goss,options=rbind:ro", opts.DockerContextPath),
		opts.ImageReference,
		"goss-test",
		"sh", "-c", fmt.Sprintf("cd /goss && goss validate %s", strings.Join(args, " ")),
	}

	// Execute nsenter for rootless mode or ctr for rootfull mode
	if rootlessutil.IsRootless() {
		stateDir, err := rootlessutil.RootlessKitStateDir()
		if err != nil {
			return err
		}
		childPid, err := rootlessutil.RootlessKitChildPid(stateDir)
		if err != nil {
			return err
		}

		// Check if containerd socket matches buildkit worker for rootless mode
		containerdSocket := fmt.Sprintf("/proc/%d/root/run/containerd/containerd.sock", childPid)
		match, err := IsContainerdSocketMatchingBuildkitWorker(
			opts.BuildkitHost,
			containerdSocket,
			&childPid,
		)
		if err != nil {
			return err
		}
		if !match {
			return fmt.Errorf("rootless containerd server UUID does not match the buildkit worker containerd UUID")
		}

		// ctr needs to be executed inside the daemon namespaces (Rootlesskit namespace)
		// https://github.com/containerd/containerd/blob/main/docs/rootless.md#client
		// we may need to re-associate the network namespace if detached-netns mode is enabled
		nsenterArgs := []string{
			"-U",
			"--preserve-credentials",
			"-m",
			"-t", fmt.Sprintf("%d", childPid),
			"--",
			"ctr",
			// sock address of rootless containerd based on
			// https://github.com/containerd/nerdctl/blob/main/docs/faq.md#containerd-socket-address
			"--address", containerdSocket,
		}
		nsenterArgs = append(nsenterArgs, ctrArgs...)

		return shell.ExecuteWithWriter(output, "nsenter", nsenterArgs...)
	}

	containerdSocket := "/run/containerd/containerd.sock"
	if _, exists := os.LookupEnv("CONTAINERD_ADDRESS"); exists {
		containerdSocket = os.Getenv("CONTAINERD_ADDRESS")
	}

	if match, err := IsContainerdSocketMatchingBuildkitWorker(opts.BuildkitHost, containerdSocket, nil); err != nil {
		if !match {
			return fmt.Errorf("containerd server UUID does not match the buildkit worker containerd UUID")
		}
	} else {
		return err
	}

	ctrArgs = append([]string{"--address", containerdSocket}, ctrArgs...)

	return shell.ExecuteWithWriter(output, "ctr", ctrArgs...)
}

func gossBinary() (string, error) {
	return osExec.LookPath("goss")
}

// IsContainerdSocketMatchingBuildkitWorker checks if the given containerd socket matches the UUID
// of the default buildkit worker.
// It compares the server UUID from `ctr info` with the UUID from the default worker in
// `buildctl debug workers`.
// Returns true if the UUIDs match, false otherwise.
func IsContainerdSocketMatchingBuildkitWorker(buildkitHost, containerdSocket string, childPid *int) (bool, error) {
	if _, err := os.Stat(containerdSocket); err != nil {
		return false, fmt.Errorf("containerd socket not found at %s: %w", containerdSocket, err)
	}

	var ctrOutput []byte
	var err error

	ctrCmd := osExec.Command("ctr", "--address", containerdSocket, "info")
	ctrOutput, err = ctrCmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to run ctr info: %w", err)
	}

	// Parse the JSON output to get the server UUID
	var ctrInfo map[string]interface{}
	if err := json.Unmarshal(ctrOutput, &ctrInfo); err != nil {
		return false, fmt.Errorf("failed to parse ctr info output: %w", err)
	}

	serverInfo, ok := ctrInfo["server"].(map[string]interface{})
	if !ok {
		return false, errors.New("failed to get server info from ctr info output")
	}

	serverUUID, ok := serverInfo["uuid"].(string)
	if !ok {
		return false, errors.New("failed to get server UUID from ctr info output")
	}

	logger.Infof("Containerd server UUID: %s", serverUUID)

	buildctl, err := buildkit.BuildctlBinary()
	if err != nil {
		return false, err
	}

	// Run buildctl debug workers to get the worker UID
	args := []string{
		"--addr=" + buildkitHost,
		"debug",
		"workers",
		"--format={{json .}}",
	}
	buildctlCmd := osExec.Command(buildctl, args...) //nolint:gosec
	buildctlOutput, err := buildctlCmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to run buildctl debug workers: %w", err)
	}

	var workers []map[string]interface{}
	if err := json.Unmarshal(buildctlOutput, &workers); err != nil {
		return false, fmt.Errorf("failed to parse buildctl workers output: %w", err)
	}

	if len(workers) == 0 {
		return false, errors.New("no buildkit workers found")
	}

	// Check if the default worker (index 0) has a matching UUID
	labels, ok := workers[0]["labels"].(map[string]interface{})
	if !ok {
		return false, errors.New("worker info not found or invalid format")
	}

	workerUUIDStr, ok := labels["org.mobyproject.buildkit.worker.containerd.uuid"].(string)
	if !ok {
		return false, errors.New("invalid worker or UUID not found in labels")
	}

	logger.Infof("Buildkit worker containerd UUID: %s", workerUUIDStr)

	if serverUUID == workerUUIDStr {
		return true, nil
	}

	return false, nil
}
