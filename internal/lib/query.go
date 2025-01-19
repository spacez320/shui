//
// Logic for 'query' mode.

package lib

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const (
	QUERY_MODE_COMMAND int = iota + 1 // Queries are commands.
	QUERY_MODE_PROFILE                // Queries are PIDs to profile.
	QUERY_MODE_STDIN                  // Results are fron stdin.
)

var (
	stdinScanner = bufio.NewScanner(os.Stdin) // Scanner for standard input queries.
)

// Wrapper for query execution.
func runQuery(
	query string,
	attempts, delay int,
	history bool,
	doneChan, pauseChan chan bool,
	queryFunc func(string, bool) bool,
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
			if !queryFunc(query, history) {
				// In the event that queryFunc returns false, allow this to signal query completion, even if
				// attempts are not satisifed.
				attempts = 0
			}

			// This is not the last execution--add a delay.
			if i != attempts {
				time.Sleep(time.Duration(delay) * time.Second)
			}
		}
	}

	slog.Debug("Query done", "query", query)
	doneChan <- true
}

// Executes a query as a command to exec.
func runQueryExec(query string, history bool) bool {
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

	return true
}

// Executes a query as a process to profile.
func runQueryProfile(pid string, history bool) bool {
	slog.Debug("Profiling pid", "pid", pid)

	pidInt, err := strconv.Atoi(pid)
	e(err)
	AddResult(pid, runProfile(pidInt), history)

	return true
}

// Reads standard input for results.
func runQueryStdin(query string, history bool) bool {
	var success = true

	slog.Debug("Reading stdin")

	if stdinScanner.Scan() {
		AddResult(query, stdinScanner.Text(), history)
	} else {
		success = false
	}

	return success
}

// Entrypoint for 'query' mode.
func Query(
	queryMode, attempts, delay int,
	queries []string,
	port int,
	history bool,
	resultsReadyChan chan bool,
) (chan bool, map[string]chan bool) {
	var (
		doneQueriesChan = make(chan bool)                          // Signals overall completion.
		doneQueryChan   = make(chan bool, len(queries))            // Signals specific query completions.
		pauseQueryChans = make(map[string]chan bool, len(queries)) // Signals query pausing.
	)

	// Start the RPC server.
	initServer(fmt.Sprintf("%d", port))

	go func() {
		// Wait for result consumption to become ready.
		slog.Debug("Waiting for results readiness")
		<-resultsReadyChan

		// Execute the queries.
		switch queryMode {
		case QUERY_MODE_COMMAND:
			slog.Debug("Executing in query mode command")

			for _, query := range queries {
				// Initialize pause channels.
				pauseQueryChans[query] = make(chan bool)

				go runQuery(
					query,
					attempts,
					delay,
					history,
					doneQueryChan,
					pauseQueryChans[query],
					runQueryExec,
				)
			}
		case QUERY_MODE_PROFILE:
			slog.Debug("Executing in query mode profile")

			for _, query := range queries {
				// Initialize pause channels.
				pauseQueryChans[query] = make(chan bool)

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
		case QUERY_MODE_STDIN:
			// When executing by reading standard input, there is only ever one "query".
			slog.Debug("Executing in query mode stdin")

			// Initialize pause channels.
			pauseQueryChans[queries[0]] = make(chan bool)

			go runQuery(
				queries[0],
				attempts,
				delay,
				history,
				doneQueryChan,
				pauseQueryChans[queries[0]],
				runQueryStdin,
			)
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
