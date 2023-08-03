package api

import (
	"net/http"
)

// healthHandler handles health routes.
type healthHandler struct {
	router *Router
}

// health creates a health check handler indicating that container is running.
func (h *healthHandler) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("OK"))
}

func CreateHealthRoutes(router *Router) error {
	handler := &healthHandler{
		router: router,
	}

	router.Get("/healthz", handler.health)

	return nil
}
