package types

import "io"

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
	// Labels a key/value set of labels to add to the image.
	Labels map[string]string
	// BuildArgs a key/value set of build args to pass to the build command.
	BuildArgs map[string]string
	// Push instructs to push to the remote registry after build, or skip it.
	Push bool
	// LogOutput is writer where build logs should be written
	LogOutput io.Writer
}

// ImageTagger is an abstraction for tagging docker images.
type ImageTagger interface {
	Tag(from, to string) error
}

// TestRunner is an interface for dealing with docker tests, such as dgoss, trivy.
type TestRunner interface {
	RunTest(opts RunTestOptions) error
}

type RunTestOptions struct {
	ImageName         string
	ImageReference    string
	DockerContextPath string
}

// DockerRegistry is an interface for dealing with docker registries.
type DockerRegistry interface {
	RefExists(imageRef string) (bool, error)
}
