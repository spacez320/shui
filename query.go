//
// Logic for 'query' mode.

package main

import (
	"io"
	"os/exec"
	"time"
)

func modeQuery() {
	// Start the RPC server.
	initServer()

	// This loop executes as long as attempts has not been reached or
	// indefinitely if attempts is less than zero.
	for i := 0; attempts < 0 || i < attempts; i++ {
		// Prepare query execution.
		logger.Printf("Executing query: '%s' ...\n", query)
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
			logger.Fatalf("Error is: \n%s\n", cmd_stderr_output)
		}

		// Interpret results.
		cmd_output, cmd_output_err := io.ReadAll(stdout)
		e(cmd_output_err)
		results.Put(cmd_output)
		logger.Printf("Result is: \n%s\n", cmd_output)

		// Clean-up.
		cmd.Wait()

		// This is not the last execution--add a delay.
		if i != attempts {
			time.Sleep(time.Duration(delay) * time.Second)
		}
	}

	// Print out results for debugging.
	logger.Printf("Results are: %v\n", results)
}
