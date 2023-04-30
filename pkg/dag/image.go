package dag

import (
	"fmt"

	"github.com/radiofrance/dib/pkg/dockerfile"
	"gopkg.in/yaml.v3"
)

// Image holds the docker image information.
type Image struct {
	Name      string `yaml:"name"`
	ShortName string `yaml:"short_name"`
	// Hash of the build context "At the moment"
	Hash string `yaml:"hash"`
	// A list of tags to make in addition to image hash.
	ExtraTags      []string               `yaml:"extra_tags,flow,omitempty"`
	Dockerfile     *dockerfile.Dockerfile `yaml:"dockerfile"`
	IgnorePatterns []string               `yaml:"ignore_patterns,flow,omitempty"`
	NeedsRebuild   bool                   `yaml:"-"`
	NeedsTests     bool                   `yaml:"-"`
	RetagDone      bool                   `yaml:"-"`
	RebuildDone    bool                   `yaml:"-"`
	RebuildFailed  bool                   `yaml:"-"`
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

func (img Image) Print() string {
	strImg, err := yaml.Marshal(img)
	if err != nil {
		return err.Error()
	}
	return string(strImg)
}
