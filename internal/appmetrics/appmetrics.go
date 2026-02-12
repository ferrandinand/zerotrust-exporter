package appmetrics

import (
	"github.com/VictoriaMetrics/metrics"
)

// Prometheus Endpoint metrics
var (
	UpMetric         = metrics.NewGauge("zerotrust_exporter_up", nil)
	ScrapeDuration   = metrics.NewHistogram("zerotrust_exporter_scrape_duration_seconds")
	ApiCallCounter   = metrics.NewCounter("zerotrust_exporter_api_calls_total")
	ApiErrorsCounter = metrics.NewCounter("zerotrust_exporter_api_errors_total")
)

func init() {
	// Set the up metric to 1 on initialization
	UpMetric.Set(1)
}

func SetScrapeDuration(value float64) {
	ScrapeDuration.Update(value)
}

func IncApiCallCounter() {
	ApiCallCounter.Inc()
}

func IncApiErrorsCounter() {
	ApiErrorsCounter.Inc()
}
