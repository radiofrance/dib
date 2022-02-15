package dib

import (
	"strings"

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

// TagWithExtraTags iterates over the graph to retag each image with the tags defined in dib.extra-tags LABEL.
func TagWithExtraTags(graph *dag.DAG, tagger types.ImageTagger, tag string) error {
	return graph.WalkErr(func(node *dag.Node) error {
		img := node.Image
		if img.TagWithExtraTagsDone {
			return nil
		}
		defer func() {
			img.TagWithExtraTagsDone = true
		}()

		if img.Dockerfile == nil || img.Dockerfile.Labels == nil {
			return nil
		}

		extraTags, hasLabel := img.Dockerfile.Labels["dib.extra-tags"]
		if !hasLabel {
			return nil
		}

		for _, extraTag := range strings.Split(extraTags, ",") {
			logrus.Debugf("Add tag %s to image %s:%s", extraTag, img.Name, tag)
			if err := tagger.Tag(img.DockerRef(tag), img.DockerRef(extraTag)); err != nil {
				return err
			}
		}

		return nil
	})
}
