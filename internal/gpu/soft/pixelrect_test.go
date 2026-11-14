package soft

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"math/rand"
	"runtime"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// The rectangles on whole pixels (step 45.1): a clip to one and a fill of one make no
// coverage mask. The mask path stays for every other shape, and is the reference here: the
// same rectangle given as a path is rasterized, and must give the same clip and the same
// pixels.

func TestPixelRect(t *testing.T) {
	translated := func(x, y float32) *ui.Matrix4 { return ui.NewMatrix4TranslationValues(x, y, 0) }
	scaled := func(x, y float32) *ui.Matrix4 {
		m := ui.NewMatrix4Identity()
		m.Scale(x, y, 1)
		return m
	}
	rotated := ui.NewMatrix4Identity()
	rotated.RotateZ(math.Pi / 2)
	skewed := ui.NewMatrix4Identity()
	skewed.SetEntry(0, 1, 0.5)
	perspective := ui.NewMatrix4Identity()
	perspective.SetEntry(3, 0, 0.001)
	tests := []struct {
		name string
		m    *ui.Matrix4
		r    ui.Rect
		want image.Rectangle
		ok   bool
	}{
		{"whole numbers", ui.NewMatrix4Identity(), ui.NewRectFromLTRB(2, 3, 7, 9), image.Rect(2, 3, 7, 9), true},
		{"a whole translation", translated(10, -4), ui.NewRectFromLTRB(2, 3, 7, 9), image.Rect(12, -1, 17, 5), true},
		{"a half translation", translated(0.5, 0), ui.NewRectFromLTRB(2, 3, 7, 9), image.Rectangle{}, false},
		{"halves that a translation makes whole", translated(0.5, 0.5), ui.NewRectFromLTRB(1.5, 2.5, 6.5, 8.5), image.Rect(2, 3, 7, 9), true},
		{"a fractional left edge", ui.NewMatrix4Identity(), ui.NewRectFromLTRB(2.25, 3, 7, 9), image.Rectangle{}, false},
		{"a fractional bottom edge", ui.NewMatrix4Identity(), ui.NewRectFromLTRB(2, 3, 7, 9.5), image.Rectangle{}, false},
		{"a scale that keeps the edges whole", scaled(2, 3), ui.NewRectFromLTRB(1, 1, 2.5, 3), image.Rect(2, 3, 5, 9), true},
		{"a scale of a quarter more", scaled(1.25, 1.25), ui.NewRectFromLTRB(4, 8, 12, 16), image.Rect(5, 10, 15, 20), true},
		{"a scale that makes fractions", scaled(1.25, 1.25), ui.NewRectFromLTRB(1, 8, 12, 16), image.Rectangle{}, false},
		{"a flip", scaled(-1, 1), ui.NewRectFromLTRB(2, 3, 7, 9), image.Rect(-7, 3, -2, 9), true},
		{"an empty rectangle", ui.NewMatrix4Identity(), ui.NewRectFromLTRB(3, 3, 3, 8), image.Rect(3, 3, 3, 8), true},
		{"a quarter turn", rotated, ui.NewRectFromLTRB(2, 3, 7, 9), image.Rectangle{}, false},
		{"a skew", skewed, ui.NewRectFromLTRB(2, 3, 7, 9), image.Rectangle{}, false},
		{"perspective", perspective, ui.NewRectFromLTRB(2, 3, 7, 9), image.Rectangle{}, false},
		{"not a number", ui.NewMatrix4Identity(), ui.NewRectFromLTRB(math.NaN(), 3, 7, 9), image.Rectangle{}, false},
		{"infinite", ui.NewMatrix4Identity(), ui.NewRectFromLTRB(2, 3, math.Inf(1), 9), image.Rectangle{}, false},
		{"past the rasterizer's coordinates", ui.NewMatrix4Identity(), ui.NewRectFromLTRB(0, 0, 1e9, 9), image.Rectangle{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pixelRect(tt.m, tt.r)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Errorf("pixelRect = %v, %v; want %v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

// sameClip says whether two clips let the same through (both empty, or the same bounds and
// the same coverage): it returns what differs, or nothing.
func sameClip(a, b clip) string {
	if a.empty() || b.empty() {
		if a.empty() != b.empty() {
			return fmt.Sprintf("empty = %v and %v", a.empty(), b.empty())
		}
		return ""
	}
	if a.bounds != b.bounds {
		return fmt.Sprintf("bounds = %v and %v", a.bounds, b.bounds)
	}
	if (a.mask == nil) != (b.mask == nil) {
		return fmt.Sprintf("has a mask = %v and %v", a.mask != nil, b.mask != nil)
	}
	if a.mask == nil {
		return ""
	}
	if a.mask.Rect != b.mask.Rect {
		return fmt.Sprintf("mask rect = %v and %v", a.mask.Rect, b.mask.Rect)
	}
	for i, v := range a.mask.Cov {
		if v != b.mask.Cov[i] {
			return fmt.Sprintf("coverage %d = %v and %v", i, v, b.mask.Cov[i])
		}
	}
	return ""
}

// randomRectCase returns a transform that scales along the axes and translates, and a
// rectangle that it puts on whole pixels (whole) or not. A wide rectangle covers the
// middle of a surface of about 100 pixels; the others lie anywhere around it.
func randomRectCase(rng *rand.Rand, whole, wide bool) (*ui.Matrix4, ui.Rect) {
	scales := []float32{1, 1, 1, 2, 0.5, 1.25, -1}
	sx, sy := scales[rng.Intn(len(scales))], scales[rng.Intn(len(scales))]
	tx, ty := float64(rng.Intn(41)-20), float64(rng.Intn(41)-20)
	m := ui.NewMatrix4TranslationValues(float32(tx), float32(ty), 0)
	m.Scale(sx, sy, 1)
	// An edge in device pixels, a multiple of 20, taken back through the transform (which
	// is exact for these scales).
	edge := func(scale float32, shift float64, from, n int) float64 {
		v := float64(from+rng.Intn(n)) * 20
		if !whole {
			v += []float64{0.5, 0.25, 0.3, 0.75}[rng.Intn(4)]
		}
		return (v - shift) / float64(scale)
	}
	var l, tp, r, b float64
	if wide {
		l, tp, r, b = edge(sx, tx, -2, 4), edge(sy, ty, -2, 4), edge(sx, tx, 3, 4), edge(sy, ty, 3, 4)
	} else {
		l, tp, r, b = edge(sx, tx, -6, 21), edge(sy, ty, -6, 21), edge(sx, tx, -6, 21), edge(sy, ty, -6, 21)
	}
	return m, ui.NewRectFromLTRB(min(l, r), min(tp, b), max(l, r), max(tp, b))
}

func TestRectClipIsTheClipOfTheRasterizedRectangle(t *testing.T) {
	surface := image.Rect(0, 0, 96, 80)
	rounded := ui.NewPath()
	rounded.AddRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(6.5, 4.25, 80, 70), 18, 14))
	identity := &state{m: ui.NewMatrix4Identity(), clip: newClip(surface)}
	priors := []struct {
		name string
		clip clip
	}{
		{"the surface", newClip(surface)},
		{"a part of the surface", newClip(image.Rect(10, 12, 70, 60))},
		{"a rounded rectangle", shapeClip(identity, rounded, true, false)},
		{"nothing", clip{}},
	}
	if priors[2].clip.mask == nil {
		t.Fatal("the rounded clip should have a mask")
	}
	for _, prior := range priors {
		for _, whole := range []bool{true, false} {
			for _, antiAlias := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/whole=%v/antiAlias=%v", prior.name, whole, antiAlias), func(t *testing.T) {
					rng := rand.New(rand.NewSource(45))
					for i := 0; i < 200; i++ {
						m, r := randomRectCase(rng, whole, i%2 == 0)
						st := &state{m: m, clip: prior.clip}
						_, isWhole := pixelRect(m, r)
						if whole && !isWhole {
							t.Fatalf("case %d: %v under %v should be on whole pixels", i, r, m.Storage())
						}
						got := rectClip(st, r, antiAlias)
						want := shapeClip(st, rectPath(r), antiAlias, false)
						if diff := sameClip(got, want); diff != "" {
							t.Fatalf("case %d: rect %v under %v: %s", i, r, m.Storage(), diff)
						}
						if isWhole && prior.clip.mask == nil && got.mask != nil {
							t.Fatalf("case %d: a rectangle on whole pixels made a mask", i)
						}
					}
				})
			}
		}
	}
}

// TestRectanglesDrawThePixelsOfTheirPaths draws scenes with rectangles (clip layers of each
// behavior, clips in a picture, fills) and the same scenes with each rectangle given as a
// path, which is rasterized into a mask: the pixels must be the same, byte for byte.
func TestRectanglesDrawThePixelsOfTheirPaths(t *testing.T) {
	const w, h = 96, 80
	gradient := ui.NewPaint()
	gradient.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 90, Dy: 70},
		[]ui.Color{ui.NewColor(0xFFE04030), ui.NewColor(0x8030A0E0)}))
	hard := paint(0xC020A040)
	hard.SetIsAntiAlias(false)
	multiply := paint(0xFF80C0FF)
	multiply.SetBlendMode(ui.BlendModeMultiply)
	paints := []*ui.Paint{paint(0xFF204080), paint(0x80F0A020), gradient, hard, multiply}
	behaviors := []ui.Clip{ui.ClipHardEdge, ui.ClipAntiAlias, ui.ClipAntiAliasWithSaveLayer, ui.ClipNone}

	// build makes the scene of one case, with the rectangles as rectangles or as paths.
	build := func(seed int64, whole, asPaths bool) *ui.Scene {
		rng := rand.New(rand.NewSource(seed))
		b := ui.NewSceneBuilder()
		back := ui.NewPictureRecorder()
		c := ui.NewCanvas(back)
		c.DrawColor(ui.NewColor(0xFFF4F0E8), ui.BlendModeSrcOver)
		c.DrawCircle(ui.Offset{Dx: 40, Dy: 36}, 30, paint(0xFF30C070))
		b.AddPicture(ui.Offset{}, back.EndRecording())
		if rng.Intn(2) == 0 {
			// A clip with a mask around everything, which the rectangles narrow.
			b.PushClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(3.5, 2.5, 88, 74), 20, 16))
		}
		for layer := 0; layer < 2; layer++ {
			// The rectangles are in the coordinates of the layer's transform, so the
			// transform of each is the one they were made for.
			m, r := randomRectCase(rng, whole, true)
			b.PushTransform(m)
			behavior := behaviors[rng.Intn(len(behaviors))]
			if asPaths {
				b.PushClipPath(rectPath(r), ui.PushClipPathOpts{ClipBehavior: behavior})
			} else {
				b.PushClipRect(r, ui.PushClipRectOpts{ClipBehavior: behavior})
			}
			rec := ui.NewPictureRecorder()
			c := ui.NewCanvas(rec)
			for i := 0; i < 3; i++ {
				cm, cr := randomRectCase(rng, whole, true)
				aa := rng.Intn(2) == 0
				c.Save()
				c.Transform(cm)
				if asPaths {
					c.ClipPath(rectPath(cr), ui.ClipOpts{DoAntiAlias: &aa})
				} else {
					c.ClipRect(cr, ui.ClipRectOpts{DoAntiAlias: &aa})
				}
				for j := 0; j < 2; j++ {
					fm, fr := randomRectCase(rng, whole, j == 0)
					c.Save()
					c.Transform(fm)
					p := paints[rng.Intn(len(paints))]
					if asPaths {
						c.DrawPath(rectPath(fr), p)
					} else {
						c.DrawRect(fr, p)
					}
					c.Restore()
				}
				c.Restore()
			}
			b.AddPicture(ui.Offset{}, rec.EndRecording())
		}
		return b.Build()
	}
	for _, whole := range []bool{true, false} {
		t.Run(fmt.Sprintf("whole=%v", whole), func(t *testing.T) {
			drawn := 0
			for seed := int64(0); seed < 150; seed++ {
				got, err := Render(build(seed, whole, false), w, h)
				if err != nil {
					t.Fatalf("seed %d: Render: %v", seed, err)
				}
				want, err := Render(build(seed, whole, true), w, h)
				if err != nil {
					t.Fatalf("seed %d: Render of the paths: %v", seed, err)
				}
				if !bytes.Equal(got.Pix, want.Pix) {
					for i := 0; i < len(got.Pix); i += 4 {
						if !bytes.Equal(got.Pix[i:i+4], want.Pix[i:i+4]) {
							t.Fatalf("seed %d: pixel %d,%d = %v, the paths give %v", seed, i/4%w, i/4/w, got.Pix[i:i+4], want.Pix[i:i+4])
						}
					}
				}
				plain, err := Render(build(seed, whole, false), w, h)
				if err != nil {
					t.Fatalf("seed %d: Render: %v", seed, err)
				}
				background, err := Render(sceneOf(ui.Offset{}, func(c *ui.Canvas) {
					c.DrawColor(ui.NewColor(0xFFF4F0E8), ui.BlendModeSrcOver)
					c.DrawCircle(ui.Offset{Dx: 40, Dy: 36}, 30, paint(0xFF30C070))
				}), w, h)
				if err != nil {
					t.Fatalf("seed %d: Render of the background: %v", seed, err)
				}
				if !bytes.Equal(plain.Pix, background.Pix) {
					drawn++
				}
			}
			// The cases must not all be clipped away, or the test would compare nothing.
			if drawn < 50 {
				t.Errorf("only %d of 150 scenes drew a rectangle", drawn)
			}
		})
	}
}

// workbenchScene is the clips of a widget tree at the size of a screen: panes side by
// side, each a clip layer of whole pixels with a viewport's clip inside it and rows that
// clip and fill rectangles, as the parts of an editor do.
func workbenchScene(w, h int) *ui.Scene {
	b := ui.NewSceneBuilder()
	back := ui.NewPictureRecorder()
	ui.NewCanvas(back).DrawRect(ui.NewRectFromLTWH(0, 0, float64(w), float64(h)), paint(0xFF1E1E1E))
	b.AddPicture(ui.Offset{}, back.EndRecording())
	const panes = 4
	pw := float64(w / panes)
	for i := 0; i < panes; i++ {
		b.PushOffset(float64(i)*pw, 0)
		b.PushClipRect(ui.NewRectFromLTWH(0, 0, pw, float64(h)), ui.PushClipRectOpts{ClipBehavior: ui.ClipHardEdge})
		b.PushClipRect(ui.NewRectFromLTWH(0, 30, pw, float64(h-30)))
		rec := ui.NewPictureRecorder()
		c := ui.NewCanvas(rec)
		c.DrawRect(ui.NewRectFromLTWH(0, 0, pw, float64(h)), paint(0xFF252526))
		for y := 30.0; y < float64(h); y += 20 {
			c.Save()
			c.ClipRect(ui.NewRectFromLTWH(8, y, pw-16, 20))
			c.DrawRect(ui.NewRectFromLTWH(8, y+2, pw-40, 16), paint(0xFF2D2D30))
			c.DrawRect(ui.NewRectFromLTWH(12, y+6, 120, 8), paint(0x80D4D4D4))
			c.Restore()
		}
		b.AddPicture(ui.Offset{}, rec.EndRecording())
		b.Pop()
		b.Pop()
		b.Pop()
	}
	return b.Build()
}

// allocatedBy returns the bytes that fn allocates.
func allocatedBy(fn func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// TestRectanglesOnWholePixelsMakeNoMasks holds the memory of a frame of rectangular clips
// and fills: before step 45.1 each clip and each fill made coverage masks the size of the
// clip around it (149 MB for this scene, and about as much for a frame of Emetos's workbench
// at 1920x1080); now the frame allocates a fraction of the surface it draws into.
func TestRectanglesOnWholePixelsMakeNoMasks(t *testing.T) {
	const w, h = 1920, 1080
	scene := workbenchScene(w, h)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var err error
	got := allocatedBy(func() { err = RenderInto(img, scene, img.Rect) })
	if err != nil {
		t.Fatalf("RenderInto: %v", err)
	}
	if limit := uint64(w * h); got > limit {
		t.Errorf("a frame allocated %d bytes, want at most %d (a quarter of the surface)", got, limit)
	}
	// Something was drawn in every pane.
	for _, p := range []image.Point{{20, 40}, {w/4 + 20, 500}, {w - 100, h - 30}} {
		if a := img.RGBAAt(p.X, p.Y).A; a != 255 {
			t.Errorf("alpha at %v = %d, want 255", p, a)
		}
	}
}

// BenchmarkRectangleClips is a frame of the clips and fills of a widget tree at 1920x1080
// (workbenchScene). Run it with
//
//	go test ./internal/gpu/soft -run xxx -bench RectangleClips -benchmem
func BenchmarkRectangleClips(b *testing.B) {
	const w, h = 1920, 1080
	scene := workbenchScene(w, h)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := RenderInto(img, scene, img.Rect); err != nil {
			b.Fatal(err)
		}
	}
}
