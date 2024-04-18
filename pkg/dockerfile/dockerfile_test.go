package dockerfile_test

import (
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDockerfile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		filename       string
		expectedFrom   []dockerfile.ImageRef
		expectedLabels map[string]string
		expectedArgs   map[string]string
	}{
		"simple dockerfile": {
			filename: "simple.dockerfile",
			expectedFrom: []dockerfile.ImageRef{
				{
					Name:   "registry.com/example",
					Tag:    "",
					Digest: "",
				},
			},
			expectedLabels: map[string]string{
				"name": "example",
			},
			expectedArgs: map[string]string{},
		},
		"simple dockerfile with digest": {
			filename: "simple-digest.dockerfile",
			expectedFrom: []dockerfile.ImageRef{
				{
					Name:   "registry.com/example",
					Tag:    "",
					Digest: "sha256:d23df29669d05462cf55ce2274a3a897aa2e2655d0fad104375f8ef06164b575",
				},
			},
			expectedLabels: map[string]string{
				"name": "example",
			},
			expectedArgs: map[string]string{},
		},
		"simple dockerfile with tag": {
			filename: "simple-tag.dockerfile",
			expectedFrom: []dockerfile.ImageRef{
				{
					Name:   "registry.com/example",
					Tag:    "latest",
					Digest: "",
				},
			},
			expectedLabels: map[string]string{
				"name": "example",
			},
			expectedArgs: map[string]string{},
		},
		"simple dockerfile with arg": {
			filename: "simple-arg.dockerfile",
			expectedFrom: []dockerfile.ImageRef{
				{
					Name:   "registry.com/example",
					Tag:    "latest",
					Digest: "",
				},
			},
			expectedLabels: map[string]string{},
			expectedArgs: map[string]string{
				"HELLO": `ARG HELLO="there"`,
			},
		},
		"simple dockerfile with tag and digest": {
			filename: "simple-tag-digest.dockerfile",
			expectedFrom: []dockerfile.ImageRef{
				{
					Name:   "registry.com/example",
					Tag:    "latest",
					Digest: "sha256:d23df29669d05462cf55ce2274a3a897aa2e2655d0fad104375f8ef06164b575",
				},
			},
			expectedLabels: map[string]string{
				"name": "example",
			},
			expectedArgs: map[string]string{},
		},
		"multistage dockerfile": {
			filename: "multistage.dockerfile",
			expectedFrom: []dockerfile.ImageRef{
				{
					Name:   "registry.com/builder",
					Tag:    "",
					Digest: "",
				},
				{
					Name:   "registry.com/example",
					Tag:    "",
					Digest: "",
				},
			},
			expectedLabels: map[string]string{
				"name": "example",
			},
			expectedArgs: map[string]string{},
		},
		"multistage dockerfile with digest": {
			filename: "multistage-digest.dockerfile",
			expectedFrom: []dockerfile.ImageRef{
				{
					Name:   "registry.com/builder",
					Tag:    "",
					Digest: "sha256:d23df29669d05462cf55ce2274a3a897aa2e2655d0fad104375f8ef06164b575",
				},
				{
					Name:   "registry.com/example",
					Tag:    "",
					Digest: "",
				},
			},
			expectedLabels: map[string]string{
				"name": "example",
			},
			expectedArgs: map[string]string{},
		},
		"multistage dockerfile with tag": {
			filename: "multistage-tag.dockerfile",
			expectedFrom: []dockerfile.ImageRef{
				{
					Name:   "registry.com/builder",
					Tag:    "latest",
					Digest: "",
				},
				{
					Name:   "registry.com/example",
					Tag:    "",
					Digest: "",
				},
			},
			expectedLabels: map[string]string{
				"name": "example",
			},
			expectedArgs: map[string]string{},
		},
		"multistage dockerfile with tag and digest": {
			filename: "multistage-tag-digest.dockerfile",
			expectedFrom: []dockerfile.ImageRef{
				{
					Name:   "registry.com/builder",
					Tag:    "latest",
					Digest: "sha256:d23df29669d05462cf55ce2274a3a897aa2e2655d0fad104375f8ef06164b575",
				},
				{
					Name:   "registry.com/example",
					Tag:    "",
					Digest: "",
				},
			},
			expectedLabels: map[string]string{
				"name": "example",
			},
			expectedArgs: map[string]string{},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cwd, err := os.Getwd()
			require.NoError(t, err)

			fullpath := path.Join(cwd, "../../test/fixtures/dockerfile", test.filename)
			result, err := dockerfile.ParseDockerfile(fullpath)
			require.NoError(t, err)

			assert.Equal(t, test.expectedFrom, result.From)
			assert.Equal(t, test.expectedLabels, result.Labels)
			assert.Equal(t, test.expectedArgs, result.Args)
		})
	}
}

func TestReplaceAndReset(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filename := path.Join(tmpDir, "replace.dockerfile")
	oldContent := `FROM registry.com/example\nLABEL name="example"`
	require.NoError(t, os.WriteFile(filename,
		[]byte(oldContent), 0o600))
	diff := map[string]string{
		"registry.com": "registries.io",
		"example":      "other",
	}
	newContent := `FROM registries.io/other\nLABEL name="other"`
	require.NoError(t, dockerfile.ReplaceInFile(filename, diff))
	content, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Equal(t, newContent, string(content))

	require.NoError(t, dockerfile.ResetFile(filename, diff))
	content, err = os.ReadFile(filename)
	require.NoError(t, err)
	assert.Equal(t, oldContent, string(content))
}
