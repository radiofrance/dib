package dib

import (
	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/types"
)

// Retag iterates over the graph to tag all images.
func Retag(graph *dag.DAG, tagger types.ImageTagger, placeholderTag string, release bool) error {
	return graph.WalkAsyncErr(func(node *dag.Node) error {
		img := node.Image
		if img.RetagDone {
			return nil
		}

		current := img.CurrentRef()
		final := img.DockerRef(img.Hash)
		if current != final {
			logger.Debugf("Tagging \"%s\" from \"%s\"", final, current)
			if err := tagger.Tag(current, final); err != nil {
				return err
			}
		}

		if release {
			if err := tagger.Tag(final, img.DockerRef(placeholderTag)); err != nil {
				return err
			}

			for _, tag := range img.ExtraTags {
				extra := img.DockerRef(tag)
				logger.Debugf("Tagging \"%s\" from \"%s\"", extra, final)
				if err := tagger.Tag(final, extra); err != nil {
					return err
				}
			}
		}

		img.RetagDone = true
		return nil
	})
}
