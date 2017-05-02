package daemon

import (
	"context"
	"expvar"
	"fmt"
	"net/http"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/noxiouz/stout/pkg/exportmetrics"
	"github.com/noxiouz/stout/pkg/fds"
	"github.com/noxiouz/stout/pkg/log"
	"github.com/noxiouz/stout/version"
	metrics "github.com/rcrowley/go-metrics"
)

const (
	desiredRlimit = 65535
)

var (
	openFDs    = metrics.NewGauge()
	goroutines = metrics.NewGauge()
	threads    = metrics.NewGauge()
	conns      = metrics.NewCounter()

	registry = metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, "daemon_")
)

func init() {
	registry.Register("open_fds", openFDs)
	registry.Register("goroutines", goroutines)
	registry.Register("threads", threads)
	registry.Register("connections", conns)

	registry.Register("hc_openfd", metrics.NewHealthcheck(fdHealthCheck))
	registry.Register("hc_threads", metrics.NewHealthcheck(threadHealthCheck))

	http.Handle("/metrics", exportmetrics.HTTPExport(metrics.DefaultRegistry))

	expvar.NewString("version_info").Set(fmt.Sprintf("version:%s hash:%s build:%s tag:%s",
		version.Version, version.GitHash, version.Build, version.GitTag))
}

func fdHealthCheck(h metrics.Healthcheck) {
	var l syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &l); err != nil {
		h.Unhealthy(err)
		return
	}

	if val := openFDs.Value(); uint64(val) >= l.Cur-100 {
		h.Unhealthy(fmt.Errorf("too many open files %d (max %d)", val, l.Cur))
		return
	}

	h.Healthy()
}

func threadHealthCheck(h metrics.Healthcheck) {
	if val := threads.Value(); val >= 5000 {
		h.Unhealthy(fmt.Errorf("too many OS threads %d (max 10000)", val))
		return
	}

	h.Healthy()
}

func Collect(ctx context.Context, period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	collect := func() {
		goroutines.Update(int64(runtime.NumGoroutine()))
		count, err := fds.GetOpenFds()
		if err != nil {
			log.G(ctx).WithError(err).Error("get open fd count")
			return
		}

		openFDs.Update(int64(count))
		threads.Update(int64(pprof.Lookup("threadcreate").Count()))

		metrics.DefaultRegistry.RunHealthchecks()
	}
	collect()
	for {
		select {
		case <-ticker.C:
			collect()
		case <-ctx.Done():
			return
		}
	}
}

func checkLimits(ctx context.Context) {
	var l syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &l); err != nil {
		log.G(ctx).WithError(err).Error("get RLIMIT_NOFILE")
		return
	}

	if l.Cur < desiredRlimit {
		log.G(ctx).Warnf("RLIMIT_NOFILE %d is less that desired %d", l.Cur, desiredRlimit)
	}
}
