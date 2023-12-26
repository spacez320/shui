module cryptarch

go 1.20

require golang.org/x/exp v0.0.0-20231127185646-65229373498e

require internal/lib v0.0.0

require pkg/storage v0.0.0 // indirect

require (
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/gdamore/tcell/v2 v2.6.1-0.20231203215052-2917c3801e73 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mum4k/termdash v0.18.0 // indirect
	github.com/rivo/tview v0.0.0-20231206124440-5f078138442e // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/term v0.9.0 // indirect
	golang.org/x/text v0.12.0 // indirect
)

replace internal/lib => ./internal/lib

replace pkg/storage => ./pkg/storage
