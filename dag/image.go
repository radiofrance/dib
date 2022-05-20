package dag

import (
	"fmt"

	"github.com/radiofrance/dib/dockerfile"
)

// Image holds the docker image information.
type Image struct {
	Name           string
	ShortName      string
	CurrentTag     string   // Current tag expected to be present on the registry before the build.
	TargetTag      string   // New tag, not present in registry until the image is built and pushed.
	ExtraTags      []string // A list of tags to make in addition to TargetTag.
	Hash           string   // Hash of the build context "At the moment"
	Dockerfile     *dockerfile.Dockerfile
	IgnorePatterns []string
	NeedsRebuild   bool
	NeedsTests     bool
	NeedsRetag     bool
	RetagDone      bool
	RebuildDone    bool
	RebuildFailed  bool
}

// DockerRef returns the fully-qualified docker ref for a given version.
func (img Image) DockerRef(version string) string {
	return fmt.Sprintf("%s:%s", img.Name, version)
}
