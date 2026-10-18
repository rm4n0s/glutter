package soft

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func open(pts ...float64) ui.FlatContour { return ui.FlatContour{Points: poly(pts...)} }

func closedC(pts ...float64) ui.FlatContour {
	return ui.FlatContour{Points: poly(pts...), Closed: true}
}

// strokeArea rasterizes the stroke and returns its area and bounds.
func strokeArea(c ui.FlatContour, st StrokeStyle) (float64, image.Rectangle) {
	m := Rasterize(Stroke([]ui.FlatContour{c}, st), FillNonZero, big)
	if m == nil {
		return 0, image.Rectangle{}
	}
	return m.Sum(), m.Rect
}

func TestStrokeArea(t *testing.T) {
	tests := []struct {
		name string
		c    ui.FlatContour
		st   StrokeStyle
		want float64
	}{
		{"butt line", open(10, 10, 30, 10), StrokeStyle{Width: 4}, 80},
		{"square caps add half a width at both ends", open(10, 10, 30, 10), StrokeStyle{Width: 4, Cap: ui.StrokeCapSquare}, 24 * 4},
		{"round caps add a circle", open(10, 10, 30, 10), StrokeStyle{Width: 4, Cap: ui.StrokeCapRound}, 80 + math.Pi*4},
		{"diagonal butt line", open(0, 0, 30, 40), StrokeStyle{Width: 2}, 100},
		{"closed square ring", closedC(10, 10, 30, 10, 30, 30, 10, 30), StrokeStyle{Width: 2}, 22*22 - 18*18},
		{"closed square ring, round joins", closedC(10, 10, 30, 10, 30, 30, 10, 30), StrokeStyle{Width: 2, Join: ui.StrokeJoinRound}, 20*20 - 18*18 + 4*20*1 + math.Pi},
		{"closed square ring, bevel joins", closedC(10, 10, 30, 10, 30, 30, 10, 30), StrokeStyle{Width: 2, Join: ui.StrokeJoinBevel}, 20*20 - 18*18 + 4*20*1 + 4*0.5},
		{"two collinear segments", open(0, 0, 10, 0, 20, 0), StrokeStyle{Width: 2}, 40},
		{"zero length with round cap", open(5, 5, 5, 5), StrokeStyle{Width: 10, Cap: ui.StrokeCapRound}, math.Pi * 25},
		{"zero length with square cap", open(5, 5, 5, 5), StrokeStyle{Width: 10, Cap: ui.StrokeCapSquare}, 100},
		{"zero length with butt cap", open(5, 5, 5, 5), StrokeStyle{Width: 10}, 0},
		{"repeated points", open(0, 0, 0, 0, 10, 0, 10, 0), StrokeStyle{Width: 2}, 20},
		{"closed contour that is one point", closedC(5, 5), StrokeStyle{Width: 10, Cap: ui.StrokeCapRound}, math.Pi * 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.st.Scale = 10 // many segments for round shapes
			got, _ := strokeArea(tt.c, tt.st)
			if math.Abs(got-tt.want) > tt.want*0.003+1e-3 {
				t.Errorf("area = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrokeMiter(t *testing.T) {
	// A right angle: miter length ratio √2, which is within the default limit 4.
	right := open(0, 20, 0, 0, 20, 0)
	miter, _ := strokeArea(right, StrokeStyle{Width: 4})
	bevel, _ := strokeArea(right, StrokeStyle{Width: 4, Join: ui.StrokeJoinBevel})
	if miter-bevel < 1.9 || miter-bevel > 2.1 { // the corner square of 2x2 minus the bevel triangle: 4 - 2
		t.Errorf("miter %v and bevel %v differ by %v, want 2", miter, bevel, miter-bevel)
	}
	// A very sharp angle exceeds the limit and falls back to the bevel.
	sharp := open(0, 0, 100, 5, 0, 10)
	limited, _ := strokeArea(sharp, StrokeStyle{Width: 4, MiterLimit: 4})
	bevelled, _ := strokeArea(sharp, StrokeStyle{Width: 4, Join: ui.StrokeJoinBevel})
	if math.Abs(limited-bevelled) > 1e-6 {
		t.Errorf("over the limit: %v, want the bevel's %v", limited, bevelled)
	}
	// A raised limit lets the miter through.
	raised, _ := strokeArea(sharp, StrokeStyle{Width: 4, MiterLimit: 100})
	if raised < limited+10 {
		t.Errorf("limit raised: %v, not larger than %v", raised, limited)
	}
}

func TestStrokeBounds(t *testing.T) {
	tests := []struct {
		name string
		st   StrokeStyle
		want image.Rectangle
	}{
		{"butt", StrokeStyle{Width: 6}, image.Rect(10, 17, 50, 23)},
		{"square", StrokeStyle{Width: 6, Cap: ui.StrokeCapSquare}, image.Rect(7, 17, 53, 23)},
		{"round", StrokeStyle{Width: 6, Cap: ui.StrokeCapRound}, image.Rect(7, 17, 53, 23)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, r := strokeArea(open(10, 20, 50, 20), tt.st)
			if r != tt.want {
				t.Errorf("bounds = %v, want %v", r, tt.want)
			}
		})
	}
}

func TestStrokeInvalid(t *testing.T) {
	c := []ui.FlatContour{open(0, 0, 10, 0)}
	for _, w := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if got := Stroke(c, StrokeStyle{Width: w}); got != nil {
			t.Errorf("width %v: got %d polygons", w, len(got))
		}
	}
}

func TestStrokeReversal(t *testing.T) {
	// A path that goes out and comes back on itself: no hole, no double cover.
	c := open(0, 0, 20, 0, 0, 0)
	for _, j := range []ui.StrokeJoin{ui.StrokeJoinMiter, ui.StrokeJoinRound, ui.StrokeJoinBevel} {
		got, _ := strokeArea(c, StrokeStyle{Width: 4, Join: j, Scale: 10})
		want := 80.0
		if j == ui.StrokeJoinRound {
			want += math.Pi * 2 // the round tip adds half a circle, the other end is the butt
		}
		// The two sides retrace each other around the tip, and the coverage of
		// the pixels along coincident edges adds up, so the area is a little high.
		if math.Abs(got-want) > want*0.015 {
			t.Errorf("join %v: area = %v, want %v", j, got, want)
		}
	}
}
