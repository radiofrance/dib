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

	dir := t.TempDir()

	dotFile := path.Join(dir, "dib.dot")
	err = graphviz.GenerateDotviz(graph, dotFile)
	require.NoError(t, err)
	assert.FileExists(t, dotFile)

	content, err := os.ReadFile(dotFile)
	require.NoError(t, err)
	f := string(content)
	assert.Len(t, f, 1490)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/root1" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/root1" -> "eu.gcr.io/my-test-repository/custom-hash-list";`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/root1" -> "eu.gcr.io/my-test-repository/dockerignore";`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/root1" -> "eu.gcr.io/my-test-repository/multistage";`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/root1" -> "eu.gcr.io/my-test-repository/sub1";`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/sub1" -> "eu.gcr.io/my-test-repository/sub2";`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/root1" -> "eu.gcr.io/my-test-repository/with-a-file";`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/dockerignore" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/multistage" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/with-a-file" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/custom-hash-list" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/sub1" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/sub2" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/root2" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/root3" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/two-parents" [fillcolor=white style=filled];`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/root1" -> "eu.gcr.io/my-test-repository/two-parents";`)
	assert.Contains(t, f, `"eu.gcr.io/my-test-repository/root2" -> "eu.gcr.io/my-test-repository/two-parents";`)
}
