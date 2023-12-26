//
// Storage management for query results.

package storage

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
type Result struct {
	Time   time.Time
	Value  interface{}
	Values []interface{}
}

// Collection of results.
type Results struct {
	Labels  []string
	Results []Result
}

////////////////////////////////////////////////////////////////////////////////
//
// Variables
//
////////////////////////////////////////////////////////////////////////////////

var PutEvents = make(chan Result, 128)

////////////////////////////////////////////////////////////////////////////////
//
// Public
//
////////////////////////////////////////////////////////////////////////////////

// Get a result based on a timestamp.
func (r *Results) Get(time time.Time) Result {
	for _, result := range (*r).Results {
		if result.Time.Compare(time) == 0 {
			// We found a result to return.
			return result
		}
	}

	// Return an empty result if nothing was discovered.
	return Result{}
}

// Gets results based on a start and end timestamp.
func (r *Results) GetRange(startTime time.Time, endTime time.Time) (found []Result) {
	for _, result := range (*r).Results {
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

// Put a new compound result.
func (r *Results) Put(value string, values ...interface{}) []interface{} {
	next := Result{
		Time:   time.Now(),
		Value:  value,
		Values: values,
	}

	(*r).Results = append((*r).Results, next)
	PutEvents <- next

	return values
}

// Show all currently stored results.
func (r *Results) Show() {
	for _, result := range (*r).Results {
		fmt.Printf("Label: %v, Time: %v, Value: %v, Values: %v\n",
			(*r).Labels, result.Time, result.Value, result.Values)
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
