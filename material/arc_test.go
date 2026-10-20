package material_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
)

// These tests are ported from packages/flutter/test/material/arc_test.dart (the
// "has one line description" matcher is the String of the tweens).

func near(a, b ui.Offset, distance float64) bool { return a.Sub(b).Distance() <= distance }

func sameRect(a, b ui.Rect) bool {
	return math.Abs(a.Left-b.Left) < 2 && math.Abs(a.Top-b.Top) < 2 && math.Abs(a.Right-b.Right) < 2 && math.Abs(a.Bottom-b.Bottom) < 2
}

func TestMaterialArcControl(t *testing.T) {
	pointA := NewMaterialPointArcTween(ui.OffsetZero, ui.Offset{Dx: 0, Dy: 10})
	pointB := NewMaterialPointArcTween(ui.OffsetZero, ui.Offset{Dx: 0, Dy: 10})
	rectA := NewMaterialRectArcTween(ui.NewRectFromLTWH(0, 0, 10, 10), ui.NewRectFromLTWH(0, 10, 10, 10))
	rectB := NewMaterialRectArcTween(ui.NewRectFromLTWH(0, 0, 10, 10), ui.NewRectFromLTWH(0, 10, 10, 10))
	centerA := NewMaterialRectCenterArcTween(ui.NewRectFromLTWH(0, 0, 10, 10), ui.NewRectFromLTWH(0, 10, 10, 10))
	centerB := NewMaterialRectCenterArcTween(ui.NewRectFromLTWH(0, 0, 10, 10), ui.NewRectFromLTWH(0, 10, 10, 10))
	for name, pair := range map[string][2]string{
		"point":  {pointA.String(), pointB.String()},
		"rect":   {rectA.String(), rectB.String()},
		"center": {centerA.String(), centerB.String()},
	} {
		if pair[0] != pair[1] || pair[0] == "" {
			t.Errorf("%s: %q and %q should be equal descriptions", name, pair[0], pair[1])
		}
		for _, c := range pair[0] {
			if c == '\n' {
				t.Errorf("%s: the description %q has more than one line", name, pair[0])
			}
		}
	}
}

func TestOnAxisMaterialPointArcTween(t *testing.T) {
	tests := []struct {
		name     string
		end      ui.Offset
		wantHalf ui.Offset
	}{
		{"vertical", ui.Offset{Dx: 0, Dy: 10}, ui.Offset{Dx: 0, Dy: 5}},
		{"horizontal", ui.Offset{Dx: 10, Dy: 0}, ui.Offset{Dx: 5, Dy: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tween := NewMaterialPointArcTween(ui.OffsetZero, tt.end)
			if got := tween.Lerp(0.5); got != tt.wantHalf {
				t.Errorf("Lerp(0.5) = %v, want %v", got, tt.wantHalf)
			}
			if tween.Center().IsSome() || tween.Radius().IsSome() || tween.BeginAngle().IsSome() {
				t.Error("an on-axis tween has no arc")
			}
		})
	}
}

func TestOnAxisMaterialRectArcTween(t *testing.T) {
	tests := []struct {
		name     string
		end      ui.Rect
		wantHalf ui.Rect
	}{
		{"vertical", ui.NewRectFromLTWH(0, 10, 10, 10), ui.NewRectFromLTWH(0, 5, 10, 10)},
		{"horizontal", ui.NewRectFromLTWH(10, 0, 10, 10), ui.NewRectFromLTWH(5, 0, 10, 10)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tween := NewMaterialRectArcTween(ui.NewRectFromLTWH(0, 0, 10, 10), tt.end)
			if got := tween.Lerp(0.5); got != tt.wantHalf {
				t.Errorf("Lerp(0.5) = %v, want %v", got, tt.wantHalf)
			}
		})
	}
}

func TestMaterialPointArcTween(t *testing.T) {
	begin, end := ui.Offset{Dx: 180, Dy: 110}, ui.Offset{Dx: 37, Dy: 250}
	tests := []struct {
		name       string
		from, to   ui.Offset
		quarter    ui.Offset
		threeQuart ui.Offset
	}{
		{"forward", begin, end, ui.Offset{Dx: 126, Dy: 120}, ui.Offset{Dx: 48, Dy: 196}},
		{"backward", end, begin, ui.Offset{Dx: 91, Dy: 239}, ui.Offset{Dx: 168.3, Dy: 163.8}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tween := NewMaterialPointArcTween(tt.from, tt.to)
			if tween.Lerp(0) != tt.from || tween.Lerp(1) != tt.to {
				t.Errorf("the ends are %v and %v", tween.Lerp(0), tween.Lerp(1))
			}
			if got := tween.Lerp(0.25); !near(got, tt.quarter, 2) {
				t.Errorf("Lerp(0.25) = %v, want near %v", got, tt.quarter)
			}
			if got := tween.Lerp(0.75); !near(got, tt.threeQuart, 2) {
				t.Errorf("Lerp(0.75) = %v, want near %v", got, tt.threeQuart)
			}
			if !tween.Center().IsSome() || !tween.Radius().IsSome() || !tween.BeginAngle().IsSome() || !tween.EndAngle().IsSome() {
				t.Error("a tween that is off-axis has an arc")
			}
		})
	}
}

func TestMaterialRectArcTween(t *testing.T) {
	begin := ui.NewRectFromLTRB(180, 100, 330, 200)
	end := ui.NewRectFromLTRB(32, 275, 132, 425)
	tests := []struct {
		name       string
		from, to   ui.Rect
		quarter    ui.Rect
		threeQuart ui.Rect
	}{
		{"forward", begin, end, ui.NewRectFromLTRB(120, 113, 259, 237), ui.NewRectFromLTRB(42.3, 206.5, 153.5, 354.7)},
		{"backward", end, begin, ui.NewRectFromLTRB(92, 262, 203, 388), ui.NewRectFromLTRB(169.7, 168.5, 308.5, 270.3)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tween := NewMaterialRectArcTween(tt.from, tt.to)
			if tween.Lerp(0) != tt.from || tween.Lerp(1) != tt.to {
				t.Errorf("the ends are %v and %v", tween.Lerp(0), tween.Lerp(1))
			}
			if got := tween.Lerp(0.25); !sameRect(got, tt.quarter) {
				t.Errorf("Lerp(0.25) = %v, want near %v", got, tt.quarter)
			}
			if got := tween.Lerp(0.75); !sameRect(got, tt.threeQuart) {
				t.Errorf("Lerp(0.75) = %v, want near %v", got, tt.threeQuart)
			}
			if tween.BeginArc() == nil || tween.EndArc() == nil {
				t.Error("the tween has two arcs")
			}
		})
	}
}

func TestMaterialRectCenterArcTween(t *testing.T) {
	begin := ui.NewRectFromLTRB(180, 100, 330, 200)
	end := ui.NewRectFromLTRB(32, 275, 132, 425)
	tween := NewMaterialRectCenterArcTween(begin, end)
	if tween.Lerp(0) != begin || tween.Lerp(1) != end {
		t.Fatalf("the ends are %v and %v", tween.Lerp(0), tween.Lerp(1))
	}
	got := tween.Lerp(0.5)
	// The size is the linear interpolation of the sizes.
	wantW, wantH := (150.0+100.0)/2, (100.0+150.0)/2
	if math.Abs(got.Width()-wantW) > 1e-9 || math.Abs(got.Height()-wantH) > 1e-9 {
		t.Errorf("size at 0.5 = %v x %v, want %v x %v", got.Width(), got.Height(), wantW, wantH)
	}
	// The center follows the arc of the centers.
	if want := tween.CenterArc().Lerp(0.5); !near(got.Center(), want, 1e-9) {
		t.Errorf("center at 0.5 = %v, want %v", got.Center(), want)
	}
	if tween.CenterArc().Lerp(0.5) == begin.Center().Lerp(end.Center(), 0.5) {
		t.Error("the center should not move on the straight line")
	}
}

func TestMaterialArcTweensAsHeroTweens(t *testing.T) {
	begin := ui.NewRectFromLTRB(180, 100, 330, 200)
	end := ui.NewRectFromLTRB(32, 275, 132, 425)
	for name, tween := range map[string]interface {
		Transform(float64) foundation.Opt[ui.Rect]
	}{
		"rect":   NewMaterialRectArcTween(begin, end).Tween(),
		"center": NewMaterialRectCenterArcTween(begin, end).Tween(),
		"create": MaterialRectArcTweenCreate(foundation.Some(begin), foundation.Some(end)),
	} {
		if tween.Transform(0).Value() != begin || tween.Transform(1).Value() != end {
			t.Errorf("%s: the ends are %v and %v", name, tween.Transform(0), tween.Transform(1))
		}
		if tween.Transform(0.5).Value() == begin.Lerp(end, 0.5) {
			t.Errorf("%s: the middle is the straight one", name)
		}
	}
	// A rectangle that is none counts as zero when the other one is some.
	none := MaterialRectArcTweenCreate(foundation.None[ui.Rect](), foundation.None[ui.Rect]())
	if none.Transform(0.5).IsSome() {
		t.Error("two none rectangles give none")
	}
	if one := MaterialRectArcTweenCreate(foundation.None[ui.Rect](), foundation.Some(end)); !one.Transform(0.5).IsSome() {
		t.Error("one rectangle that is none counts as zero")
	}
}
