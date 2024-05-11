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
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

const (
	MAX_EXTERNAL_STORAGES  = 128            // Maximum external storage integrations.
	MAX_RESULTS            = 128            // Maximum number of result series that may be maintained.
	PUT_EVENT_CHANNEL_SIZE = 128            // Size of put channels, controlling the amount of waiting results.
	STORAGE_FILE_DIR       = "cryptarch"    // Directory in user cache to use for storage.
	STORAGE_FILE_NAME      = "storage.json" // Filename to use for actual storage.
)

// Returns a results series that has been filtered to a specific set of labels.
func filterResult(query string, filters, labels []string, result Result) (filteredResult Result) {
	var (
		filteredIndexes = make([]int, len(filters))         // Indexes corresponding to filtered labels.
		filteredValues  = make([]interface{}, len(filters)) // Values after filtering.
	)

	if len(filters) > 0 {
		// Find indexes to pursue for results.
		for i, filter := range filters {
			filteredIndexes[i] = slices.Index(labels, filter)
		}

		// Filter the results.
		filteredValues = filterSlice(result.Values, filteredIndexes)

		// Reconstruct the result with filtered values.
		filteredResult = Result{
			Time:   result.Time,
			Value:  result.Value,
			Values: filteredValues,
		}
	} else {
		// If not filters were provided, just return the result itself.
		filteredResult = result
	}

	return
}

// Pick items from an arbitrary slice according to provided indexes. If indexes is empty, it will
// just return the original slice.
func filterSlice[T interface{}](in []T, indexes []int) (out []T) {
	if len(indexes) == 0 {
		out = in
	} else {
		for _, index := range indexes {
			out = append(out, in[index])
		}
	}

	return
}

// Collection of results mapped to their queries.
type Storage struct {
	externalStorages []externalStorage        // Integrated external storages.
	putEventChans    map[string](chan Result) // Map of queries to put even channels.
	storageFile      *os.File                 // File for persisting results.
	storageMutex     *sync.Mutex              // Mutex for managing persistence writes.

	Results map[string]*Results // Map of queries to results.
}

// Initializes a new results series in storage. Must be called when a new results series is created.
// This function is idempotent in that it will check if results for a query have already been
// initialized and pass silently if so.
func (s *Storage) newResults(query string, size int) {
	var (
		results Results // Results to initialize.
	)

	if _, ok := (*s).Results[query]; !ok {
		// Initialize results.
		results = newResults(size)
		(*s).Results[query] = &results

		// Initialize the query's put event channel.
		(*s).putEventChans[query] = make(chan Result, PUT_EVENT_CHANNEL_SIZE)
	}
}

// Saves current storage to disk. Currently this replaces the entire storage file with all the data
// in storage (a full write-over).
func (s *Storage) save() error {
	var (
		err         error  // General error holder.
		resultsJson []byte // Bytes as json.
	)

	// Lock storage to prevent dirty writes.
	(*s).storageMutex.Lock()
	defer (*s).storageMutex.Unlock()

	// Translate current storage results into binary json and save it.
	resultsJson, err = json.MarshalIndent(&s.Results, "", "\t")
	_, err = (*s).storageFile.WriteAt(resultsJson, 0)

	return err
}

// Adds an external storage.
func (s *Storage) AddExternalStorage(e externalStorage) {
	(*s).externalStorages = append((*s).externalStorages, e)
}

// Closes a storage. Should be called after all storage operations cease.
func (s *Storage) Close() {
	(*s).storageFile.Close()
}

// Get a result based on a timestamp.
func (s *Storage) Get(query string, time time.Time) Result {
	return (*s).Results[query].get(time)
}

// Get all results.
func (s *Storage) GetAll(query string) []Result {
	return (*s).Results[query].Results
}

// Get a result's labels.
func (s *Storage) GetLabels(query string, filters []string) []string {
	var (
		filteredIndexes = make([]int, len(filters))  // Indexes for filtering.
		labels          = (*s).Results[query].Labels // Labels associated with this query.
	)

	// Filter labels, if needed.
	if len(filters) > 0 {
		for i, filter := range filters {
			filteredIndexes[i] = slices.Index(labels, filter)
		}

		labels = filterSlice(labels, filteredIndexes)
	}

	return labels
}

// Gets results based on a start and end timestamp.
func (s *Storage) GetRange(query string, startTime, endTime time.Time) []Result {
	return (*s).Results[query].getRange(startTime, endTime)
}

// Given results up to a reader index (a.k.a. "playback").
func (s *Storage) GetToIndex(query string, filters []string, index *ReaderIndex) []Result {
	var (
		results         = (*s).Results[query].Results[:(*index)+1] // Queried results.
		filteredResults = make([]Result, len(results))             // Results after filtering.
		labels          = (*s).Results[query].Labels               // Labels associated with this query.
	)

	for i, result := range results {
		filteredResults[i] = filterResult(query, filters, labels, result)
	}

	return filteredResults
}

// Given a filter, return the corresponding value index.
func (s *Storage) GetValueIndex(query, filter string) int {
	return (*s).Results[query].getValueIndex(filter)
}

// Initialize a new reader index. Will attempt to set the initial value to the end of existing
// results, if results already exist.
func (s *Storage) NewReaderIndex(query string) *ReaderIndex {
	var (
		reader ReaderIndex // Reader index to initialize.
	)

	if _, ok := (*s).Results[query]; !ok {
		// There is no data.
		reader = ReaderIndex(0)
	} else {
		// There is existing data to account for.
		reader = ReaderIndex(len((*s).Results[query].Results))
	}

	return &reader
}

// Retrieve the next result from a put event channel, blocking if none exists.
func (s *Storage) Next(query string, filters []string, reader *ReaderIndex) (next Result) {
	// Read from the event channel.
	next = <-(*s).putEventChans[query]
	reader.Inc()

	// Apply filters.
	next = filterResult(query, filters, (*s).Results[query].Labels, next)

	return
}

// Retrieve the next result from a put event channel, returning an empty result if nothing exists.
func (s *Storage) NextOrEmpty(query string, reader *ReaderIndex) (next Result) {
	select {
	case next = <-(*s).putEventChans[query]:
		// Only increment the read counter if something consumed the event.
		reader.Inc()
	default:
	}

	return
}

// Put a new result.
func (s *Storage) Put(
	query, value string,
	persistence bool,
	values ...interface{},
) (result Result, err error) {
	// Initialize the result.
	s.newResults(query, len(values))
	result = (*s).Results[query].put(value, values...)

	slog.Debug(
		"Placing results in storage",
		"value",
		value,
		"values",
		values,
		"labels",
		(*s).Results[query].Labels,
	)

	// Send a non-blocking put event. Put events are lossy and clients may lose information if not
	// actively listening.
	select {
	case (*s).putEventChans[query] <- result:
	default:
	}

	// Persist data to disk.
	if persistence {
		err = s.save()
	}
	if err != nil {
		return
	}

	// Persist data to external sources.
	for _, externalStore := range (*s).externalStorages {
		err = externalStore.Put(query, (*s).Results[query].Labels, result)
		if err != nil {
			return
		}
	}

	return
}

// Assigns explicit labels to a results series.
func (s *Storage) PutLabels(query string, labels []string) {
	s.newResults(query, len(labels))
	(*s).Results[query].Labels = labels
}

// Show all currently stored results.
func (s *Storage) Show(query string) {
	(*s).Results[query].show()
}

// Initializes a new storage, loading in any saved storage data.
func NewStorage(persistence bool) (storage Storage, err error) {
	var (
		cryptarchUserCacheDir string              // Cryptarch specific user cache data.
		results               map[string]*Results // Pre-built storage with existing data.
		storageData           []byte              // Raw read storage data.
		storageFilepath       string              // Filepath for storage.
		storageStat           fs.FileInfo         // Stat for the storage file.
		userCacheDir          string              // User cache directory, contextual to OS.
	)

	// Initialize storage.
	storage = Storage{
		Results:       make(map[string]*Results, MAX_RESULTS),
		putEventChans: make(map[string](chan Result), MAX_RESULTS),
		storageMutex:  &sync.Mutex{},
	}

	// If we have disabled persistence, simply return the new storage instance.
	if !persistence {
		return
	}

	// Retrieve the user cache directory.
	userCacheDir, err = os.UserCacheDir()
	if err != nil {
		return
	}

	// Create the user cache directory for data.
	cryptarchUserCacheDir = filepath.Join(userCacheDir, STORAGE_FILE_DIR)
	err = os.MkdirAll(cryptarchUserCacheDir, fs.FileMode(0770))
	if err != nil {
		return
	}

	// Instantiate the storage file.
	storageFilepath = filepath.Join(cryptarchUserCacheDir, STORAGE_FILE_NAME)
	storage.storageFile, err = os.OpenFile(
		storageFilepath,
		os.O_CREATE|os.O_RDWR,
		fs.FileMode(0770),
	)
	if err != nil {
		return
	}

	if storageStat, err = os.Stat(storageFilepath); storageStat.Size() > 0 {
		// There is pre-existing storage data.
		if err != nil {
			return
		}

		// Read in storage data and supply it to storage. We must initialize any results series before
		// populating data.
		storageData, err = io.ReadAll(storage.storageFile)
		if err != nil {
			return
		}
		json.Unmarshal(storageData, &results)

		for query := range results {
			// TODO Results loading should also preserve and restore actual labels.
			storage.newResults(query, len(results[query].Results[0].Values))
			storage.Results[query] = results[query]
		}
	}

	return
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
