package version

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/radiofrance/dib/types"

	"github.com/go-git/go-git/v5/plumbing"

	"github.com/go-git/go-git/v5"
	"github.com/radiofrance/dib/exec"
)

var ErrNoPreviousBuild = errors.New("no previous build was found in git history")

// GetDiffSinceLastDockerVersionChange computes the diff since last version.
// The last version is the git revision hash of the .docker-version file.
// It returns the hash of the compared revision, and the list of modified files.
func GetDiffSinceLastDockerVersionChange(repositoryPath string, exec exec.Executor,
	registry types.DockerRegistry, dockerVersionFile, referentialImage string,
) (string, []string, error) {
	repo, err := git.PlainOpen(repositoryPath)
	if err != nil {
		return "", nil, err
	}

	lastChangedDockerVersionHash, err := getLastChangedDockerVersion(repo, registry, dockerVersionFile, referentialImage)
	if err != nil {
		return "", nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return "", nil, err
	}

	if lastChangedDockerVersionHash == plumbing.ZeroHash {
		// No previous build was found
		return "", nil, nil
	}

	// We need to run git diff in both direction (CommitA..CommitB & CommitB..CommitA) to identify
	// files that have been renamed, and to keep both old name and new name
	versions := fmt.Sprintf("%s..%s", head.Hash().String(), lastChangedDockerVersionHash.String())
	out, err := exec.Execute("git", "diff", versions, "--name-only")
	if err != nil {
		return "", nil, err
	}
	diffs := strings.Split(strings.TrimSuffix(out, "\n"), "\n")

	versions = fmt.Sprintf("%s..%s", lastChangedDockerVersionHash.String(), head.Hash().String())
	out, err = exec.Execute("git", "diff", versions, "--name-only")
	if err != nil {
		return "", nil, err
	}
	diffs = append(diffs, strings.Split(strings.TrimSuffix(out, "\n"), "\n")...)

	var fullPathDiffs []string
	for _, filename := range diffs {
		fullPathDiffs = append(fullPathDiffs, path.Join(repositoryPath, filename))
	}

	dockerVersionContent, err := getDockerVersionContentForHash(repo, lastChangedDockerVersionHash, dockerVersionFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get %s content for hash %s",
			dockerVersionFile, lastChangedDockerVersionHash.String())
	}

	return strings.TrimSuffix(dockerVersionContent, "\n"), fullPathDiffs, nil
}

func getDockerVersionContentForHash(repo *git.Repository, lastChangedHash plumbing.Hash, dockerVersionFile string,
) (string, error) {
	commitObject, err := repo.CommitObject(lastChangedHash)
	if err != nil {
		return "", err
	}
	tree, err := commitObject.Tree()
	if err != nil {
		return "", err
	}
	file, err := tree.File(dockerVersionFile)
	if err != nil {
		return "", err
	}
	content, err := file.Contents()
	if err != nil {
		return "", err
	}
	return content, nil
}

func getLastChangedDockerVersion(repository *git.Repository, registry types.DockerRegistry,
	dockerVersionFile, referentialImage string,
) (plumbing.Hash, error) {
	commitLog, err := repository.Log(&git.LogOptions{
		PathFilter: func(p string) bool {
			return p == dockerVersionFile
		},
	})
	if err != nil {
		return plumbing.Hash{}, err
	}

	for {
		commit, err := commitLog.Next()
		if err != nil {
			return plumbing.Hash{}, err
		}
		commitID := commit.ID()
		dockerVersionContent, err := getDockerVersionContentForHash(repository, commitID, dockerVersionFile)
		if err != nil {
			if errors.Is(err, object.ErrFileNotFound) {
				// We consider we went as far as we could to find a matching commit that was already built
				return plumbing.Hash{}, ErrNoPreviousBuild
			}
			return plumbing.Hash{}, fmt.Errorf("failed to get %s content for hash %s", dockerVersionFile, commitID)
		}
		refExists, err := registry.RefExists(fmt.Sprintf("%s:%s", referentialImage, dockerVersionContent))
		if err != nil {
			return plumbing.Hash{}, err
		}
		if refExists {
			return commitID, nil
		}
	}
}
