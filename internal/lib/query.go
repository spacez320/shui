//
// Logic for 'query' mode.

package lib

import (
	"io"
	"log/slog"
	"os/exec"
	"strconv"
	"time"
)

const (
	QUERY_MODE_COMMAND int = iota + 1 // Queries are commands.
	QUERY_MODE_PROFILE                // Queries are PIDs to profile.
)

// Wrapper for query execution.
func runQuery(
	query string,
	attempts, delay int,
	history bool,
	doneChan, pauseChan chan bool,
	queryFunc func(string, bool),
) {
	// This loop executes as long as attempts has not been reached, or indefinitely if attempts is
	// less than zero.
	for i := 0; attempts < 0 || i < attempts; i++ {
		select {
		case <-pauseChan:
			// Manage pausing. If we receive from the pause channel, wait for another message from the
			// pause channel.
			<-pauseChan
		default:
			queryFunc(query, history)

			// This is not the last execution--add a delay.
			if i != attempts {
				time.Sleep(time.Duration(delay) * time.Second)
			}
		}
	}

	doneChan <- true
}

// Executes a query as a command to exec.
func runQueryExec(query string, history bool) {
	slog.Debug("Executing query", "query", query)

	// Prepare query execution.
	cmd := exec.Command("bash", "-c", query)

	// Set-up pipes for command output.
	stdout, stdout_err := cmd.StdoutPipe()
	stderr, stderr_err := cmd.StderrPipe()
	e(stdout_err)
	e(stderr_err)

	// Execute the query.
	cmd_err := cmd.Start()
	e(cmd_err)

	// Manage potential errors coming from the command itself.
	cmd_stderr_output, cmd_stderr_output_err := io.ReadAll(stderr)
	e(cmd_stderr_output_err)
	if len(cmd_stderr_output) != 0 {
		slog.Error("Query error", "query", query)
	}

	// Interpret results.
	cmd_output, cmd_output_err := io.ReadAll(stdout)
	e(cmd_output_err)

	// Store results.
	slog.Debug("Query success", "query", query, "result", cmd_output)
	AddResult(query, string(cmd_output), history)

	// Clean-up.
	cmd.Wait()
}

// Executes a query as a process to profile.
func runQueryProfile(pid string, history bool) {
	slog.Debug("Profiling pid", "pid", pid)

	pidInt, err := strconv.Atoi(pid)
	e(err)
	AddResult(pid, runProfile(pidInt), history)
}

// Entrypoint for 'query' mode.
func Query(
	queryMode, attempts, delay int,
	queries []string,
	port string,
	history bool,
	resultsReadyChan chan bool,
) (chan bool, map[string]chan bool) {
	var (
		doneQueriesChan = make(chan bool)                          // Signals overall completion.
		doneQueryChan   = make(chan bool, len(queries))            // Signals specific query completions.
		pauseQueryChans = make(map[string]chan bool, len(queries)) // Signals query pausing.
	)

	// Start the RPC server.
	initServer(port)

	for _, query := range queries {
		// Initialize pause channels.
		pauseQueryChans[query] = make(chan bool)
	}

	go func() {
		// Wait for result consumption to become ready.
		slog.Debug("Waiting for results readiness")
		<-resultsReadyChan

		for _, query := range queries {
			// Execute the queries.
			switch queryMode {
			case QUERY_MODE_COMMAND:
				slog.Debug("Executing in query mode command")
				go runQuery(
					query,
					attempts,
					delay,
					history,
					doneQueryChan,
					pauseQueryChans[query],
					runQueryExec,
				)
			case QUERY_MODE_PROFILE:
				slog.Debug("Executing in query mode profile")
				go runQuery(
					query,
					attempts,
					delay,
					history,
					doneQueryChan,
					pauseQueryChans[query],
					runQueryProfile,
				)
			}
		}
	}()

	// Begin the goroutine to wait for query completion.
	go func() {
		defer close(doneQueryChan)

		// Wait for the queries to finish.
		for i := 0; i < len(queries); i++ {
			<-doneQueryChan
		}

		// Signal overall completion.
		doneQueriesChan <- true
	}()

	return doneQueriesChan, pauseQueryChans
}
