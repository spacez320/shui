//
// Controls shared configuration settings.

package lib

import "golang.org/x/exp/slog"

var (
	logLevelStrtoSlogLevel = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	} // Mapping of human-readable log levels to Slog levels.
)

// Shareable configuration.
type Config struct {
	LogLevel string
}

// Retrieves an Slog level from a human-readable level string.
func (c *Config) SlogLogLevel() slog.Level {
	return logLevelStrtoSlogLevel[(*c).LogLevel]
}
