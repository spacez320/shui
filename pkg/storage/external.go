//
// External integrations.

package storage

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"
)

const (
	DUMMY_OUTBOUND_ADDR         = "8.8.8.8:80"             // Some outbound address for dummy requests.
	PROMETHEUS_JOB              = "cryptarch"              // What to apply for the Prometheus job.
	PROMETHEUS_METRICS_ENDPOINT = "/results"               // Endpoint where Prometheus metrics are presented.
	PROMETHEUS_METRICS_HELP     = "Produced by Cryptarch." // Help text for all Prometheus metrics.
	PROMETHEUS_METRIC_LABEL     = "cryptarch_label"        // What Prometheus label to use for the Cryptarch label.
	PROMETHEUS_METRIC_PREFIX    = "cryptarch"              // Prefix for all Prometheus metrics.
)

var (
	// Regular expression used for constructing strings (names, labels, etc.) safely useable by
	// external sources. Represents the negation of characters allowed in order to sanitize unwanted
	// characters.
	//
	// For Prometheus, we also replace the valid character ':' as they are for recording rules.
	//
	// See: https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
	normalize_regexp = regexp.MustCompile("[^a-zA-Z0-9_]+")
)

// Interface for any external storage system.
type externalStorage interface {
	// Add a result to the external storage.
	Put(query string, labels []string, result Result) error
}

////////////////////////////////////////////////////////////////////////////////////////////////////
//
// ElasticsearchStorage
//
////////////////////////////////////////////////////////////////////////////////////////////////////

type ElasticsearchStorage struct {
	address  string // Address to connect to Elasticsearch.
	index    string // Index to supply documents to.
	user     string // Username for HTTP Baisc Auth.
	password string // Password for HTTP Baisc Auth.
}

func (e *ElasticsearchStorage) Put(query string, labels []string, result Result) error {
	var (
		es      *elasticsearch.Client // Client for querying Elasticsearch.
		payload []byte                // Query payload to turn results into documents.

		// Configuration for the Elasticsearch client.
		esConfig = elasticsearch.Config{
			Addresses: []string{e.address},
			Password:  e.password,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			Username: e.user,
		}
	)

	// Initialize an Elasticsearch client.
	es, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return err
	}

	// Build the document body.
	payload, err = resultToElasticsearchDocument(query, labels, result)
	if err != nil {
		return err
	}

	slog.Debug("Pushing to Elasticsearch", "result", result)
	es.Index(e.index, bytes.NewReader(payload))

	return nil
}

// Creates a new storage for Elasticsearch.
func NewElasticsearchStorage(address, index, password, user string) ElasticsearchStorage {
	return ElasticsearchStorage{
		address:  address,
		index:    index,
		password: password,
		user:     user,
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
//
// PushgatewayStorage
//
////////////////////////////////////////////////////////////////////////////////////////////////////

// Prometheus Pushgateway specific external storage system.
type PushgatewayStorage struct {
	address  string               // Address to connect to Pushgateway.
	registry *prometheus.Registry // Prometheus registry to use.
}

// Add a result to Prometheus Pushgtateway.
func (p *PushgatewayStorage) Put(query string, labels []string, result Result) error {
	var (
		err      error                // General error holder.
		instance string               // Prometheus instance value.
		metric   *prometheus.GaugeVec // Produced metric to push.

		name = normalizeString(query) // Name for the metric.
	)

	// Get the instance value.
	instance, err = getPromInstance()
	if err != nil {
		return err
	}

	// Build the metric.
	metric, err = resultToPromMetric(name, labels, result)
	if err != nil {
		return err
	}
	(*p).registry.Register(metric)

	slog.Debug("Pushing to Pushgtateway", "name", name, "result", result)
	push.New((*p).address, PROMETHEUS_JOB).Grouping("instance", instance).Gatherer((*p).registry).Push()

	return nil
}

// Create a new storage for Pushgateway.
func NewPushgatewayStorage(address string) PushgatewayStorage {
	return PushgatewayStorage{
		address:  address,
		registry: prometheus.NewRegistry(),
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
//
// PrometheusStorage
//
////////////////////////////////////////////////////////////////////////////////////////////////////

type PrometheusStorage struct {
	registry *prometheus.Registry // Prometheus registry to use.
}

// Register a result in a Prometheus registry.
func (p *PrometheusStorage) Put(query string, labels []string, result Result) error {
	var (
		err    error                // General error holder.
		metric *prometheus.GaugeVec // Produced metric to push.

		name = normalizeString(query) // Name for the metric.
	)

	// Build the metric.
	metric, err = resultToPromMetric(name, labels, result)
	if err != nil {
		return err
	}
	(*p).registry.Register(metric)

	slog.Debug("Pushing to Prometheus", "name", name, "result", result)

	return nil
}

// Create a new storage for Prometheus.
func NewPrometheusStorage(address string) PrometheusStorage {
	var registry = prometheus.NewRegistry()

	// Start the metrics endpoint for results.
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry}))
	go http.ListenAndServe(address, nil)

	return PrometheusStorage{
		registry: registry,
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
//
// Private Functions
//
////////////////////////////////////////////////////////////////////////////////////////////////////

// Get an instance value for Prometheus metrics.
func getPromInstance() (localIP string, err error) {
	var (
		localAddrs []net.Addr // Interface addresses.
	)

	// Make a dummy request to get the default outbound IP. Kind of hacky.
	conn, err := net.Dial("udp", DUMMY_OUTBOUND_ADDR)
	if err != nil {
		// We couldn't make anoutbound connection--try to just list interfaces and grab the first one.
		localAddrs, err = net.InterfaceAddrs()
		if err != nil {
			return "", err
		}
		localIP = localAddrs[0].(*net.IPNet).IP.String()
	} else {
		// We got an accurate local IP.
		defer conn.Close()
		localIP = conn.LocalAddr().(*net.UDPAddr).IP.String()
	}

	return localIP, err
}

// Converts a string to something acceptable as a name or label useable by external sources.
func normalizeString(s string) string {
	// The operations are:
	//
	// 1. Replace all non-alphanumeric characters with an underscore.
	// 2. Replace multiple, adjacent underscores with a single underscore.
	// 3. Trim extra underscore prefixes and suffixes.
	return strings.Trim(string(regexp.MustCompile("_+").ReplaceAll(
		normalize_regexp.ReplaceAll([]byte(s), []byte("_")),
		[]byte("_"),
	)), "_")
}

// Converts a result to an Elasticsearch document.
func resultToElasticsearchDocument(
	query string,
	labels []string,
	result Result,
) (document []byte, err error) {
	// Payload to construct the document from, accounting for the two additional fields added.
	var payload = make(map[string]interface{}, len(labels)+2)

	// Add fields for each value.
	for k, v := range result.Map(labels) {
		payload[fmt.Sprintf("cryptarch.value.%s", normalizeString(k))] = v
	}

	// Add additional fields to the payload.
	payload["timestamp"] = result.Time
	payload["cryptarch.query"] = query

	// Build the document body.
	document, err = json.Marshal(payload)

	return
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
			metric.With(prometheus.Labels{
				PROMETHEUS_METRIC_LABEL: labels[i],
			}).Set(float64(value.(int64)))
		case float64:
			metric.With(prometheus.Labels{
				PROMETHEUS_METRIC_LABEL: labels[i],
			}).Set(value.(float64))
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

////////////////////////////////////////////////////////////////////////////////////////////////////
//
// Public Functions
//
////////////////////////////////////////////////////////////////////////////////////////////////////
