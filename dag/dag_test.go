package dag_test

import (
	"os"
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/docker"
)

func newImage(name string, contextPath string) *dag.Image {
	return &dag.Image{
		Name:          name,
		ShortName:     path.Base(contextPath),
		InlineVersion: "v1",
		Dockerfile: &docker.Dockerfile{
			ContextPath: contextPath,
			Filename:    "Dockerfile",
			From:        []string{"debian"},
			Labels: map[string]string{
				"name":    path.Base(contextPath),
				"version": "v1",
			},
		},
		Children:     nil,
		Parents:      nil,
		NeedsRebuild: false,
		RetagDone:    false,
		RebuildDone:  false,
		RebuildCond:  sync.NewCond(&sync.Mutex{}),
	}
}

func Test_CheckForDiff_RebuildAllChildren(t *testing.T) {
	t.Parallel()

	rootImg := newImage("bullseye", "/root/docker/bullseye")

	firstChildImg := newImage("eu.gcr.io/my-test-repository/first", "/root/docker/bullseye/first")
	firstChildImg.Parents = []*dag.Image{rootImg}

	secondChildImg := newImage("eu.gcr.io/my-test-repository/second", "/root/docker/bullseye/second")
	secondChildImg.Parents = []*dag.Image{rootImg}

	subChildImg := newImage("eu.gcr.io/my-test-repository/third", "/root/docker/bullseye/second/third")
	subChildImg.Parents = []*dag.Image{secondChildImg}

	secondChildImg.Children = []*dag.Image{subChildImg}

	rootImg.Children = []*dag.Image{firstChildImg, secondChildImg}

	DAG := dag.DAG{Images: []*dag.Image{rootImg}}

	diff := []string{
		"/root/docker/bullseye/Dockerfile",
	}

	DAG.CheckForDiff(diff)

	assert.True(t, rootImg.NeedsRebuild)        // Root image was modified.
	assert.True(t, firstChildImg.NeedsRebuild)  // First image was NOT modified, but its parent was.
	assert.True(t, secondChildImg.NeedsRebuild) // Second image was NOT modified, but its parent was.
	assert.True(t, subChildImg.NeedsRebuild)    // Second's child image was NOT modified but its parent's parent was.
}

func Test_CheckForDiff_RebuildOnlyChildren(t *testing.T) {
	t.Parallel()

	rootImg := newImage("bullseye", "/root/docker/bullseye")

	firstChildImg := newImage("eu.gcr.io/my-test-repository/first", "/root/docker/bullseye/first")
	firstChildImg.Parents = []*dag.Image{rootImg}

	secondChildImg := newImage("eu.gcr.io/my-test-repository/second", "/root/docker/bullseye/second")
	secondChildImg.Parents = []*dag.Image{rootImg}

	subChildImg := newImage("eu.gcr.io/my-test-repository/third", "/root/docker/bullseye/second/third")
	subChildImg.Parents = []*dag.Image{secondChildImg}

	secondChildImg.Children = []*dag.Image{subChildImg}

	rootImg.Children = []*dag.Image{firstChildImg, secondChildImg}

	DAG := dag.DAG{Images: []*dag.Image{rootImg}}

	diff := []string{
		"/root/docker/bullseye/first/nginx.conf",
		"/root/docker/bullseye/second/third/Dockerfile",
	}

	DAG.CheckForDiff(diff)

	assert.False(t, rootImg.NeedsRebuild)        // Root image was NOT modified.
	assert.True(t, firstChildImg.NeedsRebuild)   // First image was modified.
	assert.False(t, secondChildImg.NeedsRebuild) // Second image was NOT modified, nor its parent.
	assert.True(t, subChildImg.NeedsRebuild)     // Second's child image was modified.

	DAG.TagForRebuild()

	assert.True(t, rootImg.NeedsRebuild)
	assert.True(t, firstChildImg.NeedsRebuild)
	assert.True(t, secondChildImg.NeedsRebuild)
	assert.True(t, subChildImg.NeedsRebuild)
}

func Test_GenerateDAG(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory.")
	}

	buildPath := path.Join(cwd, "../test/fixtures/docker")

	DAG := dag.DAG{}
	DAG.GenerateDAG(buildPath, "eu.gcr.io/my-test-repository")

	assert.Len(t, DAG.Images, 1)

	rootImage := DAG.Images[0]
	assert.Equal(t, "eu.gcr.io/my-test-repository/bullseye", rootImage.Name)
	assert.Equal(t, "bullseye", rootImage.ShortName)
	assert.Len(t, rootImage.Parents, 0)
	assert.Len(t, rootImage.Children, 3)
}
