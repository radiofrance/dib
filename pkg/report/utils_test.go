package report_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/radiofrance/dib/pkg/report"
	"github.com/stretchr/testify/assert"
)

func TestDIBReport_RemoveTerminalColors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid 1",
			input:    "\u001b[31mHello World",
			expected: "Hello World",
		},
		{
			name:     "valid 2",
			input:    "\u001b[30mA \u001b[31m B \u001b[32m C \u001b[33m D\u001b[0m",
			expected: "A  B  C  D",
		},
		{
			name:     "valid 3",
			input:    "\u001B[91mE: Unable to locate package lorem",
			expected: "E: Unable to locate package lorem",
		},
		{
			name:     "valid 4",
			input:    "\u001B[0mThe command 'apt-get install -y lorem' returned a non-zero code: 100",
			expected: "The command 'apt-get install -y lorem' returned a non-zero code: 100",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := report.RemoveTerminalColors([]byte(test.input))
			assert.Equal(t, test.expected, string(actual))
		})
	}
}

func TestDIBReport_StripKanikoBuildLogs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid 1 (single line)",
			input:    "../../test/fixtures/report/build_logs/kaniko/1_raw_log.txt",
			expected: "../../test/fixtures/report/build_logs/kaniko/1_parsed_log.txt",
		},
		{
			name:     "valid 1 (real case)",
			input:    "../../test/fixtures/report/build_logs/kaniko/2_raw_log.txt",
			expected: "../../test/fixtures/report/build_logs/kaniko/2_parsed_log.txt",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(test.input)
			assert.NoError(t, err)

			expected, err := os.ReadFile(test.expected)
			assert.NoError(t, err)

			actual := report.StripKanikoBuildLogs(input)
			assert.Equal(t, string(expected), actual)
		})
	}
}

func TestDIBReport_GetRootDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid report root dir 1",
			input:    "lorem",
			expected: "reports/lorem",
		},
		{
			name:     "valid report root dir 2",
			input:    "20220823180000",
			expected: "reports/20220823180000",
		},
	}

	for _, test := range tests {
		test := test

		dibReport := report.Report{
			Name: test.input,
			Dir:  "reports",
		}
		dibReport.Name = test.input

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := dibReport.GetRootDir()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestDIBReport_GetBuildLogDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid report build logs dir 1",
			input:    "lorem",
			expected: "reports/lorem/builds",
		},
		{
			name:     "valid report build logs dir 2",
			input:    "20220823180000",
			expected: "reports/20220823180000/builds",
		},
	}

	for _, test := range tests {
		test := test

		dibReport := report.Report{
			Name: test.input,
			Dir:  "reports",
		}
		dibReport.Name = test.input

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := dibReport.GetBuildLogsDir()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestDIBReport_GetJunitReportDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid report junit report dir 1",
			input:    "lorem",
			expected: "reports/lorem/junit",
		},
		{
			name:     "valid report junit report dir 2",
			input:    "20220823180000",
			expected: "reports/20220823180000/junit",
		},
	}

	for _, test := range tests {
		test := test

		dibReport := report.Report{
			Name: test.input,
			Dir:  "reports",
		}

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := dibReport.GetJunitReportDir()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestReport_GetReportURL_Gitlab(t *testing.T) { //nolint:paralleltest
	t.Setenv("CI_JOB_URL", "https://gitlab.com/example-repository/-/jobs/123456")

	dibReport := report.Report{
		Name: "20220823183000",
		Dir:  "reports",
	}

	actual := dibReport.GetReportURL()
	expected := "https://gitlab.com/example-repository/-/jobs/123456/artifacts/file/reports/20220823183000/index.html"
	assert.Equal(t, expected, actual)
}

func TestReport_GetReportURL_Local(t *testing.T) { //nolint:paralleltest
	dibReport := report.Report{
		Name: "20220823183000",
		Dir:  "reports",
	}
	actual := dibReport.GetReportURL()
	expected := regexp.MustCompile("file://.*/reports/20220823183000/index.html")
	assert.Regexp(t, expected, actual)
}
