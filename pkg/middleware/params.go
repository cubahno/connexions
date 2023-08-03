package middleware

import (
	"bytes"
	"net/http"

	"github.com/cubahno/connexions/v2/internal/history"
	"github.com/cubahno/connexions/v2/pkg/config"
)

type Params struct {
	ServiceConfig *config.ServiceConfig
	History       *history.CurrentRequestStorage
}

// responseWriter is a custom response writer that captures the response body
type responseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write intercepts the response and writes to a buffer
func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.body.Write(b)
}
