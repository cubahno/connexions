package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// JSONResponse is a response builder for JSON responses.
type JSONResponse struct {
	w          http.ResponseWriter
	statusCode int
	headers    map[string]string
}

// NewJSONResponse creates a new JSONResponse instance.
func NewJSONResponse(w http.ResponseWriter) *JSONResponse {
	return &JSONResponse{
		w:       w,
		headers: make(map[string]string),
	}
}

// WithHeader adds a header to the response.
func (r *JSONResponse) WithHeader(key string, value string) *JSONResponse {
	r.headers[key] = value
	return r
}

// WithStatusCode sets the status code of the response.
func (r *JSONResponse) WithStatusCode(code int) *JSONResponse {
	r.statusCode = code
	return r
}

// Send sends the data as JSON to the client.
func (r *JSONResponse) Send(data any) {
	statusCode := r.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	// Set content-type header
	r.w.Header().Set("Content-Type", "application/json")

	// Set custom headers
	for k, v := range r.headers {
		r.w.Header().Set(k, v)
	}

	// Handle nil data
	if data == nil {
		r.w.WriteHeader(statusCode)
		_, _ = r.w.Write([]byte("null"))
		return
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		r.w.WriteHeader(http.StatusInternalServerError)
		_, _ = r.w.Write([]byte(`{"error":"failed to marshal response"}`))
		return
	}

	r.w.WriteHeader(statusCode)
	_, _ = r.w.Write(jsonBytes)
}

// SimpleResponse is a simple response type to indicate the success of an operation.
type SimpleResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SendHTML is a helper function to send HTML responses.
func SendHTML(w http.ResponseWriter, statusCode int, data []byte) {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write(data)
}

// StaticResponse represents a static response for static handlers.
type StaticResponse struct {
	ContentType string
	Content     string
}

// UnmarshalResponseInto unmarshals response data into the provided destination.
// Content type handling:
//   - JSON types (application/json, +json variants) → JSON unmarshal
//   - Everything else → assigns directly ([]byte or string)
func UnmarshalResponseInto[T any](data []byte, contentType string, dest *T) error {
	if len(data) == 0 {
		return nil
	}

	// JSON types - unmarshal
	if isJSONContentType(contentType) {
		return json.Unmarshal(data, dest)
	}

	// Non-JSON: try []byte first, then string
	if bytesPtr, ok := any(dest).(*[]byte); ok {
		*bytesPtr = data
		return nil
	}

	if strPtr, ok := any(dest).(*string); ok {
		*strPtr = string(data)
		return nil
	}

	// Fallback: try JSON unmarshal anyway
	return json.Unmarshal(data, dest)
}

// isJSONContentType returns true for JSON content types.
func isJSONContentType(contentType string) bool {
	ct := contentType
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}

	return ct == "application/json" ||
		strings.HasSuffix(ct, "+json")
}
