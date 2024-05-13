//
// Laucnher for Cryptarch.

package cryptarch

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/exp/slog"

	"github.com/spacez320/cryptarch/internal/lib"
)

// Represents the mode value.
type queryMode int

// Mode constants.
const (
	MODE_QUERY   queryMode = iota + 1 // For running in 'query' mode.
	MODE_PROFILE                      // For running in 'profile' mode.
	MODE_READ                         // For running in 'read' mode.
)

var (
	ctx = context.Background() // Initialize context.
)

// Executes a Cryptarch.
func Run(config lib.Config, displayConfig lib.DisplayConfig) {
	var (
		doneQueriesChan chan bool            // Channel for tracking query completion.
		pauseQueryChans map[string]chan bool // Channels for pausing queries.

		resultsReadyChan = make(chan bool) // Channel for signaling results readiness.
	)
	slog.Debug("Running with config", "config", config)
	slog.Debug("Running with display config", "displayConfig", displayConfig)

	// Execute the specified mode.
	switch {
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
	ctx = context.WithValue(ctx, "filters", config.Filters)
	ctx = context.WithValue(ctx, "queries", config.Queries)

	// Execute result viewing.
	if !config.Silent {
		lib.Results(
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

	// XXX This isn't strictly necessary, mainly because getting here shouldn't be possible
	// (`lib.Results` does not have any intentional return condition), but it's being left here in
	// case in the future we do want to control for query completion.
	<-doneQueriesChan
	close(doneQueriesChan)
}
