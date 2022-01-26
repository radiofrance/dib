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

const distDirectory = "dist"

func GenerateGraph(dag *dag.DAG) error {
	if err := os.MkdirAll(distDirectory, 0o755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", distDirectory, err)
	}
	if err := GenerateDotviz(dag, path.Join(distDirectory, "dib.dot")); err != nil {
		return err
	}
	shell := &exec.ShellExecutor{
		Dir: distDirectory,
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
