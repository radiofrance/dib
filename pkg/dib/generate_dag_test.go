package dib_test

import (
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureDir = "../../test/fixtures/docker"

//nolint:paralleltest
func TestGenerateDAG(t *testing.T) {
	t.Run("basic tests", func(t *testing.T) {
		graph, err := dib.GenerateDAG(fixtureDir,
			"eu.gcr.io/my-test-repository", "", map[string]string{})
		require.NoError(t, err)

		nodes := flattenNodes(graph)
		rootNode := nodes["bullseye"]
		subNode := nodes["sub-image"]
		multistageNode := nodes["multistage"]

		rootImage := rootNode.Image
		assert.Equal(t, "eu.gcr.io/my-test-repository/bullseye", rootImage.Name)
		assert.Equal(t, "bullseye", rootImage.ShortName)
		assert.Empty(t, rootNode.Parents())
		assert.Len(t, rootNode.Children(), 3)
		assert.Len(t, subNode.Parents(), 1)
		assert.Len(t, multistageNode.Parents(), 1)
		assert.Equal(t, []string{"latest"}, multistageNode.Image.ExtraTags)
	})

	t.Run("modifying the root node should change all hashes", func(t *testing.T) {
		tmpDir := copyFixtures(t)

		graph0, err := dib.GenerateDAG(tmpDir,
			"eu.gcr.io/my-test-repository", "", map[string]string{})
		require.NoError(t, err)

		nodes0 := flattenNodes(graph0)
		rootNode0 := nodes0["bullseye"]
		subNode0 := nodes0["sub-image"]
		multistageNode0 := nodes0["multistage"]

		// When I add a new file in bullseye/ (root node)
		require.NoError(t, os.WriteFile(
			path.Join(tmpDir, "bullseye/newfile"),
			[]byte("any content"),
			os.ModePerm))

		// Then ONLY the hash of the child node bullseye/multistage should have changed
		graph1, err := dib.GenerateDAG(tmpDir,
			"eu.gcr.io/my-test-repository", "", map[string]string{})
		require.NoError(t, err)

		nodes1 := flattenNodes(graph1)
		rootNode1 := nodes1["bullseye"]
		subNode1 := nodes1["sub-image"]
		multistageNode1 := nodes1["multistage"]

		assert.NotEqual(t, rootNode0.Image.Hash, rootNode1.Image.Hash)
		assert.NotEqual(t, subNode0.Image.Hash, subNode1.Image.Hash)
		assert.NotEqual(t, multistageNode0.Image.Hash, multistageNode1.Image.Hash)
	})

	t.Run("modifying a child node should change only its hash", func(t *testing.T) {
		tmpDir := copyFixtures(t)

		graph0, err := dib.GenerateDAG(tmpDir,
			"eu.gcr.io/my-test-repository", "", map[string]string{})
		require.NoError(t, err)

		nodes0 := flattenNodes(graph0)
		rootNode0 := nodes0["bullseye"]
		subNode0 := nodes0["sub-image"]
		multistageNode0 := nodes0["multistage"]

		// When I add a new file in bullseye/multistage/ (child node)
		require.NoError(t, os.WriteFile(
			path.Join(tmpDir, "bullseye/multistage/newfile"),
			[]byte("file contents"),
			os.ModePerm))

		// Then ONLY the hash of the child node bullseye/multistage should have changed
		graph1, err := dib.GenerateDAG(tmpDir,
			"eu.gcr.io/my-test-repository", "", map[string]string{})
		require.NoError(t, err)

		nodes1 := flattenNodes(graph1)
		rootNode1 := nodes1["bullseye"]
		subNode1 := nodes1["sub-image"]
		multistageNode1 := nodes1["multistage"]

		assert.Equal(t, rootNode0.Image.Hash, rootNode1.Image.Hash)
		assert.Equal(t, subNode0.Image.Hash, subNode1.Image.Hash)
		assert.NotEqual(t, multistageNode0.Image.Hash, multistageNode1.Image.Hash)
	})

	t.Run("using custom hash list should change only hashes of nodes with custom label", func(t *testing.T) {
		graph0, err := dib.GenerateDAG(fixtureDir,
			"eu.gcr.io/my-test-repository", "", map[string]string{})
		require.NoError(t, err)

		graph1, err := dib.GenerateDAG(fixtureDir,
			"eu.gcr.io/my-test-repository",
			"../../test/fixtures/dib/valid_wordlist.txt",
			map[string]string{})
		require.NoError(t, err)

		nodes0 := flattenNodes(graph0)
		rootNode0 := nodes0["bullseye"]
		subNode0 := nodes0["sub-image"]
		nodes1 := flattenNodes(graph1)
		rootNode1 := nodes1["bullseye"]
		subNode1 := nodes1["sub-image"]

		assert.Equal(t, rootNode1.Image.Hash, rootNode0.Image.Hash)
		assert.Equal(t, "violet-minnesota-alabama-alpha", subNode0.Image.Hash)
		assert.Equal(t, "golduck-dialga-abra-aegislash", subNode1.Image.Hash)
	})

	t.Run("using arg used in root node should change all hashes", func(t *testing.T) {
		graph0, err := dib.GenerateDAG(fixtureDir,
			"eu.gcr.io/my-test-repository", "",
			map[string]string{})
		require.NoError(t, err)

		graph1, err := dib.GenerateDAG(fixtureDir,
			"eu.gcr.io/my-test-repository", "",
			map[string]string{
				"HELLO": "world",
			})
		require.NoError(t, err)

		nodes0 := flattenNodes(graph0)
		rootNode0 := nodes0["bullseye"]
		nodes1 := flattenNodes(graph1)
		rootNode1 := nodes1["bullseye"]

		assert.NotEqual(t, rootNode1.Image.Hash, rootNode0.Image.Hash)
	})
}

// copyFixtures copies the directory fixtureDir into a temporary one to be free to edit files.
func copyFixtures(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	src := path.Join(cwd, fixtureDir)
	dest := t.TempDir()
	cmd := exec.Command("cp", "-r", src, dest)
	require.NoError(t, cmd.Run())
	return dest + "/docker"
}

func flattenNodes(graph *dag.DAG) map[string]*dag.Node {
	flatNodes := map[string]*dag.Node{}

	graph.Walk(func(node *dag.Node) {
		flatNodes[node.Image.ShortName] = node
	})

	return flatNodes
}
