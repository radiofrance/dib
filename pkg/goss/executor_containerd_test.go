//nolint:testpackage
package goss

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest
func Test_ContainerdGossExecutor_NewContainerdGossExecutorUsesDefaultShell(t *testing.T) {
	err := os.Unsetenv("SHELL")
	require.NoError(t, err)

	executor := NewContainerdGossExecutor()

	assert.Equal(t, "/bin/bash", executor.Shell)
}

func Test_ContainerdGossExecutor_NewContainerdGossExecutorDetectsShellFromEnv(t *testing.T) {
	t.Setenv("SHELL", "/path/to/shell")

	executor := NewContainerdGossExecutor()

	assert.Equal(t, "/path/to/shell", executor.Shell)
}
