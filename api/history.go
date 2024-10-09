package api

import (
	"github.com/cubahno/connexions_plugin"
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type CurrentRequestStorage struct {
	data map[string]*connexions_plugin.RequestedResource
	mu   sync.RWMutex
}

func NewCurrentRequestStorage() *CurrentRequestStorage {
	storage := &CurrentRequestStorage{
		data: make(map[string]*connexions_plugin.RequestedResource),
	}
	startResetTicker(storage)
	return storage
}

func startResetTicker(storage *CurrentRequestStorage) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Start a goroutine that clears the storage every time the ticker triggers
	go func() {
		for {
			<-ticker.C
			storage.Clear()
			log.Println("Current request storage cleared")
		}
	}()
}

// Get retrieves a value from the storage
func (s *CurrentRequestStorage) Get(req *http.Request) (*connexions_plugin.RequestedResource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[s.getKey(req)]
	return value, ok
}

// Set adds or updates a value in the storage
func (s *CurrentRequestStorage) Set(resource string, req *http.Request, response *connexions_plugin.HistoryResponse) {
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

	s.data[s.getKey(req)] = &connexions_plugin.RequestedResource{
		Resource: resource,
		Method:   req.Method,
		URL:      req.URL,
		Headers:  req.Header,
		Body:     body,
		Response: response,
	}
}

// SetResponse updates response value in the storage
func (s *CurrentRequestStorage) SetResponse(request *http.Request, response *connexions_plugin.HistoryResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the request exists
	res, exists := s.data[s.getKey(request)]
	if !exists {
		// Log a message if the request is not found
		log.Printf("Request for URL %s not found. Cannot set response.\n", request.URL.String())
		return
	}

	if res.Response != nil {
		log.Println("response was already set, will not overwrite")
		return
	}

	res.Response = response
}

// Clear removes all keys from the storage
func (s *CurrentRequestStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]*connexions_plugin.RequestedResource)
}

func (s *CurrentRequestStorage) getKey(req *http.Request) string {
	return middleware.GetReqID(req.Context())
}
