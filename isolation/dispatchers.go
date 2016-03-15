package isolation

import (
	"fmt"

	"golang.org/x/net/context"
)

const (
	spool       = 0
	spoolCancel = 0

	replySpoolOk    = 0
	replySpoolError = 1

	spawn     = 1
	spawnKill = 0
)

type noneDispatch struct{}

func (d *noneDispatch) Handle(*message) (Dispatcher, error) {
	return d, fmt.Errorf("no transitions from NonDispatch")
}

type initialDispatch struct {
	ctx context.Context
}

func newInitialDispatch(ctx context.Context) Dispatcher {
	return &initialDispatch{
		ctx: ctx,
	}
}

func (d *initialDispatch) Handle(msg *message) (Dispatcher, error) {
	switch msg.Number {
	case spool:
		var (
			opts profile
			name string
		)

		if err := unpackArgs(d.ctx, msg.Args, &opts, &name); err != nil {
			return nil, err
		}

		if opts.Isolate.Type == "" {
			return nil, fmt.Errorf("corrupted profile: %v", opts)
		}

		box, ok := getIsolationBoxes(d.ctx)[opts.Isolate.Type]
		if !ok {
			return nil, fmt.Errorf("isolation type %s is not available", opts.Isolate.Type)
		}

		ctx, cancel := context.WithCancel(d.ctx)

		go func() {
			if err := box.Spool(ctx, name, opts); err != nil {
				reply(ctx, replySpoolError, [2]int{42, 42}, err.Error())
				return
			}
			// NOTE: make sure that nil is packed as []interface{}
			reply(ctx, replySpoolOk, nil)
		}()

		return newSpoolCancelationDispatch(ctx, cancel), nil

	case spawn:
		return nil, fmt.Errorf("spawn is not implemented")
	default:
		return nil, fmt.Errorf("unknown transition id: %d", msg.Number)
	}
}

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
		return &noneDispatch{}, nil
	default:
		return nil, fmt.Errorf("unknown transition id: %d", msg.Number)
	}
}
