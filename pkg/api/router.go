package api

import (
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/cubahno/connexions/v2/internal/contexts"
	"github.com/cubahno/connexions/v2/internal/history"
	"github.com/cubahno/connexions/v2/pkg/config"
	"github.com/cubahno/connexions/v2/pkg/middleware"
	"github.com/cubahno/connexions/v2/resources"
	"github.com/go-chi/chi/v5"
	chiMw "github.com/go-chi/chi/v5/middleware"
)

// Router is the central HTTP router for all services
type Router struct {
	chi.Router

	config   *config.AppConfig
	contexts []map[string]map[string]any
	history  *history.CurrentRequestStorage

	mu       sync.RWMutex
	services map[string]*ServiceItem
}

// Handler defines a minimal interface for request handlers.
type Handler interface {
	Routes() RouteDescriptions
	RegisterRoutes(router chi.Router)
	Generate(w http.ResponseWriter, r *http.Request)
}

// NewRouter creates a new central router with default middleware
func NewRouter(options ...RouterOption) *Router {
	r := chi.NewRouter()

	// create default replacement context
	ctxs := contexts.Load(map[string][]byte{
		"common": resources.CommonContextYAMLContents,
		"fake":   resources.FakeContextYAMLContents,
		"words":  resources.WordsContextYAMLContents,
	}, nil)
	defaultContexts := []map[string]map[string]any{
		{"common": ctxs["common"]},
		{"fake": ctxs["fake"]},
		{"words": ctxs["words"]},
	}

	// Apply default middleware
	r.Use(chiMw.RequestID)
	r.Use(chiMw.RealIP)
	r.Use(middleware.LoggerMiddleware)
	r.Use(middleware.DurationMiddleware)
	r.Use(chiMw.Recoverer)
	r.Use(chiMw.Timeout(60 * time.Second))

	cfg := config.NewDefaultAppConfig(".")
	if err := env.Parse(cfg); err != nil {
		slog.Error("Failed to parse env", "error", err)
	}

	res := &Router{
		Router:   r,
		config:   cfg,
		contexts: defaultContexts,
		services: make(map[string]*ServiceItem),
	}

	for _, opt := range options {
		opt(res)
	}

	cfg = res.config

	historyDuration := cfg.HistoryDuration
	res.history = history.NewCurrentRequestStorage(historyDuration)

	if err := createUIFileStructure(res.config); err != nil {
		slog.Error("Failed to create ui file structure", "error", err)
	}

	return res
}

// RegisterService registers a service with the router.
// The service config must have a Name field set.
// Service middleware is applied AFTER the standard middleware chain.
// The service will be registered at the route "/{cfg.Name}".
func (r *Router) RegisterService(
	cfg *config.ServiceConfig,
	handler Handler,
	serviceMiddleware []func(*middleware.Params) func(http.Handler) http.Handler,
) {
	mwParams := &middleware.Params{
		ServiceConfig: cfg,
		History:       r.history,
	}

	// Use cfg.Name as the route prefix (ensure it starts with /)
	prefix := "/" + cfg.Name
	r.Route(prefix, func(subRouter chi.Router) {
		// Standard middleware (always applied)
		subRouter.Use(middleware.CreateLatencyAndErrorMiddleware(mwParams))
		subRouter.Use(middleware.CreateCacheReadMiddleware(mwParams))
		subRouter.Use(middleware.CreateUpstreamRequestMiddleware(mwParams))
		subRouter.Use(middleware.CreateCacheWriteMiddleware(mwParams))

		// Service-specific middleware (applied after standard middleware)
		for _, createMw := range serviceMiddleware {
			subRouter.Use(createMw(mwParams))
		}

		handler.RegisterRoutes(subRouter)
	})

	r.mu.Lock()
	defer r.mu.Unlock()

	r.services[cfg.Name] = &ServiceItem{
		Name:    cfg.Name,
		Handler: handler,
	}
}

// Config returns the app configuration
func (r *Router) Config() *config.AppConfig {
	return r.config
}

func (r *Router) GetServices() map[string]*ServiceItem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make(map[string]*ServiceItem, len(r.services))
	for k, v := range r.services {
		res[k] = v
	}

	return res
}

// GetHistory returns the history storage
func (r *Router) GetHistory() *history.CurrentRequestStorage {
	return r.history
}

// GetContexts returns the list of contexts
func (r *Router) GetContexts() []map[string]map[string]any {
	return r.contexts
}

type RouterOption func(*Router)

func WithConfigOption(cfg *config.AppConfig) RouterOption {
	return func(r *Router) {
		r.config = cfg
	}
}

// createUIFileStructure creates the necessary directories and files
func createUIFileStructure(config *config.AppConfig) error {
	if config.DisableUI {
		return nil
	}

	paths := config.Paths
	dirs := []string{paths.Resources, paths.UI}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.Mkdir(dir, os.ModePerm); err != nil {
				return err
			}
		}
	}

	return nil
}
