package isolate

import (
	"fmt"

	"github.com/tinylib/msgp/msgp"

	"golang.org/x/net/context"
)

const (
	spoolCancel = 0

	replyCancelOk    = 0
	replyCancelError = 1
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

func (s *spoolCancelationDispatch) Handle(id int, r *msgp.Reader) (Dispatcher, error) {
	switch id {
	case spoolCancel:
		// Skip empty array
		r.Skip()
		// TODO: cancel only if I'm spooling
		s.cancel()
		// reply(s.ctx, replyCancelOk, nil)
		// NOTE: do not return an err on purpose
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown transition id: %d", id)
	}
}
