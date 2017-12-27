//go:generate msgp --tests=false
//msgp:ignore isolate.MarkedContainerMetrics
package isolate

type (
	NetStat struct {
		RxBytes uint64 `msg:"rx_bytes"`
		TxBytes uint64 `msg:"tx_bytes"`
	}

	ContainerMetrics struct {
		UptimeSec uint64 `msg:"uptime"`
		CpuUsageSec uint64 `msg:"cpu_usage"`

		CpuLoad float64 `msg:"cpu_load"`

		Mem uint64 `msg:"mem"`

		// iface -> net stat
		Net map[string]NetStat `msg:"net"`
	}

	MetricsResponse map[string]*ContainerMetrics

	MarkedContainerMetrics struct {
		uuid string
		m *ContainerMetrics
	}
)

func NewContainerMetrics() (c ContainerMetrics) {
	c.Net = make(map[string]NetStat)
	return
}

func NewMarkedMetrics(uuid string, cm *ContainerMetrics) MarkedContainerMetrics {
	return MarkedContainerMetrics{uuid: uuid, m: cm}
}
