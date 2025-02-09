package testhelpers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
