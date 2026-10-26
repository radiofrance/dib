package dib

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/radiofrance/dib/pkg/dag"
)

// lfsPointerMaxSize is the maximum size of a Git LFS pointer file, as defined by the spec:
// https://github.com/git-lfs/git-lfs/blob/main/docs/spec.md
const lfsPointerMaxSize = 1024

const lfsPointerVersionLine = "version https://git-lfs.github.com/spec/v1\n"

// CheckNoLFSPointers returns an error when a file in the build context of an image is a Git LFS pointer,
// e.g. in a clone made without git-lfs. Such a file has the same digest as the real file (see fileDigest),
// so building from it would push an image holding the pointer under the hash of the real image.
func CheckNoLFSPointers(graph *dag.DAG) error {
	return graph.WalkErr(func(node *dag.Node) error {
		for _, filename := range node.Image.ContextFiles {
			_, isPointer, err := lfsPointerOID(filename)
			if err != nil {
				return err
			}

			if isPointer {
				return fmt.Errorf("file %q of image %q is a Git LFS pointer, run `git lfs pull` before building",
					filename, node.Image.Name)
			}
		}

		return nil
	})
}

// fileDigest returns the sha256 of the contents of a file.
// When the file is a Git LFS pointer (e.g. in a clone made without git-lfs), it returns
// the oid recorded in the pointer instead, which is the sha256 of the real file contents.
// This way, the hash of an image is the same whether or not the LFS objects were fetched.
func fileDigest(filename string) ([]byte, error) {
	oid, isPointer, err := lfsPointerOID(filename)
	if err != nil {
		return nil, err
	}

	if isPointer {
		return oid, nil
	}

	file, err := os.Open(filename) //nolint:gosec
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = file.Close()
	}()

	hash := sha256.New()

	_, err = io.Copy(hash, file)
	if err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

// lfsPointerOID returns the oid of a file when it is a Git LFS pointer, and false otherwise.
func lfsPointerOID(filename string) ([]byte, bool, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return nil, false, err
	}

	if info.Size() > lfsPointerMaxSize {
		return nil, false, nil
	}

	content, err := os.ReadFile(filename) //nolint:gosec
	if err != nil {
		return nil, false, err
	}

	oid, isPointer := parseLFSPointer(content)

	return oid, isPointer, nil
}

// parseLFSPointer returns the sha256 oid of a Git LFS pointer, decoded from hex,
// and false when content is not a valid Git LFS pointer.
func parseLFSPointer(content []byte) ([]byte, bool) {
	lines, found := bytes.CutPrefix(content, []byte(lfsPointerVersionLine))
	if !found {
		return nil, false
	}

	for line := range bytes.SplitSeq(lines, []byte("\n")) {
		value, found := bytes.CutPrefix(line, []byte("oid sha256:"))
		if !found {
			continue
		}

		oid, err := hex.DecodeString(string(value))
		if err != nil || len(oid) != sha256.Size {
			return nil, false
		}

		return oid, true
	}

	return nil, false
}
