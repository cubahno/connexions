package middleware

import (
	"bytes"
	"net/http"

	"github.com/cubahno/connexions/v2/pkg/config"
	"github.com/cubahno/connexions/v2/pkg/db"
)

// ResponseHeaderSource is the response header indicating where the response came from.
const ResponseHeaderSource = "X-Cxs-Source"

// ResponseHeaderSource values.
const (
	ResponseHeaderSourceUpstream  = "upstream"
	ResponseHeaderSourceCache     = "cache"
	ResponseHeaderSourceGenerated = "generated"
)

// Params provides access to service configuration and database for middleware.
type Params struct {
	ServiceConfig *config.ServiceConfig
	StorageConfig *config.StorageConfig
	database      db.DB
}

// NewParams creates a new Params instance with the given configuration and database.
func NewParams(serviceConfig *config.ServiceConfig, storageConfig *config.StorageConfig, database db.DB) *Params {
	return &Params{
		ServiceConfig: serviceConfig,
		StorageConfig: storageConfig,
		database:      database,
	}
}

// DB returns the per-service database instance.
func (p *Params) DB() db.DB {
	return p.database
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
