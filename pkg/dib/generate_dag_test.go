package dib_test

import (
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dib"
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

	nodes := flattenNodes(DAG)

	multistageNode, exists := nodes["multistage"]
	require.True(t, exists)
	assert.Equal(t, []string{"latest"}, multistageNode.Image.ExtraTags)
}

//nolint:govet
func Test_GenerateDAG_HashesChangeWhenImageContextChanges(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		AddFileAtPath                        string
		ExpectRootImageHashesToBeEqual       bool
		ExpectSubImageHashesToBeEqual        bool
		ExpectMultistageImageHashesToBeEqual bool
	}{
		"Child image hash changes when child context changes": {
			AddFileAtPath:                        "bullseye/multistage/newfile",
			ExpectRootImageHashesToBeEqual:       true,
			ExpectSubImageHashesToBeEqual:        true,
			ExpectMultistageImageHashesToBeEqual: false,
		},
		"Child hash changes when parent context changes": {
			AddFileAtPath:                        "bullseye/newfile",
			ExpectRootImageHashesToBeEqual:       false,
			ExpectSubImageHashesToBeEqual:        false,
			ExpectMultistageImageHashesToBeEqual: false,
		},
	}

	//nolint:paralleltest
	for name, testcase := range testcases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Given I have a docker directory with some Dockerfiles inside
			dockerDir := setupFixtures(t)
			initialDAG := dib.GenerateDAG(dockerDir, "eu.gcr.io/my-test-repository")
			initialNodes := flattenNodes(initialDAG)

			initialRootNode, exists := initialNodes["bullseye"]
			require.True(t, exists)

			initialSubNode, exists := initialNodes["sub-image"]
			require.True(t, exists)

			initialMultistageNode, exists := initialNodes["multistage"]
			require.True(t, exists)

			// When I add a new file in bullseye/multistage/ (child node)
			err := os.WriteFile(
				path.Join(dockerDir, testcase.AddFileAtPath),
				[]byte("file contents"),
				os.ModePerm,
			)
			require.NoError(t, err)

			// Then ONLY the hash of the child node bullseye/multistage should have changed
			newDAG := dib.GenerateDAG(dockerDir, "eu.gcr.io/my-test-repository")
			newNodes := flattenNodes(newDAG)

			newRootNode, exists := newNodes["bullseye"]
			require.True(t, exists)

			newSubNode, exists := newNodes["sub-image"]
			require.True(t, exists)

			newMultistageNode, exists := newNodes["multistage"]
			require.True(t, exists)

			if testcase.ExpectRootImageHashesToBeEqual {
				assert.Equal(t, initialRootNode.Image.Hash, newRootNode.Image.Hash)
			} else {
				assert.NotEqual(t, initialRootNode.Image.Hash, newRootNode.Image.Hash)
			}

			if testcase.ExpectSubImageHashesToBeEqual {
				assert.Equal(t, initialSubNode.Image.Hash, newSubNode.Image.Hash)
			} else {
				assert.NotEqual(t, initialSubNode.Image.Hash, newSubNode.Image.Hash)
			}

			if testcase.ExpectMultistageImageHashesToBeEqual {
				assert.Equal(t, initialMultistageNode.Image.Hash, newMultistageNode.Image.Hash)
			} else {
				assert.NotEqual(t, initialMultistageNode.Image.Hash, newMultistageNode.Image.Hash)
			}
		})
	}
}

// setupFixtures create a tmp directory with some Dockerfiles inside.
func setupFixtures(t *testing.T) string {
	t.Helper()

	tmpDockerDir := t.TempDir()
	err := os.MkdirAll(path.Join(tmpDockerDir, "bullseye/multistage"), os.ModePerm)
	require.NoError(t, err)
	err = os.MkdirAll(path.Join(tmpDockerDir, "bullseye/sub-image"), os.ModePerm)
	require.NoError(t, err)
	err = os.MkdirAll(path.Join(tmpDockerDir, "bullseye/skipbuild"), os.ModePerm)
	require.NoError(t, err)
	err = os.WriteFile(
		path.Join(tmpDockerDir, "bullseye/Dockerfile"),
		[]byte(`
FROM debian:bullseye

LABEL name="bullseye"
LABEL version="v1"
		`),
		os.ModePerm,
	)
	require.NoError(t, err)
	err = os.WriteFile(
		path.Join(tmpDockerDir, "bullseye/sub-image/Dockerfile"),
		[]byte(`
FROM eu.gcr.io/my-test-repository/bullseye:v1

LABEL name="sub-image"
LABEL version="v1"
		`),
		os.ModePerm,
	)
	require.NoError(t, err)
	err = os.WriteFile(
		path.Join(tmpDockerDir, "bullseye/multistage/Dockerfile"),
		[]byte(`
FROM eu.gcr.io/my-test-repository/bullseye:v1 as builder
FROM eu.gcr.io/my-test-repository/node:v1

LABEL name="multistage"
LABEL dib.extra-tags="latest"
		`),
		os.ModePerm,
	)
	require.NoError(t, err)
	err = os.WriteFile(
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
