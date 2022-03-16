package dib

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/radiofrance/dib/ratelimit"

	"github.com/docker/docker/pkg/fileutils"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
)

const dockerignore = ".dockerignore"

// Plan decides which actions need to be performed on each image.
func Plan(graph *dag.DAG, registry types.DockerRegistry, rateLimiter ratelimit.RateLimiter,
	diffs []string, oldTag, newTag string, forceRebuild, testsEnabled bool,
) error {
	if forceRebuild {
		logrus.Info("force rebuild mode enabled, all images will be rebuild regardless of their changes")
		graph.Walk(func(node *dag.Node) {
			node.Image.NeedsRebuild = true
			node.Image.NeedsTests = testsEnabled
		})
		return nil
	}
	checkDiff(graph, diffs)

	err := checkAlreadyBuilt(graph, registry, rateLimiter, newTag)
	if err != nil {
		return err
	}

	err = checkNeedsRetag(graph, registry, rateLimiter, oldTag, newTag)
	if err != nil {
		return err
	}

	if !testsEnabled {
		return nil
	}

	// Enable tests on images that need to be rebuilt.
	graph.Walk(func(node *dag.Node) {
		if node.Image.NeedsRebuild {
			node.Image.NeedsTests = true
		}
	})
	return nil
}

// checkDiff checks the diffs and marks images to be rebuilt if files in their context have changed.
func checkDiff(graph *dag.DAG, diffs []string) {
	diffBelongsTo := map[string]*dag.Node{}
	for _, file := range diffs {
		diffBelongsTo[file] = nil
	}

	// First, we do a depth-first search in the image graph to check if the files in diff belong to an image,
	// or is dockerignored
	// We start from the most specific image paths (children of children of children...), and we get back up
	// to parent images, to avoid false-positive and false-negative matches.
	graph.WalkInDepth(func(node *dag.Node) {
		for _, file := range diffs {
			if !strings.HasPrefix(file, node.Image.Dockerfile.ContextPath) {
				// The current file is not lying in the current image build context, nor in a subdirectory.
				continue
			}

			if diffBelongsTo[file] != nil {
				// The current file has already been assigned to an image, most likely to a child image.
				continue
			}

			if path.Base(file) == dockerignore {
				// We ignore dockerignore file itself for simplicity
				// In the real world, this file should not be ignored but it
				// helps us in managing refactoring
				continue
			}

			if node.Image.IgnorePatterns != nil {
				if matchPattern(node, file) {
					// The current file matches a pattern in the dockerignore file
					continue
				}
			}

			// If we reach here, the diff file is part of the current image's context, we mark it as so.
			diffBelongsTo[file] = node
		}
	})

	for file, node := range diffBelongsTo {
		if node != nil {
			logrus.Debugf("Image \"%s\" needs a rebuild because file \"%s\" has changed", node.Image.Name, file)
			// Mark image and all its children for rebuild.
			node.Walk(tagForRebuildFunc)
		}
	}
}

func matchPattern(node *dag.Node, file string) bool {
	ignorePatternMatcher, err := fileutils.NewPatternMatcher(node.Image.IgnorePatterns)
	if err != nil {
		logrus.Errorf("Could not create pattern matcher for %s, ignoring", node.Image.ShortName)
		return false
	}

	prefix := strings.TrimPrefix(strings.TrimPrefix(file, node.Image.Dockerfile.ContextPath), "/")
	match, err := ignorePatternMatcher.Matches(prefix)
	if err != nil {
		logrus.Errorf("Could not match pattern for %s, ignoring", node.Image.ShortName)
		return false
	}
	return match
}

func tagForRebuildFunc(node *dag.Node) {
	node.Image.Locker.Lock()
	node.Image.NeedsRebuild = true
	node.Image.Locker.Unlock()
}

// checkAlreadyBuilt iterates over the graph to find out which images
// already exist in the new version, so they don't need to be built again.
func checkAlreadyBuilt(graph *dag.DAG, registry types.DockerRegistry,
	rateLimiter ratelimit.RateLimiter, newTag string) error {
	errChan := make(chan error)
	wgBuild := sync.WaitGroup{}

	graph.Walk(func(node *dag.Node) {
		img := node.Image
		img.Locker.Lock()
		if !img.NeedsRebuild {
			img.Locker.Unlock()
			// Don't rebuild images that didn't change since last built revision.
			return
		}
		img.Locker.Unlock()

		wgBuild.Add(1)
		go checkNodeAlreadyBuilt(&wgBuild, rateLimiter, registry, node, newTag, errChan)
	})

	go func() {
		wgBuild.Wait()
		close(errChan)
	}()

	hasError := false
	for err := range errChan {
		hasError = true
		logrus.Errorf("Error checking image refs on registry: %v", err)
	}

	if hasError {
		return fmt.Errorf("one of the registry actions failed, see logs for more details")
	}
	return nil
}

func checkNodeAlreadyBuilt(wgBuild *sync.WaitGroup, rateLimiter ratelimit.RateLimiter,
	registry types.DockerRegistry, node *dag.Node, newTag string, errChan chan error) {
	defer wgBuild.Done()
	rateLimiter.Acquire()
	defer rateLimiter.Release()
	// Let's check on the registry if the new tag exists
	img := node.Image
	refAlreadyExists, err := registry.RefExists(img.DockerRef(newTag))
	if err != nil {
		errChan <- err
		return
	}
	if refAlreadyExists {
		// Looks like dib has already built this image in a previous run,
		// we can skip the build, but we don't want to disable the tests.
		// This is to avoid the situation where the tests failed on previous dib run,
		// then they can no longer be triggered because the image was built and push.
		img.Locker.Lock()
		img.RebuildDone = true
		img.Locker.Unlock()

		logrus.Debugf("Image \"%s\" is tagged for rebuild but ref is already present on the registry, skipping."+
			" if you want to rebuild anyway, use --force-rebuild", img.Name)
	}
}

// checkNeedsRetag iterates over the graph to find out which images need
// to be tagged with the new tag from the latest version.
func checkNeedsRetag(graph *dag.DAG, registry types.DockerRegistry, rateLimiter ratelimit.RateLimiter,
	oldTag string, newTag string) error {

	err := graph.WalkAsyncErr(func(node *dag.Node, wg *sync.WaitGroup, errChan chan error) {
		defer wg.Done()
		img := node.Image
		rateLimiter.Acquire()
		defer rateLimiter.Release()

		currentTagExists, err := registry.RefExists(img.DockerRef(newTag))
		if err != nil {
			errChan <- fmt.Errorf("could not check if tag exists in registry: %w", err)
			return
		}
		if currentTagExists {
			logrus.Debugf("Current tag \"%s:%s\" already exists, nothing to do", img.Name, newTag)
			return
		}

		img.Locker.Lock()
		previousTagExists, err := registry.RefExists(img.DockerRef(oldTag))
		img.Locker.Unlock()
		if err != nil {
			errChan <- fmt.Errorf("could not check if tag exists in registry: %w", err)
			return
		}
		if previousTagExists {
			logrus.Debugf("Previous tag \"%s:%s\" exists, image will be retagged", img.Name, oldTag)
			img.Locker.Lock()
			img.NeedsRetag = true
			img.Locker.Unlock()
			return
		}

		logrus.Warnf("Previous tag \"%s:%s\" missing, image will be rebuilt", img.Name, oldTag)
		node.Walk(tagForRebuildFunc)
	}, func(err error) {
		logrus.Errorf("Error while checking if image current tag exists: %v", err)
	})

	if err != nil {
		return fmt.Errorf("one of the registry actions failed, see logs for more details")
	}
	return nil
}
