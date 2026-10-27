//go:build linux && !android

package wayland

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/input"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// tablet is the zwp_tablet_seat_v2 of the seat: the drawing tablets and their
// tools. Each tool becomes a device of stylus (or eraser, finger or mouse) pointer
// data (SPEC §8.11).
type tablet struct {
	p     *Platform
	seat  *wl.ZwpTabletSeatV2
	b     *input.StylusBatcher
	tools map[*wl.ZwpTabletToolV2]*tabletTool
	next  int
}

// tabletTool is one tool, such as a pen or an eraser.
type tabletTool struct {
	id   int
	obj  *wl.ZwpTabletToolV2
	kind ui.PointerDeviceKind
	// shape is the cursor shape device of the tool, nil without wp_cursor_shape_v1.
	shape *wl.WpCursorShapeDeviceV1
	// focus is the window the tool is near, and serial the serial of its proximity
	// event, which the cursor needs.
	focus  *window
	serial uint32
}

// newTablet follows the tablets of the seat. It returns nil when the compositor has
// no zwp_tablet_manager_v2.
func newTablet(p *Platform) *tablet {
	mgr := p.sess.tabletManager
	if mgr == nil {
		return nil
	}
	t := &tablet{p: p, b: input.NewStylusBatcher(p.now, p.pointerIDs), tools: map[*wl.ZwpTabletToolV2]*tabletTool{}}
	t.seat = mgr.GetTabletSeat(p.sess.seat)
	t.seat.SetEvents(&wl.ZwpTabletSeatV2Events{OnToolAdded: t.toolAdded})
	return t
}

// kindOf maps the type of a tool to the kind of pointer: a pen, a brush, a pencil
// or an airbrush are a stylus, an eraser the inverted stylus.
func kindOf(t wl.ZwpTabletToolV2Type) ui.PointerDeviceKind {
	switch t {
	case wl.ZwpTabletToolV2TypeEraser:
		return ui.PointerDeviceKindInvertedStylus
	case wl.ZwpTabletToolV2TypeFinger:
		return ui.PointerDeviceKindTouch
	case wl.ZwpTabletToolV2TypeMouse, wl.ZwpTabletToolV2TypeLens:
		return ui.PointerDeviceKindMouse
	}
	return ui.PointerDeviceKindStylus
}

// toolAdded follows a tool that the tablet seat announced. The description (type,
// serial and capabilities) comes before done; the tool is known to the batcher then.
func (t *tablet) toolAdded(obj *wl.ZwpTabletToolV2) {
	tool := &tabletTool{id: t.next, obj: obj, kind: ui.PointerDeviceKindStylus}
	t.next++
	t.tools[obj] = tool
	if m := t.p.sess.cursorShape; m != nil {
		tool.shape = m.GetTabletToolV2(obj)
	}
	id := tool.id
	obj.SetEvents(&wl.ZwpTabletToolV2Events{
		OnType: func(tt wl.ZwpTabletToolV2Type) { tool.kind = kindOf(tt) },
		OnDone: func() { t.b.AddTool(id, tool.kind) },
		OnRemoved: func() {
			t.b.RemoveTool(id)
			t.flush()
			if tool.shape != nil {
				tool.shape.Destroy()
			}
			obj.Destroy()
			delete(t.tools, obj)
		},
		OnProximityIn: func(serial uint32, _ *wl.ZwpTabletV2, surface *wl.WlSurface) {
			w := t.p.windowOf(surface)
			if w == nil {
				return
			}
			tool.focus, tool.serial = w, serial
			t.p.inputSerial = serial
			t.b.Scale = w.scaleFactor()
			t.b.ProximityIn(id, w.id)
			t.applyCursor(tool)
		},
		OnProximityOut: func() { tool.focus = nil; t.b.ProximityOut(id) },
		OnDown:         func(serial uint32) { t.p.inputSerial = serial; t.p.pointerSerial = serial; t.b.Down(id) },
		OnUp:           func() { t.b.Up(id) },
		OnMotion:       func(x, y float64) { t.b.Motion(id, x, y) },
		OnPressure:     func(v uint32) { t.b.Pressure(id, v) },
		OnDistance:     func(v uint32) { t.b.Distance(id, v) },
		OnTilt:         func(x, y float64) { t.b.Tilt(id, x, y) },
		OnRotation:     func(degrees float64) { t.b.Rotation(id, degrees) },
		OnButton: func(serial uint32, button uint32, state wl.ZwpTabletToolV2ButtonState) {
			t.p.inputSerial = serial
			t.b.Button(id, button, state == wl.ZwpTabletToolV2ButtonStatePressed)
		},
		OnFrame: func(uint32) { t.b.Frame(id); t.flush() },
	})
}

// applyCursor sets the cursor of the tool near a window: the window's, by shape.
// Without wp_cursor_shape_v1 the compositor keeps the cursor it has for the tool.
func (t *tablet) applyCursor(tool *tabletTool) {
	w := tool.focus
	if w == nil {
		return
	}
	switch {
	case w.cursor == platform.MouseCursorNone:
		tool.obj.SetCursor(tool.serial, nil, 0, 0)
	case tool.shape != nil:
		shape, ok := cursorShapes[w.cursor]
		if !ok {
			shape = wl.WpCursorShapeDeviceV1ShapeDefault
		}
		tool.shape.SetShape(tool.serial, shape)
	}
}

// cursorChanged sets the cursor again for the tools near the window.
func (t *tablet) cursorChanged(w *window) {
	for _, tool := range t.tools {
		if tool.focus == w {
			t.applyCursor(tool)
		}
	}
}

// flush hands the collected data to the handler.
func (t *tablet) flush() {
	if batch := t.b.Take(); len(batch) > 0 {
		t.p.deliverPointers(batch)
	}
}

// windowClosed cancels the tools that are near a window that is destroyed.
func (t *tablet) windowClosed(w *window) {
	for _, tool := range t.tools {
		if tool.focus == w {
			tool.focus = nil
		}
	}
	t.b.CancelView(w.id)
	t.flush()
}

// close drops the tablet seat. Tools that are near a window are removed.
func (t *tablet) close() {
	for obj, tool := range t.tools {
		t.b.RemoveTool(tool.id)
		if tool.shape != nil {
			tool.shape.Destroy()
		}
		obj.Destroy()
	}
	t.flush()
	t.tools = nil
	t.seat.Destroy()
}
