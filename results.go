//
// Results management.

package main

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"

	"github.com/rivo/tview"
)

var (
	app         *tview.Application // Application for display.
	logsView    *tview.TextView    // View for miscellaneous log output.
	resultsView *tview.TextView    // View for results.
	results     Results            // Stored results.
)

// Initializes the results display.
func init() {
	app = tview.NewApplication()

	resultsView = tview.NewTextView().SetChangedFunc(
		func() {
			app.Draw()
		})
	resultsView.SetBorder(true).SetTitle("Results")

	logsView = tview.NewTextView().SetChangedFunc(
		func() {
			app.Draw()
		})
	logsView.SetBorder(true).SetTitle("Logs")

	flexBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(resultsView, 0, 3, false).
		AddItem(logsView, 0, 1, false)

	app = app.SetRoot(flexBox, true).SetFocus(flexBox)
}

// Adds a result to the result store.
//
// TODO In the future, multiple result stores could be implemented by making
// this a function of an interface.
func AddResult(result string) {
	results.PutC(TokenizeResult(result)...)
}

// Parses a result into tokens for compound storage.
//
// TODO In the future, multiple result stores could be implemented by making
// this a function of an interface.
func TokenizeResult(result string) (parsedResult []interface{}) {
	var (
		s    scanner.Scanner // Scanner for tokenization.
		next string          // Next token to consider.
	)

	s.Init(strings.NewReader(result))
	s.IsIdentRune = func(r rune, i int) bool {
		// Separate all tokens exclusively by whitespace.
		return !unicode.IsSpace(r)
	}

	for token := s.Scan(); token != scanner.EOF; token = s.Scan() {
		next = s.TokenText()

		// Attempt to parse this value as an integer.
		nextInt, err := strconv.ParseInt(next, 10, 0)
		if err == nil {
			parsedResult = append(parsedResult, nextInt)
			continue
		}

		// Attempt to parse this value as a float.
		nextFloat, err := strconv.ParseFloat(next, 10)
		if err == nil {
			parsedResult = append(parsedResult, nextFloat)
			continue
		}

		// Everything else has failed--just pass it as a string.
		parsedResult = append(parsedResult, next)
	}

	return
}

// Update the results pane with new results as they are generated.
func StreamResults() {
	go func() {
		for {
			fmt.Fprintln(resultsView, <-PutEvents)
		}
	}()

	// Start the display.
	err := app.Run()
	e(err)
}

// Presents raw output.
func RawResults() {
	go func() {
		for {
			fmt.Println(<-PutEvents)
		}
	}()
}
