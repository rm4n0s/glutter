package offscreen_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/gpu/gputest/editorscene"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/internal/proof"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TestLeaksOffscreen checks the memory of the off-screen drawing (step 29.21, SPEC §15 "Memory
// over time"): frames of a surface and of a view, and views that are made and closed.
func TestLeaksOffscreen(t *testing.T) {
	size := ui.Size{Width: counterW, Height: counterH}
	tests := []struct {
		name string
		n    int
		step func(t *testing.T) func()
	}{
		{"surface, caret frames of the editor", 300, func(t *testing.T) func() {
			on, off := editorscene.CaretScenes(t, 200)
			d := offscreen.NewDetacher()
			scenes := [2]*ui.Scene{d.Detach(on), d.Detach(off)}
			s := offscreen.NewSurface(editorscene.Width, editorscene.Height)
			i := 0
			return func() {
				if _, err := s.Draw(scenes[i%2]); err != nil {
					t.Fatal(err)
				}
				i++
			}
		}},
		{"surface, a new picture under a spinner for each frame", 300, func(t *testing.T) func() {
			// The surface keeps the mipmap chains of the pictures it scales down (step 49.1): the
			// chain of a picture whose handle is disposed goes at the end of the next frame.
			s := offscreen.NewSurface(320, 240)
			var prev *ui.Image
			i := 0
			return func() {
				img := picture(256, 192, uint8(i))
				if _, err := s.Draw(pictureUnderSpinner(img, i)); err != nil {
					t.Fatal(err)
				}
				if prev != nil {
					prev.Dispose()
				}
				prev = img
				i++
			}
		}},
		{"surface, a spinner over the same picture", 300, func(t *testing.T) func() {
			s := offscreen.NewSurface(320, 240)
			img := picture(256, 192, 7)
			i := 0
			return func() {
				if _, err := s.Draw(pictureUnderSpinner(img, i)); err != nil {
					t.Fatal(err)
				}
				i++
			}
		}},
		{"detacher, a new scene for each frame", 300, func(t *testing.T) func() {
			on, off := editorscene.CaretScenes(t, 200)
			d := offscreen.NewDetacher()
			scenes := [2]*ui.Scene{on, off}
			i := 0
			return func() { _ = d.Detach(scenes[i%2]); i++ }
		}},
		{"glyph sheets and font metrics", 60, func(t *testing.T) func() {
			// 256 clusters of two sizes: the caches of the engine are bounded, and the sheet is the caller's.
			engine := proof.BundledTextEngine()
			d := ui.NewPlatformDispatcher(nil)
			d.SetTextEngine(engine)
			clusters := latinClusters(256)
			i := 0
			return func() {
				style := robotoStyle(float64(12 + 2*(i%2)))
				i++
				sheet, err := offscreen.NewGlyphSheet(engine, style, 1+float64(i%2)/2, clusters)
				if err != nil {
					t.Fatal(err)
				}
				sheet.Image.Dispose()
				if _, err := d.FontMetrics(style); err != nil {
					t.Fatal(err)
				}
			}
		}},
		{"a view that gets taps", 100, func(t *testing.T) func() {
			probe := &counterProbe{}
			h := newHarness(t, newCounter(probe), size)
			h.step(0)
			taps := 0
			return func() {
				tap(h.view, fabX, fabY)
				h.step(pace)
				if taps++; taps%10 == 0 {
					// The text of the counter goes back to 0: the shaped runs and analyses of the
					// text are kept in caches of a bound, which a text that never repeats would fill.
					probe.s.SetState(func() { probe.s.count = 0 })
					h.step(pace)
				}
				h.mu.Lock()
				h.frames = h.frames[:0] // the harness keeps a copy of every frame
				h.mu.Unlock()
			}
		}},
		{"views made and closed", 40, func(t *testing.T) func() {
			parent := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 320, Height: 200}, DiscardScenes: true})
			parent.PumpWidget(widgets.NewSizedBoxShrink(nil))
			return func() {
				v, err := offscreen.NewViewWithClock(parent.Binding, size, newLightCounter(nil), func(offscreen.Frame) {}, parent.Clock)
				if err != nil {
					t.Fatal(err)
				}
				parent.Pump(0)
				v.WaitIdle()
				parent.Pump(40 * time.Millisecond)
				v.WaitIdle()
				v.Close()
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaktest.Run(t, "offscreen/"+tt.name, func() func() { return tt.step(t) }, leaktest.Opts{N: tt.n})
		})
	}
}

// picture is an image of w x h pixels of blocks of 16 pixels in two colors that seed changes.
func picture(w, h int, seed uint8) *ui.Image {
	pix := make([]byte, 4*w*h)
	for y := range h {
		for x := range w {
			c := [4]uint8{200, seed, 40, 255}
			if (x/16+y/16)%2 == 1 {
				c = [4]uint8{20, 90, seed, 255}
			}
			copy(pix[4*(y*w+x):], c[:])
		}
	}
	return ui.NewImageFromRGBA(pix, w, h)
}

// pictureUnderSpinner draws the image at half its size with mipmaps and a square turned by turn
// over it.
func pictureUnderSpinner(img *ui.Image, turn int) *ui.Scene {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	w, h := float64(img.Width()), float64(img.Height())
	p := ui.NewPaint()
	p.SetFilterQuality(ui.FilterQualityMedium)
	c.DrawImageRect(img, ui.NewRectFromLTWH(0, 0, w, h), ui.NewRectFromLTWH(0, 0, w/2, h/2), p)
	c.Translate(w/4, h/4)
	c.Rotate(float64(turn) * 0.4)
	sp := ui.NewPaint()
	sp.SetColor(ui.NewColor(0xFF2060FF))
	c.DrawRect(ui.NewRectFromLTWH(-10, -10, 20, 20), sp)
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	return sb.Build()
}
