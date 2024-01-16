//
// Logic for 'query' mode.

package lib

import (
	"fmt"
	"io"
	"os/exec"
	"time"

	"golang.org/x/exp/slog"
)

// Executes a query.
func runQuery(query string, attempts, delay int, doneChan, pauseChan chan bool) {
	// This loop executes as long as attempts has not been reached, or
	// indefinitely if attempts is less than zero.
	for i := 0; attempts < 0 || i < attempts; i++ {
		select {
		case <-pauseChan:
			// Manage pausing. If we receive from the pause channel, wait for another
			// message from the pause channel.
			slog.Debug(fmt.Sprintf("Pausing query: '%s'.\n.", query))
			<-pauseChan
			slog.Debug(fmt.Sprintf("Unpausing query: '%s'.\n.", query))
		default:
			// Prepare query execution.
			slog.Debug(fmt.Sprintf("Executing query: '%s' ...\n", query))
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
				slog.Error(fmt.Sprintf("Query '%s' error is: \n%s\n", query, cmd_stderr_output))
			}

			// Interpret results.
			cmd_output, cmd_output_err := io.ReadAll(stdout)
			e(cmd_output_err)
			AddResult(query, string(cmd_output))
			slog.Debug(fmt.Sprintf("Query '%s' result is: \n%s\n", query, cmd_output))

			// Clean-up.
			cmd.Wait()

			// This is not the last execution--add a delay.
			if i != attempts {
				time.Sleep(time.Duration(delay) * time.Second)
			}
		}
	}

	doneChan <- true // Signals that this query is finished.
}

// Entrypoint for 'query' mode.
func Query(
	queries []string,
	attempts int,
	delay int,
	port string,
) (chan bool, map[string]chan bool) {
	var (
		doneQueriesChan = make(chan bool)                          // Signals overall completion.
		doneQueryChan   = make(chan bool, len(queries))            // Signals query completions.
		pauseQueryChans = make(map[string]chan bool, len(queries)) // Pause channels for queries.
	)

	// Start the RPC server.
	initServer(port)

	for _, query := range queries {
		// Initialize pause channels.
		pauseQueryChans[query] = make(chan bool)

		// Execute the queries.
		go runQuery(query, attempts, delay, doneQueryChan, pauseQueryChans[query])
	}

	// Begin the goroutine to wait for query completion.
	go func() {
		// Wait for the queries to finish.
		for i := 0; i < len(queries); i++ {
			<-doneQueryChan
		}
		close(doneQueryChan)

		// Signal overall completion.
		doneQueriesChan <- true
	}()

	return doneQueriesChan, pauseQueryChans
}
