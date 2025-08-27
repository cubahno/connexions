package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/cubahno/connexions/internal/context"
	"github.com/cubahno/connexions/internal/files"
	"github.com/go-chi/chi/v5"
)

// ContextHandler handles context routes.
type ContextHandler struct {
	*BaseHandler
	router *Router
	mu     sync.Mutex
}

// createHomeRoutes creates routes to handle contexts.
// Implements RouteRegister interface.
func createContextRoutes(router *Router) error {
	if router.Config.App.DisableUI || router.Config.App.ContextURL == "" {
		return nil
	}

	handler := &ContextHandler{
		router: router,
	}

	url := router.Config.App.ContextURL
	url = "/" + strings.Trim(url, "/")
	slog.Info(fmt.Sprintf("Mounting context URLs at %s", url))

	router.Route(url, func(r chi.Router) {
		r.Get("/", handler.list)
		r.Put("/", handler.save)
		r.Get("/{name}", handler.details)
		r.Delete("/{name}", handler.delete)
	})

	return nil
}

// list returns a list of context namespaces.
func (h *ContextHandler) list(w http.ResponseWriter, r *http.Request) {
	var names []string
	for name := range h.router.GetContexts() {
		names = append(names, name)
	}

	sort.SliceStable(names, func(i, j int) bool {
		return names[i] < names[j]
	})

	h.JSONResponse(w).Send(ContextListResponse{
		Items: names,
	})
}

// details returns a context details map.
func (h *ContextHandler) details(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, found := h.router.GetContexts()[name]; !found {
		h.JSONResponse(w).WithStatusCode(http.StatusNotFound).Send(&SimpleResponse{Message: "Context not found"})
		return
	}

	ctxDir := h.router.Config.App.Paths.Contexts
	http.ServeFile(w, r, filepath.Join(ctxDir, fmt.Sprintf("%s.yml", name)))
}

// delete deletes a context including file.
func (h *ContextHandler) delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, found := h.router.GetContexts()[name]; !found {
		h.Error(http.StatusNotFound, "Context not found", w)
		return
	}
	ctxDir := h.router.Config.App.Paths.Contexts
	filePath := filepath.Join(ctxDir, fmt.Sprintf("%s.yml", name))
	_ = os.Remove(filePath)

	h.router.RemoveContext(name)
	h.Success("Context deleted!", w)
}

// save saves a context to file.
// Existing context will be overwritten.
func (h *ContextHandler) save(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	name := r.FormValue("name")
	if name == "" {
		h.Error(http.StatusBadRequest, "Name is required", w)
		return
	}

	match, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", name)
	if !match || len(name) > 20 {
		h.Error(http.StatusBadRequest, "Invalid name: must be alpha-numeric, _, - and not exceed 20 chars", w)
		return
	}

	ctxDir := h.router.Config.App.Paths.Contexts
	filePath := filepath.Join(ctxDir, fmt.Sprintf("%s.yml", name))
	content := r.FormValue("content")

	// ignore result as we need to reload them all because of the possible cross-references in aliases
	_, err := context.ParseContextFromBytes([]byte(content), context.Fakes)
	if err != nil {
		h.Error(http.StatusBadRequest, "Invalid context: "+err.Error(), w)
		return
	}

	if err = files.SaveFile(filePath, []byte(content)); err != nil {
		h.Error(http.StatusInternalServerError, err.Error(), w)
		return
	}

	// there's no error
	_ = loadContexts(h.router)

	h.Success("Context saved", w)
}

// ContextListResponse is a response for context list.
type ContextListResponse struct {
	Items []string `json:"items"`
}
