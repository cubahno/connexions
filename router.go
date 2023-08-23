package connexions

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
	"path/filepath"
	"sync"
)

type RouteRegister func(router *Router) error

type Router struct {
	*chi.Mux
	Services     map[string]*ServiceItem
	Config       *Config
	Contexts     map[string]map[string]any
	ContextNames []map[string]string
	Paths        *Paths
	mu           sync.Mutex
}

type ErrorMessage struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error   string          `json:"error"`
	Details []*ErrorMessage `json:"details"`
}

func GetPayload[T any](req *http.Request) (*T, error) {
	var payload T
	err := json.NewDecoder(req.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}
	return &payload, nil
}

func GetErrorResponse(err error) *ErrorMessage {
	return &ErrorMessage{
		Message: err.Error(),
	}
}

func (r *Router) RemoveContext(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.Contexts, name)
}

func (r *Router) GetResourcesPath() string {
	return r.Paths.Resources
}

func (r *Router) GetConfigPath() string {
	return r.Paths.ConfigFile
}

func (r *Router) GetContextsPath() string {
	return r.Paths.Contexts
}

func (r *Router) GetUIPath() string {
	return filepath.Join(r.GetResourcesPath(), "ui")
}
