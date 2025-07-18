package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

const fmTemplate = `---
title: "%s"
---
`

var cmdDocPath string

func docgenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "docgen",
		Short:        "Generate the documentation for the CLI commands.",
		Hidden:       true,
		RunE:         docgenAction,
		SilenceUsage: true,
	}
	cmd.Flags().StringVar(&cmdDocPath, "path", "./docs/cmd",
		"path to write the generated documentation to")

	return cmd
}

func docgenAction(_ *cobra.Command, _ []string) error {
	if err := os.MkdirAll(cmdDocPath, 0o750); err != nil {
		return err
	}

	return doc.GenMarkdownTreeCustom(rootCmd, cmdDocPath, filePrepender, linkHandler)
}

func filePrepender(filename string) string {
	name := filepath.Base(filename)
	base := strings.TrimSuffix(name, path.Ext(name))
	title := strings.ReplaceAll(base, "_", " ")

	return fmt.Sprintf(fmTemplate, title)
}

func linkHandler(name string) string {
	base := strings.TrimSuffix(name, path.Ext(name))
	return "../" + strings.ToLower(base) + "/"
}
