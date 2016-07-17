// +build linux

package process

import (
	"syscall"
)

func getSysProctAttr() *syscall.SysProcAttr {
	attrs := &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGKILL,
	}

	return attrs
}
