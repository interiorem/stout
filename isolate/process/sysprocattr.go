// +build !linux

package process

import (
	"syscall"
)

func getSysProctAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
