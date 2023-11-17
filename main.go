package main

import (
	"flag"
	"io/ioutil"
	"log"
)

type mode_ int

const (
	mode_query mode_ = iota
	mode_read
)

var (
	attempts int     // Number of attempts to execute the query.
	delay    int     // Delay between queries.
	mode     int     // Mode to execute in.
	port     string  // Port for RPC.
	query    string  // Query to execute upon.
	results  Results // Stored results.
	silent   bool    // Whether or not to be quiet.

	logger = log.Default() // Logging system.
)

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
	flag.StringVar(&port, "p", "12345", "Port for RPC.")
	flag.IntVar(&mode, "m", int(mode_query), "Mode to execute in.")
	flag.StringVar(&query, "q", "whoami", "Query to execute.")
	flag.Parse()

	// Quiet logging if specified.
	if silent {
		logger.SetOutput(ioutil.Discard)
	}

	switch {
	case mode == int(mode_query):
		logger.Println("Executing in query mode.")
		modeQuery()
	case mode == int(mode_read):
		logger.Println("Executing in read mode.")
		modeRead()
	default:
		logger.Fatalf("Invalid mode: %v", mode)
	}
}
