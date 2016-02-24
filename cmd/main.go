package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/distribution/registry/listener"

	"github.com/noxiouz/stout/isolate"
	logformatter "github.com/noxiouz/stout/pkg/formatter"
	"github.com/noxiouz/stout/server"
	"github.com/noxiouz/stout/version"
)

type logLevelFlag log.Level

func (l *logLevelFlag) Set(val string) error {
	level, err := log.ParseLevel(strings.ToLower(val))
	if err != nil {
		return err
	}
	(*l) = logLevelFlag(level)
	return nil
}

func (l *logLevelFlag) TologLevel() log.Level {
	return log.Level(*l)
}

func (l *logLevelFlag) String() string {
	return l.TologLevel().String()
}

func (l *logLevelFlag) UnmarshalJSON(data []byte) error {
	return l.Set(strings.Trim(string(data), "\" "))
}

type httpEndpoints []string

func (e *httpEndpoints) Set(val string) error {
	*e = strings.Split(val, ",")
	return nil
}

func (e *httpEndpoints) String() string {
	return strings.Join(*e, ",")
}

var (
	config struct {
		HTTP                         httpEndpoints `json:"http"`
		Loglevel                     logLevelFlag  `json:"loglevel"`
		LogFile                      string        `json:"logfile"`
		isolate.PortoIsolationConfig `json:"isolate"`
	}

	fileconfig         string
	showVersion        bool
	showFullDebVersion bool
)

func init() {
	config.Loglevel = logLevelFlag(log.DebugLevel)
	config.HTTP = httpEndpoints{":5432"}

	flag.BoolVar(&showVersion, "version", false, "show the version")
	flag.BoolVar(&showFullDebVersion, "fulldebversion", false, "show full debian version")
	flag.StringVar(&fileconfig, "config", "", "path to a configuration file")

	flag.Var(&config.Loglevel, "loglevel", "debug|info|warn|warning|error|panic")
	flag.StringVar(&config.LogFile, "logfile", "", "path to a logfile")
	flag.Var(&config.HTTP, "http", "comma separated list of endpoints to listen on")
	flag.StringVar(&config.RootNamespace, "root", "cocs", "name of the root container")
	flag.StringVar(&config.Layers, "layers", "/tmp/isolate", "path to a temp dir for layers")
	flag.StringVar(&config.Volumes, "volumes", "/cocaine-porto", "dir for volumes")
}

func main() {
	flag.Parse()
	if showVersion {
		fmt.Printf("version: `%s`\n", version.Version)
		return
	}

	if showFullDebVersion {
		fmt.Printf("version: `%s`\n", version.Version)
		fmt.Printf("hash: `%s`\n", version.GitHash)
		fmt.Printf("build utc time: `%s`\n", version.Build)
		return
	}

	if fileconfig != "" {
		func() {
			file, err := os.Open(fileconfig)
			if err != nil {
				fmt.Printf("unable to open config `%s`: %v\n", fileconfig, err)
				os.Exit(128)
			}
			defer file.Close()

			if err := json.NewDecoder(file).Decode(&config); err != nil {
				fmt.Printf("unable to decode config: %v\n", err)
				os.Exit(128)
			}
		}()
	}

	if config.LogFile != "" {
		// TODO: wrap it to support SIGHUP
		logfile, err := os.OpenFile(config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("unable to open logfile `%s`: %v\n", config.LogFile, err)
			os.Exit(128)
		}
		defer logfile.Close()

		log.SetOutput(logfile)
	}

	log.SetFormatter(&logformatter.CombaineFormatter{})
	log.SetLevel(config.Loglevel.TologLevel())

	// TODO: rewrite this ugly code
	serverConfig := server.Config{
		PortoIsolationConfig: config.PortoIsolationConfig,
	}

	_ = serverConfig

	isolateServer, err := server.NewIsolateServer(&serverConfig)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	for _, endpoint := range config.HTTP {
		var (
			proto = "tcp"
			addr  = endpoint
		)

		if strings.HasPrefix(endpoint, "unix://") {
			proto = "unix"
			addr = endpoint[len("unix://"):]
		}

		l, err := listener.NewListener(proto, addr)
		if err != nil {
			log.Fatal(err)
		}

		wg.Add(1)
		go func(l net.Listener) {
			defer wg.Done()
			defer l.Close()

			server := http.Server{
				Handler: isolateServer.Router,
			}
			log.WithField("endpoint", l.Addr()).Info("Starting cocaine-porto")
			server.Serve(l)
		}(l)
	}

	wg.Wait()
}
