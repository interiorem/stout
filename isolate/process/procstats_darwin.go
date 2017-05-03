// +build darwin,cgo

package process

//#include<libproc.h>
import "C"

import (
	"time"
	"unsafe"

	"github.com/noxiouz/stout/isolate/stats"
)

func getStatistics(pid int) stats.Statistics {
	var pti C.struct_proc_taskinfo
	ssize := C.proc_pidinfo(C.int(pid), C.PROC_PIDTASKINFO, 0, unsafe.Pointer(&pti), C.PROC_PIDTASKINFO_SIZE)
	if ssize != C.PROC_PIDTASKINFO_SIZE {
		return stats.Statistics{}
	}

	return stats.Statistics{
		TS:          time.Now(),
		MemoryRSS:   uint64(pti.pti_resident_size),
		MemoryVS:    uint64(pti.pti_virtual_size),
		CPUSysTime:  uint64(pti.pti_total_system),
		CPUUserTime: uint64(pti.pti_total_user),
	}
}
