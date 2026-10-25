package gdiwindow

import (
	"image"
	"image/color"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// blitCall is one StretchDIBits as the fake saw it.
type blitCall struct {
	rect  image.Rectangle
	first [4]byte // the BGRA bytes of the top left pixel of the rectangle
	width int
}

// fakeAPI is a windowAPI that records the calls.
type fakeAPI struct {
	calls     []string
	blits     []blitCall
	blitLines int // what blit returns for each call; -1 for the height of the rectangle
	dcErr     error
	handlers  map[uintptr]func(uintptr)
	paintClip image.Rectangle
	paintErr  error
}

func newFakeAPI() *fakeAPI { return &fakeAPI{blitLines: -1, handlers: map[uintptr]func(uintptr){}} }

func (f *fakeAPI) getDC(window uintptr) (uintptr, error) {
	f.calls = append(f.calls, "getDC")
	return 7, f.dcErr
}
func (f *fakeAPI) releaseDC(window, dc uintptr) { f.calls = append(f.calls, "releaseDC") }
func (f *fakeAPI) blit(dc uintptr, bgra []byte, width int, r image.Rectangle) int {
	f.calls = append(f.calls, "blit")
	c := blitCall{rect: r, width: width}
	copy(c.first[:], bgra[(r.Min.Y*width+r.Min.X)*4:])
	f.blits = append(f.blits, c)
	if f.blitLines < 0 {
		return r.Dy()
	}
	return f.blitLines
}
func (f *fakeAPI) paint(window uintptr, draw func(dc uintptr, clip image.Rectangle)) error {
	f.calls = append(f.calls, "paint")
	if f.paintErr != nil {
		return f.paintErr
	}
	draw(9, f.paintClip)
	return nil
}
func (f *fakeAPI) setPaintHandler(window uintptr, fn func(uintptr)) {
	if fn == nil {
		delete(f.handlers, window)
		f.calls = append(f.calls, "unsetPaint")
		return
	}
	f.handlers[window] = fn
	f.calls = append(f.calls, "setPaint")
}

var win = gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Window: 0x42}

// solid is an image of one color.
func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return img
}

func TestPresentWholeFrameFirst(t *testing.T) {
	api := newFakeAPI()
	p := newWindowPresenter(api)
	img := solid(8, 4, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	if err := p.Present(win, img, gpu.Damage{}); err != nil { // no damage, but the first frame is whole
		t.Fatal(err)
	}
	if len(api.blits) != 1 || api.blits[0].rect != image.Rect(0, 0, 8, 4) || api.blits[0].width != 8 {
		t.Fatalf("blits = %+v, want the whole 8x4 frame", api.blits)
	}
	if api.blits[0].first != [4]byte{30, 20, 10, 255} {
		t.Errorf("the pixel is %v, want BGRA 30 20 10 255", api.blits[0].first)
	}
	want := []string{"setPaint", "getDC", "blit", "releaseDC"}
	if !equalStrings(api.calls, want) {
		t.Errorf("calls = %v, want %v", api.calls, want)
	}
}

func TestPresentDamageRectangles(t *testing.T) {
	api := newFakeAPI()
	p := newWindowPresenter(api)
	first := solid(100, 80, color.RGBA{R: 255, A: 255})
	if err := p.Present(win, first, gpu.Damage{Full: true}); err != nil {
		t.Fatal(err)
	}
	api.calls, api.blits = nil, nil

	second := solid(100, 80, color.RGBA{R: 255, A: 255})
	for y := 10; y < 30; y++ { // a blue area inside the first damage rectangle
		for x := 20; x < 60; x++ {
			second.SetRGBA(x, y, color.RGBA{B: 255, A: 255})
		}
	}
	rects := []image.Rectangle{image.Rect(20, 10, 60, 30), image.Rect(70, 50, 100, 80)}
	if err := p.Present(win, second, gpu.Damage{Rects: rects}); err != nil {
		t.Fatal(err)
	}
	if len(api.blits) != 2 || api.blits[0].rect != rects[0] || api.blits[1].rect != rects[1] {
		t.Fatalf("blits = %+v, want one for each damage rectangle", api.blits)
	}
	if api.blits[0].first != [4]byte{255, 0, 0, 255} {
		t.Errorf("the first rectangle starts with %v, want blue as BGRA", api.blits[0].first)
	}
	if !equalStrings(api.calls, []string{"getDC", "blit", "blit", "releaseDC"}) {
		t.Errorf("calls = %v: the device context is got once for the frame", api.calls)
	}
	// The kept frame has the new pixels where the damage was and the old ones elsewhere.
	k := p.frames[win.Window]
	if px := k.bgra[(15*100+30)*4:][:4]; px[0] != 255 || px[2] != 0 {
		t.Errorf("the kept frame at 30,15 = %v, want blue", px)
	}
}

func TestPresentNothingChanged(t *testing.T) {
	api := newFakeAPI()
	p := newWindowPresenter(api)
	img := solid(10, 10, color.RGBA{A: 255})
	if err := p.Present(win, img, gpu.Damage{Full: true}); err != nil {
		t.Fatal(err)
	}
	api.calls = nil
	if err := p.Present(win, img, gpu.Damage{}); err != nil {
		t.Fatal(err)
	}
	if len(api.calls) != 0 {
		t.Errorf("a frame with no damage touched the window: %v", api.calls)
	}
}

func TestPresentNewSizeShowsAllOfIt(t *testing.T) {
	api := newFakeAPI()
	p := newWindowPresenter(api)
	if err := p.Present(win, solid(10, 10, color.RGBA{A: 255}), gpu.Damage{Full: true}); err != nil {
		t.Fatal(err)
	}
	api.calls, api.blits = nil, nil
	if err := p.Present(win, solid(20, 12, color.RGBA{A: 255}), gpu.Damage{Rects: []image.Rectangle{image.Rect(0, 0, 1, 1)}}); err != nil {
		t.Fatal(err)
	}
	if len(api.blits) != 1 || api.blits[0].rect != image.Rect(0, 0, 20, 12) {
		t.Errorf("blits = %+v, want the whole new size", api.blits)
	}
	for _, c := range api.calls {
		if c == "setPaint" {
			t.Error("a new size set the paint handler again")
		}
	}
}

func TestPresentClampsDamage(t *testing.T) {
	api := newFakeAPI()
	p := newWindowPresenter(api)
	if err := p.Present(win, solid(10, 10, color.RGBA{A: 255}), gpu.Damage{Full: true}); err != nil {
		t.Fatal(err)
	}
	api.blits = nil
	damage := gpu.Damage{Rects: []image.Rectangle{image.Rect(5, 5, 50, 50), image.Rect(20, 20, 30, 30)}}
	if err := p.Present(win, solid(10, 10, color.RGBA{A: 255}), damage); err != nil {
		t.Fatal(err)
	}
	if len(api.blits) != 1 || api.blits[0].rect != image.Rect(5, 5, 10, 10) {
		t.Errorf("blits = %+v, want the part inside the frame, and none for a rectangle outside", api.blits)
	}
}

func TestPresentFailures(t *testing.T) {
	t.Run("GDI copies nothing", func(t *testing.T) {
		api := newFakeAPI()
		api.blitLines = 0
		p := newWindowPresenter(api)
		err := p.Present(win, solid(4, 4, color.RGBA{A: 255}), gpu.Damage{Full: true})
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
		}
		if !appErr.HasRoute("windowPresenter.Present.GDIPresentFailed") {
			t.Errorf("unexpected failure path: %s", appErr.Route())
		}
		if appErr.Metadata["window"] != win.Window || appErr.Metadata["width"] != 4 {
			t.Errorf("metadata = %v", appErr.Metadata)
		}
		if calls := api.calls; calls[len(calls)-1] != "releaseDC" {
			t.Errorf("the device context was not released after the failure: %v", calls)
		}
	})
	t.Run("no device context", func(t *testing.T) {
		api := newFakeAPI()
		api.dcErr = errors.New(testTagCallFailed, "GetDC failed")
		p := newWindowPresenter(api)
		err := p.Present(win, solid(4, 4, color.RGBA{A: 255}), gpu.Damage{Full: true})
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != testTagCallFailed {
			t.Errorf("the error of the device context was changed: %v", err)
		}
	})
}

func TestExposureRepaintsTheKeptFrame(t *testing.T) {
	api := newFakeAPI()
	p := newWindowPresenter(api)
	img := solid(100, 80, color.RGBA{G: 200, A: 255})
	if err := p.Present(win, img, gpu.Damage{Full: true}); err != nil {
		t.Fatal(err)
	}
	api.calls, api.blits = nil, nil
	repaint := api.handlers[win.Window]
	if repaint == nil {
		t.Fatal("the first frame did not set the paint handler")
	}
	api.paintClip = image.Rect(40, 30, 140, 90) // partly outside the frame
	repaint(win.Window)
	if !equalStrings(api.calls, []string{"paint", "blit"}) || len(api.blits) != 1 || api.blits[0].rect != image.Rect(40, 30, 100, 80) {
		t.Errorf("calls %v, blits %+v: want the invalid part inside the frame, painted from the kept frame", api.calls, api.blits)
	}
	if api.blits[0].first != [4]byte{0, 200, 0, 255} {
		t.Errorf("the repainted pixel = %v", api.blits[0].first)
	}
	// An exposure with an empty invalid rectangle, or of a window with no kept frame, still ends the paint.
	api.calls, api.blits = nil, nil
	api.paintClip = image.Rectangle{}
	repaint(win.Window)
	p.Release(win)
	api.calls = nil
	repaint(win.Window)
	if len(api.blits) != 0 || !equalStrings(api.calls, []string{"paint"}) {
		t.Errorf("calls %v, blits %+v", api.calls, api.blits)
	}
}

func TestReleaseForgetsTheWindow(t *testing.T) {
	api := newFakeAPI()
	p := newWindowPresenter(api)
	p.Release(win) // never presented: nothing happens
	if len(api.calls) != 0 {
		t.Errorf("calls = %v", api.calls)
	}
	if err := p.Present(win, solid(4, 4, color.RGBA{A: 255}), gpu.Damage{Full: true}); err != nil {
		t.Fatal(err)
	}
	p.Release(win)
	if _, ok := api.handlers[win.Window]; ok || p.frames[win.Window] != nil {
		t.Error("the window is not forgotten")
	}
	api.calls = nil
	if err := p.Present(win, solid(4, 4, color.RGBA{A: 255}), gpu.Damage{}); err != nil {
		t.Fatal(err)
	}
	if api.calls[0] != "setPaint" {
		t.Errorf("a window presented again does not get its paint handler: %v", api.calls)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// testTagCallFailed is the tag of the error of a Win32 call, as the fake device context returns it.
const testTagCallFailed = "Win32CallFailed"
