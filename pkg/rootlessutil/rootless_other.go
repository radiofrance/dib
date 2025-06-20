//go:build !linux

package rootlessutil

import (
	"errors"
)

func RootlessKitStateDir() (string, error) {
	return "", errors.New("unsupported platform")
}

func RootlessKitChildPid(_ string) (int, error) {
	return 0, errors.New("unsupported platform")
}
