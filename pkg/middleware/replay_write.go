package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"time"

	"github.com/cubahno/connexions/v2/pkg/config"
)

// CreateReplayWriteMiddleware returns middleware that records responses for replay.
// Activates when the X-Cxs-Replay header is present, or when auto-replay is enabled
// in config for the matching endpoint.
// It wraps downstream handlers, captures the response, and stores it indexed by match field values.
// Responses sourced from cache or replay are never recorded. When upstream-only is set,
// only upstream responses are recorded.
func CreateReplayWriteMiddleware(params *Params) func(http.Handler) http.Handler {
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
			ctx := req.Context()

			// Skip if already recorded
			if _, exists := table.Get(ctx, key); exists {
				next.ServeHTTP(w, req)
				return
			}

			// Wrap response writer to capture output
			rw := &responseWriter{
				ResponseWriter: w,
				body:           new(bytes.Buffer),
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, req)

			respContent := rw.body.Bytes()
			respStatusCode := rw.statusCode
			respContentType := rw.Header().Get("Content-Type")

			// Check source — don't record cache or replay responses
			source := rw.Header().Get(ResponseHeaderSource)
			if source == ResponseHeaderSourceCache || source == ResponseHeaderSourceReplay {
				writeThrough(w, rw)
				return
			}

			// Check upstream-only: if configured, only record upstream responses.
			// Return an error so the caller knows recording was skipped.
			isFromUpstream := source == ResponseHeaderSourceUpstream
			replayCfg := cfg.Cache != nil && cfg.Cache.Replay != nil
			if replayCfg && cfg.Cache.Replay.UpstreamOnly && !isFromUpstream {
				w.Header().Set(ResponseHeaderSource, source)
				http.Error(w, "replay: upstream-only is configured but response source is "+source, http.StatusBadGateway)
				return
			}

			// Resolve TTL
			ttl := config.DefaultReplayTTL
			if replayCfg && cfg.Cache.Replay.TTL > 0 {
				ttl = cfg.Cache.Replay.TTL
			}

			// Extract match values for debugging
			matchValues := make(map[string]any, len(matchFields))
			for _, field := range matchFields {
				matchValues[field] = extractJSONPath(body, field)
			}

			// Capture response headers
			headers := make(map[string]string)
			for k := range rw.Header() {
				// Skip our internal headers
				if k == ResponseHeaderSource || k == "X-Cxs-Duration" {
					continue
				}
				headers[k] = rw.Header().Get(k)
			}

			rec := &ReplayRecord{
				Data:           respContent,
				Headers:        headers,
				StatusCode:     respStatusCode,
				ContentType:    respContentType,
				IsFromUpstream: isFromUpstream,
				RequestBody:    body,
				MatchValues:    matchValues,
				CreatedAt:      time.Now(),
			}

			table.Set(ctx, key, rec, ttl)
			slog.Info("Replay recorded", "method", req.Method, "path", req.URL.Path)

			writeThrough(w, rw)
		})
	}
}

// writeThrough writes the captured response to the real response writer.
func writeThrough(w http.ResponseWriter, rw *responseWriter) {
	for k, vals := range rw.Header() {
		for _, v := range vals {
			w.Header().Set(k, v)
		}
	}
	w.WriteHeader(rw.statusCode)
	_, _ = w.Write(rw.body.Bytes())
}
