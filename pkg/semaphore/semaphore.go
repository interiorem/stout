package semaphore

import "golang.org/x/net/context"

type Semaphore interface {
	Acquire(ctx context.Context) error
	Release()
}

// TODO: make it cancellable
type spawnSemaphore struct {
	sm chan struct{}
}

// New returns Semaphore with a given size
func New(size uint) Semaphore {
	if size == 0 {
		panic("Semaphore: size must be positive")
	}
	return &spawnSemaphore{
		sm: make(chan struct{}, size),
	}
}

var _ Semaphore = &spawnSemaphore{}

func (s *spawnSemaphore) Acquire(ctx context.Context) error {
	select {
	case s.sm <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *spawnSemaphore) Release() {
	<-s.sm
}
