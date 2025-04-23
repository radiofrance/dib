package rootlessutil

import (
	"os"
)

func IsRootless() bool {
	return os.Geteuid() != 0
}
