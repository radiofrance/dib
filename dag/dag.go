package dag

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/radiofrance/dib/docker"
	"github.com/sirupsen/logrus"
)

type DAG struct {
	Images      []*Image
	Registry    DockerRegistry
	Builder     ImageBuilder
	TestRunners []TestRunner
}

type ImageBuilder interface {
	Build(opts docker.ImageBuilderOpts) error
}

type TestRunner interface {
	RunTest(ref, path string) error
}

// DockerRegistry is an interface for dealing with docker registries.
type DockerRegistry interface {
	RefExists(imageRef string) (bool, error)
	Retag(existingRef, toCreateRef string) error
}

func (dag *DAG) GenerateDAG(buildPath string, registryPrefix string) {
	cache := make(map[string]*Image)

	allParents := make(map[string][]string)
	err := filepath.Walk(buildPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if docker.IsDockerfile(filePath) {
			dockerfile, err := docker.ParseDockerfile(filePath)
			if err != nil {
				return err
			}

			skipBuild, hasSkipLabel := dockerfile.Labels["skipbuild"]
			if hasSkipLabel && skipBuild == "true" {
				return nil
			}
			imageShortName, hasSkipLabel := dockerfile.Labels["name"]
			if !hasSkipLabel {
				return fmt.Errorf("missing label \"image\" in Dockerfile at path \"%s\"", filePath)
			}
			version, hasSkipLabel := dockerfile.Labels["version"]
			if !hasSkipLabel {
				logrus.Warnf("missing label \"version\" in Dockerfile at path \"%s\"", filePath)
			}
			img := &Image{
				Name:          fmt.Sprintf("%s/%s", registryPrefix, imageShortName),
				ShortName:     imageShortName,
				InlineVersion: version,
				Dockerfile:    dockerfile,
				RebuildCond:   sync.NewCond(&sync.Mutex{}),
				Builder:       dag.Builder,
				Registry:      dag.Registry,
				TestRunners:   dag.TestRunners,
			}

			allParents[img.Name] = dockerfile.From
			cache[img.Name] = img
		}
		return nil
	})
	if err != nil {
		logrus.Fatal(err)
	}

	// Fill parents for each image, for simplicity of use in other functions
	for name, parents := range allParents {
		for _, parent := range parents {
			if p, ok := cache[parent]; ok {
				cache[name].Parents = append(cache[name].Parents, cache[parent])
				p.Children = append(p.Children, cache[name])
			}
		}
	}

	// If an image has no parents in the DAG, we consider it a root image
	for name, img := range cache {
		if len(img.Parents) == 0 {
			dag.Images = append(dag.Images, cache[name])
		}
	}
}

func (dag *DAG) TagForRebuild() {
	for _, image := range dag.Images {
		image.tagForRebuild()
	}
}

// CheckForDiff checks the diffs and marks images to be rebuilt if files in their context have changed.
func (dag *DAG) CheckForDiff(diffs []string) {
	diffBelongsTo := map[string]*Image{}
	for _, file := range diffs {
		diffBelongsTo[file] = nil
	}

	// First, we do a depth-first search in the image graph to check if the files in diff belong to an image.
	// We start from the most specific image paths (children of children of children...), and we get back up
	// to parent images, to avoid false-positive and false-negative matches.
	for _, rootImg := range dag.Images {
		rootImg.checkDiffRecursive(diffs, diffBelongsTo)
	}

	for file, img := range diffBelongsTo {
		if img != nil {
			logrus.Debugf("Image \"%s\" needs a rebuild because file \"%s\" has changed", img.Name, file)
			img.tagForRebuild()
		}
	}
}

func (dag *DAG) Retag(newTag, oldTag string) error {
	for _, img := range dag.Images {
		err := img.retag(newTag, oldTag)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dag *DAG) Rebuild(newTag string, forceRebuild, runTests bool) {
	errs := make(chan error, 1)
	for _, img := range dag.Images {
		go img.Rebuild(newTag, forceRebuild, runTests, errs)
	}
	for i := 0; i < len(dag.Images); i++ {
		err := <-errs
		if err != nil {
			logrus.Errorf("Error building image: %v", err)
		}
	}
	close(errs)
}
