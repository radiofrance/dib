package docker

import (
	"github.com/radiofrance/dib/exec"
	"github.com/sirupsen/logrus"
)

// ImageBuilder builds an image using the docker command-line executable.
type ImageBuilder struct {
	exec   exec.Executor
	dryRun bool
}

// ImageBuilderOpts holds the options to be used to build the image.
type ImageBuilderOpts struct {
	// Path to the build context.
	Context string
	// Name of the tag to build, same as passed to the '-t' flag of the docker build command.
	Tag string
}

// NewImageBuilder creates a new instance of an ImageBuilder.
func NewImageBuilder(executor exec.Executor, dryRun bool) *ImageBuilder {
	return &ImageBuilder{executor, dryRun}
}

// Build the image using the docker executable.
// If the image is built successfully, the image will be pushed to the registry.
func (b ImageBuilder) Build(opts ImageBuilderOpts) error {
	if b.dryRun {
		logrus.Infof("[DRY-RUN] docker build --no-cache --pull -t %s %s", opts.Tag, opts.Context)
		logrus.Infof("[DRY-RUN] docker push %s", opts.Tag)
		return nil
	}

	err := b.exec.ExecuteStdout("docker", "build", "--no-cache", "--pull", "-t", opts.Tag, opts.Context)
	if err != nil {
		return err
	}

	err = b.exec.ExecuteStdout("docker", "push", opts.Tag)
	if err != nil {
		return err
	}

	return nil
}
