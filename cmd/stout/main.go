package main

import (
	"crypto/md5"
	"encoding/json"
	"expvar"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"

	apexlog "github.com/apex/log"
	"github.com/rcrowley/go-metrics"
	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolate"
	_ "github.com/noxiouz/stout/isolate/docker"
	_ "github.com/noxiouz/stout/isolate/porto"
	_ "github.com/noxiouz/stout/isolate/process"

	"github.com/noxiouz/stout/pkg/config"
	"github.com/noxiouz/stout/pkg/exportmetrics"
	"github.com/noxiouz/stout/pkg/fds"
	"github.com/noxiouz/stout/pkg/log"
	"github.com/noxiouz/stout/pkg/logutils"
	"github.com/noxiouz/stout/version"

	flag "github.com/ogier/pflag"
)

const (
	desiredRlimit = 65535
)

var (
	configpath  string
	showVersion bool
)

var (
	openFDs    = metrics.NewGauge()
	goroutines = metrics.NewGauge()
	threads    = metrics.NewGauge()
	conns      = metrics.NewCounter()
)

const requiredConfigVersion = 2

func init() {
	registry := metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, "daemon_")
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

func collect(ctx context.Context) {
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

func init() {
	flag.StringVarP(&configpath, "config", "c", "/etc/stout/stout-default.conf", "path to a configuration file")
	flag.BoolVarP(&showVersion, "version", "v", false, "show version and exit")
	flag.Parse()
}

func printVersion() {
	fmt.Printf("version: `%s`\n", version.Version)
	fmt.Printf("hash: `%s`\n", version.GitHash)
	fmt.Printf("git tag: `%s`\n", version.GitTag)
	fmt.Printf("build utc time: `%s`\n", version.Build)
}

func main() {
	if showVersion {
		printVersion()
		return
	}

	data, err := ioutil.ReadFile(configpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read config: %v\n", err)
		os.Exit(1)
	}

	config, err := config.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config is invalid: %v\n", err)
		os.Exit(1)
	}

	if config.Version != requiredConfigVersion {
		fmt.Fprintf(os.Stderr, "invalid config version (%d). %d is required\n", config.Version, requiredConfigVersion)
		os.Exit(1)
	}

	output, err := logutils.NewLogFileOutput(config.Logger.Output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to open logfile output: %v\n", err)
		os.Exit(1)
	}
	defer output.Close()

	logger := &apexlog.Logger{
		Level:   apexlog.Level(config.Logger.Level),
		Handler: logutils.NewLogHandler(output),
	}

	ctx := log.WithLogger(context.Background(), apexlog.NewEntry(logger))
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	switch name := config.Metrics.Type; name {
	case "graphite":
		var cfg exportmetrics.GraphiteConfig
		if err = json.Unmarshal(config.Metrics.Args, &cfg); err != nil {
			logger.WithError(err).WithField("name", name).Fatal("unable to decode graphite exporter config")
		}

		sender, err := exportmetrics.NewGraphiteExporter(&cfg)
		if err != nil {
			logger.WithError(err).WithField("name", name).Fatal("unable to create GraphiteExporter")
		}

		minimalPeriod := 5 * time.Second
		period := time.Duration(config.Metrics.Period)
		if period < minimalPeriod {
			logger.Warnf("metrics: specified period is too low. Set %s", minimalPeriod)
			period = minimalPeriod
		}

		go func(ctx context.Context, p time.Duration) {
			for {
				select {
				case <-time.After(p):
					if err := sender.Send(ctx, metrics.DefaultRegistry); err != nil {
						logger.WithError(err).WithField("name", name).Error("unable to send metrics")
					}
				case <-ctx.Done():
					return
				}
			}
		}(ctx, period)
	case "":
		logger.Warn("metrics: exporter is not specified")
	default:
		logger.WithError(err).WithField("exporter", name).Fatal("unknown exporter")
	}
	go func() {
		collect(ctx)
		for range time.Tick(30 * time.Second) {
			collect(ctx)
		}
	}()

	checkLimits(ctx)

	boxTypes := map[string]struct{}{}
	boxes := isolate.Boxes{}
	for name, cfg := range config.Isolate {
		if _, ok := boxTypes[cfg.Type]; ok {
			logger.WithField("box", name).WithField("type", cfg.Type).Fatal("dublicated box type")
		}
		boxCtx := log.WithLogger(ctx, logger.WithField("box", name))
		box, err := isolate.ConstructBox(boxCtx, cfg.Type, cfg.Args)
		if err != nil {
			logger.WithError(err).WithField("box", name).WithField("type", cfg.Type).Fatal("unable to create box")
		}
		boxes[name] = box
		boxTypes[cfg.Type] = struct{}{}
	}

	ctx = context.WithValue(ctx, isolate.BoxesTag, boxes)

	if config.DebugServer != "" {
		logger.WithField("endpoint", config.DebugServer).Info("start debug HTTP-server")
		go func() {
			logger.WithError(http.ListenAndServe(config.DebugServer, nil)).Error("debug server is listening")
		}()
	}

	var wg sync.WaitGroup
	for _, endpoint := range config.Endpoints {
		logger.WithField("endpoint", endpoint).Info("start TCP server")
		ln, err := net.Listen("tcp", endpoint)
		if err != nil {
			logger.WithError(err).WithField("endpoint", endpoint).Fatal("unable to listen to")
		}
		defer ln.Close()

		wg.Add(1)
		func(ln net.Listener) {
			defer wg.Done()
			lnLogger := logger.WithField("listener", ln.Addr())
			for {
				conn, err := ln.Accept()
				if err != nil {
					lnLogger.WithError(err).Error("Accept")
					continue
				}

				// TODO: more optimal way
				connID := fmt.Sprintf("%.4x", md5.Sum([]byte(fmt.Sprintf("%s.%d", conn.RemoteAddr().String(), time.Now().Unix()))))
				lnLogger.WithFields(apexlog.Fields{"remote.addr": conn.RemoteAddr(), "conn.id": connID}).Info("accepted new connection")

				connHandler, err := isolate.NewConnectionHandler(context.WithValue(ctx, "conn.id", connID))
				if err != nil {
					lnLogger.WithError(err).Fatal("unable to create connection handler")
				}

				go func() {
					conns.Inc(1)
					defer conns.Dec(1)
					connHandler.HandleConn(conn)
				}()
			}
		}(ln)
	}

	wg.Wait()
}
