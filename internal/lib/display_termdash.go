//
// Display management for modes using Termdash.

package lib

import (
	"context"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgetapi"
	"github.com/mum4k/termdash/widgets/text"
	"golang.org/x/exp/slog"
)

// Used to provide an io.Writer implementation of termdash text widgets.
type termdashTextWriter struct {
	text text.Text
}

// Implements io.Writer.
func (t *termdashTextWriter) Write(p []byte) (n int, err error) {
	t.text.Write(string(p))
	return len(p), nil
}

var (
	appTermdash *tcell.Terminal    // Termdash display.
	cancel      context.CancelFunc // Cancel function for the termdash display.
)

// Function to call on keyboard events.
func keyboardTermdashHandler(key *terminalapi.Keyboard) {
	switch key.Key {
	case keyboard.KeyEsc:
		// Escape quits the program.
		slog.Debug("Quitting.")

		currentCtx = context.WithValue(currentCtx, "quit", true)
		cancel()
		appTermdash.Close()
	case keyboard.KeyTab:
		// Tab switches display modes.
		slog.Debug("Switching display mode.")

		interruptChan <- true
		currentCtx = context.WithValue(currentCtx, "advanceDisplayMode", true)
		cancel()
		appTermdash.Close()
	case 'n':
		// 'n' switches queries.
		slog.Debug("Switching query.")

		interruptChan <- true
		currentCtx = context.WithValue(currentCtx, "advanceQuery", true)
		cancel()
		appTermdash.Close()
	case ' ':
		// Space pauses.
		slog.Debug("Pausing.")

		pauseDisplayChan <- true
		pauseQueryChans[currentCtx.Value("query").(string)] <- true
	}
}

// Error management for termdash.
func errorTermdashHandler(e error) {
	// If we hit an error from termdash, just log it and try to continue. Cases
	// of errors seen so far make sense to ignore:
	//
	// - Unimplemented key-strokes.
	slog.Error(e.Error())
}

// Sets-up the termdash container, which defines the overall layout, and begins
// running the display.
func initDisplayTermdash(resultsWidget, helpWidget, logsWidget widgetapi.Widget) {
	var (
		ctx context.Context // Termdash specific context.
		err error           // General error holder.
	)

	// Set-up the context and enable it to close on key-press.
	ctx, cancel = context.WithCancel(context.Background())

	// Set-up the layout.
	appTermdash, err = tcell.New()
	e(err)

	// Render the widget.
	c, err := container.New(
		appTermdash,
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
					container.SplitOption(container.SplitPercent(RelativePerc(RESULTS_SIZE, HELP_SIZE))),
				),
			),
			container.SplitOption(container.SplitPercent(RESULTS_SIZE)),
		),
	)
	e(err)

	// Run the display.
	termdash.Run(
		ctx,
		appTermdash,
		c,
		termdash.ErrorHandler(errorTermdashHandler),
		termdash.KeyboardSubscriber(keyboardTermdashHandler),
	)
}
