package semaphore

type Semaphore interface {
	Acquire()
	Release()
}

// TODO: make it cancellable
type spawnSemaphore struct {
	sm chan struct{}
}

// New returns Semaphore with a given size
func New(size uint) Semaphore {
	return &spawnSemaphore{
		sm: make(chan struct{}, size),
	}
}

var _ Semaphore = &spawnSemaphore{}

func (s *spawnSemaphore) Acquire() {
	s.sm <- struct{}{}
}

func (s *spawnSemaphore) Release() {
	<-s.sm
}
