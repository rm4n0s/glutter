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
	bAgain := rectPicture(ui.NewRectFromLTWH(0, 0, 40, 40), 0xFF00FF00) // recorded again, the same
	b2 := rectPicture(ui.NewRectFromLTWH(0, 0, 40, 40), 0xFF00FF80)     // recorded again, another color
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
		{name: "a picture recorded again that draws the same changes nothing",
			scene: func() *ui.Scene {
				return sceneOf(damageItem{pic: a, at: ui.Offset{Dx: 10, Dy: 10}}, damageItem{pic: bAgain, at: ui.Offset{Dx: 200, Dy: 200}})
			}},
		{name: "a picture recorded again that draws something else damages the tiles it covers",
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
		again[i] = rectPicture(ui.NewRectFromLTWH(0, 0, 10, 10), 0xFF000080)
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

// rowsPicture records a picture of rows, as a list or the lines of an editor are: a
// background, and for each row a rectangle 20 pixels high from the top of the picture down.
// edit changes the canvas calls of one row.
func rowsPicture(rows int, draw func(c *ui.Canvas, row int, box ui.Rect)) *ui.Picture {
	return pictureOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 600, float64(rows*20)), paintOf(0xFFFFFFFF))
		for row := 0; row < rows; row++ {
			draw(c, row, ui.NewRectFromLTWH(10, float64(row*20)+2, 400, 16))
		}
	})
}

// tilesOf is the rectangle of the tiles of 64 pixels that hold the pixels from x0, y0 up
// to x1, y1.
func tilesOf(x0, y0, x1, y1 int) image.Rectangle {
	return image.Rect(x0/damageTile*damageTile, y0/damageTile*damageTile, (x1+damageTile-1)/damageTile*damageTile, (y1+damageTile-1)/damageTile*damageTile)
}

// TestDamageOfAPictureRecordedAgain records a picture of 30 rows again with one thing
// changed, and checks that the damage is the tiles of what changed and not the picture.
func TestDamageOfAPictureRecordedAgain(t *testing.T) {
	const w, h = 640, 640
	plain := func(c *ui.Canvas, row int, box ui.Rect) { c.DrawRect(box, paintOf(0xFF202020)) }
	// only makes a drawing that differs from plain in one row.
	only := func(row int, draw func(c *ui.Canvas, box ui.Rect)) func(c *ui.Canvas, row int, box ui.Rect) {
		return func(c *ui.Canvas, r int, box ui.Rect) {
			if r == row {
				draw(c, box)
				return
			}
			plain(c, r, box)
		}
	}
	gradient := func(to float64) *ui.Paint {
		p := ui.NewPaint()
		p.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: to}, []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF0000FF)}))
		return p
	}
	// Row 12 is the pixels from y 242 to 258, x 10 to 410: the tiles from 192 to 320 down,
	// 0 to 448 across.
	row12 := tilesOf(10, 242, 410, 258)
	tests := []struct {
		name   string
		before func(c *ui.Canvas, row int, box ui.Rect)
		after  func(c *ui.Canvas, row int, box ui.Rect)
		// want holds the damage; nil means that there is none.
		want []image.Rectangle
		// notIn are pixels that must not be damaged.
		notIn []image.Point
	}{
		{name: "the same drawing", before: plain, after: plain},
		{name: "the color of a row", before: plain,
			after: only(12, func(c *ui.Canvas, box ui.Rect) { c.DrawRect(box, paintOf(0xFF2020F0)) }),
			want:  []image.Rectangle{row12}, notIn: []image.Point{{20, 100}, {20, 400}, {500, 250}}},
		{name: "a row that is shorter", before: plain,
			after: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.DrawRect(ui.NewRectFromLTRB(box.Left, box.Top, 100, box.Bottom), paintOf(0xFF202020))
			}),
			want: []image.Rectangle{row12}, notIn: []image.Point{{20, 100}, {20, 400}}},
		{name: "a row that draws nothing", before: plain,
			after: only(12, func(*ui.Canvas, ui.Rect) {}),
			want:  []image.Rectangle{row12}, notIn: []image.Point{{20, 100}, {20, 400}}},
		{name: "a row that draws something more", before: plain,
			after: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.DrawRect(box, paintOf(0xFF202020))
				c.DrawCircle(ui.Offset{Dx: 500, Dy: 250}, 6, paintOf(0xFFFF0000))
			}),
			want: []image.Rectangle{tilesOf(493, 243, 507, 257)}, notIn: []image.Point{{20, 250}, {20, 100}}},
		{name: "a row that moved", before: plain,
			after: only(3, func(c *ui.Canvas, box ui.Rect) { c.DrawRect(box.Shift(ui.Offset{Dx: 150}), paintOf(0xFF202020)) }),
			want:  []image.Rectangle{tilesOf(10, 62, 560, 78)}, notIn: []image.Point{{20, 200}, {600, 20}}},
		{name: "a row under another transform", before: plain,
			after: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.Save()
				c.Translate(0, 1)
				c.DrawRect(box, paintOf(0xFF202020))
				c.Restore()
			}),
			want: []image.Rectangle{row12}, notIn: []image.Point{{20, 100}, {20, 400}}},
		{name: "a row under another clip",
			before: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.Save()
				c.ClipRect(ui.NewRectFromLTRB(0, box.Top, 300, box.Bottom))
				c.DrawRect(box, paintOf(0xFF202020))
				c.Restore()
			}),
			after: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.Save()
				c.ClipRect(ui.NewRectFromLTRB(0, box.Top, 200, box.Bottom))
				c.DrawRect(box, paintOf(0xFF202020))
				c.Restore()
			}),
			want: []image.Rectangle{tilesOf(10, 242, 300, 258)}, notIn: []image.Point{{20, 100}, {400, 250}}},
		{name: "a row in a layer of another alpha",
			before: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.SaveLayer(&box, paintOf(0x80000000))
				c.DrawRect(box, paintOf(0xFF202020))
				c.Restore()
			}),
			after: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.SaveLayer(&box, paintOf(0x40000000))
				c.DrawRect(box, paintOf(0xFF202020))
				c.Restore()
			}),
			want: []image.Rectangle{row12}, notIn: []image.Point{{20, 100}, {20, 400}}},
		{name: "a gradient made again",
			before: only(12, func(c *ui.Canvas, box ui.Rect) { c.DrawRect(box, gradient(400)) }),
			after:  only(12, func(c *ui.Canvas, box ui.Rect) { c.DrawRect(box, gradient(400)) })},
		{name: "another gradient",
			before: only(12, func(c *ui.Canvas, box ui.Rect) { c.DrawRect(box, gradient(400)) }),
			after:  only(12, func(c *ui.Canvas, box ui.Rect) { c.DrawRect(box, gradient(300)) }),
			want:   []image.Rectangle{row12}, notIn: []image.Point{{20, 100}, {20, 400}}},
		{name: "two rows that swap what they draw over each other",
			before: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.DrawRect(box, paintOf(0xFFFF0000))
				c.DrawRect(box, paintOf(0xFF0000FF))
			}),
			after: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.DrawRect(box, paintOf(0xFF0000FF))
				c.DrawRect(box, paintOf(0xFFFF0000))
			}),
			want: []image.Rectangle{row12}, notIn: []image.Point{{20, 100}, {20, 400}}},
		{name: "a hairline along an axis that moved",
			before: only(12, func(c *ui.Canvas, box ui.Rect) {
				p := paintOf(0xFF000000)
				p.SetStyle(ui.PaintingStyleStroke)
				c.DrawLine(ui.Offset{Dx: 10, Dy: 250}, ui.Offset{Dx: 60, Dy: 250}, p)
			}),
			after: only(12, func(c *ui.Canvas, box ui.Rect) {
				p := paintOf(0xFF000000)
				p.SetStyle(ui.PaintingStyleStroke)
				c.DrawLine(ui.Offset{Dx: 10, Dy: 330}, ui.Offset{Dx: 60, Dy: 330}, p)
			}),
			want: []image.Rectangle{tilesOf(10, 249, 60, 251), tilesOf(10, 329, 60, 331)}, notIn: []image.Point{{20, 100}, {200, 250}}},
		{name: "a fill of the clip of a row",
			before: plain,
			after: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.Save()
				c.ClipRect(box)
				c.DrawColor(ui.NewColor(0xFF00FF00), ui.BlendModeSrcOver)
				c.Restore()
			}),
			want: []image.Rectangle{row12}, notIn: []image.Point{{20, 100}, {20, 400}}},
		{name: "a picture in the picture",
			before: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.DrawPicture(pictureOf(func(c *ui.Canvas) { c.DrawRect(box, paintOf(0xFF102030)) }))
			}),
			after: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.DrawPicture(pictureOf(func(c *ui.Canvas) { c.DrawRect(box, paintOf(0xFF102031)) }))
			}),
			want: []image.Rectangle{row12}, notIn: []image.Point{{20, 100}, {20, 400}}},
		{name: "the same picture in the picture, made again",
			before: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.DrawPicture(pictureOf(func(c *ui.Canvas) { c.DrawRect(box, paintOf(0xFF102030)) }))
			}),
			after: only(12, func(c *ui.Canvas, box ui.Rect) {
				c.DrawPicture(pictureOf(func(c *ui.Canvas) { c.DrawRect(box, paintOf(0xFF102030)) }))
			})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewDamageTracker()
			at := ui.Offset{Dx: 0, Dy: 0}
			tr.Update(sceneOf(damageItem{pic: rowsPicture(30, tt.before), at: at}), w, h)
			d := tr.Update(sceneOf(damageItem{pic: rowsPicture(30, tt.after), at: at}), w, h)
			if d.Full {
				t.Fatalf("the whole surface is damaged")
			}
			if tt.want == nil {
				if len(d.Rects) != 0 {
					t.Errorf("damage = %v, want none", d.Rects)
				}
				return
			}
			for _, want := range tt.want {
				for _, p := range []image.Point{want.Min, {X: want.Max.X - 1, Y: want.Max.Y - 1}} {
					if !covers(d, p) {
						t.Errorf("%v is not damaged: %v, want %v", p, d.Rects, tt.want)
					}
				}
			}
			for _, p := range tt.notIn {
				if covers(d, p) {
					t.Errorf("%v is damaged: %v, want only %v", p, d.Rects, tt.want)
				}
			}
			// The damage is what was asked for and no more.
			area := 0
			for _, r := range d.Rects {
				area += r.Dx() * r.Dy()
			}
			wantArea := 0
			for _, r := range tt.want {
				wantArea += r.Dx() * r.Dy()
			}
			if area > wantArea {
				t.Errorf("damage = %v (%d pixels), want %v (%d pixels)", d.Rects, area, tt.want, wantArea)
			}
		})
	}
}

// mutePainter is a paragraph layout that cannot draw itself, so that it has no signature.
type mutePainter struct{ ui.ParagraphLayout }

// TestAnOperationWithoutASignatureIsDamagedWhenRecordedAgain checks the rule for what the
// tracker cannot compare: it is taken by the identity of its picture, as every operation
// was before step 45.2.
func TestAnOperationWithoutASignatureIsDamagedWhenRecordedAgain(t *testing.T) {
	para := ui.NewParagraph(&mutePainter{})
	record := func() *ui.Picture {
		return pictureOf(func(c *ui.Canvas) {
			c.DrawRect(ui.NewRectFromLTWH(300, 300, 20, 20), paintOf(0xFF000000))
			c.Save()
			c.ClipRect(ui.NewRectFromLTWH(10, 10, 30, 30))
			c.DrawParagraph(para, ui.Offset{Dx: 10, Dy: 10})
			c.Restore()
		})
	}
	tr := NewDamageTracker()
	pic := record()
	tr.Update(sceneOf(damageItem{pic: pic}), 512, 512)
	if d := tr.Update(sceneOf(damageItem{pic: pic}), 512, 512); d.Full || len(d.Rects) != 0 {
		t.Errorf("the same picture: damage = %+v, want none", d)
	}
	d := tr.Update(sceneOf(damageItem{pic: record()}), 512, 512)
	if d.Full || !covers(d, image.Pt(20, 20)) || covers(d, image.Pt(310, 310)) {
		t.Errorf("the picture recorded again: damage = %+v, want the clip of the paragraph and not the rectangle", d)
	}
}

// TestDamageOfAPictureOfManyOperations records a picture of more operations than the
// tracker keeps an item for, which it folds into a grid: the damage of a change is still
// near the change, the same picture again still damages nothing, and the order of two
// operations still counts.
func TestDamageOfAPictureOfManyOperations(t *testing.T) {
	const w, h = 1024, 1024
	const cols, rows = 50, 50 // 2,500 squares of 16 pixels, 20 apart
	record := func(changed int, argb uint32, fillFirst bool) *ui.Picture {
		return pictureOf(func(c *ui.Canvas) {
			if fillFirst {
				c.DrawColor(ui.NewColor(0x20FF0000), ui.BlendModeSrcOver)
			}
			for i := 0; i < cols*rows; i++ {
				color := uint32(0xFF404040)
				if i == changed {
					color = argb
				}
				c.DrawRect(ui.NewRectFromLTWH(float64(i%cols*20), float64(i/cols*20), 16, 16), paintOf(color))
			}
			if !fillFirst {
				c.DrawColor(ui.NewColor(0x20FF0000), ui.BlendModeSrcOver)
			}
		})
	}
	base := record(-1, 0, true)
	tr := NewDamageTracker()
	wk := &damageWalk{t: tr, cur: map[*ui.Picture]*pictureInfo{}}
	if n := len(wk.measure(base).items); n > maxPictureItems {
		t.Fatalf("the picture has %d items, want at most %d", n, maxPictureItems)
	}
	// Square 1275 is column 25, row 25: the pixels from 500 to 516 both ways.
	tests := []struct {
		name    string
		picture *ui.Picture
		in      []image.Point
		notIn   []image.Point
	}{
		{name: "the same drawing", picture: record(-1, 0, true)},
		{name: "one square of another color", picture: record(1275, 0xFFFF0000, true),
			in: []image.Point{{505, 505}}, notIn: []image.Point{{100, 100}, {900, 505}, {505, 900}, {300, 505}}},
		{name: "the fill of the clip after the squares", picture: record(-1, 0, false),
			in: []image.Point{{5, 5}, {505, 505}, {990, 990}, {1010, 1010}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewDamageTracker()
			tr.Update(sceneOf(damageItem{pic: base}), w, h)
			d := tr.Update(sceneOf(damageItem{pic: tt.picture}), w, h)
			if d.Full {
				t.Fatal("the whole surface is damaged")
			}
			if tt.in == nil && len(d.Rects) != 0 {
				t.Errorf("damage = %v, want none", d.Rects)
			}
			for _, p := range tt.in {
				if !covers(d, p) {
					t.Errorf("%v is not damaged: %v", p, d.Rects)
				}
			}
			for _, p := range tt.notIn {
				if covers(d, p) {
					t.Errorf("%v is damaged: %v", p, d.Rects)
				}
			}
		})
	}
}

// BenchmarkDamageOfKeptPictures is an Update of a scene whose pictures were all kept: what
// every frame of a window pays for the damage. Run it with
//
//	go test ./internal/gpu -run xxx -bench DamageOfKeptPictures -benchmem
func BenchmarkDamageOfKeptPictures(b *testing.B) {
	var items []damageItem
	for i := 0; i < 40; i++ {
		pic := rowsPicture(30, func(c *ui.Canvas, row int, box ui.Rect) { c.DrawRect(box, paintOf(0xFF202020)) })
		items = append(items, damageItem{pic: pic, at: ui.Offset{Dx: float64(i % 8 * 240), Dy: float64(i / 8 * 216)}})
	}
	tr := NewDamageTracker()
	scenes := [2]*ui.Scene{sceneOf(items...), sceneOf(items...)}
	tr.Update(scenes[0], 1920, 1080)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if d := tr.Update(scenes[(i+1)%2], 1920, 1080); d.Full || len(d.Rects) != 0 {
			b.Fatalf("damage = %+v", d)
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
