package goss

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	osExec "os/exec"
	"strconv"
	"strings"

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
			"--address", fmt.Sprintf("/proc/%d/root/run/containerd/containerd.sock", childPid),
		}
		nsenterArgs = append(nsenterArgs, ctrArgs...)

		return shell.ExecuteWithWriter(output, "nsenter", nsenterArgs...)
	}

	if _, exists := os.LookupEnv("CONTAINERD_ADDRESS"); !exists {
		socket, err := GetContainerdRootfullSocket()
		if err != nil {
			return err
		}
		ctrArgs = append([]string{"--address", socket}, ctrArgs...)
	}

	return shell.ExecuteWithWriter(output, "ctr", ctrArgs...)
}

func gossBinary() (string, error) {
	return osExec.LookPath("goss")
}

// GetContainerdRootfullSocket returns the path to the containerd rootfull socket.
// It checks the default socket path /run/containerd/containerd.sock.
// If the socket is found, it runs ctr info with this socket address and compares the UID
// with the UID of the worker returned by buildctl debug workers.
// We may have multiple containerd instances, so we need to point to the containerd that is the buildkit worker.
// If the socket is not found, it returns an error.
func GetContainerdRootfullSocket() (string, error) {
	// Check the default socket path based on
	// https://github.com/containerd/nerdctl/blob/main/docs/faq.md#containerd-socket-address
	defaultSocket := "/run/containerd/containerd.sock"
	if _, err := os.Stat(defaultSocket); err != nil {
		return "", fmt.Errorf("containerd socket not found at %s: %w", defaultSocket, err)
	}

	// Run ctr info with the default socket address
	ctrCmd := osExec.Command("ctr", "--address", defaultSocket, "info")
	ctrOutput, err := ctrCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run ctr info: %w", err)
	}

	// Parse the JSON output to get the UID
	var ctrInfo map[string]interface{}
	if err := json.Unmarshal(ctrOutput, &ctrInfo); err != nil {
		return "", fmt.Errorf("failed to parse ctr info output: %w", err)
	}

	// Extract the UID from the ctr info output
	ctrUID, ok := ctrInfo["uid"].(float64)
	if !ok {
		return "", errors.New("failed to get UID from ctr info output")
	}

	buildctl, err := buildkit.BuildctlBinary()
	if err != nil {
		return "", err
	}
	// Run buildctl debug workers to get the worker UID
	buildctlCmd := osExec.Command(buildctl, "debug", "workers", "--format={{json .}}") //nolint:gosec
	buildctlOutput, err := buildctlCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run buildctl debug workers: %w", err)
	}

	// Parse the JSON output to get the worker UID
	var workers []map[string]interface{}
	if err := json.Unmarshal(buildctlOutput, &workers); err != nil {
		return "", fmt.Errorf("failed to parse buildctl workers output: %w", err)
	}

	if len(workers) == 0 {
		return "", errors.New("no buildkit workers found")
	}

	// Extract the worker UID
	workerInfo, ok := workers[0]["info"].(map[string]interface{})
	if !ok {
		return "", errors.New("worker info not found or invalid format")
	}

	workerUID, ok := workerInfo["uid"].(float64)
	if !ok {
		// Try to parse it as a string
		uidStr, ok := workerInfo["uid"].(string)
		if !ok {
			return "", errors.New("worker UID not found or invalid format")
		}
		uid, err := strconv.ParseFloat(uidStr, 64)
		if err != nil {
			return "", fmt.Errorf("failed to parse worker UID: %w", err)
		}
		workerUID = uid
	}

	// Compare the UIDs
	if int(ctrUID) != int(workerUID) {
		return "", fmt.Errorf("containerd UID (%d) does not match worker UID (%d)", int(ctrUID), int(workerUID))
	}

	return defaultSocket, nil
}
