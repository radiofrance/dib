package preflight

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/radiofrance/dib/internal/logger"
)

func RunPreflightChecks(requiredCommands []string) {
	shouldSkipPreflightTest := os.Getenv("SKIP_PREFLIGHT_CHECKS")
	if len(shouldSkipPreflightTest) == 0 {
		logger.Infof("Running preflights checks...")
		for _, bin := range requiredCommands {
			err := isBinInstalled(bin)
			if err != nil {
				logger.Warnf("%s", err)
			}
		}
	}
}

// isBinInstalled checks if given binary exist on host system by checking exit code of the command "which xxx"
// If binary does not exist,or  exit code is != 0, it will return an error.
func isBinInstalled(bin string) error {
	cmd := exec.Command("which", bin)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return fmt.Errorf("unable to check if \"%s\" is installed, error: %w", bin, err)
		}

		if _, ok := exitErr.Sys().(syscall.WaitStatus); !ok {
			return fmt.Errorf("unable to check if \"%s\" is installed, error: %w", bin, err)
		}

		return fmt.Errorf("\"%s\" does not seem to be installed on your system, "+
			"you have to install it before using dib", bin)
	}
	return nil
}
