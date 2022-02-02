package dib_test

import (
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/dib"
	"github.com/stretchr/testify/assert"
)

func Test_GenerateDAG(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory.")
	}

	DAG := dib.GenerateDAG(
		path.Join(cwd, "../test/fixtures/docker"),
		"eu.gcr.io/my-test-repository",
	)

	assert.Len(t, DAG.Nodes(), 1)

	rootNode := DAG.Nodes()[0]
	rootImage := rootNode.Image
	assert.Equal(t, "eu.gcr.io/my-test-repository/bullseye", rootImage.Name)
	assert.Equal(t, "bullseye", rootImage.ShortName)
	assert.Len(t, rootNode.Parents(), 0)
	assert.Len(t, rootNode.Children(), 3)
}
