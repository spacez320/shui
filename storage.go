package main

import (
	"github.com/nakabonne/tstorage"
)

var Storage tstorage.Storage

// Initializes storage.
func init() {
	Storage, _ = tstorage.NewStorage(
		tstorage.WithTimestampPrecision(tstorage.Seconds),
	)
}

// Stores query results.
func Get(metric string) []*tstorage.DataPoint {
	points, _ := Storage.Select(metric, nil, 1600000000, 1600000000+1)

	return points
}

// Retrieves query results.
func Put(metric string, value float64) {
	Storage.InsertRows([]tstorage.Row{
		{
			DataPoint: tstorage.DataPoint{Timestamp: 1600000000, Value: value},
			Metric:    metric,
		},
	})
}
