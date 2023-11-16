package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"time"
)

var (
	logger = log.Default() // Logging system.
)

// General error manager.
func e(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

func main() {
	var (
		attempts int     // Number of attempts to execute the query.
		delay    int     // Delay between queries.
		query    string  // Query to execute upon.
		results  Results // Stored results.
		silent   bool    // Whether or not to be quiet.
	)

	// Define arguments.
	flag.IntVar(&delay, "d", 3, "Delay between queries (seconds).")
	flag.IntVar(&attempts, "t", 1, "Number of query executions. -1 for continuous.")
	flag.StringVar(&query, "q", "whoami", "Query to execute.")
	flag.BoolVar(&silent, "s", false, "Don't output anything to a console.")
	flag.Parse()

	// Quiet logging if specified.
	if silent {
		logger.SetOutput(ioutil.Discard)
	}

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
