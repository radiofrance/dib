package dib

import (
	"fmt"
	"strings"
	"sync"

	"github.com/radiofrance/dib/ratelimit"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
)

// Retag iterates over the graph to retag each image with the given tag.
func Retag(graph *dag.DAG, tagger types.ImageTagger, rateLimiter ratelimit.RateLimiter,
	oldTag string, newTag string) error {
	errChan := make(chan error)
	wgBuild := sync.WaitGroup{}

	graph.Walk(func(node *dag.Node) {
		img := node.Image
		if !img.NeedsRetag {
			return
		}
		if img.RetagDone {
			return
		}

		wgBuild.Add(1)
		go tagImage(&wgBuild, rateLimiter, img, newTag, tagger, oldTag, errChan)
	})

	go func() {
		wgBuild.Wait()
		close(errChan)
	}()

	hasError := false
	for err := range errChan {
		hasError = true
		logrus.Errorf("Error during image tag: %v", err)
	}

	if hasError {
		return fmt.Errorf("one of the image tag actions failed, see logs for more details")
	}
	return nil
}

func tagImage(wgBuild *sync.WaitGroup, rateLimiter ratelimit.RateLimiter,
	img *dag.Image, newTag string, tagger types.ImageTagger, oldTag string, errChan chan error) {
	defer wgBuild.Done()
	rateLimiter.Acquire()
	defer rateLimiter.Release()

	logrus.Debugf("Retag image %s with version %s", img.Name, newTag)
	if err := tagger.Tag(img.DockerRef(oldTag), img.DockerRef(newTag)); err != nil {
		errChan <- err
		return
	}
	img.RetagDone = true
}

// TagWithExtraTags iterates over the graph to retag each image with the tags defined in dib.extra-tags LABEL.
func TagWithExtraTags(graph *dag.DAG, tagger types.ImageTagger, rateLimiter ratelimit.RateLimiter, tag string) error {
	errChan := make(chan error)
	wgTags := sync.WaitGroup{}

	graph.Walk(func(node *dag.Node) {
		img := node.Image
		if img.TagWithExtraTagsDone {
			return
		}

		wgTags.Add(1)
		tagNodeWithExtraTags(&wgTags, rateLimiter, img, tag, tagger, errChan)
	})

	go func() {
		wgTags.Wait()
		close(errChan)
	}()

	hasError := false
	for err := range errChan {
		hasError = true
		logrus.Errorf("Error during image tag: %v", err)
	}

	if hasError {
		return fmt.Errorf("one of the image tag actions failed, see logs for more details")
	}
	return nil
}

func tagNodeWithExtraTags(wgTags *sync.WaitGroup, rateLimiter ratelimit.RateLimiter, img *dag.Image,
	tag string, tagger types.ImageTagger, errChan chan error) {
	defer wgTags.Done()
	rateLimiter.Acquire()
	defer rateLimiter.Release()
	defer func() {
		img.TagWithExtraTagsDone = true
	}()

	if img.Dockerfile == nil || img.Dockerfile.Labels == nil {
		return
	}

	extraTags, hasLabel := img.Dockerfile.Labels["dib.extra-tags"]
	if !hasLabel {
		return
	}

	for _, extraTag := range strings.Split(extraTags, ",") {
		logrus.Debugf("Add tag %s to image %s:%s", extraTag, img.Name, tag)
		if err := tagger.Tag(img.DockerRef(tag), img.DockerRef(extraTag)); err != nil {
			errChan <- err
			return
		}
	}
}
