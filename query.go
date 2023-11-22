//
// Logic for 'query' mode.

package main

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"text/scanner"
	"time"
	"unicode"

	"golang.org/x/exp/slog"
)

// Entrypoint for 'query' mode.
func modeQuery() {
	var (
		doneQuery = make(chan bool, len(queries))
	)

	// Start the RPC server.
	initServer()

	// Execute the queries.
	for _, query := range queries {
		go runQuery(query, doneQuery)
	}

	// Wait for the queries to finish.
	for i := 0; i < len(queries); i++ {
		<-doneQuery
	}

	// Print out results for debugging.
	results.Show()
}

// Parses a query result into compound values.
func parseQueryOutput(output string) (parsed []interface{}) {
	var (
		s    scanner.Scanner // Scanner for tokenization.
		next string          // Next token to consider.
	)

	s.Init(strings.NewReader(output))
	s.IsIdentRune = func(r rune, i int) bool {
		// Separate all tokens exclusively by whitespace.
		return !unicode.IsSpace(r)
	}

	for token := s.Scan(); token != scanner.EOF; token = s.Scan() {
		next = s.TokenText()

		// Attempt to parse this value as an integer.
		nextInt, err := strconv.ParseInt(next, 10, 0)
		if err == nil {
			parsed = append(parsed, nextInt)
			continue
		}

		// Attempt to parse this value as a float.
		nextFloat, err := strconv.ParseFloat(next, 10)
		if err == nil {
			parsed = append(parsed, nextFloat)
			continue
		}

		// Everything else has failed--just pass it as a string.
		parsed = append(parsed, next)
	}

	return
}

// Executes a query.
func runQuery(query string, doneQuery chan bool) {
	// This loop executes as long as attempts has not been reached or
	// indefinitely if attempts is less than zero.
	for i := 0; attempts < 0 || i < attempts; i++ {
		// Prepare query execution.
		slog.Debug("Executing query: '%s' ...\n", query)
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
		// results.Put(cmd_output) // TODO Preserving, pending the removal of simple results.
		results.PutC(parseQueryOutput(string(cmd_output))...)
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
