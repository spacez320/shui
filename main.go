package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
)

////////////////////////////////////////////////////////////////////////////////
//
// Types
//
////////////////////////////////////////////////////////////////////////////////

// Represents the mode value.
type mode_ int

// Queries provided as flags.
type queries_ []string

func (q *queries_) String() string {
	return fmt.Sprintf("%v", &q)
}

func (q *queries_) Set(query string) error {
	*q = append(*q, query)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
//
// Variables
//
////////////////////////////////////////////////////////////////////////////////

const (
	MODE_QUERY mode_ = iota // For running in 'query' mode.
	MODE_READ               // For running in 'read' mode.
)

var (
	attempts int      // Number of attempts to execute the query.
	delay    int      // Delay between queries.
	mode     int      // Mode to execute in.
	port     string   // Port for RPC.
	queries  queries_ // Queries to execute.
	results  Results  // Stored results.
	silent   bool     // Whether or not to be quiet.

	logger = log.Default() // Logging system.
)

////////////////////////////////////////////////////////////////////////////////
//
// Private
//
////////////////////////////////////////////////////////////////////////////////

// General error manager.
func e(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

func main() {
	// Define arguments.
	flag.BoolVar(&silent, "s", false, "Don't output anything to a console.")
	flag.IntVar(&attempts, "t", 1, "Number of query executions. -1 for continuous.")
	flag.IntVar(&delay, "d", 3, "Delay between queries (seconds).")
	flag.IntVar(&mode, "m", int(MODE_QUERY), "Mode to execute in.")
	flag.StringVar(&port, "p", "12345", "Port for RPC.")
	flag.Var(&queries, "q", "Query to execute.")
	flag.Parse()

	// Quiet logging if specified.
	if silent {
		logger.SetOutput(ioutil.Discard)
	}

	switch {
	case mode == int(MODE_QUERY):
		logger.Println("Executing in query mode.")
		modeQuery()
	case mode == int(MODE_READ):
		logger.Println("Executing in read mode.")
		modeRead()
	default:
		logger.Fatalf("Invalid mode: %v", mode)
	}
}
