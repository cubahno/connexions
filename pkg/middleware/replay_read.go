package middleware

import (
	"net/http"
)

// CreateReplayReadMiddleware returns middleware that checks for a matching replay recording.
// Activates when the X-Cxs-Replay header is present, or when auto-replay is enabled
// in config for the matching endpoint.
// On hit, it returns the stored response with X-Cxs-Source: replay.
// On miss, it passes through to the next handler.
func CreateReplayReadMiddleware(params *Params) func(http.Handler) http.Handler {
	log := params.Logger("replay-read")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			cfg := params.ServiceConfig
			if cfg == nil {
				next.ServeHTTP(w, req)
				return
			}

			matchFields, patternPath := resolveReplayParams(req, cfg)
			if len(matchFields) == 0 {
				next.ServeHTTP(w, req)
				return
			}

			body := readAndRestoreBody(req)
			key := buildReplayKey(req.Method, patternPath, matchFields, body)

			table := params.DB().Table("replay")
			val, exists := table.Get(req.Context(), key)
			if !exists {
				next.ServeHTTP(w, req)
				return
			}

			rec := deserializeReplayRecord(val)
			if rec == nil {
				next.ServeHTTP(w, req)
				return
			}

			log.Info("Replay hit", "method", req.Method, "path", req.URL.Path)

			// Restore recorded headers
			for k, v := range rec.Headers {
				w.Header().Set(k, v)
			}

			SetDurationHeader(w, req)
			w.Header().Set(ResponseHeaderSource, ResponseHeaderSourceReplay)
			if rec.ContentType != "" {
				w.Header().Set("Content-Type", rec.ContentType)
			}
			w.WriteHeader(rec.StatusCode)
			_, _ = w.Write(rec.Data)
		})
	}
}
