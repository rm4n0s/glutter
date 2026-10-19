package painting

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// recordOps runs fn on a canvas and returns the operations it recorded.
func recordOps(fn func(c *ui.Canvas)) []ui.DisplayOp {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	fn(c)
	return rec.EndRecording().DisplayList().Ops()
}

// lerpBorder is the _LerpBorder of shape_border_test.dart.
type lerpBorder struct {
	ShapeBorderDefaults
	t    foundation.Opt[float64]
	side BorderSide
}

func (b lerpBorder) Side() BorderSide                         { return b.side }
func (b lerpBorder) Dimensions() EdgeInsetsGeometry           { return EdgeInsetsZero }
func (b lerpBorder) CopyWithSide(s BorderSide) OutlinedBorder { return lerpBorder{t: b.t, side: s} }
func (b lerpBorder) GetInnerPath(r ui.Rect, _ ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRect(r)
	return p
}
func (b lerpBorder) GetOuterPath(r ui.Rect, td ui.TextDirection) *ui.Path {
	return b.GetInnerPath(r, td)
}
func (b lerpBorder) Paint(*ui.Canvas, ui.Rect, ui.TextDirection) {}
func (b lerpBorder) Scale(t float64) ShapeBorder {
	return lerpBorder{t: foundation.Some(t), side: b.side.Scale(t)}
}
func (b lerpBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	return ShapeBorderDefaultLerpFrom(b, a, t)
}
func (b lerpBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	return ShapeBorderDefaultLerpTo(b, o, t)
}
func (b lerpBorder) String() string { return fmt.Sprintf("_LerpBorder(t: %v)", b.t) }

type reverseLerpToBorder struct{ lerpBorder }

func (b reverseLerpToBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if _, ok := o.(lerpBorder); ok {
		return lerpBorder{t: foundation.Some(t), side: BorderSideNone}
	}
	return b.lerpBorder.LerpTo(o, t)
}

type reverseLerpFromBorder struct{ lerpBorder }

func (b reverseLerpFromBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if _, ok := a.(lerpBorder); ok {
		return lerpBorder{t: foundation.Some(t), side: BorderSideNone}
	}
	return b.lerpBorder.LerpFrom(a, t)
}

func TestShapeBorderLerpIdentical(t *testing.T) {
	var border Border
	if BorderLerp(nil, nil, 0) != nil {
		t.Error("Border lerp of nils should be nil")
	}
	if got := BorderLerp(&border, &border, 0.5); got != &border {
		t.Error("Border lerp of identical borders should return the same pointer")
	}
	var boxBorder BoxBorder = border
	if got := BoxBorderLerp(boxBorder, boxBorder, 0.5); got != boxBorder {
		t.Error("BoxBorder lerp of identical borders should return the same border")
	}
	var dir BorderDirectional
	if BorderDirectionalLerp(nil, nil, 0) != nil {
		t.Error("BorderDirectional lerp of nils should be nil")
	}
	if got := ShapeBorderLerp(dir, dir, 0.5); got != ShapeBorder(dir) {
		t.Error("ShapeBorder lerp of identical borders should return the same border")
	}
	var outlined OutlinedBorder = NewRoundedRectangleBorder()
	if OutlinedBorderLerp(nil, nil, 0) != nil {
		t.Error("OutlinedBorder lerp of nils should be nil")
	}
	if got := OutlinedBorderLerp(outlined, outlined, 0.5); got != outlined {
		t.Error("OutlinedBorder lerp of identical borders should return the same border")
	}
	var shape ShapeBorder = NewCircleBorder()
	if ShapeBorderLerp(nil, nil, 0) != nil {
		t.Error("ShapeBorder lerp of nils should be nil")
	}
	if got := ShapeBorderLerp(shape, shape, 0.5); got != shape {
		t.Error("ShapeBorder lerp of identical borders should return the same border")
	}
}

func TestShapeBorderLerpTriesReverseInterpolation(t *testing.T) {
	want := lerpBorder{t: foundation.Some(0.75), side: BorderSideNone}
	tests := []struct {
		name string
		a, b OutlinedBorder
	}{
		{"to", lerpBorder{side: BorderSideNone}, reverseLerpToBorder{lerpBorder{side: BorderSideNone}}},
		{"from", reverseLerpFromBorder{lerpBorder{side: BorderSideNone}}, lerpBorder{side: BorderSideNone}},
	}
	for _, tt := range tests {
		t.Run("ShapeBorder "+tt.name, func(t *testing.T) {
			if got := ShapeBorderLerp(tt.a, tt.b, 0.25); got != ShapeBorder(want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
		t.Run("OutlinedBorder "+tt.name, func(t *testing.T) {
			if got := OutlinedBorderLerp(tt.a, tt.b, 0.25); got != OutlinedBorder(want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestOutlinedShapeBordersInterpolateSymmetrically(t *testing.T) {
	borders := []OutlinedBorder{NewRoundedRectangleBorder(), NewStadiumBorder(), NewCircleBorder(), NewOvalBorder(), NewStarBorder()}
	for _, a := range borders {
		for _, b := range borders {
			if x, y := fmt.Sprintf("%T", ShapeBorderLerp(a, b, 0.25)), fmt.Sprintf("%T", ShapeBorderLerp(b, a, 0.75)); x != y {
				t.Errorf("ShapeBorderLerp(%v, %v, 0.25) is a %s, but the reverse is a %s", a, b, x, y)
			}
			if x, y := fmt.Sprintf("%T", OutlinedBorderLerp(a, b, 0.25)), fmt.Sprintf("%T", OutlinedBorderLerp(b, a, 0.75)); x != y {
				t.Errorf("OutlinedBorderLerp(%v, %v, 0.25) is a %s, but the reverse is a %s", a, b, x, y)
			}
		}
	}
}

func TestCompoundBorders(t *testing.T) {
	green, blue := ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF)
	b1 := NewBorderAll(BorderSideOpts{Color: foundation.Some(green)})
	b2 := NewBorderAll(BorderSideOpts{Color: foundation.Some(blue)})
	cs := func(c ui.Color) string { return c.String() }
	tests := []struct {
		name string
		got  ShapeBorder
		want string
	}{
		{"b1 + b2", ShapeBorderPlus(b1, b2), "Border.all(BorderSide(color: " + cs(green) + ")) + Border.all(BorderSide(color: " + cs(blue) + "))"},
		{"b1 + (b2 + b2)", ShapeBorderPlus(b1, ShapeBorderPlus(b2, b2)), "Border.all(BorderSide(color: " + cs(green) + ")) + Border.all(BorderSide(color: " + cs(blue) + ", width: 2.0))"},
		{"(b1 + b2) + b2", ShapeBorderPlus(ShapeBorderPlus(b1, b2), b2), "Border.all(BorderSide(color: " + cs(green) + ")) + Border.all(BorderSide(color: " + cs(blue) + ", width: 2.0))"},
		{"scale 3", ShapeBorderPlus(b1, b2).Scale(3), "Border.all(BorderSide(color: " + cs(green) + ", width: 3.0)) + Border.all(BorderSide(color: " + cs(blue) + ", width: 3.0))"},
		{"scale 0", ShapeBorderPlus(b1, b2).Scale(0), "Border.all(BorderSide(color: " + cs(green) + ", width: 0.0, style: none)) + Border.all(BorderSide(color: " + cs(blue) + ", width: 0.0, style: none))"},
		{"lerp 0", ShapeBorderLerp(ShapeBorderPlus(b2, b1), ShapeBorderPlus(b1, b2), 0), "Border.all(BorderSide(color: " + cs(blue) + ")) + Border.all(BorderSide(color: " + cs(green) + "))"},
		{"lerp .2", ShapeBorderLerp(ShapeBorderPlus(b2, b1), ShapeBorderPlus(b1, b2), 0.2), "Border.all(BorderSide(color: " + cs(ui.NewColor(0xff0033cc)) + ")) + Border.all(BorderSide(color: " + cs(ui.NewColor(0xff00cc33)) + "))"},
		{"lerp 1/3", ShapeBorderLerp(ShapeBorderPlus(b2, b1), ShapeBorderPlus(b1, b2), 1.0/3), "Border.all(BorderSide(color: " + cs(ui.NewColor(0xff0055aa)) + ")) + Border.all(BorderSide(color: " + cs(ui.NewColor(0xff00aa55)) + "))"},
		{"lerp 1", ShapeBorderLerp(ShapeBorderPlus(b2, b1), ShapeBorderPlus(b1, b2), 1), "Border.all(BorderSide(color: " + cs(green) + ")) + Border.all(BorderSide(color: " + cs(blue) + "))"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.got.String(); foundation.DebugMode && got != tt.want {
				t.Errorf("String =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
	if got, want := ShapeBorderPlus(ShapeBorderPlus(b1, b2), b2).String(), ShapeBorderPlus(b1, ShapeBorderPlus(b2, b2)).String(); got != want {
		t.Errorf("(b1 + b2) + b2 = %s, b1 + (b2 + b2) = %s", got, want)
	}
	if got := ShapeBorderPlus(b1, b2).Dimensions(); !got.Equal(NewEdgeInsetsAll(2)) {
		t.Errorf("dimensions = %v", got)
	}
	rect := ui.NewRectFromLTRB(11, 15, 299, 175)
	// The compound border paints the last border added first, on the outside.
	paintedRects := func(b ShapeBorder) []ui.Rect {
		var got []ui.Rect
		for _, op := range recordOps(func(c *ui.Canvas) { b.Paint(c, rect, ui.TextDirectionUnspecified) }) {
			if d, ok := op.(ui.OpDrawRect); ok {
				got = append(got, d.Rect)
			}
		}
		return got
	}
	want := []ui.Rect{rect.Deflate(0.5), rect.Deflate(1.5)}
	if got := paintedRects(ShapeBorderPlus(b1, b2)); len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("painted rects = %v, want %v", got, want)
	}
	three := ShapeBorderPlus(ShapeBorderPlus(b1, b2), b1)
	if got := three.Dimensions(); !got.Equal(NewEdgeInsetsAll(3)) {
		t.Errorf("three dimensions = %v", got)
	}
	want = []ui.Rect{rect.Deflate(0.5), rect.Deflate(1.5), rect.Deflate(2.5)}
	if got := paintedRects(three); len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("painted rects = %v, want %v", got, want)
	}
}

func TestCompoundDirectionalBorders(t *testing.T) {
	green, blue := ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF)
	s1 := side(BorderSideOpts{Color: foundation.Some(green)})
	s2 := side(BorderSideOpts{Color: foundation.Some(blue)})
	b1 := BorderDirectional{Top: s1, Start: s1, End: s1, Bottom: s1}
	b2 := BorderDirectional{Top: s2, Start: s2, End: s2, Bottom: s2}
	if got, want := ShapeBorderPlus(ShapeBorderPlus(b1, b2), b2).String(), ShapeBorderPlus(b1, ShapeBorderPlus(b2, b2)).String(); got != want {
		t.Errorf("(b1 + b2) + b2 = %s, b1 + (b2 + b2) = %s", got, want)
	}
	if got := ShapeBorderPlus(b1, b2).Dimensions(); !got.Equal(NewEdgeInsetsDirectionalFromSTEB(2, 2, 2, 2)) {
		t.Errorf("dimensions = %v", got)
	}
	plus := ShapeBorderPlus(b1, b2).String()
	if want := b1.String() + " + " + b2.String(); plus != want {
		t.Errorf("String = %s, want %s", plus, want)
	}
	rect := ui.NewRectFromLTRB(11, 15, 299, 175)
	var got []ui.Rect
	for _, op := range recordOps(func(c *ui.Canvas) { ShapeBorderPlus(b1, b2).Paint(c, rect, ui.TextDirectionRTL) }) {
		if d, ok := op.(ui.OpDrawRect); ok {
			got = append(got, d.Rect)
		}
	}
	if len(got) != 2 || got[0] != rect.Deflate(0.5) || got[1] != rect.Deflate(1.5) {
		t.Errorf("painted rects = %v", got)
	}
}
