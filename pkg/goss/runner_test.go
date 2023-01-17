package goss_test

import (
	"context"
	"io"
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/goss"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/types"
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
			ContextPath: "../../test/fixtures",
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

func Test_TestRunner_RunTest_Junit(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory.")
	}

	fakeExecutor := &fakeExecutor{}
	runner := goss.NewTestRunner(fakeExecutor, goss.TestRunnerOptions{
		WorkingDirectory: path.Join(cwd, "../../test"),
	})

	dibReport := report.InitDibReport("reports")
	err = dibReport.CreateJunitReportDir()
	assert.NoError(t, err)

	opts := types.RunTestOptions{
		ImageName:         "image",
		ImageReference:    "gcr.io/project/image:tag",
		DockerContextPath: path.Join(cwd, "../../test/fixtures"),
		ReportJunitDir:    dibReport.GetJunitReportDir(),
	}

	fakeExecutor.Output = `<testcase name="hello"></testcase>`

	err = runner.RunTest(opts)
	assert.NoError(t, err)
	assert.Equal(t, opts, fakeExecutor.RecordedOpts)
	assert.Equal(t, []string{"--format", "junit"}, fakeExecutor.RecordedArgs)

	testReportPath := path.Join(dibReport.GetJunitReportDir(), "junit-image.xml")
	assert.FileExists(t, testReportPath)
	expectedJunit := `<testcase classname="goss-image" file="fixtures" name="hello"></testcase>`
	actualJunit, err := os.ReadFile(testReportPath)
	require.NoError(t, err)
	assert.Equal(t, expectedJunit, string(actualJunit))
	_ = os.RemoveAll("reports")
}
