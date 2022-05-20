package dib_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/dib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GenerateDAG(t *testing.T) {
	t.Parallel()

	dockerDir := setupFixtures(t)
	DAG := dib.GenerateDAG(dockerDir, "eu.gcr.io/my-test-repository")

	assert.Len(t, DAG.Nodes(), 1)

	rootNode := DAG.Nodes()[0]
	rootImage := rootNode.Image
	assert.Equal(t, "eu.gcr.io/my-test-repository/bullseye", rootImage.Name)
	assert.Equal(t, "bullseye", rootImage.ShortName)
	assert.Len(t, rootNode.Parents(), 0)
	assert.Len(t, rootNode.Children(), 2)
}

func Test_GenerateDAG_ChildHashChangesWhenChildContextChanged(t *testing.T) {
	t.Parallel()

	// Given
	dockerDir := setupFixtures(t)
	initialDAG := dib.GenerateDAG(dockerDir, "eu.gcr.io/my-test-repository")

	initialNodes := flattenNodes(initialDAG)

	initialRootNode, ok := initialNodes["bullseye"]
	require.True(t, ok)

	initialSubNode, ok := initialNodes["sub-image"]
	require.True(t, ok)

	initialMultistageNode, ok := initialNodes["multistage"]
	require.True(t, ok)

	// When
	err := ioutil.WriteFile(
		path.Join(dockerDir, "bullseye/multistage/newfile"),
		[]byte("something"),
		os.ModePerm,
	)
	require.NoError(t, err)

	// Then
	newDAG := dib.GenerateDAG(dockerDir, "eu.gcr.io/my-test-repository")
	newNodes := flattenNodes(newDAG)

	newRootNode, ok := newNodes["bullseye"]
	require.True(t, ok)

	newSubNode, ok := newNodes["sub-image"]
	require.True(t, ok)

	newMultistageNode, ok := newNodes["multistage"]
	require.True(t, ok)

	assert.Equal(t, initialRootNode.Image.Hash, newRootNode.Image.Hash)
	assert.Equal(t, initialSubNode.Image.Hash, newSubNode.Image.Hash)
	assert.NotEqual(t, initialMultistageNode.Image.Hash, newMultistageNode.Image.Hash)
}

func Test_GenerateDAG_ChildHashChangesWhenParentContextChanged(t *testing.T) {
	t.Parallel()

	// Given
	dockerDir := setupFixtures(t)
	initialDAG := dib.GenerateDAG(dockerDir, "eu.gcr.io/my-test-repository")

	initialNodes := flattenNodes(initialDAG)

	initialRootNode, ok := initialNodes["bullseye"]
	require.True(t, ok)

	initialSubNode, ok := initialNodes["sub-image"]
	require.True(t, ok)

	initialMultistageNode, ok := initialNodes["multistage"]
	require.True(t, ok)

	// When
	err := ioutil.WriteFile(
		path.Join(dockerDir, "bullseye/newfile"),
		[]byte("something"),
		os.ModePerm,
	)
	require.NoError(t, err)

	// Then
	newDAG := dib.GenerateDAG(dockerDir, "eu.gcr.io/my-test-repository")
	newNodes := flattenNodes(newDAG)

	newRootNode, ok := newNodes["bullseye"]
	require.True(t, ok)

	newSubNode, ok := newNodes["sub-image"]
	require.True(t, ok)

	newMultistageNode, ok := newNodes["multistage"]
	require.True(t, ok)

	assert.NotEqual(t, initialRootNode.Image.Hash, newRootNode.Image.Hash)
	assert.NotEqual(t, initialSubNode.Image.Hash, newSubNode.Image.Hash)
	assert.NotEqual(t, initialMultistageNode.Image.Hash, newMultistageNode.Image.Hash)
}

func setupFixtures(t *testing.T) string {
	t.Helper()

	tmpDockerDir := t.TempDir()
	err := os.MkdirAll(path.Join(tmpDockerDir, "bullseye/multistage"), os.ModePerm)
	require.NoError(t, err)
	err = os.MkdirAll(path.Join(tmpDockerDir, "bullseye/sub-image"), os.ModePerm)
	require.NoError(t, err)
	err = os.MkdirAll(path.Join(tmpDockerDir, "bullseye/skipbuild"), os.ModePerm)
	require.NoError(t, err)
	err = ioutil.WriteFile(
		path.Join(tmpDockerDir, "bullseye/Dockerfile"),
		[]byte(`
FROM debian:bullseye

LABEL name="bullseye"
LABEL version="v1"
		`),
		os.ModePerm,
	)
	require.NoError(t, err)
	err = ioutil.WriteFile(
		path.Join(tmpDockerDir, "bullseye/sub-image/Dockerfile"),
		[]byte(`
FROM eu.gcr.io/my-test-repository/bullseye:v1

LABEL name="sub-image"
LABEL version="v1"
		`),
		os.ModePerm,
	)
	require.NoError(t, err)
	err = ioutil.WriteFile(
		path.Join(tmpDockerDir, "bullseye/multistage/Dockerfile"),
		[]byte(`
FROM eu.gcr.io/my-test-repository/bullseye:v1 as builder
FROM eu.gcr.io/my-test-repository/node:v1

LABEL name="multistage"
LABEL version="v1"
		`),
		os.ModePerm,
	)
	require.NoError(t, err)
	err = ioutil.WriteFile(
		path.Join(tmpDockerDir, "bullseye/skipbuild/Dockerfile"),
		[]byte(`
FROM eu.gcr.io/my-test-repository/bullseye:v1

LABEL name="skipbuild"
LABEL skipbuild="true"
		`),
		os.ModePerm,
	)
	require.NoError(t, err)

	return tmpDockerDir
}

func flattenNodes(graph *dag.DAG) map[string]*dag.Node {
	flatNodes := map[string]*dag.Node{}

	graph.Walk(func(node *dag.Node) {
		flatNodes[node.Image.ShortName] = node
	})

	return flatNodes
}
