//go:build linux

package wayland

import (
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/internal/platform/linux/xkb"
	"github.com/rm4n0s/glutter/ui"
)

// Default key repeat, used until (or unless) the compositor sends
// wl_keyboard.repeat_info: 25 keys per second after 600 ms.
const (
	defaultRepeatRate  = 25
	defaultRepeatDelay = 600
)

// keyboard is the wl_keyboard of the seat: it follows the keymap and the
// modifiers the compositor sends, turns key events into ui.KeyData, repeats the
// key that is held down, and keeps the pressed keys consistent when the focus
// changes (SPEC §8.5, §8.11 "Input"). It belongs to the UI thread.
type keyboard struct {
	p   *Platform
	obj *wl.WlKeyboard

	ctx    *xkb.Context
	layout *xkb.Layout // nil until the compositor sent a usable keymap
	// keymapSeen is whether a wl_keyboard.keymap has been handled, usable or not.
	keymapSeen bool

	// focus is the window that has keyboard focus, or nil.
	focus *window
	// pressed holds the keys that are down, in the order they went down, with the
	// data of their key-down event: the key-up event reuses its physical and
	// logical key, even if the modifiers changed in between.
	pressed []pressedKey

	// Key repeat: rate keys per second after delay milliseconds. At most one key
	// repeats, the one pressed last.
	rate, delay int32
	repeat      *repeating

	// after schedules fn on the UI thread; tests replace it with a fake clock.
	after func(d time.Duration, fn func()) (stop func())
}

type pressedKey struct {
	evdev uint32
	data  ui.KeyData
}

// repeating is the key that repeats, with the function that cancels its timer.
type repeating struct {
	evdev uint32
	data  ui.KeyData
	stop  func()
}

func newKeyboard(p *Platform) *keyboard {
	k := &keyboard{
		p:     p,
		rate:  defaultRepeatRate,
		delay: defaultRepeatDelay,
		after: p.afterFunc,
	}
	k.obj = p.sess.seat.GetKeyboard()
	k.obj.SetEvents(&wl.WlKeyboardEvents{
		OnKeymap:     k.onKeymap,
		OnEnter:      k.onEnter,
		OnLeave:      k.onLeave,
		OnKey:        k.onKey,
		OnModifiers:  k.onModifiers,
		OnRepeatInfo: k.onRepeatInfo,
	})
	return k
}

// afterFunc runs fn on the UI thread after d, and returns the function that
// cancels it.
func (p *Platform) afterFunc(d time.Duration, fn func()) (stop func()) {
	t, err := p.loop.AfterFunc(d, fn)
	if err != nil {
		p.fail(err)
	}
	return func() { t.Stop() }
}

// close drops the keyboard object and everything the keymap needed.
func (k *keyboard) close() {
	k.loseFocus() // releases the keys that are held, and stops the repeat
	if k.p.sess.seatVersion >= 3 {
		k.obj.Release() // wl_keyboard.release is since version 3
	}
	k.obj = nil
	if k.layout != nil {
		k.layout.Close()
		k.layout = nil
	}
	if k.ctx != nil {
		k.ctx.Close()
		k.ctx = nil
	}
}

// onKeymap reads the keymap that the compositor sends in a file. A keymap that
// cannot be used leaves the keyboard without a layout: the keys are ignored
// (and the error is logged) rather than ending the app.
func (k *keyboard) onKeymap(format wl.WlKeyboardKeymapFormat, fd int, size uint32) {
	k.keymapSeen = true
	defer func() {
		if err := syscall.Close(fd); err != nil {
			slog.Warn("closing the keymap file failed", "err", err)
		}
	}()
	if k.layout != nil {
		k.layout.Close()
		k.layout = nil
	}
	if format != wl.WlKeyboardKeymapFormatXkbV1 {
		slog.Error("the compositor sent a keymap that is not xkb_v1", "format", uint32(format))
		return
	}
	if k.ctx == nil {
		ctx, err := xkb.NewContext()
		if err != nil {
			slog.Error("the keyboard cannot be used", "err", err)
			return
		}
		k.ctx = ctx
	}
	keymap, err := k.ctx.KeymapFromFD(fd, size)
	if err != nil {
		slog.Error("the keymap of the compositor was rejected", "err", err)
		return
	}
	// A locale without a compose table only loses dead keys and compose sequences.
	compose, err := k.ctx.NewCompose(xkb.LocaleFromEnv(os.Getenv))
	if err != nil {
		slog.Warn("no compose table: dead keys and compose sequences are off", "err", err)
		compose = nil
	}
	if k.layout, err = xkb.NewLayout(keymap, compose); err != nil {
		slog.Error("the keyboard cannot be used", "err", err)
	}
}

func (k *keyboard) onModifiers(_, depressed, latched, locked, group uint32) {
	if k.layout != nil {
		k.layout.UpdateMask(depressed, latched, locked, group)
	}
}

func (k *keyboard) onRepeatInfo(rate, delay int32) {
	k.rate, k.delay = rate, delay
	if rate <= 0 {
		k.cancelRepeat() // a rate of 0 turns repeating off
	}
}

// onEnter gives the window focus and reports the keys that are already held as
// synthesized key-down events, so that the framework's pressed-key state matches
// the keyboard.
func (k *keyboard) onEnter(serial uint32, surface *wl.WlSurface, keys []uint32) {
	k.p.inputSerial = serial
	w := k.p.windowOf(surface)
	if w == nil {
		return // a surface that is not ours
	}
	k.loseFocus() // normally the leave came first
	k.focus = w
	if h := k.p.handler; h != nil {
		h.OnViewFocus(w.id, true)
	}
	if d := k.p.dbusIME; d != nil {
		defer d.focusChanged(w, true)
	}
	if k.layout == nil {
		return
	}
	for _, evdev := range keys {
		key, err := k.layout.Lookup(evdev)
		if err != nil {
			k.p.fail(err)
			return
		}
		data := ui.KeyData{
			TimeStamp: k.p.now(), Type: ui.KeyEventTypeDown, DeviceType: ui.KeyEventDeviceTypeKeyboard,
			Physical: key.Physical, Logical: key.Logical, Synthesized: true,
		}
		k.pressed = append(k.pressed, pressedKey{evdev, data})
		k.deliver(data)
	}
}

func (k *keyboard) onLeave(_ uint32, surface *wl.WlSurface) {
	if k.focus != nil && k.focus.surface == surface {
		k.loseFocus()
	}
}

// loseFocus stops the repeat and releases the keys that are still down, as
// synthesized key-up events: the key-up will not reach us once the focus is gone.
func (k *keyboard) loseFocus() {
	k.cancelRepeat()
	released := k.pressed
	k.pressed = nil
	if d := k.p.dbusIME; d != nil && k.focus != nil {
		d.focusChanged(k.focus, false)
	}
	if h, w := k.p.handler, k.focus; h != nil && w != nil {
		h.OnViewFocus(w.id, false)
	}
	k.focus = nil
	for _, pk := range released {
		up := pk.data
		up.TimeStamp = k.p.now()
		up.Type = ui.KeyEventTypeUp
		up.Character = ""
		up.Synthesized = true
		k.deliver(up)
	}
}

// windowClosed is called when a window is destroyed: it loses the focus.
func (k *keyboard) windowClosed(w *window) {
	if k.focus == w {
		k.loseFocus()
	}
}

func (k *keyboard) onKey(serial uint32, _ uint32, evdev uint32, state wl.WlKeyboardKeyState) {
	k.p.inputSerial = serial
	if k.focus == nil || k.layout == nil {
		return
	}
	switch state {
	case wl.WlKeyboardKeyStatePressed:
		k.press(evdev)
	case wl.WlKeyboardKeyStateReleased:
		k.release(evdev)
	}
	// wl_keyboard_key_state.repeated is not used: the repeats are made here.
}

func (k *keyboard) press(evdev uint32) {
	if k.pressedIndex(evdev) >= 0 {
		return // already down: the framework would see a second key-down
	}
	key, err := k.layout.Press(evdev)
	if err != nil {
		k.p.fail(err)
		return
	}
	data := ui.KeyData{
		TimeStamp: k.p.now(), Type: ui.KeyEventTypeDown, DeviceType: ui.KeyEventDeviceTypeKeyboard,
		Physical: key.Physical, Logical: key.Logical, Character: key.Character,
	}
	k.pressed = append(k.pressed, pressedKey{evdev, data})
	k.route(evdev, data)

	// Pressing another key ends the repeat of the previous one.
	k.cancelRepeat()
	if key.Repeats && k.rate > 0 {
		r := &repeating{evdev: evdev, data: data}
		r.stop = k.after(time.Duration(k.delay)*time.Millisecond, func() { k.fireRepeat(r) })
		k.repeat = r
	}
}

func (k *keyboard) release(evdev uint32) {
	i := k.pressedIndex(evdev)
	if i < 0 {
		return // the key-down was never seen, so there is nothing to release
	}
	up := k.pressed[i].data
	k.pressed = append(k.pressed[:i], k.pressed[i+1:]...)
	if k.repeat != nil && k.repeat.evdev == evdev {
		k.cancelRepeat()
	}
	up.TimeStamp = k.p.now()
	up.Type = ui.KeyEventTypeUp
	up.Character = ""
	up.Synthesized = false
	k.route(evdev, up)
}

// fireRepeat sends one repeat event and schedules the next.
func (k *keyboard) fireRepeat(r *repeating) {
	if k.repeat != r || k.rate <= 0 {
		return
	}
	data := r.data
	data.TimeStamp = k.p.now()
	data.Type = ui.KeyEventTypeRepeat
	k.route(r.evdev, data)
	// deliver may have ended this repeat (the handler closed the window).
	if k.repeat == r {
		r.stop = k.after(time.Second/time.Duration(k.rate), func() { k.fireRepeat(r) })
	}
}

func (k *keyboard) cancelRepeat() {
	if k.repeat != nil {
		k.repeat.stop()
		k.repeat = nil
	}
}

func (k *keyboard) pressedIndex(evdev uint32) int {
	for i, pk := range k.pressed {
		if pk.evdev == evdev {
			return i
		}
	}
	return -1
}

func (k *keyboard) deliver(data ui.KeyData) {
	if h := k.p.handler; h != nil {
		h.OnKey(data)
	}
}

// windowOf returns the window that draws on the surface, or nil.
func (p *Platform) windowOf(surface *wl.WlSurface) *window {
	if surface == nil {
		return nil
	}
	for _, w := range p.windows {
		if w.surface == surface {
			return w
		}
	}
	return nil
}
