package docker

import (
	"expvar"

	"github.com/rcrowley/go-metrics"
)

var (
	// how many conatainers are queued to be spawned
	spawningQueueSize = metrics.NewCounter()

	// containers that has been tried to spawning
	containersCreatedCounter = metrics.NewCounter()
	// containers that crashed during spawning
	containersErroredCounter = metrics.NewCounter()

	totalSpawnTimer = metrics.NewTimer()

	dockerConfig = expvar.NewString("docker_config")
)

func init() {
	registry := metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, "docker_")
	registry.Register("spawning_queue_size", spawningQueueSize)
	registry.Register("containers_created", containersCreatedCounter)
	registry.Register("containers_errored", containersErroredCounter)
	registry.Register("total_spawn_timer", totalSpawnTimer)
}
