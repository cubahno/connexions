package xs

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func CreateContextRoutes(router *Router) error {
	if !router.Config.App.ServeUI || router.Config.App.ContextURL == "" {
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

	res := &ContextListResponse{
		Items: names,
	}

	NewJSONResponse(http.StatusOK, res, w)
}

func (h *ContextHandler) details(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, found := h.router.Contexts[name]; !found {
		NewJSONResponse(http.StatusNotFound, "Context not found", w)
		return
	}

	http.ServeFile(w, r, filepath.Join(ContextPath, fmt.Sprintf("%s.yml", name)))
}

func (h *ContextHandler) delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, found := h.router.Contexts[name]; !found {
		h.error(http.StatusNotFound, "Context not found", w)
		return
	}
	filePath := filepath.Join(ContextPath, fmt.Sprintf("%s.yml", name))
	if err := os.Remove(filePath); err != nil {
		h.error(http.StatusInternalServerError, err.Error(), w)
		return
	}

	h.router.RemoveContext(name)
	h.success("Context deleted", w)
}

func (h *ContextHandler) save(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 * 1024 * 1024)
	if err != nil {
		h.error(400, err.Error(), w)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println("Request Body:", string(body))

	name := r.FormValue("name")
	if name == "" {
		h.error(400, "Name is required", w)
		return
	}

	filePath := filepath.Join(ContextPath, fmt.Sprintf("%s.yml", name))
	content := r.FormValue("content")

	ctx, err := ParseContextFromBytes([]byte(content))
	if err != nil {
		h.error(400, "Invalid context: "+err.Error(), w)
		return
	}

	if err := SaveFile(filePath, []byte(content)); err != nil {
		h.error(500, err.Error(), w)
		return
	}

	h.router.AddContext(name, ctx)
	h.success("Context saved", w)
}
