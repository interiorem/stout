package isolate

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

	errExternalCategory = 42
)

type initialDispatch struct {
	ctx context.Context
}

func newInitialDispatch(ctx context.Context) Dispatcher {
	return &initialDispatch{ctx: ctx}
}

func (d *initialDispatch) Handle(msg *message) (Dispatcher, error) {
	switch msg.Number {
	case spool:
		return d.onSpool(msg)
	case spawn:
		return d.onSpawn(msg)
	default:
		return nil, fmt.Errorf("unknown transition id: %d", msg.Number)
	}
}

func (d *initialDispatch) onSpool(msg *message) (Dispatcher, error) {
	var (
		opts Profile
		name string
	)

	if err := unpackArgs(d.ctx, msg.Args, &opts, &name); err != nil {
		GetLogger(d.ctx).WithError(err).Error("unable to unpack a message.args")
		reply(d.ctx, replySpawnError, errBadMsg, err.Error())
		return nil, err
	}

	isolateType := opts.Type()
	if isolateType == "" {
		err := fmt.Errorf("corrupted profile: %v", opts)
		GetLogger(d.ctx).Error("unable to detect isolate type from a profile")
		reply(d.ctx, replySpawnError, errBadProfile, err.Error())
		return nil, err
	}

	box, ok := getBoxes(d.ctx)[isolateType]
	if !ok {
		GetLogger(d.ctx).WithField("isolatetype", isolateType).Error("requested isolate type is not available")
		err := fmt.Errorf("isolate type %s is not available", isolateType)
		reply(d.ctx, replySpawnError, errUnknownIsolate, err.Error())
		return nil, err
	}

	ctx, cancel := context.WithCancel(d.ctx)

	go func() {
		if err := box.Spool(ctx, name, opts); err != nil {
			reply(ctx, replySpoolError, errSpoolingFailed, err.Error())
			return
		}
		// NOTE: make sure that nil is packed as []interface{}
		reply(ctx, replySpoolOk, nil)
	}()

	return newSpoolCancelationDispatch(ctx, cancel), nil
}

func (d *initialDispatch) onSpawn(msg *message) (Dispatcher, error) {
	var (
		opts             Profile
		name, executable string
		args, env        map[string]string
	)

	if err := unpackArgs(d.ctx, msg.Args, &opts, &name, &executable, &args, &env); err != nil {
		GetLogger(d.ctx).WithError(err).Error("unable to unpack a message.args")
		reply(d.ctx, replySpawnError, errBadMsg, err.Error())
		return nil, err
	}

	isolateType := opts.Type()
	if isolateType == "" {
		err := fmt.Errorf("corrupted profile: %v", opts)
		GetLogger(d.ctx).Error("unable to detect isolate type from a profile")
		reply(d.ctx, replySpawnError, errBadProfile, err.Error())
		return nil, err
	}

	box, ok := getBoxes(d.ctx)[isolateType]
	if !ok {
		GetLogger(d.ctx).WithField("isolatetype", isolateType).Error("requested isolate type is not available")
		err := fmt.Errorf("isolate type %s is not available", isolateType)
		reply(d.ctx, replySpawnError, errUnknownIsolate, err.Error())
		return nil, err
	}

	prCh := make(chan Process, 1)
	go func() {
		defer close(prCh)

		pr, err := box.Spawn(d.ctx, opts, name, executable, args, env)
		if err != nil {
			GetLogger(d.ctx).WithError(err).Error("unable to spawn")
			reply(d.ctx, replySpawnError, errSpawningFailed, err.Error())
			return
		}

		d.trackOutput(pr)
		prCh <- pr
	}()

	return newSpawnDispatch(d.ctx, prCh), nil
}

func (d *initialDispatch) trackOutput(pr Process) {
	for {
		select {
		case output, ok := <-pr.Output():
			if !ok {
				reply(d.ctx, replySpawnError, errOutputError, "output has been closed")
				return
			}

			if output.Err != nil {
				reply(d.ctx, replySpawnError, errOutputError, output.Err.Error())
			} else {
				reply(d.ctx, replySpawnWrite, output.Data)
			}
		case <-d.ctx.Done():
			return
		}
	}
}
