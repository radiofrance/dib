package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestBuildCommand(t *testing.T) {
	t.Parallel()
	cmd := buildCommand()

	assert.NotNil(t, cmd)
	assert.IsType(t, &cobra.Command{}, cmd)
	assert.Equal(t, "build", cmd.Use)
	assert.Equal(t, "Run oci images builds", cmd.Short)
	assert.Contains(t, cmd.Long, "dib build will compute the graph of images")
	assert.NotNil(t, cmd.RunE)
}
