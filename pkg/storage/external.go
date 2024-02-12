//
// External integrations.

package storage

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Interface for any external storage system.
type externalStorage interface {
	// Add a result to the external storage.
	Put(Result) error
}

// Prometheus Pushgateway specific external storage system.
type PushgatewayStorage struct {
	address  string                // Address to connect to Pushgateway.
	registry prometheus.Registerer // Prometheus registry to use.
}

// Add a result to Prometheus Pushgtateway.
func (p *PushgatewayStorage) Put(result Result) error {
	gauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "test",
			Name:      "foo",
			Help:      "Test gauge",
		},
	)
	p.registry.MustRegister(gauge)

	return nil
}

// Create a new storage for Pushgateway.
func NewPushgatewayStorage(address string) PushgatewayStorage {
	return PushgatewayStorage{
		address:  address,
		registry: prometheus.NewRegistry(),
	}
}
