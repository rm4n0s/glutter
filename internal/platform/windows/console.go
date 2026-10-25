package windows

import (
	"bufio"
	"io"
	"strings"
)

// consoleAction is what the platform does about the output of a GUI program (SPEC §25.1).
type consoleAction int

// The actions of consolePlan.
const (
	// consoleKeep: the standard handles are valid (a terminal, a pipe or a file), so the output goes
	// where the starter put it.
	consoleKeep consoleAction = iota
	// consoleAttach: the handles are not valid but the parent has a console: write there.
	consoleAttach
	// consoleDebugger: there is no console either; the output goes to OutputDebugStringW, where a
	// debugger or DebugView shows it.
	consoleDebugger
)

// consolePlan decides what to do about standard output and error. A program built with
// -H=windowsgui has no standard handles unless it was started with redirection, so it attaches to
// its parent's console when there is one and logs to the debugger when there is not.
//
// Flutter: no counterpart
func consolePlan(handlesValid bool, attachParent func() bool) consoleAction {
	if handlesValid {
		return consoleKeep
	}
	if attachParent() {
		return consoleAttach
	}
	return consoleDebugger
}

// validHandle says whether a standard handle can be written to: neither NULL (a GUI program that
// was started without redirection) nor INVALID_HANDLE_VALUE.
func validHandle(h uintptr) bool { return h != 0 && h != ^uintptr(0) }

// forwardLines reads r to its end and passes each line to out, without its line break; a last
// line with no break counts. It is how what a program writes reaches OutputDebugStringW, which
// takes whole lines.
func forwardLines(r io.Reader, out func(line string)) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 4096), 1<<20)
	for sc.Scan() {
		out(strings.TrimRight(sc.Text(), "\r"))
	}
	if sc.Err() != nil {
		// A line too long for the buffer: drop the rest, so that the writer never blocks on a full pipe.
		_, _ = io.Copy(io.Discard, r)
	}
}
