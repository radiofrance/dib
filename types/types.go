package types

import "time"

// ImageBuilder is the interface for building Docker images.
type ImageBuilder interface {
	Build(opts ImageBuilderOpts) error
}

// ImageBuilderOpts holds the options to be used to build the image.
type ImageBuilderOpts struct {
	// Path to the build context.
	Context string
	// Name of the tag to build, same as passed to the '-t' flag of the docker build command.
	Tag string
	// Date and time on which the image was built (string, date-time as defined by RFC 3339).
	CreationTime *time.Time
	// Contact details of the people or organization responsible for the image (freeform string)
	Authors *string
	// URL to get source code for building the image (string)
	Source *string
	// Source control revision identifier for the packaged software.
	Revision *string
}

// TestRunner is an interface for dealing with docker tests, such as dgoss, trivy.
type TestRunner interface {
	RunTest(ref, path string) error
}

// DockerRegistry is an interface for dealing with docker registries.
type DockerRegistry interface {
	RefExists(imageRef string) (bool, error)
	Retag(existingRef, toCreateRef string) error
}
