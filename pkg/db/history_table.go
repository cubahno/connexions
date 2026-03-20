package db

import (
	"context"
	"net/http"
	"time"
)

// HistoryTable provides typed access to request/response history.
type HistoryTable interface {
	// Get retrieves the latest request record matching the HTTP request's method and URL.
	Get(ctx context.Context, req *http.Request) (*HistoryEntry, bool)

	// Set stores a request record with a unique ID.
	Set(ctx context.Context, resource string, req *http.Request, response *HistoryResponse) *HistoryEntry

	// SetResponse updates the response for the latest request record matching the HTTP request.
	SetResponse(ctx context.Context, req *http.Request, response *HistoryResponse)

	// Data returns all request records as an ordered log.
	Data(ctx context.Context) []*HistoryEntry

	// Len returns the number of history entries.
	Len(ctx context.Context) int

	// Clear removes all history records.
	Clear(ctx context.Context)
}

// HistoryEntry represents a recorded request in the history.
// ID is a unique identifier for this entry
// Resource is the openapi resource path, i.e. /pets, /pets/{id}
// Body is the request body if method is not GET
// Response is the response if present
// Request is the http request
// RemoteAddr is the client IP address (from req.RemoteAddr)
type HistoryEntry struct {
	ID         string
	Resource   string
	Body       []byte
	Response   *HistoryResponse
	Request    *http.Request
	RemoteAddr string
	CreatedAt  time.Time
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
