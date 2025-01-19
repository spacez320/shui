//
// Entrypoint for shui execution.

package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spacez320/shui"
	"github.com/spacez320/shui/internal/lib"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Used to supply build information.
type buildInfo struct {
	buildDate string
	gitCommit string
	goVersion string
	version   string
}

// Misc. constants.
const (
	CONFIG_FILE_DIR  = "shui"      // Directory for Shui configuration.
	CONFIG_FILE_NAME = "shui.toml" // Shui configuration file.
)

var (
	// Whether or not to read from standard input.
	readStdin bool

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

func main() {
	var (
		err           error  // General error holder.
		userConfigDir string // User configuration directory.
	)

	// Retrieve the user config directory.
	userConfigDir, err = os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	// Define configuration defaults.
	viper.SetDefault("count", 1)
	viper.SetDefault("delay", 3)
	viper.SetDefault("display", int(lib.DISPLAY_MODE_RAW))
	viper.SetDefault("elasticsearch-addr", "")
	viper.SetDefault("elasticsearch-index", "")
	viper.SetDefault("elasticsearch-password", "")
	viper.SetDefault("elasticsearch-user", "")
	viper.SetDefault("expr", []string{})
	viper.SetDefault("filters", []string{})
	viper.SetDefault("history", true)
	viper.SetDefault("labels", []string{})
	viper.SetDefault("log-file", "")
	viper.SetDefault("log-level", "error")
	viper.SetDefault("mode", int(shui.MODE_QUERY))
	viper.SetDefault("outer-padding-bottom", -1)
	viper.SetDefault("outer-padding-left", -1)
	viper.SetDefault("outer-padding-right", -1)
	viper.SetDefault("outer-padding-top", -1)
	viper.SetDefault("prometheus-exporter", "")
	viper.SetDefault("prometheus-pushgateway", "")
	viper.SetDefault("query", []string{})
	viper.SetDefault("rpc-port", 12345)
	viper.SetDefault("show-help", true)
	viper.SetDefault("show-logs", false)
	viper.SetDefault("show-status", true)
	viper.SetDefault("silent", false)
	viper.SetDefault("version", false)

	viper.SetConfigFile(filepath.Join(userConfigDir, CONFIG_FILE_DIR, CONFIG_FILE_NAME))
	err = viper.ReadInConfig()
	if err != nil {
		// Exclude errors that indicate a missing configuration file.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// FIXME There is currently a bug preventing Viper from ever returning
			// `ConfigFileNotFoundError`. For now, skip over any configuration file errors.
			//
			// See: https://github.com/spf13/viper/issues/1783
			// panic(err)
		}
	}

	// Define arguments.
	flag.Bool("history", viper.GetBool("history"), "Whether or not to use or preserve history.")
	flag.Bool("show-help", viper.GetBool("show-help"), "Whether or not to show help displays.")
	flag.Bool("show-logs", viper.GetBool("show-logs"), "Whether or not to show log displays.")
	flag.Bool("show-status", viper.GetBool("show-status"), "Whether or not to show status displays.")
	flag.Bool("silent", viper.GetBool("silent"), "Don't output anything to a console.")
	flag.Bool("version", viper.GetBool("version"), "Show version.")
	flag.Int("count", viper.GetInt("count"), "Number of query executions. -1 for continuous.")
	flag.Int("delay", viper.GetInt("delay"), "Delay between queries (seconds).")
	flag.Int("display", viper.GetInt("display"), "Result mode to display.")
	flag.Int("mode", viper.GetInt("mode"), "Mode to execute in.")
	flag.Int("outer-padding-bottom", viper.GetInt("outer-padding-bottom"), "Bottom display padding.")
	flag.Int("outer-padding-left", viper.GetInt("outer-padding-left"), "Left display padding.")
	flag.Int("outer-padding-right", viper.GetInt("outer-padding-right"), "Right display padding.")
	flag.Int("outer-padding-top", viper.GetInt("outer-padding-top"), "Top display padding.")
	flag.Int("rpc-port", viper.GetInt("rpc-port"), "Port for RPC.")
	flag.String("elasticsearch-addr", viper.GetString("elasticsearch-addr"),
		"Address to present Elasticsearch document updates.")
	flag.String("elasticsearch-index", viper.GetString("elasticsearch-index"),
		"Index to use for Elasticsearch document updates. It is expected that the index already "+
			"exists or will automatically be created.")
	flag.String("elasticsearch-password", viper.GetString("elasticsearch-password"),
		"Password to use for Elasticsearch basic auth.")
	flag.String("elasticsearch-user", viper.GetString("elasticsearch-user"),
		"User to use for Elasticsearch basic auth.")
	flag.String("log-file", viper.GetString("log-file"), "Log file to write to.")
	flag.String("log-level", viper.GetString("log-level"), "Log level.")
	flag.String("prometheus-exporter", viper.GetString("prometheus-exporter"),
		"Address to present Prometheus metrics.")
	flag.String("prometheus-pushgateway", viper.GetString("prometheus-pushgateway"),
		"Address for Prometheus Pushgateway.")
	flag.StringArray("expr", viper.GetStringSlice("expr"),
		"Expression to apply to output. Can be supplied multiple times.")
	flag.StringArray("query", viper.GetStringSlice("query"), "Query to execute. Can be supplied "+
		"multiple times. When in query mode, this is expected to be some command. When in profile "+
		"mode it is expected to be PID. At least one query must be provided.")
	flag.StringSlice("filters", viper.GetStringSlice("filters"), "Results filters.")
	flag.StringSlice("labels", viper.GetStringSlice("labels"),
		"Labels to apply to query values, separated by commas.")
	flag.Parse()
	viper.BindPFlags(flag.CommandLine)

	// Display a version.
	if viper.GetBool("version") {
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
		if len(viper.GetStringSlice("query")) == 0 {
			flag.Usage()
			fmt.Fprintf(os.Stderr, "Missing required argument --query\n")
			os.Exit(1)
		}
	}

	// Set-up logging.
	if viper.GetBool("silent") {
		// Silence all output.
		logger.SetOutput(io.Discard)
	} else if viper.GetString("log-file") != "" {
		// Write logs to a file.
		logF, err := os.OpenFile(viper.GetString("log-file"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			panic(err)
		}
		defer logF.Close()
		slog.SetDefault(slog.New(
			slog.NewTextHandler(
				logF,
				&slog.HandlerOptions{Level: logLevelStrToSlogLevel[viper.GetString("log-level")]},
			)))
	} else {
		// Set the default to be standard output--result modes may change this.
		slog.SetDefault(slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: logLevelStrToSlogLevel[viper.GetString("log-level")]},
		)))
	}

	// Build general configuration.
	config := lib.Config{
		Count:                  viper.GetInt("count"),
		Delay:                  viper.GetInt("delay"),
		DisplayMode:            viper.GetInt("display"),
		ElasticsearchAddr:      viper.GetString("elasticsearch-addr"),
		ElasticsearchIndex:     viper.GetString("elasticsearch-index"),
		ElasticsearchPassword:  viper.GetString("elasticsearch-password"),
		ElasticsearchUser:      viper.GetString("elasticsearch-user"),
		Expressions:            viper.GetStringSlice("expr"),
		Filters:                viper.GetStringSlice("filters"),
		History:                viper.GetBool("history"),
		Labels:                 viper.GetStringSlice("labels"),
		LogLevel:               viper.GetString("log-level"),
		LogMulti:               viper.GetString("log-file") != "",
		Mode:                   viper.GetInt("mode"),
		Port:                   viper.GetInt("port"),
		PrometheusExporterAddr: viper.GetString("prometheus-exporter"),
		PushgatewayAddr:        viper.GetString("prometheus-pushgateway"),
		Queries:                viper.GetStringSlice("query"),
		ReadStdin:              readStdin,
	}

	// Build display configuration.
	displayConfig := lib.NewDisplayConfig()
	displayConfig.ShowHelp = viper.GetBool("show-help")
	displayConfig.ShowLogs = viper.GetBool("show-logs")
	displayConfig.ShowStatus = viper.GetBool("show-status")
	if viper.GetInt("outer-padding-bottom") >= 0 {
		displayConfig.OuterPaddingBottom = viper.GetInt("outer-padding-bottom")
	}
	if viper.GetInt("outer-padding-left") >= 0 {
		displayConfig.OuterPaddingLeft = viper.GetInt("outer-padding-left")
	}
	if viper.GetInt("outer-padding-right") >= 0 {
		displayConfig.OuterPaddingRight = viper.GetInt("outer-padding-right")
	}
	if viper.GetInt("outer-padding-top") >= 0 {
		displayConfig.OuterPaddingTop = viper.GetInt("outer-padding-top")
	}

	shui.Run(config, *displayConfig)
}
