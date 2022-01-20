package kaniko_test

import (
	"testing"

	"github.com/radiofrance/dib/kaniko"
	"github.com/stretchr/testify/assert"
)

func Test_LocalContextProvider(t *testing.T) {
	contextProvider := kaniko.NewLocalContextProvider()

	opts := provideDefaultBuildOptions()

	URL, err := contextProvider.PrepareContext(opts)

	assert.NoError(t, err)
	assert.Equal(t, "dir:///tmp/kaniko-context", URL)
}
