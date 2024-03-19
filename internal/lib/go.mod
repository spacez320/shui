module lib

go 1.20

require (
	github.com/rivo/tview v0.0.0-20231206124440-5f078138442e
	golang.org/x/exp v0.0.0-20231226003508-02704c960a9b
)

require (
	github.com/gdamore/tcell/v2 v2.7.4
	github.com/mum4k/termdash v0.20.0
	github.com/prometheus/procfs v0.12.0
	pkg/storage v0.0.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/prometheus/client_golang v1.18.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/term v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

replace pkg/storage => ../../pkg/storage
