//
// Launcher for Shui.

package shui

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spacez320/shui/internal/lib"
)

// Represents the mode value.
type queryMode int

// Mode constants.
const (
	MODE_QUERY   queryMode = iota + 1 // For running in 'query' mode.
	MODE_PROFILE                      // For running in 'profile' mode.
	MODE_READ                         // For running in 'read' mode.
)

const (
	STDIN_QUERY_NAME = "stdin" // Named query value for reading stdin.
)

var (
	ctx = context.Background() // Initialize context.
)

// Executes a Shui.
func Run(config lib.Config, displayConfig lib.DisplayConfig) {
	var (
		doneQueriesChan chan bool            // Channel for tracking query completion.
		pauseQueryChans map[string]chan bool // Channels for pausing queries.

		resultsReadyChan = make(chan bool) // Channel for signaling results readiness.
	)

	slog.Debug("Running with config", "config", config)
	slog.Debug("Running with display config", "displayConfig", displayConfig)

	// Define a special query value when reading standard input.
	if config.ReadStdin {
		config.Queries = []string{STDIN_QUERY_NAME}
	}

	// Execute the specified mode.
	switch {
	case config.ReadStdin:
		slog.Debug("Reading from standard input")

		doneQueriesChan, pauseQueryChans = lib.Query(
			lib.QUERY_MODE_STDIN,
			-1, // Stdin mode is always continuous and the query itself must detect EOF.
			config.Delay,
			config.Queries,
			config.Port,
			config.History,
			resultsReadyChan,
		)

		// Use labels that match the defined value for queries.
		ctx = context.WithValue(ctx, "labels", config.Labels)
	case config.Mode == int(MODE_PROFILE):
		slog.Debug("Executing in profile mode")

		doneQueriesChan, pauseQueryChans = lib.Query(
			lib.QUERY_MODE_PROFILE,
			config.Count,
			config.Delay,
			config.Queries,
			config.Port,
			config.History,
			resultsReadyChan,
		)

		// Process mode has specific labels--ignore user provided ones.
		ctx = context.WithValue(ctx, "labels", lib.ProfileLabels)
	case config.Mode == int(MODE_QUERY):
		slog.Debug("Executing in query mode")

		doneQueriesChan, pauseQueryChans = lib.Query(
			lib.QUERY_MODE_COMMAND,
			config.Count,
			config.Delay,
			config.Queries,
			config.Port,
			config.History,
			resultsReadyChan,
		)

		// Rely on user-defined labels.
		ctx = context.WithValue(ctx, "labels", config.Labels)
	case config.Mode == int(MODE_READ):
		slog.Debug("Executing in read mode")

	// FIXME Temporarily disabling read mode.
	// 	done = lib.Read(port)
	default:
		slog.Error(fmt.Sprintf("Invalid mode: %d\n", config.Mode))
		os.Exit(1)
	}

	// Initialize remaining context.
	ctx = context.WithValue(ctx, "expressions", config.Expressions)
	ctx = context.WithValue(ctx, "filters", config.Filters)
	ctx = context.WithValue(ctx, "queries", config.Queries)

	// Execute result viewing.
	if !config.Silent {
		go lib.Results(
			ctx,
			lib.DisplayMode(config.DisplayMode),
			ctx.Value("queries").([]string)[0], // Always start with the first query.
			config.History,
			&displayConfig,
			&config,
			pauseQueryChans,
			resultsReadyChan,
		)
	}

	<-doneQueriesChan
	slog.Debug("Received the last result, nothing left to do")
	close(doneQueriesChan)
}
