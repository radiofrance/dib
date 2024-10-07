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

	if err := computeGraphHashes(graph, customHashListPath, buildArgs); err != nil {
		return nil, fmt.Errorf("could not compute graph hashes: %w", err)
	}

	return graph, nil
}

func buildGraph(buildPath, registryPrefix string) (*dag.DAG, error) {
	nodes := make(map[string]*dag.Node)
	if err := filepath.WalkDir(buildPath, func(filePath string, dirEntry os.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case filePath == buildPath:
			return nil
		case dirEntry.IsDir():
			if err := filepath.WalkDir(filePath, func(otherFile string, dirEntry os.DirEntry, err error) error {
				switch {
				case err != nil:
					return err
				case dirEntry.IsDir():
					return nil
				default:
					if _, ok := nodes[filePath]; !ok {
						nodes[filePath] = dag.NewNode(nil)
					}
					nodes[filePath].AddFile(otherFile)
					return nil
				}
			}); err != nil {
				return err
			}
		case dockerfile.IsDockerfile(filePath):
			nodes, err = processDockerfile(filePath, registryPrefix, nodes)
			if err != nil {
				return err
			}
		default:
			nodes = processFile(filePath, nodes)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	for key, node := range nodes {
		// Remove nodes that should be skipped
		if node.Image.SkipBuild {
			delete(nodes, key)
		}

		// Remove ignored files and .dockerignore from the root folder of the node
		// We ignore .dockerignore files for simplicity
		// In the real world, this file should not be ignored, but it
		// helps us in managing refactoring.
		files := []string{}
		for _, file := range node.Files {
			if !isFileIgnored(node, file) {
				files = append(files, file)
			}
		}
		node.Files = files
	}

	graph := assembleGraph(nodes)
	return graph, nil
}

func processDockerfile(filePath, registryPrefix string, nodes map[string]*dag.Node) (map[string]*dag.Node, error) {
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
		return nil, fmt.Errorf("could not read ignore patterns: %w", err)
	}

	for _, node := range nodes {
		if node.Image != nil && node.Image.Name == imageName {
			return nil, fmt.Errorf("duplicate image name %q found while reading file %q: previous file was %q",
				imageName, filePath, path.Join(node.Image.Dockerfile.ContextPath, node.Image.Dockerfile.Filename))
		}
	}

	if _, ok := nodes[dckfile.ContextPath]; !ok {
		nodes[dckfile.ContextPath] = dag.NewNode(nil)
	}
	nodes[dckfile.ContextPath].Image = &dag.Image{
		Name:              imageName,
		ShortName:         shortName,
		ExtraTags:         extraTags,
		Dockerfile:        dckfile,
		IgnorePatterns:    ignorePatterns,
		SkipBuild:         skipBuild,
		UseCustomHashList: useCustomHashList,
	}

	return nodes, nil
}

func processFile(filePath string, nodes map[string]*dag.Node) map[string]*dag.Node {
	dirPath := path.Dir(filePath)
	if _, ok := nodes[dirPath]; !ok {
		nodes[dirPath] = dag.NewNode(nil)
	}

	alreadyAdded := false
	for _, file := range nodes[dirPath].Files {
		if file == filePath {
			alreadyAdded = true
		}
	}
	if !alreadyAdded {
		nodes[dirPath].AddFile(filePath)
	}

	return nodes
}

func assembleGraph(nodes map[string]*dag.Node) *dag.DAG {
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

func computeGraphHashes(graph *dag.DAG, customHashListPath string, buildArgs map[string]string) error {
	customHumanizedHashList, err := LoadCustomHashList(customHashListPath)
	if err != nil {
		return fmt.Errorf("could not load custom humanized hash list: %w", err)
	}

	currNodes := graph.Nodes()
	for len(currNodes) > 0 {
		for _, node := range currNodes {
			if err := computeNodeHash(node, customHumanizedHashList, buildArgs); err != nil {
				return fmt.Errorf("could not compute hash for image %q: %w", node.Image.Name, err)
			}
		}

		nextNodes := []*dag.Node{}
		for _, currNode := range currNodes {
			nextNodes = append(nextNodes, currNode.Children()...)
		}
		currNodes = nextNodes
	}

	return nil
}

func computeNodeHash(node *dag.Node, customHumanizedHashList []string, buildArgs map[string]string) error {
	var parentHashes []string
	for _, parent := range node.Parents() {
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

	var err error
	node.Image.Hash, err = HashFiles(node.Image.Dockerfile.ContextPath, node.Files, parentHashes, humanizedKeywords)
	if err != nil {
		return fmt.Errorf("could not hash files: %w", err)
	}

	return nil
}

// isFileIgnored checks whether a file matches the images ignore patterns or is .dockerignore.
// It returns true if the file matches at least one pattern (meaning it should be ignored).
func isFileIgnored(node *dag.Node, file string) bool {
	if len(node.Image.IgnorePatterns) == 0 {
		return false
	}

	ignorePatternMatcher, err := patternmatcher.New(node.Image.IgnorePatterns)
	if err != nil {
		logger.Errorf("Could not create pattern matcher for %s, ignoring", node.Image.ShortName)
		return false
	}

	prefix := strings.TrimPrefix(strings.TrimPrefix(file, node.Image.Dockerfile.ContextPath), "/")
	if prefix == dockerignore {
		return true
	}

	ignored, err := ignorePatternMatcher.MatchesOrParentMatches(prefix)
	if err != nil {
		logger.Errorf("Could not match pattern for %s, ignoring", node.Image.ShortName)
		return false
	}

	return ignored
}

// HashFiles computes the sha256 from the contents of the files passed as argument.
// The files are alphabetically sorted so the returned hash is always the same.
// This also means the hash will change if the file names change but the contents don't.
func HashFiles(baseDir string, files, parentHashes, customHumanizedHashWordList []string) (string, error) {
	hash := sha256.New()
	slices.Sort(files)
	for _, filename := range files {
		if strings.Contains(filename, "\n") {
			return "", errors.New("file names with newlines are not supported")
		}

		file, err := os.Open(filename)
		if err != nil {
			return "", err
		}
		defer file.Close()

		hashFile := sha256.New()
		if _, err := io.Copy(hashFile, file); err != nil {
			return "", err
		}

		filename := strings.TrimPrefix(filename, baseDir)
		if _, err := fmt.Fprintf(hash, "%x  %s\n", hashFile.Sum(nil), filename); err != nil {
			return "", err
		}
	}

	parentHashes = append([]string(nil), parentHashes...)
	slices.Sort(parentHashes)
	for _, parentHash := range parentHashes {
		hash.Write([]byte(parentHash))
	}

	worldListToUse := humanhash.DefaultWordList
	if customHumanizedHashWordList != nil {
		worldListToUse = customHumanizedHashWordList
	}

	humanReadableHash, err := humanhash.HumanizeUsing(hash.Sum(nil), humanizedHashWordLength, worldListToUse, "-")
	if err != nil {
		return "", fmt.Errorf("could not humanize hash: %w", err)
	}

	return humanReadableHash, nil
}

// LoadCustomHashList try to load & parse a list of custom humanized hash to use.
func LoadCustomHashList(filepath string) ([]string, error) {
	if filepath == "" {
		return nil, nil
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	var lines []string
	for fileScanner.Scan() {
		lines = append(lines, fileScanner.Text())
	}

	return lines, nil
}
