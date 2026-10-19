package painting

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func widthSide(w float64) BorderSide { return sideW(w) }

func centerSide(w float64) BorderSide {
	return side(BorderSideOpts{Width: foundation.Some(w), StrokeAlign: foundation.Some(StrokeAlignCenter)})
}

func outsideSide(w float64) BorderSide {
	return side(BorderSideOpts{Width: foundation.Some(w), StrokeAlign: foundation.Some(StrokeAlignOutside)})
}

func rectNear(a, b ui.Rect) bool { return rectNearTol(a, b, 1e-6) }

func rectNearTol(a, b ui.Rect, d float64) bool {
	return near(a.Left, b.Left, d) && near(a.Top, b.Top, d) && near(a.Right, b.Right, d) && near(a.Bottom, b.Bottom, d)
}

func isUnitCircle(p *ui.Path) bool {
	for _, tt := range []struct {
		x, y float64
		in   bool
	}{{0, 0, true}, {0.9, 0, true}, {0, 0.9, true}, {-0.9, 0, true}, {0, -0.9, true}, {0.7, 0.7, true}, {1.1, 0, false}, {0.8, 0.8, false}, {-0.8, -0.8, false}} {
		if p.Contains(ui.Offset{Dx: tt.x, Dy: tt.y}) != tt.in {
			return false
		}
	}
	return true
}

// looksLike checks the points the path includes and excludes.
func looksLike(p *ui.Path, includes, excludes []ui.Offset) bool {
	for _, o := range includes {
		if !p.Contains(o) {
			return false
		}
	}
	for _, o := range excludes {
		if p.Contains(o) {
			return false
		}
	}
	return true
}

func TestCircleBorder(t *testing.T) {
	if b := NewCircleBorder(); b.Side() != BorderSideNone || b.Eccentricity != 0 {
		t.Errorf("defaults = %v", b)
	}
	circleRect := ui.NewRectFromLTWH(50, 0, 100, 100)
	rect := ui.NewRectFromLTWH(0, 0, 200, 100)
	oval := NewCircleBorder(CircleBorderOpts{Eccentricity: 1})
	o10 := NewCircleBorder(CircleBorderOpts{Side: widthSide(10), Eccentricity: 1})
	tests := []struct {
		name string
		got  ui.Rect
		want ui.Rect
	}{
		{"inner", NewCircleBorder().GetInnerPath(rect, ui.TextDirectionLTR).GetBounds(), circleRect},
		{"outer", NewCircleBorder().GetOuterPath(rect, ui.TextDirectionLTR).GetBounds(), circleRect},
		{"oval outer", oval.GetOuterPath(rect, ui.TextDirectionLTR).GetBounds(), rect},
		{"oval inner", oval.GetInnerPath(rect, ui.TextDirectionLTR).GetBounds(), rect},
		{"o10 outer", o10.GetOuterPath(rect, ui.TextDirectionLTR).GetBounds(), ui.NewRectFromLTWH(0, 0, 200, 100)},
		{"o10 inner", o10.GetInnerPath(rect, ui.TextDirectionLTR).GetBounds(), ui.NewRectFromLTWH(10, 10, 180, 80)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !rectNear(tt.got, tt.want) {
				t.Errorf("bounds = %v, want %v", tt.got, tt.want)
			}
		})
	}
	side := NewBorderSide(BorderSideOpts{Width: foundation.Some(10.0), Color: foundation.Some(ui.NewColor(0xff123456))})
	if NewCircleBorder().CopyWith(CircleBorderCopy{}) != NewCircleBorder() ||
		NewCircleBorder().CopyWith(CircleBorderCopy{Side: foundation.Some(side)}) != NewCircleBorder(CircleBorderOpts{Side: side}) ||
		NewCircleBorder().CopyWith(CircleBorderCopy{Eccentricity: foundation.Some(0.5)}) != NewCircleBorder(CircleBorderOpts{Eccentricity: 0.5}) {
		t.Error("CopyWith is wrong")
	}

	c10 := NewCircleBorder(CircleBorderOpts{Side: widthSide(10)})
	c15 := NewCircleBorder(CircleBorderOpts{Side: widthSide(15)})
	c20 := NewCircleBorder(CircleBorderOpts{Side: widthSide(20)})
	if !c10.Dimensions().Equal(NewEdgeInsetsAll(10)) || c10.Scale(2) != ShapeBorder(c20) || c20.Scale(0.5) != ShapeBorder(c10) {
		t.Error("dimensions or scale is wrong")
	}
	if ShapeBorderLerp(c10, c20, 0) != ShapeBorder(c10) || ShapeBorderLerp(c10, c20, 0.5) != ShapeBorder(c15) || ShapeBorderLerp(c10, c20, 1) != ShapeBorder(c20) {
		t.Error("lerp is wrong")
	}
	unit := ui.NewRectFromCircle(ui.Offset{}, 1)
	if !isUnitCircle(c10.GetInnerPath(unit.Inflate(10), ui.TextDirectionLTR)) || !isUnitCircle(c10.GetOuterPath(unit, ui.TextDirectionLTR)) {
		t.Error("the paths should be the unit circle")
	}
	ops := recordOps(func(c *ui.Canvas) { c10.Paint(c, ui.NewRectFromLTWH(10, 20, 30, 40), ui.TextDirectionLTR) })
	if len(ops) != 1 {
		t.Fatalf("ops = %v", ops)
	}
	if d, ok := ops[0].(ui.OpDrawCircle); !ok || d.Center != (ui.Offset{Dx: 25, Dy: 40}) || d.Radius != 10 || d.Paint.StrokeWidth() != 10 {
		t.Errorf("paint = %v", ops[0])
	}
}

func TestOvalBorder(t *testing.T) {
	if b := NewOvalBorder(); b.Side() != BorderSideNone || b.Eccentricity != 1.0 {
		t.Errorf("defaults = %v", b)
	}
	side := NewBorderSide(BorderSideOpts{Width: foundation.Some(10.0)})
	if got := NewOvalBorder().CopyWith(CircleBorderCopy{Side: foundation.Some(side)}); got != NewOvalBorder(OvalBorderOpts{Side: side}) {
		t.Errorf("CopyWith = %v", got)
	}
	if got := NewOvalBorder().CopyWith(CircleBorderCopy{Eccentricity: foundation.Some(0.5)}); got.Eccentricity != 0.5 {
		t.Errorf("CopyWith eccentricity = %v", got.Eccentricity)
	}
	o10 := NewOvalBorder(OvalBorderOpts{Side: widthSide(10)})
	o20 := NewOvalBorder(OvalBorderOpts{Side: widthSide(20)})
	if got := o10.Scale(2); got != ShapeBorder(o20) {
		t.Errorf("Scale = %v", got)
	}
	if got := ShapeBorderLerp(o10, o20, 0.5).(OvalBorder); got.Side().Width != 15 || got.Eccentricity != 1 {
		t.Errorf("lerp = %v", got)
	}
	if got := NewOvalBorder().String(); foundation.DebugMode && got != "OvalBorder(BorderSide(width: 0.0, style: none))" {
		t.Errorf("String = %q", got)
	}
	if got := NewOvalBorder(OvalBorderOpts{Eccentricity: foundation.Some(0.5)}).String(); foundation.DebugMode && got != "OvalBorder(BorderSide(width: 0.0, style: none), eccentricity: 0.5)" {
		t.Errorf("String = %q", got)
	}
	// An OvalBorder lerps with a CircleBorder as a CircleBorder does.
	if _, ok := ShapeBorderLerp(NewOvalBorder(), NewCircleBorder(), 0.5).(CircleBorder); !ok {
		t.Errorf("oval to circle = %T", ShapeBorderLerp(NewOvalBorder(), NewCircleBorder(), 0.5))
	}
}

func TestRoundedRectangleBorder(t *testing.T) {
	if b := NewRoundedRectangleBorder(); b.Side() != BorderSideNone || !b.radius().Equal(BorderRadiusZero) {
		t.Errorf("defaults = %v", b)
	}
	mk := func(w, r float64) RoundedRectangleBorder {
		return NewRoundedRectangleBorder(RoundedRectangleBorderOpts{Side: widthSide(w), BorderRadius: NewBorderRadiusCircular(r)})
	}
	c10, c15, c20 := mk(10, 100), mk(15, 150), mk(20, 200)
	if !c10.Dimensions().Equal(NewEdgeInsetsAll(10)) || c10.Scale(2) != ShapeBorder(c20) || c20.Scale(0.5) != ShapeBorder(c10) {
		t.Error("dimensions or scale is wrong")
	}
	if ShapeBorderLerp(c10, c20, 0) != ShapeBorder(c10) || ShapeBorderLerp(c10, c20, 0.5) != ShapeBorder(c15) || ShapeBorderLerp(c10, c20, 1) != ShapeBorder(c20) {
		t.Error("lerp is wrong")
	}
	c1 := NewRoundedRectangleBorder(RoundedRectangleBorderOpts{Side: side(BorderSideOpts{}), BorderRadius: NewBorderRadiusCircular(1)})
	c2 := NewRoundedRectangleBorder(RoundedRectangleBorderOpts{Side: side(BorderSideOpts{}), BorderRadius: NewBorderRadiusCircular(2)})
	if !isUnitCircle(c2.GetInnerPath(ui.NewRectFromCircle(ui.Offset{}, 2), ui.TextDirectionLTR)) || !isUnitCircle(c1.GetOuterPath(ui.NewRectFromCircle(ui.Offset{}, 1), ui.TextDirectionLTR)) {
		t.Error("the paths should be the unit circle")
	}
	rect := ui.NewRectFromLTRB(10, 20, 80, 190)
	ops := recordOps(func(c *ui.Canvas) { c10.Paint(c, rect, ui.TextDirectionLTR) })
	if len(ops) != 1 {
		t.Fatalf("ops = %v", ops)
	}
	d, ok := ops[0].(ui.OpDrawDRRect)
	if !ok || d.Outer != ui.NewRRectFromRectAndRadius(rect, ui.NewRadiusCircular(100)) || d.Inner != ui.NewRRectFromRectAndRadius(rect.Deflate(10), ui.NewRadiusCircular(90)) {
		t.Errorf("paint = %v", ops[0])
	}
	directional := NewRoundedRectangleBorder(RoundedRectangleBorderOpts{BorderRadius: BorderRadiusDirectional{TopStart: ui.NewRadiusCircular(20)}})
	if got, want := ShapeBorderLerp(directional, c10, 1), ShapeBorderLerp(c10, directional, 0); got != want {
		t.Errorf("lerp(directional, c10, 1) = %v, lerp(c10, directional, 0) = %v", got, want)
	}
	r := NewRoundedRectangleBorder(RoundedRectangleBorderOpts{BorderRadius: NewBorderRadiusCircular(5)})
	copied := NewRoundedRectangleBorder().CopyWith(RoundedRectangleBorderCopy{Side: foundation.Some(widthSide(10)), BorderRadius: NewBorderRadiusCircular(5)})
	if copied != NewRoundedRectangleBorder(RoundedRectangleBorderOpts{Side: widthSide(10), BorderRadius: NewBorderRadiusCircular(5)}) || r.Side() != BorderSideNone {
		t.Errorf("CopyWith = %v", copied)
	}
}

func TestShapesToCircleBorder(t *testing.T) {
	rect := ui.NewRectFromLTWH(0, 0, 100, 20)
	includesRounded := []ui.Offset{{Dx: 30, Dy: 10}, {Dx: 50, Dy: 10}}
	excludes := []ui.Offset{{Dx: 1, Dy: 1}, {Dx: 99, Dy: 19}}
	circleIncludes := []ui.Offset{{Dx: 50, Dy: 10}}
	circleExcludes := []ui.Offset{{Dx: 1, Dy: 1}, {Dx: 30, Dy: 10}, {Dx: 99, Dy: 19}}
	circle := NewCircleBorder()
	shapes := []struct {
		name string
		b    ShapeBorder
	}{
		{"RoundedRectangleBorder", NewRoundedRectangleBorder(RoundedRectangleBorderOpts{BorderRadius: NewBorderRadiusCircular(10)})},
		{"RoundedSuperellipseBorder", NewRoundedSuperellipseBorder(RoundedSuperellipseBorderOpts{BorderRadius: NewBorderRadiusCircular(10)})},
		{"StadiumBorder", NewStadiumBorder()},
	}
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			lerp := ShapeBorderLerp
			like := func(b ShapeBorder, wantCircle bool) bool {
				p := b.GetOuterPath(rect, ui.TextDirectionLTR)
				if wantCircle {
					return looksLike(p, circleIncludes, circleExcludes)
				}
				return looksLike(p, includesRounded, excludes)
			}
			checks := []struct {
				name       string
				b          ShapeBorder
				wantCircle bool
			}{
				{"shape", s.b, false}, {"circle", circle, true},
				{"0.1", lerp(s.b, circle, 0.1), false}, {"0.9", lerp(s.b, circle, 0.9), true},
				{"0.9 back 0.1", lerp(lerp(s.b, circle, 0.9), s.b, 0.1), true},
				{"0.9 back 0.9", lerp(lerp(s.b, circle, 0.9), s.b, 0.9), false},
				{"0.1 on 0.1", lerp(lerp(s.b, circle, 0.1), circle, 0.1), false},
				{"0.1 on 0.9", lerp(lerp(s.b, circle, 0.1), circle, 0.9), true},
				{"mid 0.1", lerp(lerp(s.b, circle, 0.1), lerp(s.b, circle, 0.9), 0.1), false},
				{"mid 0.9", lerp(lerp(s.b, circle, 0.1), lerp(s.b, circle, 0.9), 0.9), true},
				{"shape to 0.9 at 0.1", lerp(s.b, lerp(s.b, circle, 0.9), 0.1), false},
				{"shape to 0.9 at 0.9", lerp(s.b, lerp(s.b, circle, 0.9), 0.9), true},
				{"circle to 0.1 at 0.1", lerp(circle, lerp(s.b, circle, 0.1), 0.1), true},
				{"circle to 0.1 at 0.9", lerp(circle, lerp(s.b, circle, 0.1), 0.9), false},
			}
			for _, c := range checks {
				if !like(c.b, c.wantCircle) {
					t.Errorf("%s: the outer path doesn't look like a %s", c.name, map[bool]string{false: "rounded shape", true: "circle"}[c.wantCircle])
				}
			}
		})
	}
	rr := shapes[0].b
	wantRR := func(pct string) string {
		return "RoundedRectangleBorder(BorderSide(width: 0.0, style: none), BorderRadius.circular(10.0), " + pct + "% of the way to being a CircleBorder)"
	}
	for _, tt := range []struct {
		name string
		b    ShapeBorder
		want string
	}{
		{"0.1", ShapeBorderLerp(rr, circle, 0.1), wantRR("10.0")},
		{"0.2", ShapeBorderLerp(rr, circle, 0.2), wantRR("20.0")},
		{"compound", ShapeBorderLerp(ShapeBorderLerp(rr, circle, 0.1), ShapeBorderLerp(rr, circle, 0.9), 0.9), wantRR("82.0")},
		{"circle first 0.9", ShapeBorderLerp(circle, rr, 0.9), wantRR("10.0")},
		{"circle first 0.8", ShapeBorderLerp(circle, rr, 0.8), wantRR("20.0")},
	} {
		if got := tt.b.String(); foundation.DebugMode && got != tt.want {
			t.Errorf("%s: String = %q, want %q", tt.name, got, tt.want)
		}
	}
	oval := NewCircleBorder(CircleBorderOpts{Eccentricity: 0.5})
	if got, want := ShapeBorderLerp(rr, oval, 0.1).String(),
		"RoundedRectangleBorder(BorderSide(width: 0.0, style: none), BorderRadius.circular(10.0), 10.0% of the way to being a CircleBorder that is 50.0% oval)"; foundation.DebugMode && got != want {
		t.Errorf("oval String = %q, want %q", got, want)
	}
	if got, want := ShapeBorderLerp(NewStadiumBorder(), circle, 0.25).String(),
		"StadiumBorder(BorderSide(width: 0.0, style: none), 25.0% of the way to being a CircleBorder)"; foundation.DebugMode && got != want {
		t.Errorf("stadium String = %q, want %q", got, want)
	}
	if got, want := ShapeBorderLerp(NewStadiumBorder(), rr, 0.25).String(),
		"StadiumBorder(BorderSide(width: 0.0, style: none), BorderRadius.circular(10.0), 25.0% of the way to being a RoundedRectangleBorder)"; foundation.DebugMode && got != want {
		t.Errorf("stadium to rounded String = %q, want %q", got, want)
	}
}

func TestStadiumBorder(t *testing.T) {
	if NewStadiumBorder().Side() != BorderSideNone {
		t.Error("default side should be none")
	}
	mk := func(w float64) StadiumBorder { return NewStadiumBorder(StadiumBorderOpts{Side: widthSide(w)}) }
	c10, c15, c20 := mk(10), mk(15), mk(20)
	if !c10.Dimensions().Equal(NewEdgeInsetsAll(10)) || c10.Scale(2) != ShapeBorder(c20) || c20.Scale(0.5) != ShapeBorder(c10) {
		t.Error("dimensions or scale is wrong")
	}
	if ShapeBorderLerp(c10, c20, 0.5) != ShapeBorder(c15) {
		t.Error("lerp is wrong")
	}
	c1 := NewStadiumBorder(StadiumBorderOpts{Side: side(BorderSideOpts{})})
	if !isUnitCircle(c1.GetOuterPath(ui.NewRectFromCircle(ui.Offset{}, 1), ui.TextDirectionLTR)) || !isUnitCircle(c1.GetInnerPath(ui.NewRectFromCircle(ui.Offset{}, 2), ui.TextDirectionLTR)) {
		t.Error("the paths should be the unit circle")
	}
	rect := ui.NewRectFromLTRB(10, 20, 100, 200)
	paintedRRect := func(b StadiumBorder) ui.OpDrawRRect {
		ops := recordOps(func(c *ui.Canvas) { b.Paint(c, rect, ui.TextDirectionLTR) })
		if len(ops) != 1 {
			t.Fatalf("ops = %v", ops)
		}
		return ops[0].(ui.OpDrawRRect)
	}
	if d := paintedRRect(c10); d.RRect != ui.NewRRectFromRectAndRadius(rect.Deflate(5), ui.NewRadiusCircular(rect.ShortestSide()/2-5)) || d.Paint.StrokeWidth() != 10 {
		t.Errorf("inside paint = %v", d.RRect)
	}
	center := NewStadiumBorder(StadiumBorderOpts{Side: centerSide(10)})
	outside := NewStadiumBorder(StadiumBorderOpts{Side: outsideSide(10)})
	if !center.Dimensions().Equal(NewEdgeInsetsAll(5)) || !outside.Dimensions().Equal(EdgeInsetsZero) {
		t.Error("stroke align dimensions are wrong")
	}
	base := ui.NewRRectFromRectAndRadius(rect, ui.NewRadiusCircular(rect.ShortestSide()/2))
	if d := paintedRRect(center); d.RRect != base {
		t.Errorf("center paint = %v", d.RRect)
	}
	if d := paintedRRect(outside); d.RRect != base.Inflate(5) {
		t.Errorf("outside paint = %v", d.RRect)
	}
}

func TestBeveledAndContinuousRectangleBorders(t *testing.T) {
	rect := ui.NewRectFromLTRB(10, 20, 100, 200)
	bev := NewBeveledRectangleBorder(BeveledRectangleBorderOpts{Side: widthSide(2), BorderRadius: NewBorderRadiusCircular(20)})
	if b := bev.GetOuterPath(rect, ui.TextDirectionLTR).GetBounds(); !rectNear(b, rect) {
		t.Errorf("beveled bounds = %v", b)
	}
	if bev.GetOuterPath(rect, ui.TextDirectionLTR).Contains(ui.Offset{Dx: 11, Dy: 21}) {
		t.Error("the beveled corner should be cut")
	}
	if !bev.GetOuterPath(rect, ui.TextDirectionLTR).Contains(ui.Offset{Dx: 55, Dy: 110}) {
		t.Error("the beveled center should be inside")
	}
	cont := NewContinuousRectangleBorder(ContinuousRectangleBorderOpts{Side: widthSide(2), BorderRadius: NewBorderRadiusCircular(20)})
	if b := cont.GetOuterPath(rect, ui.TextDirectionLTR).GetBounds(); !rectNear(b, rect) {
		t.Errorf("continuous bounds = %v", b)
	}
	if !cont.Dimensions().Equal(NewEdgeInsetsAll(2)) || !bev.Dimensions().Equal(NewEdgeInsetsAll(2)) {
		t.Error("dimensions are wrong")
	}
	for name, mk := range map[string]func(w, r float64) ShapeBorder{
		"beveled": func(w, r float64) ShapeBorder {
			return NewBeveledRectangleBorder(BeveledRectangleBorderOpts{Side: widthSide(w), BorderRadius: NewBorderRadiusCircular(r)})
		},
		"continuous": func(w, r float64) ShapeBorder {
			return NewContinuousRectangleBorder(ContinuousRectangleBorderOpts{Side: widthSide(w), BorderRadius: NewBorderRadiusCircular(r)})
		},
	} {
		t.Run(name, func(t *testing.T) {
			c10, c15, c20 := mk(10, 10), mk(15, 15), mk(20, 20)
			if c10.Scale(2) != c20 || c20.Scale(0.5) != c10 {
				t.Error("scale is wrong")
			}
			if ShapeBorderLerp(c10, c20, 0.5) != c15 || ShapeBorderLerp(c10, c20, 0) != c10 || ShapeBorderLerp(c10, c20, 1) != c20 {
				t.Error("lerp is wrong")
			}
			ops := recordOps(func(c *ui.Canvas) { c10.Paint(c, rect, ui.TextDirectionLTR) })
			if len(ops) != 1 {
				t.Errorf("paint ops = %v", ops)
			}
			if got := recordOps(func(c *ui.Canvas) { c10.Paint(c, ui.Rect{}, ui.TextDirectionLTR) }); len(got) != 0 {
				t.Errorf("painting an empty rect recorded %v", got)
			}
		})
	}
}

func TestRoundedSuperellipseBorder(t *testing.T) {
	zero := NewRoundedSuperellipseBorder()
	rect := ui.NewRectFromLTRB(10, 20, 80, 190)
	if b := zero.GetOuterPath(rect, ui.TextDirectionLTR).GetBounds(); !rectNear(b, rect) {
		t.Errorf("zero bounds = %v", b)
	}
	mk := func(w, r float64) RoundedSuperellipseBorder {
		return NewRoundedSuperellipseBorder(RoundedSuperellipseBorderOpts{Side: widthSide(w), BorderRadius: NewBorderRadiusCircular(r)})
	}
	c10, c15, c20 := mk(10, 100), mk(15, 150), mk(20, 200)
	if c10.Scale(2) != ShapeBorder(c20) || c20.Scale(0.5) != ShapeBorder(c10) || ShapeBorderLerp(c10, c20, 0.5) != ShapeBorder(c15) {
		t.Error("scale or lerp is wrong")
	}
	if b := c10.GetOuterPath(rect, ui.TextDirectionLTR).GetBounds(); !rectNearTol(b, rect, 0.01) {
		t.Errorf("bounds = %v", b)
	}
	ops := recordOps(func(c *ui.Canvas) { mk(2, 10).Paint(c, rect, ui.TextDirectionLTR) })
	if len(ops) != 1 {
		t.Fatalf("ops = %v", ops)
	}
	if _, ok := ops[0].(ui.OpDrawRSuperellipse); !ok {
		t.Errorf("paint = %T", ops[0])
	}
	ops = recordOps(func(c *ui.Canvas) {
		NewRoundedSuperellipseBorder(RoundedSuperellipseBorderOpts{Side: widthSide(2)}).Paint(c, rect, ui.TextDirectionLTR)
	})
	if d, ok := ops[0].(ui.OpDrawRect); !ok || d.Rect != rect.Deflate(1) {
		t.Errorf("zero-radius paint = %v", ops[0])
	}
}

func TestStarBorder(t *testing.T) {
	b := NewStarBorder()
	if b.Points != 5 || b.InnerRadiusRatio() != 0.4 || b.PointRounding != 0 || b.ValleyRounding != 0 || b.Rotation() != 0 || b.Squash != 0 {
		t.Errorf("defaults = %v", b)
	}
	poly := NewStarBorderPolygon()
	if poly.Points != 5 || !near(poly.InnerRadiusRatio(), math.Cos(math.Pi/5), 1e-12) {
		t.Errorf("polygon defaults = %v", poly)
	}
	rect := ui.NewRectFromLTWH(0, 0, 100, 100)
	bounds := b.GetOuterPath(rect, ui.TextDirectionLTR).GetBounds()
	if bounds.Width() <= 0 || bounds.Height() <= 0 || !rect.Inflate(1).Contains(bounds.TopLeft()) || !rect.Inflate(1).Contains(bounds.BottomRight()) {
		t.Errorf("star bounds = %v", bounds)
	}
	if !b.GetOuterPath(rect, ui.TextDirectionLTR).Contains(ui.Offset{Dx: 50, Dy: 50}) {
		t.Error("the center of the star should be inside")
	}
	if b.GetOuterPath(rect, ui.TextDirectionLTR).Contains(ui.Offset{Dx: 2, Dy: 50}) {
		t.Error("a valley point of the star should be outside")
	}
	// lerps
	s10 := NewStarBorder(StarBorderOpts{Side: widthSide(10)})
	s20 := NewStarBorder(StarBorderOpts{Side: widthSide(20), Points: foundation.Some(7.0)})
	mid := ShapeBorderLerp(s10, s20, 0.5).(StarBorder)
	if mid.Side().Width != 15 || mid.Points != 6 {
		t.Errorf("mid = %v", mid)
	}
	if ShapeBorderLerp(s10, s20, 0) != ShapeBorder(s10) || ShapeBorderLerp(s10, s20, 1) != ShapeBorder(s20) {
		t.Error("lerp endpoints are wrong")
	}
	if got := s10.Scale(2).(StarBorder); got.Side().Width != 20 {
		t.Errorf("scale = %v", got)
	}
	if got := b.String(); foundation.DebugMode && got != "StarBorder(BorderSide(width: 0.0, style: none), points: 5.0, innerRadiusRatio: 0.4)" {
		t.Errorf("String = %q", got)
	}
	// A star lerps with the other outlined borders in two phases.
	for _, other := range []ShapeBorder{NewCircleBorder(), NewStadiumBorder(), NewRoundedRectangleBorder()} {
		for _, tm := range []float64{0.1, 0.5, 0.9} {
			if ShapeBorderLerp(b, other, tm) == nil || ShapeBorderLerp(other, b, tm) == nil {
				t.Errorf("lerp of star and %T at %v is nil", other, tm)
			}
		}
	}
	// fractional points
	if p := NewStarBorder(StarBorderOpts{Points: foundation.Some(5.5)}).GetOuterPath(rect, ui.TextDirectionLTR); p.GetBounds().Width() <= 0 {
		t.Error("a star with fractional points should have a path")
	}
	// polygon/star conversion by CopyWith
	if got := poly.CopyWith(StarBorderCopy{PointRounding: foundation.Some(0.25)}); got.PointRounding != 0.25 {
		t.Errorf("CopyWith = %v", got)
	}
}

func TestStarBorderDebugRanges(t *testing.T) {
	// Out-of-range arguments are covered in the debug tests; here the valid edges.
	_ = NewStarBorder(StarBorderOpts{Points: foundation.Some(2.0), InnerRadiusRatio: foundation.Some(1.0), PointRounding: 0.5, ValleyRounding: 0.5, Squash: 1})
}

func TestLinearBorder(t *testing.T) {
	if got := NewLinearBorder(); got != LinearBorderNone || (foundation.DebugMode && got.String() != "LinearBorder.none") {
		t.Errorf("none = %v", got)
	}
	lb := NewLinearBorderBottom(LinearBorderEdgeOnlyOpts{Side: widthSide(2), Size: foundation.Some(0.5), Alignment: 1})
	if lb.Bottom != foundation.Some(LinearBorderEdge{Size: 0.5, Alignment: 1}) || lb.Top.IsSome() {
		t.Errorf("bottom = %v", lb)
	}
	if got, want := lb.Dimensions(), NewEdgeInsetsDirectionalFromSTEB(0, 0, 0, 2); !got.Equal(want) {
		t.Errorf("Dimensions = %v, want %v", got, want)
	}
	start := NewLinearBorderStart(LinearBorderEdgeOnlyOpts{Side: widthSide(3)})
	if got, want := start.Dimensions().Resolve(ui.TextDirectionRTL), (EdgeInsets{Right: 3}); got != want {
		t.Errorf("start RTL dimensions = %v, want %v", got, want)
	}
	if got := start.String(); foundation.DebugMode && got != "LinearBorder(side: BorderSide(width: 3.0), start: LinearBorderEdge())" {
		t.Errorf("String = %q", got)
	}
	rect := ui.NewRectFromLTWH(10, 20, 100, 40)
	// Painting the bottom edge fills a thin rectangle along the bottom (like Flutter,
	// its x does not add the left of the rect).
	ops := recordOps(func(c *ui.Canvas) { lb.Paint(c, rect, ui.TextDirectionLTR) })
	if len(ops) != 1 {
		t.Fatalf("ops = %v", ops)
	}
	d := ops[0].(ui.OpDrawPath)
	if b := d.Path.GetBounds(); !rectNear(b, ui.NewRectFromLTWH(50, 58, 50, 2)) {
		t.Errorf("bottom edge = %v", b)
	}
	// lerps
	a := NewLinearBorderBottom(LinearBorderEdgeOnlyOpts{Side: widthSide(2)})
	b := NewLinearBorderBottom(LinearBorderEdgeOnlyOpts{Side: widthSide(4), Size: foundation.Some(0.5)})
	mid := ShapeBorderLerp(a, b, 0.5).(LinearBorder)
	if mid.Side().Width != 3 || mid.Bottom.Value().Size != 0.75 {
		t.Errorf("mid = %v", mid)
	}
	if got := LinearBorderEdgeLerp(foundation.None[LinearBorderEdge](), foundation.Some(LinearBorderEdge{Size: 1, Alignment: 1}), 0.5).Value(); got != (LinearBorderEdge{Size: 0.5, Alignment: 1}) {
		t.Errorf("edge lerp from nothing = %v", got)
	}
	if got := (LinearBorderEdge{Size: 0.5, Alignment: 1}).String(); got != "LinearBorderEdge(size: 0.5, alignment: 1.0)" {
		t.Errorf("edge String = %q", got)
	}
}
