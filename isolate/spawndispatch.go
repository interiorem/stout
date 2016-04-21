package isolate

import (
	"fmt"

	"golang.org/x/net/context"
)

type spawnDispatch struct {
	ctx     context.Context
	process <-chan Process
}

func newSpawnDispatch(ctx context.Context, prCh <-chan Process) *spawnDispatch {
	return &spawnDispatch{
		ctx:     ctx,
		process: prCh,
	}
}

func (d *spawnDispatch) Handle(msg *message) (Dispatcher, error) {
	switch msg.Number {
	case spawnKill:
		go func() {
			select {
			case pr, ok := <-d.process:
				if ok {
					pr.Kill()
				}
			case <-d.ctx.Done():
			}
		}()
		// NOTE: do not return an err on purpose
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown transition id: %d", msg.Number)
	}
}
