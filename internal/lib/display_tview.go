//
// Display management for modes using tview.

package lib

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/exp/slog"
)

// Widgets for tview displays.
type tviewWidgets struct {
	flexBox                                                        *tview.Flex
	filterWidget, helpWidget, labelWidget, logsWidget, queryWidget *tview.TextView
	resultsWidget                                                  tview.Primitive
}

var (
	appTview = tview.NewApplication() // Tview application.
)

// Function to call on keyboard events.
func keyboardTviewHandler(event *tcell.EventKey) *tcell.EventKey {
	// Key events are wrapped in goroutines to avoid deadlocks with tview.
	//
	// See: https://github.com/rivo/tview/issues/784
	switch key := event.Key(); key {
	case tcell.KeyEscape:
		// Escape quits the program.
		slog.Debug("Quitting.")

		currentCtx = context.WithValue(currentCtx, "quit", true)
		appTview.Stop()
	case tcell.KeyRune:
		switch event.Rune() {
		case 'n':
			// 'n' switches queries.
			slog.Debug("Switching query.")

			go func() {
				// When a user presses Tab, stop the display but continue running.
				interruptChan <- true
				currentCtx = context.WithValue(currentCtx, "advanceQuery", true)
				appTview.Stop()
			}()
		case ' ':
			// Space pauses.
			slog.Debug("Pausing.")

			go func() {
				pauseDisplayChan <- true
				pauseQueryChans[currentCtx.Value("query").(string)] <- true
			}()
		}
	case tcell.KeyTab:
		// Tab switches display modes.
		slog.Debug("Switching display mode.")

		go func() {
			interruptChan <- true
			currentCtx = context.WithValue(currentCtx, "advanceDisplayMode", true)
			appTview.Stop()
		}()
	}

	return event
}

// Display init function specific to table results.
func initDisplayTviewTable(
	query string,
	filters, labels []string,
	displayConfig *DisplayConfig,
) (widgets tviewWidgets) {
	// Initialize the results view.
	widgets.resultsWidget = tview.NewTable()
	widgets.resultsWidget.(*tview.Table).SetBorders(true).SetBorder(true).SetTitle("Results")

	initDisplayTview(&widgets, query, filters, labels, displayConfig)

	return
}

// Display init function specific to text results.
func initDisplayTviewText(
	query string,
	filters, labels []string,
	displayConfig *DisplayConfig,
) (widgets tviewWidgets) {
	// Initialize the results viw.
	widgets.resultsWidget = tview.NewTextView()
	widgets.resultsWidget.(*tview.TextView).
		SetChangedFunc(func() { appTview.Draw() }).
		SetBorder(true).
		SetTitle("Results")

	initDisplayTview(&widgets, query, filters, labels, displayConfig)

	return
}

// Sets-up the tview flex box, which defines the overall layout. Meant to encapsulate the common
// things needed regardless of what from the results view takes (assuming it fits into flex box).
//
// Note that the app needs to be run separately from initialization in the coroutine display
// function. Note also that direct manipulation of the tview Primitives as subclasses (like
// tview.Box) needs to happen outside this function, as well.
func initDisplayTview(
	widgets *tviewWidgets,
	query string,
	filters, labels []string,
	displayConfig *DisplayConfig,
) {
	var (
		statusWidgets *tview.Flex // Container for status widgets.
	)

	widgets.filterWidget = tview.NewTextView()
	widgets.flexBox = tview.NewFlex()
	widgets.helpWidget = tview.NewTextView()
	widgets.labelWidget = tview.NewTextView()
	widgets.logsWidget = tview.NewTextView()
	widgets.queryWidget = tview.NewTextView()

	statusWidgets = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(widgets.queryWidget, 0, 1, false).
		AddItem(widgets.labelWidget, 0, 1, false).
		AddItem(widgets.filterWidget, 0, 1, false)

	// Set-up the layout and apply views.
	widgets.flexBox = widgets.flexBox.
		SetDirection(tview.FlexRow).
		AddItem(statusWidgets, 3, 0, false).
		AddItem(widgets.resultsWidget, 0, displayConfig.ResultsSize, false).
		AddItem(widgets.helpWidget, 3, 0, false).
		AddItem(widgets.logsWidget, 0, displayConfig.LogsSize, false)
	widgets.flexBox.SetBorderPadding(
		displayConfig.OuterPaddingTop,
		displayConfig.OuterPaddingBottom,
		displayConfig.OuterPaddingLeft,
		displayConfig.OuterPaddingRight,
	)
	widgets.flexBox.SetInputCapture(keyboardTviewHandler)
	appTview.SetRoot(widgets.flexBox, true).SetFocus(widgets.resultsWidget)

	// Initialize the help view.
	widgets.helpWidget.SetBorder(true).SetTitle("Help")
	fmt.Fprint(widgets.helpWidget, HELP_TEXT)

	// Initialize the top-line status widgets.
	widgets.filterWidget.SetBorder(true).SetTitle("Filters")
	fmt.Fprintf(widgets.filterWidget, "%v", filters)
	widgets.labelWidget.SetBorder(true).SetTitle("Labels")
	fmt.Fprintf(widgets.labelWidget, "%v", labels)
	widgets.queryWidget.SetBorder(true).SetTitle("Query")
	fmt.Fprintf(widgets.queryWidget, query)

	// Initialize the logs view.
	widgets.logsWidget.SetScrollable(false).SetChangedFunc(func() { appTview.Draw() })
	widgets.logsWidget.SetBorder(true).SetTitle("Logs")
	slog.SetDefault(slog.New(slog.NewTextHandler(
		widgets.logsWidget,
		&slog.HandlerOptions{Level: config.SlogLogLevel()},
	)))

	// Hide displays we don't want to show.
	if !displayConfig.ShowHelp {
		widgets.flexBox.RemoveItem(widgets.helpWidget)
	}
	if !displayConfig.ShowLogs {
		widgets.flexBox.RemoveItem(widgets.logsWidget)
	}
	if !displayConfig.ShowStatus {
		widgets.flexBox.RemoveItem(statusWidgets)
	}

	return
}
