// Package client expose an embed filesystem to hold our Svelte static site build.
// It is used as base to generate "dib build" html report outputs.
package client

import "embed"

const AssetsRootDir = "build"

//go:embed build/*
var AssetsFS embed.FS
