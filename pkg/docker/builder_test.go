package docker_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/radiofrance/dib/pkg/docker"
	"github.com/radiofrance/dib/pkg/mock"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		Push:      true,
		LogOutput: &bytes.Buffer{},
	}
}

func Test_Build_DryRun(t *testing.T) {
	t.Parallel()

	fakeExecutor := mock.NewExecutor(nil)
	builder := docker.NewImageBuilderTagger(fakeExecutor, true)

	opts := provideDefaultOptions()

	err := builder.Build(opts)
	require.NoError(t, err)
	assert.Empty(t, fakeExecutor.Executed)
}

func Test_Build_Executes(t *testing.T) {
	t.Parallel()

	fakeExecutor := mock.NewExecutor(nil)
	builder := docker.NewImageBuilderTagger(fakeExecutor, false)

	opts := provideDefaultOptions()

	err := builder.Build(opts)

	require.NoError(t, err)
	assert.Len(t, fakeExecutor.Executed, 3)

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

	assert.Equal(t, "docker", fakeExecutor.Executed[0].Command)
	assert.ElementsMatch(t, expectedBuildArgs, fakeExecutor.Executed[0].Args)

	assert.Equal(t, "docker", fakeExecutor.Executed[1].Command)
	assert.ElementsMatch(t, expectedPushArgs, fakeExecutor.Executed[1].Args)
	assert.Equal(t, "docker", fakeExecutor.Executed[2].Command)
	assert.ElementsMatch(t, expectedPushLatestArgs, fakeExecutor.Executed[2].Args)
}

func Test_Build_ExecutesDisablesPush(t *testing.T) {
	t.Parallel()

	fakeExecutor := mock.NewExecutor(nil)
	builder := docker.NewImageBuilderTagger(fakeExecutor, false)

	opts := provideDefaultOptions()
	opts.Push = false

	err := builder.Build(opts)

	require.NoError(t, err)
	assert.Len(t, fakeExecutor.Executed, 1)

	expectedBuildArgs := []string{
		"build",
		"--no-cache",
		"--tag=gcr.io/project-id/image:version",
		"--tag=gcr.io/project-id/image:latest",
		"--build-arg=someArg=someValue",
		"--label=someLabel=someValue",
		"/tmp/docker-context",
	}

	assert.Equal(t, "docker", fakeExecutor.Executed[0].Command)
	assert.ElementsMatch(t, expectedBuildArgs, fakeExecutor.Executed[0].Args)
}

func Test_Build_FailsOnExecutorError(t *testing.T) {
	t.Parallel()

	fakeExecutor := mock.NewExecutor([]mock.ExecutorResult{
		{
			Output: "",
			Error:  errors.New("something wrong happened"),
		},
	})
	builder := docker.NewImageBuilderTagger(fakeExecutor, false)

	err := builder.Build(provideDefaultOptions())

	require.EqualError(t, err, "something wrong happened")
}

func Test_Tag(t *testing.T) {
	t.Parallel()

	fakeExecutor := mock.NewExecutor(nil)
	tagger := docker.NewImageBuilderTagger(fakeExecutor, false)

	err := tagger.Tag("registry/image:src-tag", "registry/image:dest-tag")

	require.NoError(t, err)
	assert.Len(t, fakeExecutor.Executed, 1)

	expectedTagArgs := []string{
		"tag",
		"registry/image:src-tag",
		"registry/image:dest-tag",
	}

	assert.Equal(t, "docker", fakeExecutor.Executed[0].Command)
	assert.Equal(t, expectedTagArgs, fakeExecutor.Executed[0].Args)
}

func Test_Tag_DryRun(t *testing.T) {
	t.Parallel()

	fakeExecutor := mock.NewExecutor(nil)
	tagger := docker.NewImageBuilderTagger(fakeExecutor, true)

	err := tagger.Tag("registry/image:src-tag", "registry/image:dest-tag")

	require.NoError(t, err)
	assert.Empty(t, fakeExecutor.Executed)
}
