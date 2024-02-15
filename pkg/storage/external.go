//
// External integrations.

package storage

import (
	"fmt"
	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"golang.org/x/exp/slog"
)

const (
	PROMETHEUS_METRICS_HELP  = "Produced by Cryptarch." // Help text for all Prometheus metrics.
	PROMETHEUS_METRIC_LABEL  = "cryptarch_label"        // What Prometheus label to use for the Cryptarch label.
	PROMETHEUS_METRIC_PREFIX = "cryptarch"              // Prefix for all Prometheus metrics.
)

var (
	// Regular expression used for constructing Prometheus metric names. Represents the negation of
	// characters allowed in order to sanitize bad characters. We also replace the valid character ':'
	// as they are for recording rules.
	//
	// See: https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
	prometheus_metric_name_replace_regexp = regexp.MustCompile("[^a-zA-Z0-9_]")
)

// Interface for any external storage system.
type externalStorage interface {
	// Add a result to the external storage.
	Put(query string, labels []string, result Result) error
}

// Prometheus Pushgateway specific external storage system.
type PushgatewayStorage struct {
	address  string                // Address to connect to Pushgateway.
	registry prometheus.Registerer // Prometheus registry to use.
}

// Add a result to Prometheus Pushgtateway.
func (p *PushgatewayStorage) Put(query string, labels []string, result Result) error {
	var (
		name = queryToPromName(query)
		reg  = prometheus.NewRegistry()
	)

	metric, err := resultToPromMetric(name, labels, result)
	if err != nil {
		return err
	}
	reg.MustRegister(metric)

	slog.Debug(fmt.Sprintf("Pushing to Pushgtateway, name: %s, result: %v ...", name, result))
	push.New((*p).address, "test").Gatherer(reg).Push()

	return nil
}

// Converts a query string to something acceptable as a Prometheus metric name.
func queryToPromName(query string) string {
	return string(prometheus_metric_name_replace_regexp.ReplaceAll([]byte(query), []byte("_")))
}

// Converts a result to a Prometheus metric.
func resultToPromMetric(
	name string,
	labels []string,
	result Result,
) (metric *prometheus.GaugeVec, err error) {
	metric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_%s", PROMETHEUS_METRIC_PREFIX, name),
			Help: PROMETHEUS_METRICS_HELP,
		},
		[]string{PROMETHEUS_METRIC_LABEL},
	)

	for i, value := range result.Values {
		switch value.(type) {
		case int64:
			metric.With(prometheus.Labels{PROMETHEUS_METRIC_LABEL: labels[i]}).Set(float64(value.(int64)))
		case float64:
			metric.With(prometheus.Labels{PROMETHEUS_METRIC_LABEL: labels[i]}).Set(value.(float64))
		default:
			// We encountered a value Prometheus can't digest.
			err = &NaNError{Value: value}

			// TODO For now, we give-up if any value is non-pushable. In the future, we might consider
			// still attempting to push some values, but this would also require better error handling in
			// `Put`.
			break
		}
	}

	return
}

// Create a new storage for Pushgateway.
func NewPushgatewayStorage(address string) PushgatewayStorage {
	return PushgatewayStorage{
		address:  address,
		registry: prometheus.NewRegistry(),
	}
}
