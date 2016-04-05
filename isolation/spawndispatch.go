package isolation

import (
	"fmt"

	"golang.org/x/net/context"
)

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
