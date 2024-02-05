//
// Manages individual or series of results for storage.

package storage

import (
	"fmt"
	"reflect"
	"time"

	"golang.org/x/exp/slices"
)

// Individual result.
type Result struct {
	Time   time.Time     // Time the result was created.
	Value  interface{}   // Raw value of the result.
	Values []interface{} // Tokenized value of the result.
}

// Collection of results.
type Results struct {
	Labels  []string // Meta field for result values acting as a name, corresponding by index.
	Results []Result // Stored results.
}

// Get a result based on a timestamp.
func (r *Results) get(time time.Time) Result {
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
func (r *Results) getRange(startTime time.Time, endTime time.Time) (found []Result) {
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

// Given a filter, return the corresponding value index.
func (r *Results) getValueIndex(filter string) int {
	return slices.Index((*r).Labels, filter)
}

// Put a new compound result.
func (r *Results) put(value string, values ...interface{}) Result {
	next := Result{
		Time:   time.Now(),
		Value:  value,
		Values: values,
	}

	(*r).Results = append((*r).Results, next)

	return next
}

// Show all currently stored results.
func (r *Results) show() {
	for _, result := range (*r).Results {
		fmt.Printf("Label: %v, Time: %v, Value: %v, Values: %v\n",
			(*r).Labels, result.Time, result.Value, result.Values)
	}
}

// Determines whether this is an empty result.
func (r *Result) IsEmpty() bool {
	return reflect.DeepEqual(*r, Result{})
}
