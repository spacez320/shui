//
// Display management for modes using Termdash.

package lib

import (
	"context"
	"os"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgetapi"
	"github.com/mum4k/termdash/widgets/text"
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
		// When a user presses Esc, close the application.
		currentCtx = context.WithValue(currentCtx, "quit", true)
		cancel()
		appTermdash.Close()
		os.Exit(0)
	case keyboard.KeyTab:
		// When a user presses Tab, stop the display but continue running.
		currentCtx = context.WithValue(currentCtx, "advanceQuery", true)
		cancel()
		appTermdash.Close()
	}
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
	termdash.Run(ctx, appTermdash, c, termdash.KeyboardSubscriber(keyboardTermdashHandler))
}
