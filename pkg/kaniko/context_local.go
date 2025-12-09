package kaniko

import (
	"context"
	"fmt"

	"github.com/radiofrance/dib/pkg/types"
)

// LocalContextProvider provides a local build context.
type LocalContextProvider struct{}

// NewLocalContextProvider creates a new instance of LocalContextProvider.
func NewLocalContextProvider() *LocalContextProvider {
	return &LocalContextProvider{}
}

// PrepareContext has nothing to do because the build context already exists locally.
// It just returns the path of the existing local context, prefixed by the Kaniko `dir://` indicator.
func (c *LocalContextProvider) PrepareContext(_ context.Context, opts types.ImageBuilderOpts) (string, error) {
	return fmt.Sprintf("dir://%s", opts.Context), nil
}
