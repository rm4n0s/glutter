package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

func colored(c uint32, width ...float64) painting.BorderSide {
	o := painting.BorderSideOpts{Color: foundation.Some(ui.NewColor(c))}
	if len(width) > 0 {
		o.Width = foundation.Some(width[0])
	}
	return painting.NewBorderSide(o)
}

func TestTableBorderConstructor(t *testing.T) {
	border := rendering.NewTableBorder(rendering.TableBorder{
		Left: painting.NewBorderSide(), Right: colored(0xFF00FF00), VerticalInside: painting.NewBorderSide()})
	if border.Top != painting.BorderSideNone || border.Bottom != painting.BorderSideNone || border.HorizontalInside != painting.BorderSideNone {
		t.Errorf("unset sides are not BorderSideNone: %v %v %v", border.Top, border.Bottom, border.HorizontalInside)
	}
	if border.Right != colored(0xFF00FF00) || border.Left != painting.NewBorderSide() {
		t.Errorf("sides = %v / %v", border.Right, border.Left)
	}
	if got, want := border.Dimensions(), painting.NewEdgeInsetsSymmetric(0, 1); got != want {
		t.Errorf("dimensions = %v, want %v", got, want)
	}
	if border.IsUniform() {
		t.Error("the border is uniform")
	}
	want := rendering.NewTableBorder(rendering.TableBorder{
		Left:           painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(2.0)}),
		Right:          painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(2.0), Color: foundation.Some(ui.NewColor(0xFF00FF00))}),
		VerticalInside: painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(2.0)}),
	})
	if got := border.Scale(2); got != want {
		t.Errorf("scale(2) = %v, want %v", got, want)
	}
}

func TestTableBorderAll(t *testing.T) {
	cyan := ui.NewColor(0xFF00FFFF)
	border := rendering.NewTableBorderAll(rendering.TableBorderAllOpts{Width: foundation.Some(2.0), Color: foundation.Some(cyan)})
	side := painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(2.0), Color: foundation.Some(cyan)})
	for name, got := range map[string]painting.BorderSide{"top": border.Top, "right": border.Right, "bottom": border.Bottom,
		"left": border.Left, "horizontalInside": border.HorizontalInside, "verticalInside": border.VerticalInside} {
		if got != side {
			t.Errorf("%s = %v, want %v", name, got, side)
		}
	}
	if got, want := border.Dimensions(), painting.NewEdgeInsetsSymmetric(2, 2); got != want {
		t.Errorf("dimensions = %v, want %v", got, want)
	}
	if !border.IsUniform() {
		t.Error("TableBorder.all is not uniform")
	}
	if got, want := border.Scale(0.5), rendering.NewTableBorderAll(rendering.TableBorderAllOpts{Color: foundation.Some(cyan)}); got != want {
		t.Errorf("scale(0.5) = %v, want %v", got, want)
	}
	radius := painting.NewBorderRadiusAll(ui.NewRadiusCircular(8))
	if got := rendering.NewTableBorderAll(rendering.TableBorderAllOpts{BorderRadius: radius}).BorderRadius; got != radius {
		t.Errorf("border radius = %v", got)
	}
}

func TestTableBorderSymmetric(t *testing.T) {
	red := ui.NewColor(0xFFFF0000)
	inside := painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(3.0)})
	outside := painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(red)})
	border := rendering.NewTableBorderSymmetric(inside, outside, painting.BorderRadiusZero)
	if border.Top != outside || border.Right != outside || border.Bottom != outside || border.Left != outside {
		t.Errorf("outer sides = %v", border)
	}
	if border.HorizontalInside != inside || border.VerticalInside != inside {
		t.Errorf("inner sides = %v", border)
	}
	if got, want := border.Dimensions(), painting.NewEdgeInsetsSymmetric(1, 1); got != want {
		t.Errorf("dimensions = %v, want %v", got, want)
	}
	if border.IsUniform() {
		t.Error("the border is uniform")
	}
	scaled := painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(0.0), Color: foundation.Some(red), Style: painting.BorderStyleNone})
	if got, want := border.Scale(0), rendering.NewTableBorderSymmetric(painting.BorderSideNone, scaled, painting.BorderRadiusZero); got != want {
		t.Errorf("scale(0) = %v, want %v", got, want)
	}
}

func TestTableBorderLerp(t *testing.T) {
	sides := make([]painting.BorderSide, 6)
	for i := range sides {
		w := float64(i + 1)
		sides[i] = painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(w), Color: foundation.Some(ui.NewColor(uint32(i + 1)))})
	}
	sides[0] = painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(ui.NewColor(1))})
	mk := func(k float64) rendering.TableBorder {
		return rendering.TableBorder{Top: sides[0].Scale(k), Right: sides[1].Scale(k), Bottom: sides[2].Scale(k),
			Left: sides[3].Scale(k), HorizontalInside: sides[4].Scale(k), VerticalInside: sides[5].Scale(k)}
	}
	a, b, c := mk(1), mk(2), mk(3)
	if a.IsUniform() {
		t.Error("tableA is uniform")
	}
	if got, want := a.Dimensions(), painting.NewEdgeInsetsFromLTRB(4, 1, 2, 3); got != want {
		t.Errorf("dimensions = %v, want %v", got, want)
	}
	if got, want := b.Dimensions(), painting.NewEdgeInsetsFromLTRB(4, 1, 2, 3).Mul(2); got != want {
		t.Errorf("dimensions(b) = %v, want %v", got, want)
	}
	for _, tc := range []struct {
		name         string
		from, to, at *rendering.TableBorder
		t            float64
		want         rendering.TableBorder
	}{
		{"midpoint", &a, &c, nil, 0.5, b},
		{"extrapolate up", &a, &b, nil, 2, c},
		{"extrapolate down", &b, &c, nil, -1, a},
	} {
		got := rendering.TableBorderLerp(tc.from, tc.to, tc.t)
		if got == nil || *got != tc.want {
			t.Errorf("%s: lerp = %v, want %v", tc.name, got, tc.want)
		}
	}
	lerped := rendering.TableBorderLerp(&a, &c, 0.9195)
	if lerped.IsUniform() {
		t.Error("the interpolated border is uniform")
	}
	want := painting.EdgeInsetsLerp(ptr(a.Dimensions()), ptr(c.Dimensions()), 0.9195)
	if got := lerped.Dimensions(); got != *want {
		t.Errorf("interpolated dimensions = %v, want %v", got, *want)
	}
}

func ptr[T any](v T) *T { return &v }

func TestTableBorderLerpIdenticalAndNulls(t *testing.T) {
	if rendering.TableBorderLerp(nil, nil, 0) != nil {
		t.Error("lerp(null, null) is not null")
	}
	border := rendering.NewTableBorder(rendering.TableBorder{})
	if got := rendering.TableBorderLerp(&border, &border, 0.5); got != &border {
		t.Error("lerp of identical borders is not the same object")
	}
	two := rendering.NewTableBorderAll(rendering.TableBorderAllOpts{Width: foundation.Some(2.0)})
	one := rendering.NewTableBorderAll()
	if got := rendering.TableBorderLerp(&two, nil, 0.5); got == nil || *got != one {
		t.Errorf("lerp(table2, null) = %v, want %v", got, one)
	}
	if got := rendering.TableBorderLerp(nil, &two, 0.5); got == nil || *got != one {
		t.Errorf("lerp(null, table2) = %v, want %v", got, one)
	}
}

func TestTableBorderObjectAPI(t *testing.T) {
	none, zero := painting.BorderSideNone.String(), painting.BorderRadiusZero.String()
	want := "TableBorder(" + none + ", " + none + ", " + none + ", " + none + ", " + none + ", " + none + ", " + zero + ")"
	if got := (rendering.TableBorder{}).String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if rendering.NewTableBorder(rendering.TableBorder{}) == rendering.NewTableBorder(rendering.TableBorder{Top: painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(0.0)})}) {
		t.Error("different borders are equal")
	}
}

func TestTableBorderOuterUniformityAndRadius(t *testing.T) {
	blue := ui.NewColor(0xFF0000FF)
	uniform := rendering.TableBorder{
		Top: painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(2.0)}), Right: painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(2.0)}),
		Bottom: painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(2.0)}), Left: painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(2.0)}),
		HorizontalInside: colored(0xFF0000FF), VerticalInside: colored(0xFF0000FF),
	}
	if uniform.IsUniform() {
		t.Error("a border with different inside lines is uniform")
	}
	if uniform.Right != uniform.Top || uniform.Bottom != uniform.Top || uniform.Left != uniform.Top {
		t.Error("the outer sides differ")
	}
	// A non-uniform-width border with a uniform color is painted with its radius.
	radius := painting.NewBorderRadiusAll(ui.NewRadiusCircular(8))
	border := rendering.TableBorder{Top: colored(0xFF0000FF, 3), Right: painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(blue)}),
		Bottom: colored(0xFF0000FF, 2), Left: colored(0xFF0000FF, 1.5), BorderRadius: radius}
	canvasOps := func(b rendering.TableBorder) []any {
		rec := ui.NewPictureRecorder()
		canvas := ui.NewCanvas(rec)
		b.Paint(canvas, ui.NewRectFromLTWH(0, 0, 100, 50), nil, nil)
		return displayOps(rec.EndRecording())
	}
	ops := canvasOps(border)
	if countOps[ui.OpDrawDRRect](ops) != 1 || countOps[ui.OpDrawPath](ops) != 0 {
		t.Errorf("a rounded border with uniform colors was drawn with %d DRRects and %d paths, want 1 and 0",
			countOps[ui.OpDrawDRRect](ops), countOps[ui.OpDrawPath](ops))
	}
	square := border
	square.BorderRadius = painting.BorderRadiusZero
	if ops := canvasOps(square); countOps[ui.OpDrawPath](ops) != 4 {
		t.Errorf("a square border was drawn with %d paths, want 4", countOps[ui.OpDrawPath](ops))
	}
	mixed := border
	mixed.Right = colored(0xFF00FF00, 1)
	if ops := canvasOps(mixed); countOps[ui.OpDrawPath](ops) != 4 {
		t.Errorf("a border with two colors was drawn with %d paths, want 4", countOps[ui.OpDrawPath](ops))
	}
}

func TestTableBorderInsideLines(t *testing.T) {
	border := rendering.NewTableBorderAll()
	rec := ui.NewPictureRecorder()
	border.Paint(ui.NewCanvas(rec), ui.NewRectFromLTWH(0, 0, 100, 50), []float64{10, 20}, []float64{30})
	ops := displayOps(rec.EndRecording())
	// Two for the inside lines and four for the outer sides.
	if got := countOps[ui.OpDrawPath](ops); got != 6 {
		t.Errorf("paths = %d, want 6", got)
	}
}
