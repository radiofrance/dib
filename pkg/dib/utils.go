package dib

import (
	"sort"

	"github.com/pterm/pterm"
	"github.com/radiofrance/dib/pkg/dag"
)

// printDAGViz print a human readable viz of current dag in a tree.
func printDAGViz(graph []string) error {
	var list []pterm.BulletListItem
	for _, image := range graph {
		list = append(list, pterm.BulletListItem{
			Level: 0,
			Text:  image,
		})
	}

	return pterm.DefaultBulletList.WithItems(list).Render()
}

func GetSortedIMG(graph dag.DAG) []string {
	var buildPlannedImg []string
	graph.WalkInDepth(func(node *dag.Node) {
		if node.Image.NeedsRebuild {
			buildPlannedImg = append(buildPlannedImg, node.Image.Name)
		}
	})
	sort.Strings(buildPlannedImg)
	return buildPlannedImg
}
