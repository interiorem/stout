package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"

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

var (
	config struct {
		HTTP                         string       `json:"http"`
		Loglevel                     logLevelFlag `json:"loglevel"`
		isolate.PortoIsolationConfig `json:"isolate"`
	}

	fileconfig  string
	showVersion bool
	showFullDebVersion bool
)

func init() {
	config.Loglevel = logLevelFlag(log.DebugLevel)

	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.BoolVar(&showFullDebVersion, "fulldebversion", false, "show full debian version")
	flag.StringVar(&fileconfig, "config", "", "path to configuration file")

	flag.Var(&config.Loglevel, "loglevel", "debug|info|warn|warning|error|panic")
	flag.StringVar(&config.HTTP, "http", ":5432", "endpoint to serve http on")
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

	log.SetFormatter(&logformatter.CombaineFormatter{})
	log.SetLevel(config.Loglevel.TologLevel())

	// TODO: rewrite this ugly code
	serverConfig := server.Config{
		PortoIsolationConfig: config.PortoIsolationConfig,
	}

	isolateServer, err := server.NewIsolateServer(&serverConfig)
	if err != nil {
		log.Fatal(err)
	}

	server := http.Server{
		Addr:    config.HTTP,
		Handler: isolateServer.Router,
	}

	log.WithField("endpoint", config.HTTP).Info("Starting cocaine-porto")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
