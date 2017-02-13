package exportmetrics

import (
	"encoding/json"
	"fmt"

	metrics "github.com/rcrowley/go-metrics"
	"golang.org/x/net/context"

	"github.com/noxiouz/stout/pkg/config"
	"github.com/noxiouz/stout/pkg/log"
)

type Sender interface {
	Send(ctx context.Context, registry metrics.Registry) error
}

type noopSender struct{}

func (noopSender) Send(ctx context.Context, registry metrics.Registry) error {
	return nil
}

// New create new Sender
func New(ctx context.Context, configuration *config.Config) (Sender, error) {
	switch name := configuration.Metrics.Type; name {
	case "graphite":
		var cfg GraphiteConfig
		if err := json.Unmarshal(configuration.Metrics.Args, &cfg); err != nil {
			log.G(ctx).WithError(err).WithField("name", name).Error("unable to decode graphite exporter config")
			return nil, err
		}
		return NewGraphiteExporter(&cfg)
	case "":
		log.G(ctx).Warn("metrics: exporter is not specified")
		return noopSender{}, nil
	default:
		log.G(ctx).WithField("exporter", name).Error("unknown exporter")
		return nil, fmt.Errorf("unknown sender type %s", name)
	}
}
