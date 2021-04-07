package process

import (
	"io"
	"os/exec"
	"syscall"

	"golang.org/x/net/context"

	"github.com/interiorem/stout/pkg/log"
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
		log.G(ctx).WithError(err).Errorf("unable to start executable %s", pr.cmd.Path)
		return nil, err
	}

	log.G(ctx).WithField("pid", pr.cmd.Process.Pid).Info("executable has been launched")
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
