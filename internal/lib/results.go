//
// Results management.
//
// Managing results involves:
//
// -  Organizing a storage of results.
// -  Managing the TUI libraries--rendering and interaction for results.
// -  Finding a place for accessory output, like logs.

package lib

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"

	"pkg/storage"

	"github.com/gdamore/tcell/v2"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/keyboard"
	termdashTcell "github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgetapi"
	"github.com/mum4k/termdash/widgets/sparkline"
	"github.com/rivo/tview"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Types
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Represents the display mode.
type display_ int

// Represents the result mode value.
type ResultMode int

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Variables
//
///////////////////////////////////////////////////////////////////////////////////////////////////

const (
	LOGS_SIZE     = 1 // Proportional size of the logs widget.
	RESULTS_SIZE  = 3 // Proportional size of the results widget.
	TABLE_PADDING = 2 // Padding for table cell entries.
)

// Display constants. Each result mode uses a specific display.
const (
	DISPLAY_RAW      display_ = iota + 1 // Used for direct output.
	DISPLAY_TVIEW                        // Used when tview is the TUI driver.
	DISPLAY_TERMDASH                     // Used when termdash is the TUI driver.

)

// Result mode constants.
const (
	RESULT_MODE_RAW    ResultMode = iota + 1 // For running in 'raw' result mode.
	RESULT_MODE_STREAM                       // For running in 'stream' result mode.
	RESULT_MODE_TABLE                        // For running in 'table' result mode.
	RESULT_MODE_GRAPH                        // For running in 'graph' result mode.
)

var (
	// Application for display. Only applicable for tview result modes.
	app *tview.Application
	// Display mode, dictated by the results.
	mode display_
	// Stored results.
	results storage.Results

	// Widget for displaying logs. Publicly offered to allow log configuration.
	// Only applicable for tview result modes.
	LogsView *tview.TextView
)

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Private
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Set-up the sync for logs used by some result modes.
func init() {
	// Initialized specifically for showing logs in a tview pane. Currently,
	// tview is the only supported display backend that supports logging, and
	// termdash will not show logs.
	//
	// Initializing this is harmless, even if tview won't be used.
	//
	// TODO This should be probably be managed outside of init and should be made
	// display mode agnostic.
	LogsView = tview.NewTextView().SetChangedFunc(func() { app.Draw() })
	LogsView.SetBorder(true).SetTitle("Logs")
}

// Sets-up the termdash display and renders a widget.
func initDisplayTermdash(resultsWidget widgetapi.Widget) {
	// Set-up the context and enable it to close on key-press.
	ctx, cancel := context.WithCancel(context.Background())

	// Set-up the layout.
	t, err := termdashTcell.New()
	e(err)

	// Render the widget.
	c, err := container.New(t, container.PlaceWidget(resultsWidget))
	e(err)

	// Run the display.
	termdash.Run(ctx, t, c, termdash.KeyboardSubscriber(
		func(k *terminalapi.Keyboard) {
			// When a user presses Esc, close the application.
			if k.Key == keyboard.KeyEsc {
				cancel()
				t.Close()
				os.Exit(0)
			}
		},
	))
}

// Sets-up the tview flex box with results and logs views, which defines the
// overall layout.
//
// Note that the app needs to be run separately from initialization in the
// coroutine display function.
func initDisplayTview(resultsView tview.Primitive, logsView tview.Primitive) {
	// Initialize the app.
	app = tview.NewApplication()

	// Set-up the layout and apply views.
	flexBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(resultsView, 0, RESULTS_SIZE, false).
		AddItem(logsView, 0, LOGS_SIZE, false)
	app.SetRoot(flexBox, true).SetFocus(resultsView)
}

// Starts the display. Expects a function to execute within a goroutine to
// update the display.
func display(f func()) {
	// Execute the update function.
	go func() { f() }()

	switch mode {
	case DISPLAY_TVIEW:
		// Start the tview-specific display.
		err := app.Run()
		e(err)
	case DISPLAY_TERMDASH:
		// Start the termdash-specific display.
		// Nothing to do, yet.
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Public
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Adds a result to the result store.
//
// TODO In the future, multiple result stores could be implemented by making
// this a function of an interface.
func AddResult(result string) {
	result = strings.TrimSpace(result)
	results.Put(result, TokenizeResult(result)...)
}

// Creates a result with filtered values.
func FilterResult(result storage.Result, labels []string, filters []string) storage.Result {
	var (
		// Indexes of labels from filters, corresponding to result values.
		labelIndexes = make([]int, len(filters))
		// Found result values.
		resultValues = make([]interface{}, len(filters))
	)

	// Find indexes to pursue for results.
	for i, filter := range filters {
		labelIndexes[i] = slices.Index(labels, filter)
	}

	// Filter the results.
	for i, index := range labelIndexes {
		resultValues[i] = result.Values[index]
	}

	return storage.Result{
		Time:   result.Time,
		Value:  result.Value,
		Values: resultValues,
	}
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

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Result Modes
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Entry-point function for results.
func Results(resultMode ResultMode, labels []string, filters []string, config Config) {
	// Set up labelling or any schema for the results store.
	results.Labels = labels

	switch resultMode {
	case RESULT_MODE_RAW:
		mode = DISPLAY_RAW
		RawResults()
	case RESULT_MODE_STREAM:
		// Pass logs into the logs view pane.
		slog.SetDefault(slog.New(slog.NewTextHandler(
			LogsView,
			&slog.HandlerOptions{Level: config.SlogLogLevel()},
		)))

		mode = DISPLAY_TVIEW
		StreamResults()
	case RESULT_MODE_TABLE:
		// Pass logs into the logs view pane.
		slog.SetDefault(slog.New(slog.NewTextHandler(
			LogsView,
			&slog.HandlerOptions{Level: config.SlogLogLevel()},
		)))

		mode = DISPLAY_TVIEW
		TableResults(filters)
	case RESULT_MODE_GRAPH:
		mode = DISPLAY_TERMDASH
		GraphResults(filters)
	default:
		slog.Error(fmt.Sprintf("Invalid result mode: %d\n", resultMode))
		os.Exit(1)
	}
}

// Presents raw output.
func RawResults() {
	go func() {
		for {
			fmt.Println(<-storage.PutEvents)
		}
	}()
}

// Update the results pane with new results as they are generated.
func StreamResults() {
	// Initialize the results view.
	resultsView := tview.NewTextView().SetChangedFunc(
		func() {
			app.Draw()
		}).SetDoneFunc(
		func(key tcell.Key) {
			switch key {
			case tcell.KeyEscape:
				// When a user presses Esc, close the application.
				app.Stop()
				os.Exit(0)
			}
		},
	)
	resultsView.SetBorder(true).SetTitle("Results")

	// Initialize the display.
	initDisplayTview(resultsView, LogsView)

	// Start the display.
	display(
		func() {
			// Print labels as the first line, if they are present.
			if len(results.Labels) > 0 {
				fmt.Fprintln(resultsView, results.Labels)
			}

			// Print results.
			for {
				fmt.Fprintln(resultsView, (<-storage.PutEvents).Value)
			}
		},
	)
}

// Creates a table of results for the results pane.
func TableResults(filters []string) {
	var (
		tableCellPadding = strings.Repeat(" ", TABLE_PADDING) // Padding to add to table cell content.
		valueIndexes     = []int{}                            // Indexes of the result values to add to the table.
	)

	// Initialize the results view.
	resultsView := tview.NewTable().SetBorders(true).SetDoneFunc(
		func(key tcell.Key) {
			switch key {
			case tcell.KeyEscape:
				// When a user presses Esc, close the application.
				app.Stop()
				os.Exit(0)
			}
		},
	)

	// Determine the value indexes to populate into the graph. If no filter is
	// provided, the index is assumed to be zero.
	if len(filters) > 0 {
		for _, filter := range filters {
			valueIndexes = append(valueIndexes, results.GetValueIndex(filter))
		}
	}

	// Initialize the display.
	initDisplayTview(resultsView, LogsView)

	// Start the display.
	display(
		func() {
			var (
				i = 0 // Used to determine the next row index.
			)

			// Create the table header.
			if len(results.Labels) > 0 {
				// Labels to apply.
				labels := FilterSlice(results.Labels, valueIndexes)
				// Row to contain the labels.
				headerRow := resultsView.InsertRow(i)

				for j, label := range labels {
					headerRow.SetCellSimple(i, j, tableCellPadding+label+tableCellPadding)
				}

				app.Draw()
				i += 1
			}

			for {
				// Retrieve specific next values.
				values := FilterSlice((<-storage.PutEvents).Values, valueIndexes)
				// Row to contain the result.
				row := resultsView.InsertRow(i)

				for j, value := range values {
					var nextCellContent string

					// Extrapolate the field types in order to print them out.
					switch value.(type) {
					case int64:
						nextCellContent = strconv.FormatInt(value.(int64), 10)
					case float64:
						nextCellContent = strconv.FormatFloat(value.(float64), 'f', -1, 64)
					default:
						nextCellContent = value.(string)
					}
					row.SetCellSimple(i, j, tableCellPadding+nextCellContent+tableCellPadding)
				}

				app.Draw()
				i += 1
			}
		},
	)
}

// Creates a graph of results for the results pane.
func GraphResults(filters []string) {
	var (
		valueIndex = 0 // Index of the result value to graph.
	)

	// Initialize the results view.
	graph, err := sparkline.New(
		sparkline.Label("Results"),
		sparkline.Color(cell.ColorGreen),
	)
	e(err)

	// Determine the values to populate into the graph. If no filter is provided,
	// the first value is taken.
	if len(filters) > 0 {
		valueIndex = results.GetValueIndex(filters[0])
	}

	// Start the display.
	display(
		func() {
			for {
				value := (<-storage.PutEvents).Values[valueIndex]

				switch value.(type) {
				case int64:
					graph.Add([]int{int(value.(int64))})
				case float64:
					graph.Add([]int{int(value.(float64))})
				}
			}
		},
	)

	// Initialize the display. This must happen after the display function is
	// invoked, otherwise data will never appear.
	initDisplayTermdash(graph)
}
