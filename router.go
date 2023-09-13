package connexions

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"net/http"
	"sync"
	"time"
)

type RouteRegister func(router *Router) error

type Router struct {
	*chi.Mux
	Services     map[string]*ServiceItem
	Config       *Config
	Contexts     map[string]map[string]any
	ContextNames []map[string]string
	mu           sync.Mutex
}

type ErrorMessage struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error   string          `json:"error"`
	Details []*ErrorMessage `json:"details"`
}

func NewRouter(config *Config) *Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	return &Router{
		Mux:    r,
		Config: config,
	}
}

func GetJSONPayload[T any](req *http.Request) (*T, error) {
	var payload T
	// if len(req.Body) == 0 {
	// 	return &payload, nil
	// }
	dec := json.NewDecoder(req.Body)
	err := dec.Decode(&payload)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &payload, nil
}

func NewErrorMessage(err error) *ErrorMessage {
	return &ErrorMessage{
		Message: err.Error(),
	}
}

func (r *Router) RemoveContext(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.Contexts, name)
}
