package goss_test

import (
	"context"
	"io"
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/goss"
	"github.com/radiofrance/dib/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeExecutor struct {
	Error        error
	Output       string
	RecordedOpts types.RunTestOptions
	RecordedArgs []string
}

func (e *fakeExecutor) Execute(_ context.Context, output io.Writer, opts types.RunTestOptions, args ...string) error {
	e.RecordedOpts = opts
	e.RecordedArgs = args
	if _, err := output.Write([]byte(e.Output)); err != nil {
		return err
	}
	return e.Error
}

func Test_TestRunner_Supports(t *testing.T) {
	t.Parallel()

	dataset := []struct {
		ContextPath string
		Expected    bool
	}{
		{
			ContextPath: "../test/fixtures",
			Expected:    true,
		},
		{
			ContextPath: "/invalid/path",
			Expected:    false,
		},
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory.")
	}

	fakeExecutor := &fakeExecutor{}
	runner := goss.NewTestRunner(fakeExecutor, goss.TestRunnerOptions{
		ReportsDirectory: path.Join(cwd, "reports"),
		WorkingDirectory: cwd,
		JUnitReports:     false,
	})

	for _, data := range dataset {
		opts := types.RunTestOptions{
			ImageName:         "image",
			ImageReference:    "gcr.io/project/image:tag",
			DockerContextPath: path.Join(cwd, data.ContextPath),
		}

		assert.Equal(t, data.Expected, runner.Supports(opts))
	}
}

func Test_TestRunner_RunTest(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory.")
	}

	fakeExecutor := &fakeExecutor{}
	runner := goss.NewTestRunner(fakeExecutor, goss.TestRunnerOptions{
		ReportsDirectory: path.Join(cwd, "reports"),
		WorkingDirectory: cwd,
		JUnitReports:     false,
	})
	opts := types.RunTestOptions{
		ImageName:         "image",
		ImageReference:    "gcr.io/project/image:tag",
		DockerContextPath: path.Join(cwd, "../test/fixtures"),
	}

	err = runner.RunTest(opts)
	assert.NoError(t, err)
	assert.Equal(t, opts, fakeExecutor.RecordedOpts)
	assert.Empty(t, fakeExecutor.RecordedArgs)

	assert.NoFileExists(t, "reports/junit-image.xml")
}

func Test_TestRunner_RunTest_Junit(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory.")
	}

	fakeExecutor := &fakeExecutor{}
	runner := goss.NewTestRunner(fakeExecutor, goss.TestRunnerOptions{
		ReportsDirectory: path.Join(cwd, "reports"),
		WorkingDirectory: path.Join(cwd, "../test"),
		JUnitReports:     true,
	})
	opts := types.RunTestOptions{
		ImageName:         "image",
		ImageReference:    "gcr.io/project/image:tag",
		DockerContextPath: path.Join(cwd, "../test/fixtures"),
	}

	fakeExecutor.Output = `<testcase name="hello"></testcase>`

	err = runner.RunTest(opts)
	assert.NoError(t, err)
	assert.Equal(t, opts, fakeExecutor.RecordedOpts)
	assert.Equal(t, []string{"--format", "junit"}, fakeExecutor.RecordedArgs)

	assert.FileExists(t, "reports/junit-image.xml")
	expectedJunit := `<testcase classname="goss-image" file="fixtures" name="hello"></testcase>`
	actualJunit, err := os.ReadFile("reports/junit-image.xml")
	require.NoError(t, err)
	assert.Equal(t, expectedJunit, string(actualJunit))
	_ = os.RemoveAll("reports")
}
