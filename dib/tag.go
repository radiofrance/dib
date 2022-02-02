package dib

import (
	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
)

// Retag iterates over the graph to retag each image with the given tag.
func Retag(graph *dag.DAG, tagger types.ImageTagger, oldTag string, newTag string) error {
	return graph.WalkErr(func(node *dag.Node) error {
		img := node.Image
		if !img.NeedsRetag {
			return nil
		}
		if img.RetagDone {
			return nil
		}
		logrus.Debugf("Retag image %s with version %s", img.Name, newTag)
		if err := tagger.Tag(img.DockerRef(oldTag), img.DockerRef(newTag)); err != nil {
			return err
		}
		img.RetagDone = true
		return nil
	})
}

// RetagLatest iterates over the graph to retag each image with the latest tag.
func RetagLatest(graph *dag.DAG, tagger types.ImageTagger, tag string) error {
	return graph.WalkErr(func(node *dag.Node) error {
		img := node.Image
		if img.RetagLatestDone {
			return nil
		}
		logrus.Debugf("Retag latest tag for image %s with version %s", img.Name, tag)
		if err := tagger.Tag(img.DockerRef(tag), img.DockerRef("latest")); err != nil {
			return err
		}
		img.RetagLatestDone = true
		return nil
	})
}
