package isolate

import (
	"sync"

	"golang.org/x/net/context"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
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

func GetLogger(ctx context.Context) log.Interface {
	logger, ok := ctx.Value("logger").(log.Interface)
	if ok {
		return logger
	}

	logDiscard.Do(func() {
		if log.Log.Handler == nil {
			log.Log.Handler = text.Default
		}
	})
	return log.Log
}
