package isolate

import (
	"fmt"

	"golang.org/x/net/context"
)

type spoolCancelationDispatch struct {
	ctx context.Context

	cancel context.CancelFunc
}

func newSpoolCancelationDispatch(ctx context.Context, cancel context.CancelFunc) *spoolCancelationDispatch {
	return &spoolCancelationDispatch{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *spoolCancelationDispatch) Handle(msg *message) (Dispatcher, error) {
	switch msg.Number {
	case spoolCancel:
		s.cancel()
		// NOTE: do not return an err on purpose
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown transition id: %d", msg.Number)
	}
}
