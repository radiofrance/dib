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
	"slices"
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
	graph, err := buildGraph(buildPath, registryPrefix)
	if err != nil {
		return nil, err
	}

	var customHashList []string
	if customHashListPath != "" {
		customHashList, err = loadCustomHashList(customHashListPath)
		if err != nil {
			return nil, fmt.Errorf("could not load custom humanized hash list: %w", err)
		}
	}

	return computeHashes(graph, customHashList, buildArgs)
}

func buildGraph(buildPath, registryPrefix string) (*dag.DAG, error) {
	nodes := make(map[string]*dag.Node)
	if err := filepath.WalkDir(buildPath, func(name string, dir os.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case dir.IsDir():
		case dockerfile.IsDockerfile(name):
			img, err := newImageFromDockerfile(name, registryPrefix)
			if err != nil {
				return err
			}

			for _, node := range nodes {
				if node.Image != nil && node.Image.Name == img.Name {
					return fmt.Errorf("duplicate image name %q found while reading file %q: previous file was %q",
						img.Name, name, path.Join(node.Image.Dockerfile.ContextPath, node.Image.Dockerfile.Filename))
				}
			}

			// Don't create the node if the image has the skipbuild label.
			if img.SkipBuild {
				return nil
			}

			nodes[path.Dir(name)] = dag.NewNode(img)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return newGraphFromNodes(nodes), nil
}

func newImageFromDockerfile(filePath, registryPrefix string) (*dag.Image, error) {
	dckfile, err := dockerfile.ParseDockerfile(filePath)
	if err != nil {
		return nil, err
	}

	skipBuild := false
	skipBuildString, hasSkipLabel := dckfile.Labels["skipbuild"]
	if hasSkipLabel && skipBuildString == "true" {
		skipBuild = true
	}

	shortName, hasNameLabel := dckfile.Labels["name"]
	if !skipBuild && !hasNameLabel {
		return nil, fmt.Errorf("missing label \"name\" in Dockerfile at path %q", filePath)
	}

	imageName := fmt.Sprintf("%s/%s", registryPrefix, shortName)

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

	ignorePatterns, err := build.ReadDockerignore(dckfile.ContextPath)
	if err != nil {
		return nil, fmt.Errorf("could not read dockerignore: %w", err)
	}

	contextFiles, err := getDockerContextFiles(dckfile.ContextPath, ignorePatterns)
	if err != nil {
		return nil, fmt.Errorf("could not get docker context files: %w", err)
	}

	return &dag.Image{
		Name:              imageName,
		ShortName:         shortName,
		ExtraTags:         extraTags,
		Dockerfile:        dckfile,
		IgnorePatterns:    ignorePatterns,
		ContextFiles:      contextFiles,
		SkipBuild:         skipBuild,
		UseCustomHashList: useCustomHashList,
	}, nil
}

func getDockerContextFiles(contextPath string, ignorePatterns []string) ([]string, error) {
	contextFiles := []string{}
	if err := filepath.WalkDir(contextPath, func(name string, dir os.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case dir.IsDir():
		default:
			// Don't add ignored files/folders and .dockerignore from the root folder of the context path.
			// We ignore .dockerignore files for simplicity
			// In the real world, this file should not be ignored, but it
			// helps us in managing refactoring.
			prefix := strings.TrimPrefix(strings.TrimPrefix(name, contextPath), "/")
			if prefix == dockerignore {
				return nil
			}

			if len(ignorePatterns) == 0 {
				contextFiles = append(contextFiles, name)
				return nil
			}

			ignorePatternMatcher, err := patternmatcher.New(ignorePatterns)
			if err != nil {
				return fmt.Errorf("could not create pattern matcher: %w", err)
			}

			ignored, err := ignorePatternMatcher.MatchesOrParentMatches(prefix)
			if err != nil {
				return fmt.Errorf("could not match pattern: %w", err)
			}

			if !ignored {
				contextFiles = append(contextFiles, name)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return contextFiles, nil
}

func newGraphFromNodes(nodes map[string]*dag.Node) *dag.DAG {
	for _, node := range nodes {
		if node.Image == nil {
			continue
		}
		for _, parent := range node.Image.Dockerfile.From {
			for _, parentNode := range nodes {
				if parentNode.Image == nil {
					continue
				}
				if parentNode.Image.Name == parent.Name {
					parentNode.AddChild(node)
				}
			}
		}
	}

	graph := &dag.DAG{}
	// If an image has no parents in the DAG, we can consider it root
	for name, img := range nodes {
		if len(img.Parents()) == 0 {
			graph.AddNode(nodes[name])
		}
	}

	return graph
}

func computeHashes(graph *dag.DAG, customHashList []string, buildArgs map[string]string) (*dag.DAG, error) {
	currNodes := graph.Nodes()
	for len(currNodes) > 0 {
		for _, node := range currNodes {
			var err error
			node.Image.Hash, err = computeNodeHash(node, customHashList, buildArgs)
			if err != nil {
				return nil, fmt.Errorf("could not compute hash for image %q: %w", node.Image.Name, err)
			}
		}

		nextNodes := []*dag.Node{}
		for _, currNode := range currNodes {
			nextNodes = append(nextNodes, currNode.Children()...)
		}
		currNodes = nextNodes
	}

	return graph, nil
}

func computeNodeHash(node *dag.Node, customHashList []string, buildArgs map[string]string) (string, error) {
	var parentHashes []string
	for _, parent := range node.Parents() {
		parentHashes = append(parentHashes, parent.Image.Hash)
	}

	var hashList []string
	if node.Image.UseCustomHashList {
		hashList = customHashList
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
		return "", fmt.Errorf("failed to replace ARG instructions in file %s: %w", filename, err)
	}
	defer func() {
		if err := dockerfile.ResetFile(
			filename, argInstructionsToReplace); err != nil {
			logger.Warnf("failed to reset ARG instructions in file %q: %v", filename, err)
		}
	}()

	return hashFiles(node.Image.Dockerfile.ContextPath, node.Image.ContextFiles, parentHashes, hashList)
}

// hashFiles computes the sha256 from the contents of the files passed as argument.
// The files are alphabetically sorted so the returned hash is always the same.
// This also means the hash will change if the file names change but the contents don't.
func hashFiles(baseDir string, files, parentHashes, hashList []string) (string, error) {
	hash := sha256.New()
	slices.Sort(files)
	for _, filename := range files {
		if strings.Contains(filename, "\n") {
			return "", errors.New("file names with newlines are not supported")
		}

		file, err := os.Open(filename) //nolint:gosec
		if err != nil {
			return "", err
		}
		defer func() {
			_ = file.Close()
		}()

		hashFile := sha256.New()
		if _, err := io.Copy(hashFile, file); err != nil {
			return "", err
		}

		filename := strings.TrimPrefix(filename, baseDir)
		if _, err := fmt.Fprintf(hash, "%x  %s\n", hashFile.Sum(nil), filename); err != nil {
			return "", err
		}
	}

	slices.Sort(parentHashes)
	for _, parentHash := range parentHashes {
		hash.Write([]byte(parentHash))
	}

	if len(hashList) == 0 {
		hashList = humanhash.DefaultWordList
	}

	humanReadableHash, err := humanhash.HumanizeUsing(hash.Sum(nil), humanizedHashWordLength, hashList, "-")
	if err != nil {
		return "", fmt.Errorf("could not humanize hash: %w", err)
	}

	return humanReadableHash, nil
}

// loadCustomHashList try to load & parse a list of custom humanized hash to use.
func loadCustomHashList(filepath string) ([]string, error) {
	file, err := os.Open(filepath) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	var lines []string
	for fileScanner.Scan() {
		lines = append(lines, fileScanner.Text())
	}

	return lines, nil
}
