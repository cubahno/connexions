package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/cubahno/connexions/v2/pkg/db"
)

// CreateCacheWriteMiddleware is a method on the Router to create a middleware
func CreateCacheWriteMiddleware(params *Params) func(http.Handler) http.Handler {
	recordHistory := params.ServiceConfig.HistoryEnabled()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Capture request body before downstream handlers consume it.
			var requestBody []byte
			if recordHistory && req.Body != nil && req.Body != http.NoBody {
				requestBody, _ = io.ReadAll(req.Body)
				req.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}

			// Create a responseWriter to capture the response.
			// default to 200 status code
			rw := &responseWriter{
				ResponseWriter: w,
				body:           new(bytes.Buffer),
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, req)

			respContent := rw.body.Bytes()
			respStatusCode := rw.statusCode
			respContentType := rw.Header().Get("Content-Type")

			// Record request + response asynchronously - no need to block the response.
			if recordHistory {
				urlCopy := *req.URL
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), asyncWriteTimeout)
					defer cancel()
					params.DB().History().Set(ctx, req.URL.Path, &http.Request{
						Method:     req.Method,
						URL:        &urlCopy,
						Body:       io.NopCloser(bytes.NewBuffer(requestBody)),
						RemoteAddr: req.RemoteAddr,
					}, &db.HistoryResponse{
						Data:        respContent,
						StatusCode:  respStatusCode,
						ContentType: respContentType,
					})
				}()
			}

			// Set our custom headers before writing
			SetDurationHeader(w, req)
			w.Header().Set(ResponseHeaderSource, ResponseHeaderSourceGenerated)
			if respContentType != "" {
				w.Header().Set("Content-Type", respContentType)
			}
			w.WriteHeader(respStatusCode)
			_, _ = w.Write(respContent)
		})
	}
}
