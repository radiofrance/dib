package graphviz_test

import (
	"os"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/graphviz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GenerateDotviz(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	graph, err := dib.GenerateDAG(
		path.Join(cwd, "../../test/fixtures/docker"),
		"eu.gcr.io/my-test-repository", "",
		map[string]string{})
	require.NoError(t, err)

	dir, err := os.MkdirTemp("/tmp", "dib-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	dotFile := path.Join(dir, "dib.dot")
	err = graphviz.GenerateDotviz(graph, dotFile)
	require.NoError(t, err)
	assert.FileExists(t, dotFile)

	content, err := os.ReadFile(dotFile)
	require.NoError(t, err)
	f := string(content)
	assert.Len(t, f, 647)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/bullseye" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/bullseye" -> "eu.gcr.io/my-test-repository/kaniko";`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/bullseye" -> "eu.gcr.io/my-test-repository/multistage";`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/bullseye" -> "eu.gcr.io/my-test-repository/sub-image";`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/kaniko" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/multistage" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/sub-image" [fillcolor=white style=filled];`)
}
