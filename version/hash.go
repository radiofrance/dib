package version

import (
	"strings"

	"github.com/radiofrance/dib/exec"
)

// getDockerVersionHash returns the revision hash of the docker directory.
func getDockerVersionHash(exec exec.Executor) (string, error) {
	cmd := "find docker -type f -print0 | sort -z | xargs -0 sha1sum | sha1sum | cut -d ' ' -f 1"
	res, err := exec.Execute("bash", "-c", cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(res, "\n"), nil
}
