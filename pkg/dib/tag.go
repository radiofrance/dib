package dib

import (
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/logger"
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

			err := tagger.Tag(current, final)
			if err != nil {
				return err
			}
		}

		if release {
			err := tagger.Tag(final, img.DockerRef(placeholderTag))
			if err != nil {
				return err
			}

			for _, tag := range img.ExtraTags {
				extra := img.DockerRef(tag)
				logger.Debugf("Tagging \"%s\" from \"%s\"", extra, final)

				err := tagger.Tag(final, extra)
				if err != nil {
					return err
				}
			}
		}

		img.RetagDone = true

		return nil
	})
}
