//
// Results management.

package lib

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"

	"pkg/storage"

	"github.com/rivo/tview"
)

var (
	app     *tview.Application // Application for display.
	results storage.Results    // Stored results.

	LogsView    *tview.TextView // View for miscellaneous log output.
	ResultsView *tview.TextView // View for results.
)

// Initializes the results display.
func init() {
	app = tview.NewApplication()

	ResultsView = tview.NewTextView().SetChangedFunc(
		func() {
			app.Draw()
		})
	ResultsView.SetBorder(true).SetTitle("Results")

	LogsView = tview.NewTextView().SetChangedFunc(
		func() {
			app.Draw()
		})
	LogsView.SetBorder(true).SetTitle("Logs")

	flexBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ResultsView, 0, 3, false).
		AddItem(LogsView, 0, 1, false)

	app = app.SetRoot(flexBox, true).SetFocus(flexBox)
}

// Adds a result to the result store.
//
// TODO In the future, multiple result stores could be implemented by making
// this a function of an interface.
func AddResult(result string) {
	result = strings.TrimSpace(result)
	results.Put(result, TokenizeResult(result)...)
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
			fmt.Fprintln(ResultsView, (<-storage.PutEvents).Value)
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
			fmt.Println(<-storage.PutEvents)
		}
	}()
}

// Creates a table of results for the results pane.
func TableResults() {
	// Override the results view to create a table, instead.

	ResultsView := tview.NewTable().SetBorders(true)

	flexBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ResultsView, 0, 3, false).
		AddItem(LogsView, 0, 1, false)

	app = app.SetRoot(flexBox, true).SetFocus(flexBox)

	go func() {
		// Draw some test data.
		i := 0
		for {
			ResultsView.InsertRow(i).SetCellSimple(i, 0, (<-storage.PutEvents).Value.(string))
			i += 1
		}
	}()

	// Start the display.
	err := app.Run()
	e(err)
}
