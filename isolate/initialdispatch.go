package isolate

import (
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"

	"golang.org/x/net/context"

	apexctx "github.com/m0sth8/context"
	"github.com/tinylib/msgp/msgp"
)

const (
	spool = 0

	replySpoolOk    = 0
	replySpoolError = 1

	spawn = 1

	replySpawnWrite = 0
	replySpawnError = 1
	replySpawnClose = 2
)

var (
	// ErrInvalidArgsNum should be returned if number of arguments is wrong
	ErrInvalidArgsNum = errors.New("invalid arguments number")
	_onSpoolArgsNum   = uint32(reflect.TypeOf(new(initialDispatch).onSpool).NumIn())
	_onSpawnArgsNum   = uint32(reflect.TypeOf(new(initialDispatch).onSpawn).NumIn())
)

func checkSize(num uint32, r *msgp.Reader) error {
	size, err := r.ReadArrayHeader()
	if err != nil {
		return err
	}

	if size != num {
		return ErrInvalidArgsNum
	}

	return nil
}

func readMapStrStr(r *msgp.Reader, mp map[string]string) (err error) {
	var sz uint32
	sz, err = r.ReadMapHeader()
	if err != nil {
		return err
	}

	for i := uint32(0); i < sz; i++ {
		var key string
		var val string
		key, err = r.ReadString()
		if err != nil {
			return err
		}
		val, err = r.ReadString()
		if err != nil {
			return err
		}
		mp[key] = val
	}

	return
}

type initialDispatch struct {
	ctx context.Context
}

func newInitialDispatch(ctx context.Context) Dispatcher {
	return &initialDispatch{ctx: ctx}
}

func (d *initialDispatch) Handle(id int64, r *msgp.Reader) (Dispatcher, error) {
	var err error
	switch id {
	case spool:
		var opts = make(Profile)
		var name string

		if err = checkSize(_onSpoolArgsNum, r); err != nil {
			return nil, err
		}

		if err = r.ReadMapStrIntf(opts); err != nil {
			return nil, err
		}

		if name, err = r.ReadString(); err != nil {
			return nil, err
		}

		return d.onSpool(opts, name)
	case spawn:
		var (
			opts             = make(Profile)
			name, executable string
			args             = make(map[string]string)
			env              = make(map[string]string)
		)
		if err = checkSize(_onSpawnArgsNum, r); err != nil {
			return nil, err
		}

		if err = r.ReadMapStrIntf(opts); err != nil {
			return nil, err
		}
		if name, err = r.ReadString(); err != nil {
			return nil, err
		}
		if executable, err = r.ReadString(); err != nil {
			return nil, err
		}

		if err = readMapStrStr(r, args); err != nil {
			return nil, err
		}

		if err = readMapStrStr(r, env); err != nil {
			return nil, err
		}

		return d.onSpawn(opts, name, executable, args, env)
	default:
		return nil, fmt.Errorf("unknown transition id: %d", id)
	}
}

func (d *initialDispatch) onSpool(opts Profile, name string) (Dispatcher, error) {
	isolateType := opts.Type()
	if isolateType == "" {
		err := fmt.Errorf("corrupted profile: %v", opts)
		apexctx.GetLogger(d.ctx).Error("unable to detect isolate type from a profile")
		reply(d.ctx, replySpawnError, errBadProfile, err.Error())
		return nil, err
	}

	box, ok := getBoxes(d.ctx)[isolateType]
	if !ok {
		apexctx.GetLogger(d.ctx).WithField("isolatetype", isolateType).Error("requested isolate type is not available")
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

func (d *initialDispatch) onSpawn(opts Profile, name, executable string, args, env map[string]string) (Dispatcher, error) {
	isolateType := opts.Type()
	if isolateType == "" {
		err := fmt.Errorf("corrupted profile: %v", opts)
		apexctx.GetLogger(d.ctx).Error("unable to detect isolate type from a profile")
		reply(d.ctx, replySpawnError, errBadProfile, err.Error())
		return nil, err
	}

	box, ok := getBoxes(d.ctx)[isolateType]
	if !ok {
		apexctx.GetLogger(d.ctx).WithField("isolatetype", isolateType).Error("requested isolate type is not available")
		err := fmt.Errorf("isolate type %s is not available", isolateType)
		reply(d.ctx, replySpawnError, errUnknownIsolate, err.Error())
		return nil, err
	}

	prCh := make(chan Process)
	flagKilled := uint32(0)
	// ctx will be passed to Spawn function
	// cancelSpawn will used by SpawnDispatch to cancel spawning
	ctx, cancelSpawn := context.WithCancel(d.ctx)
	go func() {
		defer close(prCh)

		spawnMeter.Mark(1)

		config := SpawnConfig{
			Opts:       opts,
			Name:       name,
			Executable: executable,
			Args:       args,
			Env:        env,
		}

		outputCollector := &OutputCollector{
			ctx:        d.ctx,
			flagKilled: &flagKilled,
		}
		pr, err := box.Spawn(ctx, config, outputCollector)
		if err != nil {
			apexctx.GetLogger(d.ctx).WithError(err).Error("unable to spawn")
			reply(d.ctx, replySpawnError, errSpawningFailed, err.Error())
			return
		}

		select {
		case prCh <- pr:
			// send process to SpawnDispatch
			// SpawnDispatch is resposible for killing pr now
		case <-ctx.Done():
			// SpawnDispatch has cancelled the spawning
			// Kill the process, set flagKilled to prevent trackOutput
			// sending duplicated messages, reply WithKillOk
			if atomic.CompareAndSwapUint32(&flagKilled, 0, 1) {
				if err := pr.Kill(); err != nil {
					reply(d.ctx, replyKillError, errKillError, err.Error())
					return
				}

				reply(d.ctx, replyKillOk, nil)
			}
		}
	}()

	return newSpawnDispatch(d.ctx, cancelSpawn, prCh, &flagKilled), nil
}

type OutputCollector struct {
	ctx context.Context

	flagKilled *uint32
	notified   uint32
}

func (o *OutputCollector) Write(p []byte) (int, error) {
	if atomic.LoadUint32(o.flagKilled) != 0 {
		return 0, nil
	}

	// if the first output comes earlier than Notify() is called
	if atomic.CompareAndSwapUint32(&o.notified, 0, 1) {
		reply(o.ctx, replySpawnWrite, notificationByte)
		if len(p) == 0 {
			return 0, nil
		}
	}

	reply(o.ctx, replySpawnWrite, p)
	return len(p), nil
}
