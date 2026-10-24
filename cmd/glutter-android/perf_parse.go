package main

import (
	"archive/zip"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/androidtool"
	"github.com/rm4n0s/glutter/internal/perfstats"
)

// The parsers of `glutter-android perf` (SPEC §24.13, step 26.5): each turns the text of a tool
// on the device (am, dumpsys, /proc, logcat) or a file into numbers. They are pure functions of
// their input, so that unit tests feed them the real formats of Android 8 and of Android 14.

// amStart is what `am start -W` prints once the activity is shown.
type amStart struct {
	// Status is "ok" and LaunchState "COLD", "WARM" or "HOT" (Android 10 and later; empty before).
	Status, LaunchState string
	// This, Total and Wait are ThisTime, TotalTime and WaitTime. ThisTime is only printed by
	// Android 9 and earlier.
	This, Total, Wait time.Duration
}

var amLine = regexp.MustCompile(`(?m)^\s*([A-Za-z]+):\s*(.*?)\s*$`)

// parseAmStart reads the output of `am start -W -n <activity>`. It fails with
// AndroidPerfStartRejected when the activity manager reports an error or a status other than
// ok, and with AndroidPerfAmStartParse when there is no TotalTime or WaitTime.
func parseAmStart(out string) (amStart, error) {
	var r amStart
	var haveTotal, haveWait bool
	for _, m := range amLine.FindAllStringSubmatch(out, -1) {
		key, val := m[1], m[2]
		ms := func() (time.Duration, bool) {
			n, err := strconv.Atoi(val)
			return time.Duration(n) * time.Millisecond, err == nil
		}
		switch key {
		case "Error":
			return amStart{}, errors.New(tagAndroidPerfStartRejected, "the activity manager did not start the app: "+val, "output", out)
		case "Status":
			r.Status = val
		case "LaunchState":
			r.LaunchState = val
		case "ThisTime":
			r.This, _ = ms()
		case "TotalTime":
			r.Total, haveTotal = ms()
		case "WaitTime":
			r.Wait, haveWait = ms()
		}
	}
	if r.Status != "" && r.Status != "ok" {
		return amStart{}, errors.New(tagAndroidPerfStartRejected, "am start -W says Status: "+r.Status, "output", out)
	}
	if !haveTotal || !haveWait {
		return amStart{}, errors.New(tagAndroidPerfAmStartParse, "the output of am start -W has no TotalTime and WaitTime", "output", out)
	}
	return r, nil
}

// clockTicksPerSecond is USER_HZ, the unit of utime and stime in /proc/<pid>/stat: 100 on
// Android (and on every Linux that matters).
const clockTicksPerSecond = 100

// parseProcStat returns utime+stime in clock ticks from the text of /proc/<pid>/stat. The
// second field, the command in parentheses, may hold blanks and parentheses, so the fields are
// counted from the last ')'. It fails with AndroidPerfProcStatParse.
func parseProcStat(stat string) (uint64, error) {
	end := strings.LastIndexByte(stat, ')')
	if end < 0 {
		return 0, errors.New(tagAndroidPerfProcStatParse, "/proc/<pid>/stat has no ')' after the command", "stat", stat)
	}
	f := strings.Fields(stat[end+1:])
	// f[0] is the state (field 3); utime and stime are fields 14 and 15.
	if len(f) < 13 {
		return 0, errors.New(tagAndroidPerfProcStatParse, "/proc/<pid>/stat has too few fields", "stat", stat)
	}
	utime, errU := strconv.ParseUint(f[11], 10, 64)
	stime, errS := strconv.ParseUint(f[12], 10, 64)
	if errU != nil || errS != nil {
		return 0, errors.New(tagAndroidPerfProcStatParse, "utime and stime of /proc/<pid>/stat are not numbers", "utime", f[11], "stime", f[12])
	}
	return utime + stime, nil
}

// procStatus is the resident set of a process from /proc/<pid>/status, in kilobytes.
type procStatus struct{ VmRSSKB, VmHWMKB int }

var kbLine = regexp.MustCompile(`(?m)^(VmRSS|VmHWM):\s*(\d+)\s*kB`)

// parseProcStatus reads VmRSS and VmHWM from the text of /proc/<pid>/status. It fails with
// AndroidPerfProcStatusParse when VmRSS is missing (a process that ended).
func parseProcStatus(status string) (procStatus, error) {
	var r procStatus
	var haveRSS bool
	for _, m := range kbLine.FindAllStringSubmatch(status, -1) {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		if m[1] == "VmRSS" {
			r.VmRSSKB, haveRSS = n, true
		} else {
			r.VmHWMKB = n
		}
	}
	if !haveRSS {
		return procStatus{}, errors.New(tagAndroidPerfProcStatusParse, "/proc/<pid>/status has no VmRSS: the process has ended", "status", status)
	}
	return r, nil
}

// meminfo is the total of `dumpsys meminfo <package>` in kilobytes.
type meminfo struct {
	// PSSKB is TOTAL PSS. RSSKB is TOTAL RSS, which Android 10 and later print; HasRSS says it was
	// there.
	PSSKB, RSSKB int
	HasRSS       bool
}

var (
	totalPSS      = regexp.MustCompile(`TOTAL PSS:\s*(\d+)`)
	totalRSS      = regexp.MustCompile(`TOTAL RSS:\s*(\d+)`)
	totalSummary  = regexp.MustCompile(`(?m)^\s*TOTAL:\s*(\d+)`)
	totalTableRow = regexp.MustCompile(`(?m)^\s*TOTAL((?:\s+\d+)+)\s*$`)
)

// parseMeminfo reads TOTAL PSS and TOTAL RSS from the output of `dumpsys meminfo <package>`.
// Android 14 and 15 print "TOTAL PSS: n TOTAL RSS: n TOTAL SWAP PSS: n" under the App Summary.
// Android 9 to 13 print "TOTAL: n TOTAL SWAP PSS: n" there and a table row "TOTAL  pss ..."
// (with an Rss column from Android 10, which the header names); Android 8 has the same without
// the Rss column. It fails with AndroidPerfMeminfoParse when there is no total (the process is
// not running, or the format is new).
func parseMeminfo(out string) (meminfo, error) {
	var r meminfo
	var row []string
	if m := totalTableRow.FindStringSubmatch(out); m != nil {
		row = strings.Fields(m[1])
	}
	switch {
	case totalPSS.MatchString(out):
		r.PSSKB, _ = strconv.Atoi(totalPSS.FindStringSubmatch(out)[1])
	case len(row) > 0:
		r.PSSKB, _ = strconv.Atoi(row[0])
	case totalSummary.MatchString(out):
		r.PSSKB, _ = strconv.Atoi(totalSummary.FindStringSubmatch(out)[1])
	default:
		return meminfo{}, errors.New(tagAndroidPerfMeminfoParse, "dumpsys meminfo has no TOTAL", "output", out)
	}
	if m := totalRSS.FindStringSubmatch(out); m != nil {
		r.RSSKB, r.HasRSS = atoiOK(m[1])
	} else if idx := rssColumn(out); idx >= 0 && idx < len(row) {
		r.RSSKB, r.HasRSS = atoiOK(row[idx])
	}
	return r, nil
}

func atoiOK(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	return n, err == nil
}

// rssColumn is the index of the Rss column of the table of dumpsys meminfo, counted from its
// Pss column, or -1 when the table has none (Android 9 and earlier).
func rssColumn(out string) int {
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) > 1 && f[0] == "Pss" {
			for i, name := range f {
				if name == "Rss" {
					return i
				}
			}
			return -1
		}
	}
	return -1
}

// firstFrame is the engine's "first frame presented" line.
type firstFrame struct {
	// SinceRun is the time from the start of the engine's Run to the first frame; SinceProcess
	// from the start of the process, when the engine knows it (zero when not).
	SinceRun, SinceProcess time.Duration
	// Renderer is the renderer and device, "vulkan: Adreno (TM) 740".
	Renderer string
}

// parseFirstFrame finds the first "first frame presented" line of a log and reads it; ok is
// false when the log has none (yet) or its since_run is not a duration.
func parseFirstFrame(log string) (ff firstFrame, ok bool) {
	for _, line := range strings.Split(log, "\n") {
		if !strings.Contains(line, "first frame presented") {
			continue
		}
		kv := perfstats.ParseLogLine(line)
		run, err := time.ParseDuration(kv["since_run"])
		if err != nil {
			continue
		}
		ff = firstFrame{SinceRun: run, Renderer: kv["renderer"]}
		if d, err := time.ParseDuration(kv["since_process_start"]); err == nil {
			ff.SinceProcess = d
		}
		return ff, true
	}
	return firstFrame{}, false
}

// parseFrameLines reads every "frames" statistics line of a log, in order. Lines with a value
// that is not a number are skipped.
func parseFrameLines(log string) []perfstats.Frames {
	var out []perfstats.Frames
	for _, line := range strings.Split(log, "\n") {
		if !perfstats.IsFramesLine(line) {
			continue
		}
		if f, ok := perfstats.ParseFramesLine(line); ok {
			out = append(out, f)
		}
	}
	return out
}

// pickABI chooses the ABI to install: the first one of the device's list
// (ro.product.cpu.abilist, "x86_64,arm64-v8a,armeabi-v7a") that is among the ABIs that were
// built. It fails with AndroidPerfNoABI.
func pickABI(abilist string, built []androidtool.ABI) (androidtool.ABI, error) {
	for _, name := range strings.Split(strings.TrimSpace(abilist), ",") {
		for _, abi := range built {
			if abi.Name == strings.TrimSpace(name) {
				return abi, nil
			}
		}
	}
	var names []string
	for _, abi := range built {
		names = append(names, abi.Name)
	}
	return androidtool.ABI{}, errors.New(tagAndroidPerfNoABI, "the device runs none of the ABIs that were built: add the device's to -abi",
		"device", strings.TrimSpace(abilist), "built", strings.Join(names, ","))
}

// parseABIs turns the value of -abi ("arm64,amd64", or the names arm64-v8a and x86_64) into ABIs.
// It fails with AndroidPerfBadABI.
func parseABIs(list string) ([]androidtool.ABI, error) {
	var out []androidtool.ABI
	for _, n := range strings.Split(list, ",") {
		if n = strings.TrimSpace(n); n == "" {
			continue
		}
		found := false
		for _, abi := range androidtool.ABIs {
			if abi.Short == n || abi.Name == n {
				out = append(out, abi)
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New(tagAndroidPerfBadABI, "unknown ABI "+n+": use arm64 or amd64", "abi", n)
		}
	}
	if len(out) == 0 {
		return nil, errors.New(tagAndroidPerfBadABI, "-abi names no ABI: use arm64 or amd64")
	}
	return out, nil
}

// apkInfo is what the zip of an APK says about its size.
type apkInfo struct {
	// LibBytes is the size of lib/<abi>/libapp.so by the name of the ABI, as stored in the APK
	// (the libraries are not compressed, so this is also its size in the file).
	LibBytes map[string]int64
	// Entries is the number of files in the APK.
	Entries int
}

// readAPK reads the central directory of the APK at path. It fails with AndroidPerfAPKRead.
func readAPK(path string) (apkInfo, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return apkInfo{}, errors.NewErr(tagAndroidPerfAPKRead, err).SetMetadata("path", path)
	}
	defer func() { _ = zr.Close() }() // read only: nothing is lost when closing fails
	info := apkInfo{LibBytes: map[string]int64{}, Entries: len(zr.File)}
	for _, f := range zr.File {
		parts := strings.Split(f.Name, "/")
		if len(parts) == 3 && parts[0] == "lib" && parts[2] == "libapp.so" {
			info.LibBytes[parts[1]] = int64(f.UncompressedSize64)
		}
	}
	return info, nil
}
