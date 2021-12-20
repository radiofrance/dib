package version

import (
	"fmt"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/radiofrance/dib/exec"
)

// GetDiffSinceLastDockerVersionChange computes the diff since last version.
// The last version is the git revision hash of the .docker-version file.
// It returns the hash of the compared revision, and the list of modified files.
func GetDiffSinceLastDockerVersionChange(repositoryPath string, exec exec.Executor) (string, []string, error) {
	repo, err := git.PlainOpen(repositoryPath)
	if err != nil {
		return "", nil, err
	}

	lastChangedDockerVersion, err := getLastChangedDockerVersion(repo)
	if err != nil {
		return "", nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return lastChangedDockerVersion, nil, err
	}

	versions := fmt.Sprintf("%s..%s", head.Hash().String(), lastChangedDockerVersion)
	out, err := exec.Execute("git", "diff", versions, "--name-only")
	if err != nil {
		return lastChangedDockerVersion, nil, err
	}

	diffs := strings.Split(strings.TrimSuffix(out, "\n"), "\n")

	var fullPathDiffs []string
	for _, filename := range diffs {
		fullPathDiffs = append(fullPathDiffs, path.Join(repositoryPath, filename))
	}

	return lastChangedDockerVersion, fullPathDiffs, nil
}

func getLastChangedDockerVersion(repository *git.Repository) (string, error) {
	filename := dockerVersionFilename
	commitLog, err := repository.Log(&git.LogOptions{
		FileName: &filename,
	})
	if err != nil {
		return "", err
	}

	_, err = commitLog.Next() // We skip Next, which is the commit where it was last modified. We want the one before
	if err != nil {
		return "", err
	}
	commit, err := commitLog.Next()
	if err != nil {
		return "", err
	}

	return commit.ID().String(), nil
}
