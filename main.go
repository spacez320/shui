package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os/exec"
)

// General error manager.
func e(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var (
		attempts int     // Number of attempts to execute the query.
		query    string  // Query to execute upon.
		results  Results // Stored results.
	)

	// Define arguments.
	flag.IntVar(&attempts, "t", 1, "Number of query executions.")
	flag.StringVar(&query, "q", "whoami", "Query to execute.")
	flag.Parse()

	for i := 0; i < attempts; i++ {
		// Prepare query execution.
		fmt.Printf("Executing query: '%s' ...\n", query)
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
		fmt.Printf("Result is: \n%s\n", cmd_output)

		// Clean-up.
		cmd.Wait()
	}

	// Print out results for debugging.
	fmt.Printf("Results are: %v\n", results)
}
