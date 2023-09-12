package connexions

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

type SettingsHandler struct {
	*BaseHandler
	router *Router
	mu     sync.Mutex
}

func CreateSettingsRoutes(router *Router) error {
	if router.Config.App.DisableUI || router.Config.App.SettingsURL == "" {
		return nil
	}

	handler := &SettingsHandler{
		router: router,
	}

	url := router.Config.App.SettingsURL
	url = "/" + strings.Trim(url, "/")

	router.Route(url, func(r chi.Router) {
		r.Get("/", handler.get)
		r.Put("/", handler.put)
		r.Post("/", handler.post)
	})

	return nil
}

func (h *SettingsHandler) get(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, h.router.Config.App.Paths.ConfigFile)
}

func (h *SettingsHandler) put(w http.ResponseWriter, r *http.Request) {
	payload, _ := io.ReadAll(r.Body)

	_, err := NewConfigFromContent(payload)
	if err != nil {
		h.error(http.StatusBadRequest, err.Error(), w)
		return
	}

	if err = os.WriteFile(h.router.Config.App.Paths.ConfigFile, payload, 0644); err != nil {
		h.error(http.StatusInternalServerError, err.Error(), w)
		return
	}

	h.router.Config.Reload()
	h.success("Settings saved and reloaded!", w)
}

// Restore settings from config.yml.dist
func (h *SettingsHandler) post(w http.ResponseWriter, r *http.Request) {
	dest := h.router.Config.App.Paths.ConfigFile
	src := fmt.Sprintf("%s.dist", dest)

	if err := CopyFile(src, dest); err != nil {
		h.error(http.StatusInternalServerError, "Failed to copy file contents", w)
		return
	}

	h.router.Config.Reload()
	h.success("Settings restored and reloaded!", w)
}
