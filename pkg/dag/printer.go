package dag

import (
	"fmt"
	"io"
	"strings"

	"github.com/pterm/pterm"
)

var defaultPrinter = GraphPrinter{
	TopRightCornerString: "└",
	TopRightDownString:   "├",
	HorizontalString:     "─",
	VerticalString:       "│",
	RightDownLeftString:  "┬",
	Indent:               3,
}

type GraphPrinter struct {
	Root                 *Node
	TreeStyle            *pterm.Style
	TextStyle            *pterm.Style
	TopRightCornerString string
	TopRightDownString   string
	HorizontalString     string
	VerticalString       string
	RightDownLeftString  string
	Indent               int
	Writer               io.Writer
}

// WithRoot returns a new GraphPrinter with a specific Root node.
func (p GraphPrinter) WithRoot(root *Node) *GraphPrinter {
	p.Root = root
	return &p
}

// Srender renders the graph as a string.
func (p GraphPrinter) Srender() string {
	if p.TreeStyle == nil {
		p.TreeStyle = pterm.NewStyle()
	}

	if p.TextStyle == nil {
		p.TextStyle = pterm.NewStyle()
	}

	var result string
	if p.Root.Image.Name != "" {
		result += p.TextStyle.Sprint(p.Root.Image.Name) + "\n"
	}

	result += walkOverTree(p.Root.Children(), p, "")

	return result
}

func walkOverTree(nodes []*Node, printer GraphPrinter, prefix string) string {
	var result strings.Builder

	for nodeIndex, node := range nodes {
		if node.Image == nil {
			continue
		}

		txt := fmt.Sprintf("%s [%s]\n", node.Image.ShortName, node.Image.Hash)
		if len(nodes) > nodeIndex+1 { // if not last in nodes
			if len(node.Children()) == 0 { // if there are no children
				result.WriteString(prefix + printer.TreeStyle.Sprint(printer.TopRightDownString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent) +
					printer.TextStyle.Sprint(txt))
			} else { // if there are children
				result.WriteString(prefix + printer.TreeStyle.Sprint(printer.TopRightDownString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent-1) +
					printer.TreeStyle.Sprint(printer.RightDownLeftString) +
					printer.TextStyle.Sprint(txt))
				result.WriteString(walkOverTree(node.Children(), printer,
					prefix+printer.TreeStyle.Sprint(printer.VerticalString)+strings.Repeat(" ", printer.Indent-1)))
			}
		} else if len(nodes) == nodeIndex+1 { // if last in nodes
			if len(node.Children()) == 0 { // if there are no children
				result.WriteString(prefix + printer.TreeStyle.Sprint(printer.TopRightCornerString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent) +
					printer.TextStyle.Sprint(txt))
			} else { // if there are children
				result.WriteString(prefix + printer.TreeStyle.Sprint(printer.TopRightCornerString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent-1) +
					printer.TreeStyle.Sprint(printer.RightDownLeftString) +
					printer.TextStyle.Sprint(txt))
				result.WriteString(walkOverTree(node.Children(), printer,
					prefix+strings.Repeat(" ", printer.Indent)))
			}
		}
	}

	return result.String()
}
