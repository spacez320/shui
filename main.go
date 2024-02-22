//
// Entrypoint for cryptarch execution.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"internal/lib"

	"golang.org/x/exp/slog"
)

// Represents the mode value.
type queryMode int

// Queries provided as flags.
type queriesArg []string

func (q *queriesArg) String() string {
	// XXX This is necessary to resolve the interface contract, but doesn't seem important.
	return ""
}

func (q *queriesArg) Set(query string) error {
	*q = append(*q, query)
	return nil
}

// Converts to a string slice.
func (q *queriesArg) ToStrings() (q_strings []string) {
	for _, v := range *q {
		q_strings = append(q_strings, v)
	}
	return
}

// Mode constants.
const (
	MODE_QUERY   queryMode = iota + 1 // For running in 'query' mode.
	MODE_PROFILE                      // For running in 'profile' mode.
	MODE_READ                         // For running in 'read' mode.
)

var (
	count               int        // Number of attempts to execute the query.
	delay               int        // Delay between queries.
	displayMode         int        // Result mode to display.
	filters             string     // Result filters.
	history             bool       // Whether or not to preserve or use historical results.
	logLevel            string     // Log level.
	mode                int        // Mode to execute in.
	port                string     // Port for RPC.
	promExporterAddr    string     // Address for Prometheus metrics page.
	promPushgatewayAddr string     // Address for Prometheus Pushgateway.
	queries             queriesArg // Queries to execute.
	silent              bool       // Whether or not to be quiet.
	labels              string     // Result value labels.

	ctx                    = context.Background() // Initialize context.
	logger                 = log.Default()        // Logging system.
	logLevelStrToSlogLevel = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"error": slog.LevelError,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
	} // Log levels acceptable as a flag.
)

// Parses a comma delimited argument string, returning a slice of strings if any are found, or an
// empty slice if not.
func parseCommaDelimitedArg(arg string) []string {
	if parsed := strings.Split(arg, ","); parsed[0] == "" {
		return []string{}
	} else {
		return parsed
	}
}

func main() {
	var (
		doneQueriesChan chan bool            // Channel for tracking query completion.
		pauseQueryChans map[string]chan bool // Channels for pausing queries.

		resultsReadyChan = make(chan bool) // Channel for signaling results readiness.
	)

	// Define arguments.
	flag.BoolVar(&history, "history", true, "Whether or not to use or preserve history.")
	flag.BoolVar(&silent, "silent", false, "Don't output anything to a console.")
	flag.IntVar(&count, "count", 1, "Number of query executions. -1 for continuous.")
	flag.IntVar(&delay, "delay", 3, "Delay between queries (seconds).")
	flag.IntVar(&displayMode, "display", int(lib.DISPLAY_MODE_RAW), "Result mode to display.")
	flag.IntVar(&mode, "mode", int(MODE_QUERY), "Mode to execute in.")
	flag.StringVar(&filters, "filters", "", "Results filters.")
	flag.StringVar(&labels, "labels", "", "Labels to apply to query values, separated by commas.")
	flag.StringVar(&logLevel, "log-level", "error", "Log level.")
	flag.StringVar(&port, "rpc-port", "12345", "Port for RPC.")
	flag.StringVar(&promExporterAddr, "prometheus-exporter", "127.0.0.1:8080",
		"Address to present Prometheus metrics.")
	flag.StringVar(&promPushgatewayAddr, "prometheus-pushgateway", "127.0.0.1:9091",
		"Address for Prometheus Pushgateway.")
	flag.Var(&queries, "query", "Query to execute. Can be supplied multiple times. When in query"+
		"mode, this is expected to be some command. When in profile mode it is expected to be PID.")
	flag.Parse()

	// Set-up logging.
	if silent || displayMode == int(lib.DISPLAY_MODE_GRAPH) {
		// Silence all output.
		logger.SetOutput(io.Discard)
	} else {
		// Set the default to be standard error--result modes may change this.
		slog.SetDefault(slog.New(slog.NewTextHandler(
			os.Stderr,
			&slog.HandlerOptions{Level: logLevelStrToSlogLevel[logLevel]},
		)))
	}

	// Execute the specified mode.
	switch {
	case mode == int(MODE_PROFILE):
		slog.Debug("Executing in profile mode.")

		doneQueriesChan, pauseQueryChans = lib.Query(
			lib.QUERY_MODE_PROFILE,
			count,
			delay,
			queries,
			port,
			history,
			resultsReadyChan,
		)

		// Process mode has specific labels--ignore user provided ones.
		ctx = context.WithValue(ctx, "labels", lib.ProfileLabels)
	case mode == int(MODE_QUERY):
		slog.Debug("Executing in query mode.")

		doneQueriesChan, pauseQueryChans = lib.Query(
			lib.QUERY_MODE_COMMAND,
			count,
			delay,
			queries,
			port,
			history,
			resultsReadyChan,
		)

		// Rely on user-defined labels.
		ctx = context.WithValue(ctx, "labels", parseCommaDelimitedArg(labels))
	case mode == int(MODE_READ):
		slog.Debug("Executing in read mode.")

	// FIXME Temporarily disabling read mode.
	// 	done = lib.Read(port)
	default:
		slog.Error(fmt.Sprintf("Invalid mode: %d\n", mode))
		os.Exit(1)
	}

	// Initialize remaining context.
	ctx = context.WithValue(ctx, "filters", parseCommaDelimitedArg(filters))
	ctx = context.WithValue(ctx, "queries", queries.ToStrings())

	// Execute result viewing.
	if !silent {
		lib.Results(
			ctx,
			lib.DisplayMode(displayMode),
			ctx.Value("queries").([]string)[0], // Always start with the first query.
			history,
			lib.Config{
				LogLevel:        logLevel,
				PushgatewayAddr: promPushgatewayAddr,
			},
			pauseQueryChans,
			resultsReadyChan,
		)
	}

	// XXX This isn't strictly necessary, mainly because getting here shouldn't be possible
	// (`lib.Results` does not have any intentional return condition), but it's being left here in
	// case in the future we do want to control for query completion.
	<-doneQueriesChan
	close(doneQueriesChan)
}
