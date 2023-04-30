package dag

import (
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
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

// WalkAsyncErr applies the visitor func to every children nodes, asynchronously.
// If an error occurs, it stops traversing the graph and returns the error immediately.
func (d *DAG) WalkAsyncErr(visitor NodeVisitorFuncErr) error {
	errG := new(errgroup.Group)
	for _, node := range d.nodes {
		node := node
		errG.Go(func() error {
			return node.WalkAsyncErr(visitor)
		})
	}
	return errG.Wait()
}

// WalkInDepth makes a depth-first recursive walk through the graph.
func (d *DAG) WalkInDepth(visitor NodeVisitorFunc) {
	for _, node := range d.nodes {
		node.WalkInDepth(visitor)
	}
}

func (d *DAG) ListImage() string {
	imagesList := make(map[string]Image)
	d.Walk(func(node *Node) {
		imagesList[node.Image.ShortName] = *node.Image
	})
	strImagesList, err := yaml.Marshal(imagesList)
	if err != nil {
		return err.Error()
	}
	return string(strImagesList)
}
