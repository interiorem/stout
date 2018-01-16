// TODO:
//  - log timings
//
package porto

import (
	"fmt"
	"golang.org/x/net/context"

	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/noxiouz/stout/isolate"
	"github.com/noxiouz/stout/pkg/log"

	porto "github.com/yandex/porto/src/api/go"
)

var (
	spacesRegexp, _ = regexp.Compile("[ ]+")
	metricsNames    = []string{
		"cpu_usage",
		"time",
		"memory_usage",
		"net_tx_bytes",
		"net_rx_bytes",
	}
)

const (
	nanosPerSecond = 1000000000
)

const (
	pairName = iota
	pairVal
	pairLen
)

type portoResponse map[string]map[string]porto.TPortoGetResponse
type rawMetrics map[string]porto.TPortoGetResponse

type netIfStat struct {
	name       string
	bytesCount uint64
}

//
// Parses string in format `w(lan) interface: bytes count`
//
func parseNetPair(eth string) (nstat netIfStat, err error) {
	pair := strings.Split(eth, ": ")
	if len(pair) == pairLen {
		var v uint64
		trimmedStr := strings.Trim(pair[pairVal], " ")
		v, err = strconv.ParseUint(trimmedStr, 10, 64)
		if err != nil {
			return
		}

		name := strings.Trim(pair[pairName], " ")
		name = spacesRegexp.ReplaceAllString(name, "_")

		nstat = netIfStat{
			name:       name,
			bytesCount: v,
		}
	} else {
		err = fmt.Errorf("Failed to parse net record")
	}

	return
}

// TODO: check property Error/ErrorMsg fields
func parseNetValues(val porto.TPortoGetResponse) (ifs []netIfStat) {
	for _, eth := range strings.Split(val.Value, ";") {
		if nf, err := parseNetPair(eth); err == nil {
			ifs = append(ifs, nf)
		}
	}

	return
}

// TODO: check property Error/ErrorMsg fields
func parseUintProp(raw rawMetrics, propName string) (v uint64, err error) {
	s, ok := raw[propName]
	if !ok {
		return 0, fmt.Errorf("no such prop in Porto: %s", propName)
	}

	if len(s.Value) == 0 {
		return v, fmt.Errorf("property is empty string")
	}

	return strconv.ParseUint(s.Value, 10, 64)
}

func setUintField(field *uint64, raw rawMetrics, propName string) (err error) {
	var v uint64
	if v, err = parseUintProp(raw, propName); err == nil {
		*field = v
	}

	return
}

func makeMetricsFromMap(raw rawMetrics) (m isolate.WorkerMetrics, err error) {
	m = isolate.NewWorkerMetrics()

	if err = setUintField(&m.UptimeSec, raw, "time"); err != nil {
		return
	}

	if err = setUintField(&m.CpuUsageSec, raw, "cpu_usage"); err != nil {
		return
	}

	if m.UptimeSec > 0 {
		m.CpuLoad = float64(m.CpuUsageSec) / float64(nanosPerSecond) / float64(m.UptimeSec)
	}

	// Porto's `cpu_usage` is in nanoseconds, seconds in metrics are used.
	m.CpuUsageSec /= nanosPerSecond

	if err = setUintField(&m.Mem, raw, "memory_usage"); err != nil {
		return
	}
	// `memory_usage` is in bytes, not in pages
	// m.Mem *= pageSize

	for _, netIf := range parseNetValues(raw["net_tx_bytes"]) {
		v := m.Net[netIf.name]
		v.TxBytes += netIf.bytesCount
		m.Net[netIf.name] = v
	}

	for _, netIf := range parseNetValues(raw["net_rx_bytes"]) {
		v := m.Net[netIf.name]
		v.RxBytes += netIf.bytesCount
		m.Net[netIf.name] = v
	}

	return
}

func parseMetrics(ctx context.Context, props portoResponse, idToUuid map[string]string) map[string]*isolate.WorkerMetrics {
	var parse_errors []string

	metrics := make(map[string]*isolate.WorkerMetrics, len(props))
	for id, rawMetrics := range props {
		uuid, ok := idToUuid[id]
		if !ok {
			continue
		}

		if m, err := makeMetricsFromMap(rawMetrics); err != nil {
			parse_errors = append(parse_errors, err.Error())
			continue
		} else {
			metrics[uuid] = &m
		}

	}

	if len(parse_errors) != 0 {
		log.G(ctx).Errorf("Failed to parse raw metrics with error %s", strings.Join(parse_errors, ", "))
	}

	return metrics
}

func makeIdsSlice(idToUuid map[string]string) (ids []string) {
	for id := range idToUuid {
		ids = append(ids, id)
	}
	return
}

func closeApiWithLog(ctx context.Context, portoApi porto.API) {
	if err := portoApi.Close(); err != nil {
		log.G(ctx).WithError(err).Error("Failed to close connection to Porto service")
	}
}

func (box *Box) gatherMetrics(ctx context.Context) {
	idToUuid := box.getIdUuidMapping()

	portoApi, err := portoConnect()
	if err != nil {
		log.G(ctx).WithError(err).Error("Failed to connect to Porto service for workers metrics collection")
		return
	}
	defer closeApiWithLog(ctx, portoApi)

	ids := makeIdsSlice(idToUuid)

	var props portoResponse
	props, err = portoApi.Get(ids, metricsNames)
	if err != nil {
		log.G(ctx).WithError(err).Error("Failed to connect to Porto service")
		return
	}

	metrics := parseMetrics(ctx, props, idToUuid)
	box.setMetricsMapping(metrics)
}

func (box *Box) gatherMetricsEvery(ctx context.Context, interval time.Duration) {

	if interval == 0 {
		log.G(ctx).Info("Porto metrics gatherer disabled (use config to setup)")
		return
	}

	log.G(ctx).Infof("Initializing Porto metrics gather loop with %v duration", interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			box.gatherMetrics(ctx)
		}
	}

	log.G(ctx).Info("Porto metrics gather loop canceled")
}
