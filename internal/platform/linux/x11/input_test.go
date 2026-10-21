//go:build linux

package x11

import (
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/internal/platform/linux/input"
	"github.com/rm4n0s/glutter/internal/platform/linux/xkb"
	"github.com/rm4n0s/glutter/ui"
)

// recorder is a handler that keeps the key and pointer events.
type recorder struct {
	platform.NopHandler
	keys    []ui.KeyData
	batches [][]ui.PointerData
}

func (r *recorder) OnKey(d ui.KeyData) { r.keys = append(r.keys, d) }
func (r *recorder) OnPointer(b []ui.PointerData) {
	r.batches = append(r.batches, append([]ui.PointerData(nil), b...))
}

// takeKeys returns the key events since the last call.
func (r *recorder) takeKeys() []ui.KeyData {
	k := r.keys
	r.keys = nil
	return k
}

func (r *recorder) takeBatches() [][]ui.PointerData {
	b := r.batches
	r.batches = nil
	return b
}

// newInput builds the input state around a keymap from the xkb data files, with
// no X server: the handlers are called with events that the test fills in.
func newInput(t *testing.T, layout string) (*inputState, *recorder) {
	t.Helper()
	t.Setenv("LC_ALL", "en_US.UTF-8")
	ctx, err := xkb.NewContext()
	if err != nil {
		t.Fatalf("libxkbcommon is needed (sudo dnf install libxkbcommon): %v", err)
	}
	t.Cleanup(ctx.Close)
	km, err := ctx.KeymapFromNames(xkb.Names{Layout: layout})
	if err != nil {
		t.Fatalf("keymap %q (sudo dnf install xkeyboard-config): %v", layout, err)
	}
	compose, err := ctx.NewCompose("en_US.UTF-8")
	if err != nil {
		t.Fatal(err)
	}
	l, err := xkb.NewLayout(km, compose)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(l.Close)
	rec := &recorder{}
	now := time.Now()
	p := &Platform{start: now, handler: rec, windows: map[uint64]*window{}}
	in := &inputState{
		p: p, layout: l,
		pointer: input.NewPointerBatcher(p.now), touch: input.NewTouchBatcher(p.now),
		scroll: map[scrollKey]*scrollAxis{},
	}
	return in, rec
}

// keyEvent makes an XIDeviceEvent for a key, with the modifiers.
func keyEvent(keycode int32, flags int32, mods xiModifierState) *xiDeviceEvent {
	return &xiDeviceEvent{detail: keycode, flags: flags, mods: mods}
}

const (
	xA     = 38 // the XKB keycode of the key labelled A (evdev 30 + 8)
	xE     = 26
	xGrave = 49
	xShift = 50
	xCtrl  = 37
)

func TestKeyPressReleaseAndRepeat(t *testing.T) {
	in, rec := newInput(t, "us")
	in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	in.onKey(keyEvent(xA, xiKeyRepeat, xiModifierState{}), true)
	in.onKey(keyEvent(xA, xiKeyRepeat, xiModifierState{}), true)
	in.onKey(keyEvent(xA, 0, xiModifierState{}), false)
	got := rec.takeKeys()
	types := []ui.KeyEventType{ui.KeyEventTypeDown, ui.KeyEventTypeRepeat, ui.KeyEventTypeRepeat, ui.KeyEventTypeUp}
	if len(got) != len(types) {
		t.Fatalf("%d events, want %d: %v", len(got), len(types), got)
	}
	for i, d := range got {
		if d.Type != types[i] || d.Physical != 0x70004 || d.Logical != 'a' || d.DeviceType != ui.KeyEventDeviceTypeKeyboard {
			t.Errorf("event %d: %+v", i, d)
		}
		wantChar := "a"
		if d.Type == ui.KeyEventTypeUp {
			wantChar = ""
		}
		if d.Character != wantChar {
			t.Errorf("event %d: character %q, want %q", i, d.Character, wantChar)
		}
	}
}

func TestKeyModifiersComeFromTheEvent(t *testing.T) {
	in, rec := newInput(t, "us")
	shift := xiModifierState{base: 1, effective: 1} // the Shift modifier
	in.onKey(keyEvent(xA, 0, shift), true)
	// Shift goes up before the key: the key-up still has the ids of the key-down.
	in.onKey(keyEvent(xA, 0, xiModifierState{}), false)
	in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	in.onKey(keyEvent(xA, 0, xiModifierState{}), false)
	got := rec.takeKeys()
	if len(got) != 4 || got[0].Character != "A" || got[1].Character != "" || got[2].Character != "a" {
		t.Fatalf("got %v", got)
	}
	if got[0].Logical != 'a' || got[1].Logical != 'a' || got[1].Physical != got[0].Physical {
		t.Errorf("ids: %+v then %+v", got[0], got[1])
	}
}

func TestKeyDeadKeyAndCompose(t *testing.T) {
	in, rec := newInput(t, "de")
	in.onKey(keyEvent(xGrave, 0, xiModifierState{}), true) // dead circumflex
	in.onKey(keyEvent(xGrave, 0, xiModifierState{}), false)
	in.onKey(keyEvent(xE, 0, xiModifierState{}), true)
	in.onKey(keyEvent(xE, 0, xiModifierState{}), false)
	got := rec.takeKeys()
	if len(got) != 4 || got[0].Character != "" || got[2].Character != "ê" {
		t.Fatalf("got %v", got)
	}
}

func TestKeyGroupFromTheEvent(t *testing.T) {
	// A keymap with two layouts: the event's group picks one.
	in, rec := newInput(t, "us,ru")
	in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	in.onKey(keyEvent(xA, 0, xiModifierState{}), false)
	locked := xiModifierState{locked: 1, effective: 1}
	ev := keyEvent(xA, 0, xiModifierState{})
	ev.group = locked
	in.onKey(ev, true)
	got := rec.takeKeys()
	if got[0].Character != "a" || got[2].Character != "ф" || got[2].Logical != 'a' {
		t.Errorf("got %+v", got)
	}
}

func TestKeyIgnoredWithoutKeymapAndForSmallKeycodes(t *testing.T) {
	in, rec := newInput(t, "us")
	in.onKey(keyEvent(7, 0, xiModifierState{}), true) // below the keyboard range
	in.onKey(keyEvent(0, 0, xiModifierState{}), true)
	in.layout = nil
	in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	if got := rec.takeKeys(); len(got) != 0 {
		t.Errorf("events: %v", got)
	}
}

func TestKeyDuplicateDownAndUnknownUp(t *testing.T) {
	in, rec := newInput(t, "us")
	in.onKey(keyEvent(xA, 0, xiModifierState{}), false) // never went down
	in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	in.onKey(keyEvent(xA, 0, xiModifierState{}), true) // a second, unflagged down
	got := rec.takeKeys()
	if len(got) != 1 || got[0].Type != ui.KeyEventTypeDown {
		t.Errorf("got %v", got)
	}
}

func TestKeyRepeatWithoutDownIsADown(t *testing.T) {
	in, rec := newInput(t, "us")
	in.onKey(keyEvent(xA, xiKeyRepeat, xiModifierState{}), true)
	got := rec.takeKeys()
	if len(got) != 1 || got[0].Type != ui.KeyEventTypeDown {
		t.Errorf("got %v", got)
	}
}

func TestFocusLossReleasesKeys(t *testing.T) {
	in, rec := newInput(t, "us")
	w := &window{id: 4}
	in.onFocus(w, &xiEnterEvent{detail: 0}, true)
	in.onKey(keyEvent(xShift, 0, xiModifierState{}), true)
	in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	rec.takeKeys()

	// Focus moves to a child (inferior), or to PointerRoot or None: not a loss.
	in.onFocus(w, &xiEnterEvent{detail: notifyInferior}, false)
	in.onFocus(w, &xiEnterEvent{detail: notifyPointerRoot}, false)
	in.onFocus(w, &xiEnterEvent{detail: notifyDetailNone}, false)
	if got := rec.takeKeys(); len(got) != 0 {
		t.Fatalf("keys released for a focus that did not move: %v", got)
	}
	in.onFocus(w, &xiEnterEvent{detail: 0}, false)
	got := rec.takeKeys()
	if len(got) != 2 || got[0].Physical != 0x700e1 || got[1].Physical != 0x70004 {
		t.Fatalf("got %+v", got)
	}
	for _, d := range got {
		if d.Type != ui.KeyEventTypeUp || !d.Synthesized {
			t.Errorf("released key: %+v", d)
		}
	}
	// With the focus following the pointer, the pointer leaving is a loss.
	in.onFocus(w, &xiEnterEvent{detail: notifyPointer}, true)
	in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	rec.takeKeys()
	in.onFocus(w, &xiEnterEvent{detail: notifyPointer}, false)
	if got := rec.takeKeys(); len(got) != 1 || got[0].Type != ui.KeyEventTypeUp || !got[0].Synthesized {
		t.Errorf("keys after the focus followed the pointer out: %v", got)
	}
	// Losing the focus of a window that is not the focus does nothing.
	in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	rec.takeKeys()
	in.onFocus(&window{id: 9}, &xiEnterEvent{detail: 0}, false)
	if len(rec.takeKeys()) != 0 {
		t.Error("keys released for another window")
	}
}

func TestWindowClosedReleasesKeysAndLetsThePointerGo(t *testing.T) {
	in, rec := newInput(t, "us")
	w := &window{id: 2}
	in.onFocus(w, &xiEnterEvent{}, true)
	in.onKey(keyEvent(xA, 0, xiModifierState{}), true)
	in.onEnterLeave(w, &xiEnterEvent{eventX: 5, eventY: 6}, true)
	in.onTouch(w, &xiDeviceEvent{detail: 3, eventX: 1, eventY: 1}, xiTouchBegin)
	rec.takeKeys()
	rec.takeBatches()

	in.windowClosed(w)
	if got := rec.takeKeys(); len(got) != 1 || got[0].Type != ui.KeyEventTypeUp || !got[0].Synthesized {
		t.Errorf("keys: %v", got)
	}
	var changes []ui.PointerChange
	for _, b := range rec.takeBatches() {
		for _, d := range b {
			changes = append(changes, d.Change)
		}
	}
	want := []ui.PointerChange{ui.PointerChangeRemove, ui.PointerChangeCancel, ui.PointerChangeRemove}
	if !reflect.DeepEqual(changes, want) {
		t.Errorf("pointer changes %v, want %v", changes, want)
	}
}

func pointerChanges(batches [][]ui.PointerData) []ui.PointerChange {
	var out []ui.PointerChange
	for _, b := range batches {
		for _, d := range b {
			out = append(out, d.Change)
		}
	}
	return out
}

func TestPointerEnterMotionButtonsLeave(t *testing.T) {
	in, rec := newInput(t, "us")
	w := &window{id: 1}
	in.onEnterLeave(w, &xiEnterEvent{eventX: 10, eventY: 20}, true)
	in.onMotion(w, &xiDeviceEvent{eventX: 12, eventY: 22})
	in.onMotion(w, &xiDeviceEvent{eventX: 12, eventY: 22}) // no movement: nothing
	in.onButton(w, &xiDeviceEvent{detail: 1, eventX: 12, eventY: 22}, true)
	in.onButton(w, &xiDeviceEvent{detail: 3, eventX: 12, eventY: 22}, true)
	in.onButton(w, &xiDeviceEvent{detail: 3, eventX: 12, eventY: 22}, false)
	in.onButton(w, &xiDeviceEvent{detail: 1, eventX: 15, eventY: 22}, false) // moved while pressed
	in.onEnterLeave(w, &xiEnterEvent{}, false)
	batches := rec.takeBatches()
	want := []ui.PointerChange{
		ui.PointerChangeAdd, ui.PointerChangeHover, ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeMove,
		ui.PointerChangeMove, ui.PointerChangeUp, ui.PointerChangeRemove,
	}
	if got := pointerChanges(batches); !reflect.DeepEqual(got, want) {
		t.Fatalf("changes %v, want %v", got, want)
	}
	// One batch per X event.
	if len(batches) != 7 {
		t.Errorf("%d batches, want 7", len(batches))
	}
	down := batches[2][0]
	if down.Buttons != 1 || down.PhysicalX != 12 || down.ViewID != 1 {
		t.Errorf("down: %+v", down)
	}
}

func TestPointerButtonMappingAndIgnoredButtons(t *testing.T) {
	tests := []struct {
		detail int32
		want   uint32
	}{{1, 0x110}, {2, 0x112}, {3, 0x111}, {8, 0x116}, {9, 0x115}, {4, 0}, {10, 0}, {0, 0}}
	for _, tt := range tests {
		if got := linuxButton(tt.detail); got != tt.want {
			t.Errorf("linuxButton(%d) = 0x%x, want 0x%x", tt.detail, got, tt.want)
		}
	}
	for _, tt := range []struct {
		detail int32
		axis   input.Axis
		dir    int32
		ok     bool
	}{{4, input.AxisVertical, -1, true}, {5, input.AxisVertical, 1, true}, {6, input.AxisHorizontal, -1, true}, {7, input.AxisHorizontal, 1, true}, {1, 0, 0, false}, {8, 0, 0, false}} {
		axis, dir, ok := wheelButton(tt.detail)
		if ok != tt.ok || (ok && (axis != tt.axis || dir != tt.dir)) {
			t.Errorf("wheelButton(%d) = %v, %d, %v", tt.detail, axis, dir, ok)
		}
	}
}

func TestPointerGrabAndInferiorCrossingsAreIgnored(t *testing.T) {
	in, rec := newInput(t, "us")
	w := &window{id: 1}
	in.onEnterLeave(w, &xiEnterEvent{mode: notifyGrab}, true)
	in.onEnterLeave(w, &xiEnterEvent{detail: notifyInferior}, true)
	if len(rec.takeBatches()) != 0 {
		t.Fatal("a grab or inferior crossing was an enter")
	}
	in.onEnterLeave(w, &xiEnterEvent{}, true)
	in.onEnterLeave(w, &xiEnterEvent{mode: notifyGrab}, false)
	in.onEnterLeave(w, &xiEnterEvent{detail: notifyInferior}, false)
	in.onEnterLeave(&window{id: 7}, &xiEnterEvent{}, false) // another window
	if got := pointerChanges(rec.takeBatches()); !reflect.DeepEqual(got, []ui.PointerChange{ui.PointerChangeAdd}) {
		t.Errorf("changes %v, want only the add", got)
	}
}

func TestPointerEventsOutsideTheWindowThePointerIsInAreIgnored(t *testing.T) {
	in, rec := newInput(t, "us")
	w := &window{id: 1}
	in.onMotion(w, &xiDeviceEvent{eventX: 1, eventY: 1})
	in.onButton(w, &xiDeviceEvent{detail: 1}, true)
	if len(rec.takeBatches()) != 0 {
		t.Error("events before the pointer entered")
	}
}

// valuatorEvent makes an XIDeviceEvent whose valuators carry the values for the
// numbered axes, in C memory as the server's are. free releases it.
func valuatorEvent(t *testing.T, device int32, values map[int]float64) *xiDeviceEvent {
	t.Helper()
	const maskLen = 4
	mask := dl.Calloc(1, maskLen)
	vals := dl.Calloc(uintptr(len(values)+1), 8)
	t.Cleanup(func() { dl.Free(mask); dl.Free(vals) })
	index := 0
	for number := range maskLen * 8 {
		v, ok := values[number]
		if !ok {
			continue
		}
		dl.Bytes(mask, maskLen)[number>>3] |= 1 << (number & 7)
		*(*float64)(unsafe.Add(dl.Pointer(vals), index*8)) = v
		index++
	}
	return &xiDeviceEvent{deviceID: device, valuators: xiValuatorState{maskLen: maskLen, mask: mask, values: vals}}
}

func TestSmoothScrolling(t *testing.T) {
	in, rec := newInput(t, "us")
	w := &window{id: 1}
	in.onEnterLeave(w, &xiEnterEvent{eventX: 4, eventY: 6}, true)
	rec.takeBatches()
	// Valuator 2 scrolls vertically, 3 horizontally; one notch is 15 units.
	in.scroll[scrollKey{2, 2}] = &scrollAxis{axis: input.AxisVertical, increment: 15, last: 100}
	in.scroll[scrollKey{2, 3}] = &scrollAxis{axis: input.AxisHorizontal, increment: 15, last: 0}

	// A notch down: the vertical value goes up by one increment.
	ev := valuatorEvent(t, 2, map[int]float64{2: 115})
	ev.eventX, ev.eventY = 4, 6
	in.onMotion(w, ev)
	b := rec.takeBatches()
	if len(b) != 1 || len(b[0]) != 1 || b[0][0].SignalKind != ui.PointerSignalKindScroll || b[0][0].ScrollDeltaY != 53 || b[0][0].ScrollDeltaX != 0 {
		t.Fatalf("one notch down: %+v", b)
	}
	// Half a notch to the left, and a notch up, together with other axes (0, 1 are
	// x and y, which have no scroll class).
	ev = valuatorEvent(t, 2, map[int]float64{0: 4, 1: 6, 2: 107.5, 3: -7.5})
	ev.eventX, ev.eventY = 4, 6
	in.onMotion(w, ev)
	b = rec.takeBatches()
	d := b[0][0]
	if d.ScrollDeltaY != -26.5 || d.ScrollDeltaX != -26.5 {
		t.Errorf("deltas %v, %v; want -26.5 and -26.5", d.ScrollDeltaX, d.ScrollDeltaY)
	}
	// A device that is not the one with the axes, and an event without valuators.
	in.onMotion(w, valuatorEvent(t, 9, map[int]float64{2: 500}))
	in.onMotion(w, &xiDeviceEvent{})
	for _, batch := range rec.takeBatches() {
		for _, d := range batch {
			if d.SignalKind == ui.PointerSignalKindScroll {
				t.Errorf("a scroll from nothing: %+v", d)
			}
		}
	}
}

func TestLegacyScrollButtons(t *testing.T) {
	in, rec := newInput(t, "us")
	w := &window{id: 1}
	in.onEnterLeave(w, &xiEnterEvent{eventX: 4, eventY: 6}, true)
	rec.takeBatches()

	// Without scroll axes the buttons are the scrolling: a notch per press, none
	// for the release.
	in.onButton(w, &xiDeviceEvent{detail: 5}, true)
	in.onButton(w, &xiDeviceEvent{detail: 5}, false)
	in.onButton(w, &xiDeviceEvent{detail: 4}, true)
	in.onButton(w, &xiDeviceEvent{detail: 7}, true)
	var deltas [][2]float64
	for _, batch := range rec.takeBatches() {
		for _, d := range batch {
			deltas = append(deltas, [2]float64{d.ScrollDeltaX, d.ScrollDeltaY})
		}
	}
	want := [][2]float64{{0, 53}, {0, -53}, {53, 0}}
	if !reflect.DeepEqual(deltas, want) {
		t.Errorf("deltas %v, want %v", deltas, want)
	}

	// An emulated press is not a second scroll, and a device with scroll axes does
	// not use the buttons.
	in.onButton(w, &xiDeviceEvent{detail: 5, flags: xiPointerEmulated}, true)
	in.scroll[scrollKey{3, 2}] = &scrollAxis{axis: input.AxisVertical, increment: 1}
	in.onButton(w, &xiDeviceEvent{detail: 5, deviceID: 3}, true)
	if got := rec.takeBatches(); len(got) != 0 {
		t.Errorf("batches %v for emulated scrolling", got)
	}
	if !in.hasScroll(3) || in.hasScroll(4) {
		t.Error("hasScroll is wrong")
	}
}

func TestTouchEvents(t *testing.T) {
	in, rec := newInput(t, "us")
	w := &window{id: 1}
	in.onTouch(w, &xiDeviceEvent{detail: 100, eventX: 5, eventY: 6}, xiTouchBegin)
	in.onTouch(w, &xiDeviceEvent{detail: 100, eventX: 7, eventY: 6}, xiTouchUpdate)
	in.onTouch(w, &xiDeviceEvent{detail: 100}, xiTouchEnd)
	want := []ui.PointerChange{
		ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeUp, ui.PointerChangeRemove,
	}
	batches := rec.takeBatches()
	if got := pointerChanges(batches); !reflect.DeepEqual(got, want) {
		t.Errorf("changes %v, want %v", got, want)
	}
	if batches[0][0].Kind != ui.PointerDeviceKindTouch || batches[0][0].Device != input.TouchDeviceBase+100 {
		t.Errorf("touch data: %+v", batches[0][0])
	}
}

func TestHandleDispatchesByEventType(t *testing.T) {
	in, rec := newInput(t, "us")
	w := &window{id: 1}
	in.p.windows[0x500] = w
	in.p.windows[0x600] = &window{id: 2}

	mem := dl.Calloc(1, unsafe.Sizeof(xiDeviceEvent{}))
	defer dl.Free(mem)
	ev := view[xiDeviceEvent](mem)
	ev.event, ev.detail = 0x500, xA
	in.handle(xiKeyPress, mem)
	in.handle(xiKeyRelease, mem)
	if got := rec.takeKeys(); len(got) != 2 {
		t.Errorf("%d key events through handle, want 2", len(got))
	}
	// Events for a window we do not know are dropped, as are unknown types.
	ev.event = 0x999
	in.handle(xiKeyPress, mem)
	in.handle(xiMotion, mem)
	in.handle(99, mem)
	if len(rec.takeKeys()) != 0 || len(rec.takeBatches()) != 0 {
		t.Error("events for an unknown window were delivered")
	}
}

func TestXIMask(t *testing.T) {
	tests := []struct {
		types []int
		want  []byte
	}{
		{nil, []byte{0, 0, 0, 0}},
		{[]int{xiDeviceChanged}, []byte{0x02, 0, 0, 0}},
		{[]int{xiKeyPress, xiKeyRelease}, []byte{0x0c, 0, 0, 0}},
		{[]int{xiEnter, xiLeave, xiFocusIn, xiFocusOut}, []byte{0x80, 0x07, 0, 0}},
		{[]int{xiTouchBegin, xiTouchUpdate, xiTouchEnd}, []byte{0, 0, 0x1c, 0}},
		{[]int{xiHierarchyChanged}, []byte{0, 0x08, 0, 0}},
	}
	for _, tt := range tests {
		if got := xiMask(tt.types...); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("xiMask(%v) = %v, want %v", tt.types, got, tt.want)
		}
	}
}
