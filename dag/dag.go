package dag

import (
	"fmt"
	"sync"
)

// DAG represents a direct acyclic graph.
type DAG struct {
	nodes []*Node // Root nodes of the graph.
}

// AddNode add a root node to the graph.
func (d *DAG) AddNode(node *Node) {
	d.nodes = append(d.nodes, node)
}

// Nodes returns the root nodes.
func (d *DAG) Nodes() []*Node {
	return d.nodes
}

// Walk recursively through the graph and apply the visitor func to every node.
func (d *DAG) Walk(visitor NodeVisitorFunc) {
	for _, node := range d.nodes {
		node.Walk(visitor)
	}
}

// WalkAsyncErr walks asynchronously through the graph and apply the visitor func to every node.
// and returns an error if one of the function failed
func (d *DAG) WalkAsyncErr(visitor NodeVisitorFuncAsyncErr, logError LogErrorFunc) error {
	wg := sync.WaitGroup{}
	errChan := make(chan error)
	for _, node := range d.nodes {
		node.WalkAsyncErr(visitor, &wg, errChan)
	}
	go func() {
		wg.Wait()
		close(errChan)
	}()

	hasError := false
	for err := range errChan {
		hasError = true
		logError(err)
	}

	if hasError {
		return fmt.Errorf("graph walk failed, see logs for more details")
	}
	return nil
}

// WalkErr applies the visitor func to every children nodes, recursively.
// If an error occurs, it stops traversing the graph and returns the error immediately.
func (d *DAG) WalkErr(visitor NodeVisitorFuncErr) error {
	for _, node := range d.nodes {
		err := node.WalkErr(visitor)
		if err != nil {
			return err
		}
	}
	return nil
}

// WalkInDepth makes a depth-first recursive walk through the graph.
func (d *DAG) WalkInDepth(visitor NodeVisitorFunc) {
	for _, node := range d.nodes {
		node.WalkInDepth(visitor)
	}
}
