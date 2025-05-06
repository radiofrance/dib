package buildkit

import (
	"github.com/radiofrance/dib/pkg/types"
)

// LocalContextProvider provides a local build context.
type LocalContextProvider struct{}

// NewLocalContextProvider creates a new instance of LocalContextProvider.
func NewLocalContextProvider() *LocalContextProvider {
	return &LocalContextProvider{}
}

// PrepareContext returns the local build context path without performing any additional operations.
// Since the context is already available locally, it simply returns the context path from the provided options.
func (c LocalContextProvider) PrepareContext(opts types.ImageBuilderOpts) (string, error) {
	return opts.Context, nil
}
