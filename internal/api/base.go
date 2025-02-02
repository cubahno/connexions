package api

import (
	"log"
	"net/http"
	"time"

	"github.com/cubahno/connexions/internal"
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
func HandleErrorAndLatency(svcConfig *internal.ServiceConfig, w http.ResponseWriter) bool {
	if svcConfig.Latency > 0 {
		log.Printf("Encountered latency of %s\n", svcConfig.Latency)

		time.Sleep(svcConfig.Latency)
	}

	errConfig := svcConfig.Errors
	if errConfig == nil {
		return false
	}

	err := errConfig.GetError()
	if err != 0 {
		NewAPIResponse(w).WithStatusCode(err).WithHeader("content-type", "text/plain").Send([]byte("Random config error"))
		return true
	}

	return false
}
