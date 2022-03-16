package dib

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/dockerfile"
	"github.com/radiofrance/dib/ratelimit"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
)

// Rebuild iterates over the graph to rebuild all images that are marked to be rebuilt.
// It also collects the reports ant prints them to the user.
func Rebuild(graph *dag.DAG, builder types.ImageBuilder, testRunners []types.TestRunner,
	rateLimiter ratelimit.RateLimiter, newTag string, localOnly bool,
) error {
	reportChan := make(chan BuildReport)
	wgBuild := sync.WaitGroup{}

	graph.Walk(func(node *dag.Node) {
		if !node.Image.GetNeedsRebuild() && !node.Image.GetNeedsTests() {
			return
		}

		wgBuild.Add(1)
		go RebuildNode(node, builder, testRunners, rateLimiter, newTag, localOnly, &wgBuild, reportChan)
	})

	go func() {
		wgBuild.Wait()
		close(reportChan)
	}()

	var reports []BuildReport
	for report := range reportChan {
		reports = append(reports, report)
	}
	printReports(reports)

	return checkError(reports)
}

// RebuildNode build the image on the given node, and run tests if necessary.
func RebuildNode(node *dag.Node, builder types.ImageBuilder, testRunners []types.TestRunner,
	rateLimiter ratelimit.RateLimiter, newTag string, localOnly bool, wg *sync.WaitGroup, reportChan chan BuildReport,
) {
	defer wg.Done()

	node.WaitCond.L.Lock()
	defer func() {
		node.WaitCond.Broadcast()
		node.WaitCond.L.Unlock()
	}()

	img := node.Image
	report := BuildReport{
		ImageName: img.GetShortName(),
	}

	// Wait for all parents to complete their build process
	for _, parent := range node.Parents() {
		parent.WaitCond.L.Lock()
		for parent.Image.GetNeedsRebuild() && !(parent.Image.GetRebuildDone() || parent.Image.GetRebuildFailed()) {
			logrus.Debugf("Image %s is waiting on %s to complete", img.GetShortName(), parent.Image.GetShortName())
			parent.WaitCond.Wait()
		}
		parent.WaitCond.L.Unlock()
	}

	// Return if any parent build failed
	for _, parent := range node.Parents() {
		if parent.Image.GetRebuildFailed() {
			img.SetRebuildFailed(true)
			report.BuildStatus = BuildStatusSkipped
			report.TestsStatus = TestsStatusSkipped
			reportChan <- report
			return
		}
	}

	if img.GetNeedsRebuild() && !img.GetRebuildDone() {
		err := doRebuild(img, builder, rateLimiter, newTag, localOnly)
		if err != nil {
			img.SetRebuildFailed(true)
			reportChan <- report.withError(err)
			return
		}
		report.BuildStatus = BuildStatusSuccess
	}

	if img.GetNeedsTests() {
		report.TestsStatus = TestsStatusPassed
		if err := testImage(img, testRunners, newTag); err != nil {
			report.TestsStatus = TestsStatusFailed
			report.FailureMessage = err.Error()
		}
	}

	reportChan <- report
	img.SetRebuildDone(true)
}

// doRebuild do the effective build action.
func doRebuild(img *dag.Image, builder types.ImageBuilder, rateLimiter ratelimit.RateLimiter,
	newTag string, localOnly bool,
) error {
	rateLimiter.Acquire()
	defer rateLimiter.Release()

	logrus.Infof("Building \"%s:%s\" in context \"%s\"", img.GetName(), newTag, img.GetDockerfile().ContextPath)

	if err := dockerfile.ReplaceFromTag(*img.GetDockerfile(), newTag); err != nil {
		return fmt.Errorf("failed to replace tag in dockerfile %s: %w", img.GetDockerfile().ContextPath, err)
	}
	defer func() {
		if err := dockerfile.ResetFromTag(*img.GetDockerfile(), newTag); err != nil {
			logrus.Errorf("failed to reset tag in dockerfile %s: %v", img.GetDockerfile().ContextPath, err)
		}
	}()

	now := time.Now()
	labels := map[string]string{
		"org.opencontainers.image.created": now.Format(time.RFC3339),
	}
	if rev := findRevision(); rev != "" {
		labels["org.opencontainers.image.revision"] = rev
	}
	if authors := findAuthors(); authors != "" {
		labels["org.opencontainers.image.authors"] = authors
	}
	if source := findSource(); source != "" {
		labels["org.opencontainers.image.source"] = source
	}

	if err := os.MkdirAll("dist/logs", 0o755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", "dist/logs", err)
	}
	filePath := path.Join("dist/logs", fmt.Sprintf("%s.txt", strings.ReplaceAll(img.GetShortName(), "/", "_")))
	fileOutput, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}

	opts := types.ImageBuilderOpts{
		Context:   img.GetDockerfile().ContextPath,
		Tag:       fmt.Sprintf("%s:%s", img.GetName(), newTag),
		Labels:    labels,
		Push:      !localOnly,
		LogOutput: fileOutput,
	}

	if err := builder.Build(opts); err != nil {
		return fmt.Errorf("building image %s failed: %w", img.GetShortName(), err)
	}

	return nil
}

func findSource() string {
	if url := os.Getenv("CI_PROJECT_URL"); url != "" { // gitlab predefined variable
		return url
	}
	if url := os.Getenv("GITHUB_REPOSITORY"); url != "" { // github predefined variable
		return fmt.Sprintf("https://github.com/%s", url)
	}
	return ""
}

func findAuthors() string {
	if authors := os.Getenv("GITLAB_USER_NAME"); authors != "" { // gitlab predefined variable
		return authors
	}
	if authors := os.Getenv("GITHUB_ACTOR"); authors != "" { // github predefined variable
		return authors
	}
	return ""
}

func findRevision() string {
	if rev := os.Getenv("CI_COMMIT_SHA"); rev != "" { // gitlab predefined variable
		return rev
	}
	if rev := os.Getenv("GITHUB_SHA"); rev != "" { // github predefined variable
		return rev
	}
	return ""
}
