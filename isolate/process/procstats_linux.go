// +build linux

package process

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"syscall"
	"time"

	"github.com/noxiouz/stout/isolate/stats"
)

const (
	// indexes of fields in /proc/[pid]/stat
	stime = 15
	utime = 16
	vsize = 24
	rss   = 25
)

var pageSize = uint64(syscall.Getpagesize())

func readUint64(a []byte) uint64 {
	u, _ := strconv.ParseUint(string(a), 10, 64)
	return u
}

func getStatistics(pid int) stats.Statistics {
	data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return stats.Statistics{}
	}

	fields := bytes.Fields(data)
	s := stats.Statistics{
		TS:          time.Now(),
		MemoryRSS:   readUint64(fields[rss]) / pageSize,
		MemoryVS:    readUint64(fields[vsize]),
		CPUSysTime:  readUint64(fields[stime]),
		CPUUserTime: readUint64(fields[utime]),
	}

	return s
}
