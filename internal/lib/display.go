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
)

// Display driver constants. Each display mode uses a specific display driver.
const (
	DISPLAY_RAW      DisplayDriver = iota + 1 // Used for direct output.
	DISPLAY_TVIEW                             // Used when tview is the TUI driver.
	DISPLAY_TERMDASH                          // Used when termdash is the TUI driver.
)

// Display mode constants.
const (
	DISPLAY_MODE_RAW    DisplayMode = iota + 1 // For running in 'raw' display mode.
	DISPLAY_MODE_STREAM                        // For running in 'stream' display mode.
	DISPLAY_MODE_TABLE                         // For running in 'table' display mode.
	DISPLAY_MODE_GRAPH                         // For running in 'graph' display mode.
)

const (
	HELP_TEXT            = "(ESC) Quit | (Space) Pause | (Tab) Next Display | (n) Next Query"
	HELP_SIZE            = 10 // Proportional size of the logs widget.
	LOGS_SIZE            = 15 // Proportional size of the logs widget.
	OUTER_PADDING_LEFT   = 10 // Left padding for the full display.
	OUTER_PADDING_RIGHT  = 10 // Right padding for the full display.
	OUTER_PADDING_TOP    = 5  // Top padding for the full display.
	OUTER_PADDING_BOTTOM = 5  // Bottom padding for the full display.
	RESULTS_SIZE         = 75 // Proportional size of the results widget.
	TABLE_PADDING        = 2  // Padding for table cell entries.
)

var (
	activeDisplayModes = []DisplayMode{
		// DISPLAY_MODE_RAW,  // It's impossible to escape raw mode, so we exclude it.
		DISPLAY_MODE_STREAM,
		DISPLAY_MODE_TABLE,
		DISPLAY_MODE_GRAPH,
	} // Display modes considered for use in the current session.
	interruptChan = make(chan bool) // Channel for interrupting displays.
)

// Represents the display driver.
type DisplayDriver int

// Represents the display mode.
type DisplayMode int

// Starts the display. Applies contextual logic depending on the provided display driver. Expects a
// function to execute within a goroutine to update the display.
func display(driver DisplayDriver, displayUpdateFunc func()) {
	// Execute the update function.
	go displayUpdateFunc()

	switch driver {
	case DISPLAY_TVIEW:
		// Start the tview-specific display.
		err := appTview.Run()
		e(err)
	case DISPLAY_TERMDASH:
		// Start the termdash-specific display.
		// Nothing to do, yet.
	}
}

// Clean-up display logic when fully quitting.
func displayQuit() {
	close(interruptChan)
}

// Creates help text for any display.
func helpText() string {
	return HELP_TEXT + fmt.Sprintf(
		"\nQuery: %v | Labels: %v | Filters: %v",
		currentCtx.Value("query"),
		store.GetLabels(currentCtx.Value("query").(string)),
		currentCtx.Value("filters"))
}

// Presents raw output.
func RawDisplay(query string) {
	var (
		reader = readerIndexes[query] // Reader index for the query.
	)

	// Wait for the first result to appear to synchronize storage.
	GetResultWait(query)
	reader.Dec()

	// Load existing results.
	for _, result := range store.GetToIndex(query, reader) {
		fmt.Println(result)
	}

	// Load new results.
	for {
		fmt.Println(GetResult(query))
	}
}

// Update the results pane with new results as they are generated.
func StreamDisplay(query string, showHelp, showLogs bool) {
	var (
		reader = readerIndexes[query] // Reader index for the query.
	)

	// Wait for the first result to appear to synchronize storage.
	GetResultWait(query)
	reader.Dec()

	// Initialize the display.
	resultsView, helpView, logsView, flexBox := initDisplayTviewText(helpText())

	// Hide displays we don't want to show.
	if !showHelp {
		flexBox.RemoveItem(helpView)
	}
	if !showLogs {
		flexBox.RemoveItem(logsView)
	}

	// Start the display.
	display(
		DISPLAY_TVIEW,
		func() {
			// Print labels as the first line.
			appTview.QueueUpdateDraw(func() {
				fmt.Fprintln(resultsView, store.GetLabels(query))
			})

			// Print all previous results.
			for _, result := range store.GetToIndex(query, reader) {
				fmt.Fprintln(resultsView, result.Value)
			}

			// Print results.
			for {
				// Listen for an interrupt to stop result consumption for some display change.
				select {
				case <-interruptChan:
					// We've received an interrupt.
					return
				case <-pauseDisplayChan:
					// We've received a pause and need to wait for an unpause.
					<-pauseDisplayChan
				default:
					// We can display the next result.
					fmt.Fprintln(resultsView, (GetResult(query)).Value)
				}
			}
		},
	)
}

// Creates a table of results for the results pane.
func TableDisplay(query string, filters []string, showHelp, showLogs bool) {
	var (
		reader           = readerIndexes[query]               // Reader index for the query.
		tableCellPadding = strings.Repeat(" ", TABLE_PADDING) // Padding to add to table cell content.
		valueIndexes     = []int{}                            // Indexes of the result values to add to the table.
	)

	// Wait for the first result to appear to synchronize storage.
	GetResultWait(query)
	reader.Dec()

	// Initialize the display.
	resultsView, helpView, logsView, flexBox := initDisplayTviewTable(helpText())

	// Hide displays we don't want to show.
	if !showHelp {
		flexBox.RemoveItem(helpView)
	}
	if !showLogs {
		flexBox.RemoveItem(logsView)
	}

	// Start the display.
	display(
		DISPLAY_TVIEW,
		func() {
			var (
				nextCellContent string // Next cell to add to the table.
				i               = 0    // Used to determine the next row index.
			)

			// Determine the value indexes to populate into the graph. If no filter is provided, the index
			// is assumed to be zero.
			if len(filters) > 0 {
				for _, filter := range filters {
					valueIndexes = append(valueIndexes, store.GetValueIndex(query, filter))
				}
			}

			appTview.QueueUpdateDraw(func() {
				// Row to contain the labels.
				headerRow := resultsView.InsertRow(i)

				for j, label := range FilterSlice(store.GetLabels(query), valueIndexes) {
					headerRow.SetCellSimple(i, j, tableCellPadding+label+tableCellPadding)
				}
			})
			i += 1

			// Print all previous results.
			for _, result := range store.GetToIndex(query, reader) {
				appTview.QueueUpdateDraw(func() {
					var (
						row = resultsView.InsertRow(i) // Row to contain the result.
					)

					for j, value := range FilterSlice(result.Values, valueIndexes) {
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
				})
				i += 1
			}

			// Print results.
			for {
				// Listen for an interrupt to stop result consumption for some display change.
				select {
				case <-interruptChan:
					// We've received an interrupt.
					return
				case <-pauseDisplayChan:
					// We've received a pause and need to wait for an unpause.
					<-pauseDisplayChan
				default:
					// We can display the next result.
					appTview.QueueUpdateDraw(func() {
						var (
							row = resultsView.InsertRow(i) // Row to contain the result.
						)

						for j, value := range FilterSlice((GetResult(query)).Values, valueIndexes) {
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
					})
					i += 1
				}
			}
		},
	)
}

// Creates a graph of results for the results pane.
func GraphDisplay(query string, filters []string, showHelp, showLogs bool) {
	var (
		helpWidget, logsWidget *text.Text           // Accessory widgets to display.
		resultsWidget          *sparkline.SparkLine // Results widget.

		reader     = readerIndexes[query] // Reader index for the query.
		valueIndex = 0                    // Index of the result value to graph.
	)

	// Wait for the first result to appear to synchronize storage.
	GetResultWait(query)
	reader.Dec()

	// Determine the values to populate into the graph. If none is provided, the first value is taken.
	if len(filters) > 0 {
		valueIndex = store.GetValueIndex(query, filters[0])
	}

	// Initialize the results view.
	resultsWidget, err := sparkline.New(
		sparkline.Label(store.GetLabels(query)[valueIndex]),
		sparkline.Color(cell.ColorGreen),
	)
	e(err)

	// Initialize the help view.
	if showHelp {
		helpWidget, err = text.New()
		e(err)
		helpWidget.Write(helpText())
	}

	// Initialize the logs view.
	if showLogs {
		logsWidget, err = text.New()
		e(err)
	}

	// Start the display.
	display(
		DISPLAY_TERMDASH,
		func() {
			// Print all previous results.
			for _, result := range store.GetToIndex(query, reader) {
				// We can display the next result.
				value := result.Values[valueIndex]

				switch value.(type) {
				case int64:
					resultsWidget.Add([]int{int(value.(int64))})
				case float64:
					resultsWidget.Add([]int{int(value.(float64))})
				}
			}

			for {
				// Listen for an interrupt to stop result consumption for some display change.
				select {
				case <-interruptChan:
					// We've received an interrupt.
					return
				case <-pauseDisplayChan:
					// We've received a pause and need to wait for an unpause.
					<-pauseDisplayChan
				default:
					// We can display the next result.
					value := (GetResult(query)).Values[valueIndex]

					switch value.(type) {
					case int64:
						resultsWidget.Add([]int{int(value.(int64))})
					case float64:
						resultsWidget.Add([]int{int(value.(float64))})
					}
				}
			}
		},
	)

	// Initialize the display. This must happen after the display function is invoked, otherwise data
	// will never appear.
	initDisplayTermdash(termDashWidgets{
		resultsWidget: resultsWidget,
		helpWidget:    helpWidget,
		logsWidget:    logsWidget,
	})
}
