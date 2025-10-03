//nolint:paralleltest
package dib

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/radiofrance/dib/pkg/dag"
	"github.com/radiofrance/dib/pkg/dockerfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	basePath       = "../../test/fixtures/docker"
	registryPrefix = "eu.gcr.io/my-test-repository"
)

func TestGenerateDAG(t *testing.T) {
	dirRoot1 := basePath + "/root1"
	hashRoot1, err := hashFiles(dirRoot1,
		[]string{
			dirRoot1 + "/Dockerfile",
			dirRoot1 + "/custom-hash-list/Dockerfile",
			dirRoot1 + "/dockerignore/.dockerignore",
			dirRoot1 + "/dockerignore/Dockerfile",
			dirRoot1 + "/dockerignore/ignored.txt",
			dirRoot1 + "/multistage/Dockerfile",
			dirRoot1 + "/skipbuild/Dockerfile",
			dirRoot1 + "/sub1/Dockerfile",
			dirRoot1 + "/sub1/sub2/Dockerfile",
			dirRoot1 + "/with-a-file/Dockerfile",
			dirRoot1 + "/with-a-file/included.txt",
		}, nil, nil)
	require.NoError(t, err)

	hashCHL, err := hashFiles(dirRoot1+"/custom-hash-list",
		[]string{dirRoot1 + "/custom-hash-list/Dockerfile"},
		[]string{hashRoot1}, nil)
	require.NoError(t, err)

	hashDockerignore, err := hashFiles(dirRoot1+"/dockerignore",
		[]string{dirRoot1 + "/dockerignore/Dockerfile"},
		[]string{hashRoot1}, nil)
	require.NoError(t, err)

	hashMultistage, err := hashFiles(dirRoot1+"/multistage",
		[]string{dirRoot1 + "/multistage/Dockerfile"},
		[]string{hashRoot1}, nil)
	require.NoError(t, err)

	hashSub1, err := hashFiles(dirRoot1+"/sub1",
		[]string{
			dirRoot1 + "/sub1/Dockerfile",
			dirRoot1 + "/sub1/sub2/Dockerfile",
		},
		[]string{hashRoot1}, nil)
	require.NoError(t, err)

	hashSub2, err := hashFiles(dirRoot1+"/sub1/sub2",
		[]string{
			dirRoot1 + "/sub1/sub2/Dockerfile",
		},
		[]string{hashSub1}, nil)
	require.NoError(t, err)

	hashWithAFile, err := hashFiles(dirRoot1+"/with-a-file",
		[]string{
			dirRoot1 + "/with-a-file/Dockerfile",
			dirRoot1 + "/with-a-file/included.txt",
		},
		[]string{hashRoot1}, nil)
	require.NoError(t, err)

	dirRoot2 := basePath + "/root2"
	hashRoot2, err := hashFiles(dirRoot2,
		[]string{
			dirRoot2 + "/Dockerfile",
			dirRoot2 + "/root3/Dockerfile",
		}, nil, nil)
	require.NoError(t, err)

	dirRoot3 := dirRoot2 + "/root3"
	hashRoot3, err := hashFiles(dirRoot3,
		[]string{
			dirRoot3 + "/Dockerfile",
		}, nil, nil)
	require.NoError(t, err)

	hashTwoParents, err := hashFiles(basePath+"/two-parents",
		[]string{
			basePath + "/two-parents/Dockerfile",
		},
		[]string{hashRoot1, hashRoot2}, nil)
	require.NoError(t, err)

	graph, err := GenerateDAG(basePath, registryPrefix, "", nil)
	require.NoError(t, err)

	nominalGraph := graph.Sprint(path.Base(basePath))
	assert.Equal(t, fmt.Sprintf(`docker
├──┬root1 [%s]
│  ├───custom-hash-list [%s]
│  ├───dockerignore [%s]
│  ├───multistage [%s]
│  ├──┬sub1 [%s]
│  │  └───sub2 [%s]
│  ├───two-parents [%s]
│  └───with-a-file [%s]
├──┬root2 [%s]
│  └───two-parents [%s]
└───root3 [%s]
`, hashRoot1, hashCHL, hashDockerignore, hashMultistage,
		hashSub1, hashSub2, hashTwoParents, hashWithAFile, hashRoot2, hashTwoParents, hashRoot3),
		nominalGraph)
	nominalLines := strings.Split(nominalGraph, "\n")

	t.Run("adding a file to the root1 directory", func(t *testing.T) {
		copiedDir := copyFixtures(t)

		baseDir := copiedDir + "/root1"

		// When I add a new file in root1
		newFilePath := baseDir + "/newfile"
		require.NoError(t, os.WriteFile(newFilePath, []byte("any content"), 0o600))

		graph, err := GenerateDAG(copiedDir, registryPrefix, "", nil)
		require.NoError(t, err)

		have := graph.Sprint(path.Base(copiedDir))
		newLines := strings.Split(have, "\n")
		assert.Len(t, newLines, len(nominalLines))

		for i := range nominalLines {
			switch i {
			case 0, 9, 11, 12:
				assert.Equal(t, nominalLines[i], newLines[i])
			default:
				assert.NotEqual(t, nominalLines[i], newLines[i])
			}
		}
	})

	t.Run("adding a file to the multistage directory", func(t *testing.T) {
		copiedDir := copyFixtures(t)

		baseDir := copiedDir + "/root1"

		// When I add a new file in root1/multistage
		newFilePath := baseDir + "/multistage/newfile"
		require.NoError(t, os.WriteFile(newFilePath, []byte("any content"), 0o600))

		graph, err := GenerateDAG(copiedDir, registryPrefix, "", nil)
		require.NoError(t, err)

		have := graph.Sprint(path.Base(copiedDir))
		newLines := strings.Split(have, "\n")
		assert.Len(t, newLines, len(nominalLines))

		for i := range nominalLines {
			switch i {
			case 0, 9, 11, 12:
				assert.Equal(t, nominalLines[i], newLines[i])
			default:
				assert.NotEqual(t, nominalLines[i], newLines[i])
			}
		}
	})

	t.Run("using custom hash list", func(t *testing.T) {
		copiedDir := copyFixtures(t)

		// Recompute hash of custom-hash-list, which is the only node that has the label 'dib.use-custom-hash-list'
		customHashListPath := "../../test/fixtures/dib/valid_wordlist.txt"
		customHashList, err := loadCustomHashList(customHashListPath)
		require.NoError(t, err)

		hashCHL, err := hashFiles(copiedDir+"/root1/custom-hash-list", []string{
			copiedDir + "/root1/custom-hash-list/Dockerfile",
		}, []string{hashRoot1}, customHashList)
		require.NoError(t, err)

		graph, err := GenerateDAG(copiedDir, registryPrefix, customHashListPath, nil)
		require.NoError(t, err)

		// Only the custom-hash-list node, which has the label 'dib.use-custom-hash-list', should change
		assert.Equal(t, fmt.Sprintf(`docker
├──┬root1 [%s]
│  ├───custom-hash-list [%s]
│  ├───dockerignore [%s]
│  ├───multistage [%s]
│  ├──┬sub1 [%s]
│  │  └───sub2 [%s]
│  ├───two-parents [%s]
│  └───with-a-file [%s]
├──┬root2 [%s]
│  └───two-parents [%s]
└───root3 [%s]
`, hashRoot1, hashCHL, hashDockerignore, hashMultistage,
			hashSub1, hashSub2, hashTwoParents, hashWithAFile, hashRoot2, hashTwoParents, hashRoot3),
			graph.Sprint(path.Base(basePath)))
	})

	t.Run("using build args", func(t *testing.T) {
		copiedDir := copyFixtures(t)

		baseDir := copiedDir + "/root1"

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

		graph, err := GenerateDAG(copiedDir, registryPrefix, "", buildArgs)
		require.NoError(t, err)

		// Only root1 node has the 'HELLO' argument, so its hash and all of its children should change
		have := graph.Sprint(path.Base(copiedDir))
		newLines := strings.Split(have, "\n")
		assert.Len(t, newLines, len(nominalLines))

		for i := range nominalLines {
			switch i {
			case 0, 9, 11, 12:
				assert.Equal(t, nominalLines[i], newLines[i])
			default:
				assert.NotEqual(t, nominalLines[i], newLines[i])
			}
		}
	})

	t.Run("duplicates image names", func(t *testing.T) {
		dupDir := "../../test/fixtures/docker-duplicates"
		_, err := GenerateDAG(dupDir, registryPrefix, "", nil)
		require.EqualError(t, err,
			fmt.Sprintf(`duplicate image name "%s/duplicate" found while reading file `+
				`"%s/root/duplicate2/Dockerfile": previous file was "%s/root/duplicate1/Dockerfile"`,
				registryPrefix, dupDir, dupDir))
	})
}

// copyFixtures copies the buildPath directory into a temporary one to be free to edit files.
func copyFixtures(t *testing.T) string {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	src := path.Join(cwd, basePath)
	dest := t.TempDir()
	cmd := exec.Command("cp", "-r", src, dest) //nolint:gosec,noctx
	require.NoError(t, cmd.Run())

	return path.Join(dest, path.Base(basePath))
}

func Test_buildGraph(t *testing.T) {
	graph, err := buildGraph(basePath, registryPrefix)
	require.NoError(t, err)
	graph.WalkInDepth(func(node *dag.Node) {
		files := node.Image.ContextFiles
		switch node.Image.ShortName {
		case "root1":
			require.Len(t, files, 11, spew.Sdump(files))
			assert.Contains(t, files, basePath+"/root1/Dockerfile")
			assert.Contains(t, files, basePath+"/root1/custom-hash-list/Dockerfile")
			assert.Contains(t, files, basePath+"/root1/dockerignore/.dockerignore")
			assert.Contains(t, files, basePath+"/root1/dockerignore/Dockerfile")
			assert.Contains(t, files, basePath+"/root1/dockerignore/ignored.txt")
			assert.Contains(t, files, basePath+"/root1/multistage/Dockerfile")
			assert.Contains(t, files, basePath+"/root1/skipbuild/Dockerfile")
			assert.Contains(t, files, basePath+"/root1/sub1/Dockerfile")
			assert.Contains(t, files, basePath+"/root1/sub1/sub2/Dockerfile")
			assert.Contains(t, files, basePath+"/root1/with-a-file/Dockerfile")
			assert.Contains(t, files, basePath+"/root1/with-a-file/included.txt")
		case "custom-hash-list":
			require.Len(t, files, 1, spew.Sdump(files))
			assert.Contains(t, files, basePath+"/root1/custom-hash-list/Dockerfile")
		case "dockerignore":
			require.Len(t, files, 1, spew.Sdump(files))
			assert.Contains(t, files, basePath+"/root1/dockerignore/Dockerfile")
		case "multistage":
			require.Len(t, files, 1, spew.Sdump(files))
			assert.Contains(t, files, basePath+"/root1/multistage/Dockerfile")
		case "sub1":
			require.Len(t, files, 2, spew.Sdump(files))
			assert.Contains(t, files, basePath+"/root1/sub1/Dockerfile")
			assert.Contains(t, files, basePath+"/root1/sub1/sub2/Dockerfile")
		case "sub2":
			require.Len(t, files, 1, spew.Sdump(files))
			assert.Contains(t, files, basePath+"/root1/sub1/sub2/Dockerfile")
		case "with-a-file":
			require.Len(t, files, 2, spew.Sdump(files))
			assert.Contains(t, files, basePath+"/root1/with-a-file/Dockerfile")
			assert.Contains(t, files, basePath+"/root1/with-a-file/included.txt")
		case "root2":
			require.Len(t, files, 2, spew.Sdump(files))
			assert.Contains(t, files, basePath+"/root2/Dockerfile")
			assert.Contains(t, files, basePath+"/root2/root3/Dockerfile")
		case "root3":
			require.Len(t, files, 1, spew.Sdump(files))
			assert.Contains(t, files, basePath+"/root2/root3/Dockerfile")
		case "two-parents":
			require.Len(t, files, 1, spew.Sdump(files))
			assert.Contains(t, files, basePath+"/two-parents/Dockerfile")
		default:
			t.Errorf("unexpected image: %s", node.Image.ShortName)
		}
	})
}

func Test_loadCustomHashList(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    []string
		expectedErr error
	}{
		{
			name:     "custom wordlist txt",
			input:    "../../test/fixtures/dib/wordlist.txt",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "custom wordlist yml",
			input:    "../../test/fixtures/dib/wordlist.yml",
			expected: []string{"e", "f", "g"},
		},
		{
			name:        "wordlist file not exist",
			input:       "../../test/fixtures/dib/lorem.txt",
			expectedErr: os.ErrNotExist,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actual, err := loadCustomHashList(test.input)
			if test.expectedErr == nil {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			} else {
				require.ErrorIs(t, err, test.expectedErr)
			}
		})
	}
}
