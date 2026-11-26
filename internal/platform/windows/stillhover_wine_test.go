//go:build windows && glutter_wine

package windows

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// TestWineHoversOnlyWhenThePointerMoves is the issue "A hover for a motion event that did not move
// the pointer" (step 58.1) on Windows, under Wine: the pointer is moved into the window at the first
// place of the reproduction (SetCursorPos, as a user's mouse), and the window gets a WM_MOUSEMOVE for
// each place, most of them where the pointer already is, as Windows sends one when a window under a
// pointer that does not move changes. The framework gets a hover only where the pointer went.
func TestWineHoversOnlyWhenThePointerMoves(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	steps := issues.StillHoverSteps()
	moveTo := func(s issues.StillHoverPoint) func() {
		return func() {
			lparam := uintptr(uint16(int16(s.X))) | uintptr(uint16(int16(s.Y)))<<16
			win32.SendMessage(w.hwnd, win32.WMMouseMove, 0, lparam)
		}
	}
	run := []func(){func() { moveCursorTo(t, w, int32(steps[0].X), int32(steps[0].Y)) }}
	for _, s := range steps {
		run = append(run, moveTo(s))
	}
	run = append(run, func() {})
	pointerSteps(t, p, r, 40*time.Millisecond, run...)
	rep := issues.StillHoverReportOf(r.pointers, 1)
	for _, prob := range rep.Problems() {
		t.Error(prob)
	}
	if t.Failed() {
		t.Logf("the report:\n%s\nthe changes: %v", strings.Join(rep.Lines(), "\n"), r.changes())
	}
}
