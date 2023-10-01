package connexions

import (
	"bytes"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"strings"
	"sync"
)

// SettingsHandler handles settings routes.
type SettingsHandler struct {
	*BaseHandler
	router *Router
	mu     sync.Mutex
}

// createSettingsRoutes creates routes for settings.
// Implements RouteRegister interface.
func createSettingsRoutes(router *Router) error {
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

// get returns the current settings.
func (h *SettingsHandler) get(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var b bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2) // this is what you're looking for
	_ = yamlEncoder.Encode(h.router.Config)

	// data, _ := yaml.Marshal(h.router.Config)
	data := b.Bytes()

	h.Response(w).WithHeader("Content-Type", "application/x-yaml").Send(data)
}

func (h *SettingsHandler) put(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	payload, _ := io.ReadAll(r.Body)

	_, err := NewConfigFromContent(payload)
	if err != nil {
		h.Error(http.StatusBadRequest, err.Error(), w)
		return
	}

	if err = SaveFile(h.router.Config.App.Paths.ConfigFile, payload); err != nil {
		h.Error(http.StatusInternalServerError, err.Error(), w)
		return
	}

	h.router.Config.Reload()
	h.Success("Settings saved and reloaded!", w)
}

// Restore settings saving them in config.yml
func (h *SettingsHandler) post(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	dest := h.router.Config.App.Paths.ConfigFile
	defaultCfg := NewDefaultConfig(h.router.Config.baseDir)
	defaultBts, _ := yaml.Marshal(defaultCfg)

	if err := SaveFile(dest, defaultBts); err != nil {
		h.Error(http.StatusInternalServerError, "Failed to restore config contents", w)
		return
	}

	h.router.Config.Reload()
	h.Success("Settings restored and reloaded!", w)
}
