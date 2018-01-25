// TODO:
//  - log timings
//
package process

import (
	"bytes"
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/pkg/log"
)

const clockTicks      = 100 // sysconf(_SC_CLK_TCK)

var (
	pageSize        = uint64(syscall.Getpagesize())
	spacesRegexp, _ = regexp.Compile("[ ]+")
)

type (
	memStat struct {
		vms uint64
		rss uint64
	}
)

// /proc/<pid>/statm fields (see `man proc` for details)
const (
	statmVMS = iota
	statmRSS

	statmShare
	statmText
	statmLib
	statmData
	statmDt

	statmFieldsCount
)

const (
	statUtime = 13
	statStime = 14

	statStartTime = 21

	statFieldsCount = 44
)

const (
	pairKey = iota
	pairVal
	pairLen
)

func readLines(b []byte) (text []string) {
	reader := bytes.NewBuffer(b)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		text = append(text, scanner.Text())
	}

	return
}

func loadSysBootTime() (bt uint64, err error) {
	var b []byte
	if b, err = ioutil.ReadFile("/proc/stat"); err != nil {
		return
	}

	for _, ln := range readLines(b) {
		if strings.HasPrefix(ln, "btime") {
			fields := strings.Fields(ln)
			if len(fields) < pairLen {
				return bt, fmt.Errorf("incorrect count of fields in `btime` record: %d", len(fields))
			}

			return strconv.ParseUint(fields[pairVal], 10, 64)
		}
	}

	return
}

func getProcPath(pid int, file string) string {
	return fmt.Sprintf("/proc/%d/%s", pid, file)
}

func getProcContent(pid int, file string) (content string, err error) {
	var b []byte

	if b, err = ioutil.ReadFile(getProcPath(pid, file)); err != nil {
		return
	}

	content = string(b)
	return
}

func readMemStat(pid int) (mstat memStat, err error) {
	var content string
	if content, err = getProcContent(pid, "statm"); err != nil {
		return
	}

	fields := strings.Fields(content)
	if len(fields) < statmFieldsCount {
		err = fmt.Errorf("wrong number of fields in `statm` file: %d, but shoud be greater or equal to %d", len(fields), statmFieldsCount)
		return
	}

	var vms, rss uint64
	vms, err = strconv.ParseUint(fields[statmVMS], 10, 64)
	if err != nil {
		return
	}

	rss, err = strconv.ParseUint(fields[statmRSS], 10, 64)
	if err != nil {
		return
	}

	mstat = memStat{
		vms: vms * pageSize,
		rss: rss * pageSize,
	}

	return
}

func readCPUPercent(pid int, bootTime uint64) (cpu float32, uptime uint64, err error) {
	var content string
	if content, err = getProcContent(pid, "stat"); err != nil {
		return
	}

	fields := strings.Fields(content)
	if len(fields) < statFieldsCount {
		err = fmt.Errorf("wrong number of fields in `statm` file: %d, but shoud be greater or equal to %d", len(fields), statFieldsCount)
		return
	}

	var utime, stime, startedAt uint64
	if utime, err = strconv.ParseUint(fields[statUtime], 10, 64); err != nil {
		return
	}

	if stime, err = strconv.ParseUint(fields[statStime], 10, 64); err != nil {
		return
	}

	if startedAt, err = strconv.ParseUint(fields[statStartTime], 10, 64); err != nil {
		return
	}

	utimeSec := float64(utime) / clockTicks
	stimeSec := float64(stime) / clockTicks

	startedAt =  bootTime + startedAt / clockTicks
	created := time.Unix(0, int64(startedAt * uint64(time.Second)))

	total := float64(utimeSec + stimeSec)
	if runtime := time.Since(created).Seconds(); runtime > 0 {
		uptime = uint64(runtime)
		cpu = float32(100 * total / runtime)
	}

	return
}

func makeNiceName(name string) string {
	return spacesRegexp.ReplaceAllString(name, "_")
}

func readProcStat(pid int, bootTime uint64) (stat isolate.WorkerMetrics,err error) {
	var (
		cpuload float32
		uptimeSeconds uint64
		memstat memStat
	)

	if cpuload, uptimeSeconds, err = readCPUPercent(pid, bootTime); err != nil {
		return
	}

	if memstat, err = readMemStat(pid); err != nil {
		return
	}

	stat = isolate.WorkerMetrics{
		UptimeSec: uptimeSeconds,
		// CpuUsageSec:

		CpuLoad: cpuload,
		Mem:     memstat.vms,

		// Per process net io stat is unimplemented.
		// Net: generateNetStat(netstat),
	}

	return
}

func (b *Box) gatherMetrics(ctx context.Context, bootTime uint64) {
	ids := b.getIdUuidMapping()
	metrics := make(map[string]*isolate.WorkerMetrics, len(ids))

	for pid, uuid := range ids {
		if stat, err := readProcStat(pid, bootTime); err == nil {
			metrics[uuid] = &stat
		} else {
			log.G(ctx).Errorf("Failed to read stat, pid: %d, err: %v", pid, err)
		}
	} // for each taskInfo

	b.setMetricsMapping(metrics)
}

func (b *Box) gatherMetricsEvery(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		log.G(ctx).Info("Process metrics gatherer disabled (use config to setup)")
		return
	}

	log.G(ctx).Infof("Initializing Process metrics gather loop with %v duration", interval)

	bootTime, err := loadSysBootTime()
	if err != nil {
		log.G(ctx).Errorf("Error while reading system boot time %v", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			b.gatherMetrics(ctx, bootTime)
		}
	}

	log.G(ctx).Info("Cancelling Process metrics loop")
}
