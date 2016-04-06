package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"golang.org/x/net/context"

	"github.com/apex/log"
	"github.com/apex/log/handlers/logfmt"

	"github.com/noxiouz/stout/isolation"
	"github.com/noxiouz/stout/isolation/process"
)

var (
	configpath string
)

// Config describes a configuration file for the daemon
type Config struct {
	Endpoints   []string `json:"endpoints"`
	DebugServer string   `json:"debugserver"`
	Logger      struct {
		Level  string `json:"level"`
		Output string `json:"output"`
	} `json:"logger"`
	Isolate map[string]isolation.BoxConfig `json:"isolate"`
}

func init() {
	flag.StringVar(&configpath, "config", "/etc/stout/stout-default.conf", "path to a configuration file")
	flag.Parse()
}

func main() {
	var config Config
	if err := func(path string) error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		return json.NewDecoder(f).Decode(&config)
	}(configpath); err != nil {
		fmt.Fprintf(os.Stderr, "unable to read config: %v\n", err)
		os.Exit(1)
	}

	lvl, err := log.ParseLevel(config.Logger.Level)
	if err != nil {
		lvl = log.DebugLevel
	}

	var output io.WriteCloser
	switch path := config.Logger.Output; path {
	case os.Stderr.Name():
		output = os.Stderr
	case os.Stdout.Name():
		output = os.Stdout
	default:
		output, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to create log file %s: %v\n", path, err)
			os.Exit(1)
		}
		defer output.Close()
	}

	handler := logfmt.New(output)

	logger := &log.Logger{
		Level:   lvl,
		Handler: handler,
	}

	if len(config.Isolate) == 0 {
		logger.Fatal("the isolate section is empty")
	}

	boxes := isolation.Boxes{}
	for name, cfg := range config.Isolate {
		var box isolation.Box
		switch name {
		case "docker":
			box, err = process.NewBox(cfg)
		case "process":
			box, err = process.NewBox(cfg)
		default:
			logger.WithError(err).WithField("box", name).Fatal("unknown box type")
		}
		if err != nil {
			logger.WithError(err).WithField("box", name).Fatal("unable to create box")
		}
		boxes[name] = box
	}

	ctx := context.WithValue(context.Background(), isolation.BoxesTag, boxes)
	ctx = context.WithValue(ctx, "logger", logger)

	if len(config.Endpoints) == 0 {
		logger.Fatal("no listening endpoints are specified in endpoints section")
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
					lnLogger.WithError(err).Error("Accept() error")
					return
				}
				lnLogger.WithField("remote_addr", conn.RemoteAddr()).Info("accepted new connection")

				connHandler, err := isolation.NewConnectionHandler(ctx)
				if err != nil {
					lnLogger.WithError(err).Fatal("unable to create connection handler")
				}

				go connHandler.HandleConn(conn)
			}
		}(ln)
	}

	wg.Wait()
}
