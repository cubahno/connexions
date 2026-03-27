package middleware

import (
	"net/http"
	"time"
)

func CreateLatencyAndErrorMiddleware(params *Params) func(http.Handler) http.Handler {
	log := params.Logger("latency-error")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			reqLog := RequestLog(log, req)
			cfg := params.GetServiceConfig(req)
			latency := cfg.GetLatency()
			if latency > 0 {
				reqLog.Info("Latency", "delay", latency)
				time.Sleep(latency)
			}

			errorCode := cfg.GetError()
			if errorCode > 0 {
				reqLog.Info("Simulated error", "code", errorCode)
				SetRequestIDHeader(w, req)
				SetDurationHeader(w, req)
				w.Header().Set(ResponseHeaderSource, ResponseHeaderSourceGenerated)
				http.Error(w, "Simulated error", errorCode)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}
