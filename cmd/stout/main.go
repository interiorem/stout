package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	apexlog "github.com/apex/log"
	metrics "github.com/rcrowley/go-metrics"
	"golang.org/x/net/context"

	"github.com/interiorem/stout/daemon"
	_ "github.com/interiorem/stout/isolate/docker"
	_ "github.com/interiorem/stout/isolate/porto"
	_ "github.com/interiorem/stout/isolate/process"

	"github.com/interiorem/stout/isolate"
	"github.com/interiorem/stout/pkg/exportmetrics"
	"github.com/interiorem/stout/pkg/log"
	"github.com/interiorem/stout/pkg/logutils"
	"github.com/interiorem/stout/version"

	flag "github.com/ogier/pflag"
)

var (
	configpath  string
	showVersion bool
)

const (
	requiredConfigVersion = 2

	minimalPeriod = 5 * time.Second
)

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

func sendEvery(ctx context.Context, sender exportmetrics.Sender, d time.Duration) {
	ticker := time.NewTicker(d)
	for {
		select {
		case <-ticker.C:
			if err := sender.Send(ctx, metrics.DefaultRegistry); err != nil {
				log.G(ctx).WithError(err).Error("unable to send metrics")
			}
		case <-ctx.Done():
			return
		}
	}
}

func main() {
	if showVersion {
		printVersion()
		return
	}

	// Read configuration
	data, err := ioutil.ReadFile(configpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read config: %v\n", err)
		os.Exit(1)
	}
	config, err := isolate.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config is invalid: %v\n", err)
		os.Exit(1)
	}
	if config.Version != requiredConfigVersion {
		fmt.Fprintf(os.Stderr, "invalid config version (%d). %d is required\n", config.Version, requiredConfigVersion)
		os.Exit(1)
	}

	// Create logger
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

	// Initialize metrics sender
	sender, err := exportmetrics.New(ctx, config)
	if err != nil {
		logger.Fatalf("unable to create metrics exporter %v", err)
	}
	period := time.Duration(config.Metrics.Period)
	if period < minimalPeriod {
		logger.Warnf("metrics: specified period is too low. Set %s", minimalPeriod)
		period = minimalPeriod
	}
	go sendEvery(ctx, sender, period)

	// create isolateDaemon
	isolateDaemon, err := daemon.New(ctx, config)
	if err != nil {
		logger.Fatalf("unable to initialize daemon %v", err)
	}
	defer isolateDaemon.Close()
	isolateDaemon.RegisterHTTPHandlers(ctx, http.DefaultServeMux)
	go daemon.Collect(ctx, 30*time.Second)

	if config.DebugServer != "" {
		logger.WithField("endpoint", config.DebugServer).Info("start debug HTTP-server")
		go func() {
			logger.WithError(http.ListenAndServe(config.DebugServer, nil)).Error("debug server is listening")
		}()
	}

	if err = isolateDaemon.Serve(ctx); err != nil {
		logger.Errorf("Serve error %s", err)
		os.Exit(1)
	}
}
