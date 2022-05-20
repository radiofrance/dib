package dib

import (
	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
)

// Retag iterates over the graph to retag each image with the given tag.
func Retag(graph *dag.DAG, tagger types.ImageTagger, release bool) error {
	return graph.WalkAsyncErr(func(node *dag.Node) error {
		img := node.Image
		if img.RetagDone {
			return nil
		}

		current := img.DockerRef(img.CurrentTag())
		final := img.DockerRef(img.Hash)
		if current != final {
			logrus.Debugf("Tagging \"%s\" from \"%s\"", final, current)
			if err := tagger.Tag(current, final); err != nil {
				return err
			}
		}

		if release {
			for _, tag := range img.ExtraTags {
				extra := img.DockerRef(tag)
				logrus.Debugf("Tagging \"%s\" from \"%s\"", extra, final)
				if err := tagger.Tag(final, extra); err != nil {
					return err
				}
			}
		}

		img.RetagDone = true
		return nil
	})
}
