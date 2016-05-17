package main

import (
	"expvar"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/apex/log"
	apexctx "github.com/m0sth8/context"
	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/isolate/docker"
	"github.com/noxiouz/stout/isolate/process"
	"github.com/noxiouz/stout/pkg/config"
	"github.com/noxiouz/stout/pkg/fds"
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
	daemonStats = expvar.NewMap("daemon")

	openFDs, goroutines, conns expvar.Int
)

func init() {
	daemonStats.Set("open_fds", &openFDs)
	daemonStats.Set("goroutines", &goroutines)
	daemonStats.Set("connections", &conns)
}

func collect(ctx context.Context) {
	goroutines.Set(int64(runtime.NumGoroutine()))
	count, err := fds.GetOpenFds()
	if err != nil {
		apexctx.GetLogger(ctx).WithError(err).Error("get open fd count")
		return
	}
	openFDs.Set(int64(count))
}

func checkLimits(ctx context.Context) {
	var l syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &l); err != nil {
		apexctx.GetLogger(ctx).WithError(err).Error("get RLIMIT_NOFILE")
		return
	}

	if l.Cur < desiredRlimit {
		apexctx.GetLogger(ctx).Warnf("RLIMIT_NOFILE %d is less that desired %d", l.Cur, desiredRlimit)
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

	output, err := logutils.NewLogFileOutput(config.Logger.Output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to open logfile output: %v\n", err)
		os.Exit(1)
	}
	defer output.Close()

	logger := &log.Logger{
		Level:   log.Level(config.Logger.Level),
		Handler: logutils.NewLogHandler(output),
	}

	ctx := apexctx.WithLogger(apexctx.Background(), log.NewEntry(logger))
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	go func() {
		collect(ctx)
		for range time.Tick(30 * time.Second) {
			collect(ctx)
		}
	}()

	checkLimits(ctx)

	boxes := isolate.Boxes{}
	for name, cfg := range config.Isolate {
		var box isolate.Box
		boxCtx := apexctx.WithLogger(ctx, logger.WithField("box", name))
		switch name {
		case "docker":
			box, err = docker.NewBox(boxCtx, cfg)
		case "process":
			box, err = process.NewBox(boxCtx, cfg)
		default:
			logger.WithError(err).WithField("box", name).Fatal("unknown box type")
		}
		if err != nil {
			logger.WithError(err).WithField("box", name).Fatal("unable to create box")
		}
		boxes[name] = box
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
				lnLogger.WithField("remote_addr", conn.RemoteAddr()).Info("accepted new connection")

				connHandler, err := isolate.NewConnectionHandler(ctx)
				if err != nil {
					lnLogger.WithError(err).Fatal("unable to create connection handler")
				}

				go func() {
					conns.Add(1)
					defer conns.Add(-1)
					connHandler.HandleConn(conn)
				}()
			}
		}(ln)
	}

	wg.Wait()
}
