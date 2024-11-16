//
// Entrypoint for shui execution.

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/spacez320/shui"
	"github.com/spacez320/shui/internal/lib"
)

// Used to supply build information.
type buildInfo struct {
	buildDate string
	gitCommit string
	goVersion string
	version   string
}

// Queries provided as flags.
type multiArg []string

func (q *multiArg) String() string {
	// XXX This is necessary to resolve the interface contract, but doesn't seem important.
	return ""
}

func (q *multiArg) Set(query string) error {
	*q = append(*q, query)
	return nil
}

// Converts to a string slice.
func (q *multiArg) ToStrings() (q_strings []string) {
	for _, v := range *q {
		q_strings = append(q_strings, v)
	}
	return
}

// Misc. constants.
const (
	CONFIG_FILE_DIR  = "shui"      // Directory for Shui configuration.
	CONFIG_FILE_NAME = "shui.yaml" // Shui configuration file.
)

var (
	count                 int      // Number of attempts to execute the query.
	delay                 int      // Delay between queries.
	displayMode           int      // Result mode to display.
	elasticsearchAddr     string   // Address for Elasticsearch.
	elasticsearchIndex    string   // Index to use for Elasticsearch documents.
	elasticsearchPassword string   // Password for Elasticsearch basic auth.
	elasticsearchUser     string   // User for Elasticsearch basic auth.
	expressions           multiArg // Expression to apply to output.
	filters               string   // Result filters.
	history               bool     // Whether or not to preserve or use historical results.
	labels                string   // Result value labels.
	logFile               string   // Log filte to write to.
	logLevel              string   // Log level.
	mode                  int      // Mode to execute in.
	outerPaddingBottom    int      // Bottom padding settings.
	outerPaddingLeft      int      // Left padding settings.
	outerPaddingRight     int      // Right padding settings.
	outerPaddingTop       int      // Top padding settings.
	port                  string   // Port for RPC.
	promExporterAddr      string   // Address for Prometheus metrics page.
	promPushgatewayAddr   string   // Address for Prometheus Pushgateway.
	queries               multiArg // Queries to execute.
	readStdin             bool     // Whether input comes from standard input.
	showHelp              bool     // Whether or not to show helpt
	showLogs              bool     // Whether or not to show logs.
	showStatus            bool     // Whether or not to show statuses.
	showVersion           bool     // Whether or not to display a version.
	silent                bool     // Whether or not to be quiet.

	// Supplied by the linker at build time.
	version string
	commit  string
	date    string

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
	flag.BoolVar(&showVersion, "version", false, "Show version.")
	flag.BoolVar(&silent, "silent", false, "Don't output anything to a console.")
	flag.IntVar(&count, "count", 1, "Number of query executions. -1 for continuous.")
	flag.IntVar(&delay, "delay", 3, "Delay between queries (seconds).")
	flag.IntVar(&displayMode, "display", int(lib.DISPLAY_MODE_RAW), "Result mode to display.")
	flag.IntVar(&mode, "mode", int(shui.MODE_QUERY), "Mode to execute in.")
	flag.IntVar(&outerPaddingBottom, "outer-padding-bottom", -1, "Bottom display padding.")
	flag.IntVar(&outerPaddingLeft, "outer-padding-left", -1, "Left display padding.")
	flag.IntVar(&outerPaddingRight, "outer-padding-right", -1, "Right display padding.")
	flag.IntVar(&outerPaddingTop, "outer-padding-top", -1, "Top display padding.")
	flag.StringVar(&elasticsearchAddr, "elasticsearch-addr", "",
		"Address to present Elasticsearch document updates.")
	flag.StringVar(&elasticsearchIndex, "elasticsearch-index", "",
		"Index to use for Elasticsearch document updates. It is expected that the index already "+
			"exists or will automatically be created.")
	flag.StringVar(&elasticsearchPassword, "elasticsearch-password", "",
		"Password to use for Elasticsearch basic auth.")
	flag.StringVar(&elasticsearchUser, "elasticsearch-user", "",
		"User to use for Elasticsearch basic auth.")
	flag.StringVar(&filters, "filters", "", "Results filters.")
	flag.StringVar(&labels, "labels", "", "Labels to apply to query values, separated by commas.")
	flag.StringVar(&logFile, "log-file", "", "Log file to write to.")
	flag.StringVar(&logLevel, "log-level", "error", "Log level.")
	flag.StringVar(&port, "rpc-port", "12345", "Port for RPC.")
	flag.StringVar(&promExporterAddr, "prometheus-exporter", "",
		"Address to present Prometheus metrics.")
	flag.StringVar(&promPushgatewayAddr, "prometheus-pushgateway", "",
		"Address for Prometheus Pushgateway.")
	flag.Var(&expressions, "expr", "Expression to apply to output. Can be supplied multiple times.")
	flag.Var(&queries, "query", "Query to execute. Can be supplied multiple times. When in query "+
		"mode, this is expected to be some command. When in profile mode it is expected to be PID. "+
		"At least one query must be provided.")
	flag.Parse()

	// Display a version.
	if showVersion {
		fmt.Printf("shui %#v\n", buildInfo{
			buildDate: date,
			gitCommit: commit,
			goVersion: runtime.Version(),
			version:   version,
		})
		os.Exit(0)
	}

	// Detect if running from standard input.
	f, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}
	if f.Mode()&os.ModeNamedPipe != 0 {
		// We are reading standard input.
		readStdin = true
	} else {
		// There is no standard input--queries are needed.
		if len(queries) == 0 {
			flag.Usage()
			fmt.Fprintf(os.Stderr, "Missing required argument -query\n")
			os.Exit(1)
		}
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
		ElasticsearchAddr:      elasticsearchAddr,
		ElasticsearchIndex:     elasticsearchIndex,
		ElasticsearchPassword:  elasticsearchPassword,
		ElasticsearchUser:      elasticsearchUser,
		Expressions:            expressions,
		Filters:                parseCommaDelimitedStrOrEmpty(filters),
		History:                history,
		Labels:                 parseCommaDelimitedStrOrEmpty(labels),
		LogLevel:               logLevel,
		LogMulti:               logFile != "",
		Mode:                   mode,
		Port:                   port,
		PrometheusExporterAddr: promExporterAddr,
		PushgatewayAddr:        promPushgatewayAddr,
		Queries:                queries,
		ReadStdin:              readStdin,
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

	shui.Run(config, *displayConfig)
}
