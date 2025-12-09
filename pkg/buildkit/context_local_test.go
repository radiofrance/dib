//nolint:testpackage
package buildkit

import (
	"context"
	"testing"

	"github.com/radiofrance/dib/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareLocalContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		opts     types.ImageBuilderOpts
		expected string
		err      error
	}{
		{
			name: "valid context",
			opts: types.ImageBuilderOpts{
				Context: "valid-context",
			},
			expected: "valid-context",
			err:      nil,
		},
		{
			name: "empty context",
			opts: types.ImageBuilderOpts{
				Context: "",
			},
			expected: "",
			err:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := LocalContextProvider{}
			result, err := provider.PrepareContext(context.Background(), tt.opts)
			assert.Equal(t, tt.expected, result)
			require.ErrorIs(t, err, tt.err)
		})
	}
}

func TestNewLocalContextProvider(t *testing.T) {
	t.Parallel()

	provider := NewLocalContextProvider()
	assert.NotNil(t, provider)
}
