package api

import (
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
)

type RequestedResource struct {
	Resource string
	Method   string
	URL      *url.URL
	Headers  map[string][]string
	Body     []byte
	Response *HistoryResponse
}

type HistoryResponse struct {
	Data           []byte
	StatusCode     int
	IsFromUpstream bool
}

type CurrentRequestStorage struct {
	data map[string]*RequestedResource
	mu   sync.RWMutex
}

func NewCurrentRequestStorage() *CurrentRequestStorage {
	return &CurrentRequestStorage{
		data: make(map[string]*RequestedResource),
	}
}

// Get retrieves a value from the storage
func (s *CurrentRequestStorage) Get(req *http.Request) (*RequestedResource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[s.getKey(req)]
	return value, ok
}

// Set adds or updates a value in the storage
func (s *CurrentRequestStorage) Set(resource string, req *http.Request, response *HistoryResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Extract the body (if necessary)
	var body []byte
	if req.Body != nil {
		defer req.Body.Close()
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			log.Printf("Error reading request body: %v\n", err)
		}
	}

	s.data[s.getKey(req)] = &RequestedResource{
		Resource: resource,
		Method:   req.Method,
		URL:      req.URL,
		Headers:  req.Header,
		Body:     body,
		Response: response,
	}
}

// SetResponse updates response value in the storage
func (s *CurrentRequestStorage) SetResponse(request *http.Request, response *HistoryResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the request exists
	if res, exists := s.data[s.getKey(request)]; exists {
		res.Response = response
	} else {
		// Log a message if the request is not found
		log.Printf("Request for URL %s not found. Cannot set response.\n", request.URL.String())
	}
}

// Clear removes all keys from the storage
func (s *CurrentRequestStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]*RequestedResource)
}

func (s *CurrentRequestStorage) getKey(req *http.Request) string {
	return middleware.GetReqID(req.Context())
}
