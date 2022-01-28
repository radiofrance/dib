package dag

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/radiofrance/dib/dockerfile"
	"github.com/radiofrance/dib/ratelimit"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
)

const (
	latest = "latest"
)

type Image struct {
	Name            string
	ShortName       string
	Dockerfile      *dockerfile.Dockerfile
	Children        []*Image
	Parents         []*Image
	NeedsRebuild    bool
	RetagDone       bool
	RetagLatestDone bool
	RebuildDone     bool
	RebuildCond     *sync.Cond
	Registry        types.DockerRegistry
	Builder         types.ImageBuilder
	Tagger          types.ImageTagger
	TestRunners     []types.TestRunner
	RateLimiter     ratelimit.RateLimiter
}

// Rebuild iterates over the graph to rebuild each image that is tagged for rebuild.
func (img *Image) Rebuild(newTag string, forceRebuild, disableRunTests, localOnly bool,
	wgBuilds *sync.WaitGroup, reportChan *chan BuildReport) {
	defer wgBuilds.Done()

	report := BuildReport{
		ImageName: img.ShortName,
	}

	refAlreadyExists, err := img.Registry.RefExists(img.dockerRef(newTag))
	if err != nil {
		*reportChan <- report.withError(err)
		return
	}

	img.RebuildCond.L.Lock()

	if img.NeedsRebuild && !img.RebuildDone {
		for _, parent := range img.Parents {
			if parent.NeedsRebuild && !parent.RebuildDone {
				parent.RebuildCond.Wait()
			}
		}

		if refAlreadyExists && !forceRebuild && !localOnly {
			logrus.Debugf("Image \"%s\" is tagued for rebuild but ref is already present on the registry, skipping."+
				" if you want to rebuild anyway, use --force-rebuild", img.Name)

			*reportChan <- report
			img.RebuildDone = true
			img.RebuildCond.Broadcast()
			return
		}

		err := img.doRebuild(newTag, localOnly)
		if err != nil {
			*reportChan <- report.withError(err)
			return
		}
		report.BuildStatus = BuildStatusSuccess

		if !disableRunTests {
			err = img.doTest(newTag, disableRunTests)
			if err != nil {
				report.TestsStatus = TestsStatusFailed
			}
			report.TestsStatus = TestsStatusPassed
		}

		*reportChan <- report
		img.RebuildDone = true
		img.RebuildCond.Broadcast()
	}

	img.RebuildCond.L.Unlock()

	wgBuilds.Add(len(img.Children))
	for _, child := range img.Children {
		go child.Rebuild(newTag, forceRebuild, disableRunTests, localOnly, wgBuilds, reportChan)
	}
}

// doRebuild do the effective build action.
func (img *Image) doRebuild(newTag string, localOnly bool) error {
	img.RateLimiter.Acquire()
	defer img.RateLimiter.Release()

	logrus.Infof("Building \"%s:%s\" in context \"%s\"", img.Name, newTag, img.Dockerfile.ContextPath)

	if err := dockerfile.ReplaceFromTag(*img.Dockerfile, newTag); err != nil {
		return fmt.Errorf("failed to replace tag in dockerfile %s: %w", img.Dockerfile.ContextPath, err)
	}
	defer func() {
		if err := dockerfile.ResetFromTag(*img.Dockerfile, newTag); err != nil {
			logrus.Errorf("failed to reset tag in dockerfile %s: %v", img.Dockerfile.ContextPath, err)
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
	filePath := path.Join("dist/logs", fmt.Sprintf("%s.log", img.ShortName))
	fileOutput, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}

	opts := types.ImageBuilderOpts{
		Context:   img.Dockerfile.ContextPath,
		Tag:       fmt.Sprintf("%s:%s", img.Name, newTag),
		Labels:    labels,
		Push:      !localOnly,
		LogOutput: fileOutput,
	}

	if err := img.Builder.Build(opts); err != nil {
		return fmt.Errorf("building image %s failed: %w", img.ShortName, err)
	}

	return nil
}

// doTest run the tests without building anything.
func (img *Image) doTest(newTag string, disableRunTests bool) error {
	if disableRunTests {
		return nil
	}

	logrus.Infof("Running tests for \"%s:%s\"", img.Name, newTag)
	return img.runTests(fmt.Sprintf("%s:%s", img.Name, newTag))
}

func findSource() string {
	if url := os.Getenv("CI_PROJECT_URL"); url != "" { // gitlab predefined variable
		return url
	} else if url := os.Getenv("GITHUB_REPOSITORY"); url != "" { // github predefined variable
		return fmt.Sprintf("https://github.com/%s", url)
	}
	return ""
}

func findAuthors() string {
	if authors := os.Getenv("GITLAB_USER_NAME"); authors != "" { // gitlab predefined variable
		return authors
	} else if authors := os.Getenv("GITHUB_ACTOR"); authors != "" { // github predefined variable
		return authors
	}
	return ""
}

func findRevision() string {
	if rev := os.Getenv("CI_COMMIT_SHA"); rev != "" { // gitlab predefined variable
		return rev
	} else if rev := os.Getenv("GITHUB_SHA"); rev != "" { // github predefined variable
		return rev
	}
	return ""
}

// runTests run docker tests for each TestRunner.
func (img *Image) runTests(ref string) error {
	for _, runner := range img.TestRunners {
		if err := runner.RunTest(types.RunTestOptions{
			ImageName:         img.ShortName,
			ImageReference:    ref,
			DockerContextPath: img.Dockerfile.ContextPath,
		}); err != nil {
			return err
		}
	}
	return nil
}

// retagLatest iterates over the graph to retag each image with the latest tag.
func (img *Image) retagLatest(tag string) error {
	if !img.RetagLatestDone {
		logrus.Debugf("Retag latest tag for image %s with version %s", img.Name, tag)
		if err := img.Tagger.Tag(img.dockerRef(tag), img.dockerRef(latest)); err != nil {
			return err
		}
		img.RetagLatestDone = true
	}
	for _, child := range img.Children {
		err := child.retagLatest(tag)
		if err != nil {
			return err
		}
	}
	return nil
}

// retag iterates over the graph to retag each image that is not tagged for rebuild.
func (img *Image) retag(newTag, oldTag string) error {
	if img.NeedsRebuild {
		// If this image needs rebuild, then its children too, no need to go deeper
		return nil
	}
	if !img.RetagDone {
		err := img.doRetag(newTag, oldTag)
		if err != nil {
			return err
		}
	}
	for _, child := range img.Children {
		err := child.retag(newTag, oldTag)
		if err != nil {
			return err
		}
	}
	return nil
}

// doRetag do the effective retag action.
func (img *Image) doRetag(newTag, oldTag string) error {
	currentTagExists, err := img.Registry.RefExists(img.dockerRef(newTag))
	if err != nil {
		return err
	}
	previousTagExists, err := img.Registry.RefExists(img.dockerRef(oldTag))
	if err != nil {
		return err
	}

	if currentTagExists {
		logrus.Debugf("Current tag for \"%s:%s\", already exists, nothing to do", img.Name, newTag)
		return nil
	}
	if previousTagExists {
		if err := img.Tagger.Tag(img.dockerRef(oldTag), img.dockerRef(newTag)); err != nil {
			return err
		}
		img.RetagDone = true
		return nil
	}
	logrus.Warnf("Previous tag \"%s:%s\" missing, image will be rebuilt", img.Name, oldTag)
	img.tagForRebuild()
	return nil
}

func (img *Image) dockerRef(version string) string {
	return fmt.Sprintf("%s:%s", img.Name, version)
}

// tagForRebuild will set the `Rebuild` flag on the Image to true.
// It will also do it recursively for all its children.
func (img *Image) tagForRebuild() {
	img.NeedsRebuild = true
	for _, child := range img.Children {
		child.tagForRebuild()
	}
}

// checkDiffRecursive will do a recursive, depth-first search in child images and uses the diffBelongsTo map
// to mark diff files with the image they belong to.
// If a file in the diff already belongs to an image, or if it doesn't belong to an image at all, it is left unchanged.
func (img *Image) checkDiffRecursive(diffs []string, diffBelongsTo map[string]*Image) {
	// Depth-first search.
	for _, child := range img.Children {
		child.checkDiffRecursive(diffs, diffBelongsTo)
	}

	for _, file := range diffs {
		if !strings.HasPrefix(file, img.Dockerfile.ContextPath) {
			// The current file is not lying in the current image build context, nor in a subdirectory.
			continue
		}

		if diffBelongsTo[file] != nil {
			// The current file has already been assigned to an image, most likely to a child image.
			continue
		}

		// If we reach here, the diff file is part of the current image's context, we mark it as so.
		diffBelongsTo[file] = img
	}
}
