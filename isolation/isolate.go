package isolation

import (
	"io"

	"golang.org/x/net/context"
)

type (
	decoderInit    func(io.Reader) Decoder
	dispatcherInit func(context.Context) (Dispatcher, error)

	Downstream interface {
		Reply(code int, args ...interface{}) error
	}

	ArgsUnpacker interface {
		Unpack(in interface{}, out ...interface{}) error
	}

	IsolationBox interface {
		Spool(ctx context.Context, name string, opts profile) error
		Spawn(ctx context.Context, name, executable string, args, env map[string]string) (Process, error)
	}

	Process interface {
		Output() <-chan ProcessOutput
		Kill() error
	}

	ProcessOutput struct {
		err  error
		data []byte
	}

	IsolationBoxes map[string]IsolationBox
)

const (
	IsolationBoxesTag = "isolation.boxes.tag"
	downstreamTag     = "downstream.tag"
	argsUnpackerTag   = "args.unpacker.tag"
	decoderInitTag    = "decoder.init.tag"
)

func withArgsUnpacker(ctx context.Context, au ArgsUnpacker) context.Context {
	return context.WithValue(ctx, argsUnpackerTag, au)
}

func withDecoderInit(ctx context.Context, di decoderInit) context.Context {
	return context.WithValue(ctx, decoderInitTag, di)
}

func withDownstream(ctx context.Context, dw Downstream) context.Context {
	return context.WithValue(ctx, downstreamTag, dw)
}

func getIsolationBoxes(ctx context.Context) IsolationBoxes {
	val := ctx.Value(IsolationBoxesTag)
	box, ok := val.(IsolationBoxes)
	if !ok {
		panic("context.Context does not contain IsolationBox")
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
