package isolate

import (
	"fmt"

	"golang.org/x/net/context"
)

const (
	spawnKill = 0

	replyKillOk    = 0
	replyKillError = 1
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
					if err := pr.Kill(); err != nil {
						reply(d.ctx, replyKillError, errKillError, err.Error())
						return
					}

					reply(d.ctx, replyKillOk, nil)
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
