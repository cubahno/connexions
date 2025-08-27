package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cubahno/connexions_plugin"
)

// CurrentRequestStorage is a storage for requests.
type CurrentRequestStorage struct {
	data           map[string]*connexions_plugin.RequestedResource
	serviceStorage map[string]*MemoryStorage
	cancelFunc     context.CancelFunc
	mu             sync.RWMutex
}

// NewCurrentRequestStorage creates a new CurrentRequestStorage instance.
// It also starts a goroutine that clears the storage every clearTimeout duration.
func NewCurrentRequestStorage(clearTimeout time.Duration) *CurrentRequestStorage {
	ctx, cancel := context.WithCancel(context.Background())

	storage := &CurrentRequestStorage{
		data:           make(map[string]*connexions_plugin.RequestedResource),
		serviceStorage: make(map[string]*MemoryStorage),
		cancelFunc:     cancel,
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
func (s *CurrentRequestStorage) Get(req *http.Request) (*connexions_plugin.RequestedResource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[s.getKey(req)]
	return value, ok
}

// Set adds or updates a value in the storage
func (s *CurrentRequestStorage) Set(service, resource string, req *http.Request,
	response *connexions_plugin.HistoryResponse) *connexions_plugin.RequestedResource {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.getKey(req)
	record, recordExists := s.data[key]

	// Extract the body (if necessary)
	var body []byte
	if recordExists {
		body = record.Body
	}

	if !recordExists && req.Body != nil && req.Body != http.NoBody {
		defer func() { _ = req.Body.Close() }()
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			slog.Error("Error reading request body", "error", err)
			body = []byte{}
		}
	}

	memStorage, ok := s.serviceStorage[service]
	if !ok {
		memStorage = NewMemoryStorage()
		s.serviceStorage[service] = memStorage
	}

	result := &connexions_plugin.RequestedResource{
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
func (s *CurrentRequestStorage) SetResponse(request *http.Request, response *connexions_plugin.HistoryResponse) {
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
	s.data = make(map[string]*connexions_plugin.RequestedResource)
	s.serviceStorage = make(map[string]*MemoryStorage)
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

func (s *CurrentRequestStorage) getData() map[string]*connexions_plugin.RequestedResource {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data
}

type MemoryStorage struct {
	data map[string]any
	mu   sync.Mutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]any),
	}
}

func (s *MemoryStorage) Get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.data[key]
	return data, ok
}

func (s *MemoryStorage) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *MemoryStorage) Data() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := make(map[string]any, len(s.data))
	for k, v := range s.data {
		cp[k] = v
	}
	return cp
}
