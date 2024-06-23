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
	FormatConsole        = "console"
	FormatGoTemplateFile = "go-template-file"
)

type ListOpts struct {
	// Root options
	BuildPath        string `mapstructure:"build_path"`
	RegistryURL      string `mapstructure:"registry_url"`
	PlaceholderTag   string `mapstructure:"placeholder_tag"`
	HashListFilePath string `mapstructure:"hash_list_file_path"`

	// List specific options
	Output   string   `mapstructure:"output"`
	BuildArg []string `mapstructure:"build_arg"`
}

type FormatOpts struct {
	Type         string
	TemplatePath string
}

func GenerateList(graph *dag.DAG, opts FormatOpts) error {
	imagesList := GetImagesList(graph)

	switch opts.Type {
	case FormatConsole:
		renderConsoleOutput(imagesList)
	case FormatGoTemplateFile:
		return renderGoTemplateOutput(opts.TemplatePath, imagesList)
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

// ParseListOutputOptions parse value of the "--output" flag and ensure they are valid.
// Currently, we only support the "go-template-file" and "console" output.
func ParseListOutputOptions(listOutput string) (FormatOpts, error) {
	formatOpts := FormatOpts{}
	if listOutput == "" || listOutput == FormatConsole {
		formatOpts.Type = FormatConsole
		return formatOpts, nil
	}

	parsed := strings.Split(listOutput, "=")
	switch parsed[0] {
	case FormatGoTemplateFile:
		if len(parsed) == 1 {
			return formatOpts, fmt.Errorf("you need to provide a path to template file when using \"go-template-file\" options")
		}
		formatOpts.Type = FormatGoTemplateFile
		formatOpts.TemplatePath = parsed[1]
	default:
		return formatOpts, fmt.Errorf("\"%s\" is not a valid output format", listOutput)
	}

	return formatOpts, nil
}

// renderConsoleOutput displays the list of images in stdout as a nice table.
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

// renderGoTemplateOutput displays the list of images using specified go template file.
// A slice of dag.Image is passed to the model when it executes.
func renderGoTemplateOutput(templatePath string, imagesList []dag.Image) error {
	outputTemplate, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse go-template file : %w", err)
	}

	if err := outputTemplate.Execute(os.Stdout, imagesList); err != nil {
		return fmt.Errorf("failed to render go-template file : %w", err)
	}

	return nil
}
