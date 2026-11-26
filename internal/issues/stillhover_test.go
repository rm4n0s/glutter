package issues

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// goodStillHoverReport is the report of a platform that has the fix: a hover to each place after
// the first.
func goodStillHoverReport() StillHoverReport {
	return StillHoverReport{Hovers: []StillHoverEvent{{X: 90, Y: 60, DX: 50, DY: 30}, {X: 91, Y: 60, DX: 1}}}
}

func TestStillHoverProblems(t *testing.T) {
	tests := []struct {
		name   string
		report StillHoverReport
		want   []string // a part of each problem, in order; none for a good report
	}{
		{"the fixed platform", goodStillHoverReport(), nil},
		{"a browser's first pointermove: a hover at the place of the enter", StillHoverReport{Hovers: []StillHoverEvent{
			{X: 40, Y: 30}, {X: 90, Y: 60, DX: 50, DY: 30}, {X: 91, Y: 60, DX: 1},
		}}, nil},
		{"no hover", StillHoverReport{}, []string{"no hover of the mouse reached the framework"}},
		{"the issue: a hover for each motion", StillHoverReport{Hovers: []StillHoverEvent{
			{X: 40, Y: 30}, {X: 40, Y: 30}, {X: 40, Y: 30}, {X: 40, Y: 30},
			{X: 90, Y: 60, DX: 50, DY: 30}, {X: 90, Y: 60}, {X: 90, Y: 60},
			{X: 91, Y: 60, DX: 1}, {X: 91, Y: 60}, {X: 91, Y: 60},
		}}, []string{
			"hover 1 is at 40,30, where the mouse entered, after another hover",
			"hover 2 is at 40,30, where the mouse entered",
			"hover 3 is at 40,30, where the mouse entered",
			"4 hovers at 40,30",
			"3 hovers at 90,60, want 1",
			"3 hovers at 91,60, want 1",
		}},
		{"a move that did not reach the framework", StillHoverReport{Hovers: []StillHoverEvent{{X: 90, Y: 60, DX: 50, DY: 30}}},
			[]string{"0 hovers at 91,60, want 1"}},
		{"a hover where the mouse never was", StillHoverReport{Hovers: []StillHoverEvent{
			{X: 90, Y: 60, DX: 50, DY: 30}, {X: 12, Y: 7, DX: -78, DY: -53}, {X: 91, Y: 60, DX: 79, DY: 53},
		}}, []string{"hover 1 is at 12,7, where the mouse never was"}},
		{"a wrong delta", StillHoverReport{Hovers: []StillHoverEvent{{X: 90, Y: 60, DX: 50, DY: 30}, {X: 91, Y: 60}}},
			[]string{"the hover to 91,60 has the delta 0,0, want 1,0"}},
		{"the places out of order", StillHoverReport{Hovers: []StillHoverEvent{{X: 91, Y: 60}, {X: 90, Y: 60, DX: -1}}},
			[]string{"hover 1 is at 90,60, where the mouse was before the last hover"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.report.Problems()
			if len(got) != len(tt.want) {
				t.Fatalf("problems = %q, want %d like %q", got, len(tt.want), tt.want)
			}
			for i := range got {
				if !strings.Contains(got[i], tt.want[i]) {
					t.Errorf("problem %d = %q, want a part %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestStillHoverReportIsReadBackFromItsLines(t *testing.T) {
	rep := StillHoverReport{Hovers: []StillHoverEvent{{X: 40, Y: 30}, {X: 90.5, Y: 60, DX: 50.5, DY: 30}, {X: 91, Y: 60, DX: 0.5, DY: -0.25}}}
	lines := append([]string{"I/Glutter: something else", "app: " + StillHoverName + " starts"}, rep.Lines()...)
	got, err := ParseStillHover(lines)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, rep) {
		t.Errorf("read back %+v, want %+v", got, rep)
	}
	if empty, err := ParseStillHover(StillHoverReport{}.Lines()); err != nil || len(empty.Hovers) != 0 {
		t.Errorf("an empty report read back as %+v, %v", empty, err)
	}
}

func TestParseStillHoverRejectsALineThatDoesNotRead(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"cut short", "still-hover: hover x 40 y 30"},
		{"a number that is not one", "still-hover: hover x forty y 30 dx 0 dy 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseStillHover([]string{tt.line})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("ParseStillHover." + TagIssueReportInvalid) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if appErr.Metadata["line"] != tt.line {
				t.Errorf("metadata: %v", appErr.Metadata)
			}
		})
	}
}

// echoPointer is a platform that makes a hover of every motion, as the Wayland backend did before
// step 58.1, or (dropStill) only of a motion to another place.
type echoPointer struct {
	dropStill bool
	x, y      float64
	out       []ui.PointerData
}

func (p *echoPointer) Enter(x, y float64) {
	p.x, p.y = x, y
	p.out = append(p.out, ui.PointerData{Change: ui.PointerChangeAdd, Kind: ui.PointerDeviceKindMouse, PhysicalX: 2 * x, PhysicalY: 2 * y})
}

func (p *echoPointer) Motion(x, y float64) {
	if p.dropStill && x == p.x && y == p.y {
		return
	}
	p.out = append(p.out, ui.PointerData{
		Change: ui.PointerChangeHover, Kind: ui.PointerDeviceKindMouse,
		PhysicalX: 2 * x, PhysicalY: 2 * y, PhysicalDeltaX: 2 * (x - p.x), PhysicalDeltaY: 2 * (y - p.y),
	})
	// A scroll signal and a touch are not hovers of the mouse.
	p.out = append(p.out,
		ui.PointerData{Change: ui.PointerChangeHover, Kind: ui.PointerDeviceKindMouse, SignalKind: ui.PointerSignalKindScroll, PhysicalX: 2 * x, PhysicalY: 2 * y},
		ui.PointerData{Change: ui.PointerChangeHover, Kind: ui.PointerDeviceKindTouch, PhysicalX: 2 * x, PhysicalY: 2 * y})
	p.x, p.y = x, y
}

func (p *echoPointer) Take() []ui.PointerData {
	out := p.out
	p.out = nil
	return out
}

func TestRunStillHover(t *testing.T) {
	tests := []struct {
		name      string
		dropStill bool
		hovers    int
		problems  bool
	}{
		{"a platform that drops a motion where the pointer is", true, 2, false},
		{"a platform with the issue", false, 10, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep := RunStillHover(&echoPointer{dropStill: tt.dropStill}, 2)
			if len(rep.Hovers) != tt.hovers {
				t.Errorf("%d hovers, want %d: %+v", len(rep.Hovers), tt.hovers, rep.Hovers)
			}
			if got := rep.Problems(); (len(got) != 0) != tt.problems {
				t.Errorf("problems = %q", got)
			}
		})
	}
}
