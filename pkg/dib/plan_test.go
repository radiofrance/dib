package dib_test

import (
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/radiofrance/dib/pkg/mock"
)

func newNode(name, hash, contextPath string) *dag.Node {
	return dag.NewNode(&dag.Image{
		Name:      name,
		Hash:      hash,
		ShortName: path.Base(contextPath),
		Dockerfile: &dockerfile.Dockerfile{
			ContextPath: contextPath,
			Filename:    "Dockerfile",
			From: []dockerfile.ImageRef{
				{Name: "debian"},
			},
			Labels: map[string]string{
				"name":    path.Base(contextPath),
				"version": "v1",
			},
		},
	})
}

//nolint:lll,dupl
func Test_Plan_RebuildAll(t *testing.T) {
	t.Parallel()

	rootNode := newNode("bullseye", "notexists0", "/root/docker/bullseye")

	firstChildNode := newNode("eu.gcr.io/my-test-repository/first", "exists1", "/root/docker/bullseye/first")
	secondChildNode := newNode("eu.gcr.io/my-test-repository/second", "exists2", "/root/docker/bullseye/second")
	subChildNode := newNode("eu.gcr.io/my-test-repository/third", "exists3", "/root/docker/bullseye/second/third")

	secondChildNode.AddChild(subChildNode)

	rootNode.AddChild(firstChildNode)
	rootNode.AddChild(secondChildNode)

	graph := &dag.DAG{}
	graph.AddNode(rootNode)

	registry := &mock.Registry{Lock: &sync.Mutex{}}
	registry.ExistingRefs = []string{
		"bullseye:notexists0",
		"eu.gcr.io/my-test-repository/first:exists1",
		"eu.gcr.io/my-test-repository/second:exists2",
		"eu.gcr.io/my-test-repository/third:exists3",
	}

	dibBuilder := &dib.Builder{
		Graph: graph,
		BuildOpts: dib.BuildOpts{
			ForceRebuild: true,
			NoTests:      false,
		},
	}
	err := dibBuilder.Plan(registry)
	require.NoError(t, err)

	assert.True(t, rootNode.Image.NeedsRebuild)        // Root image was modified.
	assert.True(t, firstChildNode.Image.NeedsRebuild)  // First image was NOT modified, but its parent was.
	assert.True(t, secondChildNode.Image.NeedsRebuild) // Second image was NOT modified, but its parent was.
	assert.True(t, subChildNode.Image.NeedsRebuild)    // Second's child image was NOT modified but its parent's parent was.

	// All images will be rebuilt, they all need to be tested because tests are enabled.
	assert.True(t, rootNode.Image.NeedsTests)
	assert.True(t, firstChildNode.Image.NeedsTests)
	assert.True(t, secondChildNode.Image.NeedsTests)
	assert.True(t, subChildNode.Image.NeedsTests)

	// No image was already built because none exist in registry.
	assert.False(t, rootNode.Image.RebuildDone)
	assert.False(t, firstChildNode.Image.RebuildDone)
	assert.False(t, secondChildNode.Image.RebuildDone)
	assert.False(t, subChildNode.Image.RebuildDone)
}

//nolint:dupl
func Test_Plan_RebuildOnlyModifiedImages(t *testing.T) {
	t.Parallel()

	rootNode := newNode("bullseye", "exists0", "/root/docker/bullseye")

	firstChildNode := newNode("eu.gcr.io/my-test-repository/first", "notexists1", "/root/docker/bullseye/first")
	secondChildNode := newNode("eu.gcr.io/my-test-repository/second", "exists2", "/root/docker/bullseye/second")
	subChildNode := newNode("eu.gcr.io/my-test-repository/third", "notexists3", "/root/docker/bullseye/second/third")

	secondChildNode.AddChild(subChildNode)

	rootNode.AddChild(firstChildNode)
	rootNode.AddChild(secondChildNode)

	graph := &dag.DAG{}
	graph.AddNode(rootNode)

	registry := &mock.Registry{Lock: &sync.Mutex{}}
	registry.ExistingRefs = []string{
		"bullseye:exists0",
		"eu.gcr.io/my-test-repository/first:exists1",
		"eu.gcr.io/my-test-repository/second:exists2",
		"eu.gcr.io/my-test-repository/third:exists3",
	}

	dibBuilder := &dib.Builder{
		Graph: graph,
		BuildOpts: dib.BuildOpts{
			ForceRebuild: false,
			NoTests:      false,
		},
	}
	err := dibBuilder.Plan(registry)
	require.NoError(t, err)

	assert.False(t, rootNode.Image.NeedsRebuild)        // Root image was NOT modified.
	assert.True(t, firstChildNode.Image.NeedsRebuild)   // First image was modified.
	assert.False(t, secondChildNode.Image.NeedsRebuild) // Second image was NOT modified, nor its parent.
	assert.True(t, subChildNode.Image.NeedsRebuild)     // Second's child image was modified.

	// Images that need rebuild need to be tested as tests are enabled
	assert.False(t, rootNode.Image.NeedsTests)
	assert.True(t, firstChildNode.Image.NeedsTests)
	assert.False(t, secondChildNode.Image.NeedsTests)
	assert.True(t, subChildNode.Image.NeedsTests)

	// No image was already built because none exist in registry.
	assert.False(t, rootNode.Image.RebuildDone)
	assert.False(t, firstChildNode.Image.RebuildDone)
	assert.False(t, secondChildNode.Image.RebuildDone)
	assert.False(t, subChildNode.Image.RebuildDone)
}

func Test_Plan_TestsDisabled(t *testing.T) {
	t.Parallel()

	rootNode := newNode("bullseye", "exists0", "/root/docker/bullseye")

	firstChildNode := newNode("eu.gcr.io/my-test-repository/first", "notexists1", "/root/docker/bullseye/first")
	secondChildNode := newNode("eu.gcr.io/my-test-repository/second", "notexists2", "/root/docker/bullseye/second")
	subChildNode := newNode("eu.gcr.io/my-test-repository/third", "exists3", "/root/docker/bullseye/second/third")

	secondChildNode.AddChild(subChildNode)

	rootNode.AddChild(firstChildNode)
	rootNode.AddChild(secondChildNode)

	graph := &dag.DAG{}
	graph.AddNode(rootNode)

	registry := &mock.Registry{Lock: &sync.Mutex{}}
	registry.ExistingRefs = []string{}

	dibBuilder := &dib.Builder{
		Graph: graph,
		BuildOpts: dib.BuildOpts{
			ForceRebuild: true,
			NoTests:      true,
		},
	}
	err := dibBuilder.Plan(registry)
	require.NoError(t, err)

	assert.True(t, rootNode.Image.NeedsRebuild)
	assert.True(t, firstChildNode.Image.NeedsRebuild)
	assert.True(t, secondChildNode.Image.NeedsRebuild)
	assert.True(t, subChildNode.Image.NeedsRebuild)

	assert.False(t, rootNode.Image.NeedsTests)
	assert.False(t, firstChildNode.Image.NeedsTests)
	assert.False(t, secondChildNode.Image.NeedsTests)
	assert.False(t, subChildNode.Image.NeedsTests)
}
