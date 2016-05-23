package isolate

import (
	"fmt"
	"sync/atomic"

	"golang.org/x/net/context"
)

const (
	spawnKill = 0

	replyKillOk    = 0
	replyKillError = 1
)

type spawnDispatch struct {
	ctx         context.Context
	cancelSpawn context.CancelFunc
	killed      *uint32
	process     <-chan Process
}

func newSpawnDispatch(ctx context.Context, cancelSpawn context.CancelFunc, prCh <-chan Process, flagKilled *uint32) *spawnDispatch {
	return &spawnDispatch{
		ctx:         ctx,
		cancelSpawn: cancelSpawn,
		killed:      flagKilled,
		process:     prCh,
	}
}

func (d *spawnDispatch) Handle(msg *message) (Dispatcher, error) {
	switch msg.Number {
	case spawnKill:
		// There are 3 cases:
		// * If the process has been spawned - kill it
		// * if the process has not been spawned yet - cancel it
		//		It's not our repsonsibility to clean up resources and kill anything
		// * if ctx has been cancelled - exit
		go func() {
			select {
			case pr, ok := <-d.process:
				if ok {
					if atomic.CompareAndSwapUint32(d.killed, 0, 1) {
						killMeter.Mark(1)
						if err := pr.Kill(); err != nil {
							reply(d.ctx, replyKillError, errKillError, err.Error())
							return
						}

						reply(d.ctx, replyKillOk, nil)
					}
				}
			case <-d.ctx.Done():
			default:
				// cancel spawning process
				spawnCancelMeter.Mark(1)
				d.cancelSpawn()
			}
		}()
		// NOTE: do not return an err on purpose
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown transition id: %d", msg.Number)
	}
}
