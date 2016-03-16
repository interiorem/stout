package process

import (
	"bufio"
	"io"
	"os/exec"

	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolation"
)

var (
	_ isolation.Process = &process{}
)

type process struct {
	ctx context.Context
	cmd *exec.Cmd

	started chan struct{}
	output  chan isolation.ProcessOutput
}

func newProcess(ctx context.Context, executable string, args, env map[string]string, workDir string) (isolation.Process, error) {
	pr := process{
		ctx:     ctx,
		started: make(chan struct{}),
		output:  make(chan isolation.ProcessOutput),
	}

	packedEnv := make([]string, 0, len(env))
	for k, v := range env {
		packedEnv = append(packedEnv, k+"="+v)
	}

	packedArgs := make([]string, 0, len(args)*2)
	for k, v := range args {
		packedArgs = append(packedArgs, k, v)
	}

	go func() {
		cmd := &exec.Cmd{
			Env:  packedEnv,
			Args: packedArgs,
			Dir:  workDir,
			Path: executable,
		}

		pr.cmd = cmd
		if err := cmd.Start(); err != nil {
			return
		}

		close(pr.output)
	}()

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

func (p *process) Output() <-chan isolation.ProcessOutput {
	select {
	case <-p.started:
	case <-p.ctx.Done():
		return p.output
	}

	collector := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			p.output <- isolation.ProcessOutput{
				Data: []byte(scanner.Text()),
				Err:  nil,
			}
		}

		if err := scanner.Err(); err != nil {
			return
		}
	}

	// stdout
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return p.output
	}
	collector(stdout)
	// stderr
	stderr, err := p.cmd.StderrPipe()
	if err != nil {
		return p.output
	}
	collector(stderr)

	return p.output
}
