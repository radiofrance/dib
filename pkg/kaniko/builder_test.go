package kaniko_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/radiofrance/dib/pkg/kaniko"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
)

type fakeExecutor struct {
	Args     []string
	Executed bool
	Error    error
}

func (e *fakeExecutor) Execute(_ context.Context, _ io.Writer, args []string) error {
	e.Args = args
	e.Executed = true

	return e.Error
}

type fakeContextProvider struct {
	ContextURL string
	Error      error
}

func (p fakeContextProvider) PrepareContext(_ types.ImageBuilderOpts) (string, error) {
	return p.ContextURL, p.Error
}

func provideDefaultOptions() types.ImageBuilderOpts {
	return types.ImageBuilderOpts{
		Context: "/tmp/kaniko-context",
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
	builder := kaniko.NewBuilder(fakeExecutor, kaniko.NewLocalContextProvider())
	builder.DryRun = true

	opts := provideDefaultOptions()

	err := builder.Build(opts)
	assert.NoError(t, err)

	assert.False(t, fakeExecutor.Executed)
}

func Test_Build_Executes(t *testing.T) {
	t.Parallel()

	fakeExecutor := &fakeExecutor{}
	builder := kaniko.NewBuilder(fakeExecutor, kaniko.NewLocalContextProvider())

	opts := provideDefaultOptions()

	err := builder.Build(opts)

	assert.NoError(t, err)
	assert.True(t, fakeExecutor.Executed)

	expectedArgs := []string{
		"--context=dir:///tmp/kaniko-context",
		"--destination=gcr.io/project-id/image:version",
		"--destination=gcr.io/project-id/image:latest",
		"--log-format=text",
		"--snapshotMode=redo",
		"--single-snapshot",
		"--build-arg=someArg=someValue",
		"--label=someLabel=someValue",
	}

	assert.ElementsMatch(t, expectedArgs, fakeExecutor.Args)
}

func Test_Build_ExecutesDisablesPush(t *testing.T) {
	t.Parallel()

	fakeExecutor := &fakeExecutor{}
	builder := kaniko.NewBuilder(fakeExecutor, kaniko.NewLocalContextProvider())

	opts := provideDefaultOptions()
	opts.Push = false

	err := builder.Build(opts)

	assert.NoError(t, err)
	assert.True(t, fakeExecutor.Executed)

	expectedArgs := []string{
		"--context=dir:///tmp/kaniko-context",
		"--destination=gcr.io/project-id/image:version",
		"--destination=gcr.io/project-id/image:latest",
		"--log-format=text",
		"--snapshotMode=redo",
		"--single-snapshot",
		"--build-arg=someArg=someValue",
		"--label=someLabel=someValue",
		"--no-push",
	}

	assert.ElementsMatch(t, expectedArgs, fakeExecutor.Args)
}

func Test_Build_FailsOnContextError(t *testing.T) {
	t.Parallel()

	fakeExecutor := &fakeExecutor{}
	fakeContextProvider := &fakeContextProvider{
		Error: errors.New("something wrong happened"),
	}
	builder := kaniko.NewBuilder(fakeExecutor, fakeContextProvider)

	err := builder.Build(provideDefaultOptions())

	assert.EqualError(t, err, "cannot prepare kaniko build context: something wrong happened")
}

func Test_Build_FailsOnExecutorError(t *testing.T) {
	t.Parallel()

	fakeExecutor := &fakeExecutor{}
	fakeExecutor.Error = errors.New("something wrong happened")
	builder := kaniko.NewBuilder(fakeExecutor, kaniko.NewLocalContextProvider())

	err := builder.Build(provideDefaultOptions())

	assert.EqualError(t, err, "something wrong happened")
	assert.True(t, fakeExecutor.Executed)
}
