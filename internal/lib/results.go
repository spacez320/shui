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
	"unicode"

	"pkg/storage"

	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

var (
	config     Config          // Global configuration.
	currentCtx context.Context // Current context.
	driver     DisplayDriver   // Display driver, dictated by the results.

	ctxDefaults = map[string]interface{}{
		"advanceDisplayMode": false,
		"advanceQuery":       false,
		"quit":               false,
	} // Defaults applied to context.
	store = storage.NewStorage() // Stored results.
)

// Resets the current context to its default values.
func resetContext() {
	for k, v := range ctxDefaults {
		currentCtx = context.WithValue(currentCtx, k, v)
	}
}

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
func Results(
	ctx context.Context,
	displayMode DisplayMode,
	query string,
	labels, filters []string,
	inputConfig Config,
) {
	// Assign global config.
	config = inputConfig

	for {
		// Assign current context and restore default values.
		currentCtx = ctx
		resetContext()

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

		// If we get here, it's because the display functions have returned, probably
		// because of an interrupt. Assuming we haven't reached some other terminal
		// situation, restart the results display, adjusting for context.

		if currentCtx.Value("quit").(bool) {
			// Guess I'll die.
			os.Exit(0)
		}
		if currentCtx.Value("advanceDisplayMode").(bool) {
			// Adjust the display mode.
			displayMode = GetNextSliceRing(activeDisplayModes, displayMode)
		}
		if currentCtx.Value("advanceQuery").(bool) {
			// Adjust the query.
			query = GetNextSliceRing(ctx.Value("queries").([]string), query)
		}
	}
}
