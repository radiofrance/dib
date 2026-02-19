package graphviz_test

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/graphviz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GenerateGraph(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	graph, err := dib.GenerateDAG(
		path.Join(cwd, "../../test/fixtures/docker"),
		"eu.gcr.io/my-test-repository", "",
		map[string]string{})
	require.NoError(t, err)

	dir := t.TempDir()
	err = graphviz.GenerateGraph(context.Background(), graph, dir)
	require.NoError(t, err)
	assert.FileExists(t, path.Join(dir, "dib.dot"))
	assert.FileExists(t, path.Join(dir, "dib.png"))
}

func Test_GenerateRawOutput_EmptyDAG(t *testing.T) {
	t.Parallel()

	expectedDAG := "digraph images {\n" +
		"  rankdir = \"LR\";\n" +
		"  node[fontsize=10, shape=cds, height=0.4];\n" +
		"  edge[fontsize=10, arrowhead=vee];\n" +
		"\n" +
		"}\n"

	tests := []struct {
		name     string
		input    *dag.DAG
		expected string
	}{
		{
			name:     "nil graph",
			input:    nil,
			expected: expectedDAG,
		},
		{
			name:     "empty graph",
			input:    &dag.DAG{},
			expected: expectedDAG,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := graphviz.GenerateRawOutput(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func Test_GenerateRawOutput_ValidDAG(t *testing.T) {
	t.Parallel()

	sub1 := dag.NewNode(&dag.Image{
		Name: "registry.localhost/image1-child1-sub1",
	})

	child1 := dag.NewNode(&dag.Image{
		Name:         "registry.localhost/image1-child1",
		NeedsRebuild: true,
	})
	child2 := dag.NewNode(&dag.Image{
		Name:         "registry.localhost/image1-child2",
		NeedsRebuild: true,
	})

	node1 := dag.NewNode(&dag.Image{
		Name:         "registry.localhost/image1",
		NeedsRebuild: true,
	})
	node2 := dag.NewNode(&dag.Image{
		Name:         "registry.localhost/image2",
		NeedsRebuild: false,
	})
	node3 := dag.NewNode(&dag.Image{
		Name:         "registry.localhost/image3",
		NeedsRebuild: false,
	})

	child1.AddChild(sub1)
	node1.AddChild(child1)
	node1.AddChild(child2)

	inputGraph := &dag.DAG{}
	inputGraph.AddNode(node1)
	inputGraph.AddNode(node2)
	inputGraph.AddNode(node3)

	expected := "digraph images {\n" +
		"  rankdir = \"LR\";\n" +
		"  node[fontsize=10, shape=cds, height=0.4];\n" +
		"  edge[fontsize=10, arrowhead=vee];\n" +
		"\n" +
		"  \"registry.localhost/image1\" [fillcolor=red, style=filled];\n" +
		"  \"registry.localhost/image1\" -> \"registry.localhost/image1-child1\" [dir=forward];\n" +
		"  \"registry.localhost/image1\" -> \"registry.localhost/image1-child2\" [dir=forward];\n" +
		"  \"registry.localhost/image1-child1\" [fillcolor=red, style=filled];\n" +
		"  \"registry.localhost/image1-child1\" -> \"registry.localhost/image1-child1-sub1\" [dir=forward];\n" +
		"  \"registry.localhost/image1-child1-sub1\" [fillcolor=white, style=filled];\n" +
		"  \"registry.localhost/image1-child2\" [fillcolor=red, style=filled];\n" +
		"  \"registry.localhost/image2\" [fillcolor=white, style=filled];\n" +
		"  \"registry.localhost/image3\" [fillcolor=white, style=filled];\n" +
		"}\n"

	actual := graphviz.GenerateRawOutput(inputGraph)
	assert.Equal(t, expected, actual)
}
