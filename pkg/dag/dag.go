package dag

import (
	"sync"

	"github.com/radiofrance/dib/internal/logger"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// DAG represents a direct acyclic graph.
type DAG struct {
	nodes []*Node // Root nodes of the graph.
}

func (d *DAG) Print(name string) {
	rootNode := &Node{
		Image:    &Image{Name: name},
		children: d.nodes,
	}
	if err := DefaultTree.WithRoot(rootNode).Render(); err != nil {
		logger.Fatalf(err.Error())
	}
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
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

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
		}()
	}

	uniqueVisitor := createUniqueVisitor(parallelVisitor)
	for _, node := range d.nodes {
		node.walk(uniqueVisitor)
	}

	waitGroup.Wait()
}

// Filter creates a new DAG only populated by nodes that fulfil the condition checked by filterFunc.
// All children nodes that no longer have any parents in the resulting graph will become a root node of the DAG.
func (d *DAG) Filter(filterFunc func(*Node) bool) *DAG {
	filteredGraph := DAG{}
	replacements := make(map[*Node]*Node)
	var orphans []*Node

	d.Walk(func(node *Node) {
		if !filterFunc(node) {
			// The node does not fulfil the condition, skip it.
			return
		}

		// Create a replacement node containing the same Image.
		newNode := NewNode(node.Image)
		replacements[node] = newNode
		// Check if the node is orphan
		for _, parent := range node.Parents() {
			if _, ok := replacements[parent]; ok {
				// The node has at least one parent, nothing to do.
				return
			}
		}
		// The node no longer has parents, we add it to the list of orphans.
		orphans = append(orphans, node)
	})

	for _, orphan := range orphans {
		orphan.walkInDepth(createUniqueVisitor(func(node *Node) {
			replacement, ok := replacements[node]
			if !ok {
				// This node on the source graph should not be in the filtered graph.
				return
			}

			// Add new child nodes
			for _, child := range node.Children() {
				childReplacement, ok := replacements[child]
				if !ok {
					// This child node on the source graph should not be in the filtered graph.
					continue
				}
				replacement.AddChild(childReplacement)
			}
		}))
		// Add the orphan node to the root nodes of the graph.
		filteredGraph.AddNode(replacements[orphan])
	}

	return &filteredGraph
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
