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
func keyboardTviewHandler(key tcell.Key) {
	switch key {
	case tcell.KeyEscape:
		// When a user presses Esc, close the application.
		currentCtx = context.WithValue(currentCtx, "quit", true)
		appTview.Stop()
	case tcell.KeyTab:
		// This is wrapped in a goroutine to avoid deadlocks with tview.
		//
		// See: https://github.com/rivo/tview/issues/784
		go func() {
			// When a user presses Tab, stop the display but continue running.
			interruptChan <- true
			currentCtx = context.WithValue(currentCtx, "advanceQuery", true)
			appTview.Stop()
		}()
	}
}

// Display init function specific to table results.
func initDisplayTviewTable(helpText string) (
	resultsView *tview.Table,
	helpView, logsView *tview.TextView,
) {
	resultsView = tview.NewTable()
	helpView = tview.NewTextView()
	logsView = tview.NewTextView()

	// Initialize the results view.
	resultsView.SetBorders(true).SetDoneFunc(keyboardTviewHandler)
	resultsView.SetBorder(true).SetTitle("Results")

	initDisplayTview(resultsView, helpView, logsView, helpText)

	return
}

// Display init function specific to text results.
func initDisplayTviewText(helpText string) (resultsView, helpView, logsView *tview.TextView) {
	resultsView = tview.NewTextView()
	helpView = tview.NewTextView()
	logsView = tview.NewTextView()

	// Initialize the results view.
	resultsView.SetChangedFunc(
		func() {
			appTview.Draw()
		}).SetDoneFunc(keyboardTviewHandler)
	resultsView.SetBorder(true).SetTitle("Results")

	initDisplayTview(resultsView, helpView, logsView, helpText)

	return
}

// Sets-up the tview flex box, which defines the overall layout. Meant to
// encapsulate the common things needed regardless of what from the results
// view takes (assuming it fits into flex box).
//
// Note that the app needs to be run separately from initialization in the
// coroutine display function. Note also that direct manipulation of the tview
// Primitives as subclasses (like tview.Box) needs to happen outside this
// function, as well.
func initDisplayTview(
	resultsView tview.Primitive,
	helpView, logsView *tview.TextView,
	helpText string,
) {
	var (
		flexBox = tview.NewFlex() // Tview flexbox.
	)

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
	appTview.SetRoot(flexBox, true).SetFocus(resultsView)

	// Initialize the help view.
	helpView.SetBorder(true).SetTitle("Help")
	fmt.Fprintln(helpView, helpText)

	// Initialize the logs view.
	logsView.SetScrollable(false)
	logsView.SetBorder(true).SetTitle("Logs")
	slog.SetDefault(slog.New(slog.NewTextHandler(
		logsView,
		&slog.HandlerOptions{Level: config.SlogLogLevel()},
	)))
}
