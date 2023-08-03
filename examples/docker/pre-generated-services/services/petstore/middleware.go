package petstore

import (
	"net/http"

	"github.com/cubahno/connexions/v2/pkg/middleware"
)

// getMiddleware returns custom middleware for the petstore service.
//
// This function is called during service registration.
// Middleware returned here will be applied AFTER the standard middleware chain.
// Example:
//
//	return []func(*middleware.Params) func(http.Handler) http.Handler{
//	    createAuthMiddleware,
//	    createLoggingMiddleware,
//	}
func getMiddleware() []func(*middleware.Params) func(http.Handler) http.Handler {
	return []func(*middleware.Params) func(http.Handler) http.Handler{
		// Add your custom middleware here
	}
}

// Example middleware - uncomment and customize as needed:
//
// func createAuthMiddleware(params *middleware.Params) func(http.Handler) http.Handler {
//     return func(next http.Handler) http.Handler {
//         return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
//             // Your authentication logic here
//             // Example: check headers, validate tokens, etc.
//
//             next.ServeHTTP(w, req)
//         })
//     }
// }
