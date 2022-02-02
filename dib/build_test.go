package dib_test

import (
	"sync"
	"testing"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/dib"
	"github.com/radiofrance/dib/dockerfile"
	"github.com/radiofrance/dib/mock"
	"github.com/radiofrance/dib/types"
	"github.com/stretchr/testify/assert"
)

var testRunners []types.TestRunner

func Test_Rebuild_NothingToDo(t *testing.T) {
	t.Parallel()

	builder := &mock.Builder{}
	node := createNode()
	node.Image.NeedsRebuild = false
	node.Image.NeedsTests = false

	reportChan := make(chan dib.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, "new-123", false, &wg, reportChan)
	wg.Wait()
	close(reportChan)

	for report := range reportChan {
		assert.Equal(t, dib.BuildStatusSkipped, report.BuildStatus)
		assert.Equal(t, dib.TestsStatusSkipped, report.TestsStatus)
	}

	assert.Equal(t, 0, builder.CallCount)
}

func Test_Rebuild_BuildAndTest(t *testing.T) {
	t.Parallel()

	builder := &mock.Builder{}
	node := createNode()
	node.Image.NeedsRebuild = true
	node.Image.NeedsTests = true

	reportChan := make(chan dib.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, "new-123", false, &wg, reportChan)
	wg.Wait()
	close(reportChan)

	for report := range reportChan {
		assert.Equal(t, dib.BuildStatusSuccess, report.BuildStatus)
		assert.Equal(t, dib.TestsStatusPassed, report.TestsStatus)
	}

	assert.Equal(t, 1, builder.CallCount)
}

func Test_Rebuild_TestOnly(t *testing.T) {
	t.Parallel()

	builder := &mock.Builder{}
	node := createNode()
	node.Image.NeedsRebuild = false
	node.Image.NeedsTests = true

	reportChan := make(chan dib.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, "new-123", false, &wg, reportChan)
	wg.Wait()
	close(reportChan)

	for report := range reportChan {
		assert.Equal(t, dib.BuildStatusSkipped, report.BuildStatus)
		assert.Equal(t, dib.TestsStatusPassed, report.TestsStatus)
	}

	assert.Equal(t, 0, builder.CallCount)
}

func createNode() *dag.Node {
	return dag.NewNode(&dag.Image{
		Name:      "eu.gcr.io/my-test-repository/test",
		ShortName: "test",
		Dockerfile: &dockerfile.Dockerfile{
			ContextPath: "../test/fixtures",
			Filename:    "Dockerfile",
		},
	})
}
