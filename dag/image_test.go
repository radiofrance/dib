package dag_test

import (
	"testing"

	"github.com/radiofrance/dib/dag"
	"github.com/stretchr/testify/assert"
)

func Test_DockerRef(t *testing.T) {
	t.Parallel()

	image := dag.NewImage(dag.NewImageArgs{
		Name: "gcr.io/project-id/nginx",
	})

	assert.Equal(t, "gcr.io/project-id/nginx:version", image.DockerRef("version"))
}
