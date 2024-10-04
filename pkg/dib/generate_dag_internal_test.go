package dib

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	buildPath      = "../../test/fixtures/docker"
	registryPrefix = "eu.gcr.io/my-test-repository"
)

func Test_buildGraph(t *testing.T) {
	t.Parallel()

	graph, err := buildGraph(buildPath, registryPrefix)
	require.NoError(t, err)
	graph.WalkInDepth(func(node *dag.Node) {
		switch node.Image.ShortName {
		case "root1":
			require.Len(t, node.Files, 9, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/root1/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/root1/custom-hash-list/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/root1/dockerignore/.dockerignore")
			assert.Contains(t, node.Files, buildPath+"/root1/dockerignore/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/root1/dockerignore/ignored.txt")
			assert.Contains(t, node.Files, buildPath+"/root1/multistage/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/root1/skipbuild/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/root1/with-a-file/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/root1/with-a-file/included.txt")
		case "custom-hash-list":
			require.Len(t, node.Files, 1, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/root1/custom-hash-list/Dockerfile")
		case "dockerignore":
			require.Len(t, node.Files, 1, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/root1/dockerignore/Dockerfile")
		case "multistage":
			require.Len(t, node.Files, 1, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/root1/multistage/Dockerfile")
		case "with-a-file":
			require.Len(t, node.Files, 2, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/root1/with-a-file/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/root1/with-a-file/included.txt")
		case "root2":
			require.Len(t, node.Files, 2, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/root2/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/root2/root3/Dockerfile")
		case "root3":
			require.Len(t, node.Files, 1, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/root2/root3/Dockerfile")
		default:
			t.Errorf("unexpected image: %s", node.Image.ShortName)
		}
	})
}
