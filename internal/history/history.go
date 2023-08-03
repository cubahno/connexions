package history

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CurrentRequestStorage is a storage for requests.
type CurrentRequestStorage struct {
	cancelFunc context.CancelFunc

	mu             sync.RWMutex
	data           map[string]*RequestedResource
	serviceStorage map[string]*MemoryStorage
}

// NewCurrentRequestStorage creates a new CurrentRequestStorage instance.
// It also starts a goroutine that clears the storage every clearTimeout duration.
func NewCurrentRequestStorage(clearTimeout time.Duration) *CurrentRequestStorage {
	ctx, cancel := context.WithCancel(context.Background())

	storage := &CurrentRequestStorage{
		data:           make(map[string]*RequestedResource),
		serviceStorage: make(map[string]*MemoryStorage),
		cancelFunc:     cancel,
	}
	startResetTicker(ctx, storage, clearTimeout)
	return storage
}

// Get retrieves a value from the storage
func (s *CurrentRequestStorage) Get(req *http.Request) (*RequestedResource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[s.getKey(req)]
	return value, ok
}

// Set adds or updates a value in the storage
func (s *CurrentRequestStorage) Set(service, resource string, req *http.Request, response *HistoryResponse) *RequestedResource {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.getKey(req)
	record, recordExists := s.data[key]

	// Extract the body from the request
	var body []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			slog.Error("Error reading request body", "error", err)
			body = []byte{}
		}
		// Restore the body so it can be read by subsequent handlers
		req.Body = io.NopCloser(bytes.NewBuffer(body))
	} else if recordExists {
		// If no body in request but record exists, reuse the old body
		body = record.Body
	}

	memStorage, ok := s.serviceStorage[service]
	if !ok {
		memStorage = NewMemoryStorage()
		s.serviceStorage[service] = memStorage
	}

	result := &RequestedResource{
		Resource:       resource,
		Body:           body,
		Request:        req,
		Response:       response,
		ServiceStorage: memStorage,
	}

	s.data[key] = result
	return result
}

// SetResponse updates response value in the storage
func (s *CurrentRequestStorage) SetResponse(request *http.Request, response *HistoryResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the request exists
	res, exists := s.data[s.getKey(request)]
	if !exists {
		slog.Info(fmt.Sprintf("Request for URL %s not found. Cannot set response", request.URL.String()))
		return
	}

	res.Response = response
}

// Clear removes all keys from the storage
func (s *CurrentRequestStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]*RequestedResource)
	s.serviceStorage = make(map[string]*MemoryStorage)
}

// Cancel stops the goroutine that clears the storage
func (s *CurrentRequestStorage) Cancel() {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
}

func (s *CurrentRequestStorage) Data() map[string]*RequestedResource {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data
}

func (s *CurrentRequestStorage) getKey(req *http.Request) string {
	builder := []string{
		req.Method,
		req.URL.String(),
	}

	return strings.Join(builder, ":")
}
