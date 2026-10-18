package ui

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
)

// verbString lists the verbs of the path, such as "MLLC" for move, line, line,
// close.
func verbString(p *Path) string {
	letters := map[PathVerb]byte{PathVerbMove: 'M', PathVerbLine: 'L', PathVerbQuad: 'Q', PathVerbConic: 'O', PathVerbCubic: 'C', PathVerbClose: 'Z'}
	out := make([]byte, 0, len(p.verbs))
	for _, v := range p.verbs {
		out = append(out, letters[v])
	}
	return string(out)
}

func pointsNear(t *testing.T, name string, got, want []Offset, eps float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: %d points %v, want %d %v", name, len(got), got, len(want), want)
		return
	}
	for i := range got {
		if !near(got[i].Dx, want[i].Dx, eps) || !near(got[i].Dy, want[i].Dy, eps) {
			t.Errorf("%s: point %d = %v, want %v", name, i, got[i], want[i])
		}
	}
}

func TestPathBuilders(t *testing.T) {
	tests := []struct {
		name   string
		build  func(p *Path)
		verbs  string
		points []Offset
	}{
		{"empty", func(*Path) {}, "", nil},
		{"move and lines", func(p *Path) { p.MoveTo(1, 2); p.LineTo(3, 4); p.LineTo(5, 6) }, "MLL", []Offset{{1, 2}, {3, 4}, {5, 6}}},
		{"a line first starts at the origin", func(p *Path) { p.LineTo(3, 4) }, "ML", []Offset{{0, 0}, {3, 4}}},
		{"a quad first starts at the origin", func(p *Path) { p.QuadraticBezierTo(1, 1, 2, 0) }, "MQ", []Offset{{0, 0}, {1, 1}, {2, 0}}},
		{"a cubic first starts at the origin", func(p *Path) { p.CubicTo(1, 1, 2, 1, 3, 0) }, "MC", []Offset{{0, 0}, {1, 1}, {2, 1}, {3, 0}}},
		{"relative line", func(p *Path) { p.MoveTo(10, 10); p.RelativeLineTo(1, 2) }, "ML", []Offset{{10, 10}, {11, 12}}},
		{"relative line first", func(p *Path) { p.RelativeLineTo(1, 2) }, "ML", []Offset{{0, 0}, {1, 2}}},
		{"relative move", func(p *Path) { p.MoveTo(10, 10); p.RelativeMoveTo(1, 2) }, "MM", []Offset{{10, 10}, {11, 12}}},
		{"relative quad", func(p *Path) { p.MoveTo(10, 10); p.RelativeQuadraticBezierTo(1, 1, 2, 0) }, "MQ", []Offset{{10, 10}, {11, 11}, {12, 10}}},
		{"relative cubic", func(p *Path) { p.MoveTo(10, 10); p.RelativeCubicTo(1, 1, 2, 1, 3, 0) }, "MC", []Offset{{10, 10}, {11, 11}, {12, 11}, {13, 10}}},
		{"close", func(p *Path) { p.MoveTo(0, 0); p.LineTo(1, 0); p.LineTo(1, 1); p.Close() }, "MLLZ", []Offset{{0, 0}, {1, 0}, {1, 1}}},
		{"a second close is ignored", func(p *Path) { p.MoveTo(0, 0); p.LineTo(1, 0); p.Close(); p.Close() }, "MLZ", []Offset{{0, 0}, {1, 0}}},
		{"close on an empty path is ignored", func(p *Path) { p.Close() }, "", nil},
		{"a line after a close starts at the start of the contour", func(p *Path) {
			p.MoveTo(5, 5)
			p.LineTo(6, 5)
			p.Close()
			p.LineTo(9, 9)
		}, "MLZML", []Offset{{5, 5}, {6, 5}, {5, 5}, {9, 9}}},
		{"a relative line after a close is relative to the start of the contour", func(p *Path) {
			p.MoveTo(5, 5)
			p.LineTo(6, 5)
			p.Close()
			p.RelativeLineTo(1, 1)
		}, "MLZML", []Offset{{5, 5}, {6, 5}, {5, 5}, {6, 6}}},
		{"a move after a close needs no injection", func(p *Path) {
			p.MoveTo(5, 5)
			p.LineTo(6, 5)
			p.Close()
			p.MoveTo(20, 20)
			p.LineTo(21, 20)
		}, "MLZML", []Offset{{5, 5}, {6, 5}, {20, 20}, {21, 20}}},
		{"two moves in a row are kept", func(p *Path) { p.MoveTo(1, 1); p.MoveTo(2, 2) }, "MM", []Offset{{1, 1}, {2, 2}}},
		{"close after a lone move", func(p *Path) { p.MoveTo(1, 1); p.Close() }, "MZ", []Offset{{1, 1}}},
	}
	for _, tt := range tests {
		p := NewPath()
		tt.build(p)
		if got := verbString(p); got != tt.verbs {
			t.Errorf("%s: verbs %q, want %q", tt.name, got, tt.verbs)
		}
		pointsNear(t, tt.name, p.points, tt.points, 0)
	}
}

func TestPathConicTo(t *testing.T) {
	tests := []struct {
		name    string
		w       float64
		verbs   string
		points  []Offset
		weights []float64
	}{
		{"a weight", 0.5, "MO", []Offset{{0, 0}, {1, 1}, {2, 0}}, []float64{0.5}},
		{"a weight of 1 is a quad", 1, "MQ", []Offset{{0, 0}, {1, 1}, {2, 0}}, nil},
		{"a weight of 0 is a line", 0, "ML", []Offset{{0, 0}, {2, 0}}, nil},
		{"a negative weight is a line", -3, "ML", []Offset{{0, 0}, {2, 0}}, nil},
		{"NaN is a line", math.NaN(), "ML", []Offset{{0, 0}, {2, 0}}, nil},
		{"an infinite weight is two lines", math.Inf(1), "MLL", []Offset{{0, 0}, {1, 1}, {2, 0}}, nil},
	}
	for _, tt := range tests {
		p := NewPath()
		p.MoveTo(0, 0)
		p.ConicTo(1, 1, 2, 0, tt.w)
		if got := verbString(p); got != tt.verbs {
			t.Errorf("%s: verbs %q, want %q", tt.name, got, tt.verbs)
		}
		pointsNear(t, tt.name, p.points, tt.points, 0)
		if len(p.weights) != len(tt.weights) || (len(tt.weights) > 0 && p.weights[0] != tt.weights[0]) {
			t.Errorf("%s: weights %v, want %v", tt.name, p.weights, tt.weights)
		}
	}
	p := NewPath()
	p.MoveTo(10, 10)
	p.RelativeConicTo(1, 1, 2, 0, 0.5)
	pointsNear(t, "relative conic", p.points, []Offset{{10, 10}, {11, 11}, {12, 10}}, 0)
}

func TestPathFillTypeAndReset(t *testing.T) {
	p := NewPath()
	if p.FillType() != PathFillTypeNonZero {
		t.Errorf("the default fill type is nonZero, got %v", p.FillType())
	}
	p.SetFillType(PathFillTypeEvenOdd)
	p.MoveTo(1, 1)
	p.LineTo(2, 2)
	c := NewPathFrom(p)
	if c.FillType() != PathFillTypeEvenOdd || verbString(c) != "ML" {
		t.Error("NewPathFrom must copy the fill type and the segments")
	}
	c.LineTo(3, 3)
	if verbString(p) != "ML" {
		t.Error("changing a copy changed the original")
	}
	p.Reset()
	if !p.IsEmpty() || p.FillType() != PathFillTypeEvenOdd {
		t.Error("Reset clears the segments and keeps the fill type")
	}
	p.LineTo(1, 1)
	if verbString(p) != "ML" || p.points[0] != (Offset{}) {
		t.Errorf("after Reset a line starts at the origin again: %v %v", verbString(p), p.points)
	}
	p.SetFillType(PathFillTypeNonZero)
	if p.fillType != PathFillTypeUnspecified {
		t.Error("nonZero is stored as the zero value")
	}
	if p.String() != "Path" || PathFillTypeEvenOdd.String() != "PathFillType.evenOdd" {
		t.Error("String wrong")
	}
}

func TestPathAddRectOvalPolygon(t *testing.T) {
	r := NewPath()
	r.AddRect(Rect{10, 20, 30, 40})
	if verbString(r) != "MLLLZ" {
		t.Errorf("rect verbs = %s", verbString(r))
	}
	pointsNear(t, "rect", r.points, []Offset{{10, 20}, {30, 20}, {30, 40}, {10, 40}}, 0)

	o := NewPath()
	o.AddOval(Rect{0, 0, 20, 10})
	if verbString(o) != "MOOOOZ" {
		t.Errorf("oval verbs = %s", verbString(o))
	}
	// Starts at the right center and goes clockwise, as conics with the corners
	// of the rectangle as control points.
	pointsNear(t, "oval", o.points, []Offset{{20, 5}, {20, 10}, {10, 10}, {0, 10}, {0, 5}, {0, 0}, {10, 0}, {20, 0}, {20, 5}}, 0)
	for _, w := range o.weights {
		if w != math.Sqrt2/2 {
			t.Errorf("oval weight = %v", w)
		}
	}

	poly := NewPath()
	poly.AddPolygon([]Offset{{0, 0}, {10, 0}, {10, 10}}, false)
	poly2 := NewPath()
	poly2.AddPolygon([]Offset{{0, 0}, {10, 0}, {10, 10}}, true)
	empty := NewPath()
	empty.AddPolygon(nil, true)
	if verbString(poly) != "MLL" || verbString(poly2) != "MLLZ" || !empty.IsEmpty() {
		t.Errorf("polygons: %s %s %v", verbString(poly), verbString(poly2), empty.verbs)
	}
}

func TestPathAddPath(t *testing.T) {
	src := NewPath()
	src.MoveTo(0, 0)
	src.LineTo(10, 0)
	src.QuadraticBezierTo(10, 10, 0, 10)
	src.Close()

	tests := []struct {
		name   string
		build  func(dst *Path)
		verbs  string
		points []Offset
	}{
		{"offset", func(d *Path) { d.AddPath(src, Offset{5, 7}) }, "MLQZ", []Offset{{5, 7}, {15, 7}, {15, 17}, {5, 17}}},
		{"matrix and offset", func(d *Path) {
			d.AddPath(src, Offset{1, 1}, AddPathOpts{Matrix4: NewMatrix4Diagonal3Values(2, 2, 1)})
		}, "MLQZ", []Offset{{1, 1}, {21, 1}, {21, 21}, {1, 21}}},
		{"after other segments it is appended as it is", func(d *Path) {
			d.MoveTo(100, 100)
			d.LineTo(101, 100)
			d.AddPath(src, Offset{})
		}, "MLMLQZ", []Offset{{100, 100}, {101, 100}, {0, 0}, {10, 0}, {10, 10}, {0, 10}}},
		{"extend joins with a line", func(d *Path) {
			d.MoveTo(100, 100)
			d.ExtendWithPath(src, Offset{})
		}, "MLLQZ", []Offset{{100, 100}, {0, 0}, {10, 0}, {10, 10}, {0, 10}}},
		{"extend on an empty path is an add", func(d *Path) { d.ExtendWithPath(src, Offset{1, 2}) }, "MLQZ", []Offset{{1, 2}, {11, 2}, {11, 12}, {1, 12}}},
		{"extend does not add a line to the same point", func(d *Path) {
			d.MoveTo(0, 0)
			d.ExtendWithPath(src, Offset{})
		}, "MLQZ", []Offset{{0, 0}, {10, 0}, {10, 10}, {0, 10}}},
		{"adding an empty path changes nothing", func(d *Path) { d.AddPath(NewPath(), Offset{1, 1}) }, "", nil},
	}
	for _, tt := range tests {
		dst := NewPath()
		tt.build(dst)
		if got := verbString(dst); got != tt.verbs {
			t.Errorf("%s: verbs %q, want %q", tt.name, got, tt.verbs)
		}
		pointsNear(t, tt.name, dst.points, tt.points, 1e-9)
	}

	// A line added after AddPath of a closed contour begins at the added contour's start.
	d := NewPath()
	d.AddPath(src, Offset{3, 3})
	d.LineTo(50, 50)
	if verbString(d) != "MLQZML" || d.points[4] != (Offset{3, 3}) {
		t.Errorf("segment after an added closed contour: %s %v", verbString(d), d.points)
	}
	// Adding a path to itself works on a copy.
	self := NewPath()
	self.AddRect(Rect{0, 0, 1, 1})
	self.AddPath(self, Offset{5, 5})
	if verbString(self) != "MLLLZMLLLZ" || self.points[4] != (Offset{5, 5}) {
		t.Errorf("self add: %s %v", verbString(self), self.points)
	}
}

func TestPathShiftAndTransform(t *testing.T) {
	p := NewPath()
	p.SetFillType(PathFillTypeEvenOdd)
	p.MoveTo(1, 1)
	p.ConicTo(2, 2, 3, 1, 0.5)
	p.Close()
	s := p.Shift(Offset{10, 20})
	pointsNear(t, "shift", s.points, []Offset{{11, 21}, {12, 22}, {13, 21}}, 0)
	if s.fillType != PathFillTypeEvenOdd || s.weights[0] != 0.5 || p.points[0] != (Offset{1, 1}) {
		t.Error("Shift copies the path, with its fill type and weights")
	}
	sc := p.Transform(NewMatrix4Diagonal3Values(2, 3, 1))
	pointsNear(t, "scale", sc.points, []Offset{{2, 3}, {4, 6}, {6, 3}}, 0)
	rot := p.Transform(NewMatrix4RotationZ(math.Pi / 2))
	pointsNear(t, "rotate", rot.points, []Offset{{-1, 1}, {-2, 2}, {-1, 3}}, 1e-6)
	if id := p.Transform(NewMatrix4Identity()); id == p || id.points[1] != p.points[1] {
		t.Error("Transform by the identity gives a copy")
	}
	// A perspective transform changes the weight of a conic.
	persp := NewMatrix4Identity()
	persp.storage[3] = 0.01
	pp := p.Transform(persp)
	if pp.weights[0] == 0.5 {
		t.Errorf("perspective must change the conic weight, got %v", pp.weights)
	}
}

func TestPathBounds(t *testing.T) {
	nan := math.NaN()
	tests := []struct {
		name  string
		build func(p *Path)
		want  Rect // GetBounds
		tight Rect // GetTightBounds
	}{
		{"empty", func(*Path) {}, Rect{}, Rect{}},
		{"a lone move", func(p *Path) { p.MoveTo(3, 4) }, Rect{3, 4, 3, 4}, Rect{3, 4, 3, 4}},
		{"lines", func(p *Path) { p.MoveTo(1, 5); p.LineTo(-2, 3); p.LineTo(4, 9) }, Rect{-2, 3, 4, 9}, Rect{-2, 3, 4, 9}},
		{"a quad includes its control point", func(p *Path) { p.MoveTo(0, 0); p.QuadraticBezierTo(5, 10, 10, 0) }, Rect{0, 0, 10, 10}, Rect{0, 0, 10, 5}},
		{"a cubic includes its control points", func(p *Path) { p.MoveTo(0, 0); p.CubicTo(0, 10, 10, 10, 10, 0) }, Rect{0, 0, 10, 10}, Rect{0, 0, 10, 7.5}},
		{"an oval", func(p *Path) { p.AddOval(Rect{0, 0, 10, 20}) }, Rect{0, 0, 10, 20}, Rect{0, 0, 10, 20}},
		{"NaN gives the empty rectangle", func(p *Path) { p.MoveTo(1, 1); p.LineTo(nan, 2) }, Rect{}, Rect{}},
		{"infinity gives the empty rectangle", func(p *Path) { p.MoveTo(1, 1); p.LineTo(math.Inf(1), 2) }, Rect{}, Rect{}},
	}
	for _, tt := range tests {
		p := NewPath()
		tt.build(p)
		rectNear(t, p.GetBounds(), tt.want, 1e-12)
		rectNear(t, p.GetTightBounds(), tt.tight, 1e-9)
	}
	// A conic through the rectangle corners is tight at the oval.
	c := NewPath()
	c.MoveTo(0, 0)
	c.ConicTo(10, 0, 10, 10, math.Sqrt2/2)
	rectNear(t, c.GetTightBounds(), Rect{0, 0, 10, 10}, 1e-9)
	rectNear(t, c.GetBounds(), Rect{0, 0, 10, 10}, 0)
	// A line along an axis has no area, as the docs say.
	l := NewPath()
	l.MoveTo(0, 0)
	l.LineTo(10, 0)
	if !l.GetBounds().IsEmpty() || l.GetBounds().Width()+l.GetBounds().Height() <= 0 {
		t.Error("a horizontal line has empty bounds with a width")
	}
}

func TestPathSegments(t *testing.T) {
	p := NewPath()
	p.MoveTo(0, 0)
	p.LineTo(1, 0)
	p.QuadraticBezierTo(2, 1, 3, 0)
	p.ConicTo(4, 1, 5, 0, 0.5)
	p.CubicTo(6, 1, 7, 1, 8, 0)
	p.Close()
	want := []PathSegment{
		{Verb: PathVerbMove, P0: Offset{0, 0}},
		{Verb: PathVerbLine, P0: Offset{0, 0}, P1: Offset{1, 0}},
		{Verb: PathVerbQuad, P0: Offset{1, 0}, P1: Offset{2, 1}, P2: Offset{3, 0}},
		{Verb: PathVerbConic, P0: Offset{3, 0}, P1: Offset{4, 1}, P2: Offset{5, 0}, Weight: 0.5},
		{Verb: PathVerbCubic, P0: Offset{5, 0}, P1: Offset{6, 1}, P2: Offset{7, 1}, P3: Offset{8, 0}},
		{Verb: PathVerbClose, P0: Offset{8, 0}, P1: Offset{0, 0}},
	}
	got := p.Segments()
	if len(got) != len(want) {
		t.Fatalf("%d segments, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("segment %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPathMultipleOptionsPanics(t *testing.T) {
	defer func() {
		appErr, ok := errors.FromError(recoverError(t, recover()))
		if !ok || appErr.Tag != "MultipleOptions" {
			t.Errorf("expected MultipleOptions, got %v", appErr)
		}
	}()
	NewPath().AddPath(NewPath(), Offset{}, AddPathOpts{}, AddPathOpts{})
}
