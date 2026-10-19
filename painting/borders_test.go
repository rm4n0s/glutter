package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

var (
	someBlue   = ui.NewColor(0xFF0000FF)
	someGreen  = ui.NewColor(0xFF00FF00)
	someYellow = ui.NewColor(0xFFFFFF00)
)

func side(opts BorderSideOpts) BorderSide { return NewBorderSide(opts) }

func sideW(w float64) BorderSide { return side(BorderSideOpts{Width: foundation.Some(w)}) }

type sideFixtures struct {
	blue, blue2, green, green2, green3, green5 BorderSide
	none, none2, none3                         BorderSide
	side2, side3, side5, solid, yellowNone     BorderSide
}

func newSideFixtures() sideFixtures {
	c := func(col ui.Color) foundation.Opt[ui.Color] { return foundation.Some(col) }
	w := func(v float64) foundation.Opt[float64] { return foundation.Some(v) }
	return sideFixtures{
		blue: side(BorderSideOpts{Color: c(someBlue)}), blue2: side(BorderSideOpts{Color: c(someBlue), Width: w(2)}),
		green: side(BorderSideOpts{Color: c(someGreen)}), green2: side(BorderSideOpts{Color: c(someGreen), Width: w(2)}),
		green3: side(BorderSideOpts{Color: c(someGreen), Width: w(3)}), green5: side(BorderSideOpts{Color: c(someGreen), Width: w(5)}),
		none:  side(BorderSideOpts{Style: BorderStyleNone}),
		none2: side(BorderSideOpts{Color: c(someBlue), Width: w(2), Style: BorderStyleNone}),
		none3: side(BorderSideOpts{Style: BorderStyleNone, Width: w(3)}),
		side2: sideW(2), side3: sideW(3), side5: sideW(5), solid: side(BorderSideOpts{}),
		yellowNone: side(BorderSideOpts{Style: BorderStyleNone, Color: c(someYellow), Width: w(0)}),
	}
}

func TestBorderSideMerging(t *testing.T) {
	f := newSideFixtures()
	canMerge := []struct {
		name string
		a, b BorderSide
		want bool
	}{
		{"none none", BorderSideNone, BorderSideNone, true},
		{"none side2", BorderSideNone, f.side2, true},
		{"none yellowNone", BorderSideNone, f.yellowNone, true},
		{"green blue", f.green, f.blue, false},
		{"green2 blue2", f.green2, f.blue2, false},
		{"green2 green3", f.green2, f.green3, true},
		{"green2 none2", f.green2, f.none2, false},
		{"none3 none", f.none3, BorderSideNone, true},
		{"none3 side2", f.none3, f.side2, false},
		{"none3 yellowNone", f.none3, f.yellowNone, true},
		{"side2 none", f.side2, BorderSideNone, true},
		{"side2 none3", f.side2, f.none3, false},
		{"side2 side3", f.side2, f.side3, true},
		{"side2 yellowNone", f.side2, f.yellowNone, true},
		{"side3 side2", f.side3, f.side2, true},
		{"solid none", f.solid, f.none, false},
		{"yellowNone side2", f.yellowNone, f.side2, true},
		{"yellowNone yellowNone", f.yellowNone, f.yellowNone, true},
	}
	for _, tt := range canMerge {
		t.Run("canMerge "+tt.name, func(t *testing.T) {
			if got := BorderSideCanMerge(tt.a, tt.b); got != tt.want {
				t.Errorf("BorderSideCanMerge = %v, want %v", got, tt.want)
			}
		})
	}
	merge := []struct {
		name       string
		a, b, want BorderSide
	}{
		{"none none", BorderSideNone, BorderSideNone, BorderSideNone},
		{"none side2", BorderSideNone, f.side2, f.side2},
		{"none yellowNone", BorderSideNone, f.yellowNone, BorderSideNone},
		{"green2 green3", f.green2, f.green3, f.green5},
		{"none3 none", f.none3, BorderSideNone, f.none3},
		{"none3 yellowNone", f.none3, f.yellowNone, f.none3},
		{"side2 none", f.side2, BorderSideNone, f.side2},
		{"side2 side3", f.side2, f.side3, f.side5},
		{"side2 yellowNone", f.side2, f.yellowNone, f.side2},
		{"side3 side2", f.side3, f.side2, f.side5},
		{"yellowNone side2", f.yellowNone, f.side2, f.side2},
		{"yellowNone yellowNone", f.yellowNone, f.yellowNone, BorderSideNone},
	}
	for _, tt := range merge {
		t.Run("merge "+tt.name, func(t *testing.T) {
			if got := BorderSideMerge(tt.a, tt.b); got != tt.want {
				t.Errorf("BorderSideMerge = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBorderSideCopyWith(t *testing.T) {
	f := newSideFixtures()
	got := f.green2.CopyWith(BorderSideCopy{Color: foundation.Some(someBlue), Width: foundation.Some(3.0)})
	tests := []struct {
		name      string
		got, want BorderSide
	}{
		{"color and width", got, side(BorderSideOpts{Color: foundation.Some(someBlue), Width: foundation.Some(3.0)})},
		{"width", f.green2.CopyWith(BorderSideCopy{Width: foundation.Some(3.0)}), f.green3},
		{"color", f.green2.CopyWith(BorderSideCopy{Color: foundation.Some(someBlue)}), f.blue2},
		{"style", f.green2.CopyWith(BorderSideCopy{Style: BorderStyleNone}), side(BorderSideOpts{Color: foundation.Some(someGreen), Width: foundation.Some(2.0), Style: BorderStyleNone})},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestBorderSideScale(t *testing.T) {
	s3 := side(BorderSideOpts{Width: foundation.Some(3.0), Color: foundation.Some(someBlue)})
	s6 := side(BorderSideOpts{Width: foundation.Some(6.0), Color: foundation.Some(someBlue)})
	none := side(BorderSideOpts{Style: BorderStyleNone, Width: foundation.Some(0.0), Color: foundation.Some(someBlue)})
	tests := []struct {
		name      string
		got, want BorderSide
	}{
		{"x2", s3.Scale(2), s6}, {"x0.5", s6.Scale(0.5), s3}, {"x0", s6.Scale(0), none}, {"x-1", s6.Scale(-1), none}, {"none x2", none.Scale(2), none},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestBorderSideToPaint(t *testing.T) {
	p1 := side(BorderSideOpts{Width: foundation.Some(2.5), Color: foundation.Some(someYellow)}).ToPaint()
	if p1.StrokeWidth() != 2.5 || p1.Style() != ui.PaintingStyleStroke || p1.Color() != someYellow || p1.BlendMode() != ui.BlendModeSrcOver {
		t.Errorf("solid paint = %v", p1)
	}
	p2 := side(BorderSideOpts{Width: foundation.Some(2.5), Color: foundation.Some(someYellow), Style: BorderStyleNone}).ToPaint()
	if p2.StrokeWidth() != 0 || p2.Style() != ui.PaintingStyleStroke || p2.Color() != ui.NewColor(0x00000000) || p2.BlendMode() != ui.BlendModeSrcOver {
		t.Errorf("none paint = %v", p2)
	}
}

func TestBorderSideLerp(t *testing.T) {
	side0, side1, side2 := sideW(0), side(BorderSideOpts{}), sideW(2)
	tests := []struct {
		name      string
		got, want BorderSide
	}{
		{"no negative widths 1", BorderSideLerp(side2, side1, 10), BorderSideNone},
		{"no negative widths 2", BorderSideLerp(side1, side2, -10), BorderSideNone},
		{"extrapolate", BorderSideLerp(side0, side1, 2), side2},
		{"extrapolate to none", BorderSideLerp(side1, side0, 2), BorderSideNone},
		{"extrapolate 2", BorderSideLerp(side2, side1, 2), side0},
		{"identical", BorderSideLerp(side1, side1, 0.5), side1},
		{"align t=0", BorderSideLerp(sideW(2), side(BorderSideOpts{Width: foundation.Some(2.0), StrokeAlign: foundation.Some(StrokeAlignOutside)}), 0), sideW(2)},
		{
			"align t=.5", BorderSideLerp(sideW(2), side(BorderSideOpts{Width: foundation.Some(2.0), StrokeAlign: foundation.Some(StrokeAlignOutside)}), 0.5),
			side(BorderSideOpts{Width: foundation.Some(2.0), StrokeAlign: foundation.Some(StrokeAlignCenter)}),
		},
		{
			"align t=1", BorderSideLerp(sideW(2), side(BorderSideOpts{Width: foundation.Some(2.0), StrokeAlign: foundation.Some(StrokeAlignOutside)}), 1),
			side(BorderSideOpts{Width: foundation.Some(2.0), StrokeAlign: foundation.Some(StrokeAlignOutside)}),
		},
		{
			"center t=.5", BorderSideLerp(sideW(2), side(BorderSideOpts{Width: foundation.Some(2.0), StrokeAlign: foundation.Some(StrokeAlignCenter)}), 0.5),
			side(BorderSideOpts{Width: foundation.Some(2.0), StrokeAlign: foundation.Some(-0.5)}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestBorderSideString(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("release and profile builds describe objects by their short names")
	}
	s := side(BorderSideOpts{Color: foundation.Some(ui.NewColor(0xFFAABBCC)), Width: foundation.Some(1.2345)})
	if got, want := s.String(), "BorderSide(color: "+ui.NewColor(0xFFAABBCC).String()+", width: 1.2)"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}

func TestBorderSideStrokeGeometry(t *testing.T) {
	tests := []struct {
		name                  string
		align                 float64
		inset, outset, offset float64
	}{
		{"inside", StrokeAlignInside, 2, 0, -2},
		{"center", StrokeAlignCenter, 1, 1, 0},
		{"outside", StrokeAlignOutside, 0, 2, 2},
	}
	for _, tt := range tests {
		s := side(BorderSideOpts{Width: foundation.Some(2.0), StrokeAlign: foundation.Some(tt.align)})
		if s.StrokeInset() != tt.inset || s.StrokeOutset() != tt.outset || s.StrokeOffset() != tt.offset {
			t.Errorf("%s: inset %v outset %v offset %v", tt.name, s.StrokeInset(), s.StrokeOutset(), s.StrokeOffset())
		}
	}
}
