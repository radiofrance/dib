package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
)

func Test_exitCode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, exitCode(errors.New("x")))
	assert.Equal(t, 75, exitCode(fmt.Errorf("build failed: %w", fmt.Errorf("report: %w", types.ErrPodDisrupted))))
}
