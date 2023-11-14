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
	Put[string]("test")
	Put[string]("herp")
	Put[string]("derp")
	Put[float64](0.1)
	Put[float64](0.2)
	Put[float64](0.3)

	fmt.Println("Results are:")
	Show()

	fmt.Println("Querying entries ...")
	fmt.Println(Get(then))                       // Should return the first entry.
	fmt.Println(Get(time.Now()))                 // Should return a nil entry.
	fmt.Println(len(GetRange(then, time.Now()))) // Should return all entries.
}
