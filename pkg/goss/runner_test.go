package goss_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/goss"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	lvl := "fatal"
	logger.SetLevel(&lvl)
	os.Exit(m.Run())
}

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
			ContextPath: "../../test/fixtures/build",
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

		assert.Equal(t, data.Expected, runner.IsConfigured(opts))
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

	dibReport := report.Init("1.0.0", "reports", false, nil, "")
	err = os.MkdirAll(dibReport.GetJunitReportDir(), 0o750)
	require.NoError(t, err)

	opts := types.RunTestOptions{
		ImageName:         "image",
		ImageReference:    "gcr.io/project/image:tag",
		DockerContextPath: path.Join(cwd, "../../test/fixtures/build"),
		ReportJunitDir:    dibReport.GetJunitReportDir(),
	}

	fakeExecutor.Output = `<testcase name="hello"></testcase>`

	err = runner.RunTest(opts)
	require.NoError(t, err)
	assert.Equal(t, opts, fakeExecutor.RecordedOpts)
	assert.Equal(t, []string{"--format", "junit"}, fakeExecutor.RecordedArgs)

	testReportPath := path.Join(dibReport.GetJunitReportDir(), "junit-image.xml")
	assert.FileExists(t, testReportPath)
	expectedJunit := `<testcase classname="goss-image" file="fixtures/build" name="hello"></testcase>`
	actualJunit, err := os.ReadFile(testReportPath) //nolint:gosec
	require.NoError(t, err)
	assert.Equal(t, expectedJunit, string(actualJunit))
	_ = os.RemoveAll("reports")
}

func Test_CreateTestRunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		kubernetesEnabled    bool
		localOnly            bool
		backend              string
		expectedExecutorType string
	}{
		// kubernetes test case should be enabled when integration tests are introduced,
		// as it requires a real Kubernetes environment to run.
		{
			name:                 "local only with docker backend",
			kubernetesEnabled:    false,
			localOnly:            true,
			backend:              types.BackendDocker,
			expectedExecutorType: "*goss.DGossExecutor",
		},
		{
			name:                 "local only with buildkit backend",
			kubernetesEnabled:    false,
			localOnly:            true,
			backend:              types.BuildKitBackend,
			expectedExecutorType: "*goss.ContainerdGossExecutor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a config with the kubernetes enabled flag
			config := goss.Config{}
			config.Executor.Kubernetes.Enabled = tt.kubernetesEnabled

			// Create a test runner using our helper function
			runner, err := goss.CreateTestRunner(config, tt.localOnly, "", tt.backend)
			require.NoError(t, err)

			// Verify the results
			require.NotNil(t, runner)

			// Check the type of the executor
			executorType := fmt.Sprintf("%T", runner.Executor)
			assert.Equal(t, tt.expectedExecutorType, executorType)
		})
	}
}
