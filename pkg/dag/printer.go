package dag

import (
	"io"
	"strings"

	"github.com/pterm/pterm"
)

// DefaultTree contains standards, which can be used to render a TreePrinter.
var DefaultTree = TreePrinter{
	TreeStyle:            &pterm.ThemeDefault.TreeStyle,
	TextStyle:            &pterm.ThemeDefault.TreeTextStyle,
	TopRightCornerString: "└",
	HorizontalString:     "─",
	TopRightDownString:   "├",
	VerticalString:       "│",
	RightDownLeftString:  "┬",
	Indent:               2,
}

// TreePrinter is able to render a list.
type TreePrinter struct {
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

// WithTreeStyle returns a new list with a specific tree style.
func (p TreePrinter) WithTreeStyle(style *pterm.Style) *TreePrinter {
	p.TreeStyle = style
	return &p
}

// WithTopRightCornerString returns a new list with a specific TopRightCornerString.
func (p TreePrinter) WithTopRightCornerString(s string) *TreePrinter {
	p.TopRightCornerString = s
	return &p
}

// WithTopRightDownStringOngoing returns a new list with a specific TopRightDownString.
func (p TreePrinter) WithTopRightDownStringOngoing(s string) *TreePrinter {
	p.TopRightDownString = s
	return &p
}

// WithHorizontalString returns a new list with a specific HorizontalString.
func (p TreePrinter) WithHorizontalString(s string) *TreePrinter {
	p.HorizontalString = s
	return &p
}

// WithVerticalString returns a new list with a specific VerticalString.
func (p TreePrinter) WithVerticalString(s string) *TreePrinter {
	p.VerticalString = s
	return &p
}

// WithRoot returns a new list with a specific Root.
func (p TreePrinter) WithRoot(root *Node) *TreePrinter {
	p.Root = root
	return &p
}

// WithIndent returns a new list with a specific amount of spacing between the levels.
// Indent must be at least 1.
func (p TreePrinter) WithIndent(indent int) *TreePrinter {
	if indent < 1 {
		indent = 1
	}
	p.Indent = indent
	return &p
}

// Render prints the list to the terminal.
func (p TreePrinter) Render() error {
	s, _ := p.Srender()
	pterm.Fprintln(p.Writer, s)

	return nil
}

// Srender renders the list as a string.
func (p TreePrinter) Srender() (string, error) {
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

// walkOverTree is a recursive function,
// which analyzes a TreePrinter and connects the items with specific characters.
// Returns TreePrinter as string.
func walkOverTree(nodes []*Node, printer TreePrinter, prefix string) string {
	var ret string
	for nodeIndex, node := range nodes {
		if len(nodes) > nodeIndex+1 { // if not last in nodes
			if len(node.Children()) == 0 { // if there are no children
				ret += prefix + printer.TreeStyle.Sprint(printer.TopRightDownString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent) +
					printer.TextStyle.Sprint(node.Image.Name) + "\n"
			} else { // if there are children
				ret += prefix + printer.TreeStyle.Sprint(printer.TopRightDownString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent-1) +
					printer.TreeStyle.Sprint(printer.RightDownLeftString) +
					printer.TextStyle.Sprint(node.Image.Name) + "\n"
				ret += walkOverTree(node.Children(), printer,
					prefix+printer.TreeStyle.Sprint(printer.VerticalString)+strings.Repeat(" ", printer.Indent-1))
			}
		} else if len(nodes) == nodeIndex+1 { // if last in nodes
			if len(node.Children()) == 0 { // if there are no children
				ret += prefix + printer.TreeStyle.Sprint(printer.TopRightCornerString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent) +
					printer.TextStyle.Sprint(node.Image.Name) + "\n"
			} else { // if there are children
				ret += prefix + printer.TreeStyle.Sprint(printer.TopRightCornerString) +
					strings.Repeat(printer.TreeStyle.Sprint(printer.HorizontalString), printer.Indent-1) +
					printer.TreeStyle.Sprint(printer.RightDownLeftString) +
					printer.TextStyle.Sprint(node.Image.Name) + "\n"
				ret += walkOverTree(node.Children(), printer,
					prefix+strings.Repeat(" ", printer.Indent))
			}
		}
	}
	return ret
}
