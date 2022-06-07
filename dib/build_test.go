package dib_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/dib"
	"github.com/radiofrance/dib/dockerfile"
	"github.com/radiofrance/dib/mock"
	"github.com/radiofrance/dib/types"
	"github.com/stretchr/testify/assert"
)

func Test_Rebuild_NothingToDo(t *testing.T) {
	t.Parallel()

	builder := &mock.Builder{}
	var testRunners []types.TestRunner
	node := createNode()
	node.Image.NeedsRebuild = false
	node.Image.NeedsTests = false

	reportChan := make(chan dib.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, "DIB_MANAGED_VERSION", false, &wg, reportChan)
	wg.Wait()
	close(reportChan)

	assert.Len(t, reportChan, 1)
	for report := range reportChan {
		assert.Equal(t, dib.BuildStatusSkipped, report.BuildStatus)
		assert.Equal(t, dib.TestsStatusSkipped, report.TestsStatus)
	}

	assert.Equal(t, 0, builder.CallCount)
}

func Test_Rebuild_BuildAndTest(t *testing.T) {
	t.Parallel()

	builder := &mock.Builder{}
	var testRunners []types.TestRunner
	node := createNode()
	node.Image.NeedsRebuild = true
	node.Image.NeedsTests = true

	reportChan := make(chan dib.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, "DIB_MANAGED_VERSION", false, &wg, reportChan)
	wg.Wait()
	close(reportChan)

	assert.Len(t, reportChan, 1)
	for report := range reportChan {
		assert.Equal(t, dib.BuildStatusSuccess, report.BuildStatus)
		assert.Equal(t, dib.TestsStatusPassed, report.TestsStatus)
	}

	assert.Equal(t, 1, builder.CallCount)
}

func Test_Rebuild_TestOnly(t *testing.T) {
	t.Parallel()

	builder := &mock.Builder{}
	var testRunners []types.TestRunner
	node := createNode()
	node.Image.NeedsRebuild = false
	node.Image.NeedsTests = true

	reportChan := make(chan dib.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, "DIB_MANAGED_VERSION", false, &wg, reportChan)
	wg.Wait()
	close(reportChan)

	assert.Len(t, reportChan, 1)
	for report := range reportChan {
		assert.Equal(t, dib.BuildStatusSkipped, report.BuildStatus)
		assert.Equal(t, dib.TestsStatusPassed, report.TestsStatus)
	}

	assert.Equal(t, 0, builder.CallCount)
}

func Test_Rebuild_TestNotSupported(t *testing.T) {
	t.Parallel()

	builder := &mock.Builder{}
	testRunners := []types.TestRunner{&mock.TestRunner{
		ExpectedError: fmt.Errorf("mock test failed"),
		ShouldSupport: false,
	}}
	node := createNode()
	node.Image.NeedsRebuild = false
	node.Image.NeedsTests = true

	reportChan := make(chan dib.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, "DIB_MANAGED_VERSION", false, &wg, reportChan)
	wg.Wait()
	close(reportChan)

	assert.Len(t, reportChan, 1)
	for report := range reportChan {
		assert.Equal(t, dib.BuildStatusSkipped, report.BuildStatus)
		assert.Equal(t, dib.TestsStatusPassed, report.TestsStatus)
	}

	assert.Equal(t, 0, builder.CallCount)
}

func Test_Rebuild_TestError(t *testing.T) {
	t.Parallel()

	testRunners := []types.TestRunner{&mock.TestRunner{
		ExpectedError: fmt.Errorf("mock test failed"),
		ShouldSupport: true,
	}}

	builder := &mock.Builder{}
	node := createNode()
	node.Image.NeedsRebuild = false
	node.Image.NeedsTests = true

	reportChan := make(chan dib.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, "DIB_MANAGED_VERSION", false, &wg, reportChan)
	wg.Wait()
	close(reportChan)

	assert.Len(t, reportChan, 1)
	for report := range reportChan {
		assert.Equal(t, dib.BuildStatusSkipped, report.BuildStatus)
		assert.Equal(t, dib.TestsStatusFailed, report.TestsStatus)
		assert.Equal(t, "mock test failed", report.FailureMessage)
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
