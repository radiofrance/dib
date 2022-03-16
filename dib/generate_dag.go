package dib

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/docker/cli/cli/command/image/build"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/dockerfile"
	"github.com/sirupsen/logrus"
)

// GenerateDAG discovers and parses all Dockerfiles at a given path,
// and generates the DAG representing the relationships between images.
func GenerateDAG(buildPath string, registryPrefix string) *dag.DAG {
	cache := make(map[string]*dag.Node)
	allParents := make(map[string][]string)
	err := filepath.Walk(buildPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if dockerfile.IsDockerfile(filePath) {
			dckfile, err := dockerfile.ParseDockerfile(filePath)
			if err != nil {
				return err
			}

			skipBuild, hasSkipLabel := dckfile.Labels["skipbuild"]
			if hasSkipLabel && skipBuild == "true" {
				return nil
			}
			imageShortName, hasSkipLabel := dckfile.Labels["name"]
			if !hasSkipLabel {
				return fmt.Errorf("missing label \"image\" in Dockerfile at path \"%s\"", filePath)
			}
			img := dag.NewImage(dag.NewImageArgs{
				Name:       fmt.Sprintf("%s/%s", registryPrefix, imageShortName),
				ShortName:  imageShortName,
				Dockerfile: dckfile,
			})

			ignorePatterns, err := build.ReadDockerignore(path.Dir(filePath))
			if err != nil {
				return fmt.Errorf("could not read ignore patterns: %w", err)
			}
			img.SetIgnorePatterns(ignorePatterns)

			allParents[img.GetName()] = dckfile.From
			cache[img.GetName()] = dag.NewNode(img)
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
				p.AddChild(cache[name])
			}
		}
	}

	DAG := dag.DAG{}
	// If an image has no parents in the DAG, we consider it a root image
	for name, img := range cache {
		if len(img.Parents()) == 0 {
			DAG.AddNode(cache[name])
		}
	}
	return &DAG
}
