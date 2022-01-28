package dib_test

import (
	"testing"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/dib"
	"github.com/radiofrance/dib/mock"
	"github.com/stretchr/testify/assert"
)

func Test_Retag_DoesNotRetagIfNoRetagNeeded(t *testing.T) {
	t.Parallel()

	DAG := &dag.DAG{}
	DAG.AddNode(dag.NewNode(&dag.Image{
		Name:       "registry.example.org/image",
		ShortName:  "image",
		NeedsRetag: false,
		RetagDone:  false,
	}))

	tagger := &mock.Tagger{}
	err := dib.Retag(DAG, tagger, "old", "new")

	assert.NoError(t, err)
	assert.Empty(t, tagger.RecordedCallsArgs)
}

func Test_Retag_DoesNotRetagIfAlreadyDone(t *testing.T) {
	t.Parallel()

	DAG := &dag.DAG{}
	DAG.AddNode(dag.NewNode(&dag.Image{
		Name:       "registry.example.org/image",
		ShortName:  "image",
		NeedsRetag: true,
		RetagDone:  true,
	}))

	tagger := &mock.Tagger{}
	err := dib.Retag(DAG, tagger, "old", "new")

	assert.NoError(t, err)
	assert.Empty(t, tagger.RecordedCallsArgs)
}

func Test_Retag_Retag(t *testing.T) {
	t.Parallel()

	img := &dag.Image{
		Name:       "registry.example.org/image",
		ShortName:  "image",
		NeedsRetag: true,
		RetagDone:  false,
	}
	DAG := &dag.DAG{}
	DAG.AddNode(dag.NewNode(img))

	tagger := &mock.Tagger{}
	err := dib.Retag(DAG, tagger, "old", "new")

	assert.NoError(t, err)

	assert.Len(t, tagger.RecordedCallsArgs, 1)
	args := tagger.RecordedCallsArgs[0]
	assert.Equal(t, "registry.example.org/image:old", args.Src)
	assert.Equal(t, "registry.example.org/image:new", args.Dest)

	assert.True(t, img.RetagDone)
}

func Test_RetagLatest_DoesNotRetagIfAlreadyDone(t *testing.T) {
	t.Parallel()

	DAG := &dag.DAG{}
	DAG.AddNode(dag.NewNode(&dag.Image{
		Name:            "registry.example.org/image",
		ShortName:       "image",
		RetagLatestDone: true,
	}))

	tagger := &mock.Tagger{}
	err := dib.RetagLatest(DAG, tagger, "old")

	assert.NoError(t, err)
	assert.Empty(t, tagger.RecordedCallsArgs)
}

func Test_RetagLatest_Retag(t *testing.T) {
	t.Parallel()

	img := &dag.Image{
		Name:            "registry.example.org/image",
		ShortName:       "image",
		RetagLatestDone: false,
	}
	DAG := &dag.DAG{}
	DAG.AddNode(dag.NewNode(img))

	tagger := &mock.Tagger{}
	err := dib.RetagLatest(DAG, tagger, "old")

	assert.NoError(t, err)

	assert.Len(t, tagger.RecordedCallsArgs, 1)
	args := tagger.RecordedCallsArgs[0]
	assert.Equal(t, "registry.example.org/image:old", args.Src)
	assert.Equal(t, "registry.example.org/image:latest", args.Dest)

	assert.True(t, img.RetagLatestDone)
}
