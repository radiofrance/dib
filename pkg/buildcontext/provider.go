package buildcontext

import (
	"context"

	"github.com/radiofrance/dib/pkg/types"
)

// ContextProvider provides a layer of abstraction for different build context sources.
type ContextProvider interface {
	// PrepareContext allows doing some operations on the build context before the executor runs,
	// like moving it to a remote location to be accessible by remote executors.
	// It must return a URL compatible with Buildkit and Kaniko `--context` flags.
	PrepareContext(ctx context.Context, opts types.ImageBuilderOpts) (string, error)
}
