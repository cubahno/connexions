package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/mockzilla/connexions/v2/pkg/db"
	assert2 "github.com/stretchr/testify/assert"
)

func TestDecodeCBState(t *testing.T) {
	assert := assert2.New(t)

	t.Run("direct CBState value", func(t *testing.T) {
		input := CBState{State: "closed", Requests: 10, TotalFailures: 2, FailureRatio: 0.2}
		result, ok := decodeCBState(input)
		assert.True(ok)
		assert.Equal("closed", result.State)
		assert.Equal(uint32(10), result.Requests)
		assert.Equal(0.2, result.FailureRatio)
	})

	t.Run("pointer to CBState", func(t *testing.T) {
		input := &CBState{State: "open", TotalFailures: 5}
		result, ok := decodeCBState(input)
		assert.True(ok)
		assert.Equal("open", result.State)
		assert.Equal(uint32(5), result.TotalFailures)
	})

	t.Run("map from JSON round-trip", func(t *testing.T) {
		input := map[string]interface{}{
			"state":                "half-open",
			"requests":             float64(7),
			"totalSuccesses":       float64(4),
			"totalFailures":        float64(3),
			"consecutiveSuccesses": float64(1),
			"consecutiveFailures":  float64(0),
			"failureRatio":         0.42,
			"lastUpdated":          "2025-01-01T00:00:00Z",
		}
		result, ok := decodeCBState(input)
		assert.True(ok)
		assert.Equal("half-open", result.State)
		assert.Equal(uint32(7), result.Requests)
		assert.Equal(uint32(4), result.TotalSuccesses)
		assert.Equal(uint32(3), result.TotalFailures)
		assert.Equal(0.42, result.FailureRatio)
	})

	t.Run("unsupported type returns false", func(t *testing.T) {
		result, ok := decodeCBState("not-a-state")
		assert.False(ok)
		assert.Nil(result)
	})
}

func TestDecodeCBEvents(t *testing.T) {
	assert := assert2.New(t)

	t.Run("direct []CBEvent", func(t *testing.T) {
		input := []CBEvent{
			{From: "closed", To: "open", Timestamp: "2025-01-01T00:00:00Z"},
		}
		result := decodeCBEvents(input)
		assert.Len(result, 1)
		assert.Equal("closed", result[0].From)
		assert.Equal("open", result[0].To)
	})

	t.Run("[]interface{} from JSON round-trip", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"from":      "closed",
				"to":        "open",
				"timestamp": "2025-01-01T00:00:00Z",
			},
			map[string]interface{}{
				"from":      "open",
				"to":        "half-open",
				"timestamp": "2025-01-01T01:00:00Z",
			},
		}
		result := decodeCBEvents(input)
		assert.Len(result, 2)
		assert.Equal("closed", result[0].From)
		assert.Equal("open", result[0].To)
		assert.Equal("open", result[1].From)
		assert.Equal("half-open", result[1].To)
	})

	t.Run("unsupported type returns nil", func(t *testing.T) {
		result := decodeCBEvents("not-events")
		assert.Nil(result)
	})

	t.Run("nil returns nil", func(t *testing.T) {
		result := decodeCBEvents(nil)
		assert.Nil(result)
	})
}

func TestGetCBState(t *testing.T) {
	assert := assert2.New(t)

	t.Run("returns state from table", func(t *testing.T) {
		storage := db.NewStorage(nil)
		defer storage.Close()
		database := storage.NewDB("test", 100*time.Second)
		table := database.Table("circuit-breaker")

		ctx := context.Background()
		state := CBState{State: "closed", Requests: 5}
		table.Set(ctx, cbKeyState, state, 0)

		result, ok := GetCBState(ctx, table)
		assert.True(ok)
		assert.Equal("closed", result.State)
		assert.Equal(uint32(5), result.Requests)
	})

	t.Run("returns false when not set", func(t *testing.T) {
		storage := db.NewStorage(nil)
		defer storage.Close()
		database := storage.NewDB("test", 100*time.Second)
		table := database.Table("circuit-breaker")

		result, ok := GetCBState(context.Background(), table)
		assert.False(ok)
		assert.Nil(result)
	})
}

func TestGetCBEvents(t *testing.T) {
	assert := assert2.New(t)

	t.Run("returns events from table", func(t *testing.T) {
		storage := db.NewStorage(nil)
		defer storage.Close()
		database := storage.NewDB("test", 100*time.Second)
		table := database.Table("circuit-breaker")

		ctx := context.Background()
		events := []CBEvent{
			{From: "closed", To: "open", Timestamp: "2025-01-01T00:00:00Z"},
		}
		table.Set(ctx, cbKeyEvents, events, 0)

		result := GetCBEvents(ctx, table)
		assert.Len(result, 1)
		assert.Equal("closed", result[0].From)
	})

	t.Run("returns nil when not set", func(t *testing.T) {
		storage := db.NewStorage(nil)
		defer storage.Close()
		database := storage.NewDB("test", 100*time.Second)
		table := database.Table("circuit-breaker")

		result := GetCBEvents(context.Background(), table)
		assert.Nil(result)
	})
}

func TestAppendCBEvent(t *testing.T) {
	assert := assert2.New(t)

	t.Run("appends event to empty table", func(t *testing.T) {
		storage := db.NewStorage(nil)
		defer storage.Close()
		database := storage.NewDB("test", 100*time.Second)
		table := database.Table("circuit-breaker")

		ctx := context.Background()
		appendCBEvent(ctx, table, CBEvent{From: "closed", To: "open", Timestamp: "2025-01-01T00:00:00Z"})

		events := GetCBEvents(ctx, table)
		assert.Len(events, 1)
		assert.Equal("closed", events[0].From)
	})

	t.Run("appends event to existing events", func(t *testing.T) {
		storage := db.NewStorage(nil)
		defer storage.Close()
		database := storage.NewDB("test", 100*time.Second)
		table := database.Table("circuit-breaker")

		ctx := context.Background()
		table.Set(ctx, cbKeyEvents, []CBEvent{
			{From: "closed", To: "open", Timestamp: "2025-01-01T00:00:00Z"},
		}, 0)

		appendCBEvent(ctx, table, CBEvent{From: "open", To: "half-open", Timestamp: "2025-01-01T01:00:00Z"})

		events := GetCBEvents(ctx, table)
		assert.Len(events, 2)
		assert.Equal("open", events[1].From)
		assert.Equal("half-open", events[1].To)
	})

	t.Run("caps events at max", func(t *testing.T) {
		storage := db.NewStorage(nil)
		defer storage.Close()
		database := storage.NewDB("test", 100*time.Second)
		table := database.Table("circuit-breaker")

		ctx := context.Background()

		// Seed with cbMaxEvents events
		existing := make([]CBEvent, cbMaxEvents)
		for i := range existing {
			existing[i] = CBEvent{From: "closed", To: "open", Timestamp: "2025-01-01T00:00:00Z"}
		}
		table.Set(ctx, cbKeyEvents, existing, 0)

		// Append one more
		appendCBEvent(ctx, table, CBEvent{From: "open", To: "half-open", Timestamp: "2025-12-31T23:59:59Z"})

		events := GetCBEvents(ctx, table)
		assert.Len(events, cbMaxEvents)
		// The last event should be the one we just appended
		assert.Equal("half-open", events[cbMaxEvents-1].To)
		assert.Equal("open", events[cbMaxEvents-1].From)
	})
}
