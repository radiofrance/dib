package dib_test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/radiofrance/dib/pkg/dag"
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

func TestGenerateDAG(t *testing.T) {
	t.Parallel()

	baseDir := path.Join(buildPath, "bullseye")
	bullseyeHash, err := dib.HashFiles(baseDir, []string{path.Join(baseDir, "Dockerfile")}, nil, nil)
	require.NoError(t, err)

	baseDir = path.Join(buildPath, "root")
	rootHash, err := dib.HashFiles(baseDir, []string{path.Join(baseDir, "Dockerfile")}, nil, nil)
	require.NoError(t, err)

	baseDir = path.Join(buildPath, "root", "root-as-well")
	rootAsWellHash, err := dib.HashFiles(baseDir, []string{path.Join(baseDir, "Dockerfile")}, nil, nil)
	require.NoError(t, err)

	t.Run("nominal", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		graph.Walk(func(node *dag.Node) {
			assert.Len(t, node.Files, 1, spew.Sdump("node should have only one Dockerfile", node.Files))
		})

		assert.Equal(t, fmt.Sprintf(
			`docker
├──┬bullseye [%s]
│  ├───kaniko [south-foxtrot-robert-vegan]
│  ├───multistage [purple-white-arizona-mars]
│  └───sub-image [two-violet-monkey-emma]
├───root [%s]
└───root-as-well [%s]
`, bullseyeHash, rootHash, rootAsWellHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("adding a file to bullseye directory", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		// When I add a new file in bullseye
		baseDir := path.Join(buildPath, "bullseye")
		require.NoError(t, os.WriteFile(
			path.Join(baseDir, "newfile"),
			[]byte("any content"),
			os.ModePerm))

		bullseyeHash, err := dib.HashFiles(baseDir, []string{
			path.Join(baseDir, "Dockerfile"),
			path.Join(baseDir, "newfile"),
		}, nil, nil)
		require.NoError(t, err)

		// Then all hashes of children and sub children of bullseye node should change
		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		graph.Walk(func(node *dag.Node) {
			if node.Image.ShortName == "bullseye" {
				assert.Len(t, node.Files, 2, spew.Sdump("node should have a Dockerfile and the new file", node.Files))
			} else {
				assert.Len(t, node.Files, 1, spew.Sdump("node should have only one Dockerfile", node.Files))
			}
		})

		assert.Equal(t, fmt.Sprintf(
			`docker
├──┬bullseye [%s]
│  ├───kaniko [floor-apart-rugby-burger]
│  ├───multistage [twelve-happy-london-fruit]
│  └───sub-image [ack-uniform-autumn-arkansas]
├───root [%s]
└───root-as-well [%s]
`, bullseyeHash, rootHash, rootAsWellHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("adding a file to multistage directory", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		// When I add a new file in bullseye/multistage
		baseDir := path.Join(buildPath, "bullseye")
		require.NoError(t, os.WriteFile(
			path.Join(baseDir, "multistage", "newfile"),
			[]byte("any content"),
			os.ModePerm))

		// Then ONLY the hash of the leaf node bullseye/multistage should have changed
		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		graph.Walk(func(node *dag.Node) {
			if node.Image.ShortName == "multistage" {
				assert.Len(t, node.Files, 2, spew.Sdump("node should have a Dockerfile and the new file", node.Files))
			} else {
				assert.Len(t, node.Files, 1, spew.Sdump("node should have only one Dockerfile", node.Files))
			}
		})

		assert.Equal(t, fmt.Sprintf(
			`docker
├──┬bullseye [%s]
│  ├───kaniko [south-foxtrot-robert-vegan]
│  ├───multistage [muppet-illinois-video-delaware]
│  └───sub-image [two-violet-monkey-emma]
├───root [%s]
└───root-as-well [%s]
`, bullseyeHash, rootHash, rootAsWellHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("using custom hash list", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		customPath := "../../test/fixtures/dib/valid_wordlist.txt"
		graph, err := dib.GenerateDAG(buildPath, registryPrefix,
			customPath, nil)
		require.NoError(t, err)

		graph.Walk(func(node *dag.Node) {
			assert.Len(t, node.Files, 1, spew.Sdump("node should have only one Dockerfile", node.Files))
		})

		// Only the sub-image node which has the label 'dib.use-custom-hash-list' should change
		assert.Equal(t, fmt.Sprintf(
			`docker
├──┬bullseye [%s]
│  ├───kaniko [south-foxtrot-robert-vegan]
│  ├───multistage [purple-white-arizona-mars]
│  └───sub-image [girafarig-golduck-doduo-breloom]
├───root [%s]
└───root-as-well [%s]
`, bullseyeHash, rootHash, rootAsWellHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("using build args", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		buildArgs := map[string]string{
			"HELLO": "world",
		}

		baseDir := path.Join(buildPath, "bullseye")
		dckfile, err := dockerfile.ParseDockerfile(path.Join(baseDir, "Dockerfile"))
		require.NoError(t, err)
		filename := path.Join(dckfile.ContextPath, dckfile.Filename)
		argInstructionsToReplace := make(map[string]string)
		for key, newArg := range buildArgs {
			prevArgInstruction, ok := dckfile.Args[key]
			if ok {
				argInstructionsToReplace[prevArgInstruction] = fmt.Sprintf("ARG %s=%s", key, newArg)
			}
		}
		require.NoError(t, dockerfile.ReplaceInFile(filename, argInstructionsToReplace))

		bullseyeHash, err := dib.HashFiles(baseDir,
			[]string{path.Join(baseDir, "Dockerfile")}, nil, nil)
		require.NoError(t, err)

		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", buildArgs)
		require.NoError(t, err)

		graph.Walk(func(node *dag.Node) {
			assert.Len(t, node.Files, 1, spew.Sdump("node should have only one Dockerfile", node.Files))
		})

		// Only bullseye node has the 'HELLO' argument, so its hash and all of its children should change
		assert.Equal(t, fmt.Sprintf(
			`docker
├──┬bullseye [%s]
│  ├───kaniko [xray-enemy-mississippi-nebraska]
│  ├───multistage [summer-nine-one-eighteen]
│  └───sub-image [skylark-hot-tennis-one]
├───root [%s]
└───root-as-well [%s]
`, bullseyeHash, rootHash, rootAsWellHash), graph.Sprint(path.Base(buildPath)))
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
