package dib_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/radiofrance/dib/pkg/mock"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
)

func Test_Rebuild_NothingToDo(t *testing.T) {
	t.Parallel()

	builder := &mock.Builder{}
	var testRunners []types.TestRunner
	node := createNode()
	node.Image.NeedsRebuild = false
	node.Image.NeedsTests = false
	dibReport := createDibReport()

	reportChan := make(chan report.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, dib.ImageMetadata{},
		"DIB_MANAGED_VERSION", false, &wg, reportChan, dibReport)
	wg.Wait()
	close(reportChan)

	assert.Len(t, reportChan, 1)
	for reportItem := range reportChan {
		assert.Equal(t, report.BuildStatusSkipped, reportItem.BuildStatus)
		assert.Equal(t, report.TestsStatusSkipped, reportItem.TestsStatus)
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
	dibReport := createDibReport()

	reportChan := make(chan report.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, dib.ImageMetadata{},
		"DIB_MANAGED_VERSION", false, &wg, reportChan, dibReport)
	wg.Wait()
	close(reportChan)

	assert.Len(t, reportChan, 1)
	for reportItem := range reportChan {
		assert.Equal(t, report.BuildStatusSuccess, reportItem.BuildStatus)
		assert.Equal(t, report.TestsStatusPassed, reportItem.TestsStatus)
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
	dibReport := createDibReport()

	reportChan := make(chan report.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, dib.ImageMetadata{},
		"DIB_MANAGED_VERSION", false, &wg, reportChan, dibReport)
	wg.Wait()
	close(reportChan)

	assert.Len(t, reportChan, 1)
	for reportItem := range reportChan {
		assert.Equal(t, report.BuildStatusSkipped, reportItem.BuildStatus)
		assert.Equal(t, report.TestsStatusPassed, reportItem.TestsStatus)
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
	dibReport := createDibReport()

	reportChan := make(chan report.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, dib.ImageMetadata{},
		"DIB_MANAGED_VERSION", false, &wg, reportChan, dibReport)
	wg.Wait()
	close(reportChan)

	assert.Len(t, reportChan, 1)
	for reportItem := range reportChan {
		assert.Equal(t, report.BuildStatusSkipped, reportItem.BuildStatus)
		assert.Equal(t, report.TestsStatusPassed, reportItem.TestsStatus)
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
	dibReport := createDibReport()

	reportChan := make(chan report.BuildReport, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	dib.RebuildNode(node, builder, testRunners, mock.RateLimiter{}, dib.ImageMetadata{},
		"DIB_MANAGED_VERSION", false, &wg, reportChan, dibReport)
	wg.Wait()
	close(reportChan)

	assert.Len(t, reportChan, 1)
	for reportItem := range reportChan {
		assert.Equal(t, report.BuildStatusSkipped, reportItem.BuildStatus)
		assert.Equal(t, report.TestsStatusFailed, reportItem.TestsStatus)
		assert.Equal(t, "mock test failed", reportItem.FailureMessage)
	}

	assert.Equal(t, 0, builder.CallCount)
}

func createNode() *dag.Node {
	return dag.NewNode(&dag.Image{
		Name:      "eu.gcr.io/my-test-repository/test",
		ShortName: "test",
		Dockerfile: &dockerfile.Dockerfile{
			ContextPath: "../../test/fixtures/build",
			Filename:    "Dockerfile",
		},
	})
}

func createDibReport() *report.Report {
	dibReport := report.InitDibReport("reports", "1.0.0")
	return dibReport
}
