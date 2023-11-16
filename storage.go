//
// Storage management for query results.

package main

import (
	"fmt"
	"time"
)

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

// Collection of results.
type Results []result

////////////////////////////////////////////////////////////////////////////////
//
// Public
//
////////////////////////////////////////////////////////////////////////////////

// Get a result based on a timestamp. Returns the first result encountered
// which occurs after the provided time.
func (r *Results) Get(time time.Time) result {
	for _, result := range *r {
		if result.Time.Compare(time) >= 0 {
			// We found a time to return.
			return result
		}
	}

	// Return an empty result if nothing was discovered.
	return result{}
}

// Gets results based on a start and end timestamp.
func (r *Results) GetRange(startTime time.Time, endTime time.Time) (found []result) {
	for _, result := range *r {
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
func (r *Results) Put(value interface{}) interface{} {
	*r = append(*r, result{
		Time:  time.Now(),
		Value: value,
	})

	return value
}

// Show all currently stored results.
func (r *Results) Show() {
	for _, result := range *r {
		fmt.Printf("Time: %v, Value: %v\n", result.Time, result.Value)
	}
}

////////////////////////////////////////////////////////////////////////////////
//
// RPC
//
////////////////////////////////////////////////////////////////////////////////

type ArgsRPC struct{}

type ResultsRPC struct {
	Results *Results
}

func (r *Results) GetAllRPC(args *ArgsRPC, results *ResultsRPC) error {
	results.Results = r
	return nil
}
