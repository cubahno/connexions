package api

import (
	"encoding/json"
	"net/http"
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
