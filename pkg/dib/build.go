package dib

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/pterm/pterm"
	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/ratelimit"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/types"
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
	buildGraph := graph.Filter(func(node *dag.Node) bool {
		return node.Image.NeedsRebuild || node.Image.NeedsTests
	})

	var sortedImages []string
	buildGraph.WalkInDepth(func(node *dag.Node) {
		if node.Image.NeedsRebuild {
			sortedImages = append(sortedImages, node.Image.Name)
		}
	})

	sort.Strings(sortedImages)
	var list []pterm.BulletListItem
	for _, image := range sortedImages {
		list = append(list, pterm.BulletListItem{
			Level: 0,
			Text:  image,
		})
	}

	if err := pterm.DefaultBulletList.WithItems(list).Render(); err != nil {
		logger.Errorf("failed to print DAG: %s", err)
	}

	progressBar, _ := pterm.DefaultProgressbar.
		WithTotal(len(sortedImages)).
		WithTitle("Build images").
		Start()

	meta := LoadCommonMetadata(&exec.ShellExecutor{})
	reportChan := make(chan report.BuildReport)
	go func() {
		buildGraph.WalkParallel(func(node *dag.Node) {
			reportChan <- RebuildNode(node, builder, testRunners, rateLimiter, meta, placeholderTag, localOnly,
				dibReport.GetBuildLogsDir(), dibReport.GetJunitReportDir(), dibReport.GetTrivyReportDir())
			progressBar.Increment()
		})
		close(reportChan)
	}()

	for buildReport := range reportChan {
		dibReport.BuildReports = append(dibReport.BuildReports, buildReport)
	}

	dibReport.Print()
	if err := report.Generate(*dibReport, *graph); err != nil {
		return err
	}

	return dibReport.CheckError()
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
	buildReportDir string,
	junitReportDir string,
	trivyReportDir string,
) report.BuildReport {
	img := node.Image
	buildReport := report.BuildReport{Image: *img}

	// Return if any parent build failed
	for _, parent := range node.Parents() {
		if !parent.Image.RebuildFailed {
			continue
		}
		img.RebuildFailed = true
		buildReport.BuildStatus = report.BuildStatusSkipped
		buildReport.TestsStatus = report.TestsStatusSkipped

		return buildReport
	}

	if img.NeedsRebuild {
		err := doRebuild(node, builder, rateLimiter, meta, placeholderTag, localOnly, buildReportDir)
		if err != nil {
			img.RebuildFailed = true
			return buildReport.WithError(err)
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

	return buildReport
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
			logger.Warnf("failed to reset tag in dockerfile %s: %v", img.Dockerfile.ContextPath, err)
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

	logger.Infof("Building \"%s\" in context \"%s\"", img.CurrentRef(), img.Dockerfile.ContextPath)

	if err := builder.Build(opts); err != nil {
		return fmt.Errorf("building image %s failed: %w", img.ShortName, err)
	}

	return nil
}
