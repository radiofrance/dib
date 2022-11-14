package junit_test

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/radiofrance/dib/pkg/junit"
	"github.com/stretchr/testify/assert"
)

func Test_ParseRawLogs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		input            string
		expected         junit.Testsuite
		expectedErrorMsg string
	}{
		{
			name:  "Goss tests succeed",
			input: "../../test/fixtures/junit/junit-image-test.xml",
			expected: junit.Testsuite{
				XMLName:   xml.Name{Local: "testsuite"},
				Name:      "goss",
				Errors:    "0",
				Tests:     "2",
				Failures:  "0",
				Skipped:   "0",
				Time:      "0.000",
				Timestamp: "2022-10-20T18:29:26Z",
				TestCases: []junit.TestCase{
					{
						XMLName:   xml.Name{Local: "testcase"},
						ClassName: "goss-image-test",
						File:      "docker/image-test",
						Time:      "0.000",
						Name:      "Test lorem 1",
						SystemOut: "Test results lorem 1",
					},
					{
						XMLName:   xml.Name{Local: "testcase"},
						ClassName: "goss-image-test",
						File:      "docker/image-test",
						Time:      "0.000",
						Name:      "Test lorem 2",
						SystemOut: "Test results lorem 2",
					},
				},
			},
			expectedErrorMsg: "",
		},
		{
			name:  "Some Goss tests failed",
			input: "../../test/fixtures/junit/junit-image-test-fail.xml",
			expected: junit.Testsuite{
				XMLName:   xml.Name{Local: "testsuite"},
				Name:      "goss",
				Errors:    "0",
				Tests:     "2",
				Failures:  "1",
				Skipped:   "0",
				Time:      "0.000",
				Timestamp: "2022-10-20T18:29:26Z",
				TestCases: []junit.TestCase{
					{
						XMLName:   xml.Name{Local: "testcase"},
						ClassName: "goss-image-test",
						File:      "docker/image-test",
						Time:      "0.000",
						Name:      "Test lorem 1",
						SystemOut: "Test results lorem 1",
					},
					{
						XMLName:   xml.Name{Local: "testcase"},
						ClassName: "goss-image-test",
						File:      "docker/image-test",
						Time:      "0.000",
						Name:      "User debian uid",
						Failure:   "User: debian: uid: doesn't match, expect: [1666] found: [1664]",
					},
				},
			},
			expectedErrorMsg: "",
		},
		{
			name:  "Invalid XML",
			input: "../../test/fixtures/junit/junit-invalid.xml",
			expected: junit.Testsuite{
				XMLName:   xml.Name{Local: "testsuite"},
				TestCases: []junit.TestCase(nil),
			},
			expectedErrorMsg: "expected element name after </",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(test.input)
			assert.NoError(t, err)
			actual, err := junit.ParseRawLogs(data)
			assert.Equal(t, test.expected, actual)

			if test.expectedErrorMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.expectedErrorMsg)
			}
		})
	}
}
