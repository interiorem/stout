package process

import (
	"bufio"
	"io"
	"log"
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
		output:  make(chan isolation.ProcessOutput, 100),
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
		pr.cmd = &exec.Cmd{
			Env:  packedEnv,
			Args: packedArgs,
			Dir:  workDir,
			Path: executable,
		}

		log.Printf("starting executable %s", pr.cmd.Path)

		collector := func(r io.Reader) {
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				pr.output <- isolation.ProcessOutput{
					Data: []byte(scanner.Text()),
					Err:  nil,
				}
			}

			if err := scanner.Err(); err != nil {
				return
			}
		}

		// stdout
		log.Printf("attach stdout of %s", pr.cmd.Path)
		stdout, err := pr.cmd.StdoutPipe()
		if err != nil {
			log.Printf("unable to attach stdout of %s: %v", pr.cmd.Path, err)
			return
		}
		go collector(stdout)

		// stderr
		log.Printf("attach stderr of %s", pr.cmd.Path)
		stderr, err := pr.cmd.StderrPipe()
		if err != nil {
			log.Printf("unable to attach stderr of %s: %v", pr.cmd.Path, err)
			return
		}
		go collector(stderr)

		if err := pr.cmd.Start(); err != nil {
			log.Printf("unable to start executable %s: %v", pr.cmd.Path, err)
			return
		}

		log.Printf("executable %s has been launched", pr.cmd.Path)
		// NOTE: is it dangerous?
		pr.output <- isolation.ProcessOutput{
			Data: []byte(""),
			Err:  nil,
		}
		log.Printf("the notification about launching of %s has been sent", pr.cmd.Path)
		close(pr.started)
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
	return p.output
}
