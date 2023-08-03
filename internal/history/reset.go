package history

import (
	"context"
	"time"
)

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
