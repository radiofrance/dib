package dib

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/moby/patternmatcher"
	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/wolfeidau/humanhash"
)

const (
	dockerignore            = ".dockerignore"
	humanizedHashWordLength = 4
)

// GenerateDAG discovers and parses all Dockerfiles at a given path,
// and generates the DAG representing the relationships between images.
func GenerateDAG(buildPath string, registryPrefix string, customHashListPath string) (*dag.DAG, error) {
	var allFiles []string
	cache := make(map[string]*dag.Node)
	allParents := make(map[string][]dockerfile.ImageRef)
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
			imageShortName, hasNameLabel := dckfile.Labels["name"]
			if !hasNameLabel {
				return fmt.Errorf("missing label \"name\" in Dockerfile at path \"%s\"", filePath)
			}
			img := &dag.Image{
				Name:       fmt.Sprintf("%s/%s", registryPrefix, imageShortName),
				ShortName:  imageShortName,
				Dockerfile: dckfile,
			}

			extraTagsLabel, hasLabel := img.Dockerfile.Labels["dib.extra-tags"]
			if hasLabel {
				img.ExtraTags = append(img.ExtraTags, strings.Split(extraTagsLabel, ",")...)
			}

			useCustomHashList, hasLabel := img.Dockerfile.Labels["dib.use-custom-hash-list"]
			if hasLabel && useCustomHashList == "true" {
				img.UseCustomHashList = true
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
		return nil, err
	}

	// Fill parents for each image, for simplicity of use in other functions
	for name, parents := range allParents {
		for _, parent := range parents {
			node, ok := cache[parent.Name]
			if !ok {
				continue
			}

			// Check that children does not already exist to avoid duplicates.
			childAlreadyExists := false
			for _, child := range node.Children() {
				if child.Image.Name == name {
					childAlreadyExists = true
				}
			}

			if childAlreadyExists {
				continue
			}

			node.AddChild(cache[name])
		}
	}

	graph := &dag.DAG{}
	// If an image has no parents in the DAG, we consider it a root image
	for name, img := range cache {
		if len(img.Parents()) == 0 {
			graph.AddNode(cache[name])
		}
	}

	if err := generateHashes(graph, allFiles, customHashListPath); err != nil {
		return nil, err
	}

	return graph, nil
}

func generateHashes(graph *dag.DAG, allFiles []string, customHashListPath string) error {
	customHumanizedHashList, err := loadCustomHumanizedHashList(customHashListPath)
	if err != nil {
		return err
	}

	nodeFiles := map[*dag.Node][]string{}

	fileBelongsTo := map[string]*dag.Node{}
	for _, file := range allFiles {
		fileBelongsTo[file] = nil
	}

	// First, we do a depth-first search in the image graph to map every file the image they belong to.
	// We start from the most specific image paths (children of children of children...), and we get back up
	// to parent images, to avoid false-positive and false-negative matches.
	// Files matching any pattern in the .dockerignore file are ignored.
	graph.WalkInDepth(func(node *dag.Node) {
		nodeFiles[node] = []string{}
		for _, file := range allFiles {
			if !strings.HasPrefix(file, node.Image.Dockerfile.ContextPath+"/") {
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

	for {
		needRepass := false
		err := graph.WalkErr(func(node *dag.Node) error {
			var parentHashes []string
			for _, parent := range node.Parents() {
				if parent.Image.Hash == "" {
					// At least one of the parent image has not been processed yet, we'll need to do an other pass
					needRepass = true
				}
				parentHashes = append(parentHashes, parent.Image.Hash)
			}

			var humanizedKeywords []string
			if node.Image.UseCustomHashList {
				humanizedKeywords = customHumanizedHashList
			}

			hash, err := hashFiles(node.Image.Dockerfile.ContextPath, nodeFiles[node], parentHashes, humanizedKeywords)
			if err != nil {
				return fmt.Errorf("could not hash files for node %s: %w", node.Image.Name, err)
			}
			node.Image.Hash = hash
			return nil
		})
		if err != nil {
			return err
		}
		if !needRepass {
			return nil
		}
	}
}

// matchPattern checks whether a file matches the images ignore patterns.
// It returns true if the file matches at least one pattern (meaning it should be ignored).
func matchPattern(node *dag.Node, file string) bool {
	ignorePatternMatcher, err := patternmatcher.New(node.Image.IgnorePatterns)
	if err != nil {
		logger.Errorf("Could not create pattern matcher for %s, ignoring", node.Image.ShortName)
		return false
	}

	prefix := strings.TrimPrefix(strings.TrimPrefix(file, node.Image.Dockerfile.ContextPath), "/")
	match, err := ignorePatternMatcher.MatchesOrParentMatches(prefix)
	if err != nil {
		logger.Errorf("Could not match pattern for %s, ignoring", node.Image.ShortName)
		return false
	}
	return match
}

// hashFiles computes the sha256 from the contents of the files passed as argument.
// The files are alphabetically sorted so the returned hash is always the same.
// This also means the hash will change if the file names change but the contents don't.
func hashFiles(
	baseDir string,
	files []string,
	parentHashes []string,
	customHumanizedHashWordList []string,
) (string, error) {
	hash := sha256.New()
	files = append([]string(nil), files...)
	sort.Strings(files)
	for _, file := range files {
		if strings.Contains(file, "\n") {
			return "", errors.New("filenames with newlines are not supported")
		}
		readCloser, err := os.Open(file)
		if err != nil {
			return "", err
		}
		hashFile := sha256.New()
		_, err = io.Copy(hashFile, readCloser)
		if err != nil {
			return "", err
		}
		err = readCloser.Close()
		if err != nil {
			return "", err
		}
		filename := strings.TrimPrefix(file, baseDir)
		_, err = fmt.Fprintf(hash, "%x  %s\n", hashFile.Sum(nil), filename)
		if err != nil {
			return "", err
		}
	}

	parentHashes = append([]string(nil), parentHashes...)
	sort.Strings(parentHashes)
	for _, parentHash := range parentHashes {
		hash.Write([]byte(parentHash))
	}

	var humanReadableHash string
	var err error

	worldListToUse := humanhash.DefaultWordList
	if customHumanizedHashWordList != nil {
		worldListToUse = customHumanizedHashWordList
	}

	humanReadableHash, err = humanhash.HumanizeUsing(hash.Sum(nil), humanizedHashWordLength, worldListToUse, "-")
	if err != nil {
		return "", fmt.Errorf("could not humanize hash: %w", err)
	}
	return humanReadableHash, nil
}

// loadCustomHumanizedHashList try to load & parse a list of custom humanized hash to use.
func loadCustomHumanizedHashList(filepath string) ([]string, error) {
	if filepath == "" {
		return nil, nil
	}
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot load custom humanized word list file, err: %w", err)
	}

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	var lines []string
	for fileScanner.Scan() {
		lines = append(lines, fileScanner.Text())
	}
	_ = file.Close()

	return lines, nil
}
