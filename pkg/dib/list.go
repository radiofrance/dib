package dib

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/graphviz"
)

const (
	ConsoleFormat        = "console"
	GraphvizFormat       = "graphviz"
	GoTemplateFileFormat = "go-template-file"
)

type ListOpts struct {
	// Root options
	BuildPath        string `mapstructure:"build_path"`
	RegistryURL      string `mapstructure:"registry_url"`
	PlaceholderTag   string `mapstructure:"placeholder_tag"`
	HashListFilePath string `mapstructure:"hash_list_file_path"`

	// List specific options
	Output   string   `mapstructure:"output,omitempty"`
	BuildArg []string `mapstructure:"build_arg,omitempty"`
}

type FormatOpts struct {
	Type         string
	TemplatePath string
}

func GenerateList(graph *dag.DAG, opts FormatOpts) error {
	imagesList := GetImagesList(graph)

	switch opts.Type {
	case ConsoleFormat:
		err := renderConsoleOutput(imagesList)
		if err != nil {
			return fmt.Errorf("renderConsoleOutput: %w", err)
		}
	case GraphvizFormat:
		output := graphviz.GenerateRawOutput(graph)
		fmt.Println(output) //nolint:forbidigo
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

	if output == GraphvizFormat {
		formatOpts.Type = GraphvizFormat
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
func renderConsoleOutput(imagesList []dag.Image) error {
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
			Row: tw.CellConfig{
				Formatting: tw.CellFormatting{AutoWrap: tw.WrapNone},
			},
		}),
	)

	var data [][]string
	for _, image := range imagesList {
		data = append(data, []string{image.ShortName, image.Hash})
	}

	err := table.Bulk(data)
	if err != nil {
		return err
	}

	table.Header([]string{"Name", "Hash"})

	err = table.Render()
	if err != nil {
		return err
	}

	return nil
}
