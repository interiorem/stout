package metrics

import (
	"bytes"
	"encoding/json"
	"expvar"

	"github.com/rcrowley/go-metrics"
)

var (
	requestedPercentiles            = []float64{0.5, 0.75, 0.9, 0.95, 0.98, 0.99, 0.9995}
	_                    expvar.Var = TimerVar{}
)

// TimerVar adds expvar.Var interface to go-metrics.Timer
type TimerVar struct {
	metrics.Timer
}

// NewTimerVar returns new TimerVar with go-metrics.StandartTimer inside
func NewTimerVar() TimerVar {
	return TimerVar{
		Timer: metrics.NewTimer(),
	}
}

type stats struct {
	Sum    int64   `json:"sum"`
	Min    int64   `json:"min"`
	Max    int64   `json:"max"`
	Mean   float64 `json:"mean"`
	Rate1  float64 `json:"rate1"`
	Rate5  float64 `json:"rate5"`
	Rate15 float64 `json:"rate15"`
	Q50    float64 `json:"50%"`
	Q75    float64 `json:"75%"`
	Q90    float64 `json:"90%"`
	Q95    float64 `json:"95%"`
	Q98    float64 `json:"98%"`
	Q99    float64 `json:"99%"`
	Q9995  float64 `json:"99.95%"`
}

func (t TimerVar) String() string {
	ss := t.Snapshot()
	percentiles := ss.Percentiles(requestedPercentiles)
	var st = stats{
		Min:    ss.Min(),
		Max:    ss.Max(),
		Mean:   ss.Mean(),
		Rate1:  ss.Rate1(),
		Rate5:  ss.Rate5(),
		Rate15: ss.Rate15(),
		Sum:    ss.Sum(),
		Q50:    percentiles[0],
		Q75:    percentiles[1],
		Q90:    percentiles[2],
		Q95:    percentiles[3],
		Q98:    percentiles[4],
		Q99:    percentiles[5],
		Q9995:  percentiles[6],
	}

	buff := new(bytes.Buffer)
	json.NewEncoder(buff).Encode(st)
	return buff.String()
}
