//
// Storage management for query results.
//
// The storage engine is a simple, time-series indexed, multi-type data store. It interfaces as a Go
// library in a few ways, namely:
//
// - It can be used as a library.
// - It can broadcast events into public Go channels.
// - It can broadcast events via RPC.
//
// Results are stored simply in an ordered sequence, and querying time is linear.

package storage

import (
	"io"
	"os"
	"time"
)

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Types
//
///////////////////////////////////////////////////////////////////////////////////////////////////

type ReaderIndex int

// Collection of results mapped to their queries.
type Storage map[string]*Results

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Variables
//
///////////////////////////////////////////////////////////////////////////////////////////////////

const (
	// Size of Put channels. This is the amount of results that may accumulate if not being actively
	// consumed.
	PUT_EVENT_CHANNEL_SIZE = 128
	// Persistent storage path.
	STORAGE_FILE = "/tmp/storage"
)

var (
	storageF      *os.File      // File for storage writes.
	storageWriter io.ByteWriter // Writer for storage.

	PutEvents = make(map[string](chan Result)) // Channels for broadcasting Put calls.
)

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Private
//
///////////////////////////////////////////////////////////////////////////////////////////////////

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
func NewStorage() (storage Storage, err error) {
	storageF, err = os.Create(STORAGE_FILE)
	storage = Storage{}
	return
}

// Closes a storage.
func (s *Storage) Close() {
	storageF.Close()
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

// Retrieve the next result from a put event channel.
func (s *Storage) Next(query string, index *ReaderIndex) (next Result) {
	next = <-PutEvents[query]
	index.inc()
	return
}

// Retrieve the next result from a put event channel, returning an empty result
// if nothing exists.
func (s *Storage) NextOrEmpty(query string, index *ReaderIndex) (next Result) {
	select {
	case next = <-PutEvents[query]:
		index.inc()
	default:
	}
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

	// Persist data to disk.
	// storageF.Write([]byte(result.Value.(string)))

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
