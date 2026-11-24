package types_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestExitCode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, types.ExitCode(errors.New("x")))
	assert.Equal(t, 75, types.ExitCode(fmt.Errorf("build failed: %w", fmt.Errorf("report: %w", types.ErrPodDisrupted))))
}
