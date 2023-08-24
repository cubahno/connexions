package connexions

import (
    "github.com/stretchr/testify/assert"
    "reflect"
    "sync"
    "testing"
)

func TestReplaceState(t *testing.T) {
    t.Run("NewFrom", func(t *testing.T) {
        src := &ReplaceState{
            NamePath: []string{"foo", "bar"},
        }
        wanted := &ReplaceState{
            NamePath:                []string{"foo", "bar"},
            IsHeader:                false,
            ContentType:             "",
            stopCircularArrayTripOn: 0,
        }
        if got := src.NewFrom(src); !reflect.DeepEqual(got, wanted) {
            t.Errorf("NewFrom() = %v, expected %v", got, wanted)
        }
    })

    t.Run("WithName", func(t *testing.T) {
        src := &ReplaceState{
            NamePath: []string{"dice", "nice"},
        }

        res := src.WithName("mice")
        assert.Equal(t, []string{"dice", "nice", "mice"}, res.NamePath)
    })

    t.Run("WithElementIndex", func(t *testing.T) {
        src := &ReplaceState{}

        res := src.WithElementIndex(10)
        assert.Equal(t, 10, res.ElementIndex)
    })

    t.Run("WithElementIndexRacing", func(t *testing.T) {
        const numGoroutines = 1000
        const targetValue = 42

        // Create a shared ReplaceState
        state := &ReplaceState{}

        // Use a WaitGroup to wait for all goroutines to finish
        var wg sync.WaitGroup
        wg.Add(numGoroutines)

        // Start multiple goroutines that concurrently call WithElementIndex
        for i := 0; i < numGoroutines; i++ {
            go func() {
                defer wg.Done()
                state.WithElementIndex(targetValue)
            }()
        }

        // Wait for all goroutines to finish
        wg.Wait()

        if state.ElementIndex != targetValue || state.stopCircularArrayTripOn != targetValue+1 {
            t.Errorf("State not consistent: ElementIndex = %d, stopCircularArrayTripOn = %d", state.ElementIndex, state.stopCircularArrayTripOn)
        }
    })

    t.Run("WithHeader", func(t *testing.T) {
        src := &ReplaceState{}

        res := src.WithHeader()
        assert.True(t, res.IsHeader)
    })

    t.Run("WithContentType", func(t *testing.T) {
        src := &ReplaceState{}

        res := src.WithContentType("application/json")
        assert.Equal(t, "application/json", res.ContentType)
    })

    t.Run("IsCircularArrayTrip", func(t *testing.T) {
        src := &ReplaceState{}

        res := src.WithElementIndex(10)
        assert.True(t, res.IsCircularArrayTrip(10))
    })
}
