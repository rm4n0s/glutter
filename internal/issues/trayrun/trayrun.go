// Package trayrun runs the reproduction of the issue "The tray's availability" (internal/issues,
// TrayAvailableName; step 60.1) in an app: it asks desktop.TrayAvailable from a goroutine, as an app
// does, then places a SystemTray and waits for what the tray reports through the error reporter of
// the binding. The Linux test (package desktop, a real dbus-daemon), the app of the Wine and emulator
// tests (examples/dev/trayavailable) and the browser test's app (internal/platform/web/desktoptest)
// share it, so that each platform runs the same steps.
//
// It is plain Go with no OS tags and does not import testing: apps link it.
package trayrun

import (
	"context"
	"sync"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Runner is the reproduction in one app. Wrap goes around the app's content, and Report gets the
// errors of the binding's error reporter.
type Runner struct {
	reports chan error
	ready   chan struct{}

	mu    sync.Mutex
	state *slotState
}

// New makes a runner.
func New() *Runner {
	return &Runner{reports: make(chan error, 32), ready: make(chan struct{})}
}

// Report takes an error of the binding's error reporter; the app's ErrorReporter calls it. It does
// not block: an error over the buffer is dropped.
func (r *Runner) Report(err error) {
	select {
	case r.reports <- err:
	default:
	}
}

// Wrap returns a widget that builds child and, once Run asks for it, a SystemTray beside it (in a Stack, so that the
// tray's coming does not make the elements of child again).
func (r *Runner) Wrap(child widgets.Widget) widgets.Widget {
	return widgets.NewStack([]widgets.Widget{child, &slot{StatefulWidget: widgets.NewStatefulWidgetBase(), r: r}})
}

// Run runs the reproduction and returns its report: platform and want are the report's (what the
// runner calls the platform, and what its desktop is expected to answer). It waits for the app to
// be built, asks desktop.TrayAvailable, places the tray on the UI thread, and takes the first error
// reported in the wait after it (none: the tray was shown). It must not be called on the UI thread.
func (r *Runner) Run(ctx context.Context, platform, want string, wait time.Duration) issues.TrayAvailableReport {
	rep := issues.TrayAvailableReport{Platform: platform, Want: want}
	select {
	case <-r.ready:
	case <-ctx.Done():
		rep.Answer = issues.TrayAnswerOf(false, ctx.Err())
		return rep
	}
	r.mu.Lock()
	s := r.state
	r.mu.Unlock()
	available, err := desktop.TrayAvailable(ctx, s.Context())
	rep.Answer = issues.TrayAnswerOf(available, err)
	// What was reported before the tray is not the tray's.
	for drained := false; !drained; {
		select {
		case early := <-r.reports:
			_ = early // reported before the tray was asked for: another error of the app, not the tray's
		default:
			drained = true
		}
	}
	widgets.WidgetsBindingOf(s.Context()).Post(s.place)
	timer := time.NewTimer(wait)
	defer timer.Stop()
	var trayErr error
	select {
	case trayErr = <-r.reports:
	case <-timer.C:
	case <-ctx.Done():
	}
	rep.Tray = issues.TrayOutcomeOf(trayErr, errors.Is(trayErr, &errors.Error{Tag: desktop.TagTrayUnavailable}))
	return rep
}

type slot struct {
	widgets.StatefulWidget
	r *Runner
}

func (w *slot) CreateState() widgets.State { return &slotState{} }

type slotState struct {
	widgets.StateOf[*slot]
	placed bool
}

func (s *slotState) InitState() {
	r := s.Widget().r
	r.mu.Lock()
	first := r.state == nil
	r.state = s
	r.mu.Unlock()
	if first {
		close(r.ready)
	}
}

// place puts the tray in the tree; it runs on the UI thread.
func (s *slotState) place() {
	if s.Mounted() {
		s.SetState(func() { s.placed = true })
	}
}

func (s *slotState) Build(widgets.BuildContext) widgets.Widget {
	if !s.placed {
		return widgets.NewSizedBoxShrink(nil)
	}
	return desktop.NewSystemTray(trayIcon(), nil, desktop.SystemTrayOpts{Tooltip: issues.TrayAvailableName})
}

// trayIcon is a theme name and a picture of 16 pixels, which every platform's tray takes.
func trayIcon() ui.WindowIcon {
	const size = 16
	rgba := make([]byte, size*size*4)
	for i := 0; i < len(rgba); i += 4 {
		rgba[i], rgba[i+1], rgba[i+2], rgba[i+3] = 0x20, 0x80, 0xc0, 0xff
	}
	icon, err := desktop.TrayIconFromRGBA(size, rgba)
	if err != nil {
		// The length is right by construction; a theme name alone is still an icon.
		return ui.WindowIcon{Name: "dialog-information"}
	}
	icon.Name = "dialog-information"
	return icon
}
