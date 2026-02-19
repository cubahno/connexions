package db

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

// memoryHistoryTable is an in-memory implementation of HistoryTable.
type memoryHistoryTable struct {
	serviceName string
	storage     Table // reference to service's generic storage table

	mu         sync.RWMutex
	data       map[string]*RequestedResource
	cancelFunc context.CancelFunc
}

// newMemoryHistoryTable creates a new in-memory history table.
// clearTimeout specifies how often the history is cleared (0 means no auto-clear).
func newMemoryHistoryTable(serviceName string, storage Table, clearTimeout time.Duration) *memoryHistoryTable {
	ctx, cancel := context.WithCancel(context.Background())

	h := &memoryHistoryTable{
		serviceName: serviceName,
		storage:     storage,
		data:        make(map[string]*RequestedResource),
		cancelFunc:  cancel,
	}

	if clearTimeout > 0 {
		startResetTicker(ctx, h, clearTimeout)
	}

	return h
}

// Get retrieves a request record by the HTTP request.
func (h *memoryHistoryTable) Get(req *http.Request) (*RequestedResource, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	value, ok := h.data[h.getKey(req)]
	return value, ok
}

// Set stores a request record.
func (h *memoryHistoryTable) Set(resource string, req *http.Request, response *Response) *RequestedResource {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := h.getKey(req)
	record, recordExists := h.data[key]

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

	result := &RequestedResource{
		Resource: resource,
		Body:     body,
		Request:  req,
		Response: response,
		Storage:  h.storage,
	}

	h.data[key] = result
	return result
}

// SetResponse updates the response for an existing request record.
func (h *memoryHistoryTable) SetResponse(req *http.Request, response *Response) {
	h.mu.Lock()
	defer h.mu.Unlock()

	res, exists := h.data[h.getKey(req)]
	if !exists {
		slog.Info(fmt.Sprintf("Request for URL %s not found. Cannot set response", req.URL.String()))
		return
	}

	res.Response = response
}

// Data returns all request records.
func (h *memoryHistoryTable) Data() map[string]*RequestedResource {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cp := make(map[string]*RequestedResource, len(h.data))
	for k, v := range h.data {
		cp[k] = v
	}
	return cp
}

// Clear removes all history records.
func (h *memoryHistoryTable) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data = make(map[string]*RequestedResource)
}

// cancel stops the auto-clear goroutine.
func (h *memoryHistoryTable) cancel() {
	if h.cancelFunc != nil {
		h.cancelFunc()
	}
}

func (h *memoryHistoryTable) getKey(req *http.Request) string {
	return strings.Join([]string{req.Method, req.URL.String()}, ":")
}

// startResetTicker starts a goroutine that clears the history periodically.
func startResetTicker(ctx context.Context, h *memoryHistoryTable, clearTimeout time.Duration) {
	ticker := time.NewTicker(clearTimeout)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.Clear()
			}
		}
	}()
}

