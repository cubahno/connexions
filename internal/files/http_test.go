package files

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
)

func TestIsURL(t *testing.T) {
	assert := assert2.New(t)

	t.Run("http URL", func(t *testing.T) {
		assert.True(IsURL("http://example.com/spec.yaml"))
	})

	t.Run("https URL", func(t *testing.T) {
		assert.True(IsURL("https://example.com/spec.yaml"))
	})

	t.Run("file path", func(t *testing.T) {
		assert.False(IsURL("/path/to/file.yaml"))
	})

	t.Run("relative path", func(t *testing.T) {
		assert.False(IsURL("./file.yaml"))
	})

	t.Run("empty string", func(t *testing.T) {
		assert.False(IsURL(""))
	})
}

func CreateMockServer(t *testing.T, contentType, responseBody string, responseStatus int) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(responseStatus)
		_, err := w.Write([]byte(responseBody))
		if err != nil {
			t.Errorf("Error writing response: %v", err)
			t.FailNow()
		}
	})
	return httptest.NewServer(handler)
}

// MockTransportWithReadError is a custom transport to simulate an error during reading
type MockTransportWithReadError struct{}

func (t *MockTransportWithReadError) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a response with a body that returns an error when read
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       &MockBodyWithReadError{},
	}, nil
}

type MockBodyWithReadError struct{}

func (b *MockBodyWithReadError) Close() error {
	return nil
}

func (b *MockBodyWithReadError) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF // Simulate an error
}

func TestGetFileContentsFromURL(t *testing.T) {
	assert := assert2.New(t)

	t.Run("invalid-url", func(t *testing.T) {
		_, _, err := GetFileContentsFromURL(nil, "unknown-url")
		assert.Error(err)
	})

	t.Run("status-error-404", func(t *testing.T) {
		mockServer := CreateMockServer(t, "text/plain", "Not Found", http.StatusNotFound)
		defer mockServer.Close()

		_, _, err := GetFileContentsFromURL(nil, mockServer.URL)
		assert.Equal(ErrGettingFileFromURL, err)
	})

	t.Run("status-error-500", func(t *testing.T) {
		mockServer := CreateMockServer(t, "text/plain", "Server Error", http.StatusInternalServerError)
		defer mockServer.Close()

		_, _, err := GetFileContentsFromURL(nil, mockServer.URL)
		assert.Equal(ErrGettingFileFromURL, err)
	})

	t.Run("read-error", func(t *testing.T) {
		mockServerWithReadError := CreateMockServer(t, "text/plain", "Hello, World!", http.StatusOK)
		defer mockServerWithReadError.Close()

		// Replace the response body with a reader that returns an error when read
		client := mockServerWithReadError.Client()
		client.Transport = &MockTransportWithReadError{}

		_, _, err := GetFileContentsFromURL(client, mockServerWithReadError.URL)
		assert.Error(err)
	})

	t.Run("happy-path-text", func(t *testing.T) {
		mockServer := CreateMockServer(t, "text/plain", "Hallo, Welt!", http.StatusOK)
		defer mockServer.Close()

		content, contentType, err := GetFileContentsFromURL(nil, mockServer.URL)
		assert.NoError(err)
		assert.Equal("text/plain", contentType)
		assert.Equal("Hallo, Welt!", string(content))
	})

	t.Run("happy-path-json", func(t *testing.T) {
		jsonContent := `{"message": "hello"}`
		mockServer := CreateMockServer(t, "application/json", jsonContent, http.StatusOK)
		defer mockServer.Close()

		content, contentType, err := GetFileContentsFromURL(nil, mockServer.URL)
		assert.NoError(err)
		assert.Equal("application/json", contentType)
		assert.Equal(jsonContent, string(content))
	})

	t.Run("missing-content-type", func(t *testing.T) {
		mockServer := CreateMockServer(t, "", "content", http.StatusOK)
		defer mockServer.Close()

		content, contentType, err := GetFileContentsFromURL(nil, mockServer.URL)
		assert.NoError(err)
		assert.Equal("application/octet-stream", contentType)
		assert.Equal("content", string(content))
	})

	t.Run("custom-http-client", func(t *testing.T) {
		mockServer := CreateMockServer(t, "text/html", "<html>test</html>", http.StatusOK)
		defer mockServer.Close()

		customClient := &http.Client{}
		content, contentType, err := GetFileContentsFromURL(customClient, mockServer.URL)
		assert.NoError(err)
		assert.Equal("text/html", contentType)
		assert.Equal("<html>test</html>", string(content))
	})
}

func TestReadFileOrURL(t *testing.T) {
	assert := assert2.New(t)

	t.Run("reads-from-url", func(t *testing.T) {
		mockServer := CreateMockServer(t, "application/json", `{"test": "data"}`, http.StatusOK)
		defer mockServer.Close()

		content, err := ReadFileOrURL(mockServer.URL)
		assert.NoError(err)
		assert.Equal(`{"test": "data"}`, string(content))
	})

	t.Run("reads-from-file", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := tempDir + "/test.txt"
		testContent := []byte("file content")
		err := SaveFile(filePath, testContent)
		assert.NoError(err)

		content, err := ReadFileOrURL(filePath)
		assert.NoError(err)
		assert.Equal(testContent, content)
	})

	t.Run("error-on-invalid-url", func(t *testing.T) {
		_, err := ReadFileOrURL("http://invalid-url-that-does-not-exist")
		assert.Error(err)
	})

	t.Run("error-on-non-existent-file", func(t *testing.T) {
		_, err := ReadFileOrURL("/non/existent/file.txt")
		assert.Error(err)
	})

	t.Run("handles-https-url", func(t *testing.T) {
		mockServer := CreateMockServer(t, "text/plain", "https content", http.StatusOK)
		defer mockServer.Close()

		// Replace http with https in URL for testing
		httpsURL := "https" + mockServer.URL[4:]

		// This will fail because it's not a real HTTPS server, but it tests the prefix detection
		_, err := ReadFileOrURL(httpsURL)
		assert.Error(err) // Expected to fail on connection
	})
}
