//nolint:paralleltest
package dib_test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	buildPath1     = "../../test/fixtures/docker"
	buildPath2     = "../../test/fixtures/docker-duplicates"
	registryPrefix = "eu.gcr.io/my-test-repository"
)

func TestGenerateDAG(t *testing.T) {
	baseDir := path.Join(buildPath1, "root1")
	root1Files := []string{
		path.Join(baseDir, "Dockerfile"),
		path.Join(baseDir, "custom-hash-list", "Dockerfile"),
		path.Join(baseDir, "dockerignore", ".dockerignore"),
		path.Join(baseDir, "dockerignore", "Dockerfile"),
		path.Join(baseDir, "dockerignore", "ignored.txt"),
		path.Join(baseDir, "multistage", "Dockerfile"),
		path.Join(baseDir, "skipbuild", "Dockerfile"),
		path.Join(baseDir, "with-a-file", "Dockerfile"),
		path.Join(baseDir, "with-a-file", "included.txt"),
	}
	root1Hash, err := dib.HashFiles(baseDir, root1Files, nil, nil)
	require.NoError(t, err)

	t.Run("basic tests", func(t *testing.T) {
		buildPath := copyFixtures(t, buildPath1)

		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		nodes := flattenNodes(graph)
		rootNode := nodes["root1"]
		multistageNode := nodes["multistage"]

		rootImage := rootNode.Image
		assert.Equal(t, path.Join(registryPrefix, "root1"), rootImage.Name)
		assert.Equal(t, "root1", rootImage.ShortName)
		assert.Empty(t, rootNode.Parents())
		assert.Len(t, rootNode.Children(), 4)
		assert.Len(t, multistageNode.Parents(), 1)
	})

	t.Run("modifying root1 node", func(t *testing.T) {
		buildPath := copyFixtures(t, buildPath1)

		graph0, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		nodes0 := flattenNodes(graph0)
		rootNode0 := nodes0["root1"]
		childNode0 := nodes0["with-a-file"]

		// When I add a new file in the root1 folder
		require.NoError(t, os.WriteFile( //nolint:gosec
			path.Join(buildPath, "root1/newfile"),
			[]byte("any content"),
			os.ModePerm))

		// Then all the hashes of root1 and its children should have changed
		graph1, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		nodes1 := flattenNodes(graph1)
		rootNode1 := nodes1["root1"]
		childNode1 := nodes1["with-a-file"]

		assert.NotEqual(t, rootNode0.Image.Hash, rootNode1.Image.Hash)
		assert.NotEqual(t, childNode0.Image.Hash, childNode1.Image.Hash)
	})

	t.Run("modifying a leaf node", func(t *testing.T) {
		buildPath := copyFixtures(t, buildPath1)

		graph0, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		nodes0 := flattenNodes(graph0)
		rootNode0 := nodes0["root1"]
		subNode0 := nodes0["with-a-file"]
		multistageNode0 := nodes0["multistage"]

		// When I add a new file in the root1/multistage folder (leaf node)
		require.NoError(t, os.WriteFile( //nolint:gosec
			path.Join(buildPath, "root1/multistage/newfile"),
			[]byte("file contents"),
			os.ModePerm))

		// Then ONLY the hash of the leaf node root1/multistage should have changed
		graph1, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		nodes1 := flattenNodes(graph1)
		rootNode1 := nodes1["root1"]
		subNode1 := nodes1["with-a-file"]
		multistageNode1 := nodes1["multistage"]

		assert.Equal(t, rootNode0.Image.Hash, rootNode1.Image.Hash)
		assert.Equal(t, subNode0.Image.Hash, subNode1.Image.Hash)
		assert.NotEqual(t, multistageNode0.Image.Hash, multistageNode1.Image.Hash)
	})

	t.Run("using custom hash list", func(t *testing.T) {
		graph0, err := dib.GenerateDAG(buildPath1, registryPrefix, "", nil)
		require.NoError(t, err)

		graph1, err := dib.GenerateDAG(buildPath1, registryPrefix,
			"../../test/fixtures/dib/valid_wordlist.txt", nil)
		require.NoError(t, err)

		nodes0 := flattenNodes(graph0)
		rootNode0 := nodes0["root1"]
		customNode0 := nodes0["custom-hash-list"]
		nodes1 := flattenNodes(graph1)
		rootNode1 := nodes1["root1"]
		customNode1 := nodes1["custom-hash-list"]

		// Recompute hash of custom-hash-list, which is the only node that has the label 'dib.use-custom-hash-list'
		baseDir = path.Join(buildPath1, "root1", "custom-hash-list")
		subImageFiles := []string{
			path.Join(baseDir, "Dockerfile"),
		}

		oldHash, err := dib.HashFiles(baseDir, subImageFiles, []string{root1Hash}, nil)
		require.NoError(t, err)

		customHashListPath := "../../test/fixtures/dib/valid_wordlist.txt"
		customList, err := dib.LoadCustomHashList(customHashListPath)
		require.NoError(t, err)
		newHash, err := dib.HashFiles(baseDir, subImageFiles, []string{root1Hash}, customList)
		require.NoError(t, err)

		assert.Equal(t, rootNode1.Image.Hash, rootNode0.Image.Hash)
		assert.Equal(t, oldHash, customNode0.Image.Hash)
		assert.Equal(t, newHash, customNode1.Image.Hash)
		assert.NotEqual(t, customNode0.Image.Hash, customNode1.Image.Hash)
	})

	t.Run("changing build args", func(t *testing.T) {
		graph0, err := dib.GenerateDAG(buildPath1, registryPrefix, "", nil)
		require.NoError(t, err)

		graph1, err := dib.GenerateDAG(buildPath1, registryPrefix, "",
			map[string]string{
				"HELLO": "world",
			})
		require.NoError(t, err)

		nodes0 := flattenNodes(graph0)
		rootNode0 := nodes0["root1"]
		multistageNode0 := nodes0["multistage"]
		nodes1 := flattenNodes(graph1)
		rootNode1 := nodes1["root1"]
		multistageNode1 := nodes1["multistage"]

		assert.NotEqual(t, rootNode0.Image.Hash, rootNode1.Image.Hash)
		assert.NotEqual(t, multistageNode0.Image.Hash, multistageNode1.Image.Hash)
	})

	t.Run("duplicates", func(t *testing.T) {
		graph, err := dib.GenerateDAG(buildPath2, registryPrefix, "", nil)
		require.Error(t, err)
		require.Nil(t, graph)
		require.EqualError(t, err,
			fmt.Sprintf(
				"duplicate image name \"%s/duplicate\" found while reading file \"%s/root/duplicate2/Dockerfile\": previous file was \"%s/root/duplicate1/Dockerfile\"", //nolint:lll
				registryPrefix, buildPath2, buildPath2))
	})
}

// copyFixtures copies the buildPath directory into a temporary one to be free to edit files.
func copyFixtures(t *testing.T, buildPath string) string {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	src := path.Join(cwd, buildPath)
	dest := t.TempDir()
	cmd := exec.Command("cp", "-r", src, dest)
	require.NoError(t, cmd.Run())
	return dest + "/dag"
}

func flattenNodes(graph *dag.DAG) map[string]*dag.Node {
	flatNodes := map[string]*dag.Node{}

	graph.Walk(func(node *dag.Node) {
		flatNodes[node.Image.ShortName] = node
	})

	return flatNodes
}

func TestLoadCustomHashList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       string
		expected    []string
		expectedErr string
	}{
		{
			name:        "standard wordlist",
			input:       "",
			expected:    nil,
			expectedErr: "",
		},
		{
			name:        "custom wordlist txt",
			input:       "../../test/fixtures/dib/wordlist.txt",
			expected:    []string{"a", "b", "c"},
			expectedErr: "",
		},
		{
			name:        "custom wordlist yml",
			input:       "../../test/fixtures/dib/wordlist.yml",
			expected:    []string{"e", "f", "g"},
			expectedErr: "",
		},
		{
			name:        "wordlist file not exist",
			input:       "../../test/fixtures/dib/lorem.txt",
			expected:    nil,
			expectedErr: "open ../../test/fixtures/dib/lorem.txt: no such file or directory",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual, err := dib.LoadCustomHashList(test.input)
			if test.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.expectedErr)
			}
			assert.Equal(t, test.expected, actual)
		})
	}
}
