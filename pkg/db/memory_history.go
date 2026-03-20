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

	entry, ok := h.latestIndex[h.lookupKey(req)]
	return entry, ok
}

// Set stores a request record with a unique ID.
func (h *memoryHistoryTable) Set(_ context.Context, resource string, req *http.Request, response *HistoryResponse) *HistoryEntry {
	id := fmt.Sprintf("%d", h.counter.Add(1))

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
	}

	entry := &HistoryEntry{
		ID:         id,
		Resource:   resource,
		Body:       body,
		Request:    req,
		Response:   response,
		RemoteAddr: req.RemoteAddr,
		CreatedAt:  time.Now().UTC(),
	}

	h.mu.Lock()
	h.entries = append(h.entries, entry)
	h.latestIndex[h.lookupKey(req)] = entry
	h.mu.Unlock()

	return entry
}

// SetResponse updates the response for the latest request record matching the HTTP request.
func (h *memoryHistoryTable) SetResponse(_ context.Context, req *http.Request, response *HistoryResponse) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry, ok := h.latestIndex[h.lookupKey(req)]
	if !ok {
		slog.Info(fmt.Sprintf("Request for URL %s not found. Cannot set response", req.URL.String()))
		return
	}
	entry.Response = response
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

func (h *memoryHistoryTable) lookupKey(req *http.Request) string {
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
				h.Clear(ctx)
			}
		}
	}()
}
