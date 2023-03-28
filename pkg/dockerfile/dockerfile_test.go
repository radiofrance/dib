package dockerfile_test

import (
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		filename       string
		expectedFrom   []dockerfile.ImageRef
		expectedLabels map[string]string
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
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cwd, err := os.Getwd()
			if err != nil {
				t.Fatal("Failed to get current working directory.")
			}
			fullpath := path.Join(cwd, "../../test/fixtures/dockerfile", test.filename)

			result, err := dockerfile.ParseDockerfile(fullpath)
			assert.NoError(t, err)

			assert.Equal(t, test.expectedFrom, result.From)
			assert.Equal(t, test.expectedLabels, result.Labels)
		})
	}
}
