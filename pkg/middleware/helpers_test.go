package middleware

import "net/http"

// BufferedWriter is a writer that captures the response.
// Used to capture the template execution result.
type BufferedWriter struct {
	buf        []byte
	statusCode int
	header     http.Header
}

// NewBufferedResponseWriter creates a new buffered writer.
func NewBufferedResponseWriter() *BufferedWriter {
	return &BufferedWriter{
		buf:    make([]byte, 0, 1024),
		header: make(http.Header),
	}
}

// Write writes the data to the buffer.
func (bw *BufferedWriter) Write(p []byte) (int, error) {
	bw.buf = append(bw.buf, p...)
	return len(p), nil
}

// Header returns the header.
func (bw *BufferedWriter) Header() http.Header {
	return bw.header
}

// WriteHeader writes the status code.
func (bw *BufferedWriter) WriteHeader(statusCode int) {
	bw.statusCode = statusCode
}
