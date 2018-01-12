// TODO:
//  - log timings
//
package process

import (
    "context"
    "regexp"
    "syscall"
    "time"

    "github.com/noxiouz/stout/isolate"
    "github.com/noxiouz/stout/pkg/log"

    gopsutil "github.com/shirou/gopsutil/process"
    gopsnet  "github.com/shirou/gopsutil/net"
)

const eachIface = true

var (
    pageSize = uint64(syscall.Getpagesize())
    spacesRegexp, _ = regexp.Compile("[ ]+")
)

func makeNiceName(name string) string {
    return spacesRegexp.ReplaceAllString(name, "_")
}

func generateNetStat(net []gopsnet.IOCountersStat) (out map[string]isolate.NetStat) {
    out = make(map[string]isolate.NetStat, len(net))

    for _, c := range net {
        out[c.Name] = isolate.NetStat{
            RxBytes: c.BytesRecv,
            TxBytes: c.BytesSent,
        }
    }

    return
}

func readProcStat(pid int, uptimeSeconds uint64) (isolate.WorkerMetrics, error) {

    var (
        process *gopsutil.Process

        cpuload float64
        // netstat []gopsnet.IOCountersStat
        memstat *gopsutil.MemoryInfoStat

        errStub isolate.WorkerMetrics
        err error
    )

    if process, err = gopsutil.NewProcess(int32(pid)); err != nil {
        return errStub, err
    }

    if cpuload, err = process.CPUPercent(); err != nil {
        return errStub, err
    }

    if memstat, err = process.MemoryInfo(); err != nil {
        return errStub, err
    }

    //
    // TODO:
    //   There is no per process network stat yet in gopsutil,
    //   Per process view of system stat is in `netstat` slice.
    //
    //   Most commonly used (the only?) way to take per process network
    //   stats is by libpcap.
    //
    // if netstat, err = process.NetIOCounters(eachIface); err != nil {
    //
    if _, err = process.NetIOCounters(eachIface); err != nil {
        return errStub, err
    }

    return isolate.WorkerMetrics{
        UptimeSec: uptimeSeconds,
        // CpuUsageSec:

        CpuLoad: cpuload,
        Mem: memstat.VMS,

        // Per process net io stat is unimplemented.
        // Net: generateNetStat(netstat),
    }, nil
}

func (b *Box) gatherMetrics(ctx context.Context) {
    ids := b.getIdUuidMapping()
    metrics := make(map[string]*isolate.WorkerMetrics, len(ids))

    now := time.Now()

    for pid, taskInfo := range ids {
        uptimeSeconds := uint64(now.Sub(taskInfo.startTime).Seconds())

        if state, err := readProcStat(pid, uptimeSeconds); err != nil {
            log.G(ctx).Errorf("Failed to read stat for process with pid %d", pid)
            continue
        } else {
            metrics[taskInfo.uuid] = &state
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

    for {
        select {
        case <- ctx.Done():
            return
        case <-time.After(interval):
            b.gatherMetrics(ctx)
        }
    }

    log.G(ctx).Info("Cancelling Process metrics loop")
}
