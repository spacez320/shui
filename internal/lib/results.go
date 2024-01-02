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
	"github.com/mum4k/termdash/linestyle"
	termdashTcell "github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgetapi"
	"github.com/mum4k/termdash/widgets/sparkline"
	"github.com/mum4k/termdash/widgets/text"
	"github.com/rivo/tview"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Types
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Used to provide an io.Writer implementation of termdash text widgets.
type termdashTextWriter struct {
	text text.Text
}

// Implements io.Writer.
func (t *termdashTextWriter) Write(p []byte) (n int, err error) {
	t.text.Write(string(p))
	return len(p), nil
}

// Represents the display mode.
type DisplayMode int

// Represents the result mode value.
type ResultMode int

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Variables
//
///////////////////////////////////////////////////////////////////////////////////////////////////

const (
	HELP_SIZE            = 10 // Proportional size of the logs widget.
	LOGS_SIZE            = 15 // Proportional size of the logs widget.
	OUTER_PADDING_LEFT   = 10 // Left padding for the full display.
	OUTER_PADDING_RIGHT  = 10 // Right padding for the full display.
	OUTER_PADDING_TOP    = 5  // Top padding for the full display.
	OUTER_PADDING_BOTTOM = 5  // Bottom padding for the full display.
	RESULTS_SIZE         = 75 // Proportional size of the results widget.
	TABLE_PADDING        = 2  // Padding for table cell entries.
)

// Display constants. Each result mode uses a specific display.
const (
	DISPLAY_RAW      DisplayMode = iota + 1 // Used for direct output.
	DISPLAY_TVIEW                           // Used when tview is the TUI driver.
	DISPLAY_TERMDASH                        // Used when termdash is the TUI driver.

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
	// Global configuration.
	config Config
	// Display mode, dictated by the results.
	mode DisplayMode

	// Stored results.
	store = storage.NewStorage()

	// Widget for displaying logs. Publicly offered to allow log configuration.
	// Only applicable for tview result modes.
	LogsView *tview.TextView
)

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Private
//
///////////////////////////////////////////////////////////////////////////////////////////////////

// Sets-up the termdash container, which defines the overall layout, and begins
// running the display.
func initDisplayTermdash(resultsWidget, helpWidget, logsWidget widgetapi.Widget) {
	// Set-up the context and enable it to close on key-press.
	ctx, cancel := context.WithCancel(context.Background())

	// Set-up the layout.
	t, err := termdashTcell.New()
	e(err)

	// Render the widget.
	c, err := container.New(
		t,
		container.PaddingBottom(OUTER_PADDING_BOTTOM),
		container.PaddingLeft(OUTER_PADDING_LEFT),
		container.PaddingTop(OUTER_PADDING_TOP),
		container.PaddingRight(OUTER_PADDING_RIGHT),
		container.SplitHorizontal(
			container.Top(
				container.Border(linestyle.Light),
				container.BorderTitle("Results"),
				container.BorderTitleAlignCenter(),
				container.PlaceWidget(resultsWidget),
			),
			container.Bottom(
				container.SplitHorizontal(
					container.Top(
						container.Border(linestyle.Light),
						container.BorderTitle("Help"),
						container.BorderTitleAlignCenter(),
						container.PlaceWidget(helpWidget),
					),
					container.Bottom(
						container.Border(linestyle.Light),
						container.BorderTitle("Logs"),
						container.BorderTitleAlignCenter(),
						container.PlaceWidget(logsWidget),
					),
					container.SplitOption(container.SplitPercent(getRelativePerc(RESULTS_SIZE, HELP_SIZE))),
				),
			),
			container.SplitOption(container.SplitPercent(RESULTS_SIZE)),
		),
	)
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

// Sets-up the tview flex box, which defines the overall layout.
//
// Note that the app needs to be run separately from initialization in the
// coroutine display function. Note also that direct manipulation of the tview
// Primitives as subclasses (like tview.Box) needs to happen outside this
// function, as well.
func initDisplayTview(resultsView, helpView, logsView tview.Primitive) {
	// Initialize the app.
	app = tview.NewApplication()

	// Set-up the layout and apply views.
	flexBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(resultsView, 0, RESULTS_SIZE, false).
		AddItem(helpView, 0, HELP_SIZE, false).
		AddItem(logsView, 0, LOGS_SIZE, false)
	flexBox.SetBorderPadding(
		OUTER_PADDING_TOP,
		OUTER_PADDING_BOTTOM,
		OUTER_PADDING_LEFT,
		OUTER_PADDING_RIGHT,
	)
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

// Gives a new percentage based on globalRelativePerc after reducing by
// limitingPerc.
//
// For example, given a three-way percentage split of 80/10/10, this function
// will return 50 if given the arguments 80 and 10.
func getRelativePerc(limitingPerc, globalRelativePerc int) int {
	return (100 * globalRelativePerc) / (100 - limitingPerc)
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
func AddResult(query, result string) {
	result = strings.TrimSpace(result)
	store.Put(query, result, TokenizeResult(result)...)
}

// Creates a result with filtered values.
func FilterResult(result storage.Result, labels, filters []string) storage.Result {
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
	resultValues = FilterSlice(result.Values, labelIndexes)

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
func Results(resultMode ResultMode, query string, labels, filters []string, inputConfig Config) {
	// Assign global config.
	config = inputConfig
	// Set up labelling or any schema for the results store.
	store.PutLabels(query, labels)

	switch resultMode {
	case RESULT_MODE_RAW:
		mode = DISPLAY_RAW
		RawResults()
	case RESULT_MODE_STREAM:
		mode = DISPLAY_TVIEW
		StreamResults(query)
	case RESULT_MODE_TABLE:
		// Pass logs into the logs view pane.
		slog.SetDefault(slog.New(slog.NewTextHandler(
			LogsView,
			&slog.HandlerOptions{Level: config.SlogLogLevel()},
		)))

		mode = DISPLAY_TVIEW
		TableResults(query, filters)
	case RESULT_MODE_GRAPH:
		mode = DISPLAY_TERMDASH
		GraphResults(query, filters)
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
func StreamResults(query string) {
	var (
		helpText    = "(ESC) Quit"        // Text to display in the help pane.
		helpView    = tview.NewTextView() // Help text container.
		logsView    = tview.NewTextView() // Logs text container.
		resultsView = tview.NewTextView() // Results container.
	)

	// Initialize the results view.
	resultsView.SetChangedFunc(
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

	// Initialize the help view.
	helpView.SetBorder(true).SetTitle("Help")
	fmt.Fprintln(helpView, helpText)

	// Initialize the logs view.
	logsView.SetBorder(true).SetTitle("Logs")
	slog.SetDefault(slog.New(slog.NewTextHandler(
		logsView,
		&slog.HandlerOptions{Level: config.SlogLogLevel()},
	)))

	// Initialize the display.
	initDisplayTview(resultsView, helpView, logsView)

	// Start the display.
	display(
		func() {
			// Print labels as the first line, if they are present.
			if labels := store.GetLabels(query); len(labels) > 0 {
				fmt.Fprintln(resultsView, labels)
			}

			// Print results.
			for {
				fmt.Fprintln(resultsView, (<-storage.PutEvents).Value)
			}
		},
	)
}

// Creates a table of results for the results pane.
func TableResults(query string, filters []string) {
	var (
		helpText         = "(ESC) Quit"                       // Text to display in the help pane.
		helpView         = tview.NewTextView()                // Help text container.
		logsView         = tview.NewTextView()                // Logs text container.
		resultsView      = tview.NewTable()                   // Results container.
		tableCellPadding = strings.Repeat(" ", TABLE_PADDING) // Padding to add to table cell content.
		valueIndexes     = []int{}                            // Indexes of the result values to add to the table.
	)

	// Initialize the results view.
	resultsView.SetBorders(true).SetDoneFunc(
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

	// Initialize the help view.
	helpView.SetBorder(true).SetTitle("Help")
	fmt.Fprintln(helpView, helpText)

	// Initialize the logs view.
	logsView.SetBorder(true).SetTitle("Logs")
	slog.SetDefault(slog.New(slog.NewTextHandler(
		logsView,
		&slog.HandlerOptions{Level: config.SlogLogLevel()},
	)))

	// Determine the value indexes to populate into the graph. If no filter is
	// provided, the index is assumed to be zero.
	if len(filters) > 0 {
		for _, filter := range filters {
			valueIndexes = append(valueIndexes, store.GetValueIndex(query, filter))
		}
	}

	// Initialize the display.
	initDisplayTview(resultsView, helpView, logsView)

	// Start the display.
	display(
		func() {
			var (
				i = 0 // Used to determine the next row index.
			)

			// Create the table header.
			if labels := store.GetLabels(query); len(labels) > 0 {
				// Labels to apply.
				labels = FilterSlice(labels, valueIndexes)
				// Row to contain the labels.
				headerRow := resultsView.InsertRow(i)

				for j, label := range labels {
					headerRow.SetCellSimple(i, j, tableCellPadding+label+tableCellPadding)
				}

				app.Draw()
				i += 1
			}

			for {
				/* slog.Debug(fmt.Sprintf("Receiving value %v", <-storage.PutEvents)) */
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
func GraphResults(query string, filters []string) {
	var (
		helpText   = "(ESC) Quit" // Text to display in the help pane.
		valueIndex = 0            // Index of the result value to graph.
	)

	// Determine the values to populate into the graph. If no filter is provided,
	// the first value is taken.
	if len(filters) > 0 {
		valueIndex = store.GetValueIndex(query, filters[0])
	}

	// Initialize the results view.
	resultWidget, err := sparkline.New(
		sparkline.Label(store.GetLabels(query)[valueIndex]),
		sparkline.Color(cell.ColorGreen),
	)
	e(err)

	// Initialize the help view.
	helpWidget, err := text.New()
	e(err)
	helpWidget.Write(helpText)

	// Initialize the logs view.
	logsWidget, err := text.New()
	e(err)
	logsWidgetWriter := termdashTextWriter{text: *logsWidget}
	slog.SetDefault(slog.New(slog.NewTextHandler(
		&logsWidgetWriter,
		&slog.HandlerOptions{Level: config.SlogLogLevel()},
	)))

	// Start the display.
	display(
		func() {
			for {
				value := (<-storage.PutEvents).Values[valueIndex]

				switch value.(type) {
				case int64:
					resultWidget.Add([]int{int(value.(int64))})
				case float64:
					resultWidget.Add([]int{int(value.(float64))})
				}
			}
		},
	)

	// Initialize the display. This must happen after the display function is
	// invoked, otherwise data will never appear.
	initDisplayTermdash(resultWidget, helpWidget, &logsWidgetWriter.text)
}
