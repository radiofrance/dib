package goss_test

import (
	"os"
	"testing"

	"github.com/radiofrance/dib/goss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nolint:paralleltest
func Test_DGossExecutor_NewDGossExecutorUsesDefaultShell(t *testing.T) {
	err := os.Unsetenv("SHELL")
	require.NoError(t, err)

	executor := goss.NewDGossExecutor()

	assert.Equal(t, "/bin/bash", executor.Shell)
}

// nolint:paralleltest
func Test_DGossExecutor_NewDGossExecutorDetectsShellFromEnv(t *testing.T) {
	t.Setenv("SHELL", "/path/to/shell")

	executor := goss.NewDGossExecutor()

	assert.Equal(t, "/path/to/shell", executor.Shell)
}
