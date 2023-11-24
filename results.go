//
// Results management.

package main

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
	"time"
	"unicode"

	"github.com/rivo/tview"
)

var (
	app         *tview.Application
	resultsView *tview.TextView
	results     Results // Stored results.
)

func init() {
	app = tview.NewApplication()

	resultsView = tview.NewTextView().
		SetChangedFunc(func() {
			app.Draw()
		})
}

// Adds a result to the result store.
//
// TODO In the future, multiple result stores could be implemented by making
// this a function of an interface.
func AddResult(result string) {
	results.PutC(TokenizeResult(result)...)
}

// Parses a result into tokens for compound storage.
//
// TODO In the future, multiple result stores could be implemented by making
// this a function of an interface.
func TokenizeResult(result string) (parsedResult []interface{}) {
	var (
		s    scanner.Scanner // Scanner for tokenization.
		next string          // Next token to consider.
	)

	s.Init(strings.NewReader(result))
	s.IsIdentRune = func(r rune, i int) bool {
		// Separate all tokens exclusively by whitespace.
		return !unicode.IsSpace(r)
	}

	for token := s.Scan(); token != scanner.EOF; token = s.Scan() {
		next = s.TokenText()

		// Attempt to parse this value as an integer.
		nextInt, err := strconv.ParseInt(next, 10, 0)
		if err == nil {
			parsedResult = append(parsedResult, nextInt)
			continue
		}

		// Attempt to parse this value as a float.
		nextFloat, err := strconv.ParseFloat(next, 10)
		if err == nil {
			parsedResult = append(parsedResult, nextFloat)
			continue
		}

		// Everything else has failed--just pass it as a string.
		parsedResult = append(parsedResult, next)
	}

	return
}

// Presents raw output.
func RawResults() {
	go func() {
		for i := 0; i < 10; i++ {
			fmt.Fprintf(resultsView, "The next number is: %d\n", i)
			time.Sleep(1 * time.Second)
		}
	}()

	err := app.SetRoot(resultsView, true).SetFocus(resultsView).Run()
	e(err)
}
