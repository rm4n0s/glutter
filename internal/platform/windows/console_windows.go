//go:build windows

package windows

import (
	"log"
	"os"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// setupConsole gives a GUI program (built with -H=windowsgui) somewhere to write (SPEC §25.1):
// when its standard handles are not valid it attaches to the console of its parent, if there is
// one, and reopens standard output and error on it; with no parent console what it writes goes to
// OutputDebugStringW, where a debugger or DebugView shows it. A program that was started with its
// output redirected keeps it.
//
// Flutter: no counterpart
func (p *Platform) setupConsole() {
	valid := validHandle(win32.GetStdHandle(win32.StdOutputHandle)) && validHandle(win32.GetStdHandle(win32.StdErrorHandle))
	switch consolePlan(valid, func() bool { return win32.AttachConsole(win32.AttachParentProcess) }) {
	case consoleAttach:
		out, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
		if err != nil {
			return
		}
		os.Stdout, os.Stderr = out, out
		log.SetOutput(out)
	case consoleDebugger:
		r, w, err := os.Pipe()
		if err != nil {
			return
		}
		os.Stdout, os.Stderr = w, w
		log.SetOutput(w)
		go forwardLines(r, win32.OutputDebugString)
	}
}
