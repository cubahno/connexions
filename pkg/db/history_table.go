package db

import (
	"context"
	"net/http"
)

// HistoryTable provides typed access to request/response history.
type HistoryTable interface {
	// Get retrieves a request record by the HTTP request.
	Get(ctx context.Context, req *http.Request) (*HistoryEntry, bool)

	// Set stores a request record.
	Set(ctx context.Context, resource string, req *http.Request, response *HistoryResponse) *HistoryEntry

	// SetResponse updates the response for an existing request record.
	SetResponse(ctx context.Context, req *http.Request, response *HistoryResponse)

	// Data returns all request records.
	Data(ctx context.Context) map[string]*HistoryEntry

	// Clear removes all history records.
	Clear(ctx context.Context)
}

// HistoryEntry represents a recorded request in the history.
// Resource is the openapi resource path, i.e. /pets, /pets/{id}
// Body is the request body if method is not GET
// Response is the response if present
// Request is the http request
type HistoryEntry struct {
	Resource string
	Body     []byte
	Response *HistoryResponse
	Request  *http.Request
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
