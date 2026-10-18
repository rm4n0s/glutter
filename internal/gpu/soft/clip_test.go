package soft

import (
	"image"
	"math"
	"testing"

	"github.com/rm4n0s/glutter/internal/goldens"
	"github.com/rm4n0s/glutter/ui"
)

func maskOf(r image.Rectangle, v float32) *Mask {
	m := &Mask{Rect: r, Cov: make([]float32, r.Dx()*r.Dy())}
	for i := range m.Cov {
		m.Cov[i] = v
	}
	return m
}

func TestClipOperations(t *testing.T) {
	full := newClip(image.Rect(0, 0, 10, 10))
	tests := []struct {
		name       string
		clip       clip
		wantBounds image.Rectangle
		wantMask   bool
		wantAt     map[image.Point]float32
	}{
		{"full", full, image.Rect(0, 0, 10, 10), false, map[image.Point]float32{{5, 5}: 1, {10, 5}: 0, {-1, 0}: 0}},
		{"intersect with a rect", full.intersect(maskOf(image.Rect(2, 3, 6, 8), 1)), image.Rect(2, 3, 6, 8), false, map[image.Point]float32{{2, 3}: 1, {6, 3}: 0}},
		{"intersect with a half mask keeps the mask", full.intersect(maskOf(image.Rect(2, 3, 6, 8), 0.5)), image.Rect(2, 3, 6, 8), true, map[image.Point]float32{{3, 4}: 0.5}},
		{"intersect twice multiplies", full.intersect(maskOf(image.Rect(0, 0, 10, 10), 0.5)).intersect(maskOf(image.Rect(0, 0, 10, 10), 0.5)), image.Rect(0, 0, 10, 10), true, map[image.Point]float32{{1, 1}: 0.25}},
		{"disjoint", full.intersect(maskOf(image.Rect(20, 20, 30, 30), 1)), image.Rectangle{}, false, map[image.Point]float32{{5, 5}: 0}},
		{"nil shape is nothing", full.intersect(nil), image.Rectangle{}, false, map[image.Point]float32{{5, 5}: 0}},
		{"empty stays empty", clip{}.intersect(maskOf(image.Rect(0, 0, 5, 5), 1)), image.Rectangle{}, false, map[image.Point]float32{{1, 1}: 0}},
		{"subtract a rect", full.subtract(maskOf(image.Rect(0, 0, 5, 10), 1)), image.Rect(0, 0, 10, 10), true, map[image.Point]float32{{2, 2}: 0, {7, 2}: 1}},
		{"subtract everything", full.subtract(maskOf(image.Rect(0, 0, 10, 10), 1)), image.Rectangle{}, false, map[image.Point]float32{{2, 2}: 0}},
		{"subtract nil cuts nothing", full.subtract(nil), image.Rect(0, 0, 10, 10), false, map[image.Point]float32{{2, 2}: 1}},
		{"subtract outside cuts nothing", full.subtract(maskOf(image.Rect(50, 50, 60, 60), 1)), image.Rect(0, 0, 10, 10), false, map[image.Point]float32{{2, 2}: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.clip.bounds != tt.wantBounds && (!tt.clip.bounds.Empty() || !tt.wantBounds.Empty()) {
				t.Errorf("bounds = %v, want %v", tt.clip.bounds, tt.wantBounds)
			}
			if (tt.clip.mask != nil) != tt.wantMask {
				t.Errorf("has mask = %v, want %v", tt.clip.mask != nil, tt.wantMask)
			}
			if tt.clip.empty() != tt.wantBounds.Empty() {
				t.Errorf("empty = %v", tt.clip.empty())
			}
			for p, want := range tt.wantAt {
				if got := tt.clip.at(p.X, p.Y); math.Abs(float64(got-want)) > 1e-6 {
					t.Errorf("at(%v) = %v, want %v", p, got, want)
				}
			}
		})
	}
}

func TestThreshold(t *testing.T) {
	m := &Mask{Rect: image.Rect(0, 0, 4, 1), Cov: []float32{0, 0.49, 0.5, 1}}
	got := threshold(m)
	want := []float32{0, 0, 1, 1}
	for i := range want {
		if got.Cov[i] != want[i] {
			t.Errorf("threshold = %v, want %v", got.Cov, want)
			break
		}
	}
	if threshold(nil) != nil {
		t.Error("threshold(nil) should be nil")
	}
}

func TestClipPixels(t *testing.T) {
	green := paint(0xFF00FF00)
	paintAll := func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 20, 20), green) }
	tests := []struct {
		name string
		draw func(c *ui.Canvas)
		at   map[image.Point]uint8 // the alpha at each pixel
	}{
		{"rect", func(c *ui.Canvas) { c.ClipRect(ui.NewRectFromLTWH(2, 3, 5, 4)); paintAll(c) },
			map[image.Point]uint8{{2, 3}: 255, {6, 6}: 255, {7, 6}: 0, {1, 3}: 0, {2, 7}: 0}},
		{"rect with a fractional edge", func(c *ui.Canvas) { c.ClipRect(ui.NewRectFromLTWH(2.5, 3, 5, 4)); paintAll(c) },
			map[image.Point]uint8{{2, 3}: 128, {3, 3}: 255, {7, 3}: 128}},
		{"rect without anti-aliasing", func(c *ui.Canvas) {
			c.ClipRect(ui.NewRectFromLTWH(2.5, 3, 5, 4), ui.ClipRectOpts{DoAntiAlias: ui.Ptr(false)})
			paintAll(c)
		}, // A pixel whose center is on the edge counts as inside.
			map[image.Point]uint8{{2, 3}: 255, {1, 3}: 0, {7, 3}: 255, {8, 3}: 0, {6, 3}: 255}},
		{"nested rects intersect", func(c *ui.Canvas) {
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 10, 10))
			c.ClipRect(ui.NewRectFromLTWH(5, 5, 10, 10))
			paintAll(c)
		}, map[image.Point]uint8{{4, 4}: 0, {5, 5}: 255, {9, 9}: 255, {10, 10}: 0}},
		{"clip under a translation", func(c *ui.Canvas) {
			c.Translate(4, 4)
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 3, 3))
			c.Translate(-4, -4)
			paintAll(c)
		}, map[image.Point]uint8{{4, 4}: 255, {6, 6}: 255, {7, 7}: 0, {3, 3}: 0}},
		{"clip under a scale", func(c *ui.Canvas) {
			c.Scale(2, 3)
			c.ClipRect(ui.NewRectFromLTWH(1, 1, 2, 2))
			c.Scale(0.5, 1.0/3)
			paintAll(c)
		}, map[image.Point]uint8{{2, 3}: 255, {5, 8}: 255, {6, 8}: 0, {5, 9}: 0, {1, 3}: 0}},
		{"difference cuts a hole", func(c *ui.Canvas) {
			c.ClipRect(ui.NewRectFromLTWH(2, 2, 4, 4), ui.ClipRectOpts{ClipOp: ui.ClipOpDifference})
			paintAll(c)
		}, map[image.Point]uint8{{1, 1}: 255, {2, 2}: 0, {5, 5}: 0, {6, 6}: 255}},
		{"difference after intersect", func(c *ui.Canvas) {
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 8, 8))
			c.ClipRect(ui.NewRectFromLTWH(2, 2, 2, 2), ui.ClipRectOpts{ClipOp: ui.ClipOpDifference})
			paintAll(c)
		}, map[image.Point]uint8{{0, 0}: 255, {2, 2}: 0, {8, 8}: 0, {7, 7}: 255}},
		{"empty rect clips everything", func(c *ui.Canvas) { c.ClipRect(ui.NewRectFromLTWH(3, 3, 0, 5)); paintAll(c) },
			map[image.Point]uint8{{3, 4}: 0, {5, 5}: 0}},
		{"rrect", func(c *ui.Canvas) {
			c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(0, 0, 20, 20), 10, 10))
			paintAll(c)
		},
			map[image.Point]uint8{{0, 0}: 0, {10, 10}: 255, {10, 3}: 255, {19, 19}: 0}},
		{"path", func(c *ui.Canvas) {
			p := ui.NewPath()
			p.MoveTo(0, 0)
			p.LineTo(20, 0)
			p.LineTo(0, 20)
			p.Close()
			c.ClipPath(p)
			paintAll(c)
		}, map[image.Point]uint8{{1, 1}: 255, {18, 18}: 0, {5, 12}: 255}},
		{"restore brings back the clip", func(c *ui.Canvas) {
			c.Save()
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 2, 2))
			c.Restore()
			paintAll(c)
		}, map[image.Point]uint8{{10, 10}: 255}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := renderOf(t, 20, 20, tt.draw)
			for p, want := range tt.at {
				if got := px(img, p.X, p.Y)[3]; got != want && (want != 128 || (got != 127 && got != 128)) {
					t.Errorf("alpha at %v = %d, want %d", p, got, want)
				}
			}
		})
	}
}

func TestGoldenClips(t *testing.T) {
	stripes := func(c *ui.Canvas) {
		white(c)
		for i := 0; i < 8; i++ {
			col := uint32(0xFF3060C0)
			if i%2 == 1 {
				col = 0xFFF0A020
			}
			c.DrawRect(ui.NewRectFromLTWH(float64(i)*8, 0, 8, 64), paint(col))
		}
	}
	star := func() *ui.Path {
		p := ui.NewPath()
		for i := 0; i < 5; i++ {
			a := -math.Pi/2 + float64(i)*4*math.Pi/5
			x, y := 32+28*math.Cos(a), 33+28*math.Sin(a)
			if i == 0 {
				p.MoveTo(x, y)
			} else {
				p.LineTo(x, y)
			}
		}
		p.Close()
		return p
	}
	box := ui.NewRectFromLTWH(8.5, 10.25, 47, 40)
	tests := []struct {
		name string
		clip func(c *ui.Canvas)
	}{
		{"rect", func(c *ui.Canvas) { c.ClipRect(box) }},
		{"rect-hard", func(c *ui.Canvas) { c.ClipRect(box, ui.ClipRectOpts{DoAntiAlias: ui.Ptr(false)}) }},
		{"rect-difference", func(c *ui.Canvas) { c.ClipRect(box, ui.ClipRectOpts{ClipOp: ui.ClipOpDifference}) }},
		{"rrect", func(c *ui.Canvas) { c.ClipRRect(ui.NewRRectFromRectXY(box, 16, 12)) }},
		{"rrect-hard", func(c *ui.Canvas) {
			c.ClipRRect(ui.NewRRectFromRectXY(box, 16, 12), ui.ClipOpts{DoAntiAlias: ui.Ptr(false)})
		}},
		{"rrect-rotated", func(c *ui.Canvas) {
			c.Translate(32, 32)
			c.Rotate(math.Pi / 7)
			c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromCenter(ui.Offset{}, 44, 30), 12, 12))
			c.Rotate(-math.Pi / 7)
			c.Translate(-32, -32)
		}},
		{"rsuperellipse", func(c *ui.Canvas) { c.ClipRSuperellipse(ui.NewRSuperellipseFromRectXY(box, 18, 18)) }},
		{"path-star", func(c *ui.Canvas) { c.ClipPath(star()) }},
		{"path-star-hard", func(c *ui.Canvas) { c.ClipPath(star(), ui.ClipOpts{DoAntiAlias: ui.Ptr(false)}) }},
		{"nested", func(c *ui.Canvas) {
			c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(4, 4, 40, 40), 14, 14))
			c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20, 20, 40, 40), 14, 14))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := renderOf(t, 64, 64, func(c *ui.Canvas) {
				c.Save()
				tt.clip(c)
				stripes(c)
				c.Restore()
			})
			goldens.Check(t, "clip-"+tt.name, img)
		})
	}
}
