package issues

import (
	"fmt"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// The issue "A hover for a motion event that did not move the pointer (Linux)" of Theatro's list,
// on 1.17.11, fixed in step 58.1 (Glutter 1.17.12): the Wayland backend made a hover of every
// wl_pointer.motion, also of one at the place the pointer already had. GNOME's compositor sends
// such motions while the window under a pointer that does not move redraws, so a widget that
// redraws all the time (a video) got a stream of hovers with a delta of 0 while the mouse did not
// move, and Theatro's player never hid its bars and its pointer. Flutter's engine drops a hover
// whose place did not change (PointerDataPacketConverter) for every native platform; Glutter's
// desktops make their pointer data themselves, and Windows (WM_MOUSEMOVE, which Windows also sends
// when a window under the pointer changes), macOS and the web had the same fault.
//
// The reproduction moves the mouse into a view and then to a list of places (StillHoverSteps),
// most of them where the mouse already is, through the platform's own path (the compositor, the
// X server, window messages, the system's input, DOM events), and reports the hovers that reached
// the framework.

// StillHoverName is the name of the issue: what an app of a device test is asked for, and the
// prefix of the lines of its report.
const StillHoverName = "still-hover"

// StillHoverPoint is a place of the mouse in the view, in the pixels of the platform's pointer
// data: physical pixels on the desktops and on Android, the view's logical pixels in a browser.
type StillHoverPoint struct {
	X, Y float64
}

// StillHoverSteps returns the places the reproduction moves the mouse to, one after the other. The
// runner first brings the mouse into the view at the first place (an enter, or the first motion of
// a platform that has no enter) and then sends a motion to each place, the first included: four at
// the first place, three at a second one and three at a third one, a pixel from the second.
func StillHoverSteps() []StillHoverPoint {
	return []StillHoverPoint{
		{40, 30}, {40, 30}, {40, 30}, {40, 30},
		{90, 60}, {90, 60}, {90, 60},
		{91, 60}, {91, 60}, {91, 60},
	}
}

// StillHoverEvent is a hover of the mouse that reached the framework: its place and its delta, in
// the pixels of StillHoverPoint.
type StillHoverEvent struct {
	X, Y, DX, DY float64
}

// StillHoverReport is what came of the reproduction on one platform: the hovers of the mouse, in
// the order the framework got them.
type StillHoverReport struct {
	Hovers []StillHoverEvent
}

// StillHoverReportOf makes the report from the pointer data a platform gave the framework while
// the reproduction ran: its hovers of a mouse that are not signals. scale divides the places and
// the deltas (the device pixel ratio, for a runner whose steps are in logical pixels; 1 for one
// whose steps are physical).
func StillHoverReportOf(data []ui.PointerData, scale float64) StillHoverReport {
	var rep StillHoverReport
	for _, d := range data {
		if d.Kind != ui.PointerDeviceKindMouse || d.Change != ui.PointerChangeHover ||
			(d.SignalKind != ui.PointerSignalKindUnspecified && d.SignalKind != ui.PointerSignalKindNone) {
			continue
		}
		rep.Hovers = append(rep.Hovers, StillHoverEvent{
			X: d.PhysicalX / scale, Y: d.PhysicalY / scale, DX: d.PhysicalDeltaX / scale, DY: d.PhysicalDeltaY / scale,
		})
	}
	return rep
}

// StillHoverPointer is the mouse of a platform whose pointer path runs in the runner's process:
// Enter brings it into the view at a place, Motion reports it at a place (both through the
// platform's own path, as its compositor, server or system would), and Take returns the pointer
// data that the platform made for the framework since the last call.
type StillHoverPointer interface {
	Enter(x, y float64)
	Motion(x, y float64)
	Take() []ui.PointerData
}

// RunStillHover runs the reproduction on a mouse in the runner's process: the enter at the first
// place, a motion to each place of StillHoverSteps, and the report of what the platform made.
func RunStillHover(p StillHoverPointer, scale float64) StillHoverReport {
	steps := StillHoverSteps()
	p.Enter(steps[0].X, steps[0].Y)
	data := p.Take()
	for _, s := range steps {
		p.Motion(s.X, s.Y)
		data = append(data, p.Take()...)
	}
	return StillHoverReportOf(data, scale)
}

// Problems checks the report and returns what is wrong, one line for each finding; none when each
// place the mouse went to after the first got one hover, with the delta from the place before, and
// the mouse made no hover where it already was. The first place may have one hover, as the first
// event: a browser's first pointermove of a mouse makes an add and a hover at its place, as
// Flutter's web engine does, where the other platforms make only the add.
func (r StillHoverReport) Problems() []string {
	if len(r.Hovers) == 0 {
		return []string{"no hover of the mouse reached the framework"}
	}
	var places []StillHoverPoint
	for _, s := range StillHoverSteps() {
		if len(places) == 0 || places[len(places)-1] != s {
			places = append(places, s)
		}
	}
	var out []string
	count := make([]int, len(places))
	last := -1
	for i, h := range r.Hovers {
		at := -1
		for j, p := range places {
			if p.X == h.X && p.Y == h.Y {
				at = j
			}
		}
		if at < 0 {
			out = append(out, fmt.Sprintf("hover %d is at %g,%g, where the mouse never was", i, h.X, h.Y))
			continue
		}
		count[at]++
		if at < last {
			out = append(out, fmt.Sprintf("hover %d is at %g,%g, where the mouse was before the last hover", i, h.X, h.Y))
		}
		if at == 0 && i > 0 {
			out = append(out, fmt.Sprintf("hover %d is at %g,%g, where the mouse entered, after another hover", i, h.X, h.Y))
		}
		if i > 0 {
			prev := r.Hovers[i-1]
			if h.DX != h.X-prev.X || h.DY != h.Y-prev.Y {
				out = append(out, fmt.Sprintf("the hover to %g,%g has the delta %g,%g, want %g,%g", h.X, h.Y, h.DX, h.DY, h.X-prev.X, h.Y-prev.Y))
			}
		}
		last = at
	}
	if count[0] > 1 {
		out = append(out, fmt.Sprintf("%d hovers at %g,%g, where the mouse entered and did not move: want at most 1", count[0], places[0].X, places[0].Y))
	}
	for j := 1; j < len(places); j++ {
		if count[j] != 1 {
			out = append(out, fmt.Sprintf("%d hovers at %g,%g, want 1: the mouse went there once", count[j], places[j].X, places[j].Y))
		}
	}
	return out
}

// stillHoverLine is the format of a line of the report: its prefix says what the line is, so that
// the lines are found in the output of an app among others.
const stillHoverLine = StillHoverName + ": hover x %g y %g dx %g dy %g"

// Lines returns the report as text, one line for each hover, for an app on a device to write into
// a file or its output. A report with no hover has no line.
func (r StillHoverReport) Lines() []string {
	out := make([]string, 0, len(r.Hovers))
	for _, h := range r.Hovers {
		out = append(out, fmt.Sprintf(stillHoverLine, h.X, h.Y, h.DX, h.DY))
	}
	return out
}

// ParseStillHover reads a report from the lines of an app's file or output; lines that are not of
// the report are passed over. It fails with IssueReportInvalid for a line of the report that does
// not read.
func ParseStillHover(lines []string) (StillHoverReport, error) {
	var rep StillHoverReport
	for _, line := range lines {
		at := strings.Index(line, StillHoverName+": hover ")
		if at < 0 {
			continue
		}
		var h StillHoverEvent
		if _, err := fmt.Sscanf(line[at:], stillHoverLine, &h.X, &h.Y, &h.DX, &h.DY); err != nil {
			return StillHoverReport{}, errors.NewErr(TagIssueReportInvalid, err).SetMetadata("line", line)
		}
		rep.Hovers = append(rep.Hovers, h)
	}
	return rep, nil
}
