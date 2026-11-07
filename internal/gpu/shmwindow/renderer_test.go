package shmwindow

import (
	"bytes"
	"image"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

var wayland = gpu.SurfaceHandle{Kind: gpu.SurfaceKindWayland, Display: 1, Surface: 7}

const bgColor = 0xFF102030

// fakeTarget is a gpu.PixelTarget over a few buffers, with a compositor that shows the whole of
// the buffer it was given (some compositors read all of it): a buffer that is missing a rectangle
// shows wrong pixels in it. The tests also check that the damage that was reported covers what
// differs from the frame before.
type fakeTarget struct {
	buffers   [][]byte // the memory of each buffer
	shownAt   []int    // the presentation each buffer holds, 0 for none
	next      int
	width     int
	height    int
	cur       int // the buffer acquired, -1 for none
	presented int
	// shown is the compositor's copy of the window, in BGRX, and before the frame that was just
	// presented.
	shown, before []byte
	damages       [][]image.Rectangle
	acquire       int
	fail          error
}

func newFakeTarget(buffers int) *fakeTarget {
	return &fakeTarget{buffers: make([][]byte, buffers), shownAt: make([]int, buffers), cur: -1}
}

func (f *fakeTarget) AcquirePixels(width, height int) (gpu.PixelBuffer, error) {
	f.acquire++
	if f.fail != nil {
		return gpu.PixelBuffer{}, f.fail
	}
	if width != f.width || height != f.height {
		f.width, f.height = width, height
		for i := range f.buffers {
			f.buffers[i], f.shownAt[i] = nil, 0
		}
		f.shown = make([]byte, 4*width*height)
	}
	i := f.next % len(f.buffers)
	if f.buffers[i] == nil {
		f.buffers[i] = make([]byte, 4*width*height)
		for j := range f.buffers[i] {
			f.buffers[i][j] = 0xEE // not a frame of the scene: a pixel that is not written shows
		}
	}
	f.cur = i
	age := 0
	if f.shownAt[i] != 0 {
		age = f.presented + 1 - f.shownAt[i]
	}
	return gpu.PixelBuffer{Pix: f.buffers[i], Stride: 4 * width, Width: width, Height: height, Format: gpu.PixelFormatBGRX8888, Age: age}, nil
}

func (f *fakeTarget) PresentPixels(damage []image.Rectangle) error {
	f.damages = append(f.damages, damage)
	if f.cur < 0 {
		return nil
	}
	i := f.cur
	f.cur = -1
	f.presented++
	f.shownAt[i] = f.presented
	f.next++
	f.before = append(f.before[:0], f.shown...)
	copy(f.shown, f.buffers[i])
	return nil
}

// shownImage is what the compositor shows, as RGBA.
func (f *fakeTarget) shownImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, f.width, f.height))
	for i := 0; i < len(f.shown); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = f.shown[i+2], f.shown[i+1], f.shown[i], 255
	}
	return img
}

func newRenderer(t *testing.T, target gpu.PixelTarget) (*Renderer, gpu.Surface) {
	t.Helper()
	r := NewRenderer(func(gpu.SurfaceHandle) (gpu.PixelTarget, bool) { return target, true }, Options{Background: ui.NewColor(bgColor)})
	s, err := r.Attach(wayland)
	if err != nil {
		t.Fatal(err)
	}
	return r, s
}

// reference draws the scene over the background in one piece.
func reference(t *testing.T, scene *ui.Scene, w, h int) *image.RGBA {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fillRect(img, img.Rect, [3]uint8{0x10, 0x20, 0x30})
	if err := soft.RenderInto(img, scene, img.Rect); err != nil {
		t.Fatal(err)
	}
	return img
}

// square is a picture of a red square of 20 pixels, which shows on the background picture.
func square(x, y float64) *ui.Picture {
	return gputest.PictureOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(x, y, 20, 20), gputest.FillPaint(0xFFE53935)) })
}

// movingScenes are frames of a window of 192x128 pixels: a background picture and a square that
// moves to a place of its own in each frame, in a picture of its own, so that only the tiles of the
// square change and a buffer that misses a frame shows an old square.
func movingScenes() []*ui.Scene {
	back := gputest.DamagePicture(ui.NewRectFromLTWH(0, 0, 192, 128))
	var scenes []*ui.Scene
	for i := 0; i < 6; i++ {
		x := float64(i % 3 * 66)  // a tile of the damage tracker is 64 pixels wide: each place is another tile
		y := 10 + float64(i/3)*60 // two rows of tiles
		scenes = append(scenes, gputest.DamageScene(back, square(x+2, y)))
	}
	return scenes
}

// coversChanges checks that the damage the target was given last covers every pixel that differs
// from the frame before.
func (f *fakeTarget) coversChanges(t *testing.T, label string) {
	t.Helper()
	if len(f.damages) == 0 || len(f.before) != len(f.shown) {
		return
	}
	last := f.damages[len(f.damages)-1]
	for i := 0; i < len(f.shown); i += 4 {
		if bytes.Equal(f.shown[i:i+4], f.before[i:i+4]) {
			continue
		}
		p := image.Pt((i/4)%f.width, (i/4)/f.width)
		covered := false
		for _, r := range last {
			covered = covered || p.In(r)
		}
		if !covered {
			t.Fatalf("%s: the pixel %v changed and the damage %v does not cover it", label, p, last)
		}
	}
}

func samePixels(t *testing.T, label string, got, want *image.RGBA) {
	t.Helper()
	if !bytes.Equal(got.Pix, want.Pix) {
		for i := 0; i < len(got.Pix); i += 4 {
			if !bytes.Equal(got.Pix[i:i+4], want.Pix[i:i+4]) {
				p := i / 4
				t.Fatalf("%s: pixel %d,%d is %v, want %v", label, p%got.Rect.Dx(), p/got.Rect.Dx(), got.Pix[i:i+4], want.Pix[i:i+4])
			}
		}
	}
}

func TestEveryFrameShowsWhatAWholeRedrawShows(t *testing.T) {
	for _, buffers := range []int{1, 2, 3, 4} {
		t.Run(string(rune('0'+buffers))+" buffers", func(t *testing.T) {
			target := newFakeTarget(buffers)
			r, s := newRenderer(t, target)
			for i, scene := range movingScenes() {
				if err := r.Render(s, scene, ui.Size{Width: 192, Height: 128}, 1); err != nil {
					t.Fatalf("frame %d: %v", i, err)
				}
				samePixels(t, "frame "+string(rune('0'+i)), target.shownImage(), reference(t, scene, 192, 128))
				target.coversChanges(t, "frame "+string(rune('0'+i)))
			}
		})
	}
}

func TestOnlyTheChangedTilesAreReportedAndDrawn(t *testing.T) {
	target := newFakeTarget(2)
	r, s := newRenderer(t, target)
	scenes := movingScenes()
	for i := 0; i < 3; i++ {
		if err := r.Render(s, scenes[i], ui.Size{Width: 192, Height: 128}, 1); err != nil {
			t.Fatal(err)
		}
	}
	if got := target.damages[0]; len(got) != 1 || got[0] != image.Rect(0, 0, 192, 128) {
		t.Errorf("the first frame reported %v, want the whole window", got)
	}
	for i := 1; i < 3; i++ {
		got := target.damages[i]
		if len(got) == 0 {
			t.Fatalf("frame %d reported nothing, but the square moved", i)
		}
		var area int
		for _, rc := range got {
			area += rc.Dx() * rc.Dy()
		}
		if area >= 192*128/2 {
			t.Errorf("frame %d reported %v, %d pixels of %d: not only the square's tiles", i, got, area, 192*128)
		}
	}
}

func TestAFrameThatChangedNothingAcquiresNoBuffer(t *testing.T) {
	target := newFakeTarget(2)
	r, s := newRenderer(t, target)
	back := gputest.DamagePicture(ui.NewRectFromLTWH(0, 0, 192, 128))
	sq := square(10, 10)
	if err := r.Render(s, gputest.DamageScene(back, sq), ui.Size{Width: 192, Height: 128}, 1); err != nil {
		t.Fatal(err)
	}
	acquired := target.acquire
	// Another scene of the same pictures: the framework built a scene and repainted nothing.
	if err := r.Render(s, gputest.DamageScene(back, sq), ui.Size{Width: 192, Height: 128}, 1); err != nil {
		t.Fatal(err)
	}
	if target.acquire != acquired {
		t.Errorf("a frame with no damage acquired a buffer")
	}
	if last := target.damages[len(target.damages)-1]; len(last) != 0 {
		t.Errorf("a frame with no damage presented with the damage %v, want none", last)
	}
	if len(target.damages) != 2 {
		t.Errorf("%d presentations, want 2: the frame without damage still has to present, for the frame pacing", len(target.damages))
	}
}

func TestInvalidateAndResizeDrawWhole(t *testing.T) {
	target := newFakeTarget(2)
	r, s := newRenderer(t, target)
	scenes := movingScenes()
	size := ui.Size{Width: 192, Height: 128}
	for i := 0; i < 2; i++ {
		if err := r.Render(s, scenes[i], size, 1); err != nil {
			t.Fatal(err)
		}
	}
	r.Invalidate(s) // the window was hidden and shown: its content is lost
	if err := r.Render(s, scenes[2], size, 1); err != nil {
		t.Fatal(err)
	}
	if got := target.damages[len(target.damages)-1]; len(got) != 1 || got[0] != image.Rect(0, 0, 192, 128) {
		t.Errorf("the frame after Invalidate reported %v, want the whole window", got)
	}
	samePixels(t, "after Invalidate", target.shownImage(), reference(t, scenes[2], 192, 128))

	// The window grows: every buffer is new, and the frame is whole at the new size.
	big := gputest.DamageScene(gputest.DamagePicture(ui.NewRectFromLTWH(0, 0, 256, 160)), square(5, 5))
	if err := r.Render(s, big, ui.Size{Width: 256, Height: 160}, 1); err != nil {
		t.Fatal(err)
	}
	if got := target.damages[len(target.damages)-1]; len(got) != 1 || got[0] != image.Rect(0, 0, 256, 160) {
		t.Errorf("the first frame at the new size reported %v, want the whole window", got)
	}
	samePixels(t, "after the resize", target.shownImage(), reference(t, big, 256, 160))
}

func TestAnErrorFromTheTargetMakesTheNextFrameWhole(t *testing.T) {
	target := newFakeTarget(2)
	r, s := newRenderer(t, target)
	scenes := movingScenes()
	size := ui.Size{Width: 192, Height: 128}
	if err := r.Render(s, scenes[0], size, 1); err != nil {
		t.Fatal(err)
	}
	target.fail = errors.New(tagTestTargetFailed, "no memory")
	if err := r.Render(s, scenes[1], size, 1); err == nil {
		t.Fatal("Render succeeded though the target failed")
	}
	target.fail = nil
	if err := r.Render(s, scenes[2], size, 1); err != nil {
		t.Fatal(err)
	}
	if got := target.damages[len(target.damages)-1]; len(got) != 1 || got[0] != image.Rect(0, 0, 192, 128) {
		t.Errorf("the frame after an error reported %v, want the whole window", got)
	}
	samePixels(t, "after an error", target.shownImage(), reference(t, scenes[2], 192, 128))
}

func TestCaptureReturnsTheFrame(t *testing.T) {
	target := newFakeTarget(2)
	r, s := newRenderer(t, target)
	scene := movingScenes()[1]
	img, err := r.RenderCapture(s, scene, ui.Size{Width: 192, Height: 128}, 1, true)
	if err != nil || img == nil {
		t.Fatalf("RenderCapture = %v, %v", img, err)
	}
	samePixels(t, "capture", img, reference(t, scene, 192, 128))
	img.Pix[0] ^= 0xFF // the image is the caller's
	next, err := r.RenderCapture(s, movingScenes()[2], ui.Size{Width: 192, Height: 128}, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	samePixels(t, "next capture", next, reference(t, movingScenes()[2], 192, 128))
	if got, err := r.RenderCapture(s, scene, ui.Size{Width: 192, Height: 128}, 1, false); err != nil || got != nil {
		t.Errorf("RenderCapture without capture = %v, %v", got, err)
	}
}

func TestMissing(t *testing.T) {
	b := image.Rect(0, 0, 100, 100)
	r := func(x0, y0, x1, y1 int) image.Rectangle { return image.Rect(x0, y0, x1, y1) }
	tests := []struct {
		name    string
		rects   []image.Rectangle
		history [][]image.Rectangle
		age     int
		want    []image.Rectangle
	}{
		{"no age is the whole buffer", []image.Rectangle{r(0, 0, 10, 10)}, nil, 0, []image.Rectangle{b}},
		{"age 1 is this frame's damage", []image.Rectangle{r(0, 0, 10, 10)}, [][]image.Rectangle{{r(50, 50, 60, 60)}}, 1, []image.Rectangle{r(0, 0, 10, 10)}},
		{"age 2 adds the frame before", []image.Rectangle{r(0, 0, 10, 10)}, [][]image.Rectangle{{r(90, 90, 100, 100)}, {r(50, 50, 60, 60)}}, 2,
			[]image.Rectangle{r(0, 0, 10, 10), r(50, 50, 60, 60)}},
		{"overlapping damage is merged", []image.Rectangle{r(0, 0, 10, 10)}, [][]image.Rectangle{{r(5, 5, 20, 20)}}, 2, []image.Rectangle{r(0, 0, 20, 20)}},
		{"older than the history is the whole buffer", []image.Rectangle{r(0, 0, 10, 10)}, [][]image.Rectangle{{r(50, 50, 60, 60)}}, 3, []image.Rectangle{b}},
		{"damage is cut to the buffer", []image.Rectangle{r(90, 90, 120, 120)}, nil, 1, []image.Rectangle{r(90, 90, 100, 100)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := missing(tt.rects, tt.history, tt.age, b)
			if len(got) != len(tt.want) {
				t.Fatalf("missing = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("missing = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestCopyBGRXSwapsRedAndBlue(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	copy(src.Pix, []byte{1, 2, 3, 255, 4, 5, 6, 255, 7, 8, 9, 255, 10, 11, 12, 255})
	dst := gpu.PixelBuffer{Pix: make([]byte, 16), Stride: 8, Width: 2, Height: 2}
	copyBGRX(dst, src, image.Rect(1, 0, 2, 2))
	want := []byte{0, 0, 0, 0, 6, 5, 4, 255, 0, 0, 0, 0, 12, 11, 10, 255}
	if !bytes.Equal(dst.Pix, want) {
		t.Errorf("pixels = %v, want %v (the rectangle only, blue first, alpha 255)", dst.Pix, want)
	}
}

func assertRoute(t *testing.T, err error, tag, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Errorf("tag %q, route %s; want tag %q and route %s", appErr.Tag, appErr.Route(), tag, route)
	}
}

func TestErrorRoutes(t *testing.T) {
	target := newFakeTarget(2)
	targets := func(h gpu.SurfaceHandle) (gpu.PixelTarget, bool) { return target, h.Surface == 7 }
	size := ui.Size{Width: 8, Height: 8}

	t.Run("a handle of another kind", func(t *testing.T) {
		_, err := NewRenderer(targets).Attach(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Window: 1})
		assertRoute(t, err, "SurfaceKindUnsupported", "Renderer.Attach.SurfaceKindUnsupported")
	})
	t.Run("a window with no target", func(t *testing.T) {
		_, err := NewRenderer(targets).Attach(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWayland, Surface: 9})
		assertRoute(t, err, "PixelTargetMissing", "Renderer.Attach.PixelTargetMissing")
	})
	t.Run("a closed renderer", func(t *testing.T) {
		r := NewRenderer(targets)
		_ = r.Close()
		_, err := r.Attach(wayland)
		assertRoute(t, err, "RendererClosed", "Renderer.Attach.RendererClosed")
	})
	t.Run("a surface of another renderer", func(t *testing.T) {
		r, other := NewRenderer(targets), NewRenderer(targets)
		s, _ := other.Attach(wayland)
		assertRoute(t, r.Render(s, nil, size, 1), "SurfaceInvalid", "Renderer.Render->Renderer.RenderCapture.SurfaceInvalid")
	})
	t.Run("a destroyed surface", func(t *testing.T) {
		r := NewRenderer(targets)
		s, _ := r.Attach(wayland)
		s.Destroy()
		s.Destroy()
		assertRoute(t, r.Render(s, nil, size, 1), "SurfaceInvalid", "Renderer.Render->Renderer.RenderCapture.SurfaceInvalid")
	})
	t.Run("a size that is not valid", func(t *testing.T) {
		r := NewRenderer(targets)
		s, _ := r.Attach(wayland)
		assertRoute(t, r.Render(s, nil, ui.Size{Width: 0, Height: 5}, 1), "SizeInvalid", "Renderer.Render->Renderer.RenderCapture.SizeInvalid")
		assertRoute(t, r.Render(s, nil, ui.Size{Width: 40000, Height: 5}, 1), "SizeInvalid", "Renderer.Render->Renderer.RenderCapture.SizeInvalid")
	})
	t.Run("a buffer that is not the size asked for", func(t *testing.T) {
		r := NewRenderer(targets)
		s, _ := r.Attach(wayland)
		bad := &wrongSizeTarget{}
		s.(*surface).target = bad
		assertRoute(t, r.Render(s, nil, size, 1), "PixelBufferInvalid", "Renderer.Render->Renderer.RenderCapture->surface.present.PixelBufferInvalid")
	})
	t.Run("two options values", func(t *testing.T) {
		defer func() {
			v := recover()
			err, ok := v.(error)
			if !ok {
				t.Fatalf("panic value %v is not an error", v)
			}
			assertRoute(t, err, "MultipleOptions", "NewRenderer.MultipleOptions")
		}()
		NewRenderer(targets, Options{}, Options{})
	})
}

// wrongSizeTarget gives a buffer of 1x1 whatever size is asked for.
type wrongSizeTarget struct{}

func (wrongSizeTarget) AcquirePixels(int, int) (gpu.PixelBuffer, error) {
	return gpu.PixelBuffer{Pix: make([]byte, 4), Stride: 4, Width: 1, Height: 1}, nil
}
func (wrongSizeTarget) PresentPixels([]image.Rectangle) error { return nil }

// tagTestTargetFailed is the tag of the error that the fake target makes.
const tagTestTargetFailed = "TestTargetFailed"
