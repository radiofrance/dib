package dib

import (
	"fmt"
	"sync"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/types"
)

// Plan decides which actions need to be performed on each image.
func Plan(graph *dag.DAG, registry types.DockerRegistry, forceRebuild, testsEnabled bool,
) error {
	if forceRebuild {
		logger.Infof("force rebuild mode enabled, all images will be rebuild regardless of their changes")
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
		ref := img.DockerRef(img.Hash)
		tagExists, present := tagExistsMap.Load(ref)
		if !present {
			return fmt.Errorf("could not check if %s exists", ref)
		}
		if tagExists.(bool) { //nolint:forcetypeassert
			logger.Debugf("Ref \"%s\" already exists, no rebuild required", ref)
			return nil
		}

		logger.Infof("Ref \"%s\" is missing, image must be rebuilt", ref)
		img.NeedsRebuild = true
		return nil
	})
}

func refExistsMapForTag(graph *dag.DAG, registry types.DockerRegistry) (*sync.Map, error) {
	refExistsMap := &sync.Map{}
	err := graph.WalkAsyncErr(func(node *dag.Node) error {
		img := node.Image
		ref := img.DockerRef(img.Hash)
		refAlreadyExists, err := registry.RefExists(ref)
		if err != nil {
			return err
		}
		refExistsMap.Store(ref, refAlreadyExists)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error during api call to check registry if tag exists: %w", err)
	}
	return refExistsMap, nil
}
