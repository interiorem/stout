package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"

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

var (
	httpFlag    string
	showVersion bool

	loglevel = logLevelFlag(log.DebugLevel)

	root    string
	layers  string
	volumes string
)

func init() {
	flag.StringVar(&httpFlag, "http", ":5432", "endpoint to serve http on")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Var(&loglevel, "loglevel", "debug|info|warn|warning|error|panic")

	flag.StringVar(&root, "root", "cocs", "name of the root container")
	flag.StringVar(&layers, "layers", "/tmp/isolate", "path to a temp dir for layers")
	flag.StringVar(&volumes, "volumes", "/cocaine-porto", "dir for volumes")

	flag.Parse()

	log.SetFormatter(&logformatter.CombaineFormatter{})
	log.SetLevel(loglevel.TologLevel())
}

func main() {
	if showVersion {
		fmt.Printf("version: `%s`\n", version.Version)
		return
	}

	config := server.Config{}
	config.PortoIsolationConfig.RootNamespace = root
	config.PortoIsolationConfig.Layers = layers
	config.PortoIsolationConfig.Volumes = volumes

	isolateServer, err := server.NewIsolateServer(&config)
	if err != nil {
		log.Fatal(err)
	}

	server := http.Server{
		Addr:    httpFlag,
		Handler: isolateServer.Router,
	}

	log.WithField("endpoint", httpFlag).Info("Start cocaine-porto")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
