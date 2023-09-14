package connexions

import (
	"log"
	"net/http"
	"time"
)

type BaseHandler struct {
}

func (h *BaseHandler) SimpleResponse(w http.ResponseWriter) *SimpleResponse {
	return &SimpleResponse{
		w: w,
	}
}

func (h *BaseHandler) JSONResponse(w http.ResponseWriter) *JSONResponse {
	return &JSONResponse{
		w: w,
	}
}

func (h *BaseHandler) success(message string, w http.ResponseWriter) {
	h.SimpleResponse(w).WithMessage(message).WithSuccess(true).Send()
}

func (h *BaseHandler) error(code int, message string, w http.ResponseWriter) {
	h.SimpleResponse(w).WithMessage(message).WithStatusCode(code).WithSuccess(false).Send()
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
