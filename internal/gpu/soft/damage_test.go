package soft

import (
	"image"
	"math"
	"math/rand"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/material/fonts"
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

// recipeOp is one drawing of a picture, as data: a picture is recorded from its recipe, so
// that it can be recorded again, the same or with a change, from new objects each time.
type recipeOp struct {
	kind  int
	x, y  float64
	color uint32
}

// recipeKinds is how many kinds of drawing a recipe has.
const recipeKinds = 12

// recordRecipe records the drawings of a recipe into a new picture. Every path, paint,
// gradient, filter and paragraph is made again; the image is the same handle.
func recordRecipe(engine ui.ParagraphFactory, img *ui.Image, ops []recipeOp) *ui.Picture {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	for _, op := range ops {
		c.Save()
		c.Translate(op.x, op.y)
		switch op.kind {
		case 0:
			c.DrawRect(rectAt(0, 0, 40, 14), paint(op.color))
		case 1:
			p := paint(op.color)
			p.SetStyle(ui.PaintingStyleStroke)
			p.SetStrokeWidth(5)
			c.DrawCircle(ui.Offset{Dx: 14, Dy: 14}, 11, p)
		case 2:
			tri := ui.NewPath()
			tri.MoveTo(0, 0)
			tri.LineTo(30, 6)
			tri.LineTo(8, 22)
			tri.Close()
			c.DrawPath(tri, paint(op.color))
		case 3:
			box := ui.NewPath()
			box.AddRRect(ui.NewRRectFromRectXY(rectAt(0, 0, 36, 20), 5, 5))
			c.DrawShadow(box, ui.NewColor(op.color), 6, false)
		case 4:
			p := paint(op.color)
			p.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 3))
			c.ClipRect(rectAt(-12, -12, 60, 44))
			c.DrawRect(rectAt(0, 0, 30, 16), p)
		case 5:
			c.DrawImage(img, ui.Offset{}, ui.NewPaint())
		case 6:
			c.ClipRect(rectAt(0, 0, 34, 12))
			c.DrawColor(ui.NewColor(op.color), ui.BlendModeSrcOver)
		case 7:
			c.SaveLayer(ui.Ptr(rectAt(0, 0, 44, 20)), blended(0x80336699, ui.BlendModeMultiply))
			c.DrawOval(rectAt(0, 0, 44, 20), paint(op.color))
			c.Restore()
		case 8:
			c.Rotate(0.5)
			c.DrawRRect(ui.NewRRectFromRectXY(rectAt(0, 0, 36, 12), 3, 3), paint(op.color))
		case 9:
			p := ui.NewPaint()
			p.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 40}, []ui.Color{ui.NewColor(op.color), ui.NewColor(0xFFFFFFFF)}))
			c.DrawRect(rectAt(0, 0, 40, 12), p)
		case 10:
			b := ui.NewParagraphBuilder(engine, ui.NewParagraphStyle(ui.ParagraphStyleOpts{}))
			b.PushStyle(ui.NewTextStyle(ui.TextStyleOpts{FontFamily: fonts.RobotoFamily, FontSize: ui.Ptr(12.0), Color: ui.Ptr(ui.NewColor(op.color))}))
			b.AddText([]string{"damage", "a row", "typed", "Glutter"}[op.color%4])
			p := b.Build()
			p.Layout(ui.NewParagraphConstraints(80))
			c.DrawParagraph(p, ui.Offset{})
		case 11:
			p := paint(op.color)
			p.SetStyle(ui.PaintingStyleStroke)
			c.DrawLine(ui.Offset{}, ui.Offset{Dx: 40}, p)
		}
		c.Restore()
	}
	return rec.EndRecording()
}

// TestDamageOfPicturesRecordedAgainCoversEveryChangedPixel draws random pairs of scenes
// whose pictures were all recorded again, some with a drawing changed, moved, added or
// removed and some the same, and checks that every pixel that differs between the two
// frames is inside the damage of the second, that a frame that draws the same has no
// damage, and that the damage is a small part of the pictures when little changed.
func TestDamageOfPicturesRecordedAgainCoversEveryChangedPixel(t *testing.T) {
	const w, h = 320, 280
	rng := rand.New(rand.NewSource(45))
	engine := proof.BundledTextEngine()
	pix := make([]byte, 4*12*12)
	for i := range pix {
		pix[i] = byte(40 + i%200)
	}
	img := ui.NewImageFromRGBA(pix, 12, 12)
	randomOp := func() recipeOp {
		return recipeOp{kind: rng.Intn(recipeKinds), x: float64(rng.Intn(150)), y: float64(rng.Intn(130)), color: 0xFF000000 | uint32(rng.Intn(1<<24))}
	}
	type placed struct {
		at    ui.Offset
		alpha int
		clip  *ui.Rect
		ops   []recipeOp
	}
	build := func(parts []placed) *ui.Scene {
		b := ui.NewSceneBuilder()
		for _, p := range parts {
			if p.clip != nil {
				b.PushClipRect(*p.clip)
			}
			if p.alpha != 0 {
				b.PushOpacity(p.alpha)
			}
			b.AddPicture(p.at, recordRecipe(engine, img, p.ops))
			if p.alpha != 0 {
				b.Pop()
			}
			if p.clip != nil {
				b.Pop()
			}
		}
		return b.Build()
	}
	var same, partial, small int
	const iterations = 300
	for iter := 0; iter < iterations; iter++ {
		parts := make([]placed, 1+rng.Intn(3))
		for i := range parts {
			parts[i].at = ui.Offset{Dx: float64(rng.Intn(100)), Dy: float64(rng.Intn(90))}
			if rng.Intn(4) == 0 {
				parts[i].alpha = 60 + rng.Intn(180)
			}
			if rng.Intn(4) == 0 {
				c := rectAt(float64(rng.Intn(80)), float64(rng.Intn(70)), float64(120+rng.Intn(120)), float64(100+rng.Intn(100)))
				parts[i].clip = &c
			}
			parts[i].ops = make([]recipeOp, 3+rng.Intn(8))
			for j := range parts[i].ops {
				parts[i].ops[j] = randomOp()
			}
		}
		// The second frame: every picture recorded again, with at most one drawing of
		// one picture changed.
		after := make([]placed, len(parts))
		copy(after, parts)
		changes := 0
		if iter%4 != 0 {
			i := rng.Intn(len(after))
			ops := append([]recipeOp(nil), after[i].ops...)
			j := rng.Intn(len(ops))
			switch rng.Intn(5) {
			case 0:
				ops[j].color ^= 0x00FFFFFF
			case 1:
				ops[j].x += float64(rng.Intn(40) - 20)
				ops[j].y += float64(rng.Intn(40) - 20)
			case 2:
				ops = append(ops[:j], ops[j+1:]...)
			case 3:
				ops = append(ops[:j], append([]recipeOp{randomOp()}, ops[j:]...)...)
			case 4:
				ops[j].kind = (ops[j].kind + 1 + rng.Intn(recipeKinds-1)) % recipeKinds
			}
			after[i].ops = ops
			changes = 1
		}
		sa, sb := build(parts), build(after)
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
		if d.Full {
			t.Fatalf("iteration %d: the whole surface is damaged", iter)
		}
		if changes == 0 {
			same++
			if len(d.Rects) != 0 {
				t.Fatalf("iteration %d: the pictures recorded again draw the same, and the damage is %v", iter, d.Rects)
			}
		} else if len(d.Rects) > 0 {
			partial++
			area := 0
			for _, r := range d.Rects {
				area += r.Dx() * r.Dy()
			}
			if area*3 < w*h {
				small++
			}
		}
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
						iter, x, y, ia.Pix[o:o+4], ib.Pix[o:o+4], d.Rects, parts, after)
				}
			}
		}
	}
	if same != iterations/4 {
		t.Errorf("%d frames drew the same, want %d", same, iterations/4)
	}
	if partial < iterations/2 || small < partial/2 {
		t.Errorf("%d of %d changed frames had damage, %d of them under a third of the surface; the test checks too little", partial, iterations-same, small)
	}
}
