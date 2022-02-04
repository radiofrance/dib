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
	rateLimiter ratelimit.RateLimiter, localOnly bool,
) error {
	reportChan := make(chan BuildReport)
	wgBuild := sync.WaitGroup{}

	graph.Walk(func(node *dag.Node) {
		if !node.Image.NeedsRebuild && !node.Image.NeedsTests {
			return
		}

		wgBuild.Add(1)
		go RebuildNode(node, builder, testRunners, rateLimiter, localOnly, &wgBuild, reportChan)
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
	rateLimiter ratelimit.RateLimiter, localOnly bool, wg *sync.WaitGroup, reportChan chan BuildReport,
) {
	defer wg.Done()

	node.WaitCond.L.Lock()
	defer func() {
		node.WaitCond.Broadcast()
		node.WaitCond.L.Unlock()
	}()

	img := node.Image
	report := BuildReport{
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
			report.BuildStatus = BuildStatusSkipped
			report.TestsStatus = TestsStatusSkipped
			reportChan <- report
			return
		}
	}

	if img.NeedsRebuild && !img.RebuildDone {
		err := doRebuild(node, builder, rateLimiter, localOnly)
		if err != nil {
			img.RebuildFailed = true
			reportChan <- report.withError(err)
			return
		}
		report.BuildStatus = BuildStatusSuccess
	}

	if img.NeedsTests {
		report.TestsStatus = TestsStatusPassed
		if err := testImage(img, testRunners, img.TargetTag); err != nil {
			report.TestsStatus = TestsStatusFailed
			report.FailureMessage = err.Error()
		}
	}

	reportChan <- report
	img.RebuildDone = true
}

// doRebuild do the effective build action.
func doRebuild(node *dag.Node, builder types.ImageBuilder, rateLimiter ratelimit.RateLimiter, localOnly bool) error {
	rateLimiter.Acquire()
	defer rateLimiter.Release()

	img := node.Image

	// Before building the image, we need to replace all references to tags
	// of any dib-managed images used as dependencies in the Dockerfile.
	tagsToReplace := make(map[string]string)
	for _, parent := range node.Parents() {
		if parent.Image.NeedsRebuild {
			// The parent image was rebuilt, we have to use its new tag.
			tagsToReplace[parent.Image.Name] = parent.Image.DockerRef(parent.Image.TargetTag)
			continue
		}

		// The parent image has not changed, we can use the existing tag.
		tagsToReplace[parent.Image.Name] = parent.Image.DockerRef(parent.Image.CurrentTag)
	}
	if err := dockerfile.ReplaceTags(*img.Dockerfile, tagsToReplace); err != nil {
		return fmt.Errorf("failed to replace tag in dockerfile %s: %w", img.Dockerfile.ContextPath, err)
	}
	defer func() {
		if err := dockerfile.ResetTags(*img.Dockerfile, tagsToReplace); err != nil {
			logrus.Warnf("failed to reset tag in dockerfile %s: %v", img.Dockerfile.ContextPath, err)
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
	filePath := path.Join("dist/logs", fmt.Sprintf("%s.txt", strings.ReplaceAll(img.ShortName, "/", "_")))
	fileOutput, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}

	opts := types.ImageBuilderOpts{
		Context:   img.Dockerfile.ContextPath,
		Tag:       fmt.Sprintf("%s:%s", img.Name, img.TargetTag),
		Labels:    labels,
		Push:      !localOnly,
		LogOutput: fileOutput,
	}

	logrus.Infof("Building \"%s:%s\" in context \"%s\"", img.Name, img.TargetTag, img.Dockerfile.ContextPath)

	if err := builder.Build(opts); err != nil {
		return fmt.Errorf("building image %s failed: %w", img.ShortName, err)
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
