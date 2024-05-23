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
	"log/slog"
	"os"
	"strconv"
	"strings"
	"text/scanner"
	"time"
	"unicode"

	"golang.org/x/exp/slices"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/spacez320/cryptarch/pkg/storage"
)

var (
	config          Config                          // Global configuration.
	currentCtx      context.Context                 // Current context.
	driver          DisplayDriver                   // Display driver, dictated by the results.
	pauseQueryChans map[string]chan bool            // Channels for dealing with 'pause' events for results.
	readerIndexes   map[string]*storage.ReaderIndex // Collection of reader index ids per query.
	store           storage.Storage                 // Stored results.

	ctxDefaults = map[string]interface{}{
		"advanceDisplayMode": false,
		"advanceQuery":       false,
		"quit":               false,
	} // Defaults applied to context.
	pauseDisplayChan = make(chan bool) // Channel for dealing with 'pause' events for the display.
)

// Executes an expression on a result and returns a new result.
//
// TODO Currently this compiles expressions on the fly and constructs a result object on every
// iteration. In the future, it might be important from a performance perspective to pre-compile
// expressions and avoid constructing results before it's necessary.
func exprResult(
	query, expression string,
	result, prevResult storage.Result,
) (newResult storage.Result, err error) {
	var (
		env     map[string]interface{} // Environment to provide for an expression.
		output  interface{}            // Output from an expression.
		program *vm.Program            // Expression executable.
	)

	// Construct the expression environment.
	env = map[string]interface{}{
		"prevResult": prevResult.Map(store.GetLabels(query, []string{})),
		"result":     result.Map(store.GetLabels(query, []string{})),
	}
	slog.Debug("Expression executing", "query", query, "expression", expression, "env", env)

	// Execute any expression.
	program, err = expr.Compile(expression, expr.Env(env))
	if err != nil {
		slog.Error("Failed to compile expression", "expr", expression, "env", env)
		return result, err
	}
	output, err = expr.Run(program, env)
	if err != nil {
		slog.Error("Expression failed to execute", "expr", expression, "env", env)
		return result, err
	}

	// Re-define result based on the expression output.
	switch output.(type) {
	case bool:
		newResult = storage.Result{
			Time:   result.Time,
			Value:  strconv.FormatBool(output.(bool)),
			Values: storage.Values{strconv.FormatBool(output.(bool))},
		}
	case int:
		newResult = storage.Result{
			Time:   result.Time,
			Value:  strconv.Itoa(output.(int)),
			Values: storage.Values{strconv.Itoa(output.(int))},
		}
	case int64:
		newResult = storage.Result{
			Time:   result.Time,
			Value:  strconv.FormatInt(output.(int64), 10),
			Values: storage.Values{strconv.FormatInt(output.(int64), 10)},
		}
	case float64:
		newResult = storage.Result{
			Time:   result.Time,
			Value:  strconv.FormatFloat(output.(float64), 'f', -1, 64),
			Values: storage.Values{strconv.FormatFloat(output.(float64), 'f', -1, 64)},
		}
	default:
		newResult = storage.Result{
			Time:   result.Time,
			Value:  output.(string),
			Values: storage.Values{output.(string)},
		}
	}

	return
}

// Resets the current context to its default values.
func resetContext(query string) {
	for k, v := range ctxDefaults {
		currentCtx = context.WithValue(currentCtx, k, v)
	}
	currentCtx = context.WithValue(currentCtx, "query", query)
}

// Adds a result to the result store based on a string. It is assumed that all processing has
// ocurred on the result itself.
func AddResult(query, result string, history bool) {
	result = strings.TrimSpace(result)
	_, err := store.Put(query, result, history, TokenizeResult(result)...)
	e(err)
}

// Get results previous to the last read result.
func GetPrevResults(query string, filters []string) (results []storage.Result) {
	slog.Debug("Fetching previous results", "query", query)

	// Retrieve previous results.
	return store.GetToIndex(query, filters, readerIndexes[query])
}

// Retrieves a next result.
func GetResult(query string, filters []string) (result storage.Result) {
	slog.Debug("Fetching next result", "query", query)

	return store.Next(query, filters, readerIndexes[query])
}

// Returns a result after applying expressions. Requires a previous result for calculations
// requiring history. It is expected that a query can tolerate the potential emptyness of
// prevResult, namely on the first execution.
func ExprResult(
	query string,
	expressions []string,
	result, prevResult storage.Result,
) storage.Result {
	var err error // General error holder.

	// Process any expressions on the result.
	for _, expression := range expressions {
		result, err = exprResult(query, expression, result, prevResult)
		if err != nil {
			e(err)
		}
	}

	return result
}

// Retrieves a next result, waiting for a non-empty return in a non-blocking manner.
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
func FilterResult(result storage.Result, filters, labels []string) storage.Result {
	var (
		labelIndexes = make([]int, len(filters))         // Indexes of labels from filters, corresponding to result values.
		resultValues = make([]interface{}, len(filters)) // Found result values.
	)

	slog.Debug("Filtering result", "result", result, "filters", filters, "labels", labels)

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
	displayConfig *DisplayConfig,
	inputConfig *Config,
	inputPauseQueryChans map[string]chan bool,
	resultsReadyChan chan bool,
) {
	var (
		err         error                      // General error holder.
		pushgateway storage.PushgatewayStorage // Pushgateway configuration.
		prometheus  storage.PrometheusStorage  // Prometheus configuration.

		expressions = ctx.Value("expressions").([]string) // Capture expressions from context.
		filters     = ctx.Value("filters").([]string)     // Capture filters from context.
		labels      = ctx.Value("labels").([]string)      // Capture labels from context.
		queries     = ctx.Value("queries").([]string)     // Capture queries from context.
	)

	// Assign global config and global control channels.
	config, pauseQueryChans = *inputConfig, inputPauseQueryChans
	defer close(pauseDisplayChan)
	for _, pauseQueryChan := range pauseQueryChans {
		defer close(pauseQueryChan)
	}

	// Initialize storage.
	store, err = storage.NewStorage(history)
	e(err)
	defer store.Close()

	// Initialize external storage.
	if config.PushgatewayAddr != "" {
		pushgateway = storage.NewPushgatewayStorage(config.PushgatewayAddr)
		store.AddExternalStorage(&pushgateway)
	}
	if config.PrometheusExporterAddr != "" {
		prometheus = storage.NewPrometheusStorage(config.PrometheusExporterAddr)
		store.AddExternalStorage(&prometheus)
	}

	// Initialize reader indexes.
	readerIndexes = make(map[string]*storage.ReaderIndex, len(queries))
	for _, query := range queries {
		readerIndexes[query] = store.NewReaderIndex(query)
	}

	// Signals that results are ready to be received.
	slog.Debug("Results are ready")
	resultsReadyChan <- true

	for {
		// Assign current context and restore default values.
		currentCtx = ctx
		resetContext(query)

		// Set up labelling or any schema for the results store, if any were explicitly provided.
		if len(labels) > 0 {
			store.PutLabels(query, labels)
		}

		switch displayMode {
		case DISPLAY_MODE_RAW:
			driver = DISPLAY_RAW
			RawDisplay(query, filters, expressions)
		case DISPLAY_MODE_STREAM:
			driver = DISPLAY_TVIEW
			StreamDisplay(query, filters, expressions, displayConfig)
		case DISPLAY_MODE_TABLE:
			driver = DISPLAY_TVIEW
			TableDisplay(query, filters, expressions, displayConfig)
		case DISPLAY_MODE_GRAPH:
			if len(filters) == 0 {
				slog.Error("Graph mode requires a filter")
				os.Exit(1)
			}
			if len(filters) > 1 {
				slog.Warn("Graph mode can only apply one filter; ignoring all but the first")
			}
			driver = DISPLAY_TERMDASH
			GraphDisplay(query, filters[0], expressions, displayConfig)
		default:
			slog.Error("Invalid result driver", "displayMode", displayMode)
			os.Exit(1)
		}

		// If we get here, it's because the display functions have returned, probably because of an
		// interrupt. Assuming we haven't reached some other terminal situation, restart the results
		// display, adjusting for context.
		if currentCtx.Value("quit").(bool) {
			// Guess I'll die.
			displayQuit()
			os.Exit(0)
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
