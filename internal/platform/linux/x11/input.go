package x11

import (
	"log/slog"
	"os"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/internal/platform/linux/input"
	"github.com/rm4n0s/glutter/internal/platform/linux/xkb"
	"github.com/rm4n0s/glutter/ui"
)

// XInput2 event types (XI2.h).
const (
	xiDeviceChanged    = 1
	xiKeyPress         = 2
	xiKeyRelease       = 3
	xiButtonPress      = 4
	xiButtonRelease    = 5
	xiMotion           = 6
	xiEnter            = 7
	xiLeave            = 8
	xiFocusIn          = 9
	xiFocusOut         = 10
	xiHierarchyChanged = 11
	xiTouchBegin       = 18
	xiTouchUpdate      = 19
	xiTouchEnd         = 20

	xiKeyRepeat       = 1 << 16 // XIKeyRepeat: a key event is an auto-repeat
	xiPointerEmulated = 1 << 16 // XIPointerEmulated: a button event stands for a scroll

	xiScrollClass     = 3 // XIScrollClass
	xiValuatorClass   = 2 // XIValuatorClass
	xiScrollVertical  = 1 // XIScrollTypeVertical
	xiScrollHorizonal = 2 // XIScrollTypeHorizontal

	notifyGrab        = 1 // NotifyGrab
	notifyInferior    = 2 // NotifyInferior
	notifyPointer     = 5 // NotifyPointer
	notifyPointerRoot = 6
	notifyDetailNone  = 7

	// xiMaskLen is the length of an XI event mask: enough bits for the last event
	// type, 26.
	xiMaskLen = 4
)

// scrollKey identifies an axis of a device by its valuator number.
type scrollKey struct {
	device int32
	number int32
}

// scrollAxis is an axis that scrolls: which direction, how many valuator units
// make one notch, and the last value seen (the deltas are differences).
type scrollAxis struct {
	axis      input.Axis
	increment float64
	last      float64
}

// inputState is the XInput2 input of the platform: keyboard, pointer, scroll
// wheels and touch screens, for all the windows.
type inputState struct {
	p      *Platform
	xi     *xi
	opcode int
	// hasTouch is whether the server speaks XInput 2.2, which has touch.
	hasTouch bool

	// The keyboard; layout is nil when the keymap could not be read.
	xkbCtx *xkb.Context
	layout *xkb.Layout
	keys   input.KeyTracker
	// focus is the window with the keyboard focus.
	focus *window

	pointer *input.PointerBatcher
	touch   *input.TouchBatcher
	// pointerWindow is the window the pointer is in.
	pointerWindow *window
	lastX, lastY  float64
	// scroll holds the axes that scroll, by device and valuator number; devices
	// is the devices that have any, whose button 4 to 7 events are emulated.
	scroll map[scrollKey]*scrollAxis
}

// newInputState sets up XInput2 and the keyboard. It returns nil, after logging
// why, when there is no input: libXi missing, or no XInput2 on the server.
func newInputState(p *Platform) *inputState {
	x, err := loadXiOnce()
	if err != nil {
		slog.Warn("XInput2 is not available: there is no keyboard and pointer input", "error", err)
		return nil
	}
	name := dl.CString("XInputExtension")
	defer dl.Free(name)
	mem := dl.Calloc(3, 4)
	defer dl.Free(mem)
	if p.x.queryExtension(p.display, name, mem, mem+4, mem+8) == 0 {
		slog.Warn("the X server has no XInput extension: there is no keyboard and pointer input")
		return nil
	}
	in := &inputState{p: p, xi: x, opcode: int(*view[int32](mem)), scroll: map[scrollKey]*scrollAxis{}}

	// Ask for version 2.2 (touch); the server answers with what it has.
	ver := dl.Calloc(2, 4)
	defer dl.Free(ver)
	*view[int32](ver), *view[int32](ver + 4) = xiQueryVersionMajor, xiQueryVersionMinorWant
	if x.queryVersion(p.display, ver, ver+4) != 0 || *view[int32](ver) < 2 {
		slog.Warn("the X server has no XInput 2: there is no keyboard and pointer input")
		return nil
	}
	in.hasTouch = *view[int32](ver + 4) >= 2

	in.pointer = input.NewPointerBatcher(p.now)
	in.touch = input.NewTouchBatcher(p.now)
	in.setUpKeyboard()
	in.refreshDevices()
	// Learn when the devices change, so the scroll axes stay right.
	in.selectEvents(p.root, xiDeviceChanged)
	in.selectEventsFrom(xiAllDevices, p.root, xiHierarchyChanged) // only for all devices

	return in
}

// setUpKeyboard reads the keymap of the X server's core keyboard. Without it the
// pointer still works and the keys are ignored.
func (in *inputState) setUpKeyboard() {
	p := in.p
	ctx, err := xkb.NewContext()
	if err != nil {
		slog.Warn("the keyboard cannot be used", "error", err)
		return
	}
	conn := p.x.getXCBConnection(p.display)
	keymap, device, err := ctx.KeymapFromX11(conn)
	if err != nil {
		slog.Warn("the keymap of the X server cannot be read: keys are ignored", "error", err)
		ctx.Close()
		return
	}
	state, err := keymap.NewStateFromX11(conn, device)
	if err != nil {
		slog.Warn("the keyboard state cannot be read: keys are ignored", "error", err)
		keymap.Close()
		ctx.Close()
		return
	}
	// A locale without a compose table only loses dead keys and compose sequences.
	compose, err := ctx.NewCompose(xkb.LocaleFromEnv(os.Getenv))
	if err != nil {
		slog.Warn("no compose table: dead keys and compose sequences are off", "error", err)
		compose = nil
	}
	in.xkbCtx = ctx
	in.layout = xkb.NewLayoutWithState(keymap, state, compose)
}

// xiMask returns an XI event mask with the bits of the event types set.
func xiMask(types ...int) []byte {
	mask := make([]byte, xiMaskLen)
	for _, t := range types {
		mask[t>>3] |= 1 << (t & 7)
	}
	return mask
}

// selectEvents asks for XInput2 events of the types on the window, from every
// master device.
func (in *inputState) selectEvents(window uint64, types ...int) {
	in.selectEventsFrom(xiAllMasterDevices, window, types...)
}

// selectEventsFrom asks for XInput2 events of the types on the window, from the
// device (or xiAllMasterDevices, or 0 for all of them).
func (in *inputState) selectEventsFrom(device int32, window uint64, types ...int) {
	mask := xiMask(types...)
	maskMem := dl.Calloc(1, uintptr(len(mask)))
	defer dl.Free(maskMem)
	copy(dl.Bytes(maskMem, len(mask)), mask)
	em := dl.Calloc(1, unsafe.Sizeof(xiEventMask{}))
	defer dl.Free(em)
	*view[xiEventMask](em) = xiEventMask{deviceID: device, maskLen: int32(len(mask)), mask: maskMem}
	in.xi.selectEvents(in.p.display, window, em, 1)
}

// selectWindowEvents asks for the keyboard, pointer and touch events of a window.
func (in *inputState) selectWindowEvents(window uint64) {
	types := []int{xiKeyPress, xiKeyRelease, xiButtonPress, xiButtonRelease, xiMotion, xiEnter, xiLeave, xiFocusIn, xiFocusOut}
	if in.hasTouch {
		types = append(types, xiTouchBegin, xiTouchUpdate, xiTouchEnd)
	}
	in.selectEvents(window, types...)
}

// refreshDevices reads the scroll axes of the master devices. A device has scroll
// axes when its classes say so; the valuator that carries the scrolling has a
// number, and the unit per notch.
func (in *inputState) refreshDevices() {
	in.scroll = map[scrollKey]*scrollAxis{}
	count := dl.Calloc(1, 4)
	defer dl.Free(count)
	list := in.xi.queryDevice(in.p.display, xiAllMasterDevices, count)
	if list == 0 {
		return
	}
	defer in.xi.freeInfo(list)
	n := int(*view[int32](count))
	devices := unsafe.Slice(view[xiDeviceInfo](list), n)
	for _, dev := range devices {
		classes := unsafe.Slice((*uintptr)(dl.Pointer(dev.classes)), int(dev.numClasses))
		current := map[int32]float64{} // the value of each valuator now
		for _, c := range classes {
			if view[xiAnyClassInfo](c).typ == xiValuatorClass {
				v := view[xiValuatorClassInfo](c)
				current[v.number] = v.value
			}
		}
		for _, c := range classes {
			if view[xiAnyClassInfo](c).typ != xiScrollClass {
				continue
			}
			sc := view[xiScrollClassInfo](c)
			if sc.increment == 0 {
				continue
			}
			axis := input.AxisVertical
			if sc.scrollType == xiScrollHorizonal {
				axis = input.AxisHorizontal
			}
			in.scroll[scrollKey{dev.deviceID, sc.number}] = &scrollAxis{axis: axis, increment: sc.increment, last: current[sc.number]}
		}
	}
}

// hasScroll reports whether the device has axes that scroll.
func (in *inputState) hasScroll(device int32) bool {
	for k := range in.scroll {
		if k.device == device {
			return true
		}
	}
	return false
}

// close releases the keymap.
func (in *inputState) close() {
	if in.layout != nil {
		in.layout.Close()
		in.layout = nil
	}
	if in.xkbCtx != nil {
		in.xkbCtx.Close()
		in.xkbCtx = nil
	}
}

// handle handles the XInput2 event of the type, whose data is at data.
func (in *inputState) handle(evtype int, data uintptr) {
	switch evtype {
	case xiKeyPress, xiKeyRelease:
		ev := view[xiDeviceEvent](data)
		if w := in.p.windows[ev.event]; w != nil {
			in.onKey(ev, evtype == xiKeyPress)
		}
	case xiButtonPress, xiButtonRelease:
		ev := view[xiDeviceEvent](data)
		if w := in.p.windows[ev.event]; w != nil {
			in.onButton(w, ev, evtype == xiButtonPress)
		}
	case xiMotion:
		ev := view[xiDeviceEvent](data)
		if w := in.p.windows[ev.event]; w != nil {
			in.onMotion(w, ev)
		}
	case xiEnter, xiLeave:
		ev := view[xiEnterEvent](data)
		if w := in.p.windows[ev.event]; w != nil {
			in.onEnterLeave(w, ev, evtype == xiEnter)
		}
	case xiFocusIn, xiFocusOut:
		ev := view[xiEnterEvent](data)
		if w := in.p.windows[ev.event]; w != nil {
			in.onFocus(w, ev, evtype == xiFocusIn)
		}
	case xiTouchBegin, xiTouchUpdate, xiTouchEnd:
		ev := view[xiDeviceEvent](data)
		if w := in.p.windows[ev.event]; w != nil {
			in.onTouch(w, ev, evtype)
		}
	case xiDeviceChanged, xiHierarchyChanged:
		in.refreshDevices()
	}
}

// ---- keyboard ----

// evdevOffset turns an XKB keycode into an evdev scancode.
const evdevOffset = 8

func (in *inputState) onKey(ev *xiDeviceEvent, press bool) {
	if in.layout == nil || ev.detail < evdevOffset {
		return
	}
	if t := in.p.text; t != nil && t.bridge.Active() {
		in.onKeyViaIME(ev, press)
		return
	}
	in.dispatchKey(ev, press)
}

// dispatchKey sends a key event to the app.
func (in *inputState) dispatchKey(ev *xiDeviceEvent, press bool) {
	if in.layout == nil || ev.detail < evdevOffset {
		return
	}
	code := uint32(ev.detail - evdevOffset)
	// The event carries the modifiers and the layout group that apply to it.
	in.layout.UpdateMaskGroups(uint32(ev.mods.base), uint32(ev.mods.latched), uint32(ev.mods.locked),
		uint32(ev.group.base), uint32(ev.group.latched), uint32(ev.group.locked))
	now := in.p.now()

	if !press {
		if up, ok := in.keys.Up(code, now); ok {
			in.deliverKey(up)
		}
		return
	}
	if ev.flags&xiKeyRepeat != 0 {
		if down, ok := in.keys.Data(code); ok {
			// The server repeats a key: send the key-down data again.
			down.TimeStamp, down.Type = now, ui.KeyEventTypeRepeat
			in.deliverKey(down)
			return
		}
		// A repeat for a key that went down before we saw it is its first event.
	}
	key, err := in.layout.Press(code)
	if err != nil {
		in.p.fail(err)
		return
	}
	data := ui.KeyData{
		TimeStamp: now, Type: ui.KeyEventTypeDown, DeviceType: ui.KeyEventDeviceTypeKeyboard,
		Physical: key.Physical, Logical: key.Logical, Character: key.Character,
	}
	if in.keys.Down(code, data) {
		in.deliverKey(data)
	}
}

func (in *inputState) deliverKey(d ui.KeyData) {
	if h := in.p.handler; h != nil {
		h.OnKey(d)
	}
}

// onFocus follows the keyboard focus. When the window loses it, the keys that
// are held are released (synthesized), since their key-up will not come.
func (in *inputState) onFocus(w *window, ev *xiEnterEvent, in2 bool) {
	// With no window manager the focus is PointerRoot: it follows the pointer, and
	// the window under it gets the focus events with the detail NotifyPointer,
	// which are real. The events for a child window, or for the focus being set
	// to PointerRoot or None, are not about this window.
	switch ev.detail {
	case notifyInferior, notifyPointerRoot, notifyDetailNone:
		return
	}
	if in2 {
		in.focus = w
		if t := in.p.text; t != nil {
			t.focusChanged(w, true)
		}
		return
	}
	if in.focus == w {
		in.focus = nil
		in.releaseKeys()
		if t := in.p.text; t != nil {
			t.focusChanged(w, false)
		}
	}
}

func (in *inputState) releaseKeys() {
	for _, up := range in.keys.ReleaseAll(in.p.now()) {
		in.deliverKey(up)
	}
}

// ---- pointer ----

// linuxButton maps an X button number to a Linux button code, and 0 for the
// buttons that are not buttons (the scroll wheel is 4 to 7).
func linuxButton(detail int32) uint32 {
	switch detail {
	case 1:
		return 0x110 // BTN_LEFT
	case 2:
		return 0x112 // BTN_MIDDLE
	case 3:
		return 0x111 // BTN_RIGHT
	case 8:
		return 0x116 // BTN_BACK
	case 9:
		return 0x115 // BTN_FORWARD
	}
	return 0
}

// wheelButton returns the axis and direction of a legacy scroll button (4 up, 5
// down, 6 left, 7 right), and false for any other button.
func wheelButton(detail int32) (axis input.Axis, direction int32, ok bool) {
	switch detail {
	case 4:
		return input.AxisVertical, -1, true
	case 5:
		return input.AxisVertical, 1, true
	case 6:
		return input.AxisHorizontal, -1, true
	case 7:
		return input.AxisHorizontal, 1, true
	}
	return 0, 0, false
}

func (in *inputState) flushPointer() {
	if batch := in.pointer.Frame(); len(batch) > 0 {
		in.deliverPointers(batch)
	}
}

func (in *inputState) deliverPointers(batch []ui.PointerData) {
	if h := in.p.handler; h != nil {
		h.OnPointer(batch)
	}
}

// moveTo reports the pointer at a new position, if it moved.
func (in *inputState) moveTo(x, y float64) {
	if x != in.lastX || y != in.lastY {
		in.lastX, in.lastY = x, y
		in.pointer.Motion(x, y)
	}
}

func (in *inputState) onEnterLeave(w *window, ev *xiEnterEvent, enter bool) {
	// Grabs (a button held while the pointer moves out) make enter and leave
	// events that are not the pointer going in or out; so do child windows.
	if ev.mode == notifyGrab || ev.detail == notifyInferior {
		return
	}
	if enter {
		in.pointerWindow = w
		in.lastX, in.lastY = ev.eventX, ev.eventY
		in.pointer.Enter(w.id, ev.eventX, ev.eventY)
	} else {
		if in.pointerWindow != w {
			return
		}
		in.pointerWindow = nil
		in.pointer.Leave()
	}
	in.flushPointer()
}

func (in *inputState) onMotion(w *window, ev *xiDeviceEvent) {
	if in.pointerWindow != w {
		return
	}
	in.moveTo(ev.eventX, ev.eventY)
	in.scrollValuators(ev)
	in.flushPointer()
}

// scrollValuators turns the changes of the scroll axes in an event into scroll
// input: the difference to the last value, in notches, times the pixels of a
// notch.
func (in *inputState) scrollValuators(ev *xiDeviceEvent) {
	v := &ev.valuators
	if v.maskLen == 0 || v.mask == 0 {
		return
	}
	mask := dl.Bytes(v.mask, int(v.maskLen))
	values := unsafe.Slice(view[float64](v.values), int(v.maskLen)*8)
	index := 0
	for number := range int(v.maskLen) * 8 {
		if mask[number>>3]&(1<<(number&7)) == 0 {
			continue
		}
		value := values[index]
		index++
		axis, ok := in.scroll[scrollKey{ev.deviceID, int32(number)}]
		if !ok {
			continue
		}
		delta := (value - axis.last) / axis.increment
		axis.last = value
		if delta != 0 {
			in.pointer.AxisSource(input.AxisSourceContinuous)
			in.pointer.Axis(axis.axis, delta*scrollStepPixels)
		}
	}
}

// scrollStepPixels is how far one notch of a scroll axis scrolls: the same 53
// logical pixels as one notch of a wheel on Wayland.
const scrollStepPixels = 53.0

func (in *inputState) onButton(w *window, ev *xiDeviceEvent, press bool) {
	if in.pointerWindow != w {
		return
	}
	if axis, dir, ok := wheelButton(ev.detail); ok {
		// With scroll axes, the buttons are an emulation of them; without, they
		// are the only scrolling there is, a notch per press.
		if ev.flags&xiPointerEmulated == 0 && !in.hasScroll(ev.deviceID) && press {
			in.pointer.AxisSource(input.AxisSourceWheel)
			in.pointer.AxisValue120(axis, dir*120)
			in.flushPointer()
		}
		return
	}
	code := linuxButton(ev.detail)
	if code == 0 {
		return
	}
	in.moveTo(ev.eventX, ev.eventY)
	in.pointer.Button(code, press)
	in.flushPointer()
}

// ---- touch ----

func (in *inputState) onTouch(w *window, ev *xiDeviceEvent, evtype int) {
	id := ev.detail
	switch evtype {
	case xiTouchBegin:
		in.touch.Down(w.id, id, ev.eventX, ev.eventY)
	case xiTouchUpdate:
		in.touch.Motion(id, ev.eventX, ev.eventY)
	case xiTouchEnd:
		in.touch.Up(id)
	}
	if batch := in.touch.Take(); len(batch) > 0 {
		in.deliverPointers(batch)
	}
}

// windowClosed lets the input go of a window that is closing: held keys are
// released, the pointer leaves it and its fingers are cancelled.
func (in *inputState) windowClosed(w *window) {
	if in.focus == w {
		in.focus = nil
		in.releaseKeys()
	}
	if in.pointerWindow == w {
		in.pointerWindow = nil
		in.pointer.Leave()
		in.flushPointer()
	}
	in.touch.CancelView(w.id)
	if batch := in.touch.Take(); len(batch) > 0 {
		in.deliverPointers(batch)
	}
}
