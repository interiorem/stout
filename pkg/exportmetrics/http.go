package exportmetrics

import (
	"net/http"

	"github.com/rcrowley/go-metrics"
)

func HTTPExport(registry metrics.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics.WriteJSONOnce(registry, w)
	}
}
