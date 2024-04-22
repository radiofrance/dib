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
func GenerateDAG(buildPath, registryPrefix, customHashListPath string, buildArgs map[string]string) (*dag.DAG, error) {
	var allFiles []string
	nodes := make(map[string]*dag.Node)
	if err := filepath.Walk(buildPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			allFiles = append(allFiles, filePath)
		}

		if !dockerfile.IsDockerfile(filePath) {
			return nil
		}

		dckfile, err := dockerfile.ParseDockerfile(filePath)
		if err != nil {
			return err
		}

		skipBuild, hasSkipLabel := dckfile.Labels["skipbuild"]
		if hasSkipLabel && skipBuild == "true" {
			return nil
		}

		shortName, hasNameLabel := dckfile.Labels["name"]
		if !hasNameLabel {
			return fmt.Errorf("missing label \"name\" in Dockerfile at path %q", filePath)
		}

		name := fmt.Sprintf("%s/%s", registryPrefix, shortName)

		var extraTags []string
		value, hasLabel := dckfile.Labels["dib.extra-tags"]
		if hasLabel {
			extraTags = strings.Split(value, ",")
		}

		useCustomHashList := false
		value, hasLabel = dckfile.Labels["dib.use-custom-hash-list"]
		if hasLabel && value == "true" {
			useCustomHashList = true
		}

		ignorePatterns, err := build.ReadDockerignore(path.Dir(filePath))
		if err != nil {
			return fmt.Errorf("could not read ignore patterns: %w", err)
		}

		if n, ok := nodes[name]; ok {
			return fmt.Errorf("duplicate image name %q found while reading file %q: previous file was %q",
				name, filePath, path.Join(n.Image.Dockerfile.ContextPath, n.Image.Dockerfile.Filename))
		}

		nodes[name] = dag.NewNode(&dag.Image{
			Name:              name,
			ShortName:         shortName,
			ExtraTags:         extraTags,
			Dockerfile:        dckfile,
			IgnorePatterns:    ignorePatterns,
			UseCustomHashList: useCustomHashList,
		})

		return nil
	}); err != nil {
		return nil, err
	}

	for _, node := range nodes {
		for _, parent := range node.Image.Dockerfile.From {
			parentNode, ok := nodes[parent.Name]
			if ok {
				parentNode.AddChild(node)
			}
		}
	}

	graph := &dag.DAG{}
	// If an image has no parents in the DAG, we consider it a root image
	for name, img := range nodes {
		if len(img.Parents()) == 0 {
			graph.AddNode(nodes[name])
		}
	}

	if err := generateHashes(graph, allFiles, customHashListPath, buildArgs); err != nil {
		return nil, err
	}

	return graph, nil
}

func generateHashes(graph *dag.DAG, allFiles []string, customHashListPath string, buildArgs map[string]string) error {
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
		if err := graph.WalkErr(func(node *dag.Node) error {
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

			filename := path.Join(node.Image.Dockerfile.ContextPath, node.Image.Dockerfile.Filename)

			argInstructionsToReplace := make(map[string]string)
			for key, newArg := range buildArgs {
				prevArgInstruction, ok := node.Image.Dockerfile.Args[key]
				if ok {
					argInstructionsToReplace[prevArgInstruction] = fmt.Sprintf("ARG %s=%s", key, newArg)
					logger.Debugf("Overriding ARG instruction %q in %q [%q -> %q]",
						key, filename, prevArgInstruction, fmt.Sprintf("ARG %s=%s", key, newArg))
				}
			}

			if err := dockerfile.ReplaceInFile(
				filename, argInstructionsToReplace); err != nil {
				return fmt.Errorf("failed to replace ARG instructions in file %s: %w", filename, err)
			}
			defer func() {
				if err := dockerfile.ResetFile(
					filename, argInstructionsToReplace); err != nil {
					logger.Warnf("failed to reset ARG instructions in file %q: %v", filename, err)
				}
			}()

			hash, err := hashFiles(node.Image.Dockerfile.ContextPath, nodeFiles[node], parentHashes, humanizedKeywords)
			if err != nil {
				return fmt.Errorf("could not hash files for node %s: %w", node.Image.Name, err)
			}
			node.Image.Hash = hash
			return nil
		}); err != nil {
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
