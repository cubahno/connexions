package xs

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
	"sync"
)

type RouteRegister func(router *Router) error

type Router struct {
	*chi.Mux
	Services map[string]*ServiceItem
	Config   *Config
	Contexts map[string]*ReplacementContext
	mu       sync.Mutex
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

func (r *Router) AddContext(name string, ctx *ReplacementContext) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Contexts == nil {
		r.Contexts = map[string]*ReplacementContext{}
	}

	r.Contexts[name] = ctx
}
