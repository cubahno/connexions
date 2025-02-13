package plugin

import (
    "net/http"
    "net/url"
)

// RequestedResource represents the current request that is being processed.
// Resource is the openapi resource path, i.e. /pets, /pets/{id}
// Method is the HTTP method, i.e. GET, POST, PUT, DELETE
// URL is the full URL of the request
// Headers is the request headers
// Body is the request body if method is not GET
// Response is the current response if present
type RequestedResource struct {
    Resource string
    Method   string
    URL      *url.URL
    Headers  map[string][]string
    Body     []byte
    Response *HistoryResponse
}

// HistoryResponse represents the response that was generated or received from the server.
// Data is the response body
// StatusCode is the HTTP status code returned
// IsFromUpstream is true if the response was received from the upstream server
type HistoryResponse struct {
    Data           []byte
    StatusCode     int
    IsFromUpstream bool
}

// RequestTransformer is a function that modifies the request before it is sent to the server.
// Resource represents openapi resource path, i.e. /pets, /pets/{id}
type RequestTransformer func(resource string, request *http.Request) (*http.Request, error)
