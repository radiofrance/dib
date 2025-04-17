package rootlessutil

import (
	"fmt"
	"os"
)

func IsRootless() bool {
	return os.Geteuid() != 0
}

// https://specifications.freedesktop.org/basedir-spec/latest/
func XDGRuntimeDir() string {
	// XDG_RUNTIME_DIR is an environment variable specifying a user-specific directory for runtime files (e.g socket..)
	if xrd := os.Getenv("XDG_RUNTIME_DIR"); xrd != "" {
		return xrd
	}
	// Fall back to "/run/user/<euid>".
	return fmt.Sprintf("/run/user/%d", os.Geteuid())
}
