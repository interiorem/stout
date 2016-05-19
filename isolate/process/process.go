package process

import (
	"bufio"
	"io"
	"os/exec"
	"path/filepath"
	"sync/atomic"

	"golang.org/x/net/context"

	apexctx "github.com/m0sth8/context"
	"github.com/noxiouz/stout/isolate"
)

var (
	_ isolate.Process = &process{}
)

type process struct {
	ctx context.Context
	cmd *exec.Cmd

	output chan isolate.ProcessOutput
}

func newProcess(ctx context.Context, executable string, args, env map[string]string, workDir string) (*process, error) {
	pr := process{
		ctx:    ctx,
		output: make(chan isolate.ProcessOutput, 100),
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
		Env:         packedEnv,
		Args:        packedArgs,
		Dir:         workDir,
		Path:        executable,
		SysProcAttr: getSysProctAttr(),
	}

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
	stdout, err := pr.cmd.StdoutPipe()
	if err != nil {
		apexctx.GetLogger(ctx).WithError(err).Errorf("unable to attach stdout of %s", pr.cmd.Path)
		return nil, err
	}

	// stderr
	stderr, err := pr.cmd.StderrPipe()
	if err != nil {
		apexctx.GetLogger(ctx).WithError(err).Errorf("unable to attach stderr of %s", pr.cmd.Path)
		return nil, err
	}

	if err := pr.cmd.Start(); err != nil {
		apexctx.GetLogger(ctx).WithError(err).Errorf("unable to start executable %s", pr.cmd.Path)
		stdout.Close()
		stderr.Close()
		return nil, err
	}

	// NOTE: is it dangerous?
	isolate.NotifyAbouStart(pr.output)
	go collector(stdout)
	go collector(stderr)
	apexctx.GetLogger(ctx).WithField("pid", pr.cmd.Process.Pid).Info("executable has been launched")

	return &pr, nil
}

func (p *process) Kill() error {
	return p.cmd.Process.Kill()
}

func (p *process) Output() <-chan isolate.ProcessOutput {
	return p.output
}
