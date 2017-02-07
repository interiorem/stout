package isolate

import (
	"io"

	"context"
)

type SpawnConfig struct {
	Opts       RawProfile
	Name       string
	Executable string
	Args       map[string]string
	Env        map[string]string
}

type (
	dispatcherInit func(context.Context, ResponseStream) Dispatcher

	Box interface {
		Spool(ctx context.Context, name string, opts RawProfile) error
		Spawn(ctx context.Context, config SpawnConfig, output io.Writer) (Process, error)
		Close() error
	}

	ResponseStream interface {
		Write(ctx context.Context, num uint64, data []byte) error
		Error(ctx context.Context, num uint64, code [2]int, msg string) error
		Close(ctx context.Context, num uint64) error
	}

	Process interface {
		Kill() error
	}

	Boxes map[string]Box

	BoxConfig map[string]interface{}
)

const BoxesTag = "isolate.boxes.tag"

var (
	notificationByte = []byte("")
)

func NotifyAboutStart(wr io.Writer) {
	wr.Write(notificationByte)
}

func getBoxes(ctx context.Context) Boxes {
	val := ctx.Value(BoxesTag)
	box, ok := val.(Boxes)
	if !ok {
		panic("context.Context does not contain Box")
	}
	return box
}
