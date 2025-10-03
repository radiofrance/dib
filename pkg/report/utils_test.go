package report_test

import (
	"os"
	"testing"

	"github.com/radiofrance/dib/pkg/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReport_RemoveTerminalColors(t *testing.T) {
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

			actual := report.RemoveTerminalColors([]byte(test.input))
			assert.Equal(t, test.expected, string(actual))
		})
	}
}

func TestReport_StripKanikoBuildLogs(t *testing.T) {
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
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(test.input)
			require.NoError(t, err)

			expected, err := os.ReadFile(test.expected)
			require.NoError(t, err)

			actual := report.StripKanikoBuildLogs(input)
			assert.Equal(t, string(expected), actual)
		})
	}
}
