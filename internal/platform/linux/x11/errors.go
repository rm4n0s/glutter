package x11

import (
	"log/slog"
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// Xlib calls one error handler for the whole process when the server reports an
// error for a request (a window that is already gone, a bad match, ...). It gets
// no user data, so the platform of a display is found by the address of the
// display, which is in the event. There is one C callback, made on first use
// (SPEC §8.1 rule 2).
var errorTargets = struct {
	sync.Mutex
	byDisplay map[uintptr]*Platform
}{byDisplay: map[uintptr]*Platform{}}

// errorCallback is the C function pointer of the error handler.
var errorCallback = sync.OnceValue(func() uintptr {
	return dl.Callback(func(display, event uintptr) uintptr {
		ev := view[xErrorEvent](event)
		errorTargets.Lock()
		p := errorTargets.byDisplay[display]
		errorTargets.Unlock()
		if p != nil {
			p.onXError(*ev)
		}
		return 0
	})
})

// registerErrorHandler makes p receive the X errors of its display.
func registerErrorHandler(p *Platform) {
	errorTargets.Lock()
	errorTargets.byDisplay[p.display] = p
	errorTargets.Unlock()
	p.x.setErrorHandler(errorCallback())
}

// unregisterErrorHandler stops the X errors of p's display from reaching it.
func unregisterErrorHandler(p *Platform) {
	errorTargets.Lock()
	delete(errorTargets.byDisplay, p.display)
	errorTargets.Unlock()
}

// onXError records an error the server reported. Errors arrive in the middle of
// Xlib calls and cannot be returned, and most of them are harmless (a request for
// a window that was just closed), so they are logged and counted.
func (p *Platform) onXError(ev xErrorEvent) {
	p.xErrors++
	slog.Warn("X error", "code", ev.errorCode, "request", ev.requestCode, "minor", ev.minorCode, "resource", ev.resourceID)
}
