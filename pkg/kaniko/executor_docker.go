package kaniko

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/radiofrance/dib/pkg/executor"

	"github.com/radiofrance/dib/internal/logger"
)

// ContainerConfig holds the configuration options for the docker container.
type ContainerConfig struct {
	Image   string            // Image used to create the Kaniko container.
	Env     map[string]string // A map of key/value environment variables to inject in the container.
	Volumes map[string]string // A map of volumes to mount in the container.
}

// DockerExecutor will run Kaniko in a docker container.
type DockerExecutor struct {
	exec         executor.ShellExecutor
	config       ContainerConfig
	DockerConfig string // Path to the docker config directory to mount in the Kaniko container.
}

// NewDockerExecutor creates a new instance of DockerExecutor.
func NewDockerExecutor(exec executor.ShellExecutor, config ContainerConfig) *DockerExecutor {
	dockerCfg := os.Getenv("DOCKER_CONFIG")
	if dockerCfg == "" {
		dockerCfg = fmt.Sprintf("%s/.docker", os.Getenv("HOME"))
	}

	return &DockerExecutor{
		exec:         exec,
		config:       config,
		DockerConfig: dockerCfg,
	}
}

// Execute the Kaniko build using a Docker container.
func (e DockerExecutor) Execute(_ context.Context, output io.Writer, args []string) error {
	logger.Infof("Building image with kaniko local executor")

	dockerArgs := []string{
		"run",
		"--rm",
		"--tty",
		fmt.Sprintf("--volume=%s:/kaniko/.docker", e.DockerConfig),
		"--env=DOCKER_CONFIG=/kaniko/.docker",
	}

	for k, v := range e.config.Env {
		dockerArgs = append(dockerArgs, fmt.Sprintf("--env=%s=%s", k, v))
	}

	for k, v := range e.config.Volumes {
		dockerArgs = append(dockerArgs, fmt.Sprintf("--volume=%s:%s", k, v))
	}

	dockerArgs = append(dockerArgs, e.config.Image)
	dockerArgs = append(dockerArgs, args...)

	return e.exec.ExecuteWithWriter(output, "docker", dockerArgs...)
}
