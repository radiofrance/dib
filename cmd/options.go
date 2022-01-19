package main

import (
	cli "github.com/jawher/mow.cli"
)

const (
	defaultRegistryURL      = "eu.gcr.io/my-test-repository"
	defaultReferentialImage = "dib-referential"
)

type buildOpts struct {
	buildPath            string
	disableGenerateGraph bool
	disableJunitReports  bool
	disableRunTests      bool
	dryRun               bool
	forceRebuild         bool
	localOnly            bool
	referentialImage     string
	registryURL          string
	retagLatest          bool
}

func defaultOpts(opts *buildOpts, cmd *cli.Cmd) {
	cmd.Spec = "[OPTIONS] [BUILD_PATH]"

	desc := `Path to the directory you want to build All Dockerfiles within this directory will be recursively 
found and added to the build graph. You can provide any subdirectory if you want to focus on a reduced set of images, 
as long as it has at least one Dockerfile in it.

It is also expected that one of the director in this path contains a .docker-version file. This directory will 
be considered as the root directory for the hash generation and comparison`

	cmd.StringArgPtr(&opts.buildPath, "BUILD_PATH", "docker", desc)
	cmd.StringOptPtr(&opts.registryURL, "registry-url", defaultRegistryURL, "Docker registry URL where images are stored.")
	cmd.StringOptPtr(&opts.referentialImage, "referential-image", defaultReferentialImage, "Name of an image on "+
		"the registry. This image will serve as a reference for checking build completion of previous dib runs. Tags will be"+
		"added to this image but it has no other purpose.")
}
