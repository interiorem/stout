package isolate

import (
	"sync"

	"context"
)

var logDiscard sync.Once

// IsCancelled checks whether context is cancelled
func IsCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
