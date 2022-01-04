package graphviz

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/exec"

	"github.com/radiofrance/dib/dag"
)

func GenerateGraph(dag *dag.DAG, outputDir string) error {
	if err := GenerateDotviz(dag, path.Join(outputDir, "dib.dot")); err != nil {
		return err
	}
	shell := &exec.ShellExecutor{
		Dir: outputDir,
	}
	if _, err := shell.Execute("dot", "-Tpng", "dib.dot", "-o", "dib.png"); err != nil {
		return err
	}
	return nil
}

func GenerateDotviz(dag *dag.DAG, output string) error {
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

	for _, img := range dag.Images {
		if err := generateDotvizImg(img, writer); err != nil {
			return err
		}
	}
	if _, err := writer.WriteString("}"); err != nil {
		return err
	}
	return writer.Flush()
}

func generateDotvizImg(img *dag.Image, writer *bufio.Writer) error {
	color := "white"
	switch {
	case img.RebuildDone && img.RetagDone:
		return fmt.Errorf("image %s has both RebuildDone and RetagDone", img.Name)
	case img.RebuildDone:
		color = "red"
	case img.RetagDone:
		color = "yellow"
	}

	if _, err := writer.WriteString(fmt.Sprintf("\"%s\" [fillcolor=%s style=filled];\n", img.Name, color)); err != nil {
		return err
	}

	for _, child := range img.Children {
		if _, err := writer.WriteString(fmt.Sprintf("\"%s\" -> \"%s\";\n", img.Name, child.Name)); err != nil {
			return err
		}
	}
	for _, child := range img.Children {
		if err := generateDotvizImg(child, writer); err != nil {
			return err
		}
	}
	return nil
}
