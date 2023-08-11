package api

import (
	"fmt"
	"github.com/cubahno/xs"
	"io"
	"net/http"
	"os"
)

type SettingsHandler struct {
	*BaseHandler
}

func CreateSettingsRoutes(router *Router) error {
	handler := &SettingsHandler{}

	router.Get("/settings", handler.get)
	router.Put("/settings", handler.put)
	router.Post("/settings", handler.post)

	return nil
}

func (h *SettingsHandler) get(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, fmt.Sprintf("%s/config.yml", xs.ResourcePath))
}

func (h *SettingsHandler) put(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		h.error(http.StatusBadRequest, err.Error(), w)
		return
	}

	_, err = xs.NewConfigFromContent(payload)
	if err != nil {
		h.error(http.StatusBadRequest, err.Error(), w)
		return
	}

	filePath := fmt.Sprintf("%s/config.yml", xs.ResourcePath)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		h.error(http.StatusInternalServerError, err.Error(), w)
		return
	}
	defer file.Close()

	_, err = file.Write(payload)
	if err != nil {
		h.error(http.StatusInternalServerError, err.Error(), w)
		return
	}

	h.success("Settings saved and reloaded!", w)
}

// Restore settings from config.yml.dist
func (h *SettingsHandler) post(w http.ResponseWriter, r *http.Request) {
	src := fmt.Sprintf("%s/config.yml.dist", xs.ResourcePath)
	dest := fmt.Sprintf("%s/config.yml", xs.ResourcePath)

	if err := CopyFile(src, dest); err != nil {
		h.error(http.StatusInternalServerError, "Failed to copy file contents", w)
		return
	}
	h.success("Settings restored and reloaded!", w)
}
