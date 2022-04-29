package dib_test

import (
	"path"
	"sync"
	"testing"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/dib"
	"github.com/radiofrance/dib/dockerfile"
	"github.com/radiofrance/dib/mock"
	"github.com/stretchr/testify/assert"
)

func newNode(name string, contextPath string) *dag.Node {
	return dag.NewNode(&dag.Image{
		Name:      name,
		ShortName: path.Base(contextPath),
		Dockerfile: &dockerfile.Dockerfile{
			ContextPath: contextPath,
			Filename:    "Dockerfile",
			From:        []string{"debian"},
			Labels: map[string]string{
				"name":    path.Base(contextPath),
				"version": "v1",
			},
		},
	})
}

//nolint:lll
func Test_Plan_RebuildAll(t *testing.T) {
	t.Parallel()

	rootNode := newNode("bullseye", "/root/docker/bullseye")

	firstChildNode := newNode("eu.gcr.io/my-test-repository/first", "/root/docker/bullseye/first")
	secondChildNode := newNode("eu.gcr.io/my-test-repository/second", "/root/docker/bullseye/second")
	subChildNode := newNode("eu.gcr.io/my-test-repository/third", "/root/docker/bullseye/second/third")

	secondChildNode.AddChild(subChildNode)

	rootNode.AddChild(firstChildNode)
	rootNode.AddChild(secondChildNode)

	DAG := &dag.DAG{}
	DAG.AddNode(rootNode)

	diff := []string{
		"/root/docker/bullseye/Dockerfile",
	}

	registry := &mock.Registry{Lock: &sync.Mutex{}}
	registry.ExistingRefs = []string{
		// Old tag from previous version
		"bullseye:old",
		"eu.gcr.io/my-test-repository/first:old",
		"eu.gcr.io/my-test-repository/second:old",
		"eu.gcr.io/my-test-repository/third:old",
	}

	err := dib.Plan(DAG, registry, diff, "old", "new", true, true, true)
	assert.NoError(t, err)

	assert.True(t, rootNode.Image.NeedsRebuild)        // Root image was modified.
	assert.True(t, firstChildNode.Image.NeedsRebuild)  // First image was NOT modified, but its parent was.
	assert.True(t, secondChildNode.Image.NeedsRebuild) // Second image was NOT modified, but its parent was.
	assert.True(t, subChildNode.Image.NeedsRebuild)    // Second's child image was NOT modified but its parent's parent was.

	// All images will be rebuilt, no need to retag anything.
	assert.False(t, rootNode.Image.NeedsRetag)
	assert.False(t, firstChildNode.Image.NeedsRetag)
	assert.False(t, secondChildNode.Image.NeedsRetag)
	assert.False(t, subChildNode.Image.NeedsRetag)

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

func Test_Plan_RebuildOnlyDiff(t *testing.T) {
	t.Parallel()

	rootNode := newNode("bullseye", "/root/docker/bullseye")

	firstChildNode := newNode("eu.gcr.io/my-test-repository/first", "/root/docker/bullseye/first")
	secondChildNode := newNode("eu.gcr.io/my-test-repository/second", "/root/docker/bullseye/second")
	subChildNode := newNode("eu.gcr.io/my-test-repository/third", "/root/docker/bullseye/second/third")

	secondChildNode.AddChild(subChildNode)

	rootNode.AddChild(firstChildNode)
	rootNode.AddChild(secondChildNode)

	DAG := &dag.DAG{}
	DAG.AddNode(rootNode)

	diff := []string{
		"/root/docker/bullseye/first/nginx.conf",
		"/root/docker/bullseye/second/third/Dockerfile",
	}

	registry := &mock.Registry{Lock: &sync.Mutex{}}
	registry.ExistingRefs = []string{
		// Old tag from previous version
		"bullseye:old",
		"eu.gcr.io/my-test-repository/first:old",
		"eu.gcr.io/my-test-repository/second:old",
		"eu.gcr.io/my-test-repository/third:old",
	}

	err := dib.Plan(DAG, registry, diff, "old", "new", true, false, true)
	assert.NoError(t, err)

	assert.False(t, rootNode.Image.NeedsRebuild)        // Root image was NOT modified.
	assert.True(t, firstChildNode.Image.NeedsRebuild)   // First image was modified.
	assert.False(t, secondChildNode.Image.NeedsRebuild) // Second image was NOT modified, nor its parent.
	assert.True(t, subChildNode.Image.NeedsRebuild)     // Second's child image was modified.

	// Images that won't be rebuilt need a new tag
	assert.True(t, rootNode.Image.NeedsRetag)
	assert.False(t, firstChildNode.Image.NeedsRetag)
	assert.True(t, secondChildNode.Image.NeedsRetag)
	assert.False(t, subChildNode.Image.NeedsRetag)

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

func Test_Plan_ImagesAlreadyBuilt(t *testing.T) {
	t.Parallel()

	rootNode := newNode("bullseye", "/root/docker/bullseye")

	firstChildNode := newNode("eu.gcr.io/my-test-repository/first", "/root/docker/bullseye/first")
	secondChildNode := newNode("eu.gcr.io/my-test-repository/second", "/root/docker/bullseye/second")
	subChildNode := newNode("eu.gcr.io/my-test-repository/third", "/root/docker/bullseye/second/third")

	secondChildNode.AddChild(subChildNode)

	rootNode.AddChild(firstChildNode)
	rootNode.AddChild(secondChildNode)

	DAG := &dag.DAG{}
	DAG.AddNode(rootNode)

	diff := []string{
		"/root/docker/bullseye/first/nginx.conf",
		"/root/docker/bullseye/second/third/Dockerfile",
	}

	registry := &mock.Registry{Lock: &sync.Mutex{}}
	registry.ExistingRefs = []string{
		// Old tag from previous version
		"bullseye:old",
		"eu.gcr.io/my-test-repository/first:old",
		"eu.gcr.io/my-test-repository/second:old",
		"eu.gcr.io/my-test-repository/third:old",
		// New tag from current version
		"eu.gcr.io/my-test-repository/first:new",
		"eu.gcr.io/my-test-repository/third:new",
	}
	err := dib.Plan(DAG, registry, diff, "old", "new", true, false, true)
	assert.NoError(t, err)

	assert.False(t, rootNode.Image.NeedsRebuild)        // Root image was NOT modified.
	assert.True(t, firstChildNode.Image.NeedsRebuild)   // First image was modified.
	assert.False(t, secondChildNode.Image.NeedsRebuild) // Second image was NOT modified, nor its parent.
	assert.True(t, subChildNode.Image.NeedsRebuild)     // Second's child image was modified.

	// Image that already exist in registry are flagged.
	assert.False(t, rootNode.Image.RebuildDone)
	assert.True(t, firstChildNode.Image.RebuildDone)
	assert.False(t, secondChildNode.Image.RebuildDone)
	assert.True(t, subChildNode.Image.RebuildDone)

	// Only non-modified images need a new tag
	assert.True(t, rootNode.Image.NeedsRetag)
	assert.False(t, firstChildNode.Image.NeedsRetag)
	assert.True(t, secondChildNode.Image.NeedsRetag)
	assert.False(t, subChildNode.Image.NeedsRetag)

	// Images that need rebuild need to be tested as tests are enabled
	assert.False(t, rootNode.Image.NeedsTests)
	assert.True(t, firstChildNode.Image.NeedsTests)
	assert.False(t, secondChildNode.Image.NeedsTests)
	assert.True(t, subChildNode.Image.NeedsTests)
}

func Test_Plan_ImagesAlreadyTagged(t *testing.T) {
	t.Parallel()

	rootNode := newNode("bullseye", "/root/docker/bullseye")

	firstChildNode := newNode("eu.gcr.io/my-test-repository/first", "/root/docker/bullseye/first")
	secondChildNode := newNode("eu.gcr.io/my-test-repository/second", "/root/docker/bullseye/second")
	subChildNode := newNode("eu.gcr.io/my-test-repository/third", "/root/docker/bullseye/second/third")

	secondChildNode.AddChild(subChildNode)

	rootNode.AddChild(firstChildNode)
	rootNode.AddChild(secondChildNode)

	DAG := &dag.DAG{}
	DAG.AddNode(rootNode)

	diff := []string{""}

	registry := &mock.Registry{Lock: &sync.Mutex{}}
	registry.ExistingRefs = []string{
		// Old tag from previous version
		"bullseye:old",
		"eu.gcr.io/my-test-repository/first:old",
		"eu.gcr.io/my-test-repository/second:old",
		"eu.gcr.io/my-test-repository/third:old",
		// New tag from current version
		"bullseye:new",
		"eu.gcr.io/my-test-repository/first:new",
		"eu.gcr.io/my-test-repository/second:new",
		"eu.gcr.io/my-test-repository/third:new",
	}
	err := dib.Plan(DAG, registry, diff, "old", "new", true, false, true)
	assert.NoError(t, err)

	assert.False(t, rootNode.Image.NeedsRebuild)
	assert.False(t, firstChildNode.Image.NeedsRebuild)
	assert.False(t, secondChildNode.Image.NeedsRebuild)
	assert.False(t, subChildNode.Image.NeedsRebuild)

	// No need for a neg tag as it already exist in registry
	assert.False(t, rootNode.Image.NeedsRetag)
	assert.False(t, firstChildNode.Image.NeedsRetag)
	assert.False(t, secondChildNode.Image.NeedsRetag)
	assert.False(t, subChildNode.Image.NeedsRetag)
}

func Test_Plan_OldTagNotFoundInRegistry(t *testing.T) {
	t.Parallel()

	rootNode := newNode("bullseye", "/root/docker/bullseye")

	firstChildNode := newNode("eu.gcr.io/my-test-repository/first", "/root/docker/bullseye/first")
	secondChildNode := newNode("eu.gcr.io/my-test-repository/second", "/root/docker/bullseye/second")
	subChildNode := newNode("eu.gcr.io/my-test-repository/third", "/root/docker/bullseye/second/third")

	secondChildNode.AddChild(subChildNode)

	rootNode.AddChild(firstChildNode)
	rootNode.AddChild(secondChildNode)

	DAG := &dag.DAG{}
	DAG.AddNode(rootNode)

	diff := []string{
		"/root/docker/bullseye/first/nginx.conf",
		"/root/docker/bullseye/second/third/Dockerfile",
	}

	registry := &mock.Registry{Lock: &sync.Mutex{}}
	registry.ExistingRefs = []string{}
	err := dib.Plan(DAG, registry, diff, "old", "new", true, false, true)
	assert.NoError(t, err)

	assert.True(t, rootNode.Image.NeedsRebuild)
	assert.True(t, firstChildNode.Image.NeedsRebuild)
	assert.True(t, secondChildNode.Image.NeedsRebuild)
	assert.True(t, subChildNode.Image.NeedsRebuild)
}

func Test_Plan_TestsDisabled(t *testing.T) {
	t.Parallel()

	rootNode := newNode("bullseye", "/root/docker/bullseye")

	firstChildNode := newNode("eu.gcr.io/my-test-repository/first", "/root/docker/bullseye/first")
	secondChildNode := newNode("eu.gcr.io/my-test-repository/second", "/root/docker/bullseye/second")
	subChildNode := newNode("eu.gcr.io/my-test-repository/third", "/root/docker/bullseye/second/third")

	secondChildNode.AddChild(subChildNode)

	rootNode.AddChild(firstChildNode)
	rootNode.AddChild(secondChildNode)

	DAG := &dag.DAG{}
	DAG.AddNode(rootNode)

	diff := []string{
		"/root/docker/bullseye/first/nginx.conf",
		"/root/docker/bullseye/second/third/Dockerfile",
	}

	registry := &mock.Registry{Lock: &sync.Mutex{}}
	registry.ExistingRefs = []string{}
	err := dib.Plan(DAG, registry, diff, "old", "new", true, true, false)
	assert.NoError(t, err)

	assert.True(t, rootNode.Image.NeedsRebuild)
	assert.True(t, firstChildNode.Image.NeedsRebuild)
	assert.True(t, secondChildNode.Image.NeedsRebuild)
	assert.True(t, subChildNode.Image.NeedsRebuild)

	assert.False(t, rootNode.Image.NeedsTests)
	assert.False(t, firstChildNode.Image.NeedsTests)
	assert.False(t, secondChildNode.Image.NeedsTests)
	assert.False(t, subChildNode.Image.NeedsTests)
}

func Test_Plan_ReleaseModeDisabled(t *testing.T) {
	t.Parallel()

	rootNode := newNode("bullseye", "/root/docker/bullseye")

	firstChildNode := newNode("eu.gcr.io/my-test-repository/first", "/root/docker/bullseye/first")
	secondChildNode := newNode("eu.gcr.io/my-test-repository/second", "/root/docker/bullseye/second")
	subChildNode := newNode("eu.gcr.io/my-test-repository/third", "/root/docker/bullseye/second/third")

	secondChildNode.AddChild(subChildNode)

	rootNode.AddChild(firstChildNode)
	rootNode.AddChild(secondChildNode)

	DAG := &dag.DAG{}
	DAG.AddNode(rootNode)

	diff := []string{
		"/root/docker/bullseye/first/nginx.conf",
		"/root/docker/bullseye/second/third/Dockerfile",
	}

	registry := &mock.Registry{Lock: &sync.Mutex{}}
	registry.ExistingRefs = []string{
		// Old tag from previous version
		"bullseye:old",
		"eu.gcr.io/my-test-repository/first:old",
		"eu.gcr.io/my-test-repository/second:old",
		"eu.gcr.io/my-test-repository/third:old",
	}
	err := dib.Plan(DAG, registry, diff, "old", "new", false, false, true)
	assert.NoError(t, err)

	assert.False(t, rootNode.Image.NeedsRebuild)
	assert.True(t, firstChildNode.Image.NeedsRebuild)
	assert.False(t, secondChildNode.Image.NeedsRebuild)
	assert.True(t, subChildNode.Image.NeedsRebuild)

	// Nothing need to be tagged in dev mode
	assert.False(t, rootNode.Image.NeedsRetag)
	assert.False(t, firstChildNode.Image.NeedsRetag)
	assert.False(t, secondChildNode.Image.NeedsRetag)
	assert.False(t, subChildNode.Image.NeedsRetag)

	assert.Equal(t, "new", firstChildNode.Image.TargetTag)
	assert.Equal(t, "new", firstChildNode.Image.TargetTag)
}
