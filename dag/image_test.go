package dag_test

import (
	"sync"
	"testing"

	"github.com/radiofrance/dib/types"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/radiofrance/dib/dockerfile"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/mock"
)

func Test_RebuildRefExists(t *testing.T) {
	t.Parallel()

	registry := &mock.Registry{}
	builder := &mock.Builder{}
	img := createImage(registry, builder, nil)

	errChan := make(chan error, 1)
	img.Rebuild("new-123", false, true, false, errChan)
	assert.Equal(t, 1, registry.RefExistsCallCount)
	assert.Equal(t, 0, registry.RetagCallCount)
	assert.Equal(t, 0, builder.CallCount)
}

func Test_RebuildForce(t *testing.T) {
	t.Parallel()

	registry := &mock.Registry{}
	builder := &mock.Builder{}
	img := createImage(registry, builder, nil)

	errChan := make(chan error, 1)
	img.Rebuild("new-123", true, false, false, errChan)

	err := <-errChan
	require.NoError(t, err)

	assert.Equal(t, 1, registry.RefExistsCallCount)
	assert.Equal(t, 0, registry.RetagCallCount)
	assert.Equal(t, 1, builder.CallCount)
}

func TestImage_runTests(t *testing.T) {
	t.Parallel()

	registry := &mock.Registry{}
	builder := &mock.Builder{}
	tester := &mock.TestRunner{}
	img := createImage(registry, builder, tester)

	errChan := make(chan error, 1)
	img.Rebuild("new-123", true, true, false, errChan)

	err := <-errChan
	require.NoError(t, err)

	assert.Equal(t, 1, registry.RefExistsCallCount)
	assert.Equal(t, 0, registry.RetagCallCount)
	assert.Equal(t, 1, builder.CallCount)
}

func createImage(registry *mock.Registry, builder *mock.Builder, tester *mock.TestRunner) dag.Image {
	if registry == nil {
		registry = &mock.Registry{}
	}
	if builder == nil {
		builder = &mock.Builder{}
	}
	if tester == nil {
		tester = &mock.TestRunner{}
	}

	return dag.Image{
		Name:      "eu.gcr.io/my-test-repository/test",
		ShortName: "test",
		Dockerfile: &dockerfile.Dockerfile{
			ContextPath: "../test/fixtures",
			Filename:    "Dockerfile",
		},
		NeedsRebuild: true,
		RebuildCond:  sync.NewCond(&sync.Mutex{}),
		Registry:     registry,
		Builder:      builder,
		TestRunners:  []types.TestRunner{tester},
	}
}
