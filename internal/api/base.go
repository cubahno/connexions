package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/cubahno/connexions/internal/config"
)

// BaseHandler is a base handler type to be embedded in other handlers.
type BaseHandler struct {
}

func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}

// Response is a response type for API responses.
func (h *BaseHandler) Response(w http.ResponseWriter) *APIResponse {
	return NewAPIResponse(w)
}

// JSONResponse is a response type for JSON responses.
func (h *BaseHandler) JSONResponse(w http.ResponseWriter) *JSONResponse {
	return NewJSONResponse(w)
}

// Success sends a Success response.
func (h *BaseHandler) Success(message string, w http.ResponseWriter) {
	h.JSONResponse(w).Send(&SimpleResponse{
		Message: message,
		Success: true,
	})
}

// Error sends an error response.
func (h *BaseHandler) Error(code int, message string, w http.ResponseWriter) {
	h.JSONResponse(w).WithStatusCode(code).Send(&SimpleResponse{
		Message: message,
		Success: false,
	})
}

// HandleErrorAndLatency handles error and latency defined in the service configuration.
// Returns true if error was handled.
func HandleErrorAndLatency(svcConfig *config.ServiceConfig, w http.ResponseWriter) bool {
	latency := svcConfig.GetLatency()
	if latency > 0 {
		slog.Info(fmt.Sprintf("Encountered latency of %s", latency))
		time.Sleep(latency)
	}

	errCode := svcConfig.GetError()
	if errCode == 0 {
		return false
	}

	NewAPIResponse(w).
		WithStatusCode(errCode).
		WithHeader("content-type", "text/plain").
		Send([]byte(fmt.Sprintf("configured service error: %d", errCode)))
	return true
}
