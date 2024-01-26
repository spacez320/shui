//
// Logic for 'profile' mode.

package lib

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/procfs"
)

var (
	processState = map[string]string{
		"D": "uninterruptable sleep",
		"I": "idle",
		"R": "running",
		"S": "sleeping",
		"T": "stopped via signal",
		"t": "stopped via debugger",
		"Z": "zombie",
	} // Map of show to long process states.

	ProfileLabels = []string{
		"State",
		"Age (s)",
		"Threads",
		"CPU Usage (%)",
		"Resident Memory (GB)",
		"Virtual Memory (GB)",
		"Swap (GB)",
		"IO Read (MB)",
		"IO Write (MB)",
	} // Labels supplied for profile results.
)

// Converts a byte count (commonly given by /proc) to some higher delinitation.
func byteConv(bytes int, level string) (convBytes float64, err error) {
	var (
		divisor int // Divider for conversion.
	)

	switch level {
	case "gigabyte":
		divisor = 1000 * 1000 * 1000
	case "megabyte":
		divisor = 1000 * 1000
	case "kilobyte":
		divisor = 1000
	default:
		err = fmt.Errorf("Bad byte conversion: %s", level)
	}

	if divisor > 0 {
		convBytes = float64(bytes) / float64(divisor)
	}

	return
}

// Executes a pprof on a specific process, isolating specific data.
func runProfile(pid int) string {
	var (
		uptime float64 // Total system uptime.
	)

	// Read /proc/[pid] data.
	proc, err := procfs.NewProc(pid)
	e(err)
	procSmap, err := proc.ProcSMapsRollup() // Reads /proc/[pid]/smaps_rollup.
	e(err)
	procIO, err := proc.IO() // Reads /proc/[pid]/io.
	e(err)
	procStat, err := proc.Stat() // Read /proc/[pid]/stat.
	e(err)

	// Calculate CPU usage.
	startTime, err := procStat.StartTime()
	e(err)
	procUptime, err := os.Open("/proc/uptime")
	e(err)
	defer procUptime.Close()
	scanner := bufio.NewScanner(procUptime)
	for scanner.Scan() {
		uptime, err = strconv.ParseFloat(strings.Split(scanner.Text(), " ")[0], 10)
	}

	// Calculate IO usage.
	read, err := byteConv(int(procIO.ReadBytes), "megabyte")
	e(err)
	write, err := byteConv(int(procIO.WriteBytes), "megabyte")
	e(err)

	// Calculate memory usage.
	rss, err := byteConv(procStat.ResidentMemory(), "gigabyte")
	virt, err := byteConv(int(procStat.VirtualMemory()), "gigabyte")
	swap, err := byteConv(int(procSmap.Swap), "gigabyte")

	return fmt.Sprintf(
		"%s %d %d %f %f %f %f %f %f",
		processState[procStat.State],
		time.Now().Unix()-int64(startTime),
		procStat.NumThreads,
		(procStat.CPUTime()/uptime)*100,
		rss,
		virt,
		swap,
		read,
		write,
	)
}
