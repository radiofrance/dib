package report

import (
	"io/fs"
	"os"
	"path"
)

// copyAssetsFiles func iterate recursively on the "client" embed filesystem and copy it inside the report folder.
func copyAssetsFiles(filesystem fs.FS, filesystemRootDir string, dibReport *Report) error {
	subFS, err := fs.Sub(filesystem, filesystemRootDir)
	if err != nil {
		return err
	}

	return fs.WalkDir(subFS, ".", func(embedFilePath string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dirEntry.IsDir() {
			return os.MkdirAll(path.Join(dibReport.GetRootDir(), embedFilePath), 0o755)
		}

		data, err := fs.ReadFile(subFS, embedFilePath)
		if err != nil {
			return err
		}

		return os.WriteFile(path.Join(dibReport.GetRootDir(), embedFilePath), data, 0o644) //nolint:gosec
	})
}
