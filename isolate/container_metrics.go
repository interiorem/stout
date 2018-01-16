//go:generate msgp --tests=false
//msgp:ignore isolate.MarkedWorkerMetrics
package isolate

type (
	NetStat struct {
		RxBytes uint64 `msg:"rx_bytes"`
		TxBytes uint64 `msg:"tx_bytes"`
	}

	WorkerMetrics struct {
		UptimeSec   uint64 `msg:"uptime"`
		CpuUsageSec uint64 `msg:"cpu_usage"`

		CpuLoad float64 `msg:"cpu_load"`

		Mem uint64 `msg:"mem"`

		// iface -> net stat
		Net map[string]NetStat `msg:"net"`
	}

	MetricsResponse map[string]*WorkerMetrics

	MarkedWorkerMetrics struct {
		uuid string
		m    *WorkerMetrics
	}
)

func NewWorkerMetrics() (c WorkerMetrics) {
	c.Net = make(map[string]NetStat)
	return
}

func NewMarkedMetrics(uuid string, cm *WorkerMetrics) MarkedWorkerMetrics {
	return MarkedWorkerMetrics{uuid: uuid, m: cm}
}
