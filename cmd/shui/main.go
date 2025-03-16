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
	"github.com/spf13/pflag"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/exp/maps"
)

// Used to supply build information.
type buildInfo struct {
	buildDate string
	gitCommit string
	goVersion string
	version   string
}

type QueryModeArg struct {
	queryMode shui.QueryMode
}

func (q *QueryModeArg) String() string {
	return q.queryMode.String()
}

func (q *QueryModeArg) Set(v string) error {
	qVal, err := shui.QueryModeFromString(v)
	q.queryMode = qVal
	return err
}

func (q QueryModeArg) Type() string {
	return "mode"
}

const (
	DEFAULT_CONFIG_FILE_DIR  = "shui"      // Directory for Shui configuration.
	DEFAULT_CONFIG_FILE_NAME = "shui.toml" // Shui configuration file.
)

var (
	// Whether or not to read from standard input.
	readStdin bool

	// Supplied by the linker at build time.
	date, commit, version string

	// Aliases to apply for configuration settings, mainly to account for differences between flags
	// (the left column) and configuration files (the right column).
	configurationAliases = map[string]string{
		"elasticsearch.addr":     "elasticsearch-addr",
		"elasticsearch.index":    "elasticsearch-index",
		"elasticsearch.password": "elasticsearch-password",
		"elasticsearch.user":     "elasticsearch-user",
		"tui.padding.bottom":     "outer-padding-bottom",
		"tui.padding.left":       "outer-padding-left",
		"tui.padding.right":      "outer-padding-right",
		"tui.padding.top":        "outer-padding-top",
		"prometheus.exporter":    "prometheus-exporter",
		"prometheus.pushgateway": "prometheus-pushgateway",
		"tui.show.help":          "show-help",
		"tui.show.logs":          "show-logs",
		"tui.show.status":        "show-status",
	}

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
		err           error        // General error holder.
		expressions   []string     // Expressions to apply to query results.
		mode          QueryModeArg // Mode to execute under.
		queries       []string     // Queries to execute.
		userConfigDir string       // User configuration directory.
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
	viper.SetDefault("mode", "query")
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

	// Define arguments.
	flag.Bool("help", false, "Show usage")
	flag.Bool("history", viper.GetBool("history"), "Whether or not to use or preserve history.")
	flag.Bool("show-help", viper.GetBool("show-help"), "Whether or not to show help displays.")
	flag.Bool("show-logs", viper.GetBool("show-logs"), "Whether or not to show log displays.")
	flag.Bool("show-status", viper.GetBool("show-status"), "Whether or not to show status displays.")
	flag.Bool("silent", viper.GetBool("silent"), "Don't output anything to a console.")
	flag.Bool("version", viper.GetBool("version"), "Show version.")
	flag.Int("count", viper.GetInt("count"), "Number of query executions. -1 for continuous.")
	flag.Int("delay", viper.GetInt("delay"), "Delay between queries (seconds).")
	flag.Int("display", viper.GetInt("display"), "Result mode to display.")
	flag.Int("outer-padding-bottom", viper.GetInt("outer-padding-bottom"), "Bottom display padding.")
	flag.Int("outer-padding-left", viper.GetInt("outer-padding-left"), "Left display padding.")
	flag.Int("outer-padding-right", viper.GetInt("outer-padding-right"), "Right display padding.")
	flag.Int("outer-padding-top", viper.GetInt("outer-padding-top"), "Top display padding.")
	flag.Int("rpc-port", viper.GetInt("rpc-port"), "Port for RPC.")
	flag.String(
		"config",
		filepath.Join(userConfigDir, DEFAULT_CONFIG_FILE_DIR, DEFAULT_CONFIG_FILE_NAME),
		"Config file to use")
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
	flag.Var(&mode, "mode", fmt.Sprintf("Mode to execute in (%s).", maps.Values(shui.Modes)))
	flag.Parse()

	// Define configuration sources.
	viper.BindPFlags(flag.CommandLine)
	viper.SetConfigFile(viper.GetString("config"))
	err = viper.ReadInConfig()
	if err != nil {
		// Exclude errors that indicate a missing configuration file.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// FIXME There is currently a bug preventing Viper from ever returning
			// `ConfigFileNotFoundError`. For now, skip over any configuration file errors.
			//
			// See: https://github.com/spf13/viper/issues/1783
			// panic(err)
			slog.Warn(err.Error())
		}
	}

	// Manage configuration aliases.
	for k, v := range configurationAliases {
		// FIXME There is currently a bug preventing Viper from doing overrides correctly with aliases.
		// Therefore, we circumvent the alias behavior by comparing the existence of flags to
		// configuration file entries, letting the former override the latter and setting them to the
		// same thing, making the alias registration a little pointless, but safe.
		//
		// See: https://github.com/spf13/viper/issues/689
		if viper.Get(k) != nil {
			// Equalize the flag and config values.
			viper.Set(v, k)
		}
		viper.RegisterAlias(v, k)
	}

	// Display usage.
	if viper.GetBool("help") {
		pflag.Usage()
		os.Exit(0)
	}

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

	// Determine queries to run. In order of preference, queries may come from stdin, flags, or
	// configuration files, but may not combine from multiple sources.
	f, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}
	if f.Mode()&os.ModeNamedPipe != 0 {
		// Queries are coming from standard input.
		readStdin = true
	} else if pflag.Lookup("query").Changed {
		// Queries were provided as flags.
		queries = viper.GetStringSlice("query")

		// Warn if providing queries are also present in the configuration file.
		if viper.InConfig("query") {
			slog.Warn("Queries are defined in both flags and configuration--using flags only")
		}
	} else if viper.InConfig("query") {
		// Queries are provided in the configuration file.
		for _, query := range viper.Get("query").([]interface{}) {
			queries = append(queries, query.(map[string]interface{})["command"].(string))
		}
	} else {
		// No queries were provided.
		flag.Usage()
		fmt.Fprintf(os.Stderr, "At least one query must be defined\n")
		os.Exit(1)
	}

	// Determine expressions to use. In order of preference, expressions may come from flags or
	// configuration files, but may not combine from multiple sources. This is mostly to mirror what
	// queries are doing.
	if pflag.Lookup("expr").Changed {
		expressions = viper.GetStringSlice("expr")
	} else if viper.InConfig("expressions") {
		expressions = viper.GetStringSlice("expressions")
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
		Expressions:            expressions,
		Filters:                viper.GetStringSlice("filters"),
		History:                viper.GetBool("history"),
		Labels:                 viper.GetStringSlice("labels"),
		LogLevel:               viper.GetString("log-level"),
		LogMulti:               viper.GetString("log-file") != "",
		Mode:                   int(mode.queryMode),
		Port:                   viper.GetInt("port"),
		PrometheusExporterAddr: viper.GetString("prometheus-exporter"),
		PushgatewayAddr:        viper.GetString("prometheus-pushgateway"),
		Queries:                queries,
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
