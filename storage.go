package main

import (
	"fmt"
	"time"
)

// Old implementation with tstorage.

// import (
// 	"github.com/nakabonne/tstorage"
// )

// var Storage tstorage.Storage

// // Initializes storage.
// func init() {
// 	Storage, _ = tstorage.NewStorage(
// 		tstorage.WithTimestampPrecision(tstorage.Seconds),
// 	)
// }

// // Stores query results.
// func Get(metric string) []*tstorage.DataPoint {
// 	points, _ := Storage.Select(metric, nil, 1600000000, 1600000000+1)

// 	return points
// }

// // Retrieves query results.
// func Put(metric string, value float64) {
// 	Storage.InsertRows([]tstorage.Row{
// 		{
// 			DataPoint: tstorage.DataPoint{Timestamp: 1600000000, Value: value},
// 			Metric:    metric,
// 		},
// 	})
// }

////////////////////////////////////////////////////////////////////////////////
//
// Types
//
////////////////////////////////////////////////////////////////////////////////

// Individual result.
type result struct {
	Time  time.Time
	Value interface{}
}

////////////////////////////////////////////////////////////////////////////////
//
// Variables
//
////////////////////////////////////////////////////////////////////////////////

// Collection of results.
var results []result

////////////////////////////////////////////////////////////////////////////////
//
// Public
//
////////////////////////////////////////////////////////////////////////////////

// Get a result based on a timestamp. Returns the first result encountered
// which occurs after the provided time.
func Get(time time.Time) result {
	for _, result := range results {
		if result.Time.Compare(time) >= 0 {
			// We found a time to return.
			return result
		}
	}

	// Return an empty result if nothing was discovered.
	return result{}
}

// Gets results based on a start and end timestamp.
func GetRange(startTime time.Time, endTime time.Time) (found []result) {
	for _, result := range results {
		if result.Time.Compare(startTime) >= 0 {
			found = append(found, result)
		}

		if result.Time.Compare(endTime) > 0 {
			break
		}
	}

	return
}

// Put a new result.
func Put[T interface{}](value T) T {
	results = append(results, result{
		Time:  time.Now(),
		Value: value,
	})

	return value
}

// Show all currently stored results.
func Show() {
	for _, result := range results {
		fmt.Printf("Time: %v, Value: %v\n", result.Time, result.Value)
	}
}
