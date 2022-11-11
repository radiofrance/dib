package report_test

import (
	"testing"

	"github.com/radiofrance/dib/pkg/report"
	"github.com/stretchr/testify/assert"
)

func Test_CheckError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []report.BuildReport
		expected string
	}{
		{
			name: "All succeed",
			input: []report.BuildReport{
				{BuildStatus: 0, TestsStatus: 0},
				{BuildStatus: 0, TestsStatus: 0},
			},
			expected: "",
		},
		{
			name: "One of the image build failed",
			input: []report.BuildReport{
				{BuildStatus: 0, TestsStatus: 0},
				{BuildStatus: 2, TestsStatus: 0},
			},
			expected: "one of the image build failed, see logs for more details",
		},
		{
			name: "Some tests failed",
			input: []report.BuildReport{
				{BuildStatus: 0, TestsStatus: 0},
				{BuildStatus: 0, TestsStatus: 2},
			},
			expected: "some tests failed, see logs for more details",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := report.CheckError(test.input)
			if test.expected == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, test.expected)
			}
		})
	}
}
