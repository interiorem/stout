package exportmetrics

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	apexctx "github.com/m0sth8/context"
	"github.com/rcrowley/go-metrics"
)

type GraphiteExporter struct {
	prefix      string
	addr        string
	duStr       string
	du          time.Duration
	percentiles []float64
}

type GraphiteConfig struct {
	Prefix       string `json:"prefix"`
	Addr         string `json:"addr"`
	DurationUnit string `json:"duration"`
}

func NewGraphiteExporter(cfg *GraphiteConfig) (*GraphiteExporter, error) {
	if cfg.Prefix == "" {
		prefix, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		cfg.Prefix = strings.Replace(prefix, ".", "_", -1)
	}

	if cfg.DurationUnit == "" {
		cfg.DurationUnit = "1ms"
	}
	if !strings.HasPrefix(cfg.DurationUnit, "1") {
		return nil, fmt.Errorf("duStr must be 1<unit>: 1ms, 1sec, 1min")
	}
	du, err := time.ParseDuration(cfg.DurationUnit)
	if err != nil {
		return nil, err
	}

	return &GraphiteExporter{
		prefix:      cfg.Prefix,
		addr:        cfg.Addr,
		duStr:       cfg.DurationUnit[1:],
		du:          du,
		percentiles: []float64{0.5, 0.75, 0.95, 0.99, 0.999},
	}, nil
}

func (g *GraphiteExporter) Send(ctx context.Context, r metrics.Registry) error {
	d := net.Dialer{
		DualStack: true,
		Cancel:    ctx.Done(),
	}

	conn, err := d.Dial("tcp", g.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		conn.SetWriteDeadline(deadline)
	}

	w := bufio.NewWriter(conn)
	now := time.Now().Unix()
	r.Each(func(name string, value interface{}) {
		switch metric := value.(type) {
		case metrics.Counter:
			fmt.Fprintf(w, "%s.%s %d %d\n", g.prefix, name, metric.Count(), now)
		case metrics.Gauge:
			fmt.Fprintf(w, "%s.%s %d %d\n", g.prefix, name, metric.Value(), now)
		case metrics.Timer:
			t := metric.Snapshot()
			ps := t.Percentiles(g.percentiles)
			fmt.Fprintf(w, "%s.%s.count %d %d\n", g.prefix, name, t.Count(), now)
			fmt.Fprintf(w, "%s.%s.min_%s %d %d\n", g.prefix, name, g.duStr, t.Min()/int64(g.du), now)
			fmt.Fprintf(w, "%s.%s.max_%s %d %d\n", g.prefix, name, g.duStr, t.Max()/int64(g.du), now)
			fmt.Fprintf(w, "%s.%s.mean_%s %.2f %d\n", g.prefix, name, g.duStr, t.Mean()/float64(g.du), now)
			// fmt.Fprintf(w, "%s.%s.std-dev_%s %.2f %d\n", g.prefix, name, g.duStr, t.StdDev()/float64(g.du), now)
			for psIdx, psKey := range g.percentiles {
				key := strings.Replace(strconv.FormatFloat(psKey*100.0, 'f', -1, 64), ".", "", 1)
				fmt.Fprintf(w, "%s.%s.%s_%s %.2f %d\n", g.prefix, name, key, g.duStr, ps[psIdx]/float64(g.du), now)
			}
			fmt.Fprintf(w, "%s.%s.rate1m %.2f %d\n", g.prefix, name, t.Rate1(), now)
			fmt.Fprintf(w, "%s.%s.rate5m %.2f %d\n", g.prefix, name, t.Rate5(), now)
			fmt.Fprintf(w, "%s.%s.rate15m %.2f %d\n", g.prefix, name, t.Rate15(), now)
			fmt.Fprintf(w, "%s.%s.ratemean %.2f %d\n", g.prefix, name, t.RateMean(), now)
		default:
			apexctx.GetLogger(ctx).Warnf("Graphite: skip metric `%s` of unknown type %T", name, value)
		}
		w.Flush()
	})
	return nil
}
