package files

import (
	"os"
	"path/filepath"
)

// SaveFile saves a file to the specified path.
// If the destination directory doesn't exist, it will be created.
func SaveFile(filePath string, data []byte) error {
	dirPath := filepath.Dir(filePath)
	// Create directories recursively
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return err
	}

	dest, err := os.Create(filePath)
	if err != nil {
		return err
	}

	_, err = dest.Write(data)

	return err
}
