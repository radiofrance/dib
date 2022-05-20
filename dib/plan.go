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
func Plan(graph *dag.DAG, registry types.DockerRegistry, forceRebuild, testsEnabled bool) error {
	// Populate CurrentTag, TargetTag and ExtraTags for all images in the graph.
	graph.Walk(func(node *dag.Node) {
		img := node.Image

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

	tagExistsMap, err := refExistsMapForTag(graph, registry)
	if err != nil {
		return err
	}

	err = checkNeedsRebuild(graph, tagExistsMap)
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

// checkNeedsRebuild iterates over the graph to find out which images
// can't be found with their current tag, and must be rebuilt.
func checkNeedsRebuild(graph *dag.DAG, tagExistsMap *sync.Map) error {
	return graph.WalkErr(func(node *dag.Node) error {
		img := node.Image
		tagExists, present := tagExistsMap.Load(img.DockerRef(img.Hash))
		if !present {
			return fmt.Errorf("could not check if %s exists", img.DockerRef(img.Hash))
		}
		if tagExists.(bool) { //nolint:forcetypeassert
			logrus.Debugf("Tag \"%s:%s\" already exists, no rebuild required", img.Name, img.Hash)
			return nil
		}

		logrus.Warnf("Tag \"%s:%s\" is missing, image must be rebuilt", img.Name, img.Hash)
		img.NeedsRebuild = true
		return nil
	})
}

func refExistsMapForTag(graph *dag.DAG, registry types.DockerRegistry) (*sync.Map, error) {
	refExistsMap := &sync.Map{}
	err := graph.WalkAsyncErr(func(node *dag.Node) error {
		img := node.Image
		refAlreadyExists, err := registry.RefExists(img.DockerRef(img.Hash))
		if err != nil {
			return err
		}
		refExistsMap.Store(img.DockerRef(img.Hash), refAlreadyExists)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error during api call to check registry if tag exists: %w", err)
	}
	return refExistsMap, nil
}
