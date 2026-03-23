package db

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// memoryHistoryTable is an in-memory implementation of HistoryTable.
// It stores entries as an ordered log with unique IDs and maintains
// an index of METHOD:URL → latest entry for fast lookups.
type memoryHistoryTable struct {
	mu          sync.RWMutex
	entries     []*HistoryEntry
	latestIndex map[string]*HistoryEntry // lookupKey → latest entry
	counter     atomic.Int64
	cancelFunc  context.CancelFunc
}

// newMemoryHistoryTable creates a new in-memory history table.
// clearTimeout specifies how often the history is cleared (0 means no auto-clear).
func newMemoryHistoryTable(_ *memoryTable, clearTimeout time.Duration) *memoryHistoryTable {
	ctx, cancel := context.WithCancel(context.Background())

	h := &memoryHistoryTable{
		latestIndex: make(map[string]*HistoryEntry),
		cancelFunc:  cancel,
	}

	if clearTimeout > 0 {
		startResetTicker(ctx, h, clearTimeout)
	}

	return h
}

// Get retrieves the latest request record matching the HTTP request's method and URL.
func (h *memoryHistoryTable) Get(_ context.Context, req *http.Request) (*HistoryEntry, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entry, ok := h.latestIndex[lookupKey(req.Method, req.URL.String())]
	return entry, ok
}

// Set stores a request record with a unique ID.
func (h *memoryHistoryTable) Set(_ context.Context, resource string, req *HistoryRequest, response *HistoryResponse) *HistoryEntry {
	id := fmt.Sprintf("%d", h.counter.Add(1))

	entry := &HistoryEntry{
		ID:        id,
		Resource:  resource,
		Request:   req,
		Response:  response,
		CreatedAt: time.Now().UTC(),
	}

	h.mu.Lock()
	h.entries = append(h.entries, entry)
	h.latestIndex[lookupKey(req.Method, req.URL)] = entry
	h.mu.Unlock()

	return entry
}

// SetResponse updates the response for the latest request record matching the request's method and URL.
func (h *memoryHistoryTable) SetResponse(_ context.Context, req *HistoryRequest, response *HistoryResponse) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry, ok := h.latestIndex[lookupKey(req.Method, req.URL)]
	if !ok {
		slog.Info(fmt.Sprintf("Request for URL %s not found. Cannot set response", req.URL))
		return
	}
	entry.Response = response
}

// GetByID retrieves a single history entry by its ID.
func (h *memoryHistoryTable) GetByID(_ context.Context, id string) (*HistoryEntry, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, entry := range h.entries {
		if entry.ID == id {
			return entry, true
		}
	}
	return nil, false
}

// Data returns all request records as an ordered log.
func (h *memoryHistoryTable) Data(_ context.Context) []*HistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*HistoryEntry, len(h.entries))
	copy(result, h.entries)
	return result
}

// Len returns the number of history entries.
func (h *memoryHistoryTable) Len(_ context.Context) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.entries)
}

// Clear removes all history records.
func (h *memoryHistoryTable) Clear(_ context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = nil
	h.latestIndex = make(map[string]*HistoryEntry)
}

// cancel stops the auto-clear goroutine.
func (h *memoryHistoryTable) cancel() {
	if h.cancelFunc != nil {
		h.cancelFunc()
	}
}

func lookupKey(method, url string) string {
	return method + ":" + url
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
				h.Clear(ctx)
			}
		}
	}()
}
