package trivy_test

import (
	"os"
	"testing"

	"github.com/radiofrance/dib/pkg/trivy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest
func Test_LocalExecutor_NewLocalExecutorUsesDefaultShell(t *testing.T) {
	err := os.Unsetenv("SHELL")
	require.NoError(t, err)

	executor := trivy.NewLocalExecutor()

	assert.Equal(t, "/bin/bash", executor.Shell)
}

//nolint:paralleltest
func Test_LocalExecutor_NewLocalExecutorDetectsShellFromEnv(t *testing.T) {
	t.Setenv("SHELL", "/path/to/shell")

	executor := trivy.NewLocalExecutor()

	assert.Equal(t, "/path/to/shell", executor.Shell)
}
