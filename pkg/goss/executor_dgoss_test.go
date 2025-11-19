package goss_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radiofrance/dib/pkg/goss"
)

//nolint:paralleltest
func Test_DGossExecutor_NewDGossExecutorUsesDefaultShell(t *testing.T) {
	err := os.Unsetenv("SHELL")
	require.NoError(t, err)

	executor := goss.NewDGossExecutor()

	assert.Equal(t, "/bin/bash", executor.Shell)
}

func Test_DGossExecutor_NewDGossExecutorDetectsShellFromEnv(t *testing.T) {
	t.Setenv("SHELL", "/path/to/shell")

	executor := goss.NewDGossExecutor()

	assert.Equal(t, "/path/to/shell", executor.Shell)
}
