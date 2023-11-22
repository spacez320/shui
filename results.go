//
// Results management.

package main

import (
	"fmt"
	"time"

	"github.com/rivo/tview"
)

// Presents raw output.
func resultsModeRaw() {
	app := tview.NewApplication()

	rawView := tview.NewTextView().
		SetChangedFunc(func() {
			app.Draw()
		})

	go func() {
		for i := 0; i < 10; i++ {
			fmt.Fprintf(rawView, "The next number is: %d\n", i)
			time.Sleep(1 * time.Second)
		}
	}()

	err := app.SetRoot(rawView, true).SetFocus(rawView).Run()
	e(err)
}
