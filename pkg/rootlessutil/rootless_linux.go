package rootlessutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// https://specifications.freedesktop.org/basedir-spec/latest/
func XDGRuntimeDir() string {
	// XDG_RUNTIME_DIR is an environment variable specifying a user-specific directory for runtime files (e.g socket..)
	if xrd := os.Getenv("XDG_RUNTIME_DIR"); xrd != "" {
		return xrd
	}
	// Fall back to "/run/user/<euid>".
	return fmt.Sprintf("/run/user/%d", os.Geteuid())
}

func RootlessKitStateDir() (string, error) {
	if v := os.Getenv("ROOTLESSKIT_STATE_DIR"); v != "" {
		return v, nil
	}
	xdr := XDGRuntimeDir()

	// "${XDG_RUNTIME_DIR}/containerd-rootless" is hardcoded in containerd-rootless.sh
	// docker is deprecated from v0.25.0 but we keep it for backward compatibility.
	stateDir := filepath.Join(xdr, "containerd-rootless")

	if _, err := os.Stat(stateDir); err != nil {
		return "", err
	}
	return stateDir, nil
}

func RootlessKitChildPid(stateDir string) (int, error) {
	pidFilePath := filepath.Join(stateDir, "child_pid")
	if _, err := os.Stat(pidFilePath); err != nil {
		return 0, err
	}

	pidFileBytes, err := os.ReadFile(pidFilePath)
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(pidFileBytes))
	return strconv.Atoi(pidStr)
}
