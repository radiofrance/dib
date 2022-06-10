package dag_test

import (
	"testing"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/stretchr/testify/assert"
)

func Test_NewNode(t *testing.T) {
	t.Parallel()

	image := &dag.Image{}
	node := dag.NewNode(image)

	assert.Same(t, image, node.Image)
	assert.Empty(t, node.Children())
	assert.Empty(t, node.Parents())
}

func Test_AddChild_SetsParentNode(t *testing.T) {
	t.Parallel()

	node := &dag.Node{}
	child := &dag.Node{}

	node.AddChild(child)

	children := node.Children()
	assert.Len(t, children, 1)
	assert.Same(t, child, children[0])

	parents := child.Parents()
	assert.Len(t, parents, 1)
	assert.Same(t, node, parents[0])
}
