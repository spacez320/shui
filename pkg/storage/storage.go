//
// Storage management for query results.
//
// The storage engine is a simple, time-series indexed, multi-type data store.
// It interfaces as a Go library in a few ways, namely:
//
// - It can be used as a library.
// - It can broadcast events into public Go channels.
// - It can broadcast events via RPC.
//
// Results are stored simply in an ordered sequence, and querying time is
// linear.

package storage

import (
	"fmt"
	"time"

	"golang.org/x/exp/slices"
)

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Types
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Individual result.
type Result struct {
	// Time the result was created.
	Time time.Time
	// Raw value of the result.
	Value interface{}
	// Tokenized value of the result.
	Values []interface{}
}

// Collection of results.
type Results struct {
	// Meta field for result values acting as a name, corresponding by index.
	Labels []string
	// Stored results.
	Results []Result
}

type ReaderIndex int

// Collection of results mapped to their queries.
type Storage map[string]*Results

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Variables
//
///////////////////////////////////////////////////////////////////////////////////////////////////

const (
	PUT_EVENT_CHANNEL_SIZE = 128 // Size of Put channels.
)

// Channels for broadcasting Put calls.
var PutEvents = make(map[string](chan Result))

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Private
//
///////////////////////////////////////////////////////////////////////////////////////////////////

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

// Incremement a reader index, likely after a read.
func (i *ReaderIndex) inc() {
	(*i)++
}

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Public
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Initializes a new storage.
func NewStorage() Storage {
	return Storage{}
}

// Get a result based on a timestamp.
func (s *Storage) Get(query string, time time.Time) Result {
	return (*s)[query].get(time)
}

func (s *Storage) GetAll(query string) []Result {
	return (*s)[query].Results
}

// Get a result's labels.
func (s *Storage) GetLabels(query string) []string {
	return (*s)[query].Labels
}

// Gets results based on a start and end timestamp.
func (s *Storage) GetRange(query string, startTime, endTime time.Time) []Result {
	return (*s)[query].getRange(startTime, endTime)
}

func (s *Storage) GetToIndex(query string, index *ReaderIndex) []Result {
	return (*s)[query].Results[:*index]
}

// Given a filter, return the corresponding value index.
func (s *Storage) GetValueIndex(query, filter string) int {
	return (*s)[query].getValueIndex(filter)
}

// Initialize a new reader index.
func (s *Storage) NewReaderIndex() *ReaderIndex {
	r := ReaderIndex(0)
	return &r
}

// Initializes a new results series in a storage.
func (s *Storage) NewResults(query string) {
	if _, ok := (*s)[query]; !ok {
		// This is a new query, initialize an empty results.
		(*s)[query] = &Results{}
		PutEvents[query] = make(chan Result, PUT_EVENT_CHANNEL_SIZE)
	}
}

func (s *Storage) Next(query string, index *ReaderIndex) (next Result) {
	next = <-PutEvents[query]
	index.inc()
	return
}

// Put a new compound result.
func (s *Storage) Put(query string, value string, values ...interface{}) Result {
	s.NewResults(query)
	result := (*s)[query].put(value, values...)

	// Send a non-blocking put event. Put events are lossy and clients may lose
	// information if not actively listening.
	select {
	case PutEvents[query] <- result:
	default:
	}

	return result
}

// Assigns labels to a results series.
func (s *Storage) PutLabels(query string, labels []string) {
	s.NewResults(query)
	(*s)[query].Labels = labels
}

// Show all currently stored results.
func (s *Storage) Show(query string) {
	(*s)[query].show()
}

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// RPC
//
///////////////////////////////////////////////////////////////////////////////////////////////////

type ArgsRPC struct{}

type ResultsRPC struct {
	Results *Results
}

func (r *Results) GetAllRPC(args *ArgsRPC, results *ResultsRPC) error {
	results.Results = r
	return nil
}
