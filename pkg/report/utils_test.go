package report_test

import (
	"testing"

	"github.com/radiofrance/dib/pkg/report"
	"github.com/stretchr/testify/assert"
)

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

		dibReport := report.Report{}
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

		dibReport := report.Report{}
		dibReport.Name = test.input

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := dibReport.GetBuildLogsDir()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestDIBReport_GetTestLogDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid report test logs dir 1",
			input:    "lorem",
			expected: "reports/lorem/tests",
		},
		{
			name:     "valid report test logs dir 2",
			input:    "20220823180000",
			expected: "reports/20220823180000/tests",
		},
	}

	for _, test := range tests {
		test := test

		dibReport := report.Report{}
		dibReport.Name = test.input

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := dibReport.GetTestLogsDir()
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

		dibReport := report.Report{}
		dibReport.Name = test.input

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := dibReport.GetJunitReportDir()
			assert.Equal(t, test.expected, actual)
		})
	}
}
