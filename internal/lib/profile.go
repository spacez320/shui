//
// Logic for 'profile' mode.

package lib

import (
	"fmt"

	"github.com/prometheus/procfs"
)

// Executes a pprof on a specific process, isolating specific data.
func runProfile(pid string) (result string) {
	p, err := procfs.Self()
	e(err)

	stat, err := p.Stat()
	e(err)

	result = fmt.Sprintf(
		"%s %d %d %d %d %d %d %d %d %d %d",
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
		stat.RSSLimit,
	)

	return
}
