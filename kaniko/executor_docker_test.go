package kaniko_test

import (
	"context"
	"testing"

	"github.com/radiofrance/dib/kaniko"
	"github.com/radiofrance/dib/mock"
	"github.com/stretchr/testify/assert"
)

//nolint:paralleltest
func Test_DockerExecutor_Execute(t *testing.T) {
	t.Setenv("HOME", "/home/dib")

	shell := &mock.Executor{
		Output: "some output",
		Error:  nil,
	}

	executor := kaniko.NewDockerExecutor(shell, kaniko.ContainerConfig{
		Image: "gcr.io/kaniko-project/executor:latest",
		Env: map[string]string{
			"SOME_VARIABLE": "some_value",
		},
		Volumes: map[string]string{
			"/host/path/to/context": "/container/path/to/context",
		},
	})

	writer := mock.NewWriter()
	err := executor.Execute(context.Background(), writer, []string{"kaniko-arg1", "kaniko-arg2"})
	assert.Equal(t, writer.GetString(), "some output")

	assert.NoError(t, err)

	assert.Equal(t, shell.Command, "docker")
	expectedArgs := []string{
		"run",
		"--rm",
		"--tty",
		"--volume=/home/dib/.docker:/kaniko/.docker",
		"--env=DOCKER_CONFIG=/kaniko/.docker",
		"--env=SOME_VARIABLE=some_value",
		"--volume=/host/path/to/context:/container/path/to/context",
		"gcr.io/kaniko-project/executor:latest",
		"kaniko-arg1",
		"kaniko-arg2",
	}

	assert.ElementsMatch(t, shell.Args, expectedArgs)
}
