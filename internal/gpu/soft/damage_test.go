package soft

import (
	"image"
	"math"
	"math/rand"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// damagePool makes the pictures that the random scenes are built from: shapes,
// strokes, a shadow, a blurred shape, an image, a fill of the clip, a layer with a
// blend mode and a path.
func damagePool(rng *rand.Rand) []*ui.Picture {
	record := func(fn func(c *ui.Canvas)) *ui.Picture {
		rec := ui.NewPictureRecorder()
		fn(ui.NewCanvas(rec))
		return rec.EndRecording()
	}
	color := func() uint32 { return 0xFF000000 | uint32(rng.Intn(1<<24)) }
	stroke := func(w float64) *ui.Paint {
		p := paint(color())
		p.SetStyle(ui.PaintingStyleStroke)
		p.SetStrokeWidth(w)
		return p
	}
	pix := make([]byte, 4*12*12)
	for i := range pix {
		pix[i] = byte(40 + i%200)
	}
	img := ui.NewImageFromRGBA(pix, 12, 12)
	blurred := paint(color())
	blurred.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 4))
	zig := ui.NewPath()
	zig.MoveTo(0, 0)
	zig.LineTo(30, 5)
	zig.LineTo(5, 20)
	zig.LineTo(40, 40)
	box := ui.NewPath()
	box.AddRRect(ui.RRect{Left: 0, Top: 0, Right: 50, Bottom: 30, TlRadiusX: 6, TlRadiusY: 6, TrRadiusX: 6, TrRadiusY: 6, BrRadiusX: 6, BrRadiusY: 6, BlRadiusX: 6, BlRadiusY: 6})
	return []*ui.Picture{
		record(func(c *ui.Canvas) { c.DrawRect(rectAt(0, 0, 40, 30), paint(color())) }),
		record(func(c *ui.Canvas) { c.DrawCircle(ui.Offset{Dx: 20, Dy: 20}, 18, stroke(7)) }),
		record(func(c *ui.Canvas) { c.DrawPath(zig, stroke(5)) }),
		record(func(c *ui.Canvas) { c.DrawShadow(box, ui.NewColor(0xFF000000), float64(2+rng.Intn(14)), false) }),
		record(func(c *ui.Canvas) { c.DrawRect(rectAt(5, 5, 30, 30), blurred) }),
		record(func(c *ui.Canvas) { c.DrawImage(img, ui.Offset{Dx: 3, Dy: 4}, ui.NewPaint()) }),
		record(func(c *ui.Canvas) { c.DrawPaint(paint(0x40FF8000)) }),
		record(func(c *ui.Canvas) {
			c.SaveLayer(nil, blended(0x80336699, ui.BlendModeMultiply))
			c.DrawOval(rectAt(0, 0, 50, 24), paint(color()))
			c.Restore()
		}),
		record(func(c *ui.Canvas) {
			c.Save()
			c.Translate(30, 10)
			c.Rotate(0.6)
			c.DrawRRect(ui.RRect{Left: 0, Top: 0, Right: 40, Bottom: 14, TlRadiusX: 3, TlRadiusY: 3, TrRadiusX: 3, TrRadiusY: 3, BrRadiusX: 3, BrRadiusY: 3, BlRadiusX: 3, BlRadiusY: 3}, paint(color()))
			c.Restore()
		}),
	}
}

type damageSpec struct {
	pic   int
	at    ui.Offset
	alpha int
	clip  *ui.Rect
	cf    *ui.ColorFilter
}

func buildDamageScene(pool []*ui.Picture, specs []damageSpec) *ui.Scene {
	b := ui.NewSceneBuilder()
	for _, s := range specs {
		if s.clip != nil {
			b.PushClipRect(*s.clip)
		}
		if s.alpha != 0 {
			b.PushOpacity(s.alpha)
		}
		if s.cf != nil {
			b.PushColorFilter(s.cf)
		}
		b.AddPicture(s.at, pool[s.pic])
		if s.cf != nil {
			b.Pop()
		}
		if s.alpha != 0 {
			b.Pop()
		}
		if s.clip != nil {
			b.Pop()
		}
	}
	return b.Build()
}

func randomSpec(rng *rand.Rand, n int, w, h int) damageSpec {
	s := damageSpec{pic: rng.Intn(n), at: ui.Offset{Dx: float64(rng.Intn(w - 30)), Dy: float64(rng.Intn(h - 30))}}
	if rng.Intn(3) == 0 {
		s.alpha = 40 + rng.Intn(200)
	}
	if rng.Intn(3) == 0 {
		c := rectAt(float64(rng.Intn(w/2)), float64(rng.Intn(h/2)), float64(30+rng.Intn(w/2)), float64(30+rng.Intn(h/2)))
		s.clip = &c
	}
	return s
}

// TestDamageCoversEveryChangedPixel draws random pairs of scenes that share most of
// their pictures, and checks that every pixel that differs between the two frames is
// inside the damage that the tracker reports for the second.
func TestDamageCoversEveryChangedPixel(t *testing.T) {
	const w, h = 300, 260
	rng := rand.New(rand.NewSource(7))
	pool := damagePool(rng)
	cf := ui.NewColorFilterMode(ui.NewColor(0x40FF0000), ui.BlendModeSrcOver)
	var damaged, total int
	for iter := 0; iter < 250; iter++ {
		n := 2 + rng.Intn(5)
		a := make([]damageSpec, n)
		for i := range a {
			a[i] = randomSpec(rng, len(pool), w, h)
			if rng.Intn(8) == 0 {
				a[i].cf = cf
			}
		}
		b := append([]damageSpec(nil), a...)
		for i := range b {
			switch rng.Intn(5) {
			case 0:
				b[i] = randomSpec(rng, len(pool), w, h)
			case 1:
				b[i].at.Dx += float64(rng.Intn(60) - 30)
				b[i].at.Dy += float64(rng.Intn(60) - 30)
			case 2:
				b[i].alpha = 40 + rng.Intn(200)
			}
		}
		if rng.Intn(3) == 0 {
			b = b[:len(b)-1]
		}
		sa, sb := buildDamageScene(pool, a), buildDamageScene(pool, b)
		ia, err := Render(sa, w, h)
		if err != nil {
			t.Fatal(err)
		}
		ib, err := Render(sb, w, h)
		if err != nil {
			t.Fatal(err)
		}
		tr := gpu.NewDamageTracker()
		tr.Update(sa, w, h)
		d := tr.Update(sb, w, h)
		total++
		if d.Full {
			continue
		}
		damaged++
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				o := y*ia.Stride + x*4
				if string(ia.Pix[o:o+4]) == string(ib.Pix[o:o+4]) {
					continue
				}
				in := false
				for _, r := range d.Rects {
					in = in || image.Pt(x, y).In(r)
				}
				if !in {
					t.Fatalf("iteration %d: pixel (%d, %d) changed (%v to %v) outside the damage %v\nfrom %+v\nto %+v",
						iter, x, y, ia.Pix[o:o+4], ib.Pix[o:o+4], d.Rects, a, b)
				}
			}
		}
	}
	if damaged < total/2 {
		t.Errorf("only %d of %d random frames had partial damage; the test checks too little", damaged, total)
	}
}

// TestShadowBoundsHoldTheDrawnShadow draws shadows of several elevations with soft and
// checks that the pixels they change are inside gpu.ShadowBounds.
func TestShadowBoundsHoldTheDrawnShadow(t *testing.T) {
	for _, elevation := range []float64{1, 4, 8, 16, 24, 48} {
		path := ui.NewPath()
		path.AddRRect(ui.RRect{Left: 150, Top: 120, Right: 230, Bottom: 160, TlRadiusX: 8, TlRadiusY: 8, TrRadiusX: 8, TrRadiusY: 8, BrRadiusX: 8, BrRadiusY: 8, BlRadiusX: 8, BlRadiusY: 8})
		rec := ui.NewPictureRecorder()
		ui.NewCanvas(rec).DrawShadow(path, ui.NewColor(0xFF000000), elevation, false)
		b := ui.NewSceneBuilder()
		b.AddPicture(ui.Offset{}, rec.EndRecording())
		img, err := Render(b.Build(), 400, 400)
		if err != nil {
			t.Fatal(err)
		}
		bounds, ok := gpu.ShadowBounds(ui.OpDrawShadow{Path: path, Elevation: elevation})
		if !ok {
			t.Fatal("no bounds")
		}
		for y := 0; y < 400; y++ {
			for x := 0; x < 400; x++ {
				if img.Pix[y*img.Stride+x*4+3] == 0 {
					continue
				}
				if float64(x) < math.Floor(bounds.Left) || float64(x) >= math.Ceil(bounds.Right) || float64(y) < math.Floor(bounds.Top) || float64(y) >= math.Ceil(bounds.Bottom) {
					t.Fatalf("elevation %v: pixel (%d, %d) is drawn outside the bounds %v", elevation, x, y, bounds)
				}
			}
		}
	}
}
