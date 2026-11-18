package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/radiofrance/dib/pkg/kubernetes"
	"github.com/stretchr/testify/assert"
)

func Test_exitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{name: "no error", err: nil, expected: 0},
		{name: "plain error", err: errors.New("x"), expected: 1},
		{
			name:     "pod disrupted",
			err:      fmt.Errorf("build failed: %w", fmt.Errorf("report: %w", kubernetes.ErrPodDisrupted)),
			expected: ExitCodePodDisrupted,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expected, exitCode(test.err))
		})
	}

	assert.Equal(t, 75, ExitCodePodDisrupted)
}
