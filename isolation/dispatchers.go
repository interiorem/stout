package isolation

import (
	"fmt"
	"log"

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
			opts Profile
			name string
		)

		log.Printf("initialDispatch.Handle.Spool().Args. Profile `%+v`, appname `%s`",
			msg.Args[0], msg.Args[1])
		if err := unpackArgs(d.ctx, msg.Args, &opts, &name); err != nil {
			reply(d.ctx, replySpoolError, [2]int{42, 42}, fmt.Sprintf("unbale to unpack args: %v", err))
			return nil, err
		}

		isolationType := opts.Type()
		if isolationType == "" {
			err := fmt.Errorf("the profile does not have `type` option: %v", opts)
			reply(d.ctx, replySpoolError, [2]int{42, 42}, err.Error())
			return nil, err
		}

		box, ok := getBoxes(d.ctx)[isolationType]
		if !ok {
			err := fmt.Errorf("isolation type %s is not available", isolationType)
			reply(d.ctx, replySpoolError, [2]int{42, 42}, err.Error())
			return nil, err
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
			opts             Profile
			name, executable string
			args, env        map[string]string
		)

		if err := unpackArgs(d.ctx, msg.Args, &opts, &name, &executable, &args, &env); err != nil {
			return nil, err
		}

		isolationType := opts.Type()
		if isolationType == "" {
			return nil, fmt.Errorf("corrupted profile: %v", opts)
		}

		box, ok := getBoxes(d.ctx)[isolationType]
		if !ok {
			return nil, fmt.Errorf("isolation type %s is not available", isolationType)
		}

		pr, err := box.Spawn(d.ctx, name, executable, args, env)
		if err != nil {
			log.Printf("initialDispatch.Handle.Spawn(): unable to spawn %v", err)
			return nil, err
		}

		go func() {
			for {
				select {
				case output := <-pr.Output():
					if output.Err != nil {
						reply(d.ctx, replySpawnError, [2]int{42, 42}, output.Err.Error())
					} else {
						reply(d.ctx, replySpawnWrite, output.Data)
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
	process Process
}

func newSpawnDispatch(ctx context.Context, pr Process) *spawnDispatch {
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
