package buildkit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/rootlessutil"
)

const (
	// defaultDockerfileName is the Default filename, read by dib build.
	defaultDockerfileName string = "Dockerfile"
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
	if out, err := pingBKDaemon(buildkitHost); err != nil {
		if out != "" {
			logger.Errorf("%s", out)
		}
		return fmt.Errorf(getHint()+": %w", err)
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
	buildctlCheckCmd := exec.Command(buildctlBinary, args...) //nolint:gosec
	buildctlCheckCmd.Env = os.Environ()
	if out, err := buildctlCheckCmd.CombinedOutput(); err != nil {
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
	if _, err := os.Lstat(filepath.Join(absDir, file)); err != nil {
		return "", "", err
	}
	return absDir, file, nil
}
