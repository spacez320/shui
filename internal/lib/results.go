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

const (
	RESULTS_SIZE = 3 // Proportional size of the results widget.
	LOGS_SIZE    = 1 // Proportional size of the logs widget.
)

var (
	app     *tview.Application // Application for display.
	results storage.Results    // Stored results.

	// Widget for displaying logs. Publicly offered to allow log configuration.
	LogsView *tview.TextView
)

// Set-up the sync for logs used by some result modes.
func init() {
	LogsView = tview.NewTextView().SetChangedFunc(func() { app.Draw() })
	LogsView.SetBorder(true).SetTitle("Logs")
}

// Sets-up the flex box, which defines the overall layout.
func initDisplay(resultsView tview.Primitive, logsView tview.Primitive) {
	app = tview.NewApplication()

	flexBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(resultsView, 0, RESULTS_SIZE, false).
		AddItem(logsView, 0, LOGS_SIZE, false)

	app.SetRoot(flexBox, true).SetFocus(flexBox)
}

// Starts the display. Expects a function to execute within a goroutine to
// update the display.
func display(f func()) {
	// Execute the update function.
	go func() { f() }()

	// Start the display.
	err := app.Run()
	e(err)
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
	resultsView := tview.NewTextView().SetChangedFunc(
		func() {
			app.Draw()
		})
	resultsView.SetBorder(true).SetTitle("Results")

	initDisplay(resultsView, LogsView)

	display(
		func() {
			for {
				fmt.Fprintln(resultsView, (<-storage.PutEvents).Value)
			}
		},
	)
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
	resultsView := tview.NewTable().SetBorders(true)

	initDisplay(resultsView, LogsView)

	display(
		func() {
			i := 0 // Used to determine the next row index.

			for {
				// Retrieve the next result.
				next := <-storage.PutEvents

				// Display the new result.
				row := resultsView.InsertRow(i)
				for j, token := range next.Values {
					// Extrapolate the field types in order to print them out.
					switch token.(type) {
					case int64:
						row.SetCellSimple(i, j, strconv.FormatInt(token.(int64), 10))
					case float64:
						row.SetCellSimple(i, j, strconv.FormatFloat(token.(float64), 'E', -1, 64))
					default:
						row.SetCellSimple(i, j, token.(string))
					}
				}

				i += 1
			}
		},
	)

	// Start the display.
	err := app.Run()
	e(err)
}
