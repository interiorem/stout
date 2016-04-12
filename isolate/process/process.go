package process

import (
	"bufio"
	"io"
	"os/exec"
	"path/filepath"
	"sync/atomic"

	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolate"
)

var (
	_ isolate.Process = &process{}
)

type process struct {
	ctx context.Context
	cmd *exec.Cmd

	started chan struct{}
	output  chan isolate.ProcessOutput
}

func newProcess(ctx context.Context, executable string, args, env map[string]string, workDir string) (isolate.Process, error) {
	pr := process{
		ctx:     ctx,
		started: make(chan struct{}),
		output:  make(chan isolate.ProcessOutput, 100),
	}

	packedEnv := make([]string, 0, len(env))
	for k, v := range env {
		packedEnv = append(packedEnv, k+"="+v)
	}

	packedArgs := make([]string, 1, len(args)*2+1)
	packedArgs[0] = filepath.Base(executable)
	for k, v := range args {
		packedArgs = append(packedArgs, k, v)
	}

	pr.cmd = &exec.Cmd{
		Env:  packedEnv,
		Args: packedArgs,
		Dir:  workDir,
		Path: executable,
	}

	isolate.GetLogger(ctx).Infof("starting executable %+v", pr.cmd)

	// sme is used to keep track an order of output channel
	var sem uint32

	collector := func(r io.Reader) {
		defer func() {
			if atomic.AddUint32(&sem, 1) == 2 {
				close(pr.output)
			}
		}()

		// NOTE: it's dangerous actually to collect data until \n
		// An app can harm cocaine by creating really LONG strings
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			body := scanner.Bytes()
			data := make([]byte, len(body), len(body)+1)
			copy(data, body)
			pr.output <- isolate.ProcessOutput{
				Data: append(data, '\n'),
				Err:  nil,
			}
		}

		if err := scanner.Err(); err != nil {
			pr.output <- isolate.ProcessOutput{
				Data: nil,
				Err:  err,
			}
			return
		}
	}

	// stdout
	isolate.GetLogger(ctx).Infof("attach stdout of %s", pr.cmd.Path)
	stdout, err := pr.cmd.StdoutPipe()
	if err != nil {
		isolate.GetLogger(ctx).Infof("unable to attach stdout of %s: %v", pr.cmd.Path, err)
		return nil, err
	}
	go collector(stdout)

	// stderr
	isolate.GetLogger(ctx).Infof("attach stderr of %s", pr.cmd.Path)
	stderr, err := pr.cmd.StderrPipe()
	if err != nil {
		isolate.GetLogger(ctx).Infof("unable to attach stderr of %s: %v", pr.cmd.Path, err)
		return nil, err
	}
	go collector(stderr)

	if err := pr.cmd.Start(); err != nil {
		isolate.GetLogger(ctx).Infof("unable to start executable %s: %v", pr.cmd.Path, err)
		return nil, err
	}

	isolate.GetLogger(ctx).Infof("executable %s has been launched", pr.cmd.Path)
	// NOTE: is it dangerous?
	isolate.NotifyAbouStart(pr.output)
	isolate.GetLogger(ctx).Infof("the notification about launching of %s has been sent", pr.cmd.Path)
	close(pr.started)

	return &pr, nil
}

func (p *process) Kill() error {
	select {
	case <-p.started:
		return p.cmd.Process.Kill()
	case <-p.ctx.Done():
		return nil
	}
}

func (p *process) Output() <-chan isolate.ProcessOutput {
	return p.output
}
