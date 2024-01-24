package types

import (
	"io"
)

const (
	// BackendDocker use "Docker" for building Docker images.
	BackendDocker = "docker"
	// BackendKaniko use "Kaniko" for building Docker images.
	BackendKaniko = "kaniko"

	// TestRunnerGoss use Goss for testing Docker images.
	TestRunnerGoss = "goss"
	// TestRunnerTrivy use Trivy for testing Docker images.
	TestRunnerTrivy = "trivy"
)

// ImageBuilder is the interface for building Docker images.
type ImageBuilder interface {
	Build(opts ImageBuilderOpts) error
}

// ImageBuilderOpts holds the options to be used to build the image.
type ImageBuilderOpts struct {
	// Path to the build context.
	Context string
	// Name of the tags to build, same as passed to the '-t' flag of the docker build command.
	Tags []string
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

// TestRunner is an interface for dealing with docker tests, such as goss, trivy.
type TestRunner interface {
	Name() string
	IsConfigured(opts RunTestOptions) bool

	// RunTest function should execute tests (trivy scan, goss test, etc...).
	// It returns nil if test was successfully executed, an error if any problem occurs
	RunTest(opts RunTestOptions) error
}

type RunTestOptions struct {
	ImageName         string
	ImageReference    string
	DockerContextPath string
	ReportJunitDir    string
	ReportTrivyDir    string
}

// DockerRegistry is an interface for dealing with docker registries.
type DockerRegistry interface {
	RefExists(imageRef string) (bool, error)
}
