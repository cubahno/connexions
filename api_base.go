package xs

import (
	"log"
	"net/http"
	"time"
)

type BaseHandler struct {
}

func (h *BaseHandler) success(message string, w http.ResponseWriter) {
	NewJSONResponse(http.StatusOK, map[string]any{"success": true, "message": message}, w)
}

func (h *BaseHandler) error(code int, message string, w http.ResponseWriter) {
	NewJSONResponse(code, map[string]any{"success": false, "message": message}, w)
}

func handleErrorAndLatency(service string, config *Config, w http.ResponseWriter) bool {
	svcConfig := config.GetServiceConfig(service)
	if svcConfig.Latency > 0 {
		log.Printf("Latency of %s is %s\n", service, svcConfig.Latency)

		select {
		case <-time.After(svcConfig.Latency):
		}
	}

	err := svcConfig.Errors.GetError()
	if err != 0 {
		NewResponse(err, []byte("Random config error"), w)
		return true
	}

	return false
}
