package dib_test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/radiofrance/dib/pkg/dib"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	buildPath      = "../../test/fixtures/docker"
	registryPrefix = "eu.gcr.io/my-test-repository"
)

func TestGenerateDAG(t *testing.T) {
	t.Parallel()

	baseDir := buildPath + "/bullseye"
	bullseyeHash, err := dib.HashFiles(baseDir,
		[]string{
			baseDir + "/Dockerfile",
			baseDir + "/external-parent/Dockerfile",
			baseDir + "/multistage/Dockerfile",
			baseDir + "/skipbuild/Dockerfile",
			baseDir + "/sub-image/Dockerfile",
		}, nil, nil)
	require.NoError(t, err)

	extParentHash, err := dib.HashFiles(baseDir+"/external-parent",
		[]string{baseDir + "/external-parent/Dockerfile"},
		[]string{bullseyeHash}, nil)
	require.NoError(t, err)

	multistageHash, err := dib.HashFiles(baseDir+"/multistage",
		[]string{baseDir + "/multistage/Dockerfile"},
		[]string{bullseyeHash}, nil)
	require.NoError(t, err)

	subImageHash, err := dib.HashFiles(baseDir+"/sub-image",
		[]string{baseDir + "/sub-image/Dockerfile"},
		[]string{bullseyeHash}, nil)
	require.NoError(t, err)

	t.Run("nominal", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`docker
└──┬bullseye [%s]
   ├───kaniko [%s]
   ├───multistage [%s]
   └───sub-image [%s]
`, bullseyeHash, extParentHash, multistageHash, subImageHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("adding a file to the bullseye directory", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		baseDir := buildPath + "/bullseye"

		// When I add a new file in bullseye
		newFilePath := baseDir + "/newfile"
		require.NoError(t, os.WriteFile(newFilePath, []byte("any content"), 0o600))

		// Recompute hashes of bullseye and all its children because the context of bullseye has changed
		bullseyeHash, err := dib.HashFiles(baseDir, []string{
			baseDir + "/Dockerfile",
			baseDir + "/external-parent/Dockerfile",
			baseDir + "/multistage/Dockerfile",
			baseDir + "/skipbuild/Dockerfile",
			baseDir + "/sub-image/Dockerfile",
			newFilePath,
		}, nil, nil)
		require.NoError(t, err)

		extParentHash, err := dib.HashFiles(baseDir+"/external-parent",
			[]string{baseDir + "/external-parent/Dockerfile"},
			[]string{bullseyeHash}, nil)
		require.NoError(t, err)

		multistageHash, err := dib.HashFiles(baseDir+"/multistage",
			[]string{baseDir + "/multistage/Dockerfile"},
			[]string{bullseyeHash}, nil)
		require.NoError(t, err)

		subImageHash, err := dib.HashFiles(baseDir+"/sub-image",
			[]string{baseDir + "/sub-image/Dockerfile"},
			[]string{bullseyeHash}, nil)
		require.NoError(t, err)

		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`docker
└──┬bullseye [%s]
   ├───kaniko [%s]
   ├───multistage [%s]
   └───sub-image [%s]
`, bullseyeHash, extParentHash, multistageHash, subImageHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("adding a file to the multistage directory", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		baseDir := buildPath + "/bullseye"

		// When I add a new file in bullseye/multistage
		newFilePath := baseDir + "/multistage/newfile"
		require.NoError(t, os.WriteFile(newFilePath, []byte("any content"), 0o600))

		// Recompute hashes of bullseye and its children because the context of bullseye has changed
		bullseyeHash, err := dib.HashFiles(baseDir, []string{
			baseDir + "/Dockerfile",
			baseDir + "/external-parent/Dockerfile",
			baseDir + "/multistage/Dockerfile",
			baseDir + "/skipbuild/Dockerfile",
			baseDir + "/sub-image/Dockerfile",
			newFilePath,
		}, nil, nil)
		require.NoError(t, err)

		extParentHash, err := dib.HashFiles(baseDir+"/external-parent",
			[]string{baseDir + "/external-parent/Dockerfile"},
			[]string{bullseyeHash}, nil)
		require.NoError(t, err)

		multistageHash, err := dib.HashFiles(baseDir+"/multistage",
			[]string{
				baseDir + "/multistage/Dockerfile",
				newFilePath,
			},
			[]string{bullseyeHash}, nil)
		require.NoError(t, err)

		subImageHash, err := dib.HashFiles(baseDir+"/sub-image",
			[]string{baseDir + "/sub-image/Dockerfile"},
			[]string{bullseyeHash}, nil)
		require.NoError(t, err)

		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`docker
└──┬bullseye [%s]
   ├───kaniko [%s]
   ├───multistage [%s]
   └───sub-image [%s]
`, bullseyeHash, extParentHash, multistageHash, subImageHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("using custom hash list", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		// Recompute hash of sub-image, which is the only node that has the label 'dib.use-custom-hash-list'
		customHashListPath := "../../test/fixtures/dib/valid_wordlist.txt"
		list, err := dib.LoadCustomHashList(customHashListPath)
		require.NoError(t, err)

		subImageHash, err := dib.HashFiles(buildPath+"/bullseye/sub-image", []string{
			buildPath + "/bullseye/sub-image/Dockerfile",
		}, []string{bullseyeHash}, list)
		require.NoError(t, err)

		graph, err := dib.GenerateDAG(buildPath, registryPrefix, customHashListPath, nil)
		require.NoError(t, err)

		// Only the sub-image node, which has the label 'dib.use-custom-hash-list', should change
		assert.Equal(t, fmt.Sprintf(`docker
└──┬bullseye [%s]
   ├───kaniko [%s]
   ├───multistage [%s]
   └───sub-image [%s]
`, bullseyeHash, extParentHash, multistageHash, subImageHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("using build args", func(t *testing.T) {
		t.Parallel()

		buildPath := copyFixtures(t)

		baseDir := buildPath + "/bullseye"

		dckfile, err := dockerfile.ParseDockerfile(baseDir + "/Dockerfile")
		require.NoError(t, err)

		buildArgs := map[string]string{
			"HELLO": "world",
		}
		argInstructionsToReplace := make(map[string]string)
		for key, newArg := range buildArgs {
			prevArgInstruction, ok := dckfile.Args[key]
			if ok {
				argInstructionsToReplace[prevArgInstruction] = fmt.Sprintf("ARG %s=%s", key, newArg)
			}
		}
		require.NoError(t, dockerfile.ReplaceInFile(baseDir+"/Dockerfile", argInstructionsToReplace))

		// Recompute hashes of bullseye and all its children because the Dockerfile of bullseye has changed
		bullseyeHash, err := dib.HashFiles(baseDir, []string{
			baseDir + "/Dockerfile",
			baseDir + "/external-parent/Dockerfile",
			baseDir + "/multistage/Dockerfile",
			baseDir + "/skipbuild/Dockerfile",
			baseDir + "/sub-image/Dockerfile",
		}, nil, nil)
		require.NoError(t, err)

		extParentHash, err := dib.HashFiles(baseDir+"/external-parent",
			[]string{baseDir + "/external-parent/Dockerfile"},
			[]string{bullseyeHash}, nil)
		require.NoError(t, err)

		multistageHash, err := dib.HashFiles(baseDir+"/multistage",
			[]string{baseDir + "/multistage/Dockerfile"},
			[]string{bullseyeHash}, nil)
		require.NoError(t, err)

		subImageHash, err := dib.HashFiles(baseDir+"/sub-image",
			[]string{baseDir + "/sub-image/Dockerfile"},
			[]string{bullseyeHash}, nil)
		require.NoError(t, err)

		graph, err := dib.GenerateDAG(buildPath, registryPrefix, "", buildArgs)
		require.NoError(t, err)

		// Only bullseye node has the 'HELLO' argument, so its hash and all of its children should change
		assert.Equal(t, fmt.Sprintf(`docker
└──┬bullseye [%s]
   ├───kaniko [%s]
   ├───multistage [%s]
   └───sub-image [%s]
`, bullseyeHash, extParentHash, multistageHash, subImageHash),
			graph.Sprint(path.Base(buildPath)))
	})

	t.Run("duplicates image names", func(t *testing.T) {
		t.Parallel()

		buildPath := "../../test/fixtures/docker-duplicates"
		_, err := dib.GenerateDAG(buildPath, registryPrefix, "", nil)
		require.EqualError(t, err,
			fmt.Sprintf(`duplicate image name "%s/duplicate" found while reading file `+
				`"%s/bullseye/duplicate2/Dockerfile": previous file was "%s/bullseye/duplicate1/Dockerfile"`,
				registryPrefix, buildPath, buildPath))
	})
}

// copyFixtures copies the buildPath directory into a temporary one to be free to edit files.
func copyFixtures(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	src := path.Join(cwd, buildPath)
	dest := t.TempDir()
	cmd := exec.Command("cp", "-r", src, dest)
	require.NoError(t, cmd.Run())
	return dest + "/docker"
}

func TestLoadCustomHashList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       string
		expected    []string
		expectedErr string
	}{
		{
			name:        "standard wordlist",
			input:       "",
			expected:    nil,
			expectedErr: "",
		},
		{
			name:        "custom wordlist txt",
			input:       "../../test/fixtures/dib/wordlist.txt",
			expected:    []string{"a", "b", "c"},
			expectedErr: "",
		},
		{
			name:        "custom wordlist yml",
			input:       "../../test/fixtures/dib/wordlist.yml",
			expected:    []string{"e", "f", "g"},
			expectedErr: "",
		},
		{
			name:        "wordlist file not exist",
			input:       "../../test/fixtures/dib/lorem.txt",
			expected:    nil,
			expectedErr: "open ../../test/fixtures/dib/lorem.txt: no such file or directory",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual, err := dib.LoadCustomHashList(test.input)
			if test.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.expectedErr)
			}
			assert.Equal(t, test.expected, actual)
		})
	}
}
