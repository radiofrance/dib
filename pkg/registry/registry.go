package registry

import (
	"github.com/radiofrance/dib/internal/logger"
	registry "github.com/radiofrance/go-containerregistry"
)

// Registry wraps the Google Cloud Registry client library.
type Registry struct {
	gcr    *registry.Registry
	dryRun bool
}

// NewRegistry creates a new instance of Registry.
func NewRegistry(url string, dryRun bool) (*Registry, error) {
	gcr, err := registry.New(url)
	if err != nil {
		return nil, err
	}

	return &Registry{gcr, dryRun}, nil
}

// RefExists checks if the GCR contains the image ref.
func (r Registry) RefExists(imageRef string) (bool, error) {
	return r.gcr.RefExists(imageRef)
}

// Tag creates a new tag from an existing one.
func (r Registry) Tag(existingRef, toCreateRef string) error {
	if r.dryRun {
		logger.Infof("[DRY-RUN] Retagging image from \"%s\" to \"%s\"", existingRef, toCreateRef)
		return nil
	}
	logger.Debugf("Retaging image on gcr, source %s, dest %s`", existingRef, toCreateRef)
	return r.gcr.Retag(existingRef, toCreateRef)
}
