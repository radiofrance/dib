package types

import (
	"io"
)

const (
	// BackendDocker use "Docker" for building Docker images.
	BackendDocker = "docker"
	// BackendKaniko use "Kaniko" for building Docker images.
	BackendKaniko = "kaniko"
	// BuildKitBackend use buildkit for building oci images.
	BuildKitBackend = "buildkit"
	// TestRunnerGoss use Goss for testing Docker images.
	TestRunnerGoss = "goss"
	// TestRunnerTrivy use Trivy for testing Docker images.
	TestRunnerTrivy = "trivy"
)

// ImageBuilder is the interface for building oci images.
type ImageBuilder interface {
	Build(opts ImageBuilderOpts) error
}

// ImageBuilderOpts is a set of options to perform oci image build.
type ImageBuilderOpts struct {
	// BuildkitHost is the address of Buildkit host
	BuildkitHost string
	// Path to the build context.
	Context string
	// File is the name of the Dockerfile
	File string
	//  LocalOnly is true if the build context is local and not remote.
	LocalOnly bool
	// Target is the target of the build
	Target string
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
	// Progress Set type of progress output (auto, plain, tty). Use plain to show container output
	Progress string
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
	BuildkitHost      string
	ReportJunitDir    string
	ReportTrivyDir    string
}

// DockerRegistry is an interface for dealing with docker registries.
type DockerRegistry interface {
	RefExists(imageRef string) (bool, error)
}
