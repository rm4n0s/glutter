//go:build linux && !android && glutter_display

package x11

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/ui"
)

// TestDisplayHoversOnlyWhenThePointerMoves is the issue "A hover for a motion event that did not move
// the pointer" (step 58.1) on a real X server: xdotool moves the pointer through the XTest extension
// into the window and to each place of the reproduction, most of them where the pointer already is,
// and the framework gets a hover only where the pointer went. The window is at the corner of the
// screen (no window manager), so its places are those of the screen.
func TestDisplayHoversOnlyWhenThePointerMoves(t *testing.T) {
	e := startDisplay(t, windowConfig())
	e.waitMapped()
	// The pointer starts outside the window, so that the first place is where it enters.
	xdotool(t, "mousemove", "600", "500")
	time.Sleep(100 * time.Millisecond)
	_, _, _, before, _ := e.h.snapshot()
	n := len(before)
	steps := issues.StillHoverSteps()
	at := func(p issues.StillHoverPoint) {
		xdotool(t, "mousemove", strconv.Itoa(int(p.X)), strconv.Itoa(int(p.Y)))
		time.Sleep(30 * time.Millisecond)
	}
	at(steps[0])
	for _, s := range steps {
		at(s)
	}
	last := steps[len(steps)-1]
	waitFor(t, func() bool {
		_, _, _, b, _ := e.h.snapshot()
		for _, batch := range b[n:] {
			for _, d := range batch {
				if d.PhysicalX == last.X && d.PhysicalY == last.Y {
					return true
				}
			}
		}
		return false
	}, "the pointer never reached the last place of the reproduction")
	time.Sleep(200 * time.Millisecond) // anything the server still sends
	_, _, _, b, _ := e.h.snapshot()
	var data []ui.PointerData
	for _, batch := range b[n:] {
		data = append(data, batch...)
	}
	rep := issues.StillHoverReportOf(data, 1)
	for _, p := range rep.Problems() {
		t.Error(p)
	}
	if t.Failed() {
		t.Logf("the report:\n%s", strings.Join(rep.Lines(), "\n"))
	}
}
