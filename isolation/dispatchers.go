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

	replySpawnWrite = 0
	replySpawnError = 1
	replySpawnClose = 2
)

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
		var (
			opts             profile
			name, executable string
			args, env        map[string]string
		)

		if err := unpackArgs(d.ctx, msg.Args, &opts, &name, &executable, &args, &env); err != nil {
			return nil, err
		}

		if opts.Isolate.Type == "" {
			return nil, fmt.Errorf("corrupted profile: %v", opts)
		}

		box, ok := getIsolationBoxes(d.ctx)[opts.Isolate.Type]
		if !ok {
			return nil, fmt.Errorf("isolation type %s is not available", opts.Isolate.Type)
		}

		pr, err := box.Spawn(d.ctx, name, executable, args, env)
		if err != nil {
			return nil, err
		}

		go func() {
			for {
				select {
				case output := <-pr.Output():
					if output.err != nil {
						reply(d.ctx, replySpawnError, [2]int{42, 42}, output.err.Error())
					} else {
						reply(d.ctx, replySpawnWrite, output.data)
					}
				case <-d.ctx.Done():
					return
				}
			}
		}()

		return newSpawnDispatch(d.ctx, pr), nil
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

type spawnDispatch struct {
	ctx     context.Context
	process process
}

func newSpawnDispatch(ctx context.Context, pr process) *spawnDispatch {
	return &spawnDispatch{
		ctx:     ctx,
		process: pr,
	}
}

func (d *spawnDispatch) Handle(msg *message) (Dispatcher, error) {
	switch msg.Number {
	case spawnKill:
		d.process.Kill()
		return &noneDispatch{}, nil
	default:
		return nil, fmt.Errorf("unknown transition id: %d", msg.Number)
	}
}

type noneDispatch struct{}

func (d *noneDispatch) Handle(*message) (Dispatcher, error) {
	return d, fmt.Errorf("no transitions from NonDispatch")
}
