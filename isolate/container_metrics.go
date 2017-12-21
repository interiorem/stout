//go:generate msgp --tests=false
//msgp:ignore isolate.MarkedContainerMetrics
package isolate

type (
	NetStat struct {
		RxBytes, TxBytes uint64
	}

	ContainerMetrics struct {
		UptimeSec uint64

		CpuUsageNs uint64
		CpuLoad float64

		Mem uint64

		// iface -> net stat
		Net map[string]NetStat
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
