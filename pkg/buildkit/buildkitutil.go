package buildkit

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/executor"
	"github.com/radiofrance/dib/pkg/rootlessutil"
)

const (
	// defaultDockerfileName is the Default filename, read by dib build.
	defaultDockerfileName string = "Dockerfile"

	// RemoteUserId and RemoteGroupId represent the user ID and group ID that the Kubernetes container will run with.
	RemoteUserId  = 1000
	RemoteGroupId = 1000

	// OciExecutorType and containerdExecutorType represent executor types used in BuildKit worker configuration.
	OciExecutorType        = "oci"
	ContainerdExecutorType = "containerd"
)

func getHint() string {
	hint := "`buildctl` needs to be installed and `buildkitd` needs to be running, see https://github.com/moby/buildkit\n"
	if rootlessutil.IsRootless() {
		hint += "For rootless mode, use `rootlesskit buildkitd` (see https://github.com/rootless-containers/rootlesskit/)."
	}

	return hint
}

func BuildctlBinary() (string, error) {
	return exec.LookPath("buildctl")
}

func buildctlBaseArgs(buildkitHost string) []string {
	return []string{"--addr=" + buildkitHost}
}

func GetBuildkitHostAdress() (string, error) {
	path := getBuildkitHostAddress()

	logger.Debugf("ping the buildkit host %q", path)

	_, err := pingBKDaemon(path)
	if err == nil {
		logger.Debugf("buildkit host %q", path)
		return path, nil
	}

	logger.Errorf("failed to ping to host %s: %v", path, err)
	logger.Errorf("%s", getHint())

	return "", fmt.Errorf("no buildkit host is available: %w", err)
}

// PingBKDaemon checks if the buildkit daemon is running.
func PingBKDaemon(buildkitHost string) error {
	out, err := pingBKDaemon(buildkitHost)
	if err != nil {
		if out != "" {
			logger.Errorf("%s", out)
		}

		return fmt.Errorf("%s: %w", getHint(), err)
	}

	return nil
}

func pingBKDaemon(buildkitHost string) (string, error) {
	buildctlBinary, err := BuildctlBinary()
	if err != nil {
		return "", err
	}

	args := buildctlBaseArgs(buildkitHost)
	args = append(args, "debug", "workers")
	buildctlCheckCmd := exec.Command(buildctlBinary, args...) //nolint:gosec,noctx

	buildctlCheckCmd.Env = os.Environ()

	out, err := buildctlCheckCmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}

	return "", nil
}

// buildKitFile returns the values for the following buildctl args.
// --localfilename=dockerfile={absDir}.
// --opt=filename={file}.
func buildKitFile(dir, inputfile string) (string, string, error) {
	file := inputfile
	if file == "" || file == "." {
		file = defaultDockerfileName
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}

	_, err = os.Lstat(filepath.Join(absDir, file))
	if err != nil {
		return "", "", err
	}

	return absDir, file, nil
}

func GetRemoteBuildkitHostAddress(uid int) string {
	return "unix://" + filepath.Join("/run/user", fmt.Sprintf("%d", uid), "buildkit/buildkitd.sock")
}

// GetBuildkitWorkerType returns the type of buildkit worker (oci or containerd).
// TODO: Consider refactoring by introducing a helper function to extract labels,
// as this logic is used in multiple places.
func GetBuildkitWorkerType(buildctlBinary, buildkitHost string, shellExecutor executor.ShellExecutor) (string, error) {
	const (
		buildkitWorkerExecutorLabelKey = "org.mobyproject.buildkit.worker.executor"
	)

	args := buildctlBaseArgs(buildkitHost)
	args = append(args, "debug", "workers", "--format={{json .}}")

	out, err := shellExecutor.Execute(buildctlBinary, args...)
	if err != nil {
		return "", err
	}

	var workers []map[string]any

	err = json.Unmarshal([]byte(out), &workers)
	if err != nil {
		return "", fmt.Errorf("failed to parse buildkit workers output: %w", err)
	}

	if len(workers) == 0 {
		return "", fmt.Errorf("no buildkit workers found")
	}

	//nolint:lll
	// Extract the worker type from the first worker, as BuildKit can be configured to use a single worker (either oci or containerd).
	labels, ok := workers[0]["labels"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("worker labels not found or invalid format")
	}

	executorType, ok := labels[buildkitWorkerExecutorLabelKey].(string)
	if !ok {
		return "", fmt.Errorf("executor type not found or invalid format")
	}

	switch executorType {
	case OciExecutorType:
		return OciExecutorType, nil
	case ContainerdExecutorType:
		return ContainerdExecutorType, nil
	}

	return "", fmt.Errorf("unknown buildkit worker type: %s", executorType)
}
