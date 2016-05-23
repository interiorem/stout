package isolate

import (
	"github.com/rcrowley/go-metrics"
)

var (
	spawnMeter       = metrics.NewMeter()
	killMeter        = metrics.NewMeter()
	spawnCancelMeter = metrics.NewMeter()
)

func init() {
	registry := metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, "isolate_")
	registry.Register("spawn_meter", spawnMeter)
	registry.Register("kill_meter", killMeter)
	registry.Register("spawn_cancel_meter", spawnCancelMeter)
}
