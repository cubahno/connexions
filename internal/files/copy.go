package files

import (
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a file from srcPath to destPath.
// If the destination directory doesn't exist, it will be created.
// File permissions are preserved from the source file.
func CopyFile(srcPath, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	// Get source file info to preserve permissions
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	// Ensure the destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Create destination file with same permissions as source
	destFile, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// CopyDirectory copies a directory recursively.
func CopyDirectory(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return os.MkdirAll(dest, os.ModePerm)
		}

		// compose dest file path, we never get an error here
		relPath, _ := filepath.Rel(src, path)
		destPath := filepath.Join(dest, relPath)
		return CopyFile(path, destPath)
	})
}
