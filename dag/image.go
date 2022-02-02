package dag

import (
	"fmt"

	"github.com/radiofrance/dib/dockerfile"
)

// Image holds the docker image information.
type Image struct {
	Name            string
	ShortName       string
	Dockerfile      *dockerfile.Dockerfile
	NeedsRebuild    bool
	NeedsTests      bool
	NeedsRetag      bool
	RetagDone       bool
	RetagLatestDone bool
	RebuildDone     bool
}

// DockerRef returns the fully-qualified docker ref for a given version.
func (img Image) DockerRef(version string) string {
	return fmt.Sprintf("%s:%s", img.Name, version)
}
