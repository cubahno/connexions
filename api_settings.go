package connexions

import (
	"github.com/go-chi/chi/v5"
	"github.com/invopop/yaml"
	"io"
	"net/http"
	"strings"
	"sync"
)

type SettingsHandler struct {
	*BaseHandler
	router *Router
	mu     sync.Mutex
}

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

func (h *SettingsHandler) get(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, _ := yaml.Marshal(h.router.Config)

	h.Response(w).WithHeader("Content-Type", "application/x-yaml").Send(data)
}

func (h *SettingsHandler) put(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	payload, _ := io.ReadAll(r.Body)

	_, err := NewConfigFromContent(payload)
	if err != nil {
		h.error(http.StatusBadRequest, err.Error(), w)
		return
	}

	if err = SaveFile(h.router.Config.App.Paths.ConfigFile, payload); err != nil {
		h.error(http.StatusInternalServerError, err.Error(), w)
		return
	}

	h.router.Config.Reload()
	h.success("Settings saved and reloaded!", w)
}

// Restore settings saving them in config.yml
func (h *SettingsHandler) post(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	dest := h.router.Config.App.Paths.ConfigFile
	defaultCfg := NewDefaultConfig(h.router.Config.baseDir)
	defaultBts, _ := yaml.Marshal(defaultCfg)

	if err := SaveFile(dest, defaultBts); err != nil {
		h.error(http.StatusInternalServerError, "Failed to restore config contents", w)
		return
	}

	h.router.Config.Reload()
	h.success("Settings restored and reloaded!", w)
}
