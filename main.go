package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"internal/lib"

	"golang.org/x/exp/slog"
)

////////////////////////////////////////////////////////////////////////////////
//
// Types
//
////////////////////////////////////////////////////////////////////////////////

// Represents the mode value.
type queryMode int

// Queries provided as flags.
type queries_ []string

func (q *queries_) String() string {
	return fmt.Sprintf("%v", &q)
}

func (q *queries_) Set(query string) error {
	*q = append(*q, query)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
//
// Variables
//
////////////////////////////////////////////////////////////////////////////////

// Mode constants.
const (
	MODE_QUERY queryMode = iota + 1 // For running in 'query' mode.
	MODE_READ                       // For running in 'read' mode.
)

var (
	attempts    int      // Number of attempts to execute the query.
	delay       int      // Delay between queries.
	filters     string   // Result filters.
	logLevel    string   // Log level.
	mode        int      // Mode to execute in.
	port        string   // Port for RPC.
	queries     queries_ // Queries to execute.
	resultMode  int      // Result mode to display.
	silent      bool     // Whether or not to be quiet.
	valueLabels string   // Result value labels.

	logger                 = log.Default() // Logging system.
	logLevelStrToSlogLevel = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	} // Log levels acceptable as a flag.
)

////////////////////////////////////////////////////////////////////////////////
//
// Private
//
////////////////////////////////////////////////////////////////////////////////

// Parses a comma delimited argument string, returning a slice of strings if
// any are found, or an empty slice if not.
func parseCommaDelimitedArg(arg string) []string {
	if parsed := strings.Split(arg, ","); parsed[0] == "" {
		return []string{}
	} else {
		return parsed
	}
}

func main() {
	// Define arguments.

	flag.BoolVar(&silent, "s", false, "Don't output anything to a console.")
	flag.IntVar(&attempts, "t", 1, "Number of query executions. -1 for continuous.")
	flag.IntVar(&delay, "d", 3, "Delay between queries (seconds).")
	flag.StringVar(&filters, "f", "", "Results filters.")
	flag.IntVar(&mode, "m", int(MODE_QUERY), "Mode to execute in.")
	flag.StringVar(&logLevel, "l", "error", "Log level.")
	flag.IntVar(&resultMode, "r", int(lib.RESULT_MODE_RAW), "Result mode to display.")
	flag.StringVar(&port, "p", "12345", "Port for RPC.")
	flag.Var(&queries, "q", "Query to execute.")
	flag.StringVar(&valueLabels, "v", "", "Labels to apply to query values, separated by commas.")
	flag.Parse()

	// Set-up logging.

	if silent || resultMode == int(lib.RESULT_MODE_GRAPH) {
		// Silence all output.
		logger.SetOutput(ioutil.Discard)
	} else {
		// Set the default to be standard error--result modes may change this.
		slog.SetDefault(slog.New(slog.NewTextHandler(
			os.Stderr,
			&slog.HandlerOptions{Level: logLevelStrToSlogLevel[logLevel]},
		)))
	}

	// Execute the specified mode.

	var done chan int

	switch {
	case mode == int(MODE_QUERY):
		slog.Debug("Executing in query mode.")
		done = lib.Query(queries, attempts, delay, port)
	case mode == int(MODE_READ):
		slog.Debug("Executing in read mode.")
		done = lib.Read(port)
	default:
		slog.Error(fmt.Sprintf("Invalid mode: %d\n", mode))
		os.Exit(1)
	}

	// Execute result viewing.

	if !silent {
		lib.Results(
			lib.ResultMode(resultMode),
			queries[0], // TODO Until result modes support >1 query.
			parseCommaDelimitedArg(valueLabels),
			parseCommaDelimitedArg(filters),
			lib.Config{
				LogLevel: logLevel,
			},
		)
	}

	<-done
}
