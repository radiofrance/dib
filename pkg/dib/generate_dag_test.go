package dib_test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	buildPath           = "../../test/fixtures/docker"
	buildPathDuplicates = "../../test/fixtures/docker-duplicates"
	registryPrefix      = "eu.gcr.io/my-test-repository"
)

func TestGenerateDAG(t *testing.T) { //nolint:maintidx
	t.Parallel()

	baseDir := path.Join(buildPath, "bullseye")
	bullseyeFiles := []string{
		path.Join(baseDir, "Dockerfile"),
		path.Join(baseDir, "external-parent", "Dockerfile"),
		path.Join(baseDir, "multistage", "Dockerfile"),
		path.Join(baseDir, "multistage", "included.txt"),
		path.Join(baseDir, "skipbuild", "Dockerfile"),
		path.Join(baseDir, "sub-image", "Dockerfile"),
		path.Join(baseDir, "sub-image", "ignored.txt"),
	}
	bullseyeHash, err := dib.HashFiles(baseDir, bullseyeFiles, nil, nil)
	require.NoError(t, err)

	baseDir = path.Join(buildPath, "bullseye", "external-parent")
	extParentsFiles := []string{
		path.Join(baseDir, "Dockerfile"),
	}
	extParentHash, err := dib.HashFiles(baseDir, extParentsFiles, []string{bullseyeHash}, nil)
	require.NoError(t, err)

	baseDir = path.Join(buildPath, "bullseye", "multistage")
	multistageFiles := []string{
		path.Join(baseDir, "Dockerfile"),
		path.Join(baseDir, "included.txt"),
	}
	multistageHash, err := dib.HashFiles(baseDir, multistageFiles, []string{bullseyeHash}, nil)
	require.NoError(t, err)

	baseDir = path.Join(buildPath, "bullseye", "sub-image")
	subImageFiles := []string{
		path.Join(baseDir, "Dockerfile"),
	}
	subImageHash, err := dib.HashFiles(baseDir, subImageFiles, []string{bullseyeHash}, nil)
	require.NoError(t, err)

	baseDir = path.Join(buildPath, "root")
	rootFiles := []string{
		path.Join(baseDir, "Dockerfile"),
		path.Join(baseDir, "root-as-well", "Dockerfile"),
	}
	rootHash, err := dib.HashFiles(baseDir, rootFiles, nil, nil)
	require.NoError(t, err)

	baseDir = path.Join(buildPath, "root", "root-as-well")
	rootAsWellFiles := []string{
		path.Join(baseDir, "Dockerfile"),
	}
	rootAsWellHash, err := dib.HashFiles(baseDir, rootAsWellFiles, nil, nil)
	require.NoError(t, err)

	t.Run("nominal", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(
			`docker
├──┬bullseye [%s]
│  ├───external-parent [%s]
│  ├───multistage [%s]
│  └───sub-image [%s]
├───root [%s]
└───root-as-well [%s]
`, bullseyeHash, extParentHash, multistageHash, subImageHash, rootHash, rootAsWellHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("adding a file to bullseye directory", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		// When I add a new file in bullseye
		baseDir := path.Join(buildPath, "bullseye")
		newFilePath := path.Join(baseDir, "newfile")
		require.NoError(t, os.WriteFile(newFilePath, []byte("any content"), os.ModePerm))

		// Recompute hashes of bullseye and all its children because the context of bullseye has changed
		bullseyeFiles := []string{
			path.Join(baseDir, "Dockerfile"),
			path.Join(baseDir, "external-parent", "Dockerfile"),
			path.Join(baseDir, "multistage", "Dockerfile"),
			path.Join(baseDir, "multistage", "included.txt"),
			path.Join(baseDir, "skipbuild", "Dockerfile"),
			path.Join(baseDir, "sub-image", "Dockerfile"),
			path.Join(baseDir, "sub-image", "ignored.txt"),
			newFilePath,
		}
		bullseyeHash, err := dib.HashFiles(baseDir, bullseyeFiles, nil, nil)
		require.NoError(t, err)

		baseDir = path.Join(buildPath, "bullseye", "external-parent")
		extParentsFiles := []string{
			path.Join(baseDir, "Dockerfile"),
		}
		extParentHash, err := dib.HashFiles(baseDir, extParentsFiles, []string{bullseyeHash}, nil)
		require.NoError(t, err)

		baseDir = path.Join(buildPath, "bullseye", "multistage")
		multistageFiles := []string{
			path.Join(baseDir, "Dockerfile"),
			path.Join(baseDir, "included.txt"),
		}
		multistageHash, err := dib.HashFiles(baseDir, multistageFiles, []string{bullseyeHash}, nil)
		require.NoError(t, err)

		baseDir = path.Join(buildPath, "bullseye", "sub-image")
		subImageFiles := []string{
			path.Join(baseDir, "Dockerfile"),
		}
		subImageHash, err := dib.HashFiles(baseDir, subImageFiles, []string{bullseyeHash}, nil)
		require.NoError(t, err)

		// Then all hashes of children of bullseye node should change
		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(
			`docker
├──┬bullseye [%s]
│  ├───external-parent [%s]
│  ├───multistage [%s]
│  └───sub-image [%s]
├───root [%s]
└───root-as-well [%s]
`, bullseyeHash, extParentHash, multistageHash, subImageHash, rootHash, rootAsWellHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("adding a file to multistage directory", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		// When I add a new file in bullseye/multistage
		newFilePath := path.Join(buildPath, "bullseye", "multistage", "newfile")
		require.NoError(t, os.WriteFile(newFilePath, []byte("any content"), os.ModePerm))

		// Recompute hashes of bullseye and its children because the context of bullseye has changed
		baseDir := path.Join(buildPath, "bullseye")
		bullseyeFiles := []string{
			path.Join(baseDir, "Dockerfile"),
			path.Join(baseDir, "external-parent", "Dockerfile"),
			path.Join(baseDir, "multistage", "Dockerfile"),
			path.Join(baseDir, "multistage", "included.txt"),
			path.Join(baseDir, "skipbuild", "Dockerfile"),
			path.Join(baseDir, "sub-image", "Dockerfile"),
			path.Join(baseDir, "sub-image", "ignored.txt"),
			newFilePath,
		}
		bullseyeHash, err := dib.HashFiles(baseDir, bullseyeFiles, nil, nil)
		require.NoError(t, err)

		baseDir = path.Join(buildPath, "bullseye", "external-parent")
		extParentsFiles := []string{
			path.Join(baseDir, "Dockerfile"),
		}
		extParentHash, err := dib.HashFiles(baseDir, extParentsFiles, []string{bullseyeHash}, nil)
		require.NoError(t, err)

		baseDir = path.Join(buildPath, "bullseye", "multistage")
		multistageFiles := []string{
			path.Join(baseDir, "Dockerfile"),
			path.Join(baseDir, "included.txt"),
			newFilePath,
		}
		multistageHash, err := dib.HashFiles(baseDir, multistageFiles, []string{bullseyeHash}, nil)
		require.NoError(t, err)

		baseDir = path.Join(buildPath, "bullseye", "sub-image")
		subImageFiles := []string{
			path.Join(baseDir, "Dockerfile"),
		}
		subImageHash, err := dib.HashFiles(baseDir, subImageFiles, []string{bullseyeHash}, nil)
		require.NoError(t, err)

		// Then ONLY the hash of the leaf node bullseye/multistage should have changed
		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(
			`docker
├──┬bullseye [%s]
│  ├───external-parent [%s]
│  ├───multistage [%s]
│  └───sub-image [%s]
├───root [%s]
└───root-as-well [%s]
`, bullseyeHash, extParentHash, multistageHash, subImageHash, rootHash, rootAsWellHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("using custom hash list", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		customHashListPath := "../../test/fixtures/dib/valid_wordlist.txt"

		// Recompute hash of sub-image which is the only node that has the label 'dib.use-custom-hash-list'
		baseDir = path.Join(buildPath, "bullseye", "sub-image")
		subImageFiles := []string{
			path.Join(baseDir, "Dockerfile"),
		}
		list, err := dib.LoadCustomHashList(customHashListPath)
		require.NoError(t, err)
		subImageHash, err := dib.HashFiles(baseDir, subImageFiles, []string{bullseyeHash}, list)
		require.NoError(t, err)

		graph, err := dib.GenerateDAG(buildPath, registryPrefix, customHashListPath, nil)
		require.NoError(t, err)

		// Only the sub-image node which has the label 'dib.use-custom-hash-list' should change
		assert.Equal(t, fmt.Sprintf(
			`docker
├──┬bullseye [%s]
│  ├───external-parent [%s]
│  ├───multistage [%s]
│  └───sub-image [%s]
├───root [%s]
└───root-as-well [%s]
`, bullseyeHash, extParentHash, multistageHash, subImageHash, rootHash, rootAsWellHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("using build args", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		baseDir := path.Join(buildPath, "bullseye")
		dckfile, err := dockerfile.ParseDockerfile(path.Join(baseDir, "Dockerfile"))
		require.NoError(t, err)

		buildArgs := map[string]string{
			"HELLO": "world",
		}
		argInstructionsToReplace := make(map[string]string)
		for key, newArg := range buildArgs {
			prevArgInstruction, ok := dckfile.Args[key]
			if ok {
				argInstructionsToReplace[prevArgInstruction] = fmt.Sprintf("ARG %s=%s", key, newArg)
			}
		}
		require.NoError(t, dockerfile.ReplaceInFile(path.Join(baseDir, "Dockerfile"), argInstructionsToReplace))

		// Recompute hashes of bullseye and all its children because the Dockerfile of bullseye has changed
		bullseyeFiles := []string{
			path.Join(baseDir, "Dockerfile"),
			path.Join(baseDir, "external-parent", "Dockerfile"),
			path.Join(baseDir, "multistage", "Dockerfile"),
			path.Join(baseDir, "multistage", "included.txt"),
			path.Join(baseDir, "skipbuild", "Dockerfile"),
			path.Join(baseDir, "sub-image", "Dockerfile"),
			path.Join(baseDir, "sub-image", "ignored.txt"),
		}
		bullseyeHash, err := dib.HashFiles(baseDir, bullseyeFiles, nil, nil)
		require.NoError(t, err)

		baseDir = path.Join(buildPath, "bullseye", "external-parent")
		extParentsFiles := []string{
			path.Join(baseDir, "Dockerfile"),
		}
		extParentHash, err := dib.HashFiles(baseDir, extParentsFiles, []string{bullseyeHash}, nil)
		require.NoError(t, err)

		baseDir = path.Join(buildPath, "bullseye", "multistage")
		multistageFiles := []string{
			path.Join(baseDir, "Dockerfile"),
			path.Join(baseDir, "included.txt"),
		}
		multistageHash, err := dib.HashFiles(baseDir, multistageFiles, []string{bullseyeHash}, nil)
		require.NoError(t, err)

		baseDir = path.Join(buildPath, "bullseye", "sub-image")
		subImageFiles := []string{
			path.Join(baseDir, "Dockerfile"),
		}
		subImageHash, err := dib.HashFiles(baseDir, subImageFiles, []string{bullseyeHash}, nil)
		require.NoError(t, err)

		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", buildArgs)
		require.NoError(t, err)

		// Only bullseye node has the 'HELLO' argument, so its hash and all of its children should change
		assert.Equal(t, fmt.Sprintf(
			`docker
├──┬bullseye [%s]
│  ├───external-parent [%s]
│  ├───multistage [%s]
│  └───sub-image [%s]
├───root [%s]
└───root-as-well [%s]
`, bullseyeHash, extParentHash, multistageHash, subImageHash, rootHash, rootAsWellHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("duplicates image names", func(t *testing.T) {
		t.Parallel()

		_, err := dib.GenerateDAG(buildPathDuplicates, registryPrefix, "", nil)
		require.EqualError(t, err,
			fmt.Sprintf(
				"could not process Dockerfile \"%s/bullseye/duplicate2/Dockerfile\": duplicate image name \"%s/duplicate\" found: previous file was \"%s/bullseye/duplicate1/Dockerfile\"", //nolint:lll
				buildPathDuplicates, registryPrefix, buildPathDuplicates))
	})
}

// copyFixtures copies the buildPath directory into a temporary one to be free to edit files.
func copyFixtures(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	src := path.Join(cwd, buildPath)
	dest := t.TempDir()
	cmd := exec.Command("cp", "-r", src, dest)
	require.NoError(t, cmd.Run())
	return dest + "/docker"
}
