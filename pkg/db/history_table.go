package db

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"
)

// HistoryTable provides typed access to request/response history.
type HistoryTable interface {
	// Get retrieves the latest request record matching the HTTP request's method and URL.
	Get(ctx context.Context, req *http.Request) (*HistoryEntry, bool)

	// Set stores a request record with a unique ID.
	Set(ctx context.Context, resource string, req *HistoryRequest, response *HistoryResponse) *HistoryEntry

	// SetResponse updates the response for the latest request record matching the request's method and URL.
	SetResponse(ctx context.Context, req *HistoryRequest, response *HistoryResponse)

	// GetByID retrieves a single history entry by its ID.
	GetByID(ctx context.Context, id string) (*HistoryEntry, bool)

	// Data returns all request records as an ordered log.
	Data(ctx context.Context) []*HistoryEntry

	// Len returns the number of history entries.
	Len(ctx context.Context) int

	// Clear removes all history records.
	Clear(ctx context.Context)
}

// HistoryRequest represents the HTTP request stored in a history entry.
type HistoryRequest struct {
	Method     string   `json:"method"`
	URL        string   `json:"url"`
	Body       []byte   `json:"body,omitempty"`
	Headers    []string `json:"headers,omitempty"`
	RemoteAddr string   `json:"remoteAddr,omitempty"`
	RequestID  string   `json:"requestId,omitempty"`
}

// HistoryEntry represents a recorded request in the history.
// ID is a unique identifier for this entry
// Resource is the openapi resource path, i.e. /pets, /pets/{id}
// Response is the response if present
// Request is the method, URL, body, headers, and remote address of the original request
type HistoryEntry struct {
	ID        string           `json:"id"`
	Resource  string           `json:"resource"`
	Response  *HistoryResponse `json:"response,omitempty"`
	Request   *HistoryRequest  `json:"request,omitempty"`
	CreatedAt time.Time        `json:"createdAt"`
}

// HistoryResponse represents the response that was generated or received from the server.
// Body is the response body
// StatusCode is the HTTP status code returned
// ContentType is the Content-Type header of the response
// IsFromUpstream is true if the response was received from the upstream server
// UpstreamURL is the URL that was actually sent to the upstream service
// Duration is the time taken to produce the response
type HistoryResponse struct {
	Body           []byte        `json:"body"`
	StatusCode     int           `json:"statusCode"`
	ContentType    string        `json:"contentType"`
	IsFromUpstream bool          `json:"isFromUpstream"`
	UpstreamURL    string        `json:"upstreamURL"`
	Headers        []string      `json:"headers,omitempty"`
	Duration       time.Duration `json:"duration,omitempty"`
}

// FlattenHeaders converts http.Header to a sorted slice of "Key: value" strings.
func FlattenHeaders(h http.Header) []string {
	if len(h) == 0 {
		return nil
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]string, 0, len(h))
	for _, k := range keys {
		result = append(result, k+": "+strings.Join(h.Values(k), ", "))
	}
	return result
}
