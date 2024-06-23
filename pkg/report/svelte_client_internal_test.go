package report

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_copyAssetsFiles(t *testing.T) {
	t.Parallel()

	reportDir, err := os.MkdirTemp("/tmp", "dib-test-report")
	require.NoError(t, err)
	defer os.RemoveAll(reportDir)

	report := Report{
		Options: Options{RootDir: reportDir},
	}

	mockedAssetsFS := fstest.MapFS{
		// These files should not be present on final report dir
		".undesired":    &fstest.MapFile{},
		"undesired.txt": &fstest.MapFile{},
		// These files should be present on final report dir
		"build/.hidden":         &fstest.MapFile{},
		"build/_app/version.js": &fstest.MapFile{},
		"build/_app/env.js":     &fstest.MapFile{},
		"build/dag.png":         &fstest.MapFile{},
		"build/favicon.png":     &fstest.MapFile{},
		"build/index.html":      &fstest.MapFile{},
	}

	expectedFS := fstest.MapFS{
		"report":                 &fstest.MapFile{},
		"report/.hidden":         &fstest.MapFile{},
		"report/_app":            &fstest.MapFile{},
		"report/_app/version.js": &fstest.MapFile{},
		"report/_app/env.js":     &fstest.MapFile{},
		"build/dag.png":          &fstest.MapFile{},
		"report/favicon.png":     &fstest.MapFile{},
		"report/index.html":      &fstest.MapFile{},
	}

	err = copyAssetsFiles(mockedAssetsFS, "build", &report)
	require.NoError(t, err)

	// Walk on generated report...
	actualFS := make(fstest.MapFS)
	err = filepath.Walk(reportDir, func(path string, _ fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		actualFS[path] = &fstest.MapFile{}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, len(actualFS), len(expectedFS))
}
