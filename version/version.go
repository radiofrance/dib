package version

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
)

const DockerVersionFilename = ".docker-version"

// CheckDockerVersionIntegrity verifies the consistency of the hash contained in the
// .docker-version file against the hash computed from the current filesystem state.
// It returns the version if the verification is successful.
func CheckDockerVersionIntegrity(buildPath string) (string, error) {
	fileVersion, err := getDockerVersionFromFile(path.Join(buildPath, DockerVersionFilename))
	if err != nil {
		return "", err
	}

	//	dockerVersionHash, err := GetDockerVersionHash(buildPath)
	dockerVersionHash, err := HashFiles([]string{buildPath}, []string{})
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
