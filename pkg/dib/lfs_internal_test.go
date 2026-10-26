package dib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/radiofrance/dib/pkg/dag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseLFSPointer(t *testing.T) {
	t.Parallel()

	oid := "4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393"
	wantOID := decodeHex(t, oid)

	testCases := []struct {
		name    string
		content string
		wantOK  bool
	}{
		{
			name:    "valid pointer",
			content: "version https://git-lfs.github.com/spec/v1\noid sha256:" + oid + "\nsize 12345\n",
			wantOK:  true,
		},
		{
			name: "valid pointer with an extension",
			content: "version https://git-lfs.github.com/spec/v1\next-0-foo sha256:" + oid +
				"\noid sha256:" + oid + "\nsize 12345\n",
			wantOK: true,
		},
		{
			name:    "regular text file",
			content: "FROM scratch\nLABEL name=\"foo\"\n",
		},
		{
			name:    "empty file",
			content: "",
		},
		{
			name:    "version line not at the start",
			content: "# comment\nversion https://git-lfs.github.com/spec/v1\noid sha256:" + oid + "\nsize 1\n",
		},
		{
			name:    "missing oid",
			content: "version https://git-lfs.github.com/spec/v1\nsize 12345\n",
		},
		{
			name:    "oid with an unsupported hash method",
			content: "version https://git-lfs.github.com/spec/v1\noid sha1:da39a3ee5e6b4b0d3255bfef95601890afd80709\nsize 1\n",
		},
		{
			name:    "oid that is not 64 hex characters",
			content: "version https://git-lfs.github.com/spec/v1\noid sha256:not-hex\nsize 1\n",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseLFSPointer([]byte(test.content))
			assert.Equal(t, test.wantOK, ok)

			if test.wantOK {
				assert.Equal(t, wantOID, got)
			}
		})
	}
}

// Test_fileDigest_lfsPointer checks that a file and its Git LFS pointer have the same digest.
func Test_fileDigest_lfsPointer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	content := []byte("the real contents of a large file")
	sum := sha256.Sum256(content)

	realFile := filepath.Join(dir, "real")
	require.NoError(t, os.WriteFile(realFile, content, 0o600))

	pointerFile := filepath.Join(dir, "pointer")
	pointer := fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%x\nsize %d\n", sum, len(content))
	require.NoError(t, os.WriteFile(pointerFile, []byte(pointer), 0o600))

	realDigest, err := fileDigest(realFile)
	require.NoError(t, err)
	assert.Equal(t, sum[:], realDigest)

	pointerDigest, err := fileDigest(pointerFile)
	require.NoError(t, err)
	assert.Equal(t, realDigest, pointerDigest)
}

func TestCheckNoLFSPointers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	regularFile := filepath.Join(dir, "regular")
	require.NoError(t, os.WriteFile(regularFile, []byte("FROM scratch\n"), 0o600))

	pointerFile := filepath.Join(dir, "pointer")
	pointer := "version https://git-lfs.github.com/spec/v1\n" +
		"oid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393\nsize 12345\n"
	require.NoError(t, os.WriteFile(pointerFile, []byte(pointer), 0o600))

	newGraph := func(contextFiles ...string) *dag.DAG {
		graph := &dag.DAG{}
		graph.AddNode(dag.NewNode(&dag.Image{Name: "eu.gcr.io/my-test-repository/foo", ContextFiles: contextFiles}))

		return graph
	}

	require.NoError(t, CheckNoLFSPointers(newGraph(regularFile)))

	err := CheckNoLFSPointers(newGraph(regularFile, pointerFile))
	require.ErrorContains(t, err, "is a Git LFS pointer")
}

func decodeHex(t *testing.T, s string) []byte {
	t.Helper()

	b, err := hex.DecodeString(s)
	require.NoError(t, err)

	return b
}
