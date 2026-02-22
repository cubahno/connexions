package db

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// memoryHistoryTable is an in-memory implementation of HistoryTable.
// It wraps a Table from the shared storage to store history records.
type memoryHistoryTable struct {
	table      *memoryTable // underlying storage table (svc:history)
	cancelFunc context.CancelFunc
}

// newMemoryHistoryTable creates a new in-memory history table.
// clearTimeout specifies how often the history is cleared (0 means no auto-clear).
func newMemoryHistoryTable(table *memoryTable, clearTimeout time.Duration) *memoryHistoryTable {
	ctx, cancel := context.WithCancel(context.Background())

	h := &memoryHistoryTable{
		table:      table,
		cancelFunc: cancel,
	}

	if clearTimeout > 0 {
		startResetTicker(ctx, h, clearTimeout)
	}

	return h
}

// Get retrieves a request record by the HTTP request.
func (h *memoryHistoryTable) Get(ctx context.Context, req *http.Request) (*HistoryEntry, bool) {
	value, ok := h.table.Get(ctx, h.getKey(req))
	if !ok {
		return nil, false
	}
	record, ok := value.(*HistoryEntry)
	return record, ok
}

// Set stores a request record.
func (h *memoryHistoryTable) Set(ctx context.Context, resource string, req *http.Request, response *HistoryResponse) *HistoryEntry {
	key := h.getKey(req)

	// Check for existing record to reuse body
	var existingBody []byte
	if existing, ok := h.table.Get(ctx, key); ok {
		if record, ok := existing.(*HistoryEntry); ok {
			existingBody = record.Body
		}
	}

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
	} else if len(existingBody) > 0 {
		// If no body in request but record exists, reuse the old body
		body = existingBody
	}

	result := &HistoryEntry{
		Resource: resource,
		Body:     body,
		Request:  req,
		Response: response,
	}

	h.table.Set(ctx, key, result, 0)
	return result
}

// SetResponse updates the response for an existing request record.
func (h *memoryHistoryTable) SetResponse(ctx context.Context, req *http.Request, response *HistoryResponse) {
	key := h.getKey(req)
	existing, exists := h.table.Get(ctx, key)
	if !exists {
		slog.Info(fmt.Sprintf("Request for URL %s not found. Cannot set response", req.URL.String()))
		return
	}

	record, ok := existing.(*HistoryEntry)
	if !ok {
		return
	}
	record.Response = response
	h.table.Set(ctx, key, record, 0)
}

// Data returns all request records.
func (h *memoryHistoryTable) Data(ctx context.Context) map[string]*HistoryEntry {
	result := make(map[string]*HistoryEntry)
	for k, v := range h.table.Data(ctx) {
		if record, ok := v.(*HistoryEntry); ok {
			result[k] = record
		}
	}
	return result
}

// Clear removes all history records.
func (h *memoryHistoryTable) Clear(ctx context.Context) {
	h.table.Clear(ctx)
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
				h.Clear(ctx)
			}
		}
	}()
}
