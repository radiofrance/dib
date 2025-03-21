package buildkit

import (
	"path/filepath"

	"github.com/distribution/reference"
	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/strutil"
	"github.com/radiofrance/dib/pkg/types"
)

type Builder struct {
	executor       exec.Executor
	buildctlBinary string
}

// NewBuilder creates a new instance of Builder.
func NewBKBuilder(exec exec.Executor, binary string) (*Builder, error) {
	return &Builder{
		executor:       exec,
		buildctlBinary: binary,
	}, nil
}

// Build the image using the Buildkit backend.
func (b Builder) Build(opts types.ImageBuilderOpts) error {
	buildctlArgs, err := generateBuildctlArgs(opts)
	if err != nil {
		return err
	}

	logger.Debugf("running %s %v", b.buildctlBinary, buildctlArgs)

	// Currently, we only support writing to stdout. In the future, this can be improved to support custom stdio.
	if err := b.executor.ExecuteStdout(b.buildctlBinary, buildctlArgs...); err != nil {
		return err
	}

	return nil
}

func generateBuildctlArgs(opts types.ImageBuilderOpts) ([]string, error) {
	output := "type=image"

	if tags := strutil.DedupeStrSlice(opts.Tags); len(tags) > 0 {
		for _, tag := range tags {
			// Normalize the tag by transforming it from a familiar name used in Docker UI to a fully qualified reference.
			parsedReference, err := reference.ParseNormalizedNamed(tag)
			if err != nil {
				return nil, err
			}
			output += ",name=" + parsedReference.String()
		}
	} else {
		output += ",dangling-name-prefix=<none>"
	}

	if opts.Push {
		output += ",push=true"
	}

	buildctlArgs := buildctlBaseArgs(opts.BuildkitHost)

	buildctlArgs = append(buildctlArgs, []string{
		"build",
		"--progress=" + opts.Progress,
		"--frontend=dockerfile.v0",
		"--local=context=" + opts.Context,
		"--output=" + output,
	}...)

	// Set the directory and filename for the Dockerfile,
	// as the Dockerfile path may differ from the build context path.
	dir := opts.Context
	file := defaultDockerfileName
	if opts.File != "" {
		dir, file = filepath.Split(opts.File)

		if dir == "" {
			dir = "."
		}
	}

	var err error
	dir, file, err = buildKitFile(dir, file)
	if err != nil {
		return nil, err
	}

	buildctlArgs = append(buildctlArgs, "--local=dockerfile="+dir)
	buildctlArgs = append(buildctlArgs, "--opt=filename="+file)

	// The target option specifies the build stage to build.
	if opts.Target != "" {
		buildctlArgs = append(buildctlArgs, "--opt=target="+opts.Target)
	}

	for key, val := range opts.BuildArgs {
		buildctlArgs = append(buildctlArgs, "--opt=build-arg:"+key+"="+val)
	}

	for _, l := range opts.Labels {
		buildctlArgs = append(buildctlArgs, "--opt=label="+l)
	}

	return buildctlArgs, nil
}
