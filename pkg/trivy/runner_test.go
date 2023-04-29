package trivy_test

import (
	"context"
	"io"
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/trivy"

	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeExecutor struct {
	Error        error
	Output       string
	RecordedArgs []string
}

func (e *fakeExecutor) Execute(_ context.Context, output io.Writer, args ...string) error {
	e.RecordedArgs = args
	if _, err := output.Write([]byte(e.Output)); err != nil {
		return err
	}
	return e.Error
}

func Test_TestRunner_RunTest(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory.")
	}

	fakeExecutor := &fakeExecutor{}
	runner := trivy.NewTestRunner(fakeExecutor, trivy.TestRunnerOptions{
		WorkingDirectory: path.Join(cwd, "../../test"),
	})

	dibReport := report.Init(
		"1.0.0",
		"reports",
		false,
		nil,
	)
	assert.NoError(t, err)

	opts := types.RunTestOptions{
		ImageName:         "image",
		ImageReference:    "gcr.io/project/image:tag",
		DockerContextPath: path.Join(cwd, "../../test/fixtures"),
		ReportJunitDir:    dibReport.GetJunitReportDir(),
		ReportTrivyDir:    dibReport.GetTrivyReportDir(),
	}

	fakeExecutor.Output = `{}`

	err = runner.RunTest(opts)
	assert.NoError(t, err)
	assert.Equal(t, []string{"image", "--quiet", "--format", "json", "gcr.io/project/image:tag"},
		fakeExecutor.RecordedArgs)

	testReportPath := path.Join(dibReport.GetTrivyReportDir(), "image.json")
	assert.FileExists(t, testReportPath)
	expectedContent := `{}`
	actualContent, err := os.ReadFile(testReportPath)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(actualContent))
	_ = os.RemoveAll("reports")
}
