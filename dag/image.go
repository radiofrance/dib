package dag

import (
	"fmt"

	"github.com/radiofrance/dib/dockerfile"
)

// Image holds the docker image information.
type Image struct {
	Name           string
	ShortName      string
	ExtraTags      []string // A list of tags to make in addition to image hash.
	Hash           string   // Hash of the build context "At the moment"
	Dockerfile     *dockerfile.Dockerfile
	IgnorePatterns []string
	NeedsRebuild   bool
	NeedsTests     bool
	RetagDone      bool
	RebuildDone    bool
	RebuildFailed  bool
}

// CurrentRef returns the fully-qualified docker ref for the current version.
// If the image needs to be rebuilt, a temporary `dev-` prefix is added to the tag.
func (img Image) CurrentRef() string {
	tag := img.Hash

	if img.NeedsRebuild {
		tag = "dev-" + img.Hash
	}

	return img.DockerRef(tag)
}

// DockerRef returns the fully-qualified docker ref for a given version.
func (img Image) DockerRef(version string) string {
	return fmt.Sprintf("%s:%s", img.Name, version)
}
