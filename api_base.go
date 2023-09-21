package connexions

import (
	"log"
	"net/http"
	"time"
)

type BaseHandler struct {
}

// func (h *BaseHandler) CreateValueReplacer(fileProps *FileProperties, config *Config) {
// 	serviceCfg := config.GetServiceConfig(fileProps.ServiceName)
// 	valueReplaceFactory := CreateValueReplacer(config)
// }

func (h *BaseHandler) Response(w http.ResponseWriter) *APIResponse {
	return NewAPIResponse(w)
}

func (h *BaseHandler) JSONResponse(w http.ResponseWriter) *JSONResponse {
	return NewJSONResponse(w)
}

func (h *BaseHandler) success(message string, w http.ResponseWriter) {
	h.JSONResponse(w).Send(&SimpleResponse{
		Message: message,
		Success: true,
	})
}

func (h *BaseHandler) error(code int, message string, w http.ResponseWriter) {
	h.JSONResponse(w).WithStatusCode(code).Send(&SimpleResponse{
		Message: message,
		Success: false,
	})
}

func HandleErrorAndLatency(svcConfig *ServiceConfig, w http.ResponseWriter) bool {
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
