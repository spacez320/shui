//
// Logic for 'profile' mode.

package lib

import (
	"fmt"

	"github.com/prometheus/procfs"
	"golang.org/x/exp/slog"
)

var (
	ProfileLabels = []string{
		"Process State",
		"Minor Faults",
		"Major Faults",
		"User Mode Time (clock ticks)",
		"Kernel Mode Time (clock ticks)",
		"Nice",
		"Threads",
		"Time Since Boot (clock ticks)",
		"Virtual Memory (B)",
		"Resident Set (pages)",
	} // Labels supplied for profile results.
)

// Executes a pprof on a specific process, isolating specific data.
func runProfile(pid int) (result string) {
	p, err := procfs.NewProc(pid)
	e(err)

	stat, err := p.Stat()
	e(err)

	slog.Debug(fmt.Sprintf("Procfs for PID '%s' stat is: %v", pid, stat))

	result = fmt.Sprintf(
		"%s %d %d %d %d %d %d %d %d %d",
		stat.State,
		stat.MinFlt,
		stat.MajFlt,
		stat.UTime,
		stat.STime,
		stat.Nice,
		stat.NumThreads,
		stat.Starttime,
		stat.VSize,
		stat.RSS,
	)

	return
}
