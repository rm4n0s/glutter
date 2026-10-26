package svg

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// pathString writes the segments of a path: M x,y L x,y Q c,e C c1,c2,e O (a conic: c,e,weight) Z.
func pathString(p *ui.Path) string {
	var parts []string
	f := func(v float64) string { return fmt.Sprintf("%.4g", v) }
	for _, s := range p.Segments() {
		switch s.Verb {
		case ui.PathVerbMove:
			parts = append(parts, "M"+f(s.P0.Dx)+","+f(s.P0.Dy))
		case ui.PathVerbLine:
			parts = append(parts, "L"+f(s.P1.Dx)+","+f(s.P1.Dy))
		case ui.PathVerbQuad:
			parts = append(parts, "Q"+f(s.P1.Dx)+","+f(s.P1.Dy)+","+f(s.P2.Dx)+","+f(s.P2.Dy))
		case ui.PathVerbCubic:
			parts = append(parts, "C"+f(s.P1.Dx)+","+f(s.P1.Dy)+","+f(s.P2.Dx)+","+f(s.P2.Dy)+","+f(s.P3.Dx)+","+f(s.P3.Dy))
		case ui.PathVerbConic:
			parts = append(parts, "O"+f(s.P2.Dx)+","+f(s.P2.Dy))
		case ui.PathVerbClose:
			parts = append(parts, "Z")
		}
	}
	return strings.Join(parts, " ")
}

func TestParsePathData(t *testing.T) {
	tests := []struct {
		name string
		d    string
		want string
		ok   bool
	}{
		{"moveto and lineto", "M10 20 L30 40", "M10,20 L30,40", true},
		{"relative", "m10 20 l5 5", "M10,20 L15,25", true},
		{"implicit lineto after M", "M0 0 10 10 20 0", "M0,0 L10,10 L20,0", true},
		{"implicit lineto after m is relative", "m0 0 10 10 10 -10", "M0,0 L10,10 L20,0", true},
		{"horizontal and vertical", "M1 1 H5 V6 h-2 v-3", "M1,1 L5,1 L5,6 L3,6 L3,3", true},
		{"close", "M0 0 L10 0 L10 10 Z", "M0,0 L10,0 L10,10 Z", true},
		{"close lower case", "M0 0 L10 0 z", "M0,0 L10,0 Z", true},
		{"after close the current point is the start", "M5 5 L10 5 z l1 1", "M5,5 L10,5 Z M5,5 L6,6", true},
		{"m after close", "M5 5 L10 5 z m1 1 l1 0", "M5,5 L10,5 Z M6,6 L7,6", true},
		{"cubic", "M0 0 C1 2 3 4 5 6", "M0,0 C1,2,3,4,5,6", true},
		{"relative cubic", "M1 1 c1 2 3 4 5 6", "M1,1 C2,3,4,5,6,7", true},
		{"smooth cubic reflects", "M0 0 C0 10 10 10 10 0 S20 -10 20 0", "M0,0 C0,10,10,10,10,0 C10,-10,20,-10,20,0", true},
		{"smooth cubic without a previous curve", "M0 0 S10 10 20 0", "M0,0 C0,0,10,10,20,0", true},
		{"relative smooth cubic", "M0 0 C0 10 10 10 10 0 s10 -10 10 0", "M0,0 C0,10,10,10,10,0 C10,-10,20,-10,20,0", true},
		{"quad", "M0 0 Q5 10 10 0", "M0,0 Q5,10,10,0", true},
		{"smooth quad reflects", "M0 0 Q5 10 10 0 T20 0", "M0,0 Q5,10,10,0 Q15,-10,20,0", true},
		{"smooth quad without a previous curve", "M0 0 T10 0", "M0,0 Q0,0,10,0", true},
		{"smooth quad after a cubic does not reflect", "M0 0 C0 5 5 5 5 0 T10 0", "M0,0 C0,5,5,5,5,0 Q5,0,10,0", true},
		{"relative quads", "m1 1 q1 2 3 0 t3 0", "M1,1 Q2,3,4,1 Q6,-1,7,1", true},
		{"numbers without separators", "M1.5.5L-1-2", "M1.5,0.5 L-1,-2", true},
		{"a comma before a command", "M1,2,L3,4", "M1,2", false},
		{"exponent", "M1e1 2E0 L-1e-1 0", "M10,2 L-0.1,0", true},
		{"no spaces", "M1 2L3 4Z", "M1,2 L3,4 Z", true},
		{"stops at an error and keeps the earlier segments", "M0 0 L10 10 L20 foo L30 30", "M0,0 L10,10", false},
		{"too few numbers", "M0 0 L10 10 L20", "M0,0 L10,10", false},
		{"unknown command", "M0 0 L10 10 X1 1", "M0,0 L10,10", false},
		{"does not start with a moveto", "L10 10", "", false},
		{"numbers after close", "M0 0 L1 1 z 5 5", "M0,0 L1,1 Z", false},
		{"empty", "", "", true},
		{"white space only", "   ", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ok := ParsePathData(tt.d)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if got := pathString(p); got != tt.want {
				t.Errorf("path = %q\nwant   %q", got, tt.want)
			}
		})
	}
}

func TestParsePathDataArcs(t *testing.T) {
	end := func(p *ui.Path) ui.Offset {
		segs := p.Segments()
		last := segs[len(segs)-1]
		switch last.Verb {
		case ui.PathVerbConic, ui.PathVerbQuad:
			return last.P2
		case ui.PathVerbCubic:
			return last.P3
		}
		return last.P1
	}
	tests := []struct {
		name    string
		d       string
		wantEnd ui.Offset
		ok      bool
		// bounds of the arc, to tell the four arcs of two points apart
		bounds ui.Rect
		line   bool
	}{
		{"flags without separators", "M0 0 A10 10 0 0110 10", ui.Offset{Dx: 10, Dy: 10}, true, ui.NewRectFromLTRB(0, 0, 10, 10), false},
		{"flags with separators", "M0 0 A10 10 0 0 1 10 10", ui.Offset{Dx: 10, Dy: 10}, true, ui.NewRectFromLTRB(0, 0, 10, 10), false},
		{"sweep 0 goes the other way", "M0 0 A10 10 0 0 0 10 10", ui.Offset{Dx: 10, Dy: 10}, true, ui.NewRectFromLTRB(0, 0, 10, 10), false},
		{"large arc", "M0 0 A10 10 0 1 1 10 10", ui.Offset{Dx: 10, Dy: 10}, true, ui.NewRectFromLTRB(0, -10, 20, 10), false},
		{"relative", "m5 5 a10 10 0 0 1 10 10", ui.Offset{Dx: 15, Dy: 15}, true, ui.NewRectFromLTRB(5, 5, 15, 15), false},
		{"radii too small are scaled", "M0 0 A1 1 0 0 1 10 0", ui.Offset{Dx: 10, Dy: 0}, true, ui.NewRectFromLTRB(0, -5, 10, 0), false},
		{"zero radius is a line", "M0 0 A0 5 0 0 1 10 10", ui.Offset{Dx: 10, Dy: 10}, true, ui.NewRectFromLTRB(0, 0, 10, 10), true},
		{"negative radii are taken absolute", "M0 0 A-10 -10 0 0 1 10 10", ui.Offset{Dx: 10, Dy: 10}, true, ui.NewRectFromLTRB(0, 0, 10, 10), false},
		{"the flags must be 0 or 1", "M0 0 A10 10 0 2 1 10 10", ui.Offset{}, false, ui.Rect{}, false},
		{"two arcs in a row", "M0 0 A5 5 0 0 1 10 0 5 5 0 0 1 20 0", ui.Offset{Dx: 20, Dy: 0}, true, ui.NewRectFromLTRB(0, -5, 20, 0), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ok := ParsePathData(tt.d)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v (path %s)", ok, tt.ok, pathString(p))
			}
			if !tt.ok {
				return
			}
			e := end(p)
			if math.Abs(e.Dx-tt.wantEnd.Dx) > 1e-6 || math.Abs(e.Dy-tt.wantEnd.Dy) > 1e-6 {
				t.Errorf("ends at %v, want %v", e, tt.wantEnd)
			}
			b := p.GetTightBounds()
			if math.Abs(b.Left-tt.bounds.Left) > 1e-3 || math.Abs(b.Right-tt.bounds.Right) > 1e-3 ||
				math.Abs(b.Top-tt.bounds.Top) > 1e-3 || math.Abs(b.Bottom-tt.bounds.Bottom) > 1e-3 {
				t.Errorf("bounds %v, want %v (path %s)", b, tt.bounds, pathString(p))
			}
			hasConic := false
			for _, s := range p.Segments() {
				if s.Verb == ui.PathVerbConic || s.Verb == ui.PathVerbCubic || s.Verb == ui.PathVerbQuad {
					hasConic = true
				}
			}
			if hasConic == tt.line {
				t.Errorf("curve segments = %v, want %v", hasConic, !tt.line)
			}
		})
	}
}

func TestParsePathDataArcDirection(t *testing.T) {
	// From (0,0) to (10,0) with radius 5: sweep 1 goes through (5,-5)... no: the center is (5,0), and a sweep in the
	// positive-angle direction (clockwise on screen, y down) from angle 180 passes through angle 270, which is
	// (5,-5). sweep 0 goes through (5,5).
	for _, tt := range []struct {
		d    string
		topY float64 // the extreme y reached
	}{
		{"M0 0 A5 5 0 0 1 10 0", -5},
		{"M0 0 A5 5 0 0 0 10 0", 5},
	} {
		p, ok := ParsePathData(tt.d)
		if !ok {
			t.Fatalf("%s: not ok", tt.d)
		}
		b := p.GetTightBounds()
		got := b.Top
		if tt.topY > 0 {
			got = b.Bottom
		}
		if math.Abs(got-tt.topY) > 1e-3 {
			t.Errorf("%s reaches y = %v, want %v", tt.d, got, tt.topY)
		}
	}
}
