package process

import (
	"io"
	"os/exec"

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
}

func newProcess(ctx context.Context, executable string, args, env []string, workDir string, output io.Writer) (*process, error) {
	pr := process{
		ctx: ctx,
	}

	pr.cmd = &exec.Cmd{
		Env:         env,
		Args:        args,
		Dir:         workDir,
		Path:        executable,
		Stdout:      output,
		Stderr:      output,
		SysProcAttr: getSysProctAttr(),
	}

	if err := pr.cmd.Start(); err != nil {
		apexctx.GetLogger(ctx).WithError(err).Errorf("unable to start executable %s", pr.cmd.Path)
		return nil, err
	}
	apexctx.GetLogger(ctx).WithField("pid", pr.cmd.Process.Pid).Info("executable has been launched")
	return &pr, nil
}

func (p *process) Kill() error {
	return p.cmd.Process.Kill()
}
