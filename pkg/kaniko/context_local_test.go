package kaniko_test

import (
	"context"
	"testing"

	"github.com/radiofrance/dib/pkg/kaniko"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LocalContextProvider(t *testing.T) {
	t.Parallel()

	contextProvider := kaniko.NewLocalContextProvider()

	opts := provideDefaultBuildOptions()

	URL, err := contextProvider.PrepareContext(context.Background(), opts)

	require.NoError(t, err)
	assert.Equal(t, "dir:///tmp/kaniko-context", URL)
}
