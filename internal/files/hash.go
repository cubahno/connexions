package files

import (
	"crypto/sha256"
	"fmt"
	"io"
)

// GetFileHash gets the SHA256 hash of a file.
func GetFileHash(file io.Reader) string {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}
