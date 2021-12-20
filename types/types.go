package types

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
