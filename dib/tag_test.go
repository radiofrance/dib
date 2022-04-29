package dib_test

import (
	"testing"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/dib"
	"github.com/radiofrance/dib/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	err := dib.Retag(DAG, tagger)

	assert.NoError(t, err)
	assert.Empty(t, tagger.RecordedCallsArgs)
}

func Test_Retag_DoesNotRetagIfAlreadyDone(t *testing.T) {
	t.Parallel()

	DAG := &dag.DAG{}
	DAG.AddNode(dag.NewNode(&dag.Image{
		Name:       "registry.example.org/image",
		ShortName:  "image",
		CurrentTag: "old",
		TargetTag:  "new",
		NeedsRetag: true,
		RetagDone:  true,
	}))

	tagger := &mock.Tagger{}
	err := dib.Retag(DAG, tagger)

	assert.NoError(t, err)
	assert.Empty(t, tagger.RecordedCallsArgs)
}

func Test_Retag_NoExtraLabels(t *testing.T) {
	t.Parallel()

	img := &dag.Image{
		Name:       "registry.example.org/image",
		ShortName:  "image",
		CurrentTag: "old",
		TargetTag:  "new",
		NeedsRetag: true,
		RetagDone:  false,
	}
	DAG := &dag.DAG{}
	DAG.AddNode(dag.NewNode(img))

	tagger := &mock.Tagger{}
	err := dib.Retag(DAG, tagger)

	assert.NoError(t, err)

	assert.Len(t, tagger.RecordedCallsArgs, 1)
	args := tagger.RecordedCallsArgs[0]
	assert.Equal(t, "registry.example.org/image:old", args.Src)
	assert.Equal(t, "registry.example.org/image:new", args.Dest)

	assert.True(t, img.RetagDone)
}

func Test_Retag_WithExtraLabels(t *testing.T) {
	t.Parallel()

	img := &dag.Image{
		Name:       "registry.example.org/image",
		ShortName:  "image",
		CurrentTag: "old",
		TargetTag:  "new",
		ExtraTags:  []string{"latest1", "latest2"},
		NeedsRetag: true,
		RetagDone:  false,
	}
	DAG := &dag.DAG{}
	DAG.AddNode(dag.NewNode(img))

	tagger := &mock.Tagger{}
	err := dib.Retag(DAG, tagger)

	assert.NoError(t, err)

	require.Len(t, tagger.RecordedCallsArgs, 3)

	assert.Equal(t, "registry.example.org/image:old", tagger.RecordedCallsArgs[0].Src)
	assert.Equal(t, "registry.example.org/image:new", tagger.RecordedCallsArgs[0].Dest)

	assert.Equal(t, "registry.example.org/image:new", tagger.RecordedCallsArgs[1].Src)
	assert.Equal(t, "registry.example.org/image:latest1", tagger.RecordedCallsArgs[1].Dest)
	assert.Equal(t, "registry.example.org/image:new", tagger.RecordedCallsArgs[2].Src)
	assert.Equal(t, "registry.example.org/image:latest2", tagger.RecordedCallsArgs[2].Dest)

	assert.True(t, img.RetagDone)
}
