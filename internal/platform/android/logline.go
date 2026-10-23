package android

import "strings"

// Priorities of logcat, the values of ANDROID_LOG_* in <android/log.h>.
const (
	logDebug = 3
	logInfo  = 4
	logWarn  = 5
	logError = 6
)

// linePriority is the logcat priority of a line the program printed: Warn and Error when the
// line has the level of an slog record ("level=WARN", "level=ERROR", `"level":"ERROR"`, or the
// default logger's "WARN msg"), Debug for DEBUG, and Info for everything else (SPEC §24.9).
func linePriority(line string) int {
	for _, p := range []struct {
		level string
		prio  int
	}{{"ERROR", logError}, {"WARN", logWarn}, {"DEBUG", logDebug}, {"INFO", logInfo}} {
		for _, form := range []string{"level=" + p.level, `"level":"` + p.level + `"`} {
			if strings.Contains(line, form) {
				return p.prio
			}
		}
		// The default logger prints "2006/01/02 15:04:05 LEVEL message".
		if f := strings.Fields(line); len(f) > 2 && f[2] == p.level && len(f[0]) == 10 && strings.Count(f[0], "/") == 2 {
			return p.prio
		}
		if strings.HasPrefix(line, p.level+" ") {
			return p.prio
		}
	}
	return logInfo
}

// splitLines cuts the text read from the pipe into lines: buf holds what was read so far, and
// the lines that are complete are returned with the rest.
func splitLines(buf string) (lines []string, rest string) {
	for {
		i := strings.IndexByte(buf, '\n')
		if i < 0 {
			return lines, buf
		}
		lines = append(lines, strings.TrimSuffix(buf[:i], "\r"))
		buf = buf[i+1:]
	}
}
