package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

var (
	magenta     = ui.NewColor(0xFFFF00FF)
	magenta3    = side(BorderSideOpts{Color: foundation.Some(magenta), Width: foundation.Some(3.0)})
	magenta6    = side(BorderSideOpts{Color: foundation.Some(magenta), Width: foundation.Some(6.0)})
	yellow2     = side(BorderSideOpts{Color: foundation.Some(someYellow), Width: foundation.Some(2.0)})
	yellowNone0 = side(BorderSideOpts{Color: foundation.Some(someYellow), Width: foundation.Some(0.0), Style: BorderStyleNone})
)

func widthOf(w float64) BorderSide { return sideW(w) }

func TestBorderConstructors(t *testing.T) {
	s := side(BorderSideOpts{})
	b := NewBorderFromBorderSide(s)
	if b.Left != s || b.Top != s || b.Right != s || b.Bottom != s {
		t.Errorf("fromBorderSide = %v", b)
	}
	s1 := side(BorderSideOpts{Color: foundation.Some(ui.NewColor(0xFFFFFFFF))})
	sym := NewBorderSymmetric(s1, s)
	if sym.Left != s1 || sym.Top != s || sym.Right != s1 || sym.Bottom != s {
		t.Errorf("symmetric = %v", sym)
	}
	if all := NewBorderAll(BorderSideOpts{Width: foundation.Some(10.0)}); all.Dimensions().(EdgeInsets) != NewEdgeInsetsAll(10) {
		t.Errorf("all = %v", all)
	}
}

func TestBorderMergeAndAdd(t *testing.T) {
	tests := []struct {
		name       string
		a, b, want Border
	}{
		{"disjoint", Border{Top: yellow2}, Border{Right: magenta3}, Border{Top: yellow2, Right: magenta3}},
		{"same side", Border{Bottom: magenta3}, Border{Bottom: magenta3}, Border{Bottom: magenta6}},
		{"none side", Border{Left: magenta3, Right: yellowNone0}, Border{Right: yellow2}, Border{Left: magenta3, Right: yellow2}},
		{"empty", Border{}, Border{}, Border{}},
		{"b3 + b3", Border{Top: magenta3}, Border{Top: magenta3}, Border{Top: magenta6}},
		{"none + none", Border{Top: yellowNone0}, Border{Top: yellowNone0}, Border{}},
		{"zero + none", Border{}, Border{Top: yellowNone0}, Border{}},
		{"none + zero", Border{Top: yellowNone0}, Border{}, Border{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BorderMerge(tt.a, tt.b); got.normalized() != tt.want.normalized() {
				t.Errorf("BorderMerge = %v, want %v", got, tt.want)
			}
			got, ok := ShapeBorderPlus(tt.a, tt.b).(Border)
			if !ok {
				t.Fatalf("a + b is a %T, want Border", ShapeBorderPlus(tt.a, tt.b))
			}
			if got.normalized() != tt.want.normalized() {
				t.Errorf("a + b = %v, want %v", got, tt.want)
			}
		})
	}
	if _, ok := ShapeBorderPlus(Border{Left: magenta3}, Border{Left: yellow2}).(Border); ok {
		t.Error("borders that can't merge should add up to a compound border, not a Border")
	}
}

// normalized turns the zero sides into BorderSideNone so that borders compare
// like Dart's.
func (b Border) normalized() Border {
	return Border{b.Top.normalize(), b.Right.normalize(), b.Bottom.normalize(), b.Left.normalize()}
}

func TestBorderScale(t *testing.T) {
	if got := (Border{Left: magenta3}).Scale(2.0).(Border); got != (Border{Left: magenta6}).normalized() {
		t.Errorf("scale 2 = %v", got)
	}
	bY0 := Border{Top: yellowNone0}
	if got := bY0.Scale(3.0).(Border); got.Top != yellowNone0 {
		t.Errorf("scale of none = %v", got)
	}
	if got := (Border{Top: yellow2}).Scale(0).(Border); got.Top != yellowNone0 {
		t.Errorf("scale 0 = %v", got.Top)
	}
}

func TestBorderDimensions(t *testing.T) {
	inside := widthOf(10)
	center := side(BorderSideOpts{Width: foundation.Some(10.0), StrokeAlign: foundation.Some(StrokeAlignCenter)})
	outside := side(BorderSideOpts{Width: foundation.Some(10.0), StrokeAlign: foundation.Some(StrokeAlignOutside)})
	all := func(s BorderSide) Border { return NewBorderFromBorderSide(s) }
	dall := func(s BorderSide) BorderDirectional { return BorderDirectional{Top: s, Bottom: s, Start: s, End: s} }
	nonUniform := Border{
		Left: widthOf(5), Top: side(BorderSideOpts{Width: foundation.Some(10.0), StrokeAlign: foundation.Some(StrokeAlignCenter)}),
		Right:  side(BorderSideOpts{Width: foundation.Some(15.0), StrokeAlign: foundation.Some(StrokeAlignOutside)}),
		Bottom: widthOf(20),
	}
	nonUniformDir := BorderDirectional{Start: nonUniform.Left, Top: nonUniform.Top, End: nonUniform.Right, Bottom: nonUniform.Bottom}
	widthUniform := Border{Left: inside, Top: center, Right: outside, Bottom: inside}
	widthUniformDir := BorderDirectional{Start: inside, Top: center, End: outside, Bottom: inside}
	tests := []struct {
		name string
		got  EdgeInsetsGeometry
		want EdgeInsetsGeometry
	}{
		{"2 3 5 7", Border{Left: widthOf(2), Top: widthOf(3), Bottom: widthOf(5), Right: widthOf(7)}.Dimensions(), NewEdgeInsetsFromLTRB(2, 3, 7, 5)},
		{"inside", all(inside).Dimensions(), NewEdgeInsetsAll(10)},
		{"center", all(center).Dimensions(), NewEdgeInsetsAll(5)},
		{"outside", all(outside).Dimensions(), EdgeInsetsZero},
		{"inside directional", dall(inside).Dimensions(), NewEdgeInsetsDirectionalAll(10)},
		{"center directional", dall(center).Dimensions(), NewEdgeInsetsDirectionalAll(5)},
		{"outside directional", dall(outside).Dimensions(), EdgeInsetsDirectionalZero},
		{"non-uniform", nonUniform.Dimensions(), NewEdgeInsetsFromLTRB(5, 5, 0, 20)},
		{"non-uniform directional", nonUniformDir.Dimensions(), NewEdgeInsetsDirectionalFromSTEB(5, 5, 0, 20)},
		{"uniform width", widthUniform.Dimensions(), NewEdgeInsetsFromLTRB(10, 5, 0, 10)},
		{"uniform width directional", widthUniformDir.Dimensions(), NewEdgeInsetsDirectionalFromSTEB(10, 5, 0, 10)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.got.Equal(tt.want) {
				t.Errorf("Dimensions = %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestBorderIsUniform(t *testing.T) {
	col := func(c uint32) BorderSide { return side(BorderSideOpts{Color: foundation.Some(ui.NewColor(c))}) }
	none := side(BorderSideOpts{Style: BorderStyleNone})
	tests := []struct {
		name string
		b    Border
		want bool
	}{
		{"width differs", Border{widthOf(3), widthOf(3), widthOf(3.1), widthOf(3)}, false},
		{"same width", Border{widthOf(3), widthOf(3), widthOf(3), widthOf(3)}, true},
		{"color differs", Border{Left: col(0xFFFFFFFE), Top: col(0xFFFFFFFF), Right: col(0xFFFFFFFF), Bottom: col(0xFFFFFFFF)}, false},
		{"same color", Border{Left: col(0xFFFFFFFF), Top: col(0xFFFFFFFF), Right: col(0xFFFFFFFF), Bottom: col(0xFFFFFFFF)}, true},
		{"style differs", Border{Left: none, Top: none, Right: none, Bottom: widthOf(0)}, false},
		{"missing side", Border{Left: none, Top: none, Right: none}, false},
		{"stroke align differs", Border{Left: side(BorderSideOpts{}), Top: side(BorderSideOpts{StrokeAlign: foundation.Some(StrokeAlignCenter)}), Right: side(BorderSideOpts{StrokeAlign: foundation.Some(StrokeAlignOutside)})}, false},
		{"empty", Border{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b.IsUniform(); got != tt.want {
				t.Errorf("IsUniform = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBorderLerp(t *testing.T) {
	top10 := Border{Top: widthOf(10)}
	lr := func(l, r float64) Border { return Border{Left: widthOf(l), Right: widthOf(r)} }
	at0, at100 := lr(100, 200), lr(200, 100)
	tests := []struct {
		name string
		a, b *Border
		t    float64
		want *Border
	}{
		{"nil nil -1", nil, nil, -1, nil},
		{"top10 nil -1", &top10, nil, -1, &Border{Top: widthOf(20)}},
		{"nil top10 -1", nil, &top10, -1, &Border{}},
		{"at0 at100 -1", &at0, &at100, -1, &Border{Left: widthOf(0), Right: widthOf(300)}},
		{"top10 nil 0", &top10, nil, 0, &Border{Top: widthOf(10)}},
		{"nil top10 0", nil, &top10, 0, &Border{}},
		{"at0 at100 0", &at0, &at100, 0, &at0},
		{"top10 nil .25", &top10, nil, 0.25, &Border{Top: widthOf(7.5)}},
		{"nil top10 .25", nil, &top10, 0.25, &Border{Top: widthOf(2.5)}},
		{"at0 at100 .25", &at0, &at100, 0.25, &Border{Left: widthOf(125), Right: widthOf(175)}},
		{"top10 nil .75", &top10, nil, 0.75, &Border{Top: widthOf(2.5)}},
		{"nil top10 .75", nil, &top10, 0.75, &Border{Top: widthOf(7.5)}},
		{"at0 at100 .75", &at0, &at100, 0.75, &Border{Left: widthOf(175), Right: widthOf(125)}},
		{"top10 nil 1", &top10, nil, 1, &Border{}},
		{"nil top10 1", nil, &top10, 1, &Border{Top: widthOf(10)}},
		{"at0 at100 1", &at0, &at100, 1, &at100},
		{"top10 nil 2", &top10, nil, 2, &Border{}},
		{"nil top10 2", nil, &top10, 2, &Border{Top: widthOf(20)}},
		{"at0 at100 2", &at0, &at100, 2, &Border{Left: widthOf(300), Right: widthOf(0)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BorderLerp(tt.a, tt.b, tt.t)
			if tt.want == nil {
				if got != nil {
					t.Errorf("got %v, want nil", *got)
				}
				return
			}
			if got == nil || got.normalized() != tt.want.normalized() {
				t.Errorf("got %v, want %v", got, *tt.want)
			}
		})
	}
}

func TestBoxBorderLerp(t *testing.T) {
	if BoxBorderLerp(nil, nil, 0.5) != nil {
		t.Error("lerp(nil, nil) should be nil")
	}
	a := Border{Top: widthOf(10)}
	b := BorderDirectional{Start: widthOf(20)}
	if got, ok := BoxBorderLerp(a, nil, 0.5).(Border); !ok || got.Top.Width != 5 {
		t.Errorf("Border to nil = %v", BoxBorderLerp(a, nil, 0.5))
	}
	if got, ok := BoxBorderLerp(nil, b, 0.5).(BorderDirectional); !ok || got.Start.Width != 10 {
		t.Errorf("nil to BorderDirectional = %v", BoxBorderLerp(nil, b, 0.5))
	}
	if got := BoxBorderLerp(a, b, 0.25); got == nil {
		t.Error("Border to BorderDirectional should interpolate through a compound border")
	}
}

func TestBorderString(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("release and profile builds describe objects by their short names")
	}
	tests := []struct {
		got  string
		want string
	}{
		{Border{}.String(), "Border.all(BorderSide(width: 0.0, style: none))"},
		{NewBorderAll().String(), "Border.all(BorderSide)"},
		{Border{Top: widthOf(2)}.String(), "Border(top: BorderSide(width: 2.0))"},
		{BorderDirectional{Start: widthOf(2), Bottom: widthOf(3)}.String(), "BorderDirectional(start: BorderSide(width: 2.0), bottom: BorderSide(width: 3.0))"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("String = %q, want %q", tt.got, tt.want)
		}
	}
}
