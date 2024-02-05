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
	_ "fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "golang.org/x/exp/slog"
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
)

var (
	storageF     *os.File   // File for storage writes.
	storageMutex sync.Mutex // Lock for storage file writes.

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

// initializes a new results series in storage. Must be called when a new results series is created.
func (s *Storage) newResults(query string) {
	if _, ok := (*s)[query]; !ok {
		// This is a new query, initialize an empty results.
		(*s)[query] = &Results{}
		PutEvents[query] = make(chan Result, PUT_EVENT_CHANNEL_SIZE)
	}
}

// Saves current storage to disk. Currently this replaces the entire storage file with all the data
// in storage (a full write-over).
func (s *Storage) save() error {
	var (
		err         error  // General error holder.
		storageJson []byte // Bytes as json.
	)

	// Lock storage to prevent dirty writes.
	storageMutex.Lock()
	defer storageMutex.Unlock()

	// Translate current storage into binary json and save it.
	storageJson, err = json.MarshalIndent(&s, "", "\t")
	_, err = storageF.WriteAt(storageJson, 0)

	return err
}

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Public
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Initializes a new storage, loading in any saved storage data.
func NewStorage(persistence bool) (storage Storage, err error) {
	var (
		cryptarchUserCacheDir string              // Cryptarch specific user cache data.
		storageData           []byte              // Raw read storage data.
		storageFP             string              // Filepath for storage.
		storagePre            map[string]*Results // Pre-built storage with existing data.
		storageStat           fs.FileInfo         // Stat for the storage file.
		userCacheDir          string              // User cache directory, contextual to OS.
	)

	// Initialize storage.
	storage = Storage{}

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
	cryptarchUserCacheDir = filepath.Join(userCacheDir, "cryptarch")
	err = os.MkdirAll(cryptarchUserCacheDir, fs.FileMode(0770))
	if err != nil {
		return
	}

	// Instantiate the storage file.
	storageFP = filepath.Join(cryptarchUserCacheDir, "storage.json")
	storageF, err = os.OpenFile(
		storageFP,
		os.O_CREATE|os.O_RDWR,
		fs.FileMode(0770),
	)
	if err != nil {
		return
	}

	if storageStat, err = os.Stat(storageFP); storageStat.Size() > 0 {
		// There is pre-existing storage data.
		if err != nil {
			return
		}

		// Read in storage data and supply it to storage. We must initialize any results series before
		// populating data.
		storageData, err = io.ReadAll(storageF)
		if err != nil {
			return
		}
		json.Unmarshal(storageData, &storagePre)
		for query := range storagePre {
			storage.newResults(query)
		}
		storage = storagePre
	}

	return
}

// Closes a storage. Should be called after all storage operations cease.
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

// Initialize a new reader index. Will attempt to set the initial value to the end of existing
// results, if results already exist.
func (s *Storage) NewReaderIndex(query string) *ReaderIndex {
	var (
		r ReaderIndex // Reader index to return the address of.
	)

	if _, ok := (*s)[query]; !ok {
		// There is no data.
		r = ReaderIndex(0)
	} else {
		// There is existing data to account for.
		r = ReaderIndex(len((*s)[query].Results))
	}

	return &r
}

// Retrieve the next result from a put event channel.
func (s *Storage) Next(query string, index *ReaderIndex) (next Result) {
	next = <-PutEvents[query]
	index.inc()

	return
}

// Retrieve the next result from a put event channel, returning an empty result if nothing exists.
func (s *Storage) NextOrEmpty(query string, index *ReaderIndex) (next Result) {
	select {
	case next = <-PutEvents[query]:
		index.inc()
	default:
	}

	return
}

// Put a new compound result.
func (s *Storage) Put(
	query, value string,
	persistence bool,
	values ...interface{},
) (result Result, err error) {
	s.newResults(query)
	result = (*s)[query].put(value, values...)

	// Send a non-blocking put event. Put events are lossy and clients may lose information if not
	// actively listening.
	select {
	case PutEvents[query] <- result:
	default:
	}

	// Persist data to disk.
	if persistence {
		err = s.save()
	}

	return
}

// Assigns labels to a results series.
func (s *Storage) PutLabels(query string, labels []string) {
	s.newResults(query)
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
