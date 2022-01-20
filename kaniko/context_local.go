package kaniko

import (
	"fmt"

	"github.com/radiofrance/dib/types"
)

// LocalContextProvider provides a local build context.
type LocalContextProvider struct{}

// NewLocalContextProvider creates a new instance of LocalContextProvider.
func NewLocalContextProvider() *LocalContextProvider {
	return &LocalContextProvider{}
}

// PrepareContext has nothing to do because the build context already exists locally.
// It just returns the path of the existing local context, prefixed by the Kaniko `dir://` indicator.
func (c LocalContextProvider) PrepareContext(opts types.ImageBuilderOpts) (string, error) {
	return fmt.Sprintf("dir://%s", opts.Context), nil
}
