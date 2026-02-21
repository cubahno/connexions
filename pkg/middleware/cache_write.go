package middleware

import (
	"bytes"
	"net/http"

	"github.com/cubahno/connexions/v2/pkg/db"
)

// CreateCacheWriteMiddleware is a method on the Router to create a middleware
func CreateCacheWriteMiddleware(params *Params) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			history := params.DB().History()

			_, ok := history.Get(req)
			if !ok {
				_ = history.Set(req.URL.Path, req, nil)
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

			history.SetResponse(req, &db.Response{
				Data:        respContent,
				StatusCode:  respStatusCode,
				ContentType: respContentType,
			})

			_, _ = w.Write(respContent)
		})
	}
}
