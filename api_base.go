package connexions

import (
	"log"
	"net/http"
	"time"
)

type BaseHandler struct {
}

func (h *BaseHandler) success(message string, w http.ResponseWriter) {
	NewAPIJSONResponse(http.StatusOK, map[string]any{"success": true, "message": message}, w)
}

func (h *BaseHandler) error(code int, message string, w http.ResponseWriter) {
	NewAPIJSONResponse(code, map[string]any{"success": false, "message": message}, w)
}

func handleErrorAndLatency(svcConfig *ServiceConfig, w http.ResponseWriter) bool {
	if svcConfig.Latency > 0 {
		log.Printf("Encountered latency of %s\n", svcConfig.Latency)

		select {
		case <-time.After(svcConfig.Latency):
		}
	}

	errConfig := svcConfig.Errors
	if errConfig == nil {
		return false
	}

	err := errConfig.GetError()
	if err != 0 {
		NewAPIResponse(err, []byte("Random config error"), w)
		return true
	}

	return false
}
