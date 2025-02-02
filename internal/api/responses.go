package api

import (
	"encoding/json"
	"net/http"
)

// BaseResponse is a base response type.
type BaseResponse struct {
	statusCode int
	headers    map[string]string
	w          http.ResponseWriter
}

// APIResponse is a response type for API responses.
type APIResponse struct {
	*BaseResponse
}

// NewAPIResponse creates a new APIResponse instance.
func NewAPIResponse(w http.ResponseWriter) *APIResponse {
	return &APIResponse{
		&BaseResponse{
			w: w,
		},
	}
}

// WithHeader adds a header to the response.
func (r *APIResponse) WithHeader(key string, value string) *APIResponse {
	if len(r.headers) == 0 {
		r.headers = make(map[string]string)
	}
	r.headers[key] = value
	return r
}

// WithStatusCode sets the status code of the response.
func (r *APIResponse) WithStatusCode(code int) *APIResponse {
	r.statusCode = code
	return r
}

// Send sends the data to the client.
func (r *APIResponse) Send(data []byte) {
	statusCode := r.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	for k, v := range r.headers {
		r.w.Header().Set(k, v)
	}
	r.w.WriteHeader(statusCode)
	_, _ = r.w.Write(data)
}

// JSONResponse is a response type for JSON responses.
type JSONResponse struct {
	*BaseResponse
}

// NewJSONResponse creates a new JSONResponse instance.
func NewJSONResponse(w http.ResponseWriter) *JSONResponse {
	return &JSONResponse{
		&BaseResponse{
			w: w,
		},
	}
}

// WithHeader adds a header to the response.
func (r *JSONResponse) WithHeader(key string, value string) *JSONResponse {
	if len(r.headers) == 0 {
		r.headers = make(map[string]string)
	}
	r.headers[key] = value
	return r
}

// WithStatusCode sets the status code of the response.
func (r *JSONResponse) WithStatusCode(code int) *JSONResponse {
	r.statusCode = code
	return r
}

// Send sends the data as JSON to the client.
// WriteHeader must be called before any writing happens and just once.
func (r *JSONResponse) Send(data any) {
	statusCode := r.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	r.w.Header().Set("content-type", "application/json")
	for k, v := range r.headers {
		r.w.Header().Set(k, v)
	}

	if data == nil {
		r.w.WriteHeader(statusCode)
		_, _ = r.w.Write(nil)
		return
	}

	// Convert []interface{} to JSON bytes
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		r.w.WriteHeader(http.StatusInternalServerError)
		_, _ = r.w.Write([]byte(err.Error()))
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
