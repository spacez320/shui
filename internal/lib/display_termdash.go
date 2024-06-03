//
// Display management for modes using Termdash.

package lib

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgetapi"
	"github.com/mum4k/termdash/widgets/text"
	slogmulti "github.com/samber/slog-multi"
)

// Used to provide an io.Writer implementation of termdash text widgets.
type termdashTextWriter struct {
	text *text.Text
}

// Implements io.Writer.
func (t *termdashTextWriter) Write(p []byte) (n int, err error) {
	err = t.text.Write(string(p))
	return len(p), err
}

// Used to supply optional widgets to Termdash initialization.
type termdashWidgets struct {
	filterWidget, helpWidget, labelWidget, logsWidget, queryWidget *text.Text
	resultsWidget                                                  widgetapi.Widget
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
		slog.Debug("Quitting")

		currentCtx = context.WithValue(currentCtx, "quit", true)
		cancel()
		appTermdash.Close()
	case keyboard.KeyTab:
		// Tab switches display modes.
		slog.Debug("Switching display mode")

		interruptChan <- true
		currentCtx = context.WithValue(currentCtx, "advanceDisplayMode", true)
		cancel()
		appTermdash.Close()
	case 'n':
		// 'n' switches queries.
		slog.Debug("Switching query")

		interruptChan <- true
		currentCtx = context.WithValue(currentCtx, "advanceQuery", true)
		cancel()
		appTermdash.Close()
	case ' ':
		// Space pauses.
		slog.Debug("Pausing")

		pauseDisplayChan <- true
		pauseQueryChans[currentCtx.Value("query").(string)] <- true
	}
}

// Error management for termdash.
func errorTermdashHandler(e error) {
	// If we hit an error from termdash, just log it and try to continue. Cases of errors seen so far
	// make sense to ignore:
	//
	// - Unimplemented key-strokes.
	slog.Error(e.Error())
}

// Sets-up the termdash container, which defines the overall layout, and begins running the display.
// func initDisplayTermdash(resultsWidget, helpWidget, logsWidget widgetapi.Widget) {
func initDisplayTermdash(
	widgets termdashWidgets,
	query string,
	filters, labels []string,
	displayConfig *DisplayConfig,
) {
	var (
		ctx               context.Context      // Termdash specific context.
		err               error                // General error holder.
		logsWidgetWriter  termdashTextWriter   // Writer implementation for logs.
		logsWidgetHandler slog.Handler         // Log handler for Termdash apps.
		mainWidgets       []container.Option   // Status and result widgets.
		widgetContainer   *container.Container // Wrapper for widgets.
	)
	widgets.filterWidget, err = text.New()
	e(err)
	widgets.labelWidget, err = text.New()
	e(err)
	widgets.queryWidget, err = text.New()
	e(err)

	// Instantiate optional displays.
	if displayConfig.ShowHelp {
		widgets.helpWidget, err = text.New()
		e(err)
		widgets.helpWidget.Write(HELP_TEXT)
	}
	if displayConfig.ShowLogs {
		widgets.logsWidget, err = text.New(text.RollContent())
		e(err)
	}

	// Set-up the context and enable it to close on key-press.
	ctx, cancel = context.WithCancel(context.Background())

	// Set-up the layout.
	appTermdash, err = tcell.New()
	e(err)

	// Set-up the status widgets with results.
	if displayConfig.ShowStatus {
		mainWidgets = []container.Option{
			container.SplitHorizontal(
				container.Top(
					container.SplitVertical(
						container.Left(
							container.Border(linestyle.Light),
							container.BorderTitle("Query"),
							container.BorderTitleAlignCenter(),
							container.PlaceWidget(widgets.queryWidget),
						),
						container.Right(
							container.SplitVertical(
								container.Left(
									container.Border(linestyle.Light),
									container.BorderTitle("Labels"),
									container.BorderTitleAlignCenter(),
									container.PlaceWidget(widgets.labelWidget),
								),
								container.Right(
									container.Border(linestyle.Light),
									container.BorderTitle("Filters"),
									container.BorderTitleAlignCenter(),
									container.PlaceWidget(widgets.labelWidget),
								),
							),
						),
						container.SplitPercent(33),
					),
				),
				container.Bottom(
					container.Border(linestyle.Light),
					container.BorderTitle("Results"),
					container.BorderTitleAlignCenter(),
					container.PlaceWidget(widgets.resultsWidget),
				),
				container.SplitOption(container.SplitFixed(3)),
			),
		}
	} else {
		mainWidgets = []container.Option{
			container.Border(linestyle.Light),
			container.BorderTitle("Results"),
			container.BorderTitleAlignCenter(),
			container.PlaceWidget(widgets.resultsWidget),
		}
	}

	if widgets.helpWidget != nil && widgets.logsWidget != nil {
		// All widgets enabled.
		widgetContainer, err = container.New(
			appTermdash,
			container.PaddingBottom(displayConfig.OuterPaddingBottom),
			container.PaddingLeft(displayConfig.OuterPaddingLeft),
			container.PaddingTop(displayConfig.OuterPaddingTop),
			container.PaddingRight(displayConfig.OuterPaddingRight),
			container.SplitHorizontal(
				container.Top(mainWidgets...),
				container.Bottom(
					container.SplitHorizontal(
						container.Top(
							container.Border(linestyle.Light),
							container.BorderTitle("Help"),
							container.BorderTitleAlignCenter(),
							container.PlaceWidget(widgets.helpWidget),
						),
						container.Bottom(
							container.Border(linestyle.Light),
							container.BorderTitle("Logs"),
							container.BorderTitleAlignCenter(),
							container.PlaceWidget(widgets.logsWidget),
						),
						container.SplitOption(container.SplitFixed(3)),
					),
				),
				// XXX The +5 is to try to match tview's proportions.
				container.SplitOption(container.SplitPercent(displayConfig.ResultsSize+5)),
			),
		)
	} else if widgets.helpWidget != nil {
		// We have just the help widget enabled.
		widgetContainer, err = container.New(
			appTermdash,
			container.PaddingBottom(displayConfig.OuterPaddingBottom),
			container.PaddingLeft(displayConfig.OuterPaddingLeft),
			container.PaddingTop(displayConfig.OuterPaddingTop),
			container.PaddingRight(displayConfig.OuterPaddingRight),
			container.SplitHorizontal(
				container.Top(mainWidgets...),
				container.Bottom(
					container.Border(linestyle.Light),
					container.BorderTitle("Help"),
					container.BorderTitleAlignCenter(),
					container.PlaceWidget(widgets.helpWidget),
				),
				container.SplitOption(container.SplitFixedFromEnd(3)),
			),
		)
	} else if widgets.logsWidget != nil {
		// We have just the logs widget enabled.
		widgetContainer, err = container.New(
			appTermdash,
			container.PaddingBottom(displayConfig.OuterPaddingBottom),
			container.PaddingLeft(displayConfig.OuterPaddingLeft),
			container.PaddingTop(displayConfig.OuterPaddingTop),
			container.PaddingRight(displayConfig.OuterPaddingRight),
			container.SplitHorizontal(
				container.Top(mainWidgets...),
				container.Bottom(
					container.Border(linestyle.Light),
					container.BorderTitle("Logs"),
					container.BorderTitleAlignCenter(),
					container.PlaceWidget(widgets.logsWidget),
				),
				// XXX The -1 is to try to match tview's proportions.
				container.SplitOption(
					container.SplitPercent(displayConfig.ResultsSize+displayConfig.HelpSize-1)),
			),
		)
	} else {
		// Just the results pane.
		widgetContainer, err = container.New(
			appTermdash,
			container.ID("main"),
			container.MarginBottom(displayConfig.OuterPaddingBottom),
			container.MarginLeft(displayConfig.OuterPaddingLeft),
			container.MarginTop(displayConfig.OuterPaddingTop),
			container.MarginRight(displayConfig.OuterPaddingRight),
		)
		widgetContainer.Update("main", mainWidgets...)
	}
	e(err)

	if widgets.logsWidget != nil {
		// Define a logging sink for Termdash apps.
		logsWidgetWriter = termdashTextWriter{text: widgets.logsWidget}
		logsWidgetHandler = slog.NewTextHandler(
			&logsWidgetWriter,
			&slog.HandlerOptions{Level: config.SlogLogLevel()},
		)
		if config.LogMulti {
			// We need to preserve the existing log stream.
			slog.SetDefault(slog.New(slogmulti.Fanout(slog.Default().Handler(), logsWidgetHandler)))
		} else {
			// We should only log to the widget.
			slog.SetDefault(slog.New(logsWidgetHandler))
		}
	}

	// Initialize the top-line status widgets.
	widgets.queryWidget.Write(query)
	widgets.filterWidget.Write(fmt.Sprintf("%v", filters))
	widgets.labelWidget.Write(fmt.Sprintf("%v", labels))

	// Run the display.
	termdash.Run(
		ctx,
		appTermdash,
		widgetContainer,
		termdash.ErrorHandler(errorTermdashHandler),
		termdash.KeyboardSubscriber(keyboardTermdashHandler),
	)
}
