//
// Shared configuration settings.

package lib

import "log/slog"

var (
	logLevelStrtoSlogLevel = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	} // Mapping of human-readable log levels to Slog levels.
)

// Shareable configuration. See CLI flags for further details.
type Config struct {
	Count, Delay, DisplayMode, Mode       int
	ElasticsearchAddr                     string
	Expressions, Filters, Labels, Queries []string
	History, LogMulti, Silent             bool
	LogLevel                              string
	Port                                  string
	PrometheusExporterAddr                string
	PushgatewayAddr                       string
}

// Retrieves an Slog level from a human-readable level string.
func (c *Config) SlogLogLevel() slog.Level {
	return logLevelStrtoSlogLevel[(*c).LogLevel]
}
