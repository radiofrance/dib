package version

import (
	"fmt"
	"path"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"

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

	lastChangedDockerVersionHash, err := getLastChangedDockerVersion(repo)
	if err != nil {
		return "", nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return "", nil, err
	}

	versions := fmt.Sprintf("%s..%s", head.Hash().String(), lastChangedDockerVersionHash.String())
	out, err := exec.Execute("git", "diff", versions, "--name-only")
	if err != nil {
		return "", nil, err
	}

	diffs := strings.Split(strings.TrimSuffix(out, "\n"), "\n")

	var fullPathDiffs []string
	for _, filename := range diffs {
		fullPathDiffs = append(fullPathDiffs, path.Join(repositoryPath, filename))
	}

	dockerVersionContent, err := getDockerVersionContentForHash(repo, lastChangedDockerVersionHash)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get %s content for hash %s",
			dockerVersionFilename, lastChangedDockerVersionHash.String())
	}

	return dockerVersionContent, fullPathDiffs, nil
}

func getDockerVersionContentForHash(repo *git.Repository, lastChangedDockerVersionHash plumbing.Hash) (string, error) {
	commitObject, err := repo.CommitObject(lastChangedDockerVersionHash)
	if err != nil {
		return "", err
	}
	tree, err := commitObject.Tree()
	if err != nil {
		return "", err
	}
	file, err := tree.File(dockerVersionFilename)
	if err != nil {
		return "", err
	}
	content, err := file.Contents()
	if err != nil {
		return "", err
	}
	return content, nil
}

func getLastChangedDockerVersion(repository *git.Repository) (plumbing.Hash, error) {
	filename := dockerVersionFilename
	commitLog, err := repository.Log(&git.LogOptions{
		FileName: &filename,
	})
	if err != nil {
		return plumbing.Hash{}, err
	}

	_, err = commitLog.Next() // We skip Next, which is the commit where it was last modified. We want the one before
	if err != nil {
		return plumbing.Hash{}, err
	}
	commit, err := commitLog.Next()
	if err != nil {
		return plumbing.Hash{}, err
	}

	return commit.ID(), nil
}
