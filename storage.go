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
	Time   time.Time
	Value  interface{}
	Values []interface{}
}

// Collection of results.
type Results []result

////////////////////////////////////////////////////////////////////////////////
//
// Variables
//
////////////////////////////////////////////////////////////////////////////////

var PutEvents = make(chan result, 128)

////////////////////////////////////////////////////////////////////////////////
//
// Public
//
////////////////////////////////////////////////////////////////////////////////

// Get a result based on a timestamp.
func (r *Results) Get(time time.Time) result {
	for _, result := range *r {
		if result.Time.Compare(time) == 0 {
			// We found a result to return.
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
			if result.Time.Compare(endTime) > 0 {
				// Break out of the loop if we've exhausted the upper bounds of the
				// range.
				break
			} else {
				found = append(found, result)
			}
		}
	}

	return
}

// Put a new result.
func (r *Results) Put(value interface{}) interface{} {
	next := result{
		Time:   time.Now(),
		Value:  value,
		Values: nil,
	}

	*r = append(*r, next)
	PutEvents <- next

	return value
}

// Put a new compound result.
func (r *Results) PutC(values ...interface{}) []interface{} {
	next := result{
		Time:   time.Now(),
		Value:  nil,
		Values: values,
	}

	*r = append(*r, next)
	PutEvents <- next

	return values
}

// Show all currently stored results.
func (r *Results) Show() {
	for _, result := range *r {
		fmt.Printf("Time: %v, Value: %v, Values: %v\n", result.Time, result.Value, result.Values)
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
