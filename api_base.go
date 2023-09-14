package connexions

import (
	"log"
	"net/http"
	"time"
)

type BaseHandler struct {
}

func (h *BaseHandler) JSONResponse(w http.ResponseWriter) *JSONResponse {
	return &JSONResponse{
		&BaseResponse{w: w},
	}
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
		Success:false,
	})
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
		NewAPIResponse(w).WithStatusCode(err).Send([]byte("Random config error"))
		return true
	}

	return false
}
