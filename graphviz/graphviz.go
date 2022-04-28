package graphviz

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"

	"github.com/radiofrance/dib/dag"
)

const distDirectory = "dist"

func GenerateGraph(dag *dag.DAG) error {
	if err := os.MkdirAll(distDirectory, 0o755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", distDirectory, err)
	}

	g := graphviz.New()
	graph, _ := GenerateDotvizGraph(g, dag)

	_ = g.RenderFilename(graph, graphviz.XDOT, path.Join(distDirectory, "dib2.dot"))

	if err := g.RenderFilename(graph, graphviz.PNG, path.Join(distDirectory, "dib2.png")); err != nil {
		return fmt.Errorf("error rendering graph: %w", err)
	}

	return nil
}

func GenerateDotvizGraph(g *graphviz.Graphviz, graph *dag.DAG) (*cgraph.Graph, error) {
	g.SetLayout(graphviz.DOT)

	gviz, _ := g.Graph(graphviz.Directed, graphviz.Name("images"))
	gviz.SetID("images")
	gviz.SetRankDir("LR")

	_ = []string{
		"digraph images {",
		"rankdir = \"LR\";",
		"node[fontsize = 10, shape = box, height = 0.25];",
		"edge [fontsize = 10];\n",
	}

	_ = graph.WalkErr(func(node *dag.Node) error {
		img := node.Image
		color := "white"
		switch {
		case img.NeedsRebuild && img.NeedsRetag:
			return fmt.Errorf("image %s has both RebuildDone and RetagDone", img.Name)
		case img.NeedsRebuild:
			color = "red"
		case img.NeedsRetag:
			color = "yellow"
		}

		n, _ := getOrCreateNode(gviz, img.Name)
		n.SetFillColor(color)
		n.SetStyle(cgraph.FilledNodeStyle)
		n.SetFontSize(10)
		n.SetPos(0, 0)
		n.SetShowBoxes(1)
		n.SetShape(cgraph.BoxShape)
		n.SetHeight(0.25)

		for _, child := range node.Children() {
			c, _ := getOrCreateNode(gviz, child.Image.Name)

			e, _ := gviz.CreateEdge("", n, c)
			e.SetFontSize(10)
		}

		return nil
	})

	return gviz, nil
}

func getOrCreateNode(gviz *cgraph.Graph, name string) (*cgraph.Node, error) {
	n, _ := gviz.Node(name)
	if n == nil {
		n, _ = gviz.CreateNode(name)
	}

	return n, nil
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
	switch {
	case img.NeedsRebuild && img.NeedsRetag:
		return fmt.Errorf("image %s has both RebuildDone and RetagDone", img.Name)
	case img.NeedsRebuild:
		color = "red"
	case img.NeedsRetag:
		color = "yellow"
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
