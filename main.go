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
	// Query to execute upon.
	var query string

	// Define arguments.
	flag.StringVar(&query, "q", "whoami", "Query to execute.")
	flag.Parse()

	fmt.Printf("Executing query: '%s' ...\n", query)

	// Prepare query execution.
	cmd := exec.Command(query)
	stdout, stdout_err := cmd.StdoutPipe()
	e(stdout_err)

	// Execute the query.
	cmd_err := cmd.Start()
	e(cmd_err)

	// Interpret results.
	cmd_output, cmd_output_err := io.ReadAll(stdout)
	e(cmd_output_err)
	fmt.Printf("Result is: \n%s\n", cmd_output)

	// Clean-up.
	cmd.Wait()

	Put("test", 0.1)
	points := Get("test")
	for _, p := range points {
		fmt.Printf("timestamp: %v, values: %v\n", p.Timestamp, p.Value)
	}
}
