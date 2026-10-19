package wayland

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/platform/linux/ime"
	"github.com/rm4n0s/glutter/ui"
)

// dbusIME is the input method of a compositor that has no
// zwp_text_input_manager_v3: IBus or Fcitx5 over D-Bus, as on X11 (SPEC §8.3,
// §8.11 "Input"). The keys the keyboard sends go to the input method first;
// the ones it consumes are not sent to the app. It belongs to the UI thread.
type dbusIME struct {
	p      *Platform
	bridge *ime.Bridge

	// window is the window whose text field has the text input, and want the
	// state it asked for.
	window *window
	want   ui.TextInputState
	// consumed are the keys (evdev codes) whose key-down the input method
	// consumed, so their key-up is not sent either.
	consumed map[uint32]bool
}

// newDBUSIME makes the input method over D-Bus. The connection is made when
// the first text field gets the focus; open is how, and tests replace it.
func newDBUSIME(p *Platform, poster ime.Poster, open ime.Opener) *dbusIME {
	d := &dbusIME{p: p, consumed: map[uint32]bool{}}
	d.bridge = ime.New(poster, open, func(ev ui.TextInputEvent) {
		if h := p.handler; h != nil {
			h.OnTextInput(ev)
		}
	})
	return d
}

// sessionIMEOpener connects to the input method framework that the environment
// names, or the one that is running.
func sessionIMEOpener() ime.Opener {
	opts := dbus.IMEOptions{
		Hint:    dbus.IMEHintFromEnv(os.Getenv),
		Program: filepath.Base(os.Args[0]),
		Display: "wayland",
		IBusBus: dbus.IBusBusFromEnvironment("wayland"),
	}
	return func(ctx context.Context, events func(dbus.IMEEvent)) (dbus.InputMethod, error) {
		return dbus.OpenSessionInputMethod(ctx, opts, events)
	}
}

// set is Window.SetTextInput.
func (d *dbusIME) set(w *window, state ui.TextInputState) {
	if w.closed {
		return
	}
	if !state.Enabled {
		if d.window != w {
			return // another window's field has the input now
		}
		d.window = nil
		d.want = ui.TextInputState{}
		d.bridge.SetState(ui.TextInputState{}, ime.Rect{})
		return
	}
	d.window, d.want = w, state
	d.send()
}

// send gives the input method the wanted state, if the window has the
// keyboard focus. The caret is in the logical pixels of the surface: a
// Wayland client does not know where its surface is on the screen.
func (d *dbusIME) send() {
	w := d.window
	if w == nil {
		return
	}
	if kb := d.p.keyboard; kb == nil || kb.focus != w {
		return
	}
	r := d.want.CaretRect
	d.bridge.SetState(d.want, ime.Rect{X: int(r.Left), Y: int(r.Top), W: int(r.Width()), H: int(r.Height())})
}

// focusChanged follows the keyboard focus: the input method is only told
// about the field while its window has the focus.
func (d *dbusIME) focusChanged(w *window, focused bool) {
	if d.window != w {
		return
	}
	if focused {
		d.send()
		return
	}
	d.consumed = map[uint32]bool{}
	d.bridge.SetState(ui.TextInputState{}, ime.Rect{})
}

// forget is called when a window is destroyed.
func (d *dbusIME) forget(w *window) {
	if d.window != w {
		return
	}
	d.window = nil
	d.want = ui.TextInputState{}
	d.bridge.SetState(ui.TextInputState{}, ime.Rect{})
}

// close ends the connection to the input method.
func (d *dbusIME) close() { d.bridge.Close() }

// route sends a key event of the keyboard to the app, through the input method
// when a text field has the focus: if the input method consumes the key-down,
// neither it nor its key-up reaches the app. The answers come in the order of
// the keys.
func (k *keyboard) route(evdev uint32, data ui.KeyData) {
	d := k.p.dbusIME
	if d == nil || !d.bridge.Active() || k.layout == nil {
		k.deliver(data)
		return
	}
	release := data.Type == ui.KeyEventTypeUp
	d.bridge.ProcessKey(dbus.IMEKey{
		Keysym: k.layout.Keysym(evdev), Keycode: evdev, State: k.layout.XState(),
		Release: release, Time: uint32(data.TimeStamp / time.Millisecond),
	}, func(handled bool) {
		switch {
		case release && d.consumed[evdev]:
			delete(d.consumed, evdev) // the key-down never reached the app
		case !release && handled:
			d.consumed[evdev] = true
		default:
			delete(d.consumed, evdev)
			k.deliver(data)
		}
	})
}
