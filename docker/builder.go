package docker

import (
	"fmt"
	"time"

	"github.com/radiofrance/dib/exec"
	"github.com/radiofrance/dib/types"
	"github.com/sirupsen/logrus"
)

// ImageBuilderTagger builds an image using the docker command-line executable.
type ImageBuilderTagger struct {
	exec   exec.Executor
	dryRun bool
}

// NewImageBuilderTagger creates a new instance of an ImageBuilder.
func NewImageBuilderTagger(executor exec.Executor, dryRun bool) *ImageBuilderTagger {
	return &ImageBuilderTagger{executor, dryRun}
}

// Build the image using the docker executable.
// If the image is built successfully, the image will be pushed to the registry.
func (b ImageBuilderTagger) Build(opts types.ImageBuilderOpts) error {
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

// Tag runs a `docker tag`command to retag the source tag with the destination tag.
func (b ImageBuilderTagger) Tag(src, dest string) error {
	if b.dryRun {
		logrus.Infof("[DRY-RUN] docker pull %s", src)
		logrus.Infof("[DRY-RUN] docker tag %s %s", src, dest)
		return nil
	}
	logrus.Debugf("Running `docker pull %s`", src)
	if err := b.exec.ExecuteStdout("docker", "pull", src); err != nil {
		return err
	}
	logrus.Debugf("Running `docker tag %s %s`", src, dest)
	return b.exec.ExecuteStdout("docker", "tag", src, dest)
}
