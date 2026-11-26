package windows

import (
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/ui"
)

// stillHoverTracker runs the reproduction of internal/issues on the pointer tracker: Windows has no
// enter, so the enter is the first WM_MOUSEMOVE, and every motion is a WM_MOUSEMOVE.
type stillHoverTracker struct{ t *pointerTracker }

func (p stillHoverTracker) Enter(x, y float64)     { p.t.move(0, mouseAt(x, y, 0)) }
func (p stillHoverTracker) Motion(x, y float64)    { p.t.move(0, mouseAt(x, y, 0)) }
func (p stillHoverTracker) Take() []ui.PointerData { return p.t.takeForFramework() }

// TestPointerTrackerHoversOnlyWhenThePointerMoves is the issue "A hover for a motion event that did
// not move the pointer" (step 58.1) on what the Windows platform gives the framework: a WM_MOUSEMOVE
// at the place the pointer has makes no hover, and neither does the first one, at the place of the
// add.
func TestPointerTrackerHoversOnlyWhenThePointerMoves(t *testing.T) {
	tr := newPointerTracker(func() time.Duration { return 0 })
	rep := issues.RunStillHover(stillHoverTracker{tr}, 1)
	for _, p := range rep.Problems() {
		t.Error(p)
	}
	if len(rep.Hovers) != 2 {
		t.Errorf("%d hovers, want 2: one at each place after the first", len(rep.Hovers))
	}
	if t.Failed() {
		t.Logf("the report:\n%s", strings.Join(rep.Lines(), "\n"))
	}
	// A scroll signal where the pointer is, and the motion of a locked pointer, are not dropped.
	tr.scroll(0, ui.PointerDeviceKindMouse, 0, 91, 60, 0, 30)
	if got := tr.takeForFramework(); len(got) != 1 || got[0].SignalKind != ui.PointerSignalKindScroll {
		t.Errorf("the scroll where the pointer is = %+v, want the signal", got)
	}
}
