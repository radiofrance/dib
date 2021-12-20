package version

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/radiofrance/dib/exec"
)

const dockerVersionFilename = ".docker-version"

// CheckDockerVersionIntegrity verifies the consistency of the version hash
// contained in the .docker-version file against the revision hash from git.
// It returns the version if the verification is successful.
func CheckDockerVersionIntegrity(rootPath string, exec exec.Executor) (string, error) {
	fileVersion, err := getDockerVersionFromFile(path.Join(rootPath, dockerVersionFilename))
	if err != nil {
		return "", err
	}

	dockerVersionHash, err := getDockerVersionHash(exec)
	if err != nil {
		return "", fmt.Errorf("could not obtain docker-version hash: %w", err)
	}

	if fileVersion != dockerVersionHash {
		return "", fmt.Errorf(
			"inconsistency between the content of .docker-version and the result of the hash command",
		)
	}

	return dockerVersionHash, nil
}

// getDockerVersionFromFile reads the current docker version contained in a file.
func getDockerVersionFromFile(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}
