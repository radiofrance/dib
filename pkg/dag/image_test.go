package dag_test

import (
	"testing"

	"github.com/radiofrance/dib/pkg/dag"
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
