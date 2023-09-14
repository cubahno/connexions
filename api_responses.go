package connexions

import (
	"encoding/json"
	"net/http"
)

func SetAPIResponseContentType(value string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("content-type", value)
	}
}

func NewAPIResponse(statusCode int, res []byte, w http.ResponseWriter, headers ...func(w http.ResponseWriter)) {
	for _, header := range headers {
		header(w)
	}
	w.WriteHeader(statusCode)
	_, _ = w.Write(res)
}

type JSONResponse struct {
	statusCode int
	headers    map[string]string
	w          http.ResponseWriter
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

func (r *SimpleResponse) WithMessage(message string) *SimpleResponse {
	r.Message = message
	return r
}

func (r *SimpleResponse) WithSuccess(success bool) *SimpleResponse {
	r.Success = success
	return r
}

func (r *SimpleResponse) WithStatusCode(code int) *SimpleResponse {
	r.statusCode = code
	return r
}

func (r *SimpleResponse) WithHeader(key string, value string) *SimpleResponse {
	if len(r.headers) == 0 {
		r.headers = make(map[string]string)
	}
	r.headers[key] = value
	return r
}

func (r *SimpleResponse) WithJSON() *SimpleResponse {
	return r.WithHeader("content-type", "application/json")
}

func (r *SimpleResponse) WithError(err error) *SimpleResponse {
	return r.WithMessage(err.Error()).WithSuccess(false)
}

func (r *SimpleResponse) Send() {
	statusCode := r.statusCode

	// nothing was set, assume success
	if statusCode == 0 && !r.Success {
		r.Success = true
		statusCode = http.StatusOK
	}

	if statusCode < http.StatusBadRequest && !r.Success {
		r.Success = true
	}

	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	contentType := r.headers["content-type"]
	if contentType == "" {
		contentType = "application/json"
		r.w.Header().Set("content-type", contentType)
	}

	for k, v := range r.headers {
		r.w.Header().Set(k, v)
	}
	r.w.WriteHeader(statusCode)

	// Convert []interface{} to JSON bytes
	jsonBytes, err := json.Marshal(r)
	if err != nil {
		_, _ = r.w.Write([]byte(err.Error()))
		r.w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, _ = r.w.Write(jsonBytes)
}
