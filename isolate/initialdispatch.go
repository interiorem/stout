package isolate

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"syscall"

	"golang.org/x/net/context"

	"github.com/noxiouz/stout/pkg/log"
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

	containersMetrics = 2

	replyMetricsOk = 0
	replyMetricsError = 1
	replyMetricsClose = 2
)

const expectedUuidsCount = 32

var (
	// ErrInvalidArgsNum should be returned if number of arguments is wrong
	ErrInvalidArgsNum = errors.New("invalid arguments number")
	_onSpoolArgsNum   = uint32(reflect.TypeOf(new(initialDispatch).onSpool).NumIn())
	_onSpawnArgsNum   = uint32(reflect.TypeOf(new(initialDispatch).onSpawn).NumIn())
	_onMetricsArgsNum = uint32(reflect.TypeOf(new(initialDispatch).onContainersMetrics).NumIn())
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


func readSliceString(r *msgp.Reader) (uuids []string, err error) {
	var sz uint32

	sz, err = r.ReadArrayHeader()
	if err != nil {
		return nil, err
	}

	for i := uint32(0); i < sz; i++ {
		var u string
		if u, err = r.ReadString(); err == nil {
			uuids = append(uuids, u)
		} else {
			return nil, err
		}
	}

	return
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
	ctx    context.Context
	stream ResponseStream
}

func newInitialDispatch(ctx context.Context, stream ResponseStream) Dispatcher {
	return &initialDispatch{
		ctx:    ctx,
		stream: stream,
	}
}

func (d *initialDispatch) Handle(id uint64, r *msgp.Reader) (Dispatcher, error) {
	var err error

	switch id {
	case spool:
		var rawProfile = newCocaineProfile()
		var name string

		if err = checkSize(_onSpoolArgsNum, r); err != nil {
			return nil, err
		}

		var nt msgp.Type
		nt, err = r.NextType()
		if err != nil {
			return nil, err
		}
		if nt != msgp.MapType {
			return nil, fmt.Errorf("profile must be %s not %s", msgp.MapType, nt)
		}

		// NOTE: Copy profile as is w/o decoding
		_, err = r.CopyNext(rawProfile)
		if err != nil {
			return nil, err
		}

		if name, err = r.ReadString(); err != nil {
			return nil, err
		}

		return d.onSpool(rawProfile, name)
	case spawn:
		var (
			rawProfile       = newCocaineProfile()
			name, executable string
			args             = make(map[string]string)
			env              = make(map[string]string)
		)
		if err = checkSize(_onSpawnArgsNum, r); err != nil {
			return nil, err
		}

		var nt msgp.Type
		nt, err = r.NextType()
		if err != nil {
			return nil, err
		}
		if nt != msgp.MapType {
			return nil, fmt.Errorf("profile must be %s not %s", msgp.MapType, nt)
		}

		// NOTE: Copy profile as is w/o decoding
		_, err = r.CopyNext(rawProfile)
		if err != nil {
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

		return d.onSpawn(rawProfile, name, executable, args, env)
	case containersMetrics:
		if err = checkSize(_onMetricsArgsNum, r); err != nil {
			log.G(d.ctx).Errorf("wrong args count for slot %d", id)
			return nil, err
		}

		uuids := make([]string, 0, expectedUuidsCount)
		uuids, err = readSliceString(r)
		if err != nil {
			log.G(d.ctx).Errorf("wrong containersMetrics request framing: %v", err)
			return nil, err
		}

		return d.onContainersMetrics(uuids)
	default:
		return nil, fmt.Errorf("unknown transition id: %d", id)
	}
}

func (d *initialDispatch) onSpool(opts *cocaineProfile, name string) (Dispatcher, error) {
	isolateType, err := opts.Type()
	if err != nil {
		log.G(d.ctx).WithError(err).Error("unable to detect isolate type from a profile")
		err := fmt.Errorf("corrupted profile: %v", opts)
		d.stream.Error(d.ctx, replySpoolError, errBadProfile, err.Error())
		return nil, err
	}

	box, ok := getBoxes(d.ctx)[isolateType]
	if !ok {
		log.G(d.ctx).WithField("isolatetype", isolateType).Error("requested isolate type is not available")
		err := fmt.Errorf("isolate type %s is not available", isolateType)
		d.stream.Error(d.ctx, replySpoolError, errUnknownIsolate, err.Error())
		return nil, err
	}

	ctx, cancel := context.WithCancel(d.ctx)

	go func() {
		if err := box.Spool(ctx, name, opts); err != nil {
			d.stream.Error(ctx, replySpoolError, errSpoolingFailed, err.Error())
			return
		}
		// NOTE: make sure that nil is packed as []interface{}
		d.stream.Close(ctx, replySpoolOk)
	}()

	return newSpoolCancelationDispatch(ctx, cancel, d.stream), nil
}

func (d *initialDispatch) onSpawn(opts *cocaineProfile, name, executable string, args, env map[string]string) (Dispatcher, error) {
	isolateType, err := opts.Type()
	if err != nil {
		log.G(d.ctx).WithError(err).Error("unable to detect isolate type from a profile")
		err := fmt.Errorf("corrupted profile: %v", opts)
		d.stream.Error(d.ctx, replySpawnError, errBadProfile, err.Error())
		return nil, err
	}

	box, ok := getBoxes(d.ctx)[isolateType]
	if !ok {
		log.G(d.ctx).WithField("isolatetype", isolateType).Error("requested isolate type is not available")
		err := fmt.Errorf("isolate type %s is not available", isolateType)
		d.stream.Error(d.ctx, replySpawnError, errUnknownIsolate, err.Error())
		return nil, err
	}

	log.G(d.ctx).Debugf("onSpawn() Profile Dump: %s", opts)

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
			ctx:    d.ctx,
			stream: d.stream,
		}
		pr, err := box.Spawn(ctx, config, outputCollector)
		if err != nil {
			switch err {
			case ErrSpawningCancelled, context.Canceled:
				spawnCancelledMeter.Mark(1)
			case syscall.EAGAIN:
				d.stream.Error(d.ctx, replySpawnError, errSpawnEAGAIN, err.Error())
			default:
				log.G(d.ctx).WithError(err).Error("unable to spawn")
				d.stream.Error(d.ctx, replySpawnError, errSpawningFailed, err.Error())
			}
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
					d.stream.Error(d.ctx, replyKillError, errKillError, err.Error())
					return
				}

				d.stream.Close(d.ctx, replyKillOk)
			}
		}
	}()

	return newSpawnDispatch(d.ctx, cancelSpawn, prCh, &flagKilled, d.stream), nil
}

func (d *initialDispatch) onContainersMetrics(uuidsQuery []string) (Dispatcher, error) {

	log.G(d.ctx).Debugf("onContainersMetrics() Uuids query: %b", uuidsQuery)

	sendMetricsFunc := func(metrics MetricsResponse) {
		var (
			buf bytes.Buffer
			err error
		)

		if d == nil {
			log.G(d.ctx).Error("strange: dispatch is `nil`")
			return
		}

		if err = msgp.Encode(&buf, &metrics); err != nil {
			log.G(d.ctx).WithError(err).Error("unable to encode containers metrics response")
			d.stream.Error(d.ctx, replyMetricsError, errMarshallingError, err.Error())
		}

		if err = d.stream.WriteMessage(d.ctx, replyMetricsOk, buf.Bytes()); err != nil {
			log.G(d.ctx).WithError(err).Error("unable to send containers metrics")
			d.stream.Error(d.ctx, replyMetricsError, errContainerMetricsFailed, err.Error())
		}

		log.G(d.ctx).Debug("containers metrics have been sent to runtime")
	}

	go func() {
		//
		// TODO:
		//  - reduce complexity
		//  - log execution time
		//
		boxes := getBoxes(d.ctx)
		boxesSize := len(boxes)
		metricsResponse := make(MetricsResponse, len(uuidsQuery))
		queryResCh := make(chan []MarkedContainerMetrics)

		for _, b := range boxes {
			go func(b Box) {
				queryResCh <- b.QueryMetrics(uuidsQuery)
			}(b)
		}

		for i := 0; i < boxesSize; i++ {
			for _, m := range <- queryResCh {
				metricsResponse[m.uuid] = m.m
			}
		}

		sendMetricsFunc(metricsResponse)
	}()

	return nil, nil
}

type OutputCollector struct {
	ctx context.Context

	stream ResponseStream

	notified uint32
}

func (o *OutputCollector) Write(p []byte) (int, error) {
	// if the first output comes earlier than Notify() is called
	if atomic.CompareAndSwapUint32(&o.notified, 0, 1) {
		o.stream.Write(o.ctx, replySpawnWrite, notificationByte)
		if len(p) == 0 {
			return 0, nil
		}
	}

	o.stream.Write(o.ctx, replySpawnWrite, p)
	return len(p), nil
}
