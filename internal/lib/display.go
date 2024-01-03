//
// Display (TUI) related logic.

package lib

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/widgets/sparkline"
	"github.com/mum4k/termdash/widgets/text"
	"golang.org/x/exp/slog"
)

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

///////////////////////////////////////////////////////////////////////////////////////////////////
//
// Private
//
///////////////////////////////////////////////////////////////////////////////////////////////////

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

// Presents raw output.
func RawDisplay(query string) {
	go func() {
		for {
			fmt.Println(GetResult(query))
		}
	}()
}

// Update the results pane with new results as they are generated.
func StreamDisplay(query string) {
	var (
		helpText = "(ESC) Quit | (n) Next Query" // Text to display in the help pane.
	)

	// Initialize the display.
	resultsView, _, _ := initDisplayTviewText(helpText)

	// Start the display.
	display(
		func() {
			// Print labels as the first line, if they are present.
			if labels := store.GetLabels(query); len(labels) > 0 {
				fmt.Fprintln(resultsView, labels)
			}

			// Print results.
			for {
				fmt.Fprintln(resultsView, (GetResult(query)).Value)
			}
		},
	)
}

// Creates a table of results for the results pane.
func TableDisplay(query string, filters []string) {
	var (
		helpText         = "(ESC) Quit"                       // Text to display in the help pane.
		tableCellPadding = strings.Repeat(" ", TABLE_PADDING) // Padding to add to table cell content.
		valueIndexes     = []int{}                            // Indexes of the result values to add to the table.
	)

	// Determine the value indexes to populate into the graph. If no filter is
	// provided, the index is assumed to be zero.
	if len(filters) > 0 {
		for _, filter := range filters {
			valueIndexes = append(valueIndexes, store.GetValueIndex(query, filter))
		}
	}

	// Initialize the display.
	resultsView, _, _ := initDisplayTviewTable(helpText)

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
				values := FilterSlice((GetResult(query)).Values, valueIndexes)
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
func GraphDisplay(query string, filters []string) {
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
				value := (GetResult(query)).Values[valueIndex]

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
