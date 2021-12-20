package docker

import (
	"github.com/sirupsen/logrus"
	registry "gitlab.com/radiofrance/go-container-registry"
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

// Retag creates a new tag from an existing one.
func (r Registry) Retag(existingRef, toCreateRef string) error {
	if r.dryRun {
		logrus.Infof("[DRY-RUN] Retagging image from \"%s\" to \"%s\"", existingRef, toCreateRef)
		return nil
	}
	return r.gcr.Retag(existingRef, toCreateRef)
}
