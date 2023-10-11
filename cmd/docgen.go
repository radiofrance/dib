package main

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

var docgenCmd = &cobra.Command{
	Use:    "docgen",
	Short:  "Generate the documentation for the CLI commands.",
	Hidden: true,
	RunE:   docgenCmdRun,
}

func init() {
	docgenCmd.Flags().StringVar(&cmdDocPath, "path", "./docs/cmd",
		"path to write the generated documentation to")

	rootCmd.AddCommand(docgenCmd)
}

func docgenCmdRun(_ *cobra.Command, _ []string) error {
	if err := os.MkdirAll(cmdDocPath, os.ModePerm); err != nil {
		return err
	}

	err := doc.GenMarkdownTreeCustom(rootCmd, cmdDocPath, frontmatterPrepender, linkHandler)
	if err != nil {
		return err
	}
	return nil
}

func frontmatterPrepender(filename string) string {
	name := filepath.Base(filename)
	base := strings.TrimSuffix(name, path.Ext(name))
	title := strings.ReplaceAll(base, "_", " ")

	return fmt.Sprintf(fmTemplate, title)
}

func linkHandler(name string) string {
	base := strings.TrimSuffix(name, path.Ext(name))
	return "../" + strings.ToLower(base) + "/"
}
