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
		cmd := exec.Command(query)
		stdout, stdout_err := cmd.StdoutPipe()
		e(stdout_err)

		// Execute the query.
		cmd_err := cmd.Start()
		e(cmd_err)

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
