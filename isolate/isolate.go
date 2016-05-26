package isolate

import (
	"io"

	"golang.org/x/net/context"
)

type SpawnConfig struct {
	Opts       Profile
	Name       string
	Executable string
	Args       map[string]string
	Env        map[string]string
}

type (
	dispatcherInit func(context.Context) Dispatcher

	Downstream interface {
		Reply(code int64, args ...interface{}) error
	}

	ArgsUnpacker interface {
		Unpack(in interface{}, out ...interface{}) error
	}

	Box interface {
		Spool(ctx context.Context, name string, opts Profile) error
		Spawn(ctx context.Context, config SpawnConfig, output io.Writer) (Process, error)
		Close() error
	}

	Process interface {
		Kill() error
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

var (
	notificationByte = []byte("")
)

func NotifyAbouStart(wr io.Writer) {
	wr.Write(notificationByte)
}

func withArgsUnpacker(ctx context.Context, au ArgsUnpacker) context.Context {
	return context.WithValue(ctx, argsUnpackerTag, au)
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

func reply(ctx context.Context, code int64, args ...interface{}) error {
	downstream, ok := ctx.Value(downstreamTag).(Downstream)
	if !ok {
		panic("context.Context does not contain downstream")
	}

	return downstream.Reply(code, args...)
}

func unpackArgs(ctx context.Context, in interface{}, out ...interface{}) error {
	return ctx.Value(argsUnpackerTag).(ArgsUnpacker).Unpack(in, out...)
}
