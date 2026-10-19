package x11

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/internal/platform/linux/ime"
	"github.com/rm4n0s/glutter/ui"
)

// textInput is the input method of the X11 backend: IBus or Fcitx5 over D-Bus
// (SPEC §8.3, §8.6), through an ime.Bridge. It belongs to the UI thread.
//
// While a text field has the focus, every key goes to the input method first.
// A key it consumes is not sent to the app; the others are, in the order they
// came.
type textInput struct {
	p      *Platform
	bridge *ime.Bridge

	// window is the window whose text field has the text input, and want the
	// state it asked for.
	window *window
	want   ui.TextInputState
	// origin returns the position of a window's content in the root window;
	// tests replace it.
	origin func(w *window) (x, y int)
	// consumed are the keys (evdev codes) whose key-down the input method
	// consumed, so their key-up is not sent either.
	consumed map[uint32]bool
}

// newTextInput makes the text input of the platform. The connection to the
// input method is made when the first text field gets the focus; open is how,
// and tests replace it.
func newTextInput(p *Platform, poster ime.Poster, open ime.Opener) *textInput {
	t := &textInput{p: p, consumed: map[uint32]bool{}, origin: (*window).rootOrigin}
	t.bridge = ime.New(poster, open, func(ev ui.TextInputEvent) {
		if h := p.handler; h != nil {
			h.OnTextInput(ev)
		}
	})
	return t
}

// sessionOpener connects to the input method framework that the environment
// names, or the one that is running.
func sessionOpener() ime.Opener {
	opts := dbus.IMEOptions{
		Hint:    dbus.IMEHintFromEnv(os.Getenv),
		Program: filepath.Base(os.Args[0]),
		Display: "x11",
		IBusBus: dbus.IBusBusFromEnvironment("x11"),
	}
	return func(ctx context.Context, events func(dbus.IMEEvent)) (dbus.InputMethod, error) {
		return dbus.OpenSessionInputMethod(ctx, opts, events)
	}
}

// setTextInput is Window.SetTextInput.
func (p *Platform) setTextInput(w *window, state ui.TextInputState) {
	t := p.text
	if t == nil || w.closed {
		return
	}
	if !state.Enabled {
		if t.window != w {
			return // another window's field has the input now
		}
		t.window = nil
		t.want = ui.TextInputState{}
		t.bridge.SetState(ui.TextInputState{}, ime.Rect{})
		return
	}
	t.window, t.want = w, state
	t.send()
}

// send gives the input method the wanted state, if the window has the
// keyboard focus.
func (t *textInput) send() {
	w := t.window
	if w == nil || t.p.input == nil || t.p.input.focus != w {
		return
	}
	t.bridge.SetState(t.want, t.caret(w))
}

// caret returns the rectangle of the caret in screen pixels: the input method
// places its candidate window with it.
//
// TODO(step 3.7): the scale from XSETTINGS or Xft.dpi; until then a logical
// pixel is a device pixel.
func (t *textInput) caret(w *window) ime.Rect {
	x, y := t.origin(w)
	r := t.want.CaretRect
	return ime.Rect{X: x + int(r.Left), Y: y + int(r.Top), W: int(r.Width()), H: int(r.Height())}
}

// rootOrigin returns the position of the window's content in the root window.
func (w *window) rootOrigin() (x, y int) {
	p := w.p
	mem := dl.Calloc(1, 16)
	defer dl.Free(mem)
	if p.x.translateCoordinates(p.display, w.xid, p.root, 0, 0, mem, mem+4, mem+8) == 0 {
		return 0, 0
	}
	return int(*view[int32](mem)), int(*view[int32](mem + 4))
}

// textInputMoved sends the caret again when the window with the text field
// moved or changed size.
func (p *Platform) textInputMoved(w *window) {
	if t := p.text; t != nil && t.window == w {
		t.send()
	}
}

// focusChanged follows the keyboard focus: the input method is only told about
// the field while its window has the focus.
func (t *textInput) focusChanged(w *window, focused bool) {
	if t.window != w {
		return
	}
	if focused {
		t.send()
		return
	}
	t.consumed = map[uint32]bool{}
	t.bridge.SetState(ui.TextInputState{}, ime.Rect{})
}

// windowClosed forgets the window.
func (t *textInput) windowClosed(w *window) {
	if t.window != w {
		return
	}
	t.window = nil
	t.want = ui.TextInputState{}
	t.bridge.SetState(ui.TextInputState{}, ime.Rect{})
}

// close ends the connection to the input method.
func (t *textInput) close() { t.bridge.Close() }

// onKeyViaIME sends a key to the input method and, if it does not consume it,
// on to the app when the answer arrives. The answers come in the order of the
// keys.
func (in *inputState) onKeyViaIME(ev *xiDeviceEvent, press bool) {
	t := in.p.text
	code := uint32(ev.detail - evdevOffset)
	in.layout.UpdateMaskGroups(uint32(ev.mods.base), uint32(ev.mods.latched), uint32(ev.mods.locked),
		uint32(ev.group.base), uint32(ev.group.latched), uint32(ev.group.locked))
	key := dbus.IMEKey{
		Keysym: in.layout.Keysym(code), Keycode: code, State: in.layout.XState(),
		Release: !press, Time: uint32(ev.time),
	}
	saved := *ev // the C memory of the event is reused
	t.bridge.ProcessKey(key, func(handled bool) {
		if in.layout == nil {
			return // the platform was closed
		}
		switch {
		case !press && t.consumed[code]:
			delete(t.consumed, code) // the key-down never reached the app
		case press && handled:
			t.consumed[code] = true
		default:
			delete(t.consumed, code)
			in.dispatchKey(&saved, press)
		}
	})
}
