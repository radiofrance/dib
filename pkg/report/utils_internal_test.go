package report

import (
	"testing"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/goss"
	"github.com/radiofrance/dib/pkg/trivy"
	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestReport_isTestRunnerEnabled(t *testing.T) {
	t.Parallel()

	type input struct {
		name        string
		testRunners []types.TestRunner
	}

	tests := []struct {
		name     string
		input    input
		expected bool
	}{
		{
			name: "enabled",
			input: input{
				name: "goss",
				testRunners: []types.TestRunner{
					goss.TestRunner{},
					trivy.TestRunner{},
				},
			},
			expected: true,
		},
		{
			name: "disabled",
			input: input{
				name: "trivy",
				testRunners: []types.TestRunner{
					goss.TestRunner{},
				},
			},
			expected: false,
		},
		{
			name: "nil testRunners",
			input: input{
				name:        "goss",
				testRunners: nil,
			},
			expected: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := isTestRunnerEnabled(test.input.name, test.input.testRunners)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestReport_sortBuildReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []BuildReport
		expected []BuildReport
	}{
		{
			name: "valid 1",
			input: []BuildReport{
				{Image: dag.Image{ShortName: "bbb"}},
				{Image: dag.Image{ShortName: "aaa"}},
				{Image: dag.Image{ShortName: "ccc"}},
			},
			expected: []BuildReport{
				{Image: dag.Image{ShortName: "aaa"}},
				{Image: dag.Image{ShortName: "bbb"}},
				{Image: dag.Image{ShortName: "ccc"}},
			},
		},
		{
			name: "valid 2",
			input: []BuildReport{
				{Image: dag.Image{ShortName: "01bbb"}},
				{Image: dag.Image{ShortName: "#10214"}},
				{Image: dag.Image{ShortName: "01aaa"}},
				{Image: dag.Image{ShortName: "aaa"}},
			},
			expected: []BuildReport{
				{Image: dag.Image{ShortName: "#10214"}},
				{Image: dag.Image{ShortName: "01aaa"}},
				{Image: dag.Image{ShortName: "01bbb"}},
				{Image: dag.Image{ShortName: "aaa"}},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := sortBuildReport(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestReport_sortTrivyScan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    trivy.ScanReport
		expected trivy.ScanReport
	}{
		{
			name: "valid sorted Trivy ScanReport",
			input: trivy.ScanReport{
				Results: []trivy.Results{
					{
						Vulnerabilities: []trivy.Vulnerabilities{
							{Severity: "CRITICAL"},
							{Severity: "LOW"},
							{Severity: "CRITICAL"},
							{Severity: "HIGH"},
						},
					},
					{
						Vulnerabilities: []trivy.Vulnerabilities{
							{Severity: "HIGH"},
							{Severity: "CRITICAL"},
							{Severity: "LOW"},
							{Severity: "HIGH"},
							{Severity: "LOW"},
						},
					},
					{
						Vulnerabilities: []trivy.Vulnerabilities{
							{Severity: "UNKNOWN"},
							{Severity: "HIGH"},
							{Severity: "LOW"},
							{Severity: "CRITICAL"},
						},
					},
				},
			},
			expected: trivy.ScanReport{
				Results: []trivy.Results{
					{
						Vulnerabilities: []trivy.Vulnerabilities{
							{Severity: "CRITICAL"},
							{Severity: "CRITICAL"},
							{Severity: "HIGH"},
							{Severity: "LOW"},
						},
					},
					{
						Vulnerabilities: []trivy.Vulnerabilities{
							{Severity: "CRITICAL"},
							{Severity: "HIGH"},
							{Severity: "HIGH"},
							{Severity: "LOW"},
							{Severity: "LOW"},
						},
					},
					{
						Vulnerabilities: []trivy.Vulnerabilities{
							{Severity: "CRITICAL"},
							{Severity: "HIGH"},
							{Severity: "LOW"},
							{Severity: "UNKNOWN"},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := sortTrivyScan(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestReport_beautifyBuildsLogs(t *testing.T) {
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
			actual := beautifyBuildsLogs([]byte(test.input))
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestReport_sanitize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid 1",
			input:    "php8.2-fpm",
			expected: "php82-fpm",
		},
		{
			name:     "valid 2",
			input:    "a*bcd\\///@*!ef'&gh",
			expected: "abcdefgh",
		},
		{
			name:     "valid 3",
			input:    "ab cd ef gh",
			expected: "abcdefgh",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := sanitize(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}
