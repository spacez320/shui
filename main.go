package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"
)

// General error manager.
func e(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Command execution.

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

	// Testing for storage.

	then := time.Now()

	fmt.Println("Inserting entries ...")
	results := Results{}
	results.Put("test")
	results.Put("herp")
	results.Put("derp")
	results.Put(0.1)
	results.Put(0.2)
	results.Put(0.3)

	fmt.Println("Results are:")
	results.Show()

	fmt.Println("Querying entries ...")
	fmt.Println(results.Get(then))                       // Should return the first entry.
	fmt.Println(results.Get(time.Now()))                 // Should return a nil entry.
	fmt.Println(len(results.GetRange(then, time.Now()))) // Should return all entries.
}
