//
// Logic for 'query' mode.

package main

import (
	"fmt"
	"io"
	"os/exec"
	"time"

	"golang.org/x/exp/slog"
)

// Entrypoint for 'query' mode.
func Query() chan int {
	var (
		done      = make(chan int, 1)             // Signals overall completion.
		doneQuery = make(chan bool, len(queries)) // Signals query completions.
	)

	// Start the RPC server.
	initServer()

	// Execute the queries.
	for _, query := range queries {
		go runQuery(query, doneQuery)
	}

	go func() {
		// Wait for the queries to finish.
		for i := 0; i < len(queries); i++ {
			<-doneQuery
		}

		// Signal overall completion.
		done <- 1
	}()

	return done
}

// Executes a query.
func runQuery(query string, doneQuery chan bool) {
	// This loop executes as long as attempts has not been reached or
	// indefinitely if attempts is less than zero.
	for i := 0; attempts < 0 || i < attempts; i++ {
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
			slog.Error(fmt.Sprintf("Error is: \n%s\n", cmd_stderr_output))
		}

		// Interpret results.
		cmd_output, cmd_output_err := io.ReadAll(stdout)
		e(cmd_output_err)
		AddResult(string(cmd_output))
		slog.Debug(fmt.Sprintf("Result is: \n%s\n", cmd_output))

		// Clean-up.
		cmd.Wait()

		// This is not the last execution--add a delay.
		if i != attempts {
			time.Sleep(time.Duration(delay) * time.Second)
		}
	}

	doneQuery <- true // Signals that this query is finished.
}
