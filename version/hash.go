package version

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/wolfeidau/humanhash"
	"golang.org/x/mod/sumdb/dirhash"
)

const dockerIgnoreFileName = ".dockerignore"

// GetDockerVersionHash returns the revision hash of the build directory.
func GetDockerVersionHash(buildPath string) (string, error) {
	return dirhash.HashDir(buildPath, "", humanReadableHashFn)
}

func humanReadableHashFn(files []string, open func(string) (io.ReadCloser, error)) (string, error) {
	hash := sha256.New()
	files = append([]string(nil), files...)
	sort.Strings(files)
	for _, file := range files {
		if strings.Contains(file, "\n") {
			return "", errors.New("dirhash: filenames with newlines are not supported")
		}
		if file == DockerVersionFilename || path.Base(file) == dockerIgnoreFileName {
			// During the hash process, we ignore
			// - the hash file itself
			// .dockerignore files
			continue
		}
		readCloser, err := open(file)
		if err != nil {
			return "", err
		}
		hashFile := sha256.New()
		_, err = io.Copy(hashFile, readCloser)
		readCloser.Close()
		if err != nil {
			return "", err
		}
		fmt.Fprintf(hash, "%x  %s\n", hashFile.Sum(nil), file)
	}
	humanReadableHash, err := humanhash.Humanize(hash.Sum(nil), 4)
	if err != nil {
		return "", fmt.Errorf("could not humanize hash: %w", err)
	}
	return humanReadableHash, nil
}
