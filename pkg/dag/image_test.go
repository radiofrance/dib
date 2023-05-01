package dag_test

import (
	"testing"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/stretchr/testify/assert"
)

func Test_CurrentRef_EqualsHashWhenNoRebuildNeeded(t *testing.T) {
	t.Parallel()

	image := dag.Image{
		Name: "gcr.io/project-id/nginx",
		Hash: "version",
	}

	assert.Equal(t, "gcr.io/project-id/nginx:version", image.CurrentRef())
}

func Test_CurrentRef_HasDevPrefixWhenNeedsRebuild(t *testing.T) {
	t.Parallel()

	image := dag.Image{
		Name:         "gcr.io/project-id/nginx",
		Hash:         "version",
		NeedsRebuild: true,
	}

	assert.Equal(t, "gcr.io/project-id/nginx:dev-version", image.CurrentRef())
}

func Test_DockerRef(t *testing.T) {
	t.Parallel()

	image := dag.Image{
		Name: "gcr.io/project-id/nginx",
	}

	assert.Equal(t, "gcr.io/project-id/nginx:version", image.DockerRef("version"))
}

func Test_Print(t *testing.T) {
	t.Parallel()

	image := dag.Image{
		Name:      "registry.example.org/alpine-base",
		ShortName: "alpine-base",
		Hash:      "hak-una-mat-ata",
		Dockerfile: &dockerfile.Dockerfile{
			ContextPath: "/example/project/docker/base/alpine",
			Filename:    "Dockerfile",
			From: []dockerfile.ImageRef{
				{
					Name:   "alpine",
					Tag:    "3.17",
					Digest: "9ed4aefc74f6792b5a804d1d146fe4b4a2299147b0f50eaf2b08435d7b38c27e",
				},
			},
			Labels: map[string]string{
				"dib.extra-tags": "3.17",
			},
		},
	}

	expected := "" +
		"name: registry.example.org/alpine-base\n" +
		"short_name: alpine-base\n" +
		"hash: hak-una-mat-ata\n" +
		"dockerfile:\n" +
		"    contextpath: /example/project/docker/base/alpine\n" +
		"    filename: Dockerfile\n" +
		"    from:\n" +
		"        - name: alpine\n" +
		"          tag: \"3.17\"\n" +
		"          digest: 9ed4aefc74f6792b5a804d1d146fe4b4a2299147b0f50eaf2b08435d7b38c27e\n" +
		"    labels:\n" +
		"        dib.extra-tags: \"3.17\"\n"
	assert.Equal(t, expected, image.Print())
}
