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

// Render prints the graph to the terminal.
func (p GraphPrinter) Render() error {
	s, _ := p.Srender()
	pterm.Fprintln(p.Writer, s)

	return nil
}

// Srender renders the graph as a string.
func (p GraphPrinter) Srender() (string, error) {
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
	return result, nil
}

func walkOverTree(nodes []*Node, printer GraphPrinter, prefix string) string {
	var res string
	for nodeIndex, node := range nodes {
		txt := fmt.Sprintf("%s [%s]\n", node.Image.ShortName, node.Image.Hash)
		if len(nodes) > nodeIndex+1 { // if not last in nodes
			if len(node.Children()) == 0 { // if there are no children
				res += prefix + printer.TreeStyle.Sprint(printer.TopRightDownString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent) +
					printer.TextStyle.Sprint(txt)
			} else { // if there are children
				res += prefix + printer.TreeStyle.Sprint(printer.TopRightDownString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent-1) +
					printer.TreeStyle.Sprint(printer.RightDownLeftString) +
					printer.TextStyle.Sprint(txt)
				res += walkOverTree(node.Children(), printer,
					prefix+printer.TreeStyle.Sprint(printer.VerticalString)+strings.Repeat(" ", printer.Indent-1))
			}
		} else if len(nodes) == nodeIndex+1 { // if last in nodes
			if len(node.Children()) == 0 { // if there are no children
				res += prefix + printer.TreeStyle.Sprint(printer.TopRightCornerString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent) +
					printer.TextStyle.Sprint(txt)
			} else { // if there are children
				res += prefix + printer.TreeStyle.Sprint(printer.TopRightCornerString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent-1) +
					printer.TreeStyle.Sprint(printer.RightDownLeftString) +
					printer.TextStyle.Sprint(txt)
				res += walkOverTree(node.Children(), printer,
					prefix+strings.Repeat(" ", printer.Indent))
			}
		}
	}
	return res
}
