//
// Display management for modes using Tview.

package lib

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/exp/slog"
)

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
func initDisplayTviewTable(helpText string) (
	resultsView *tview.Table,
	helpView, logsView *tview.TextView,
	flexBox *tview.Flex,
) {
	resultsView = tview.NewTable()
	helpView = tview.NewTextView()
	logsView = tview.NewTextView()

	// Initialize the results view.
	resultsView.SetBorders(true)
	resultsView.SetBorder(true).SetTitle("Results")

	flexBox = initDisplayTview(resultsView, helpView, logsView, helpText)

	return
}

// Display init function specific to text results.
func initDisplayTviewText(helpText string) (
	resultsView, helpView, logsView *tview.TextView,
	flexBox *tview.Flex,
) {
	resultsView = tview.NewTextView()
	helpView = tview.NewTextView()
	logsView = tview.NewTextView()

	// Initialize the results view.
	resultsView.SetChangedFunc(func() { appTview.Draw() })
	resultsView.SetBorder(true).SetTitle("Results")

	flexBox = initDisplayTview(resultsView, helpView, logsView, helpText)

	return
}

// Sets-up the tview flex box, which defines the overall layout. Meant to encapsulate the common
// things needed regardless of what from the results view takes (assuming it fits into flex box).
//
// Note that the app needs to be run separately from initialization in the coroutine display
// function. Note also that direct manipulation of the tview Primitives as subclasses (like
// tview.Box) needs to happen outside this function, as well.
func initDisplayTview(
	resultsView tview.Primitive,
	helpView, logsView *tview.TextView,
	helpText string,
) (flexBox *tview.Flex) {
	flexBox = tview.NewFlex() // Tview flexbox.

	// Set-up the layout and apply views.
	flexBox = flexBox.SetDirection(tview.FlexRow).
		AddItem(resultsView, 0, RESULTS_SIZE, false).
		AddItem(helpView, 0, HELP_SIZE, false).
		AddItem(logsView, 0, LOGS_SIZE, false)
	flexBox.SetBorderPadding(
		OUTER_PADDING_TOP,
		OUTER_PADDING_BOTTOM,
		OUTER_PADDING_LEFT,
		OUTER_PADDING_RIGHT,
	)
	flexBox.SetInputCapture(keyboardTviewHandler)
	appTview.SetRoot(flexBox, true).SetFocus(resultsView)

	// Initialize the help view.
	helpView.SetBorder(true).SetTitle("Help")
	fmt.Fprintln(helpView, helpText)

	// Initialize the logs view.
	logsView.SetScrollable(false).SetChangedFunc(func() { appTview.Draw() })
	logsView.SetBorder(true).SetTitle("Logs")
	slog.SetDefault(slog.New(slog.NewTextHandler(
		logsView,
		&slog.HandlerOptions{Level: config.SlogLogLevel()},
	)))

	return
}
