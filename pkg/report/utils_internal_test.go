package report

import (
	"testing"

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
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := isTestRunnerEnabled(test.input.name, test.input.testRunners)
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
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			actual := beautifyBuildsLogs([]byte(test.input))
			assert.Equal(t, test.expected, actual)
		})
	}
}
