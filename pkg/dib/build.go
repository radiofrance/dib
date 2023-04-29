package dib

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/ratelimit"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/sirupsen/logrus"
)

// Rebuild iterates over the graph to rebuild all images that are marked to be rebuilt.
// It also collects the reports ant prints them to the user.
func Rebuild(
	graph *dag.DAG,
	builder types.ImageBuilder,
	testRunners []types.TestRunner,
	rateLimiter ratelimit.RateLimiter,
	placeholderTag string,
	localOnly bool,
	dibReport *report.Report,
) error {
	meta := LoadCommonMetadata(&exec.ShellExecutor{})
	reportChan := make(chan report.BuildReport)
	wgBuild := sync.WaitGroup{}

	graph.Walk(func(node *dag.Node) {
		if !node.Image.NeedsRebuild && !node.Image.NeedsTests {
			return
		}

		wgBuild.Add(1)
		go RebuildNode(node, builder, testRunners, rateLimiter, meta, placeholderTag, localOnly, &wgBuild, reportChan,
			dibReport.GetBuildLogsDir(), dibReport.GetJunitReportDir(), dibReport.GetTrivyReportDir())
	})

	go func() {
		wgBuild.Wait()
		close(reportChan)
	}()

	var reports []report.BuildReport
	for imageReport := range reportChan {
		reports = append(reports, imageReport)
	}

	dibReport.BuildReports = reports
	report.PrintReports(reports)
	if err := report.Generate(*dibReport, *graph); err != nil {
		return err
	}

	return report.CheckError(reports)
}

// RebuildNode build the image on the given node, and run tests if necessary.
func RebuildNode(
	node *dag.Node,
	builder types.ImageBuilder,
	testRunners []types.TestRunner,
	rateLimiter ratelimit.RateLimiter,
	meta ImageMetadata,
	placeholderTag string,
	localOnly bool,
	wgBuild *sync.WaitGroup,
	reportChan chan report.BuildReport,
	buildReportDir string,
	junitReportDir string,
	trivyReportDir string,
) {
	defer wgBuild.Done()

	node.WaitCond.L.Lock()
	defer func() {
		node.WaitCond.Broadcast()
		node.WaitCond.L.Unlock()
	}()

	img := node.Image
	buildReport := report.BuildReport{
		ImageName: img.ShortName,
	}

	// Wait for all parents to complete their build process
	for _, parent := range node.Parents() {
		parent.WaitCond.L.Lock()
		for parent.Image.NeedsRebuild && !(parent.Image.RebuildDone || parent.Image.RebuildFailed) {
			logrus.Debugf("Image %s is waiting on %s to complete", img.ShortName, parent.Image.ShortName)
			parent.WaitCond.Wait()
		}
		parent.WaitCond.L.Unlock()
	}

	// Return if any parent build failed
	for _, parent := range node.Parents() {
		if parent.Image.RebuildFailed {
			img.RebuildFailed = true
			buildReport.BuildStatus = report.BuildStatusSkipped
			buildReport.TestsStatus = report.TestsStatusSkipped
			reportChan <- buildReport
			return
		}
	}

	if img.NeedsRebuild && !img.RebuildDone {
		err := doRebuild(node, builder, rateLimiter, meta, placeholderTag, localOnly, buildReportDir)
		if err != nil {
			img.RebuildFailed = true
			reportChan <- buildReport.WithError(err)
			return
		}
		buildReport.BuildStatus = report.BuildStatusSuccess
	}

	if img.NeedsTests {
		buildReport.TestsStatus = report.TestsStatusPassed
		err := testImage(testRunners, types.RunTestOptions{
			ImageName:         img.ShortName,
			ImageReference:    img.CurrentRef(),
			DockerContextPath: img.Dockerfile.ContextPath,
			ReportJunitDir:    junitReportDir,
			ReportTrivyDir:    trivyReportDir,
		})
		if err != nil {
			buildReport.TestsStatus = report.TestsStatusFailed
			buildReport.FailureMessage = err.Error()
		}
	}

	reportChan <- buildReport
	img.RebuildDone = true
}

// doRebuild do the effective build action.
func doRebuild(
	node *dag.Node,
	builder types.ImageBuilder,
	rateLimiter ratelimit.RateLimiter,
	meta ImageMetadata,
	placeholderTag string,
	localOnly bool,
	buildReportDir string,
) error {
	rateLimiter.Acquire()
	defer rateLimiter.Release()

	img := node.Image

	// Before building the image, we need to replace all references to tags
	// of any dib-managed images used as dependencies in the Dockerfile.
	tagsToReplace := make(map[string]string)
	for _, parent := range node.Parents() {
		tagsToReplace[parent.Image.DockerRef(placeholderTag)] = parent.Image.CurrentRef()
	}
	if err := dockerfile.ReplaceTags(*img.Dockerfile, tagsToReplace); err != nil {
		return fmt.Errorf("failed to replace tag in dockerfile %s: %w", img.Dockerfile.ContextPath, err)
	}
	defer func() {
		if err := dockerfile.ResetTags(*img.Dockerfile, tagsToReplace); err != nil {
			logrus.Warnf("failed to reset tag in dockerfile %s: %v", img.Dockerfile.ContextPath, err)
		}
	}()

	if err := os.MkdirAll(buildReportDir, 0o755); err != nil {
		return fmt.Errorf("failed to create folder %s: %w", buildReportDir, err)
	}

	filePath := path.Join(buildReportDir, fmt.Sprintf("%s.txt", strings.ReplaceAll(img.ShortName, "/", "_")))
	fileOutput, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}

	opts := types.ImageBuilderOpts{
		Context: img.Dockerfile.ContextPath,
		Tags: []string{
			img.CurrentRef(),
		},
		Labels:    meta.WithImage(img).ToLabels(),
		Push:      !localOnly,
		LogOutput: fileOutput,
	}

	logrus.Infof("Building \"%s\" in context \"%s\"", img.CurrentRef(), img.Dockerfile.ContextPath)

	if err := builder.Build(opts); err != nil {
		return fmt.Errorf("building image %s failed: %w", img.ShortName, err)
	}

	return nil
}
