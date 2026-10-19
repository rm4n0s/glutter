package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func rad(x, y float64) ui.Radius { return ui.NewRadiusElliptical(x, y) }

func TestBorderRadiusControl(t *testing.T) {
	rect := ui.NewRectFromLTRB(19, 23, 29, 31)
	r1, r2 := rad(89, 87), rad(103, 107)
	tests := []struct {
		name string
		br   BorderRadius
		want ui.RRect
	}{
		{"all", NewBorderRadiusAll(rad(5, 7)), ui.NewRRectFromRectXY(rect, 5, 7)},
		{"circular", NewBorderRadiusCircular(3), ui.NewRRectFromRectXY(rect, 3, 3)},
		{"vertical", NewBorderRadiusVertical(r1, r2), ui.NewRRectFromRectAndCorners(rect, ui.Corners{TopLeft: r1, TopRight: r1, BottomLeft: r2, BottomRight: r2})},
		{"horizontal", NewBorderRadiusHorizontal(r1, r2), ui.NewRRectFromRectAndCorners(rect, ui.Corners{TopLeft: r1, TopRight: r2, BottomLeft: r1, BottomRight: r2})},
		{"zero", BorderRadiusZero, ui.NewRRectFromRectAndCorners(rect, ui.Corners{})},
		{"only", BorderRadius{TopRight: r1, BottomRight: r2}, ui.NewRRectFromRectAndCorners(rect, ui.Corners{TopRight: r1, BottomRight: r2})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.br.ToRRect(rect); got != tt.want {
				t.Errorf("ToRRect = %v, want %v", got, tt.want)
			}
		})
	}
	if b := NewBorderRadiusVertical(r1, r2); b.TopLeft != r1 || b.TopRight != r1 || b.BottomLeft != r2 || b.BottomRight != r2 {
		t.Errorf("vertical corners = %v", b)
	}
	if b := NewBorderRadiusHorizontal(r1, r2); b.TopLeft != r1 || b.TopRight != r2 || b.BottomLeft != r1 || b.BottomRight != r2 {
		t.Errorf("horizontal corners = %v", b)
	}
}

func TestBorderRadiusOperators(t *testing.T) {
	r1, r2 := rad(89, 87), rad(103, 107)
	tests := []struct {
		name      string
		got, want BorderRadiusGeometry
	}{
		{"subtract", BorderRadius{TopLeft: rad(1, 2)}.Subtract(BorderRadius{TopLeft: rad(3, 5)}), BorderRadius{TopLeft: rad(-2, -3)}},
		{"add", BorderRadius{TopRight: rad(1, 2)}.Add(BorderRadius{TopLeft: rad(3, 5)}), BorderRadius{TopLeft: rad(3, 5), TopRight: rad(1, 2)}},
		{"minus", BorderRadius{TopLeft: rad(1, 2)}.Minus(BorderRadius{TopLeft: rad(3, 5)}), BorderRadius{TopLeft: rad(-2, -3)}},
		{"plus", BorderRadius{TopRight: rad(1, 2)}.Plus(BorderRadius{TopLeft: rad(3, 5)}), BorderRadius{TopLeft: rad(3, 5), TopRight: rad(1, 2)}},
		{"neg", BorderRadius{TopLeft: rad(1, 2)}.Neg(), BorderRadius{TopLeft: rad(-1, -2)}},
		{"mul zero", BorderRadius{TopLeft: r1, TopRight: r2, BottomLeft: r2, BottomRight: r1}.Mul(0), BorderRadiusZero},
		{"div", NewBorderRadiusCircular(15).Div(10), NewBorderRadiusCircular(1.5)},
		{"truncDiv", NewBorderRadiusCircular(15).TruncDiv(10), NewBorderRadiusCircular(1)},
		{"mod", NewBorderRadiusCircular(15).Mod(10), NewBorderRadiusCircular(5)},
		// the directional versions
		{"directional subtract", BorderRadiusDirectional{TopStart: rad(1, 2)}.Subtract(BorderRadiusDirectional{TopStart: rad(3, 5)}), BorderRadiusDirectional{TopStart: rad(-2, -3)}},
		{"directional add", BorderRadiusDirectional{TopEnd: rad(1, 2)}.Add(BorderRadiusDirectional{TopStart: rad(3, 5)}), BorderRadiusDirectional{TopStart: rad(3, 5), TopEnd: rad(1, 2)}},
		{"directional neg", BorderRadiusDirectional{TopStart: rad(1, 2)}.Neg(), BorderRadiusDirectional{TopStart: rad(-1, -2)}},
		{"directional mul zero", BorderRadiusDirectional{TopStart: r1, TopEnd: r2, BottomStart: r2, BottomEnd: r1}.Mul(0), BorderRadiusDirectionalZero},
		{"directional div", NewBorderRadiusDirectionalCircular(15).Div(10), NewBorderRadiusDirectionalCircular(1.5)},
		{"directional truncDiv", NewBorderRadiusDirectionalCircular(15).TruncDiv(10), NewBorderRadiusDirectionalCircular(1)},
		{"directional mod", NewBorderRadiusDirectionalCircular(15).Mod(10), NewBorderRadiusDirectionalCircular(5)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.got.Equal(tt.want) {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestBorderRadiusDirectionalResolve(t *testing.T) {
	rect := ui.NewRectFromLTRB(19, 23, 29, 31)
	r1, r2 := rad(89, 87), rad(103, 107)
	corners := func(tl, tr, bl, br ui.Radius) ui.RRect {
		return ui.NewRRectFromRectAndCorners(rect, ui.Corners{TopLeft: tl, TopRight: tr, BottomLeft: bl, BottomRight: br})
	}
	tests := []struct {
		name     string
		br       BorderRadiusDirectional
		ltr, rtl ui.RRect
	}{
		{"all", NewBorderRadiusDirectionalAll(rad(5, 7)), ui.NewRRectFromRectXY(rect, 5, 7), ui.NewRRectFromRectXY(rect, 5, 7)},
		{"circular", NewBorderRadiusDirectionalCircular(3), ui.NewRRectFromRectXY(rect, 3, 3), ui.NewRRectFromRectXY(rect, 3, 3)},
		{"vertical", NewBorderRadiusDirectionalVertical(r1, r2), corners(r1, r1, r2, r2), corners(r1, r1, r2, r2)},
		{"horizontal", NewBorderRadiusDirectionalHorizontal(r1, r2), corners(r1, r2, r1, r2), corners(r2, r1, r2, r1)},
		{"zero", BorderRadiusDirectionalZero, corners(ui.Radius{}, ui.Radius{}, ui.Radius{}, ui.Radius{}), corners(ui.Radius{}, ui.Radius{}, ui.Radius{}, ui.Radius{})},
		{
			"only end", BorderRadiusDirectional{TopEnd: r1, BottomEnd: r2},
			corners(ui.Radius{}, r1, ui.Radius{}, r2), corners(r1, ui.Radius{}, r2, ui.Radius{}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.br.Resolve(ui.TextDirectionLTR).ToRRect(rect); got != tt.ltr {
				t.Errorf("ltr = %v, want %v", got, tt.ltr)
			}
			if got := tt.br.Resolve(ui.TextDirectionRTL).ToRRect(rect); got != tt.rtl {
				t.Errorf("rtl = %v, want %v", got, tt.rtl)
			}
		})
	}
}

func TestBorderRadiusLerp(t *testing.T) {
	a, b := NewBorderRadiusAll(ui.NewRadiusCircular(10)), NewBorderRadiusAll(ui.NewRadiusCircular(20))
	got := BorderRadiusLerp(&a, &b, 0.25)
	for name, want := range map[string]BorderRadius{
		"a*1.25": a.Mul(1.25), "b*0.625": b.Mul(0.625), "a+2.5": a.Plus(NewBorderRadiusCircular(2.5)), "b-7.5": b.Minus(NewBorderRadiusCircular(7.5)),
	} {
		if *got != want {
			t.Errorf("%s: got %v, want %v", name, *got, want)
		}
	}
	if BorderRadiusLerp(nil, nil, 0.25) != nil {
		t.Error("lerp(nil, nil) should be nil")
	}
	if g := BorderRadiusLerp(nil, &b, 0.25); *g != b.Mul(0.25) {
		t.Errorf("lerp(nil, b) = %v", *g)
	}
	if g := BorderRadiusLerp(&a, nil, 0.25); *g != a.Mul(0.75) {
		t.Errorf("lerp(a, nil) = %v", *g)
	}
	z := BorderRadiusZero
	if BorderRadiusLerp(&z, &z, 0.5) != &z {
		t.Error("lerp of identical values should return the same pointer")
	}

	crazyA := BorderRadius{TopLeft: rad(10, 20), TopRight: rad(30, 40), BottomLeft: rad(50, 60)}
	crazyB := BorderRadius{TopRight: rad(100, 110), BottomLeft: rad(120, 130), BottomRight: rad(140, 150)}
	crazyC := BorderRadius{TopLeft: rad(5, 10), TopRight: rad(65, 75), BottomLeft: rad(85, 95), BottomRight: rad(70, 75)}
	if g := BorderRadiusLerp(&crazyA, &crazyB, 0.5); *g != crazyC {
		t.Errorf("crazy lerp = %v, want %v", *g, crazyC)
	}
}

func TestBorderRadiusGeometryLerp(t *testing.T) {
	mixed1 := BorderRadius{TopLeft: rad(10, 20)}.Add(BorderRadiusDirectional{TopStart: rad(30, 50)})
	mixed2 := BorderRadius{TopLeft: rad(70, 110)}.Add(BorderRadiusDirectional{TopStart: rad(130, 170)})
	mixed3 := BorderRadius{TopLeft: rad(25, 42.5)}.Add(BorderRadiusDirectional{TopStart: rad(55, 80)})
	for _, d := range textDirections {
		if got := BorderRadiusGeometryLerp(mixed1, mixed2, 0.0).Resolve(d); got != mixed1.Resolve(d) {
			t.Errorf("t=0 %v: %v", d, got)
		}
		if got := BorderRadiusGeometryLerp(mixed1, mixed2, 1.0).Resolve(d); got != mixed2.Resolve(d) {
			t.Errorf("t=1 %v: %v", d, got)
		}
		if got := BorderRadiusGeometryLerp(mixed1, mixed2, 0.25).Resolve(d); got != mixed3.Resolve(d) {
			t.Errorf("t=.25 %v: %v want %v", d, got, mixed3.Resolve(d))
		}
	}
	if BorderRadiusGeometryLerp(nil, nil, 0) != nil {
		t.Error("lerp(nil, nil) should be nil")
	}
}

func TestBorderRadiusCopyWith(t *testing.T) {
	src := BorderRadius{TopLeft: rad(1, 1), TopRight: rad(2, 2), BottomLeft: rad(3, 3), BottomRight: rad(4, 4)}
	got := src.CopyWith(BorderRadiusCopy{TopLeft: foundation.Some(rad(9, 9))})
	want := BorderRadius{TopLeft: rad(9, 9), TopRight: rad(2, 2), BottomLeft: rad(3, 3), BottomRight: rad(4, 4)}
	if got != want {
		t.Errorf("CopyWith = %v, want %v", got, want)
	}
}

func TestBorderRadiusString(t *testing.T) {
	tests := []struct {
		got  BorderRadiusGeometry
		want string
	}{
		{BorderRadiusZero, "BorderRadius.zero"},
		{NewBorderRadiusCircular(5), "BorderRadius.circular(5.0)"},
		{NewBorderRadiusAll(rad(5, 7)), "BorderRadius.all(Radius.elliptical(5.0, 7.0))"},
		{BorderRadiusDirectionalZero, "BorderRadius.zero"},
	}
	for _, tt := range tests {
		if got := tt.got.String(); got != tt.want {
			t.Errorf("String = %q, want %q", got, tt.want)
		}
	}
}
