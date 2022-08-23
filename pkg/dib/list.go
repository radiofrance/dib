package dib

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/olekukonko/tablewriter"
	"github.com/radiofrance/dib/pkg/dag"
)

const (
	ConsoleFormat        = "console"
	GoTemplateFileFormat = "go-template-file"
)

type FormatOpts struct {
	Type         string
	TemplatePath string
}

func GenerateList(graph *dag.DAG, opts FormatOpts) error {
	imagesList := GetImagesList(graph)

	switch opts.Type {
	case ConsoleFormat:
		renderConsoleOutput(imagesList)
	case GoTemplateFileFormat:
		outputTemplate, err := template.ParseFiles(opts.TemplatePath)
		if err != nil {
			return fmt.Errorf("failed to parse go-template file : %w", err)
		}

		err = outputTemplate.Execute(os.Stdout, imagesList)
		if err != nil {
			return fmt.Errorf("failed to render go-template file : %w", err)
		}
	}

	return nil
}

// GetImagesList iterate over DAG nodes and return a slice of Image sorted by their ShortName.
func GetImagesList(graph *dag.DAG) []dag.Image {
	imagesList := make(map[string]dag.Image)
	graph.Walk(func(node *dag.Node) {
		imagesList[node.Image.ShortName] = *node.Image
	})

	// Sort Images by name
	var sortedImagesList []dag.Image
	for _, image := range imagesList {
		sortedImagesList = append(sortedImagesList, image)
	}

	sort.SliceStable(sortedImagesList, func(i, j int) bool {
		return sortedImagesList[i].ShortName < sortedImagesList[j].ShortName
	})

	return sortedImagesList
}

// ParseOutputOptions parse value of the "--output" flag and ensure they are valid.
// Currently, we only support the "go-template-file" and "console" output.
func ParseOutputOptions(output string) (FormatOpts, error) {
	formatOpts := FormatOpts{}
	if output == "" || output == ConsoleFormat {
		formatOpts.Type = ConsoleFormat
		return formatOpts, nil
	}

	parsed := strings.Split(output, "=")
	switch parsed[0] {
	case GoTemplateFileFormat:
		if len(parsed) == 1 {
			return formatOpts, fmt.Errorf("you need to provide a path to template file when using \"go-template-file\" options")
		}
		formatOpts.Type = GoTemplateFileFormat
		formatOpts.TemplatePath = parsed[1]
	default:
		return formatOpts, fmt.Errorf("\"%s\" is not a valid output format", output)
	}

	return formatOpts, nil
}

// renderConsoleOutput displays the list of image in stdout as a nice table.
func renderConsoleOutput(imagesList []dag.Image) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT) //nolint:nosnakecase
	table.SetAlignment(tablewriter.ALIGN_LEFT)       //nolint:nosnakecase
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")

	var data [][]string
	for _, image := range imagesList {
		data = append(data, []string{image.ShortName, image.Hash})
	}
	table.AppendBulk(data)

	table.SetHeader([]string{"Name", "Hash"})
	table.Render()
}
