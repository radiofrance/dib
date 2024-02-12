package report_test

import (
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/radiofrance/dib/internal/logger"
	"github.com/radiofrance/dib/pkg/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	lvl := "fatal"
	logger.SetLevel(&lvl)
	os.Exit(m.Run())
}

const reportsDir = "tests/reports"

func TestReport_GetRootDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid report root dir 1",
			input:    "lorem",
			expected: path.Join(reportsDir, "lorem"),
		},
		{
			name:     "valid report root dir 2",
			input:    "20220823180000",
			expected: path.Join(reportsDir, "20220823180000"),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dibReport := report.Report{
				Options: report.Options{
					Name:    test.input,
					RootDir: reportsDir,
				},
			}
			actual := dibReport.GetRootDir()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestReport_GetBuildLogDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid report build logs dir 1",
			input:    "lorem",
			expected: path.Join(reportsDir, "lorem/builds"),
		},
		{
			name:     "valid report build logs dir 2",
			input:    "20220823180000",
			expected: path.Join(reportsDir, "20220823180000/builds"),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dibReport := report.Report{
				Options: report.Options{
					RootDir: reportsDir,
					Name:    test.input,
				},
			}
			actual := dibReport.GetBuildReportDir()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestReport_GetJunitReportDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid junit report dir 1",
			input:    "lorem",
			expected: path.Join(reportsDir, "lorem/junit"),
		},
		{
			name:     "valid junit report dir 2",
			input:    "20220823180000",
			expected: path.Join(reportsDir, "20220823180000/junit"),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dibReport := report.Report{
				Options: report.Options{
					RootDir: reportsDir,
					Name:    test.input,
				},
			}
			actual := dibReport.GetJunitReportDir()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestReport_GetTrivyReportDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid trivy report dir 1",
			input:    "lorem",
			expected: path.Join(reportsDir, "lorem/trivy"),
		},
		{
			name:     "valid trivy report dir  2",
			input:    "20220823180000",
			expected: path.Join(reportsDir, "20220823180000/trivy"),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dibReport := report.Report{
				Options: report.Options{
					RootDir: reportsDir,
					Name:    test.input,
				},
			}
			actual := dibReport.GetTrivyReportDir()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestReport_GetReportURL_Gitlab(t *testing.T) { 
	t.Setenv("CI_JOB_URL", "https://gitlab.com/example-repository/-/jobs/123456")
	dibReport := report.Report{
		Options: report.Options{
			RootDir: reportsDir,
			Name:    "20220823183000",
		},
	}
	actual := dibReport.GetURL()
	expected := fmt.Sprintf(
		"https://gitlab.com/example-repository/-/jobs/123456/artifacts/file/%s/20220823183000/index.html",
		reportsDir)
	assert.Equal(t, expected, actual)
}

func TestReport_GetReportURL_Local(t *testing.T) { //nolint:paralleltest
	dibReport := report.Report{
		Options: report.Options{
			RootDir: reportsDir,
			Name:    "20220823183000",
		},
	}
	actual := dibReport.GetURL()
	expected := regexp.MustCompile(path.Join("file://.*", reportsDir, "20220823183000/index.html"))
	assert.Regexp(t, expected, actual)
}

func TestReport_CheckError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    report.Report
		expected string
	}{
		{
			name: "All succeed",
			input: report.Report{
				BuildReports: []report.BuildReport{
					{BuildStatus: 0, TestsStatus: 0},
					{BuildStatus: 0, TestsStatus: 0},
				},
			},
			expected: "",
		},
		{
			name: "One of the image build failed",
			input: report.Report{
				BuildReports: []report.BuildReport{
					{BuildStatus: 0, TestsStatus: 0},
					{BuildStatus: 2, TestsStatus: 0},
				},
			},
			expected: "one of the image build failed, see the report for more details",
		},
		{
			name: "Some tests failed",
			input: report.Report{
				BuildReports: []report.BuildReport{
					{BuildStatus: 0, TestsStatus: 0},
					{BuildStatus: 0, TestsStatus: 2},
				},
			},
			expected: "some tests failed, see report for more details",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := test.input.CheckError()
			if test.expected == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestReport_WithError(t *testing.T) {
	t.Parallel()

	expected := report.BuildReport{
		BuildStatus:    report.BuildStatusError,
		FailureMessage: "build failed for some reasons",
	}

	buildReport := report.BuildReport{}
	err := errors.New("build failed for some reasons")

	actual := buildReport.WithError(err)
	assert.Equal(t, expected, actual)
}
