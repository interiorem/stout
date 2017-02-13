package log

import (
	"github.com/apex/log"
	"golang.org/x/net/context"
)

type Logger interface {
	log.Interface
}

var (
	// G is a symlink to GetLogger
	G = GetLogger

	// L is a global logger
	L = log.Log
)

type loggerKey struct{}

func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func GetLogger(ctx context.Context) Logger {
	value := ctx.Value(loggerKey{})
	logger, ok := value.(Logger)
	if ok {
		return logger
	}

	return L
}
