package dag

import (
	"fmt"
	"strings"
	"sync"

	"github.com/radiofrance/dib/docker"
	"github.com/sirupsen/logrus"
)

const maxGoroutines = 1

var rateLimit = make(chan struct{}, maxGoroutines)

type Image struct {
	Name          string
	ShortName     string
	InlineVersion string
	Dockerfile    *docker.Dockerfile
	Children      []*Image
	Parents       []*Image
	NeedsRebuild  bool
	RetagDone     bool
	RebuildDone   bool
	RebuildCond   *sync.Cond
	Registry      DockerRegistry
	Builder       ImageBuilder
	TestRunners   []TestRunner
}

// Rebuild iterates over the graph to rebuild each image that is tagged for rebuild.
func (img *Image) Rebuild(newTag string, forceRebuild, runTests bool, errChan chan error) {
	refAlreadyExists, err := img.Registry.RefExists(img.dockerRef(newTag))
	if err != nil {
		errChan <- err
		return
	}

	img.RebuildCond.L.Lock()

	if img.NeedsRebuild && !img.RebuildDone {
		for _, parent := range img.Parents {
			if parent.NeedsRebuild && !parent.RebuildDone {
				parent.RebuildCond.Wait()
			}
		}

		if refAlreadyExists && !forceRebuild {
			logrus.Debugf("Image \"%s\" is tagued for rebuild but ref is already present on the registry, skipping."+
				" if you want to rebuild anyway, use --force-rebuild", img.Name)
		} else {
			err := img.doRebuild(newTag)
			if err != nil {
				errChan <- err
				return
			}

			if runTests {
				logrus.Infof("Running tests for \"%s:%s\"", img.Name, newTag)
				err := img.runTests(fmt.Sprintf("%s:%s", img.Name, newTag), img.Dockerfile.ContextPath)
				if err != nil {
					errChan <- fmt.Errorf("tests failed for %s:%s: %w", img.Name, newTag, err)
					return
				}
			}
		}

		img.RebuildDone = true
		img.RebuildCond.Broadcast()
	}

	img.RebuildCond.L.Unlock()

	errs := make(chan error, 1)
	for _, child := range img.Children {
		go child.Rebuild(newTag, forceRebuild, runTests, errs)
	}
	for i := 0; i < len(img.Children); i++ {
		err := <-errs
		if err != nil {
			logrus.Errorf("Error building image: %v", err)
		}
	}
	errChan <- nil
}

// doRebuild do the effective build action.
func (img *Image) doRebuild(newTag string) error {
	rateLimit <- struct{}{}
	defer func() {
		<-rateLimit
	}()

	logrus.Infof("Building \"%s:%s\" in context \"%s\"", img.Name, newTag, img.Dockerfile.ContextPath)

	if err := docker.ReplaceFromTag(*img.Dockerfile, newTag); err != nil {
		return err
	}

	err := img.Builder.Build(docker.ImageBuilderOpts{
		Context: img.Dockerfile.ContextPath,
		Tag:     fmt.Sprintf("%s:%s", img.Name, newTag),
	})
	if err != nil {
		return err
	}

	if err := docker.ResetFromTag(*img.Dockerfile, newTag); err != nil {
		return err
	}

	return nil
}

// runTests run docker tests for each TestRunner.
func (img *Image) runTests(ref, path string) error {
	rateLimit <- struct{}{}
	defer func() {
		<-rateLimit
	}()

	for _, runner := range img.TestRunners {
		if err := runner.RunTest(ref, path); err != nil {
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
	} else {
		if previousTagExists {
			return img.retagRemote(oldTag, newTag)
		} else {
			inlineVersionTagExists, err := img.Registry.RefExists(img.dockerRef(img.InlineVersion))
			if err != nil {
				return err
			}
			if inlineVersionTagExists {
				logrus.Warnf(
					"Previous tag \"%s:%s\" missing, image will be retagged with inline version \"%s\"",
					img.Name,
					oldTag,
					img.InlineVersion,
				)
				return img.retagRemote(img.InlineVersion, newTag)
			} else {
				logrus.Warnf("Previous tag \"%s:%s\" missing, image will be rebuilt", img.Name, oldTag)
				img.tagForRebuild()
			}
		}
	}
	return nil
}

func (img *Image) retagRemote(oldTag string, newTag string) error {
	logrus.Infof("Retagging image \"%s:%s\" with tag \"%s\"", img.Name, oldTag, newTag)
	err := img.Registry.Retag(img.dockerRef(oldTag), img.dockerRef(newTag))
	img.RetagDone = true
	return err
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
