package api

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cubahno/connexions/pkg/plugin"
)

// CurrentRequestStorage is a storage for requests.
type CurrentRequestStorage struct {
	data       map[string]*plugin.RequestedResource
	cancelFunc context.CancelFunc
	mu         sync.RWMutex
}

// NewCurrentRequestStorage creates a new CurrentRequestStorage instance.
// It also starts a goroutine that clears the storage every clearTimeout duration.
func NewCurrentRequestStorage(clearTimeout time.Duration) *CurrentRequestStorage {
	ctx, cancel := context.WithCancel(context.Background())

	storage := &CurrentRequestStorage{
		data:       make(map[string]*plugin.RequestedResource),
		cancelFunc: cancel,
	}
	startResetTicker(ctx, storage, clearTimeout)
	return storage
}

func startResetTicker(ctx context.Context, storage *CurrentRequestStorage, clearTimeout time.Duration) {
	ticker := time.NewTicker(clearTimeout)
	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				storage.Clear()
			}
		}
	}()
}

// Get retrieves a value from the storage
func (s *CurrentRequestStorage) Get(req *http.Request) (*plugin.RequestedResource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[s.getKey(req)]
	return value, ok
}

// Set adds or updates a value in the storage
func (s *CurrentRequestStorage) Set(resource string, req *http.Request, response *plugin.HistoryResponse) {
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

	s.data[s.getKey(req)] = &plugin.RequestedResource{
		Resource: resource,
		Method:   req.Method,
		URL:      req.URL,
		Headers:  req.Header,
		Body:     body,
		Response: response,
	}
}

// SetResponse updates response value in the storage
func (s *CurrentRequestStorage) SetResponse(request *http.Request, response *plugin.HistoryResponse) {
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
	s.data = make(map[string]*plugin.RequestedResource)
}

// Cancel stops the goroutine that clears the storage
func (s *CurrentRequestStorage) Cancel() {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
}

func (s *CurrentRequestStorage) getKey(req *http.Request) string {
	builder := []string{
		req.Method,
		req.URL.String(),
	}

	return strings.Join(builder, ":")
}

func (s *CurrentRequestStorage) getData() map[string]*plugin.RequestedResource {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data
}
