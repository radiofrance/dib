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

func (img Image) CurrentTag() string {
	if img.NeedsRebuild {
		return "dev-" + img.Hash
	}

	return img.Hash
}

// DockerRef returns the fully-qualified docker ref for a given version.
func (img Image) DockerRef(version string) string {
	return fmt.Sprintf("%s:%s", img.Name, version)
}
