package dag

import (
	"sync"

	"golang.org/x/sync/errgroup"
)

// NodeVisitorFunc visits a node of the graph.
type NodeVisitorFunc func(*Node)

// NodeVisitorFuncErr visits a node of the graph, and can return an error.
type NodeVisitorFuncErr func(*Node) error

// Node represents a node of a graph.
type Node struct {
	Image *Image
	Files []string

	waitCond *sync.Cond
	done     bool

	parents  []*Node
	children []*Node
}

// NewNode creates a new instance of a Node.
func NewNode(image *Image) *Node {
	return &Node{
		Image:    image,
		waitCond: sync.NewCond(&sync.Mutex{}),
	}
}

// AddChild adds a child node and add the current node to its parents.
func (n *Node) AddChild(node *Node) {
	n.children = append(n.children, node)

	node.parents = append(node.parents, n)
}

// Children returns the children of the node.
func (n *Node) Children() []*Node {
	return n.children
}

// Parents returns the parents of the node.
func (n *Node) Parents() []*Node {
	return n.parents
}

// walk applies the visitor func to the current node, then to every children nodes, recursively.
func (n *Node) walk(visitor NodeVisitorFunc) {
	visitor(n)
	for _, childNode := range n.children {
		childNode.walk(visitor)
	}
}

// walkErr applies the visitor func to the current node, then to every children nodes, recursively.
// If an error occurs, it stops traversing the graph and returns the error immediately.
func (n *Node) walkErr(visitor NodeVisitorFuncErr) error {
	err := visitor(n)
	if err != nil {
		return err
	}
	for _, childNode := range n.children {
		err = childNode.walkErr(visitor)
		if err != nil {
			return err
		}
	}
	return nil
}

// walkAsyncErr applies the visitor func to the current node, then to every children nodes, asynchronously.
// If an error occurs, it stops traversing the graph and returns the error immediately.
func (n *Node) walkAsyncErr(visitor NodeVisitorFuncErr) error {
	errG := new(errgroup.Group)
	errG.Go(func() error {
		return visitor(n)
	})
	for _, childNode := range n.children {
		errG.Go(func() error {
			return childNode.walkAsyncErr(visitor)
		})
	}
	return errG.Wait()
}

// walkInDepth makes a depth-first recursive walk through the graph.
// It applies the visitor func to every children node, then to the current node itself.
func (n *Node) walkInDepth(visitor NodeVisitorFunc) {
	for _, childNode := range n.children {
		childNode.walkInDepth(visitor)
	}
	visitor(n)
}

func (n *Node) AddFile(file string) {
	n.Files = append(n.Files, file)
}
