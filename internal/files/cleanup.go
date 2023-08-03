package files

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// CleanupServiceFileStructure removes empty directories from the service directory.
func CleanupServiceFileStructure(servicePath string) error {
	slog.Info(fmt.Sprintf("Cleaning up service file structure %s...", servicePath))
	return filepath.WalkDir(servicePath, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			// Skip errors for paths that were already removed
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if info == nil || !info.IsDir() {
			return nil
		}

		if IsEmptyDir(path) {
			_ = os.Remove(path)
			slog.Info(fmt.Sprintf("Removed empty directory: %s", path))
			return nil
		}

		return nil
	})
}
