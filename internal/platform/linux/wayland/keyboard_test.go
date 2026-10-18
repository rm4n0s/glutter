package wayland

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/internal/platform/linux/xkb"
	"github.com/rm4n0s/glutter/ui"
)

// Evdev scancodes and xkb modifier masks, as in the xkb package's tests.
const (
	keyQ         = 16
	keyE         = 18
	keyY         = 21
	keyA         = 30
	keyS         = 31
	keyGrave     = 41
	keyLeftShift = 42
	keySpace     = 57

	maskShift = 1 << 0
	maskAltGr = 1 << 7
)

// keymapFor returns the text of the keymap that the xkb data files have for the
// layouts, as a compositor would send it.
func keymapFor(t *testing.T, layout string) string {
	t.Helper()
	ctx, err := xkb.NewContext()
	if err != nil {
		t.Fatalf("libxkbcommon is needed (sudo dnf install libxkbcommon): %v", err)
	}
	defer ctx.Close()
	km, err := ctx.KeymapFromNames(xkb.Names{Layout: layout})
	if err != nil {
		t.Fatalf("keymap %q (sudo dnf install xkeyboard-config): %v", layout, err)
	}
	defer km.Close()
	return km.String()
}

// fakeClock is a clock that the test moves by hand. Its methods run on the UI
// thread only (the keyboard calls After from there; the test advances it through
// onLoop).
type fakeClock struct {
	now    time.Duration
	timers []*fakeTimer
}

type fakeTimer struct {
	when time.Duration
	fn   func()
	live bool
}

func (c *fakeClock) after(d time.Duration, fn func()) (stop func()) {
	t := &fakeTimer{when: c.now + d, fn: fn, live: true}
	c.timers = append(c.timers, t)
	return func() { t.live = false }
}

// advance moves the clock by d and runs the timers that fall due, in order,
// including the ones that they schedule.
func (c *fakeClock) advance(d time.Duration) {
	target := c.now + d
	for {
		var next *fakeTimer
		for _, t := range c.timers {
			if t.live && t.when <= target && (next == nil || t.when < next.when) {
				next = t
			}
		}
		if next == nil {
			break
		}
		c.now = next.when
		next.live = false
		next.fn()
	}
	c.now = target
}

// keyHandler records the key events.
type keyHandler struct {
	platform.NopHandler
	mu      sync.Mutex
	keys    []ui.KeyData
	batches [][]ui.PointerData
}

func (h *keyHandler) OnPointer(batch []ui.PointerData) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.batches = append(h.batches, append([]ui.PointerData(nil), batch...))
}

func (h *keyHandler) pointerBatches() [][]ui.PointerData {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([][]ui.PointerData(nil), h.batches...)
}

func (h *keyHandler) OnKey(d ui.KeyData) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.keys = append(h.keys, d)
}

func (h *keyHandler) seen() []ui.KeyData {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]ui.KeyData(nil), h.keys...)
}

// keyEnv is a platform with one window on a fake compositor that has a keyboard.
type keyEnv struct {
	t       *testing.T
	p       *Platform
	srv     *fakecomp.Server
	h       *keyHandler
	clock   *fakeClock
	surface uint32
	cancel  context.CancelFunc
	done    chan error
	seen    int // key events already checked

	seenBatches int // pointer batches already checked
}

// startKeyboard starts a platform whose keyboard uses the layout, with a window
// that has the keyboard focus and a fake clock for the key repeat.
func startKeyboard(t *testing.T, layout string) *keyEnv {
	t.Helper()
	t.Setenv("LC_ALL", "en_US.UTF-8") // the compose table
	e := startKeyboardNoFocus(t, fakecomp.Config{Keymap: keymapFor(t, layout)})
	e.srv.KeyboardEnter(e.surface)
	e.settle()
	return e
}

// startKeyboardNoFocus is startKeyboard for a fake compositor configuration of
// the test's choice, and does not give the window the focus.
func startKeyboardNoFocus(t *testing.T, cfg fakecomp.Config) *keyEnv {
	t.Helper()
	p, srv, err := connect(t, cfg)
	if err != nil {
		t.Fatal(err)
	}
	e := &keyEnv{t: t, p: p, srv: srv, h: &keyHandler{}, clock: &fakeClock{}}
	hasKeyboard := p.keyboard != nil
	if hasKeyboard {
		p.keyboard.after = e.clock.after
	}
	if _, err := p.NewWindow(config640()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	e.cancel = cancel
	e.done = make(chan error, 1)
	go func() { e.done <- p.Run(ctx, e.h) }()
	t.Cleanup(func() {
		cancel()
		<-e.done
	})
	waitUntil(t, func() bool { _, _, ok := srv.Mapped(); return ok }, "the window was never mapped")
	e.surface = srv.ID("wl_surface")
	if hasKeyboard {
		waitUntil(t, func() bool { return e.onLoop(func() bool { return p.keyboard.layout != nil || e.keymapRejected() }) }, "the keymap never arrived")
	}
	return e
}

// keymapRejected is for tests that send a keymap that cannot be used: it reports
// that the keymap event has been handled, which leaves the layout nil.
func (e *keyEnv) keymapRejected() bool { return e.p.keyboard.keymapSeen }

// onLoop runs fn on the UI thread and returns its result.
func (e *keyEnv) onLoop(fn func() bool) bool {
	e.t.Helper()
	res := make(chan bool, 1)
	e.p.Post(func() { res <- fn() })
	select {
	case v := <-res:
		return v
	case <-time.After(testTimeout):
		e.t.Fatal("the loop did not run a posted function")
		return false
	}
}

// onLoopErr runs fn on the UI thread and returns its error.
func (e *keyEnv) onLoopErr(fn func() error) error {
	e.t.Helper()
	res := make(chan error, 1)
	e.p.Post(func() { res <- fn() })
	select {
	case err := <-res:
		return err
	case <-time.After(testTimeout):
		e.t.Fatal("the loop did not run a posted function")
		return nil
	}
}

// settle waits until the loop has handled everything the fake compositor sent
// so far: a roundtrip on the UI thread.
func (e *keyEnv) settle() {
	e.t.Helper()
	e.onLoop(func() bool { return e.p.sess.conn.Roundtrip() == nil })
}

// advance moves the fake clock on the UI thread.
func (e *keyEnv) advance(d time.Duration) {
	e.t.Helper()
	e.onLoop(func() bool { e.clock.advance(d); return true })
}

// events returns the key events that arrived since the last call, after the
// compositor's messages were handled.
func (e *keyEnv) events() []ui.KeyData {
	e.t.Helper()
	e.settle()
	all := e.h.seen()
	out := all[e.seen:]
	e.seen = len(all)
	return out
}

// press and release send the key events and return what the app received.
func (e *keyEnv) press(evdev uint32) []ui.KeyData {
	e.srv.KeyboardKey(evdev, true)
	return e.events()
}

func (e *keyEnv) release(evdev uint32) []ui.KeyData {
	e.srv.KeyboardKey(evdev, false)
	return e.events()
}

func (e *keyEnv) modifiers(depressed, latched, locked, group uint32) {
	e.srv.KeyboardModifiers(depressed, latched, locked, group)
	e.settle()
}

// want is the part of a key event that the tests compare.
type want struct {
	typ         ui.KeyEventType
	physical    uint64
	logical     uint64
	char        string
	synthesized bool
}

func brief(events []ui.KeyData) []want {
	out := make([]want, len(events))
	for i, d := range events {
		out[i] = want{d.Type, d.Physical, d.Logical, d.Character, d.Synthesized}
	}
	return out
}

func checkEvents(t *testing.T, got []ui.KeyData, wants ...want) {
	t.Helper()
	if !reflect.DeepEqual(brief(got), append([]want{}, wants...)) {
		t.Errorf("key events:\n got  %+v\n want %+v", brief(got), wants)
	}
	for _, d := range got {
		if d.DeviceType != ui.KeyEventDeviceTypeKeyboard {
			t.Errorf("device type = %v, want keyboard", d.DeviceType)
		}
	}
}

const (
	down   = ui.KeyEventTypeDown
	up     = ui.KeyEventTypeUp
	repeat = ui.KeyEventTypeRepeat
)

func TestKeyboardPressAndRelease(t *testing.T) {
	e := startKeyboard(t, "de")
	// The key labelled Z sits where US has y.
	checkEvents(t, e.press(keyY), want{down, 0x7001c, 'z', "z", false})
	checkEvents(t, e.release(keyY), want{up, 0x7001c, 'z', "", false})

	e.modifiers(maskShift, 0, 0, 0)
	checkEvents(t, e.press(keyA), want{down, 0x70004, 'a', "A", false})
	e.modifiers(0, 0, 0, 0)
	// The key-up has the ids of the key-down, whatever the modifiers are now.
	checkEvents(t, e.release(keyA), want{up, 0x70004, 'a', "", false})

	e.modifiers(maskAltGr, 0, 0, 0)
	checkEvents(t, e.press(keyQ), want{down, 0x70014, 'q', "@", false})
	e.modifiers(0, 0, 0, 0)
	checkEvents(t, e.release(keyQ), want{up, 0x70014, 'q', "", false})
}

func TestKeyboardDeadKeyAndCompose(t *testing.T) {
	e := startKeyboard(t, "de")
	// Dead circumflex makes no text; the key after it makes ê.
	got := e.press(keyGrave)
	checkEvents(t, got, want{down, 0x70035, 0x1500000000 | 0xfe52, "", false})
	checkEvents(t, e.release(keyGrave), want{up, 0x70035, 0x1500000000 | 0xfe52, "", false})
	checkEvents(t, e.press(keyE), want{down, 0x70008, 'e', "ê", false})
	checkEvents(t, e.release(keyE), want{up, 0x70008, 'e', "", false})
	// The sequence is over: e is e again.
	checkEvents(t, e.press(keyE), want{down, 0x70008, 'e', "e", false})
}

func TestKeyboardIgnoresKeysWithoutFocus(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	e := startKeyboardNoFocus(t, fakecomp.Config{Keymap: keymapFor(t, "us")})
	checkEvents(t, e.press(keyA)) // nothing
	e.srv.KeyboardEnter(e.surface)
	e.settle()
	checkEvents(t, e.press(keyA), want{down, 0x70004, 'a', "a", false})
}

func TestKeyboardFocusSynthesizesKeys(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	e := startKeyboardNoFocus(t, fakecomp.Config{Keymap: keymapFor(t, "us")})

	// Keys that are held when the focus arrives are reported as down.
	e.srv.KeyboardEnter(e.surface, keyLeftShift, keyA)
	checkEvents(t, e.events(),
		want{down, 0x700e1, 0x200000102, "", true},
		want{down, 0x70004, 'a', "", true})

	// Keys that go down meanwhile are released, in the order they went down, when
	// the focus is lost.
	checkEvents(t, e.press(keyS), want{down, 0x70016, 's', "s", false})
	e.srv.KeyboardLeave(e.surface)
	checkEvents(t, e.events(),
		want{up, 0x700e1, 0x200000102, "", true},
		want{up, 0x70004, 'a', "", true},
		want{up, 0x70016, 's', "", true})

	// A key-up that arrives after the focus is gone is dropped.
	checkEvents(t, e.release(keyS))
}

func TestKeyboardClosingTheFocusedWindowReleasesKeys(t *testing.T) {
	e := startKeyboard(t, "us")
	checkEvents(t, e.press(keyA), want{down, 0x70004, 'a', "a", false})
	// Closing the only window ends Run, so the events are read in the same loop pass.
	var got []ui.KeyData
	e.onLoop(func() bool {
		for _, w := range e.p.windows {
			w.Close()
		}
		got = e.h.seen()[e.seen:]
		return true
	})
	checkEvents(t, got, want{up, 0x70004, 'a', "", true})
}

func TestKeyboardIgnoresDuplicateAndUnknownReleases(t *testing.T) {
	e := startKeyboard(t, "us")
	checkEvents(t, e.release(keyA)) // never went down
	checkEvents(t, e.press(keyA), want{down, 0x70004, 'a', "a", false})
	checkEvents(t, e.press(keyA)) // a second down for the same key
	checkEvents(t, e.release(keyA), want{up, 0x70004, 'a', "", false})
}

func TestKeyRepeatTiming(t *testing.T) {
	e := startKeyboard(t, "us")
	// 25 keys per second after 400 ms.
	e.srv.KeyboardRepeatInfo(25, 400)
	e.settle()

	checkEvents(t, e.press(keyA), want{down, 0x70004, 'a', "a", false})
	e.advance(399 * time.Millisecond)
	checkEvents(t, e.events()) // the delay is not over
	e.advance(1 * time.Millisecond)
	checkEvents(t, e.events(), want{repeat, 0x70004, 'a', "a", false})
	e.advance(39 * time.Millisecond)
	checkEvents(t, e.events())
	e.advance(1 * time.Millisecond)
	checkEvents(t, e.events(), want{repeat, 0x70004, 'a', "a", false})
	// A long pause is caught up one event per interval.
	e.advance(120 * time.Millisecond)
	if got := len(e.events()); got != 3 {
		t.Errorf("repeats in 120 ms = %d, want 3", got)
	}

	// Releasing stops the repeat.
	checkEvents(t, e.release(keyA), want{up, 0x70004, 'a', "", false})
	e.advance(time.Second)
	checkEvents(t, e.events())
}

func TestKeyRepeatEndsWhenAnotherKeyIsPressed(t *testing.T) {
	e := startKeyboard(t, "us")
	e.srv.KeyboardRepeatInfo(10, 100)
	e.settle()

	e.press(keyA)
	e.advance(100 * time.Millisecond) // a repeats
	checkEvents(t, e.events(), want{repeat, 0x70004, 'a', "a", false})

	checkEvents(t, e.press(keyS), want{down, 0x70016, 's', "s", false})
	e.advance(100 * time.Millisecond) // s repeats; a does not any more
	checkEvents(t, e.events(), want{repeat, 0x70016, 's', "s", false})

	// Releasing a, which does not repeat, does not stop s.
	checkEvents(t, e.release(keyA), want{up, 0x70004, 'a', "", false})
	e.advance(100 * time.Millisecond)
	checkEvents(t, e.events(), want{repeat, 0x70016, 's', "s", false})
}

func TestKeyRepeatIsOffForModifiersAndRateZero(t *testing.T) {
	e := startKeyboard(t, "us")
	e.srv.KeyboardRepeatInfo(25, 100)
	e.settle()

	checkEvents(t, e.press(keyLeftShift), want{down, 0x700e1, 0x200000102, "", false})
	e.advance(time.Second)
	checkEvents(t, e.events()) // modifiers do not repeat
	e.release(keyLeftShift)

	// A rate of 0 turns repeating off, also for a key that is already repeating.
	e.press(keyA)
	e.advance(100 * time.Millisecond)
	checkEvents(t, e.events(), want{repeat, 0x70004, 'a', "a", false})
	e.srv.KeyboardRepeatInfo(0, 100)
	e.settle()
	e.advance(time.Second)
	checkEvents(t, e.events())
	e.release(keyA)
	e.press(keyS)
	e.advance(time.Second)
	checkEvents(t, e.events())
}

func TestKeyRepeatStopsWhenTheFocusIsLost(t *testing.T) {
	e := startKeyboard(t, "us")
	e.srv.KeyboardRepeatInfo(10, 100)
	e.settle()
	e.press(keyA)
	e.srv.KeyboardLeave(e.surface)
	checkEvents(t, e.events(), want{up, 0x70004, 'a', "", true})
	e.advance(time.Second)
	checkEvents(t, e.events())
}

func TestKeyRepeatDefaultsUntilTheCompositorSaysOtherwise(t *testing.T) {
	e := startKeyboard(t, "us")
	e.press(keyA)
	e.advance(599 * time.Millisecond)
	checkEvents(t, e.events())
	e.advance(1 * time.Millisecond)
	checkEvents(t, e.events(), want{repeat, 0x70004, 'a', "a", false})
	e.advance(40 * time.Millisecond)
	checkEvents(t, e.events(), want{repeat, 0x70004, 'a', "a", false})
}

func TestKeyRepeatOfADeadKeySequenceKeepsTheKeyDownText(t *testing.T) {
	e := startKeyboard(t, "de")
	e.srv.KeyboardRepeatInfo(10, 100)
	e.settle()
	e.press(keyGrave) // dead key: no text
	e.advance(100 * time.Millisecond)
	// The repeat does not feed the compose state.
	checkEvents(t, e.events(), want{repeat, 0x70035, 0x1500000000 | 0xfe52, "", false})
	e.release(keyGrave)
	checkEvents(t, e.press(keyE), want{down, 0x70008, 'e', "ê", false})
}

func TestSeatWithoutKeyboardAndCapabilityChanges(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	e := startKeyboardNoFocus(t, fakecomp.Config{Capabilities: 1, Keymap: keymapFor(t, "us")}) // a pointer only
	if e.onLoop(func() bool { return e.p.keyboard != nil }) {
		t.Fatal("a keyboard was made for a seat without one")
	}
	for _, line := range e.srv.Requests() {
		if line == "wl_seat.get_keyboard" {
			t.Fatal("get_keyboard was sent to a seat without a keyboard")
		}
	}

	// The keyboard is plugged in.
	e.srv.SeatCapabilities(3)
	waitUntil(t, func() bool { return e.onLoop(func() bool { return e.p.keyboard != nil && e.p.keyboard.layout != nil }) }, "the keyboard was not set up")
	// The fake compositor's new keyboard has the window in focus, with a and s held.
	checkEvents(t, e.events(),
		want{down, 0x70004, 'a', "", true},
		want{down, 0x70016, 's', "", true})
	checkEvents(t, e.press(keyE), want{down, 0x70008, 'e', "e", false})

	// And unplugged, with keys still down: they are released, and the keyboard is
	// handed back to the compositor.
	e.srv.SeatCapabilities(1)
	waitUntil(t, func() bool { return e.onLoop(func() bool { return e.p.keyboard == nil }) }, "the keyboard was not dropped")
	checkEvents(t, e.events(),
		want{up, 0x70004, 'a', "", true},
		want{up, 0x70016, 's', "", true},
		want{up, 0x70008, 'e', "", true})
	waitUntil(t, func() bool {
		for _, line := range e.srv.Requests() {
			if line == "wl_keyboard.release" {
				return true
			}
		}
		return false
	}, "wl_keyboard.release was not sent")
}

func TestUnusableKeymapLeavesTheKeyboardSilent(t *testing.T) {
	// The default keymap of the fake is valid; this one is not.
	e := startKeyboardNoFocus(t, fakecomp.Config{Keymap: "this is not a keymap"})
	e.srv.KeyboardEnter(e.surface)
	e.settle()
	checkEvents(t, e.press(keyA))
	checkEvents(t, e.release(keyA))
}
