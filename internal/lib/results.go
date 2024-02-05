//
// Results management.
//
// Managing results involves:
//
// -  Organizing a storage of results.
// -  Managing the TUI libraries--rendering and interaction for results.
// -  Finding a place for accessory output, like logs.

package lib

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/scanner"
	"time"
	"unicode"

	"pkg/storage"

	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

var (
	config          Config                          // Global configuration.
	currentCtx      context.Context                 // Current context.
	driver          DisplayDriver                   // Display driver, dictated by the results.
	pauseQueryChans map[string]chan bool            // Channels for dealing with 'pause' events for results.
	readerIndexes   map[string]*storage.ReaderIndex // Reader indexes for queries.
	store           storage.Storage                 // Stored results.

	ctxDefaults = map[string]interface{}{
		"advanceDisplayMode": false,
		"advanceQuery":       false,
		"quit":               false,
	} // Defaults applied to context.
	pauseDisplayChan = make(chan bool) // Channel for dealing with 'pause' events for the display.
)

// Resets the current context to its default values.
func resetContext(query string) {
	for k, v := range ctxDefaults {
		currentCtx = context.WithValue(currentCtx, k, v)
	}
	currentCtx = context.WithValue(currentCtx, "query", query)
}

// Adds a result to the result store based on a string.
func AddResult(query, result string, history bool) {
	result = strings.TrimSpace(result)
	_, err := store.Put(query, result, history, TokenizeResult(result)...)
	e(err)
}

// Retrieves a next result.
func GetResult(query string) storage.Result {
	return store.Next(query, readerIndexes[query])
}

// Retrieves a next result, waiting for a non-empty return.
func GetResultWait(query string) (result storage.Result) {
	for {
		if result = store.NextOrEmpty(query, readerIndexes[query]); result.IsEmpty() {
			// Wait a tiny bit if we receive an empty result to avoid an excessive amount of busy waiting.
			// This wait time should be less than the query delay, otherwise displays will show a release
			// of buffered results.
			time.Sleep(time.Duration(10) * time.Millisecond)
		} else {
			// We found a result.
			break
		}
	}

	return
}

// Creates a result with filtered values.
func FilterResult(result storage.Result, labels, filters []string) storage.Result {
	var (
		labelIndexes = make([]int, len(filters))         // Indexes of labels from filters, corresponding to result values.
		resultValues = make([]interface{}, len(filters)) // Found result values.
	)

	// Find indexes to pursue for results.
	for i, filter := range filters {
		labelIndexes[i] = slices.Index(labels, filter)
	}

	// Filter the results.
	resultValues = FilterSlice(result.Values, labelIndexes)

	return storage.Result{
		Time:   result.Time,
		Value:  result.Value,
		Values: resultValues,
	}
}

// Parses a result into tokens for compound storage.
func TokenizeResult(result string) (parsedResult []interface{}) {
	var (
		s    scanner.Scanner // Scanner for tokenization.
		next string          // Next token to consider.
	)

	s.Init(strings.NewReader(result))
	s.IsIdentRune = func(r rune, i int) bool {
		// Separate all tokens exclusively by whitespace.
		return !unicode.IsSpace(r)
	}

	for token := s.Scan(); token != scanner.EOF; token = s.Scan() {
		next = s.TokenText()

		// Attempt to parse this value as an integer.
		nextInt, err := strconv.ParseInt(next, 10, 0)
		if err == nil {
			parsedResult = append(parsedResult, nextInt)
			continue
		}

		// Attempt to parse this value as a float.
		nextFloat, err := strconv.ParseFloat(next, 10)
		if err == nil {
			parsedResult = append(parsedResult, nextFloat)
			continue
		}

		// Everything else has failed--just pass it as a string.
		parsedResult = append(parsedResult, next)
	}

	return
}

// Entry-point function for results.
func Results(
	ctx context.Context,
	displayMode DisplayMode,
	query string,
	history bool,
	inputConfig Config,
	inputPauseQueryChans map[string]chan bool,
) {
	var (
		err error // General error holder.

		filters = ctx.Value("filters").([]string) // Capture filters from context.
		labels  = ctx.Value("labels").([]string)  // Capture labels from context.
		queries = ctx.Value("queries").([]string) // Capture queries from context.
	)

	// Initialize storage.
	store, err = storage.NewStorage(history)
	e(err)
	defer store.Close()

	// Assign global config and global control channels.
	config, pauseQueryChans = inputConfig, inputPauseQueryChans
	defer close(pauseDisplayChan)
	for _, pauseQueryChan := range pauseQueryChans {
		defer close(pauseQueryChan)
	}

	// Iniitialize reader indexes.
	readerIndexes = make(map[string]*storage.ReaderIndex, len(queries))
	for _, query := range queries {
		readerIndexes[query] = store.NewReaderIndex(query)
	}

	for {
		// Assign current context and restore default values.
		currentCtx = ctx
		resetContext(query)

		// Set up labelling or any schema for the results store.
		store.PutLabels(query, labels)

		switch displayMode {
		case DISPLAY_MODE_RAW:
			driver = DISPLAY_RAW
			RawDisplay(query)
		case DISPLAY_MODE_STREAM:
			driver = DISPLAY_TVIEW
			StreamDisplay(query)
		case DISPLAY_MODE_TABLE:
			driver = DISPLAY_TVIEW
			TableDisplay(query, filters)
		case DISPLAY_MODE_GRAPH:
			driver = DISPLAY_TERMDASH
			GraphDisplay(query, filters)
		default:
			slog.Error(fmt.Sprintf("Invalid result driver: %d\n", displayMode))
			os.Exit(1)
		}

		// If we get here, it's because the display functions have returned, probably because of an
		// interrupt. Assuming we haven't reached some other terminal situation, restart the results
		// display, adjusting for context.
		if currentCtx.Value("quit").(bool) {
			// Guess I'll die.
			displayQuit()
			break
		}
		if currentCtx.Value("advanceDisplayMode").(bool) {
			// Adjust the display mode.
			displayMode = GetNextSliceRing(activeDisplayModes, displayMode)
		}
		if currentCtx.Value("advanceQuery").(bool) {
			// Adjust the query.
			query = GetNextSliceRing(queries, query)
		}
	}
}
