package dib

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/radiofrance/dib/version"

	"github.com/radiofrance/dib/dag"
	"github.com/radiofrance/dib/dockerfile"
	"github.com/sirupsen/logrus"
)

// GenerateDAG discovers and parses all Dockerfiles at a given path,
// and generates the DAG representing the relationships between images.
func GenerateDAG(buildPath string, registryPrefix string) *dag.DAG {
	var allFiles []string
	cache := make(map[string]*dag.Node)
	allParents := make(map[string][]string)
	err := filepath.Walk(buildPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			allFiles = append(allFiles, filePath)
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
				return fmt.Errorf("missing label \"name\" in Dockerfile at path \"%s\"", filePath)
			}
			img := &dag.Image{
				Name:       fmt.Sprintf("%s/%s", registryPrefix, imageShortName),
				ShortName:  imageShortName,
				Dockerfile: dckfile,
			}

			ignorePatterns, err := build.ReadDockerignore(path.Dir(filePath))
			if err != nil {
				return fmt.Errorf("could not read ignore patterns: %w", err)
			}
			img.IgnorePatterns = ignorePatterns

			allParents[img.Name] = dckfile.From
			cache[img.Name] = dag.NewNode(img)
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

	DAG := &dag.DAG{}
	// If an image has no parents in the DAG, we consider it a root image
	for name, img := range cache {
		if len(img.Parents()) == 0 {
			DAG.AddNode(cache[name])
		}
	}

	if err := generateHashes(DAG, allFiles); err != nil {
		logrus.Fatal(err)
	}

	return DAG
}

func generateHashes(graph *dag.DAG, allFiles []string) error {
	nodeFiles := map[*dag.Node][]string{}

	fileBelongsTo := map[string]*dag.Node{}
	for _, file := range allFiles {
		fileBelongsTo[file] = nil
	}

	// First, we do a depth-first search in the image graph to check if the files in diff belong to an image,
	// or is dockerignored
	// We start from the most specific image paths (children of children of children...), and we get back up
	// to parent images, to avoid false-positive and false-negative matches.
	graph.WalkInDepth(func(node *dag.Node) {
		nodeFiles[node] = []string{}
		for _, file := range allFiles {
			if !strings.HasPrefix(file, node.Image.Dockerfile.ContextPath) {
				// The current file is not lying in the current image build context, nor in a subdirectory.
				continue
			}

			if fileBelongsTo[file] != nil {
				// The current file has already been assigned to an image, most likely to a child image.
				continue
			}

			if path.Base(file) == dockerignore {
				// We ignore dockerignore file itself for simplicity
				// In the real world, this file should not be ignored but it
				// helps us in managing refactoring
				continue
			}

			if node.Image.IgnorePatterns != nil {
				if matchPattern(node, file) {
					// The current file matches a pattern in the dockerignore file
					continue
				}
			}

			// If we reach here, the file is part of the current image's context, we mark it as so.
			fileBelongsTo[file] = node
			nodeFiles[node] = append(nodeFiles[node], file)
		}
	})

	return graph.WalkErr(func(node *dag.Node) error {
		var parentHashes []string
		for _, parent := range node.Parents() {
			parentHashes = append(parentHashes, parent.Image.Hash)
		}

		hash, err := version.HashFiles(nodeFiles[node], parentHashes)
		if err != nil {
			return fmt.Errorf("could not hash files for node %s: %w", node.Image.Name, err)
		}
		node.Image.Hash = hash
		return nil
	})
}

func matchPattern(node *dag.Node, file string) bool {
	ignorePatternMatcher, err := fileutils.NewPatternMatcher(node.Image.IgnorePatterns)
	if err != nil {
		logrus.Errorf("Could not create pattern matcher for %s, ignoring", node.Image.ShortName)
		return false
	}

	prefix := strings.TrimPrefix(strings.TrimPrefix(file, node.Image.Dockerfile.ContextPath), "/")
	match, err := ignorePatternMatcher.Matches(prefix)
	if err != nil {
		logrus.Errorf("Could not match pattern for %s, ignoring", node.Image.ShortName)
		return false
	}
	return match
}
