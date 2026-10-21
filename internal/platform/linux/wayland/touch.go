//go:build linux

package wayland

import (
	"github.com/rm4n0s/glutter/internal/platform/linux/input"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
)

// touch is the wl_touch of the seat.
type touch struct {
	p   *Platform
	obj *wl.WlTouch
	b   *input.TouchBatcher
}

func newTouch(p *Platform) *touch {
	t := &touch{p: p, b: input.NewTouchBatcher(p.now)}
	t.obj = p.sess.seat.GetTouch()
	t.obj.SetEvents(&wl.WlTouchEvents{
		OnDown: func(_ uint32, _ uint32, surface *wl.WlSurface, id int32, x, y float64) {
			if w := p.windowOf(surface); w != nil {
				t.b.Scale = w.scaleFactor()
				t.b.Down(w.id, id, x, y)
			}
		},
		OnUp:          func(_ uint32, _ uint32, id int32) { t.b.Up(id) },
		OnMotion:      func(_ uint32, id int32, x, y float64) { t.b.Motion(id, x, y) },
		OnShape:       func(id int32, major, minor float64) { t.b.Shape(id, major, minor) },
		OnOrientation: func(id int32, o float64) { t.b.Orientation(id, o) },
		OnFrame:       t.flush,
		OnCancel: func() {
			t.b.Cancel()
			t.flush()
		},
	})
	return t
}

// flush hands the collected touch data to the handler, as one batch.
func (t *touch) flush() {
	if batch := t.b.Take(); len(batch) > 0 {
		t.p.deliverPointers(batch)
	}
}

// windowClosed cancels the fingers on a window that is destroyed.
func (t *touch) windowClosed(w *window) {
	t.b.CancelView(w.id)
	t.flush()
}

// close drops the touch device. Fingers that are down are cancelled.
func (t *touch) close() {
	t.b.Cancel()
	t.flush()
	if t.p.sess.seatVersion >= 3 {
		t.obj.Release() // wl_touch.release is since version 3
	}
	t.obj = nil
}
