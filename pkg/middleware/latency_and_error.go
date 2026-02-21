package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func CreateLatencyAndErrorMiddleware(params *Params) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			cfg := params.ServiceConfig
			latency := cfg.GetLatency()
			if latency > 0 {
				slog.Info("Latency", slog.Duration("delay", latency))
				time.Sleep(latency)
			}

			errorCode := cfg.GetError()
			if errorCode > 0 {
				slog.Info("Simulated error", slog.Int("code", errorCode))
				SetDurationHeader(w, req)
				w.Header().Set(ResponseHeaderSource, ResponseHeaderSourceGenerated)
				http.Error(w, "Simulated error", errorCode)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}
