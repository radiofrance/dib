package dib

import (
	"fmt"
	"strings"
	"sync"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
)

const dockerignore = ".dockerignore"

// Plan decides which actions need to be performed on each image.
func Plan(graph *dag.DAG, registry types.DockerRegistry, diffs []string, oldTag, newTag string,
	releaseMode, forceRebuild, testsEnabled bool,
) error {
	// Populate CurrentTag, TargetTag and ExtraTags for all images in the graph.
	graph.Walk(func(node *dag.Node) {
		img := node.Image
		img.CurrentTag = oldTag
		img.TargetTag = "dev-" + img.Hash

		if img.Dockerfile == nil || img.Dockerfile.Labels == nil {
			return
		}

		extraTagsLabel, hasLabel := img.Dockerfile.Labels["dib.extra-tags"]
		if !hasLabel {
			return
		}

		node.Image.ExtraTags = append(node.Image.ExtraTags, strings.Split(extraTagsLabel, ",")...)
	})

	if forceRebuild {
		logrus.Info("force rebuild mode enabled, all images will be rebuild regardless of their changes")
		graph.Walk(func(node *dag.Node) {
			node.Image.NeedsRebuild = true
			node.Image.NeedsTests = testsEnabled
		})
		return nil
	}

	currentTagExistsMap, err := refExistsMapForTag(graph, registry, newTag)
	if err != nil {
		return err
	}
	previousTagExistsMap, err := refExistsMapForTag(graph, registry, oldTag)
	if err != nil {
		return err
	}

	err = checkNeedsRebuild(graph, previousTagExistsMap, oldTag)
	if err != nil {
		return err
	}

	err = checkAlreadyBuilt(graph, currentTagExistsMap, newTag)
	if err != nil {
		return err
	}

	if releaseMode { // In release mode, we retag all images.
		err = checkNeedsRetag(graph, currentTagExistsMap, newTag)
		if err != nil {
			return err
		}
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

func tagForRebuildFunc(node *dag.Node) {
	node.Image.NeedsRebuild = true
}

// checkNeedsRebuild iterates over the graph to find out which images
// can't be found with the former tag, and must be rebuilt.
func checkNeedsRebuild(graph *dag.DAG, previousTagExistsMap *sync.Map, oldTag string) error {
	return graph.WalkErr(func(node *dag.Node) error {
		img := node.Image
		previousTagExists, present := previousTagExistsMap.Load(img.DockerRef(oldTag))
		if !present {
			return fmt.Errorf("could not find ref %s in previousTagExists map", img.DockerRef(oldTag))
		}
		if previousTagExists.(bool) { //nolint:forcetypeassert
			logrus.Debugf("Previous tag \"%s:%s\" exists, no rebuild required", img.Name, oldTag)
			return nil
		}

		logrus.Warnf("Previous tag \"%s:%s\" missing, image must be rebuilt", img.Name, oldTag)
		node.Walk(tagForRebuildFunc)
		return nil
	})
}

// checkAlreadyBuilt iterates over the graph to find out which images
// already exist in the new version, so they don't need to be built again.
func checkAlreadyBuilt(graph *dag.DAG, currentTagExistsMap *sync.Map, newTag string) error {
	return graph.WalkErr(func(node *dag.Node) error {
		img := node.Image
		if !img.NeedsRebuild {
			// Don't rebuild images that didn't change since last built revision.
			return nil
		}

		refAlreadyExists, present := currentTagExistsMap.Load(img.DockerRef(newTag))
		if !present {
			return fmt.Errorf("could not find ref %s in map", img.DockerRef(newTag))
		}
		if refAlreadyExists.(bool) { //nolint:forcetypeassert
			// Looks like dib has already built this image in a previous run,
			// we can skip the build, but we don't want to disable the tests.
			// This is to avoid the situation where the tests failed on previous dib run,
			// then they can no longer be triggered because the image was built and push.
			img.RebuildDone = true

			logrus.Debugf("Image \"%s\" is tagged for rebuild but ref is already present on the registry, skipping."+
				" if you want to rebuild anyway, use --force-rebuild", img.Name)
		}
		return nil
	})
}

// checkNeedsRetag iterates over the graph to find out which images need
// to be tagged with the new tag from the latest version.
func checkNeedsRetag(graph *dag.DAG, currentTagExistsMap *sync.Map, newTag string) error {
	return graph.WalkErr(func(node *dag.Node) error {
		img := node.Image
		if img.NeedsRebuild {
			// If this image needs rebuild, then its children too, no need to go deeper
			return nil
		}

		currentTagExists, present := currentTagExistsMap.Load(img.DockerRef(newTag))
		if !present {
			return fmt.Errorf("could not find ref %s in currentTagExists map", img.DockerRef(newTag))
		}
		if currentTagExists.(bool) { //nolint:forcetypeassert
			logrus.Debugf("Current tag \"%s:%s\" already exists, nothing to do", img.Name, newTag)
			return nil
		}

		logrus.Debugf("Current tag \"%s:%s\" does not exist, image will be tagged", img.Name, newTag)
		img.NeedsRetag = true
		return nil
	})
}

func refExistsMapForTag(graph *dag.DAG, registry types.DockerRegistry, tag string) (*sync.Map, error) {
	refExistsMap := &sync.Map{}
	err := graph.WalkAsyncErr(func(node *dag.Node) error {
		img := node.Image
		refAlreadyExists, err := registry.RefExists(img.DockerRef(tag))
		if err != nil {
			return err
		}
		refExistsMap.Store(img.DockerRef(tag), refAlreadyExists)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error during api call to check registry if tag exists: %w", err)
	}
	return refExistsMap, nil
}
