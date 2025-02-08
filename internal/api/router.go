package api

import (
	"encoding/json"
	"io"
	"net/http"
	"plugin"
	"sort"
	"sync"
	"time"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/types"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type RouteRegister func(router *Router) error

// Router is a wrapper around chi.Mux that adds some extra functionality.
//
// Config is a pointer to the global Config instance.
// services: Router keeps track of registered services and their routes.
// contexts is a map of registered context namespaces.
// Each namespace is a map of context names and their values.
//
// defaultContexts is a slice of registered context namespaces.
// It can refer to complete context namespace or just a part of it:
// e.g. in yaml config
// - common:
// - fake:payments
type Router struct {
	*chi.Mux

	Config          *config.Config
	callbacksPlugin *plugin.Plugin
	services        map[string]*ServiceItem
	contexts        map[string]map[string]any
	defaultContexts []map[string]string
	history         *CurrentRequestStorage

	mu sync.RWMutex
}

// NewRouter creates a new Router instance from Config.
func NewRouter(config *config.Config) *Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(ConditionalLoggingMiddleware(config))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	return &Router{
		Mux:             r,
		Config:          config,
		services:        make(map[string]*ServiceItem),
		contexts:        make(map[string]map[string]any),
		defaultContexts: make([]map[string]string, 0),
		history:         NewCurrentRequestStorage(5 * time.Minute),
	}
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

func (r *Router) SetServices(services map[string]*ServiceItem) *Router {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services = services
	return r
}

func (r *Router) AddService(item *ServiceItem) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.services) == 0 {
		r.services = make(map[string]*ServiceItem)
	}

	r.services[item.Name] = item
}

func (r *Router) RemoveService(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.services, name)
}

func (r *Router) SetContexts(contexts map[string]map[string]any, defaultContexts []map[string]string) *Router {
	r.mu.Lock()
	defer r.mu.Unlock()

	// sort default contexts by name
	sort.Slice(defaultContexts, func(i, j int) bool {
		// Extract the first keys from the maps
		var keyI, keyJ string
		for key := range defaultContexts[i] {
			keyI = key
			break
		}
		for key := range defaultContexts[j] {
			keyJ = key
			break
		}

		// Compare the first keys
		return keyI < keyJ
	})

	r.contexts = contexts
	r.defaultContexts = defaultContexts

	return r
}

func (r *Router) GetContexts() map[string]map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return types.CopyNestedMap(r.contexts)
}

func (r *Router) GetDefaultContexts() []map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var res = make([]map[string]string, len(r.defaultContexts))

	for i, m := range r.defaultContexts {
		res[i] = make(map[string]string)
		for k, v := range m {
			res[i][k] = v
		}
	}
	return res
}

// RemoveContext removes registered context namespace from the router.
// Removing it from the service configurations seems not needed at the moment as
// it won't affect any resolving.
func (r *Router) RemoveContext(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.contexts, name)
}

// GetJSONPayload parses JSON payload from the request body into the given type.
func GetJSONPayload[T any](req *http.Request) (*T, error) {
	var payload T
	dec := json.NewDecoder(req.Body)
	err := dec.Decode(&payload)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &payload, nil
}
