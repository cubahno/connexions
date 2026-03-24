package middleware

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mockzilla/connexions/v2/pkg/db"
	"github.com/sony/gobreaker/v2"
)

const (
	cbKeyState  = "state"
	cbKeyEvents = "events"
	cbMaxEvents = 20
)

// CBState is the current circuit breaker snapshot.
type CBState struct {
	State                string  `json:"state"`
	Requests             uint32  `json:"requests"`
	TotalSuccesses       uint32  `json:"totalSuccesses"`
	TotalFailures        uint32  `json:"totalFailures"`
	ConsecutiveSuccesses uint32  `json:"consecutiveSuccesses"`
	ConsecutiveFailures  uint32  `json:"consecutiveFailures"`
	FailureRatio         float64 `json:"failureRatio"`
	LastUpdated          string  `json:"lastUpdated"`
	LastError            string  `json:"lastError,omitempty"`
}

// CBEvent records a circuit breaker state transition.
type CBEvent struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Timestamp string `json:"timestamp"`
	Error     string `json:"error,omitempty"`
}

// GetCBState reads the current CBState from the table.
func GetCBState(ctx context.Context, table db.Table) (*CBState, bool) {
	raw, ok := table.Get(ctx, cbKeyState)
	if !ok {
		return nil, false
	}
	return decodeCBState(raw)
}

// GetCBEvents reads the event history from the table.
func GetCBEvents(ctx context.Context, table db.Table) []CBEvent {
	raw, ok := table.Get(ctx, cbKeyEvents)
	if !ok {
		return nil
	}
	return decodeCBEvents(raw)
}

// newCBState builds a CBState from gobreaker counts and state.
func newCBState(state string, counts gobreaker.Counts) CBState {
	var ratio float64
	if counts.Requests > 0 {
		ratio = float64(counts.TotalFailures) / float64(counts.Requests)
	}
	return CBState{
		State:                state,
		Requests:             counts.Requests,
		TotalSuccesses:       counts.TotalSuccesses,
		TotalFailures:        counts.TotalFailures,
		ConsecutiveSuccesses: counts.ConsecutiveSuccesses,
		ConsecutiveFailures:  counts.ConsecutiveFailures,
		FailureRatio:         ratio,
		LastUpdated:          time.Now().UTC().Format(time.RFC3339),
	}
}

// decodeCBState converts a raw value to CBState.
// Handles both CBState (direct) and map[string]interface{} (after JSON round-trip in Redis).
func decodeCBState(raw any) (*CBState, bool) {
	switch v := raw.(type) {
	case CBState:
		return &v, true
	case *CBState:
		return v, true
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var s CBState
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, false
		}
		return &s, true
	default:
		return nil, false
	}
}

// decodeCBEvents converts a raw value to []CBEvent.
// Handles both []CBEvent (direct) and []interface{} (after JSON round-trip in Redis).
func decodeCBEvents(raw any) []CBEvent {
	switch v := raw.(type) {
	case []CBEvent:
		return v
	case []interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var events []CBEvent
		if err := json.Unmarshal(data, &events); err != nil {
			return nil
		}
		return events
	default:
		return nil
	}
}

// appendCBEvent appends an event and caps the list at cbMaxEvents.
func appendCBEvent(ctx context.Context, table db.Table, event CBEvent) {
	events := GetCBEvents(ctx, table)
	events = append(events, event)
	if len(events) > cbMaxEvents {
		events = events[len(events)-cbMaxEvents:]
	}
	table.Set(ctx, cbKeyEvents, events, 0)
}
