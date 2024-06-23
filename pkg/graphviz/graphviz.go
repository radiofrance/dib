package graphviz

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/exec"
)

const (
	// graphDot is the name of the file inside we put graphiz representation of the graph.
	graphDot = "dag.dot"

	// graphPng is the final file inside we put dib graph.
	graphPng = "dag.png"
)

func GenerateGraph(dag *dag.DAG, reportRootDir string) error {
	if err := GenerateDotviz(dag, path.Join(reportRootDir, graphDot)); err != nil {
		return err
	}
	shell := &exec.ShellExecutor{
		Dir: reportRootDir,
	}
	if _, err := shell.Execute("dot", "-Tpng", graphDot, "-o", graphPng); err != nil {
		return err
	}
	return nil
}

func GenerateDotviz(graph *dag.DAG, output string) error {
	file, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	writer := bufio.NewWriter(file)
	opts := []string{
		"digraph images {",
		"rankdir = \"LR\";",
		"node[fontsize = 10, shape = box, height = 0.25];",
		"edge [fontsize = 10];\n",
	}

	if _, err := writer.WriteString(strings.Join(opts, "\n")); err != nil {
		return err
	}

	err = graph.WalkErr(func(node *dag.Node) error {
		return generateDotvizImg(node, writer)
	})
	if err != nil {
		return err
	}

	if _, err := writer.WriteString("}"); err != nil {
		return err
	}
	return writer.Flush()
}

func generateDotvizImg(node *dag.Node, writer *bufio.Writer) error {
	img := node.Image
	color := "white"
	if img.NeedsRebuild {
		color = "red"
	}

	if _, err := writer.WriteString(fmt.Sprintf("\"%s\" [fillcolor=%s style=filled];\n", img.Name, color)); err != nil {
		return err
	}

	for _, child := range node.Children() {
		if _, err := writer.WriteString(fmt.Sprintf("\"%s\" -> \"%s\";\n", img.Name, child.Image.Name)); err != nil {
			return err
		}
	}
	return nil
}
