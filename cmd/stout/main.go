package main

import (
	_ "expvar"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"

	"github.com/apex/log"
	apexctx "github.com/m0sth8/context"
	"golang.org/x/net/context"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/isolate/docker"
	"github.com/noxiouz/stout/isolate/process"
	"github.com/noxiouz/stout/pkg/config"
	"github.com/noxiouz/stout/pkg/logutils"
	"github.com/noxiouz/stout/version"

	flag "github.com/ogier/pflag"
)

var (
	configpath  string
	showVersion bool
)

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
	boxes := isolate.Boxes{}
	for name, cfg := range config.Isolate {
		var box isolate.Box
		switch name {
		case "docker":
			box, err = docker.NewBox(ctx, cfg)
		case "process":
			box, err = process.NewBox(ctx, cfg)
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
		logger.WithField("endpoint", config.DebugServer).Info("start debug server")
		go func() {
			logger.WithError(http.ListenAndServe(config.DebugServer, nil)).Error("debug server is listening")
		}()
	}

	var wg sync.WaitGroup
	for _, endpoint := range config.Endpoints {
		logger.WithField("endpoint", endpoint).Info("listening")
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

				go connHandler.HandleConn(conn)
			}
		}(ln)
	}

	wg.Wait()
}
