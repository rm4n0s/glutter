//go:build linux

package linux

import (
	"bytes"
	"os"
	"strconv"
	"time"
)

// clockTicksPerSecond is what sysconf(_SC_CLK_TCK) gives on every Linux that Glutter
// supports; /proc counts times in these.
const clockTicksPerSecond = 100

// ProcessStart returns when this process started, to the tick of the kernel's clock
// (10 ms), from /proc: the start of a program is before its first Go code runs, and
// the cold start of SPEC §15 counts from it. It says false where /proc cannot be read.
//
// Flutter: no counterpart
func ProcessStart() (time.Time, bool) {
	stat, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return time.Time{}, false
	}
	now := time.Now()
	uptime, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return time.Time{}, false
	}
	return processStartFrom(stat, uptime, now)
}

// processStartFrom is ProcessStart from the contents of /proc/self/stat and /proc/uptime,
// read when the wall clock said now. The boot is the time of the clock minus the uptime;
// /proc/stat has it too, but only to the second.
func processStartFrom(stat, uptime []byte, now time.Time) (time.Time, bool) {
	// The command name, which is in parentheses and may hold spaces and parentheses,
	// ends at the last ")". Field 22, starttime, is the 20th after it.
	end := bytes.LastIndexByte(stat, ')')
	if end < 0 {
		return time.Time{}, false
	}
	fields := bytes.Fields(stat[end+1:])
	if len(fields) < 20 {
		return time.Time{}, false
	}
	ticks, err := strconv.ParseInt(string(fields[19]), 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	fields = bytes.Fields(uptime)
	if len(fields) < 1 {
		return time.Time{}, false
	}
	up, err := strconv.ParseFloat(string(fields[0]), 64)
	if err != nil || up < 0 {
		return time.Time{}, false
	}
	boot := now.Add(-time.Duration(up * float64(time.Second)))
	return boot.Add(time.Duration(ticks) * time.Second / clockTicksPerSecond), true
}
