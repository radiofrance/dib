package dag_test

import (
	"errors"
	"testing"

	"github.com/radiofrance/dib/dag"
	"github.com/stretchr/testify/assert"
)

func Test_AddNode(t *testing.T) {
	t.Parallel()

	DAG := &dag.DAG{}
	node := &dag.Node{}

	DAG.AddNode(node)

	nodes := DAG.Nodes()
	assert.Len(t, nodes, 1)
	assert.Same(t, node, nodes[0])
}

func createDAG() *dag.DAG {
	root1 := &dag.Node{}
	root1child1 := &dag.Node{}
	root1.AddChild(root1child1)
	root1child2 := &dag.Node{}
	root1.AddChild(root1child2)

	root2 := &dag.Node{}
	root2child1 := &dag.Node{}
	root2.AddChild(root2child1)
	root2child1subchild := &dag.Node{}
	root2child1.AddChild(root2child1subchild)

	DAG := dag.DAG{}

	DAG.AddNode(root1)
	DAG.AddNode(root2)

	return &DAG
}

func Test_Walk_RunsAllNodes(t *testing.T) {
	t.Parallel()

	tracking := make(map[*dag.Node]bool)

	DAG := createDAG()
	DAG.Walk(func(node *dag.Node) {
		for _, parent := range node.Parents() {
			_, ok := tracking[parent]

			assert.True(t, ok, "The visitor func is supposed to run on parent nodes before children")
		}
		for _, child := range node.Children() {
			_, ok := tracking[child]

			assert.False(t, ok, "The visitor func is supposed to run on parent nodes before children")
		}
		tracking[node] = true
	})

	// Assert that the visitor func ran on every node.
	assert.Len(t, tracking, 6)
}

func Test_WalkErr_RunsAllNodesWhenNoError(t *testing.T) {
	t.Parallel()

	tracking := make(map[*dag.Node]bool)

	DAG := createDAG()
	err := DAG.WalkErr(func(node *dag.Node) error {
		for _, parent := range node.Parents() {
			_, ok := tracking[parent]

			assert.True(t, ok, "The visitor func is supposed to run on parent nodes before children")
		}
		for _, child := range node.Children() {
			_, ok := tracking[child]

			assert.False(t, ok, "The visitor func is supposed to run on parent nodes before children")
		}
		tracking[node] = true
		return nil
	})

	assert.NoError(t, err)

	// Assert that the visitor func ran on every node.
	assert.Len(t, tracking, 6) // Total number of nodes is 5
}

func Test_WalkErr_StopsOnError(t *testing.T) {
	t.Parallel()

	tracking := make(map[*dag.Node]bool)

	DAG := createDAG()
	subchildNode := DAG.Nodes()[1].Children()[0]
	subchildNodeError := errors.New("something went wrong")

	err := DAG.WalkErr(func(node *dag.Node) error {
		tracking[node] = true

		if node == subchildNode {
			return subchildNodeError
		}
		return nil
	})

	assert.Error(t, err)
	assert.EqualError(t, err, subchildNodeError.Error())

	// Assert that the visitor stopped and didn't run on the last child.
	assert.Len(t, tracking, 5) // Total number of nodes is 6
}

func Test_WalkInDepth_RunsAllNodes(t *testing.T) {
	t.Parallel()

	tracking := make(map[*dag.Node]bool)

	DAG := createDAG()
	DAG.WalkInDepth(func(node *dag.Node) {
		for _, parent := range node.Parents() {
			_, ok := tracking[parent]

			assert.False(t, ok, "The visitor func is supposed to run on children nodes before parents")
		}
		for _, child := range node.Children() {
			_, ok := tracking[child]

			assert.True(t, ok, "The visitor func is supposed to run on children nodes before parents")
		}
		tracking[node] = true
	})

	// Assert that the visitor func ran on every node.
	assert.Len(t, tracking, 6) // Total number of nodes is 6
}
