package connexions

import (
	"encoding/json"
	"net/http"
)

type BaseResponse struct {
	statusCode int
	headers    map[string]string
	w          http.ResponseWriter
}

type APIResponse struct {
	*BaseResponse
}

func (r *APIResponse) WithHeader(key string, value string) *APIResponse{
	if len(r.headers) == 0 {
		r.headers = make(map[string]string)
	}
	r.headers[key] = value
	return r
}

func (r *APIResponse) WithStatusCode(code int) *APIResponse {
	r.statusCode = code
	return r
}

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

func NewAPIResponse(w http.ResponseWriter) *APIResponse {
	return &APIResponse{
		&BaseResponse{
			w: w,
		},
	}
}

type JSONResponse struct {
	*BaseResponse
}

func (r *JSONResponse) WithHeader(key string, value string) *JSONResponse {
	if len(r.headers) == 0 {
		r.headers = make(map[string]string)
	}
	r.headers[key] = value
	return r
}

func (r *JSONResponse) WithStatusCode(code int) *JSONResponse {
	r.statusCode = code
	return r
}

func (r *JSONResponse) Send(data any) {
	statusCode := r.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	r.w.Header().Set("content-type", "application/json")
	for k, v := range r.headers {
		r.w.Header().Set(k, v)
	}
	r.w.WriteHeader(statusCode)

	// Convert []interface{} to JSON bytes
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		_, _ = r.w.Write([]byte(err.Error()))
		r.w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, _ = r.w.Write(jsonBytes)
}

type SimpleResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`

	statusCode int
	headers    map[string]string
	w          http.ResponseWriter
}
