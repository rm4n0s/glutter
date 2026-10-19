package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestEdgeInsetsConstructors(t *testing.T) {
	ltrb := EdgeInsets{10, 20, 30, 40}
	tests := []struct {
		name string
		got  EdgeInsetsGeometry
		ltr  EdgeInsets
		rtl  EdgeInsets
	}{
		{"fromLTRB", NewEdgeInsetsFromLTRB(10, 20, 30, 40), ltrb, ltrb},
		{"all", NewEdgeInsetsAll(10), EdgeInsets{10, 10, 10, 10}, EdgeInsets{10, 10, 10, 10}},
		{"only", EdgeInsets{Left: 10, Top: 20, Right: 30, Bottom: 40}, ltrb, ltrb},
		{"symmetric", NewEdgeInsetsSymmetric(20, 10), EdgeInsets{10, 20, 10, 20}, EdgeInsets{10, 20, 10, 20}},
		{"steb", NewEdgeInsetsDirectionalFromSTEB(10, 20, 30, 40), EdgeInsets{10, 20, 30, 40}, EdgeInsets{30, 20, 10, 40}},
		{"directional all", NewEdgeInsetsDirectionalAll(10), EdgeInsets{10, 10, 10, 10}, EdgeInsets{10, 10, 10, 10}},
		{"directional symmetric", NewEdgeInsetsDirectionalSymmetric(10, 20), EdgeInsets{10, 20, 10, 20}, EdgeInsets{10, 20, 10, 20}},
		{"directional only start", EdgeInsetsDirectional{Start: 963.25}, EdgeInsets{Left: 963.25}, EdgeInsets{Right: 963.25}},
		{"directional only end", EdgeInsetsDirectional{End: 963.25}, EdgeInsets{Right: 963.25}, EdgeInsets{Left: 963.25}},
		{"directional only top", EdgeInsetsDirectional{Top: 963.25}, EdgeInsets{Top: 963.25}, EdgeInsets{Top: 963.25}},
		{"directional only bottom", EdgeInsetsDirectional{Bottom: 963.25}, EdgeInsets{Bottom: 963.25}, EdgeInsets{Bottom: 963.25}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.got.Resolve(ui.TextDirectionLTR); got != tt.ltr {
				t.Errorf("ltr = %v, want %v", got, tt.ltr)
			}
			if got := tt.got.Resolve(ui.TextDirectionRTL); got != tt.rtl {
				t.Errorf("rtl = %v, want %v", got, tt.rtl)
			}
		})
	}
}

func TestEdgeInsetsControl(t *testing.T) {
	insets := NewEdgeInsetsFromLTRB(5, 7, 11, 13)
	r := ui.NewRectFromLTRB(23, 32, 124, 143)
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"topLeft", insets.TopLeft(), ui.Offset{Dx: 5, Dy: 7}},
		{"topRight", insets.TopRight(), ui.Offset{Dx: -11, Dy: 7}},
		{"bottomLeft", insets.BottomLeft(), ui.Offset{Dx: 5, Dy: -13}},
		{"bottomRight", insets.BottomRight(), ui.Offset{Dx: -11, Dy: -13}},
		{"collapsedSize", insets.CollapsedSize(), ui.Size{Width: 16, Height: 20}},
		{"flipped", insets.Flipped(), EdgeInsets{11, 13, 5, 7}},
		{"along horizontal", insets.Along(AxisHorizontal), 16.0},
		{"along vertical", insets.Along(AxisVertical), 20.0},
		{"inflateRect", insets.InflateRect(r), ui.NewRectFromLTRB(18, 25, 135, 156)},
		{"deflateRect", insets.DeflateRect(r), ui.NewRectFromLTRB(28, 39, 113, 130)},
		{"inflateSize", insets.InflateSize(ui.Size{Width: 100, Height: 125}), ui.Size{Width: 116, Height: 145}},
		{"deflateSize", insets.DeflateSize(ui.Size{Width: 100, Height: 125}), ui.Size{Width: 84, Height: 105}},
		{"div", insets.Div(2), EdgeInsets{2.5, 3.5, 5.5, 6.5}},
		{"truncDiv", insets.TruncDiv(2), EdgeInsets{2, 3, 5, 6}},
		{"mod", insets.Mod(5), EdgeInsets{0, 2, 1, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestEdgeInsetsLerp(t *testing.T) {
	a, b := NewEdgeInsetsAll(10), NewEdgeInsetsAll(20)
	got := EdgeInsetsLerp(&a, &b, 0.25)
	for name, want := range map[string]EdgeInsets{
		"a*1.25": a.Mul(1.25), "b*0.625": b.Mul(0.625), "a+2.5": a.Plus(NewEdgeInsetsAll(2.5)), "b-7.5": b.Minus(NewEdgeInsetsAll(7.5)),
	} {
		if *got != want {
			t.Errorf("%s: got %v, want %v", name, *got, want)
		}
	}
	if EdgeInsetsLerp(nil, nil, 0.25) != nil {
		t.Error("lerp(nil, nil) should be nil")
	}
	if got := EdgeInsetsLerp(nil, &b, 0.25); *got != b.Mul(0.25) {
		t.Errorf("lerp(nil, b) = %v", *got)
	}
	if got := EdgeInsetsLerp(&a, nil, 0.25); *got != a.Mul(0.75) {
		t.Errorf("lerp(a, nil) = %v", *got)
	}
	z := EdgeInsetsZero
	if EdgeInsetsLerp(&z, &z, 0.5) != &z {
		t.Error("lerp of identical values should return the same pointer")
	}
}

func dir(start, top, end, bottom float64) EdgeInsetsDirectional {
	return NewEdgeInsetsDirectionalFromSTEB(start, top, end, bottom)
}

func TestEdgeInsetsEquality(t *testing.T) {
	mixedA := EdgeInsetsDirectional{End: 5}.Add(EdgeInsets{Right: 5})
	mixedB := EdgeInsetsDirectional{End: 5}.Add(EdgeInsets{Left: 5})
	tests := []struct {
		name string
		a, b EdgeInsetsGeometry
		want bool
	}{
		{"directional same", EdgeInsetsDirectional{Top: 5, Bottom: 7}, EdgeInsetsDirectional{Top: 5, Bottom: 7}, true},
		{"insets vs directional vertical", EdgeInsets{Top: 5, Bottom: 7}, EdgeInsetsDirectional{Top: 5, Bottom: 7}, true},
		{"directional vs insets vertical", EdgeInsetsDirectional{Top: 5, Bottom: 7}, EdgeInsets{Top: 5, Bottom: 7}, true},
		{"left vs start", EdgeInsets{Left: 5}, EdgeInsetsDirectional{Start: 5}, false},
		{"start vs left", EdgeInsetsDirectional{Start: 5}, EdgeInsets{Left: 5}, false},
		{"right vs end", EdgeInsets{Right: 5}, EdgeInsetsDirectional{End: 5}, false},
		{"mixed same", mixedA, EdgeInsetsDirectional{End: 5}.Add(EdgeInsets{Right: 5}), true},
		{"mixed different", mixedA, mixedB, false},
		{"top sum is plain", EdgeInsetsDirectional{Top: 1}.Add(EdgeInsets{Top: 2}), EdgeInsetsDirectional{Top: 3}, true},
		{"top sum is plain insets", EdgeInsetsDirectional{Top: 1}.Add(EdgeInsets{Top: 2}), EdgeInsets{Top: 3}, true},
		{"top vs bottom", EdgeInsetsDirectional{Top: 1}, EdgeInsetsDirectional{Bottom: 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.want {
				t.Errorf("Equal = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEdgeInsetsCopyWith(t *testing.T) {
	src := EdgeInsets{Left: 1, Top: 2, Bottom: 3, Right: 4}
	got := src.CopyWith(EdgeInsetsCopy{Left: foundation.Some(5.0), Top: foundation.Some(6.0)})
	if want := (EdgeInsets{Left: 5, Top: 6, Bottom: 3, Right: 4}); got != want {
		t.Errorf("CopyWith = %v, want %v", got, want)
	}
	dsrc := EdgeInsetsDirectional{Start: 1, Top: 2, Bottom: 3, End: 4}
	dgot := dsrc.CopyWith(EdgeInsetsDirectionalCopy{Start: foundation.Some(5.0), Top: foundation.Some(6.0)})
	if want := (EdgeInsetsDirectional{Start: 5, Top: 6, Bottom: 3, End: 4}); dgot != want {
		t.Errorf("directional CopyWith = %v, want %v", dgot, want)
	}
}

func TestEdgeInsetsGeometryLerp(t *testing.T) {
	tests := []struct {
		name string
		a, b EdgeInsetsGeometry
		t    float64
		want EdgeInsetsGeometry
	}{
		{"end to nil", EdgeInsetsDirectional{End: 10}, nil, 0.5, EdgeInsetsDirectional{End: 5}},
		{"start to nil", EdgeInsetsDirectional{Start: 10}, nil, 0.5, EdgeInsetsDirectional{Start: 5}},
		{"top to nil", EdgeInsetsDirectional{Top: 10}, nil, 0.5, EdgeInsetsDirectional{Top: 5}},
		{"bottom to nil", EdgeInsetsDirectional{Bottom: 10}, nil, 0.5, EdgeInsetsDirectional{Bottom: 5}},
		{"bottom to directional zero", EdgeInsetsDirectional{Bottom: 10}, EdgeInsetsDirectionalZero, 0.5, EdgeInsetsDirectional{Bottom: 5}},
		{"bottom to zero", EdgeInsetsDirectional{Bottom: 10}, EdgeInsetsZero, 0.5, EdgeInsetsDirectional{Bottom: 5}},
		{
			"start to left", EdgeInsetsDirectional{Start: 10}, EdgeInsets{Left: 20}, 0.5,
			EdgeInsetsDirectional{Start: 5}.Add(EdgeInsets{Left: 10}),
		},
		{
			"directional to mixed", EdgeInsetsDirectional{Bottom: 1}, EdgeInsetsDirectional{Start: 1, Bottom: 1}.Add(EdgeInsets{Right: 2}), 0.5,
			EdgeInsetsDirectional{Start: 0.5}.Add(EdgeInsets{Right: 1, Bottom: 1}),
		},
		{
			"insets to mixed", EdgeInsets{Bottom: 1}, EdgeInsetsDirectional{End: 1, Bottom: 1}.Add(EdgeInsets{Right: 2}), 0.5,
			EdgeInsetsDirectional{End: 0.5}.Add(EdgeInsets{Right: 1, Bottom: 1}),
		},
		{"nil nil", nil, nil, 0.25, nil},
		{"nil to directional", nil, EdgeInsetsDirectional{Start: 20, End: 20}, 0.25, EdgeInsetsDirectional{Start: 5, End: 5}},
		{"directional to nil", EdgeInsetsDirectional{Start: 10, End: 10}, nil, 0.25, EdgeInsetsDirectional{Start: 7.5, End: 7.5}},
		{
			"directional", EdgeInsetsDirectional{Start: 10, End: 10}, EdgeInsetsDirectional{Start: 20, End: 20}, 0.25,
			EdgeInsetsDirectional{Start: 12.5, End: 12.5},
		},
		{"normal", NewEdgeInsetsAll(10), NewEdgeInsetsAll(20), 0.25, NewEdgeInsetsAll(12.5)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EdgeInsetsGeometryLerp(tt.a, tt.b, tt.t)
			if tt.want == nil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
				return
			}
			if got == nil || !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
	z := EdgeInsetsGeometry(EdgeInsetsZero)
	if EdgeInsetsGeometryLerp(z, z, 0.5) != z {
		t.Error("lerp of identical values should return the same value")
	}
}

func TestEdgeInsetsOperators(t *testing.T) {
	a := NewEdgeInsetsFromLTRB(1, 2, 3, 5)
	for name, tt := range map[string]struct{ got, want EdgeInsets }{
		"mul": {a.Mul(2), EdgeInsets{2, 4, 6, 10}}, "div": {a.Div(2), EdgeInsets{0.5, 1, 1.5, 2.5}},
		"mod": {a.Mod(2), EdgeInsets{1, 0, 1, 1}}, "truncDiv": {a.TruncDiv(2), EdgeInsets{0, 1, 1, 2}},
		"plus": {a.Plus(a), a.Mul(2)}, "minus": {a.Minus(a), EdgeInsetsZero}, "neg": {a.Neg(), EdgeInsets{-1, -2, -3, -5}},
	} {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", name, tt.got, tt.want)
		}
	}
	d := dir(1, 2, 3, 5)
	for name, tt := range map[string]struct{ got, want EdgeInsetsDirectional }{
		"mul": {d.Mul(2), dir(2, 4, 6, 10)}, "div": {d.Div(2), dir(0.5, 1, 1.5, 2.5)},
		"mod": {d.Mod(2), dir(1, 0, 1, 1)}, "truncDiv": {d.TruncDiv(2), dir(0, 1, 1, 2)},
		"plus": {d.Plus(d), d.Mul(2)}, "minus": {d.Minus(d), EdgeInsetsDirectionalZero},
	} {
		if tt.got != tt.want {
			t.Errorf("directional %s: got %v, want %v", name, tt.got, tt.want)
		}
	}
	// A mixed value works through the geometry functions.
	m := d.Add(EdgeInsetsZero)
	if _, ok := m.(EdgeInsetsDirectional); ok {
		t.Error("adding EdgeInsets.zero should keep the value as mixed, not directional")
	}
	for name, tt := range map[string]struct{ got, want EdgeInsetsGeometry }{
		"mul": {EdgeInsetsGeometryMul(m, 2), dir(2, 4, 6, 10)}, "div": {EdgeInsetsGeometryDiv(m, 2), dir(0.5, 1, 1.5, 2.5)},
		"mod": {EdgeInsetsGeometryMod(m, 2), dir(1, 0, 1, 1)}, "truncDiv": {EdgeInsetsGeometryTruncDiv(m, 2), dir(0, 1, 1, 2)},
		"add": {m.Add(m), EdgeInsetsGeometryMul(m, 2)}, "subtract": {m.Subtract(m), EdgeInsetsDirectionalZero},
	} {
		if !tt.got.Equal(tt.want) {
			t.Errorf("geometry %s: got %v, want %v", name, tt.got, tt.want)
		}
	}
	if !m.Subtract(m).Equal(EdgeInsetsZero) {
		t.Error("subtracting a mixed value from itself should equal EdgeInsets.zero")
	}
}

func TestEdgeInsetsString(t *testing.T) {
	tests := []struct {
		name string
		got  EdgeInsetsGeometry
		want string
	}{
		{"zero", EdgeInsetsZero, "EdgeInsets.zero"},
		{"all", EdgeInsets{Top: 1.01, Left: 1.01, Right: 1.01, Bottom: 1.01}, "EdgeInsets.all(1.0)"},
		{"directional", EdgeInsetsDirectional{Start: 1.01, End: 1.01, Top: 1.01, Bottom: 1.01}, "EdgeInsetsDirectional(1.0, 1.0, 1.0, 1.0)"},
		{"mixed start and top", EdgeInsetsDirectional{Start: 4}.Add(EdgeInsets{Top: 3}), "EdgeInsetsDirectional(4.0, 3.0, 0.0, 0.0)"},
		{"mixed top and right", EdgeInsetsDirectional{Top: 4}.Add(EdgeInsets{Right: 3}), "EdgeInsets(0.0, 4.0, 3.0, 0.0)"},
		{
			"mixed start and left", EdgeInsetsDirectional{Start: 4}.Add(EdgeInsets{Left: 3}),
			"EdgeInsets(3.0, 0.0, 0.0, 0.0) + EdgeInsetsDirectional(4.0, 0.0, 0.0, 0.0)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.got.String(); got != tt.want {
				t.Errorf("String = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEdgeInsetsRRect(t *testing.T) {
	original := ui.NewRRectFromLTRBAndCorners(10, 20, 30, 60, ui.Corners{
		TopLeft: ui.NewRadiusCircular(8), TopRight: ui.NewRadiusCircular(9), BottomRight: ui.NewRadiusCircular(10), BottomLeft: ui.NewRadiusCircular(11),
	})
	insets := NewEdgeInsetsFromLTRB(2, 3, 4, 5)
	inflated := insets.InflateRRect(original)
	if inflated.Left != 8 || inflated.Top != 17 || inflated.Right != 34 || inflated.Bottom != 65 {
		t.Errorf("inflated edges = %v", inflated)
	}
	if got := inflated.TlRadius(); got != (ui.Radius{X: 10, Y: 11}) {
		t.Errorf("inflated tl radius = %v", got)
	}
	deflated := insets.DeflateRRect(original)
	if deflated.Left != 12 || deflated.Top != 23 || deflated.Right != 26 || deflated.Bottom != 55 {
		t.Errorf("deflated edges = %v", deflated)
	}
	if got := deflated.TlRadius(); got != (ui.Radius{X: 6, Y: 5}) {
		t.Errorf("deflated tl radius = %v", got)
	}
	if got := EdgeInsetsZero.InflateRRect(original); got != original {
		t.Errorf("zero inflate = %v, want the original", got)
	}
	if got := EdgeInsetsZero.DeflateRRect(original); got != original {
		t.Errorf("zero deflate = %v, want the original", got)
	}
}
