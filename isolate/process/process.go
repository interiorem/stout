package process

import (
	"io"
	"os/exec"
	"syscall"

	"golang.org/x/net/context"

	apexctx "github.com/m0sth8/context"
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
		SysProcAttr: getSysProctAttr(),
	}
	// It's imposible to set io.Writer directly to Cmd, because of
	// https://github.com/golang/go/issues/13155
	stdErrRd, err := pr.cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	stdOutRd, err := pr.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	go io.Copy(output, stdErrRd)
	go io.Copy(output, stdOutRd)

	if err = pr.cmd.Start(); err != nil {
		apexctx.GetLogger(ctx).WithError(err).Errorf("unable to start executable %s", pr.cmd.Path)
		return nil, err
	}

	apexctx.GetLogger(ctx).WithField("pid", pr.cmd.Process.Pid).Info("executable has been launched")
	return &pr, nil
}

func (p *process) Kill() error {
	return killPg(p.cmd.Process.Pid)
}

func killPg(pgid int) error {
	if pgid > 0 {
		pgid = -pgid
	}

	return syscall.Kill(pgid, syscall.SIGKILL)
}
