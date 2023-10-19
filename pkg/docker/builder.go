package docker

import (
	"fmt"
	"strings"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/types"
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

	for k, v := range opts.BuildArgs {
		dockerArgs = append(dockerArgs, fmt.Sprintf("--build-arg=%s=%s", k, v))
	}

	for k, v := range opts.Labels {
		dockerArgs = append(dockerArgs, fmt.Sprintf("--label=%s=%s", k, v))
	}

	for _, tag := range opts.Tags {
		dockerArgs = append(dockerArgs, "--tag="+tag)
	}

	dockerArgs = append(dockerArgs, opts.Context)

	if b.dryRun {
		logger.Infof("[DRY-RUN] docker %s", strings.Join(dockerArgs, " "))

		if opts.Push {
			for _, tag := range opts.Tags {
				logger.Infof("[DRY-RUN] docker push %s", tag)
			}
		}
		return nil
	}

	err := b.exec.ExecuteWithWriter(opts.LogOutput, "docker", dockerArgs...)
	if err != nil {
		return err
	}

	if opts.Push && len(opts.Tags) > 0 {
		for _, tag := range opts.Tags {
			err = b.exec.ExecuteWithWriter(opts.LogOutput, "docker", "push", tag)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Tag runs a `docker tag`command to retag the source tag with the destination tag.
func (b ImageBuilderTagger) Tag(src, dest string) error {
	if b.dryRun {
		logger.Infof("[DRY-RUN] docker tag %s %s", src, dest)
		return nil
	}
	logger.Debugf("Running `docker tag %s %s`", src, dest)
	return b.exec.ExecuteStdout("docker", "tag", src, dest)
}
