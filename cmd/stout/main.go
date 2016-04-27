package main

import (
	"bytes"
	"encoding/json"
	_ "expvar"
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sort"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/apex/log"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/isolate/docker"
	"github.com/noxiouz/stout/isolate/process"
	"github.com/noxiouz/stout/version"

	flag "github.com/ogier/pflag"
)

var (
	configpath  string
	showVersion bool
)

// Config describes a configuration file for the daemon
type Config struct {
	Endpoints   []string `json:"endpoints"`
	DebugServer string   `json:"debugserver"`
	Logger      struct {
		Level  string `json:"level"`
		Output string `json:"output"`
	} `json:"logger"`
	Isolate map[string]isolate.BoxConfig `json:"isolate"`
}

type logHandler struct {
	mu sync.Mutex
	io.Writer
}

func getLevel(lvl log.Level) string {
	switch lvl {
	case log.DebugLevel:
		return "DEBUG"
	case log.InfoLevel:
		return "INFO"
	case log.WarnLevel:
		return "WARN"
	case log.ErrorLevel, log.FatalLevel:
		return "ERROR"
	default:
		return lvl.String()
	}
}

func (lh *logHandler) HandleLog(entry *log.Entry) error {
	keys := make([]string, 0, len(entry.Fields))
	for k := range entry.Fields {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	buf := new(bytes.Buffer)
	buf.WriteString(entry.Timestamp.Format(time.RFC3339))
	buf.WriteByte('\t')
	buf.WriteString(getLevel(entry.Level))
	buf.WriteByte('\t')
	buf.WriteString(entry.Message)
	if i := len(entry.Fields); i > 0 {
		buf.WriteByte('\t')
		buf.WriteByte('[')

		for _, k := range keys {
			buf.WriteString(fmt.Sprintf("%s: %v", k, entry.Fields[k]))
			i--
			if i > 0 {
				buf.WriteByte(',')
				buf.WriteByte(' ')
			}
		}
		buf.WriteByte(']')
	}
	buf.WriteByte('\n')

	lh.mu.Lock()
	defer lh.mu.Unlock()

	_, err := buf.WriteTo(lh.Writer)
	return err
}

func init() {
	flag.StringVarP(&configpath, "config", "c", "/etc/stout/stout-default.conf", "path to a configuration file")
	flag.BoolVarP(&showVersion, "version", "v", false, "show version and exit")
	flag.Parse()
}

func main() {
	if showVersion {
		fmt.Printf("version: `%s`\n", version.Version)
		fmt.Printf("hash: `%s`\n", version.GitHash)
		fmt.Printf("build utc time: `%s`\n", version.Build)
		return
	}
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

	handler := &logHandler{
		Writer: output,
	}

	logger := &log.Logger{
		Level:   lvl,
		Handler: handler,
	}

	if len(config.Isolate) == 0 {
		logger.Fatal("the isolate section is empty")
	}

	boxes := isolate.Boxes{}
	for name, cfg := range config.Isolate {
		var box isolate.Box
		switch name {
		case "docker":
			box, err = docker.NewBox(cfg)
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

	ctx := context.WithValue(context.Background(), isolate.BoxesTag, boxes)
	ctx = context.WithValue(ctx, "logger", logger)

	if len(config.Endpoints) == 0 {
		logger.Fatal("no listening endpoints are specified in endpoints section")
	}

	if config.DebugServer != "" {
		logger.WithField("endpoint", config.DebugServer).Info("start debug server")
		go func(endpoint string) {
			logger.WithError(http.ListenAndServe(endpoint, nil)).Error("debug server is listening")
		}(config.DebugServer)
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
