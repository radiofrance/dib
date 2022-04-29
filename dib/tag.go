package dib

import (
	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
)

// Retag iterates over the graph to retag each image with the given tag.
func Retag(graph *dag.DAG, tagger types.ImageTagger) error {
	return graph.WalkAsyncErr(func(node *dag.Node) error {
		img := node.Image
		if !img.NeedsRetag {
			return nil
		}
		if img.RetagDone {
			return nil
		}
		src := img.DockerRef(img.CurrentTag)
		dest := img.DockerRef(img.TargetTag)
		logrus.Debugf("Tagging \"%s\" from \"%s\"", dest, src)
		if err := tagger.Tag(src, dest); err != nil {
			return err
		}

		src = img.DockerRef(img.TargetTag)
		for _, tag := range img.ExtraTags {
			dest = img.DockerRef(tag)
			logrus.Debugf("Tagging \"%s\" from \"%s\"", dest, src)
			if err := tagger.Tag(src, dest); err != nil {
				return err
			}
		}

		img.RetagDone = true
		return nil
	})
}
