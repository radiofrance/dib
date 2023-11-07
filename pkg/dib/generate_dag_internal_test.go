package dib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_loadCustomHumanizedHashList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       string
		expected    []string
		expectedErr string
	}{
		{
			name:        "standard wordlist",
			input:       "",
			expected:    nil,
			expectedErr: "",
		},
		{
			name:        "custom wordlist txt",
			input:       "../../test/fixtures/dib/wordlist.txt",
			expected:    []string{"a", "b", "c"},
			expectedErr: "",
		},
		{
			name:        "custom wordlist yml",
			input:       "../../test/fixtures/dib/wordlist.yml",
			expected:    []string{"e", "f", "g"},
			expectedErr: "",
		},
		{
			name:     "wordlist file not exist",
			input:    "../../test/fixtures/dib/lorem.txt",
			expected: nil,
			expectedErr: "cannot load custom humanized word list file," +
				" err: open ../../test/fixtures/dib/lorem.txt: no such file or directory",
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual, err := loadCustomHumanizedHashList(test.input)

			if test.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.expectedErr)
			}
			assert.Equal(t, test.expected, actual)
		})
	}
}
