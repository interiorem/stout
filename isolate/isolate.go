package isolate

import (
	"io"
	"sync"

	"golang.org/x/net/context"
)

const outputChunkSize = 1024

var (
	outputPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, outputChunkSize)
		},
	}
)

// GetPreallocatedOutputChunk returns []byte which should be used
// by boxes to collect output. Isolate is responsible for returning
// any chunk to the pool
func GetPreallocatedOutputChunk() []byte {
	return outputPool.Get().([]byte)
}

func backToPool(b []byte) {
	outputPool.Put(b)
}

type (
	decoderInit    func(io.Reader) Decoder
	dispatcherInit func(context.Context) Dispatcher

	Downstream interface {
		Reply(code int, args ...interface{}) error
	}

	ArgsUnpacker interface {
		Unpack(in interface{}, out ...interface{}) error
	}

	Box interface {
		Spool(ctx context.Context, name string, opts Profile) error
		Spawn(ctx context.Context, opts Profile, name, executable string, args, env map[string]string) (Process, error)
		Close() error
	}

	Process interface {
		Output() <-chan ProcessOutput
		Kill() error
	}

	ProcessOutput struct {
		Err  error
		Data []byte
	}

	Boxes map[string]Box

	BoxConfig map[string]interface{}
)

const (
	BoxesTag        = "isolate.boxes.tag"
	downstreamTag   = "downstream.tag"
	argsUnpackerTag = "args.unpacker.tag"
	decoderInitTag  = "decoder.init.tag"
)

func NotifyAbouStart(ch chan ProcessOutput) {
	ch <- ProcessOutput{Data: []byte(""), Err: nil}
}

func withArgsUnpacker(ctx context.Context, au ArgsUnpacker) context.Context {
	return context.WithValue(ctx, argsUnpackerTag, au)
}

func withDecoderInit(ctx context.Context, di decoderInit) context.Context {
	return context.WithValue(ctx, decoderInitTag, di)
}

func withDownstream(ctx context.Context, dw Downstream) context.Context {
	return context.WithValue(ctx, downstreamTag, dw)
}

func getBoxes(ctx context.Context) Boxes {
	val := ctx.Value(BoxesTag)
	box, ok := val.(Boxes)
	if !ok {
		panic("context.Context does not contain Box")
	}
	return box
}

func reply(ctx context.Context, code int, args ...interface{}) error {
	downstream, ok := ctx.Value(downstreamTag).(Downstream)
	if !ok {
		panic("context.Context does not contain downstream")
	}

	return downstream.Reply(code, args...)
}

func unpackArgs(ctx context.Context, in interface{}, out ...interface{}) error {
	return ctx.Value(argsUnpackerTag).(ArgsUnpacker).Unpack(in, out...)
}
