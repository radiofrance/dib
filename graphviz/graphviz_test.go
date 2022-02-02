package graphviz_test

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/dib"
	"github.com/radiofrance/dib/graphviz"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GenerateDotviz(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory.")
	}

	DAG := dib.GenerateDAG(path.Join(cwd, "../test/fixtures/docker"), "eu.gcr.io/my-test-repository")

	dir, err := ioutil.TempDir("/tmp", "dib-test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	dotFile := path.Join(dir, "dib.dot")
	err = graphviz.GenerateDotviz(DAG, dotFile)
	require.NoError(t, err)
	assert.FileExists(t, dotFile)

	content, err := ioutil.ReadFile(dotFile)
	require.NoError(t, err)
	assert.Len(t, content, 647)
	assert.Contains(t, string(content),
		"\"eu.gcr.io/my-test-repository/bullseye\" [fillcolor=white style=filled];")
	assert.Contains(t, string(content),
		"\"eu.gcr.io/my-test-repository/bullseye\" -> \"eu.gcr.io/my-test-repository/kaniko\";")
	assert.Contains(t, string(content),
		"\"eu.gcr.io/my-test-repository/bullseye\" -> \"eu.gcr.io/my-test-repository/multistage\";")
	assert.Contains(t, string(content),
		"\"eu.gcr.io/my-test-repository/bullseye\" -> \"eu.gcr.io/my-test-repository/sub-image\";")
	assert.Contains(t, string(content),
		"\"eu.gcr.io/my-test-repository/kaniko\" [fillcolor=white style=filled];")
	assert.Contains(t, string(content),
		"\"eu.gcr.io/my-test-repository/multistage\" [fillcolor=white style=filled];")
	assert.Contains(t, string(content),
		"\"eu.gcr.io/my-test-repository/sub-image\" [fillcolor=white style=filled];")
}
