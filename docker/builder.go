package docker

import (
	"fmt"
	"strings"

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
	dockerArgs := []string{
		"build",
		"--no-cache",
	}

	for k, v := range opts.Labels {
		dockerArgs = append(dockerArgs, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	dockerArgs = append(dockerArgs, "-t", opts.Tag, opts.Context)

	if b.dryRun {
		logrus.Infof("[DRY-RUN] docker %s", strings.Join(dockerArgs, " "))

		if opts.Push {
			logrus.Infof("[DRY-RUN] docker push %s", opts.Tag)
		}
		return nil
	}

	err := b.exec.ExecuteWithWriter(opts.LogOutput, "docker", dockerArgs...)
	if err != nil {
		return err
	}

	if opts.Push {
		err = b.exec.ExecuteWithWriter(opts.LogOutput, "docker", "push", opts.Tag)
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
