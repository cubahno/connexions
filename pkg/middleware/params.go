package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mockzilla/connexions/v2/pkg/config"
	"github.com/mockzilla/connexions/v2/pkg/db"
)

// asyncWriteTimeout is the maximum time allowed for background DB writes.
const asyncWriteTimeout = 5 * time.Second

// ResponseHeaderSource is the response header indicating where the response came from.
const ResponseHeaderSource = "X-Cxs-Source"

// ResponseHeaderSource values.
const (
	ResponseHeaderSourceUpstream  = "upstream"
	ResponseHeaderSourceCache     = "cache"
	ResponseHeaderSourceGenerated = "generated"
	ResponseHeaderSourceReplay    = "replay"
)

const serviceConfigKey ctxKey = "serviceConfig"

// Params provides access to service configuration and database for middleware.
type Params struct {
	serviceConfig *config.ServiceConfig
	storageConfig *config.StorageConfig
	database      db.DB
	log           *slog.Logger
	router        chi.Routes
}

// NewParams creates a new Params instance with the given configuration and database.
func NewParams(serviceConfig *config.ServiceConfig, storageConfig *config.StorageConfig, database db.DB) *Params {
	return &Params{
		serviceConfig: serviceConfig,
		storageConfig: storageConfig,
		database:      database,
		log:           slog.With("service", serviceConfig.Name),
	}
}

// GetServiceConfig returns the per-request service config from the context if set
// by the config override middleware, otherwise falls back to the shared config.
func (p *Params) GetServiceConfig(req *http.Request) *config.ServiceConfig {
	if cfg, ok := req.Context().Value(serviceConfigKey).(*config.ServiceConfig); ok {
		return cfg
	}
	return p.serviceConfig
}

// SetRouter stores the router for resource path resolution at request time.
func (p *Params) SetRouter(r chi.Routes) {
	p.router = r
}

// DB returns the per-service database instance.
func (p *Params) DB() db.DB {
	return p.database
}

// Logger returns a logger with the given middleware name added to the service context.
func (p *Params) Logger(middlewareName string) *slog.Logger {
	if p.log != nil {
		return p.log.With("middleware", middlewareName)
	}
	return slog.With("middleware", middlewareName)
}

// responseWriter is a custom response writer that captures the response body
type responseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	// Don't call underlying WriteHeader - we'll do it after setting our headers
}

// Write intercepts the response and writes to a buffer
func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.body.Write(b)
}
