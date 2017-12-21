package porto

import (
    "fmt"
    "golang.org/x/net/context"

    "regexp"
    "time"
    "strconv"
    "strings"

    "github.com/noxiouz/stout/isolate"
    "github.com/noxiouz/stout/pkg/log"

    porto "github.com/yandex/porto/src/api/go"
)

var (
    spacesRegexp, _ = regexp.Compile("[ ]+")
    metricsNames = []string{
        "cpu_usage",
        "time",
        "memory_usage",
        "net_tx_bytes",
        "net_rx_bytes",
    }
)

const (
    pairName = iota
    pairVal
    pairLen
)

type portoResponse map[string]map[string]porto.TPortoGetResponse
type rawMetrics map[string]porto.TPortoGetResponse

type netIfStat struct {
    name string
    bytesCount uint64
}

func parseStrUIntPair(eth string) (nstat netIfStat, err error) {
    pair := strings.Split(eth, ": ")
    if len(pair) == pairLen {
        var v uint64
        v, err = strconv.ParseUint(pair[pairVal], 10, 64)
        if err != nil {
            return
        }

        name := strings.Trim(pair[pairName], " ")
        name  = spacesRegexp.ReplaceAllString(name, "_")

        nstat = netIfStat{
            name: name,
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
        nf, err := parseStrUIntPair(eth)
        if err == nil {
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

    v, err = strconv.ParseUint(s.Value, 10, 64)
    return
}

func setUintField(field *uint64, raw rawMetrics, propName string) (err error) {
    var v uint64
    if v, err = parseUintProp(raw, propName); err == nil {
        *field = v
    }

    return
}

func makeMetricsFromMap(raw rawMetrics) (m isolate.ContainerMetrics, err error) {

    m = isolate.NewContainerMetrics()

    if err = setUintField(&m.CpuUsageNs, raw, "cpu_usage"); err != nil {
        return
    }

    if err = setUintField(&m.UptimeSec, raw, "time"); err != nil {
        return
    }

    if err = setUintField(&m.Mem, raw, "memory_usage"); err != nil {
        return
    }


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

    if m.UptimeSec > 0 {
        cpu_usage_sec := float64(m.CpuUsageNs >> 10)
        m.CpuLoad = cpu_usage_sec / float64(m.UptimeSec)
    }

    return
}

func parseMetrics(ctx context.Context, props portoResponse, idToUuid map[string]string) map[string]*isolate.ContainerMetrics {

    metrics := make(map[string]*isolate.ContainerMetrics, len(props))

    for id, rawMetrics := range props {
        uuid, ok := idToUuid[id]
        if !ok {
            continue
        }

        if m, err := makeMetricsFromMap(rawMetrics); err != nil {
            log.G(ctx).WithError(err).Error("Failed to parse raw metrics")
            continue
        } else {
            metrics[uuid] = &m
        }
    }

    return metrics
}

func (box *Box) gatherMetrics(ctx context.Context) {
    log.G(ctx).Debug("Initializing Porto metrics gather loop")

    idToUuid := box.getIdUuidMapping()

    portoApi, err := portoConnect()
    if err != nil {
        log.G(ctx).WithError(err).Error("Failed to connect to Porto service")
        return
    }
    defer func() {
        if err := portoApi.Close(); err != nil {
            log.G(ctx).WithError(err).Error("Failed close connection to Porto service")
        }
    }()

    ids := make([]string, 0, len(idToUuid))
    for id, _ := range idToUuid {
        ids = append(ids, id)
    }

    var props portoResponse
    props, err = portoApi.Get(ids, metricsNames)
    if err != nil {
        log.G(ctx).WithError(err).Error("Failed to connect to Porto service")
        return
    }

    metrics := parseMetrics(ctx, props, idToUuid)
    box.SetMetricsMapping(metrics)
}

func (box *Box) gatherLoop(ctx context.Context, interval time.Duration) {
    log.G(ctx).Debug("Initializing Porto metrics gather loop")

    t := time.NewTicker(interval)
    for {
        select {
        case <- ctx.Done():
            return
        case <- t.C:
            log.G(ctx).Debug("Porto metrics gather iteration")
            box.gatherMetrics(ctx)
        }
    }
}
