package isolation

import (
	"golang.org/x/net/context"
)

func IsCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
