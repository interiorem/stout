package process

import (
	"expvar"

	"github.com/rcrowley/go-metrics"
)

var (
	// how many conatainers are queued to be spawned
	spawningQueueSize = metrics.NewCounter()

	// containers that has been tried to spawning
	procsCreatedCounter = metrics.NewCounter()
	// containers that crashed during spawning
	procsErroredCounter = metrics.NewCounter()

	procsWaitedCounter = metrics.NewCounter()

	totalSpawnTimer = metrics.NewTimer()
	procsNewTimer   = metrics.NewTimer()

	zombieWaitTimer = metrics.NewTimer()

	processConfig = expvar.NewString("process_config")
)

func init() {
	registry := metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, "process_")
	registry.Register("spawning_queue_size", spawningQueueSize)
	registry.Register("procs_created", procsCreatedCounter)
	registry.Register("procs_errored", procsErroredCounter)
	registry.Register("procs_waited", procsWaitedCounter)
	registry.Register("total_spawn_timer", totalSpawnTimer)
	registry.Register("procs_new_timer", procsNewTimer)
	registry.Register("zombie_wait_timer", zombieWaitTimer)
}
