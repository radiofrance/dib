package kaniko

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/radiofrance/dib/pkg/types"
	"github.com/sirupsen/logrus"
)

// ContextProvider provides a layer of abstraction for different build context sources.
type ContextProvider interface {
	// PrepareContext allows to do some operations on the build context before the executor runs,
	// like moving it to a remote location in order to be accessible by remote executors.
	// It must returns an URL compatible with Kaniko's `--context` flag.
	PrepareContext(opts types.ImageBuilderOpts) (string, error)
}

// Executor executes the Kaniko build.
type Executor interface {
	// Execute the kaniko build, passing a slice of arguments to the kaniko command.
	Execute(ctx context.Context, output io.Writer, args []string) error
}

// Builder uses Kaniko as build backend.
type Builder struct {
	executor        Executor
	contextProvider ContextProvider
	DryRun          bool // When dry-run mode is enabled, the executor won't be called for real.
}

// NewBuilder creates a new instance of Builder.
func NewBuilder(exec Executor, contextProvider ContextProvider) *Builder {
	return &Builder{
		executor:        exec,
		contextProvider: contextProvider,
	}
}

// Build the image using the Kaniko backend.
func (b Builder) Build(opts types.ImageBuilderOpts) error {
	contextPath, err := b.contextProvider.PrepareContext(opts)
	if err != nil {
		return fmt.Errorf("cannot prepare kaniko build context: %w", err)
	}

	// More infos, on Kaniko args here: https://github.com/GoogleContainerTools/kaniko#additional-flags
	kanikoArgs := []string{
		"--context=" + contextPath,
		"--destination=" + opts.Tag,
		"--log-format=text",
		"--snapshotMode=redo",
		"--single-snapshot",
	}

	for k, v := range opts.BuildArgs {
		kanikoArgs = append(kanikoArgs, fmt.Sprintf("--build-arg=%s=%s", k, v))
	}

	if !opts.Push {
		kanikoArgs = append(kanikoArgs, "--no-push")
	}

	if b.DryRun {
		logrus.Infof("[DRY-RUN] kaniko %s", strings.Join(kanikoArgs, " "))
		return nil
	}

	return b.executor.Execute(context.Background(), opts.LogOutput, kanikoArgs)
}
