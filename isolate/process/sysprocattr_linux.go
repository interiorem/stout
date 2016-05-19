// +build linux

package process

import (
	"syscall"
)

func getSysProctAttr() *syscall.SysProcAttr {
	attrs := &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}

	return attrs
}
