package xs

import (
	"net/http"
)

type BaseHandler struct {
}

func (h *BaseHandler) success(message string, w http.ResponseWriter) {
	NewJSONResponse(http.StatusOK, map[string]any{"success": true, "message": message}, w)
}

func (h *BaseHandler) error(code int, message string, w http.ResponseWriter) {
	NewJSONResponse(code, map[string]any{"success": false, "message": message}, w)
}
