package xs

import (
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"strings"
	"sync"
)

func CreateContextRoutes(router *Router) error {
	if !router.Config.App.ServeUI || router.Config.App.ContextURL == "" {
		return nil
	}

	handler := &ServiceHandler{
		router: router,
	}

	url := router.Config.App.ContextURL
	url = "/" + strings.Trim(url, "/")
	log.Printf("Mounting context URLs at %s\n", url)

	router.Route(url, func(r chi.Router) {
		r.Get("/", handler.list)
	})

	return nil
}

type ContextHandler struct {
	*BaseHandler
	router *Router
	mu     sync.Mutex
}

func (h *ContextHandler) list(w http.ResponseWriter, r *http.Request) {
	// contexts := h.router.Contexts()
	// NewJSONResponse(http.StatusOK, contexts, w)
}
