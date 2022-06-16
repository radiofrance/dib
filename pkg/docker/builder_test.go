package docker_test

import (
	"errors"
	"io"
	"testing"

	"github.com/radiofrance/dib/pkg/docker"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
)

type fakeExecutor struct {
	ExecutedCommands []struct {
		Command string
		Args    []string
	}
	Error error
}

func (e *fakeExecutor) Execute(name string, args ...string) (string, error) {
	e.ExecutedCommands = append(e.ExecutedCommands, struct {
		Command string
		Args    []string
	}{
		Command: name,
		Args:    args,
	})

	return "", e.Error
}

func (e *fakeExecutor) ExecuteStdout(name string, args ...string) error {
	_, err := e.Execute(name, args...)
	return err
}

func (e *fakeExecutor) ExecuteWithWriter(_ io.Writer, name string, args ...string) error {
	_, err := e.Execute(name, args...)
	return err
}

func (e *fakeExecutor) ExecuteWithWriters(_, _ io.Writer, name string, args ...string) error {
	_, err := e.Execute(name, args...)
	return err
}

func provideDefaultOptions() types.ImageBuilderOpts {
	return types.ImageBuilderOpts{
		Context: "/tmp/docker-context",
		Tags: []string{
			"gcr.io/project-id/image:version",
			"gcr.io/project-id/image:latest",
		},
		BuildArgs: map[string]string{
			"someArg": "someValue",
		},
		Labels: map[string]string{
			"someLabel": "someValue",
		},
		Push: true,
	}
}

func Test_Build_DryRun(t *testing.T) {
	t.Parallel()

	fakeExecutor := &fakeExecutor{}
	builder := docker.NewImageBuilderTagger(fakeExecutor, true)

	opts := provideDefaultOptions()

	err := builder.Build(opts)
	assert.NoError(t, err)

	assert.Empty(t, fakeExecutor.ExecutedCommands)
}

func Test_Build_Executes(t *testing.T) {
	t.Parallel()

	fakeExecutor := &fakeExecutor{}
	builder := docker.NewImageBuilderTagger(fakeExecutor, false)

	opts := provideDefaultOptions()

	err := builder.Build(opts)

	assert.NoError(t, err)
	assert.Len(t, fakeExecutor.ExecutedCommands, 3)

	expectedBuildArgs := []string{
		"build",
		"--no-cache",
		"--tag=gcr.io/project-id/image:version",
		"--tag=gcr.io/project-id/image:latest",
		"--build-arg=someArg=someValue",
		"--label=someLabel=someValue",
		"/tmp/docker-context",
	}

	expectedPushArgs := []string{
		"push",
		"gcr.io/project-id/image:version",
	}

	expectedPushLatestArgs := []string{
		"push",
		"gcr.io/project-id/image:latest",
	}

	assert.Equal(t, "docker", fakeExecutor.ExecutedCommands[0].Command)
	assert.ElementsMatch(t, expectedBuildArgs, fakeExecutor.ExecutedCommands[0].Args)

	assert.Equal(t, "docker", fakeExecutor.ExecutedCommands[1].Command)
	assert.ElementsMatch(t, expectedPushArgs, fakeExecutor.ExecutedCommands[1].Args)
	assert.Equal(t, "docker", fakeExecutor.ExecutedCommands[2].Command)
	assert.ElementsMatch(t, expectedPushLatestArgs, fakeExecutor.ExecutedCommands[2].Args)
}

func Test_Build_ExecutesDisablesPush(t *testing.T) {
	t.Parallel()

	fakeExecutor := &fakeExecutor{}
	builder := docker.NewImageBuilderTagger(fakeExecutor, false)

	opts := provideDefaultOptions()
	opts.Push = false

	err := builder.Build(opts)

	assert.NoError(t, err)
	assert.Len(t, fakeExecutor.ExecutedCommands, 1)

	expectedBuildArgs := []string{
		"build",
		"--no-cache",
		"--tag=gcr.io/project-id/image:version",
		"--tag=gcr.io/project-id/image:latest",
		"--build-arg=someArg=someValue",
		"--label=someLabel=someValue",
		"/tmp/docker-context",
	}

	assert.Equal(t, "docker", fakeExecutor.ExecutedCommands[0].Command)
	assert.ElementsMatch(t, expectedBuildArgs, fakeExecutor.ExecutedCommands[0].Args)
}

func Test_Build_FailsOnExecutorError(t *testing.T) {
	t.Parallel()

	fakeExecutor := &fakeExecutor{}
	fakeExecutor.Error = errors.New("something wrong happened")
	builder := docker.NewImageBuilderTagger(fakeExecutor, false)

	err := builder.Build(provideDefaultOptions())

	assert.EqualError(t, err, "something wrong happened")
}

func Test_Tag(t *testing.T) {
	t.Parallel()

	fakeExecutor := &fakeExecutor{}
	tagger := docker.NewImageBuilderTagger(fakeExecutor, false)

	err := tagger.Tag("registry/image:src-tag", "registry/image:dest-tag")

	assert.NoError(t, err)
	assert.Len(t, fakeExecutor.ExecutedCommands, 1)

	expectedTagArgs := []string{
		"tag",
		"registry/image:src-tag",
		"registry/image:dest-tag",
	}

	assert.Equal(t, "docker", fakeExecutor.ExecutedCommands[0].Command)
	assert.Equal(t, expectedTagArgs, fakeExecutor.ExecutedCommands[0].Args)
}

func Test_Tag_DryRun(t *testing.T) {
	t.Parallel()

	fakeExecutor := &fakeExecutor{}
	tagger := docker.NewImageBuilderTagger(fakeExecutor, true)

	err := tagger.Tag("registry/image:src-tag", "registry/image:dest-tag")

	assert.NoError(t, err)
	assert.Empty(t, fakeExecutor.ExecutedCommands)
}
