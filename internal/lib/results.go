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
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"

	"pkg/storage"

	"github.com/rivo/tview"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Types
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Represents the display mode.
type DisplayMode int

// Represents the result mode value.
type ResultMode int

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Variables
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Display constants. Each result mode uses a specific display.
const (
	DISPLAY_RAW      DisplayMode = iota + 1 // Used for direct output.
	DISPLAY_TVIEW                           // Used when tview is the TUI driver.
	DISPLAY_TERMDASH                        // Used when termdash is the TUI driver.

)

// Result mode constants.
const (
	RESULT_MODE_RAW    ResultMode = iota + 1 // For running in 'raw' result mode.
	RESULT_MODE_STREAM                       // For running in 'stream' result mode.
	RESULT_MODE_TABLE                        // For running in 'table' result mode.
	RESULT_MODE_GRAPH                        // For running in 'graph' result mode.
)

var (
	// Global configuration.
	config Config
	// Display mode, dictated by the results.
	mode DisplayMode

	// Application for display. Only applicable for tview result modes.
	app = tview.NewApplication()
	// Stored results.
	store = storage.NewStorage()
)

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Public
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Adds a result to the result store.
func AddResult(query, result string) {
	result = strings.TrimSpace(result)
	store.Put(query, result, TokenizeResult(result)...)
}

// Retrieves a next result.
func GetResult(query string) storage.Result {
	return <-storage.PutEvents[query]
}

// Creates a result with filtered values.
func FilterResult(result storage.Result, labels, filters []string) storage.Result {
	var (
		// Indexes of labels from filters, corresponding to result values.
		labelIndexes = make([]int, len(filters))
		// Found result values.
		resultValues = make([]interface{}, len(filters))
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
func Results(resultMode ResultMode, query string, labels, filters []string, inputConfig Config) {
	// Assign global config.
	config = inputConfig
	// Set up labelling or any schema for the results store.
	store.PutLabels(query, labels)

	switch resultMode {
	case RESULT_MODE_RAW:
		mode = DISPLAY_RAW
		RawDisplay(query)
	case RESULT_MODE_STREAM:
		mode = DISPLAY_TVIEW
		StreamDisplay(query)
	case RESULT_MODE_TABLE:
		mode = DISPLAY_TVIEW
		TableDisplay(query, filters)
	case RESULT_MODE_GRAPH:
		mode = DISPLAY_TERMDASH
		GraphDisplay(query, filters)
	default:
		slog.Error(fmt.Sprintf("Invalid result mode: %d\n", resultMode))
		os.Exit(1)
	}
}
