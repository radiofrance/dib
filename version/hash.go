package version

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/wolfeidau/humanhash"
)

func HashFiles(files []string, parentsHash []string) (string, error) {
	hash := sha256.New()
	files = append([]string(nil), files...)
	sort.Strings(files)
	for _, file := range files {
		if strings.Contains(file, "\n") {
			return "", errors.New("filenames with newlines are not supported")
		}
		readCloser, err := os.Open(file)
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

	for _, parentHash := range parentsHash {
		hash.Write([]byte(parentHash))
	}

	humanReadableHash, err := humanhash.Humanize(hash.Sum(nil), 4)
	if err != nil {
		return "", fmt.Errorf("could not humanize hash: %w", err)
	}
	return humanReadableHash, nil
}
