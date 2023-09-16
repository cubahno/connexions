package connexions

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

func createContextRoutes(router *Router) error {
	if router.Config.App.DisableUI || router.Config.App.ContextURL == "" {
		return nil
	}

	handler := &ContextHandler{
		router: router,
	}

	url := router.Config.App.ContextURL
	url = "/" + strings.Trim(url, "/")
	log.Printf("Mounting context URLs at %s\n", url)

	router.Route(url, func(r chi.Router) {
		r.Get("/", handler.list)
		r.Put("/", handler.save)
		r.Get("/{name}", handler.details)
		r.Delete("/{name}", handler.delete)
	})

	return nil
}

type ContextListResponse struct {
	Items []string `json:"items"`
}

type ContextHandler struct {
	*BaseHandler
	router *Router
	mu     sync.Mutex
}

func (h *ContextHandler) list(w http.ResponseWriter, r *http.Request) {
	var names []string
	for name, _ := range h.router.Contexts {
		names = append(names, name)
	}

	sort.SliceStable(names, func(i, j int) bool {
		return names[i] < names[j]
	})

	h.JSONResponse(w).Send(ContextListResponse{
		Items: names,
	})
}

func (h *ContextHandler) details(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, found := h.router.Contexts[name]; !found {
		h.JSONResponse(w).WithStatusCode(http.StatusNotFound).Send(&SimpleResponse{Message: "Context not found"})
		return
	}

	ctxDir := h.router.Config.App.Paths.Contexts
	http.ServeFile(w, r, filepath.Join(ctxDir, fmt.Sprintf("%s.yml", name)))
}

func (h *ContextHandler) delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, found := h.router.Contexts[name]; !found {
		h.error(http.StatusNotFound, "Context not found", w)
		return
	}
	ctxDir := h.router.Config.App.Paths.Contexts
	filePath := filepath.Join(ctxDir, fmt.Sprintf("%s.yml", name))
	if err := os.Remove(filePath); err != nil {
		h.error(http.StatusInternalServerError, err.Error(), w)
		return
	}

	h.router.RemoveContext(name)
	h.success("Context deleted!", w)
}

func (h *ContextHandler) save(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	name := r.FormValue("name")
	if name == "" {
		h.error(http.StatusBadRequest, "Name is required", w)
		return
	}

	match, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", name)
	if !match || len(name) > 20 {
		h.error(http.StatusBadRequest, "Invalid name: must be alpha-numeric, _, - and not exceed 20 chars", w)
		return
	}

	ctxDir := h.router.Config.App.Paths.Contexts
	filePath := filepath.Join(ctxDir, fmt.Sprintf("%s.yml", name))
	content := r.FormValue("content")

	// ignore result as we need to reload them all because of the possible cross-references in aliases
	_, err := ParseContextFromBytes([]byte(content))
	if err != nil {
		h.error(http.StatusBadRequest, "Invalid context: "+err.Error(), w)
		return
	}

	if err = SaveFile(filePath, []byte(content)); err != nil {
		h.error(http.StatusInternalServerError, err.Error(), w)
		return
	}

	// there's no error
	_ = loadContexts(h.router)

	h.success("Context saved", w)
}
