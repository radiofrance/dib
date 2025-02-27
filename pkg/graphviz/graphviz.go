package graphviz

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/exec"
)

const (
	// graphDot is the name of the file containing the raw graphviz dot language representation of the dib graph.
	graphDot = "dib.dot"

	// graphPng is the final file inside we put dib graph.
	graphPng = "dib.png"
)

// GenerateGraph generates a graphviz representation (png) of the dag.DAG in the given report.Report rootDir.
func GenerateGraph(dag *dag.DAG, reportRootDir string) error {
	rawGraphvizOutput := GenerateRawOutput(dag)
	graphvizFile := path.Join(reportRootDir, graphDot)
	if err := os.WriteFile(graphvizFile, []byte(rawGraphvizOutput), 0o644); err != nil {
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

// GenerateRawOutput generates the raw graphviz dot language from the given dag.DAG.
func GenerateRawOutput(graph *dag.DAG) string {
	rawGraphvizDotLang := []string{
		"digraph images {\n",
		"  rankdir = \"LR\";\n",
		"  node[fontsize=10, shape=cds, height=0.4];\n",
		"  edge[fontsize=10, arrowhead=vee];\n",
		"\n",
	}

	if graph != nil {
		graph.Walk(func(node *dag.Node) {
			img := node.Image
			color := "white"
			if img.NeedsRebuild {
				color = "red"
			}

			rawGraphvizDotLang = append(rawGraphvizDotLang, fmt.Sprintf(
				"  \"%s\" [fillcolor=%s, style=filled];\n",
				img.Name,
				color,
			))

			for _, child := range node.Children() {
				rawGraphvizDotLang = append(rawGraphvizDotLang, fmt.Sprintf(
					"  \"%s\" -> \"%s\" [dir=forward];\n",
					img.Name,
					child.Image.Name,
				))
			}
		})
	}

	rawGraphvizDotLang = append(rawGraphvizDotLang, "}\n")

	return strings.Join(rawGraphvizDotLang, "")
}
