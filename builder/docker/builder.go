package docker

import (
	"fmt"
	"time"

	"github.com/radiofrance/dib/exec"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
)

// ImageBuilder builds an image using the docker command-line executable.
type ImageBuilder struct {
	exec   exec.Executor
	dryRun bool
}

// NewImageBuilder creates a new instance of an ImageBuilder.
func NewImageBuilder(executor exec.Executor, dryRun bool) *ImageBuilder {
	return &ImageBuilder{executor, dryRun}
}

// Build the image using the docker executable.
// If the image is built successfully, the image will be pushed to the registry.
func (b ImageBuilder) Build(opts types.ImageBuilderOpts) error {
	if b.dryRun {
		logrus.Infof("[DRY-RUN] docker build --no-cache -t %s %s", opts.Tag, opts.Context)

		if !opts.LocalOnly {
			logrus.Infof("[DRY-RUN] docker push %s", opts.Tag)
		}
		return nil
	}
	dockerArgs := []string{
		"build",
		"--no-cache",
	}

	if opts.CreationTime != nil {
		dockerArgs = append(dockerArgs, "--label", fmt.Sprintf("org.opencontainers.image.created=%s",
			opts.CreationTime.Format(time.RFC3339)))
	}
	if opts.Authors != nil {
		dockerArgs = append(dockerArgs, "--label", fmt.Sprintf("org.opencontainers.image.authors=%s",
			*opts.Authors))
	}
	if opts.Source != nil {
		dockerArgs = append(dockerArgs, "--label", fmt.Sprintf("org.opencontainers.image.source=%s",
			*opts.Source))
	}
	if opts.Revision != nil {
		dockerArgs = append(dockerArgs, "--label", fmt.Sprintf("org.opencontainers.image.revision=%s",
			*opts.Revision))
	}

	dockerArgs = append(dockerArgs, "-t", opts.Tag, opts.Context)

	err := b.exec.ExecuteStdout("docker", dockerArgs...)
	if err != nil {
		return err
	}

	if !opts.LocalOnly {
		err = b.exec.ExecuteStdout("docker", "push", opts.Tag)
		if err != nil {
			return err
		}
	}

	return nil
}
