package files

import (
	"io"
	"net/http"
	"os"
	"strings"
)

// IsURL checks if a path is a URL (starts with http:// or https://).
func IsURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// GetFileContentsFromURL fetches file contents from a URL.
func GetFileContentsFromURL(client *http.Client, url string) ([]byte, string, error) {
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, "", ErrGettingFileFromURL
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return content, contentType, nil
}

// ReadFileOrURL reads content from either a local file path or a URL.
// If the path starts with http:// or https://, it fetches from the URL.
// Otherwise, it reads from the local file system.
// Returns the file contents and an error if any.
func ReadFileOrURL(path string) ([]byte, error) {
	if IsURL(path) {
		content, _, err := GetFileContentsFromURL(nil, path)
		return content, err
	}

	return os.ReadFile(path)
}
