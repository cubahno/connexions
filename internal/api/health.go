package api

import (
    "net/http"
    "sync"
)

// HealthHandler handles health routes.
type HealthHandler struct {
    *BaseHandler
    router *Router
    mu     sync.Mutex
}

// health creates a health check handler indicating that container is running.
func (h *HealthHandler) health(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("OK"))
}

// ready creates a ready check handler indicating that container is ready to serve traffic.
func (h *HealthHandler) ready(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("OK"))
}

func createHealthRoutes(router *Router) error {
    handler := &HealthHandler{
        router: router,
    }

    // TODO: disallow service create by these names
    router.Get("/healthz", handler.health)
    router.Get("/readyz", handler.ready)

    return nil
}
