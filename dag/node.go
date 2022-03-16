package dag

import "sync"

// NodeVisitorFunc visits a node of the graph.
type NodeVisitorFunc func(*Node)

// NodeVisitorFuncAsyncErr visits a node of the graph asynchronously and returns an error in a channel.
type NodeVisitorFuncAsyncErr func(*Node, *sync.WaitGroup, chan error)

// NodeVisitorFuncErr visits a node of the graph, and can return an error.
type NodeVisitorFuncErr func(*Node) error

// LogErrorFunc handles an error caught by a walk function
type LogErrorFunc func(error)

// Node represents a node of a graph.
type Node struct {
	Image    *Image
	WaitCond *sync.Cond

	parents  []*Node
	children []*Node
}

// NewNode creates a new instance of a Node.
func NewNode(image *Image) *Node {
	return &Node{
		Image:    image,
		WaitCond: sync.NewCond(&sync.Mutex{}),
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

// Walk applies the visitor func to the current node, then to every children nodes, recursively.
func (n *Node) Walk(visitor NodeVisitorFunc) {
	visitor(n)
	for _, childNode := range n.children {
		childNode.Walk(visitor)
	}
}

// WalkAsyncErr applies the visitor func to the current node, then to every children nodes, asynchronously.
// and waits for all functions to complete
func (n *Node) WalkAsyncErr(visitor NodeVisitorFuncAsyncErr, wg *sync.WaitGroup, errChan chan error) {
	wg.Add(1)
	go visitor(n, wg, errChan)
	for _, childNode := range n.children {
		childNode.WalkAsyncErr(visitor, wg, errChan)
	}
}

// WalkErr applies the visitor func to the current node, then to every children nodes, recursively.
// If an error occurs, it stops traversing the graph and returns the error immediately.
func (n *Node) WalkErr(visitor NodeVisitorFuncErr) error {
	err := visitor(n)
	if err != nil {
		return err
	}
	for _, childNode := range n.children {
		err = childNode.WalkErr(visitor)
		if err != nil {
			return err
		}
	}
	return nil
}

// WalkInDepth makes a depth-first recursive walk through the graph.
// It applies the visitor func to every children node, then to the current node itself.
func (n *Node) WalkInDepth(visitor NodeVisitorFunc) {
	for _, childNode := range n.children {
		childNode.WalkInDepth(visitor)
	}
	visitor(n)
}
