package api

import (
	"encoding/json"
	"net/http"
)

func WithHeader(name, value string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set(name, value)
	}
}

func WithContentType(value string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("content-type", value)
	}
}

func NewJSONResponse(statusCode int, res any, w http.ResponseWriter, headers ...func(w http.ResponseWriter)) {
	w.Header().Set("content-type", "application/json")
	for _, header := range headers {
		header(w)
	}
	w.WriteHeader(statusCode)

	// Convert []interface{} to JSON bytes
	jsonBytes, err := json.Marshal(res)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(jsonBytes)
}

func NewResponse(statusCode int, res []byte, w http.ResponseWriter, headers ...func(w http.ResponseWriter)) {
	for _, header := range headers {
		header(w)
	}
	w.WriteHeader(statusCode)
	_, _ = w.Write(res)
}