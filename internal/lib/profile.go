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
	}

	ProfileLabels = []string{
		"Process State",
		"Age (s)",
		"Threads",
		"CPU Usage (%)",
		"Resident Memory (GB)",
		"Virtual Memory (GB)",
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
		return 0, err
	}

	return float64(bytes) / float64(divisor), nil
}

// Executes a pprof on a specific process, isolating specific data.
func runProfile(pid int) (result string) {
	var (
		uptime float64 // Total system uptime.
	)

	proc, err := procfs.NewProc(pid)
	e(err)

	// Reads /proc/[pid]/stat.
	stat, err := proc.Stat()
	e(err)

	// Calculate CPU usage.
	startTime, err := stat.StartTime()
	e(err)
	procUptime, err := os.Open("/proc/uptime")
	e(err)
	defer procUptime.Close()
	scanner := bufio.NewScanner(procUptime)
	for scanner.Scan() {
		uptime, err = strconv.ParseFloat(strings.Split(scanner.Text(), " ")[0], 10)
	}

	// Calculate memory usage.
	rssM, err := byteConv(stat.ResidentMemory(), "gigabyte")
	virtM, err := byteConv(int(stat.VirtualMemory()), "gigabyte")

	result = fmt.Sprintf(
		"%s %d %d %f %f %f",
		processState[stat.State],
		time.Now().Unix()-int64(startTime),
		stat.NumThreads,
		(stat.CPUTime()/uptime)*100,
		rssM,
		virtM,
	)

	return
}
