package engine

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/ui"
)

// renewable is a window that can replace its native surface.
type renewable struct {
	platform.Window
	renewed, frames int
}

func (r *renewable) RenewSurface() { r.renewed++ }
func (r *renewable) RequestFrame() { r.frames++ }

// postsInline is a connection whose Post runs the function at once.
type postsInline struct{ *conn }

func (p postsInline) Post(fn func()) { fn() }

func TestHandlerRenewsAndRedrawsTheWindowOfASurface(t *testing.T) {
	p := fake.New()
	a, err := p.NewWindow(platform.WindowConfig{Title: "a"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.NewWindow(platform.WindowConfig{Title: "b"})
	if err != nil {
		t.Fatal(err)
	}
	wa, wb := &renewable{Window: a}, &renewable{Window: b}
	plain := &window{Window: b} // a window that cannot renew its surface
	h := &handler{conn: postsInline{&conn{Platform: p}}, windows: map[ui.ViewID]*window{
		a.ID(): {Window: wa}, b.ID(): {Window: wb},
	}}
	h.renewSurface(a.NativeSurface())
	if wa.renewed != 1 || wb.renewed != 0 {
		t.Errorf("renewed a %d times and b %d times, want 1 and 0", wa.renewed, wb.renewed)
	}
	h.redrawWindow(b.NativeSurface())
	if wb.frames != 1 || wa.frames != 0 {
		t.Errorf("frames asked of a %d times and of b %d times, want 0 and 1", wa.frames, wb.frames)
	}
	// A surface that no window has, and a window that has no way to renew, are ignored.
	h.renewSurface(gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 99})
	h.redrawWindow(gpu.SurfaceHandle{Kind: gpu.SurfaceKindFake, Window: 99})
	h.windows = map[ui.ViewID]*window{b.ID(): plain}
	h.renewSurface(b.NativeSurface())
	if wa.renewed != 1 || wb.renewed != 0 || wa.frames != 0 || wb.frames != 1 {
		t.Errorf("a surface that cannot be renewed changed something: %+v %+v", wa, wb)
	}
}
