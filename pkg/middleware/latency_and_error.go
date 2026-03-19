package middleware

import (
	"net/http"
	"time"
)

func CreateLatencyAndErrorMiddleware(params *Params) func(http.Handler) http.Handler {
	log := params.Logger("latency-error")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			cfg := params.ServiceConfig
			latency := cfg.GetLatency()
			if latency > 0 {
				log.Info("Latency", "delay", latency)
				time.Sleep(latency)
			}

			errorCode := cfg.GetError()
			if errorCode > 0 {
				log.Info("Simulated error", "code", errorCode)
				SetDurationHeader(w, req)
				w.Header().Set(ResponseHeaderSource, ResponseHeaderSourceGenerated)
				http.Error(w, "Simulated error", errorCode)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}
