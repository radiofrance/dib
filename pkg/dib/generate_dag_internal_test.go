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
		case "bullseye":
			require.Len(t, node.Files, 5, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/bullseye/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/bullseye/external-parent/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/bullseye/multistage/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/bullseye/skipbuild/Dockerfile")
			assert.Contains(t, node.Files, buildPath+"/bullseye/sub-image/Dockerfile")
		case "kaniko":
			require.Len(t, node.Files, 1, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/bullseye/external-parent/Dockerfile")
		case "multistage":
			require.Len(t, node.Files, 1, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/bullseye/multistage/Dockerfile")
		case "skipbuild":
			require.Len(t, node.Files, 1, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/bullseye/skipbuild/Dockerfile")
		case "sub-image":
			require.Len(t, node.Files, 1, spew.Sdump(node.Files))
			assert.Contains(t, node.Files, buildPath+"/bullseye/sub-image/Dockerfile")
		default:
			t.Errorf("unexpected image: %s", node.Image.ShortName)
		}
	})
}
