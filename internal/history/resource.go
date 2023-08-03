package history

import "net/http"

// RequestedResource represents the current request that is being processed.
// Resource is the openapi resource path, i.e. /pets, /pets/{id}
// Body is the request body if method is not GET
// Response is the current response if present
// Request is the current http request
// ServiceStorage is the thread-safe storage that is used to store and retrieve data for this service.
// Auto-cleared by history storage timeout.
type RequestedResource struct {
	Resource       string
	Body           []byte
	Response       *HistoryResponse
	Request        *http.Request
	ServiceStorage Storage
}

// HistoryResponse represents the response that was generated or received from the server.
// Data is the response body
// StatusCode is the HTTP status code returned
// ContentType is the Content-Type header of the response
// IsFromUpstream is true if the response was received from the upstream server
type HistoryResponse struct {
	Data           []byte
	StatusCode     int
	ContentType    string
	IsFromUpstream bool
}
