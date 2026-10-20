package gpu

import (
	"image"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func rectPicture(r ui.Rect, argb uint32) *ui.Picture {
	return pictureOf(func(c *ui.Canvas) { c.DrawRect(r, paintOf(argb)) })
}

// sceneOf puts each picture at its offset, in order.
func sceneOf(items ...damageItem) *ui.Scene {
	b := ui.NewSceneBuilder()
	for _, it := range items {
		if it.clip != nil {
			b.PushClipRect(*it.clip)
		}
		if it.alpha != 0 {
			b.PushOpacity(it.alpha)
		}
		b.AddPicture(it.at, it.pic)
		if it.alpha != 0 {
			b.Pop()
		}
		if it.clip != nil {
			b.Pop()
		}
	}
	return b.Build()
}

type damageItem struct {
	pic   *ui.Picture
	at    ui.Offset
	alpha int
	clip  *ui.Rect
}

func covers(d Damage, p image.Point) bool {
	if d.Full {
		return true
	}
	for _, r := range d.Rects {
		if p.In(r) {
			return true
		}
	}
	return false
}

func TestDamageTrackerFrames(t *testing.T) {
	a := rectPicture(ui.NewRectFromLTWH(0, 0, 40, 40), 0xFFFF0000)
	b := rectPicture(ui.NewRectFromLTWH(0, 0, 40, 40), 0xFF00FF00)
	b2 := rectPicture(ui.NewRectFromLTWH(0, 0, 40, 40), 0xFF00FF00) // recorded again
	fill := pictureOf(func(c *ui.Canvas) { c.DrawPaint(paintOf(0x80000000)) })
	clip := ui.NewRectFromLTWH(300, 300, 50, 50)
	blur := pictureOf(func(c *ui.Canvas) {
		p := paintOf(0xFF000000)
		p.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 2}))
		c.SaveLayer(nil, p)
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), paintOf(0xFF000000))
		c.Restore()
	})

	type step struct {
		name  string
		scene func() *ui.Scene
		// size is the surface; zero keeps the previous one.
		size image.Point
		// invalidate calls Invalidate before the update.
		invalidate bool
		full       bool
		// rects are the exact rectangles when full is false.
		rects []image.Rectangle
	}
	var last *ui.Scene
	same := func() *ui.Scene { return last }
	steps := []step{
		{name: "first frame", size: image.Pt(512, 512), full: true,
			scene: func() *ui.Scene {
				return sceneOf(damageItem{pic: a, at: ui.Offset{Dx: 10, Dy: 10}}, damageItem{pic: b, at: ui.Offset{Dx: 200, Dy: 200}})
			}},
		{name: "the same pictures in a new scene change nothing",
			scene: func() *ui.Scene {
				return sceneOf(damageItem{pic: a, at: ui.Offset{Dx: 10, Dy: 10}}, damageItem{pic: b, at: ui.Offset{Dx: 200, Dy: 200}})
			}},
		{name: "a picture recorded again damages the tiles it covers",
			scene: func() *ui.Scene {
				return sceneOf(damageItem{pic: a, at: ui.Offset{Dx: 10, Dy: 10}}, damageItem{pic: b2, at: ui.Offset{Dx: 200, Dy: 200}})
			},
			rects: []image.Rectangle{image.Rect(192, 192, 256, 256)}},
		{name: "a picture that moved damages where it was and where it is",
			scene: func() *ui.Scene {
				return sceneOf(damageItem{pic: a, at: ui.Offset{Dx: 10, Dy: 10}}, damageItem{pic: b2, at: ui.Offset{Dx: 400, Dy: 400}})
			},
			rects: []image.Rectangle{image.Rect(192, 192, 256, 256), image.Rect(384, 384, 448, 448)}},
		{name: "a picture that went away damages where it was",
			scene: func() *ui.Scene { return sceneOf(damageItem{pic: a, at: ui.Offset{Dx: 10, Dy: 10}}) },
			rects: []image.Rectangle{image.Rect(384, 384, 448, 448)}},
		{name: "opacity is a parameter of the pictures under it",
			scene: func() *ui.Scene {
				return sceneOf(damageItem{pic: a, at: ui.Offset{Dx: 10, Dy: 10}, alpha: 100})
			},
			rects: []image.Rectangle{image.Rect(0, 0, 64, 64)}},
		{name: "the same scene again is drawn whole", scene: same, full: true},
		{name: "a fill covers its clip only",
			scene: func() *ui.Scene { return sceneOf(damageItem{pic: fill, clip: &clip}) },
			rects: []image.Rectangle{image.Rect(0, 0, 64, 64), image.Rect(256, 256, 384, 384)}},
		{name: "a fill with no clip covers the surface",
			scene: func() *ui.Scene { return sceneOf(damageItem{pic: fill}) },
			full:  false, rects: nil}, // checked below: the whole surface in rectangles
		{name: "an image filter on a layer reads the pixels around it",
			scene: func() *ui.Scene { return sceneOf(damageItem{pic: blur, at: ui.Offset{Dx: 100, Dy: 100}}) },
			full:  true},
		{name: "a size change is drawn whole", size: image.Pt(256, 256), full: true,
			scene: func() *ui.Scene { return sceneOf(damageItem{pic: a}) }},
		{name: "an invalidated tracker is drawn whole", invalidate: true, full: true,
			scene: func() *ui.Scene { return sceneOf(damageItem{pic: a}) }},
		{name: "nothing changes after that",
			scene: func() *ui.Scene { return sceneOf(damageItem{pic: a}) }},
	}
	tr := NewDamageTracker()
	size := image.Point{}
	for _, s := range steps {
		if s.size != (image.Point{}) {
			size = s.size
		}
		if s.invalidate {
			tr.Invalidate()
		}
		scene := s.scene()
		last = scene
		d := tr.Update(scene, size.X, size.Y)
		if d.Full != s.full {
			t.Errorf("%s: Full = %v, want %v (rects %v)", s.name, d.Full, s.full, d.Rects)
			continue
		}
		if s.full {
			continue
		}
		if s.name == "a fill with no clip covers the surface" {
			for _, p := range []image.Point{{0, 0}, {511, 511}, {250, 17}} {
				if !covers(d, p) {
					t.Errorf("%s: %v is not damaged: %v", s.name, p, d.Rects)
				}
			}
			continue
		}
		if len(d.Rects) != len(s.rects) {
			t.Errorf("%s: rects = %v, want %v", s.name, d.Rects, s.rects)
			continue
		}
		for _, w := range s.rects {
			found := false
			for _, r := range d.Rects {
				found = found || r.Eq(w) || w.In(r)
			}
			if !found {
				t.Errorf("%s: rects = %v, want %v", s.name, d.Rects, s.rects)
			}
		}
	}
}

func TestDamageTrackerMergesManyRectangles(t *testing.T) {
	pics := make([]*ui.Picture, 0, 20)
	for i := 0; i < 20; i++ {
		pics = append(pics, rectPicture(ui.NewRectFromLTWH(0, 0, 10, 10), 0xFF000000))
	}
	build := func(pics []*ui.Picture) *ui.Scene {
		var items []damageItem
		for i, p := range pics {
			// Diagonal tiles, so that no two dirty tiles share a column.
			items = append(items, damageItem{pic: p, at: ui.Offset{Dx: float64(i * 70), Dy: float64(i * 70)}})
		}
		return sceneOf(items...)
	}
	tr := NewDamageTracker()
	tr.Update(build(pics), 2000, 2000)
	again := make([]*ui.Picture, len(pics))
	for i := range again {
		again[i] = rectPicture(ui.NewRectFromLTWH(0, 0, 10, 10), 0xFF000000)
	}
	d := tr.Update(build(again), 2000, 2000)
	if d.Full || len(d.Rects) != 1 {
		t.Fatalf("damage = %+v, want one rectangle that holds them all", d)
	}
	for i := range pics {
		if p := image.Pt(i*70+5, i*70+5); !p.In(d.Rects[0]) {
			t.Errorf("%v is outside %v", p, d.Rects[0])
		}
	}
}

func TestShadowBoundsHoldTheShadow(t *testing.T) {
	path := ui.NewPath()
	path.AddRect(ui.NewRectFromLTWH(100, 100, 80, 40))
	tests := []struct {
		name      string
		elevation float64
		want      bool
	}{
		{"flat", 0, true},
		{"low", 2, true},
		{"high", 24, true},
		{"above the light", 700, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, ok := ShadowBounds(ui.OpDrawShadow{Path: path, Elevation: tt.elevation})
			if ok != tt.want {
				t.Fatalf("ok = %v", ok)
			}
			if pb := path.GetBounds(); b.Left > pb.Left || b.Top > pb.Top || b.Right < pb.Right || b.Bottom < pb.Bottom {
				t.Errorf("bounds %v do not hold the path %v", b, pb)
			}
		})
	}
}
