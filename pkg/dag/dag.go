package dag

import (
	"cmp"
	"slices"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
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
// Each node can only be visited once, even if it has more than one parent.
func (d *DAG) Walk(visitor NodeVisitorFunc) {
	uniqueVisitor := createUniqueVisitor(visitor)
	for _, node := range d.nodes {
		node.walk(uniqueVisitor)
	}
}

// WalkErr applies the visitor func to every children nodes, recursively.
// Each node can only be visited once, even if it has more than one parent.
// If an error occurs, it stops traversing the graph and returns the error immediately.
func (d *DAG) WalkErr(visitor NodeVisitorFuncErr) error {
	uniqueVisitor := createUniqueVisitorErr(visitor)
	for _, node := range d.nodes {
		err := node.walkErr(uniqueVisitor)
		if err != nil {
			return err
		}
	}

	return nil
}

// WalkAsyncErr applies the visitor func to every children nodes, asynchronously.
// Each node can only be visited once, even if it has more than one parent.
// If an error occurs, it stops traversing the graph and returns the error immediately.
func (d *DAG) WalkAsyncErr(visitor NodeVisitorFuncErr) error {
	uniqueVisitor := createUniqueVisitorErr(visitor)

	errG := new(errgroup.Group)
	for _, node := range d.nodes {
		errG.Go(func() error {
			return node.walkAsyncErr(uniqueVisitor)
		})
	}

	return errG.Wait()
}

// WalkInDepth makes a depth-first recursive walk through the graph.
// Each node can only be visited once, even if it has more than one parent.
func (d *DAG) WalkInDepth(visitor NodeVisitorFunc) {
	uniqueVisitor := createUniqueVisitor(visitor)
	for _, node := range d.nodes {
		node.walkInDepth(uniqueVisitor)
	}
}

// WalkParallel recursively through the graph and apply the visitor func to every node, in parallel.
// Before processing a node it waits for every parent node to be completed.
func (d *DAG) WalkParallel(visitor NodeVisitorFunc) {
	waitGroup := sync.WaitGroup{}

	parallelVisitor := func(node *Node) {
		waitGroup.Go(func() {
			node.waitCond.L.Lock()
			defer node.waitCond.L.Unlock()

			for _, parent := range node.Parents() {
				parent.waitCond.L.Lock()

				for !parent.done {
					parent.waitCond.Wait()
				}

				parent.waitCond.L.Unlock()
			}

			visitor(node)

			node.done = true
			node.waitCond.Broadcast()
		})
	}

	uniqueVisitor := createUniqueVisitor(parallelVisitor)
	for _, node := range d.nodes {
		node.walk(uniqueVisitor)
	}

	waitGroup.Wait()
}

//nolint:musttag
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

// createUniqueVisitor creates a NodeVisitorFunc that wraps the original visitor,
// ensuring it only visits nodes once.
// If a node was already visited, it is ignored and the visitor func is not called.
func createUniqueVisitor(visitor NodeVisitorFunc) NodeVisitorFunc {
	visited := sync.Map{}
	uniqueVisitor := func(node *Node) {
		_, exists := visited.Load(node)
		if exists {
			return
		}

		visited.Store(node, struct{}{})

		visitor(node)
	}

	return uniqueVisitor
}

// createUniqueVisitorErr creates a NodeVisitorFuncErr that wraps the original visitor,
// ensuring it only visits nodes once.
// If a node was already visited, it is ignored and the visitor func is not called.
func createUniqueVisitorErr(visitor NodeVisitorFuncErr) NodeVisitorFuncErr {
	visited := sync.Map{}
	uniqueVisitor := func(node *Node) error {
		_, exists := visited.Load(node)
		if exists {
			return nil
		}

		visited.Store(node, struct{}{})

		return visitor(node)
	}

	return uniqueVisitor
}

func sort(a, b *Node) int {
	return cmp.Compare(
		strings.ToLower(a.Image.ShortName),
		strings.ToLower(b.Image.ShortName),
	)
}

func (d *DAG) Sprint(name string) string {
	d.WalkInDepth(func(node *Node) {
		slices.SortFunc(node.Children(), sort)
	})

	slices.SortFunc(d.nodes, sort)
	rootNode := &Node{
		Image:    &Image{Name: name},
		children: d.nodes,
	}

	return defaultPrinter.WithRoot(rootNode).Srender()
}
