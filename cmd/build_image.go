package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/distribution/reference"
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/radiofrance/dib/pkg/exec"
	"github.com/radiofrance/dib/pkg/logger"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const defaultDockerfile = "Dockerfile"

func buildImageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build-image [CONTEXT]",
		Short: "Build a single image with the given tags",
		Long: `dib build-image builds the image of a single build context (the current directory by default),
and tags it with the tags given with --tag.

Unlike "dib build", it doesn't compute a graph of images, and doesn't tag images with a hash of their content:
the image is always built, with exactly the given tags. Use it to build the image of an application in its own
CI pipeline, for instance tagged with the commit SHA.

It uses the BuildKit backend: with the Kubernetes executor configured in the "buildkit" section of the
configuration file, or locally with --local-only.`,
		Example: `  dib build-image --tag registry.example.org/app:1.0.0
  dib build-image --file docker/Dockerfile --tag registry.example.org/app:1.0.0 --build-arg VERSION=1.0.0 .`,
		Args:         cobra.MaximumNArgs(1),
		RunE:         buildImageAction,
		SilenceUsage: true,
	}
	cmd.Flags().StringArrayP("tag", "t", []string{},
		"Image name and tag, e.g. registry.example.org/app:1.0.0. Can be repeated.")
	cmd.Flags().StringP("file", "f", "",
		`Path to the Dockerfile, relative to the build context (default "Dockerfile")`)
	cmd.Flags().String("target", "",
		"Set the target build stage to build")
	cmd.Flags().StringArray("build-arg", []string{},
		"`argument=value` to supply to the builder")
	cmd.Flags().StringArray("secret", []string{},
		"Secret to expose to the build, e.g. id=npmrc,env=NPMRC or id=npmrc,src=/path/to/.npmrc. Can be repeated.")
	cmd.Flags().StringArray("ssh", []string{},
		"SSH agent socket or keys to expose to the build, e.g. default=/path/to/key. Can be repeated.")
	cmd.Flags().String("progress", "auto",
		"Set type of progress output (auto, plain, tty). Use plain to show container output")
	cmd.Flags().String("compression", "",
		"Set the compression type (uncompressed, gzip, estargz, zstd)")
	cmd.Flags().Bool("local-only", false,
		"Build the image locally. If this flag is not set, the build will be performed in Kubernetes.")
	cmd.Flags().Bool("push", false,
		"Push the image to the registry after building it. Images built in Kubernetes are always pushed.")
	cmd.Flags().String("buildkit-host", "",
		"buildkit host address.")

	cobra.CheckErr(cmd.MarkFlagRequired("tag"))

	return cmd
}

func buildImageAction(cmd *cobra.Command, args []string) error {
	// Bind command flags to viper configuration using snake_case
	bindPFlagsSnakeCase(cmd.Flags())

	opts := dib.BuildOpts{}
	hydrateOptsFromViper(&opts)

	buildContext := "."
	if len(args) > 0 {
		buildContext = args[0]
	}

	imageOpts, err := newImageBuilderOpts(opts, buildContext, viper.GetStringSlice("tag"))
	if err != nil {
		return err
	}

	opts.BuildkitHost, err = resolveBuildkitHost(cmd, opts.LocalOnly)
	if err != nil {
		return err
	}

	imageOpts.BuildkitHost = opts.BuildkitHost

	err = doBuildImage(cmd.Context(), opts, imageOpts)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	logger.Infof("Build process completed")

	return nil
}

func doBuildImage(ctx context.Context, opts dib.BuildOpts, imageOpts types.ImageBuilderOpts) error {
	shell := exec.NewShellExecutor(workingDir, os.Environ())

	builder, err := newBuildkitBuilder(ctx, opts, shell)
	if err != nil {
		return err
	}

	logger.Infof("Building %v in context %q", imageOpts.Tags, imageOpts.Context)

	return builder.Build(ctx, imageOpts)
}

// newImageBuilderOpts checks the build context, the Dockerfile and the tags of the image to build,
// and returns the options to build it.
func newImageBuilderOpts(opts dib.BuildOpts, buildContext string, tags []string,
) (types.ImageBuilderOpts, error) {
	if len(tags) == 0 {
		return types.ImageBuilderOpts{}, errors.New("at least one tag is required")
	}

	var shortName string

	imageTags := make([]string, 0, len(tags))

	for _, tag := range tags {
		named, err := reference.ParseNormalizedNamed(tag)
		if err != nil {
			return types.ImageBuilderOpts{}, fmt.Errorf("invalid tag %q: %w", tag, err)
		}

		if shortName == "" {
			shortName = path.Base(reference.Path(named))
		}

		// Make the "latest" tag explicit when no tag is given, e.g. "registry.example.org/app".
		imageTags = append(imageTags, reference.FamiliarString(reference.TagNameOnly(named)))
	}

	contextPath := buildContext
	if !filepath.IsAbs(contextPath) {
		contextPath = filepath.Join(workingDir, contextPath)
	}

	info, err := os.Stat(contextPath)
	if err != nil {
		return types.ImageBuilderOpts{}, fmt.Errorf("invalid build context: %w", err)
	}

	if !info.IsDir() {
		return types.ImageBuilderOpts{}, fmt.Errorf("invalid build context %q: not a directory", contextPath)
	}

	// The Dockerfile must be inside the build context: with the Kubernetes executor,
	// BuildKit reads it from the build context archive.
	dockerfileRelPath := defaultDockerfile
	if opts.File != "" {
		dockerfileRelPath = filepath.Clean(opts.File)
	}

	if filepath.IsAbs(dockerfileRelPath) {
		// An absolute path outside the build context is left unchanged, and rejected below.
		dockerfileRelPath = strings.TrimPrefix(dockerfileRelPath, contextPath+string(filepath.Separator))
	}

	if !filepath.IsLocal(dockerfileRelPath) {
		return types.ImageBuilderOpts{}, fmt.Errorf("the Dockerfile %q must be inside the build context %q",
			opts.File, contextPath)
	}

	dockerfilePath := filepath.Join(contextPath, dockerfileRelPath)

	dckFile, err := dockerfile.ParseDockerfile(dockerfilePath)
	if err != nil {
		return types.ImageBuilderOpts{}, fmt.Errorf("cannot read Dockerfile: %w", err)
	}

	// Same OCI labels as "dib build": revision, source, creation date, etc.
	labels := dib.LoadCommonMetadata(&exec.ShellExecutor{}).
		WithImage(&dag.Image{ShortName: shortName, Dockerfile: dckFile}).
		ToLabels()

	imageOpts := types.ImageBuilderOpts{
		Context:     contextPath,
		File:        dockerfileRelPath,
		LocalOnly:   opts.LocalOnly,
		Target:      opts.Target,
		Tags:        imageTags,
		Labels:      labels,
		BuildArgs:   parseBuildArgs(opts.BuildArg),
		Push:        opts.Push,
		LogOutput:   os.Stdout,
		Progress:    opts.Progress,
		Compression: opts.Compression,
		Secrets:     opts.Secret,
		SSH:         opts.SSH,
	}

	if opts.LocalOnly {
		// buildctl reads the Dockerfile from the local filesystem.
		imageOpts.File = dockerfilePath
	}

	return imageOpts, nil
}
