//
// Entrypoint for cryptarch execution.

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/spacez320/cryptarch"
	"github.com/spacez320/cryptarch/internal/lib"
)

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

// Misc. constants.
const (
	CONFIG_FILE_DIR  = "cryptarch"      // Directory for Cryptarch configuration.
	CONFIG_FILE_NAME = "cryptarch.yaml" // Cryptarch configuration file.
)

var (
	count               int        // Number of attempts to execute the query.
	delay               int        // Delay between queries.
	displayMode         int        // Result mode to display.
	filters             string     // Result filters.
	history             bool       // Whether or not to preserve or use historical results.
	labels              string     // Result value labels.
	logFile             string     // Log filte to write to.
	logLevel            string     // Log level.
	mode                int        // Mode to execute in.
	outerPaddingBottom  int        // Bottom padding settings.
	outerPaddingLeft    int        // Left padding settings.
	outerPaddingRight   int        // Right padding settings.
	outerPaddingTop     int        // Top padding settings.
	port                string     // Port for RPC.
	promExporterAddr    string     // Address for Prometheus metrics page.
	promPushgatewayAddr string     // Address for Prometheus Pushgateway.
	queries             queriesArg // Queries to execute.
	showHelp            bool       // Whether or not to show helpt
	showLogs            bool       // Whether or not to show logs.
	showStatus          bool       // Whether or not to show statuses.
	silent              bool       // Whether or not to be quiet.

	logger                 = log.Default() // Logging system.
	logLevelStrToSlogLevel = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"error": slog.LevelError,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
	} // Log levels acceptable as a flag.
)

// Parses a comma delimited  string, returning a slice of strings if any are found, or an empty
// slice if not.
func parseCommaDelimitedStrOrEmpty(s string) []string {
	if parsed := strings.Split(s, ","); parsed[0] == "" {
		return []string{}
	} else {
		return parsed
	}
}

func main() {
	// Define arguments.
	flag.BoolVar(&history, "history", true, "Whether or not to use or preserve history.")
	flag.BoolVar(&showHelp, "show-help", true, "Whether or not to show help displays.")
	flag.BoolVar(&showLogs, "show-logs", false, "Whether or not to show log displays.")
	flag.BoolVar(&showStatus, "show-status", true, "Whether or not to show status displays.")
	flag.BoolVar(&silent, "silent", false, "Don't output anything to a console.")
	flag.IntVar(&count, "count", 1, "Number of query executions. -1 for continuous.")
	flag.IntVar(&delay, "delay", 3, "Delay between queries (seconds).")
	flag.IntVar(&displayMode, "display", int(lib.DISPLAY_MODE_RAW), "Result mode to display.")
	flag.IntVar(&mode, "mode", int(cryptarch.MODE_QUERY), "Mode to execute in.")
	flag.IntVar(&outerPaddingBottom, "outer-padding-bottom", -1, "Bottom display padding.")
	flag.IntVar(&outerPaddingLeft, "outer-padding-left", -1, "Left display padding.")
	flag.IntVar(&outerPaddingRight, "outer-padding-right", -1, "Right display padding.")
	flag.IntVar(&outerPaddingTop, "outer-padding-top", -1, "Top display padding.")
	flag.StringVar(&filters, "filters", "", "Results filters.")
	flag.StringVar(&labels, "labels", "", "Labels to apply to query values, separated by commas.")
	flag.StringVar(&logFile, "log-file", "", "Log file to write to.")
	flag.StringVar(&logLevel, "log-level", "error", "Log level.")
	flag.StringVar(&port, "rpc-port", "12345", "Port for RPC.")
	flag.StringVar(&promExporterAddr, "prometheus-exporter", "",
		"Address to present Prometheus metrics.")
	flag.StringVar(&promPushgatewayAddr, "prometheus-pushgateway", "",
		"Address for Prometheus Pushgateway.")
	flag.Var(&queries, "query", "Query to execute. Can be supplied multiple times. When in query "+
		"mode, this is expected to be some command. When in profile mode it is expected to be PID. "+
		"At least one query must be provided.")
	flag.Parse()

	// Check for required flags.
	if len(queries) == 0 {
		flag.Usage()
		fmt.Fprintf(os.Stderr, "Missing required argument -query\n")
		os.Exit(1)
	}

	// Set-up logging.
	if silent {
		// Silence all output.
		logger.SetOutput(io.Discard)
	} else if logFile != "" {
		// Write logs to a file.
		logF, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			panic(err)
		}
		defer logF.Close()
		slog.SetDefault(slog.New(
			slog.NewTextHandler(
				logF,
				&slog.HandlerOptions{Level: logLevelStrToSlogLevel[logLevel]},
			)))
	} else {
		// Set the default to be standard output--result modes may change this.
		slog.SetDefault(slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: logLevelStrToSlogLevel[logLevel]},
		)))
	}

	// Build general configuration.
	config := lib.Config{
		Count:                  count,
		Delay:                  delay,
		DisplayMode:            displayMode,
		Filters:                parseCommaDelimitedStrOrEmpty(filters),
		History:                history,
		Labels:                 parseCommaDelimitedStrOrEmpty(labels),
		LogLevel:               logLevel,
		Mode:                   mode,
		LogMulti:               logFile != "",
		Port:                   port,
		PrometheusExporterAddr: promExporterAddr,
		PushgatewayAddr:        promPushgatewayAddr,
		Queries:                queries,
	}

	// Build display configuration.
	displayConfig := lib.NewDisplayConfig()
	displayConfig.ShowHelp = showHelp
	displayConfig.ShowLogs = showLogs
	displayConfig.ShowStatus = showStatus
	if outerPaddingBottom >= 0 {
		displayConfig.OuterPaddingBottom = outerPaddingBottom
	}
	if outerPaddingLeft >= 0 {
		displayConfig.OuterPaddingLeft = outerPaddingLeft
	}
	if outerPaddingRight >= 0 {
		displayConfig.OuterPaddingRight = outerPaddingRight
	}
	if outerPaddingTop >= 0 {
		displayConfig.OuterPaddingTop = outerPaddingTop
	}

	cryptarch.Run(config, *displayConfig)
}
