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
	t, err := tcell.New()
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
					container.SplitOption(container.SplitPercent(RelativePerc(RESULTS_SIZE, HELP_SIZE))),
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
