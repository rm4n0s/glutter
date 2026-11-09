//go:build darwin && !ios

package macos

import (
	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the AppKit side of the keyboard (SPEC §29.6): the view's key methods, the redispatch of a key
// that nobody took, the layout of the keyboard through Carbon's text input sources, and the monitor that
// finds the key ups that AppKit does not send while Command is down. The conversion of the events is in
// keyboard.go and layout.go.

// The flags and types of NSEvent that the keyboard reads.
const (
	flagCommand uint64 = 1 << 20

	eventMaskKeyUp uint64 = 1 << 11

	// carbonShiftKey is shiftKey of Carbon's event modifiers; UCKeyTranslate wants it shifted right by 8.
	carbonShiftKey uint32 = 1 << 9
	// ucKeyActionDown is kUCKeyActionDown.
	ucKeyActionDown uint16 = 0
	// cfNotificationSuspensionDeliverImmediately is CFNotificationSuspensionBehaviorDeliverImmediately.
	cfNotificationSuspensionDeliverImmediately int64 = 4
)

// kbFuncs are the C functions of the keyboard layout: the text input sources and UCKeyTranslate of Carbon,
// the bytes of a CFData, and the distributed notification center that tells when the user changes the
// input source.
type kbFuncs struct {
	copyInputSource       func() uintptr
	copyLayoutInputSource func() uintptr
	inputSourceProperty   func(source, key uintptr) uintptr
	ucKeyTranslate        func(layout uintptr, keyCode, action uint16, modifiers, keyboardType, options uint32,
		deadKeyState *uint32, maxLength uint64, actualLength *uint64, out *uint16) int32
	// keyboardType is LMGetKbdLast, which Apple deprecated and kept; 0 when the system lacks it.
	keyboardType func() uint8
	dataBytePtr  func(data uintptr) uintptr

	distributedCenter func() uintptr
	addObserver       func(center, observer, callback, name, object uintptr, behavior int64)
	removeObservers   func(center, observer uintptr)

	// layoutDataKey is kTISPropertyUnicodeKeyLayoutData and inputSourceChanged is
	// kTISNotifySelectedKeyboardInputSourceChanged, both CFStrings.
	layoutDataKey, inputSourceChanged uintptr
	// onChanged is the callback of the notification, made once; the observer is the platform's handle.
	onChanged uintptr
}

func (f *kbFuncs) bind() error {
	carbon, err := dl.Open(dl.FrameworkPath("Carbon"), dl.FrameworkPath("Carbon")+"/Frameworks/HIToolbox.framework/HIToolbox")
	if err != nil {
		return err
	}
	cfw, err := dl.OpenFramework("CoreFoundation")
	if err != nil {
		return err
	}
	for _, b := range []struct {
		lib    *dl.Lib
		fptr   any
		symbol string
	}{
		{carbon, &f.copyInputSource, "TISCopyCurrentKeyboardInputSource"},
		{carbon, &f.copyLayoutInputSource, "TISCopyCurrentKeyboardLayoutInputSource"},
		{carbon, &f.inputSourceProperty, "TISGetInputSourceProperty"},
		{carbon, &f.ucKeyTranslate, "UCKeyTranslate"},
		{cfw, &f.dataBytePtr, "CFDataGetBytePtr"},
		{cfw, &f.distributedCenter, "CFNotificationCenterGetDistributedCenter"},
		{cfw, &f.addObserver, "CFNotificationCenterAddObserver"},
		{cfw, &f.removeObservers, "CFNotificationCenterRemoveEveryObserver"},
	} {
		if err := b.lib.Bind(b.fptr, b.symbol); err != nil {
			return err
		}
	}
	if carbon.Has("LMGetKbdLast") {
		if err := carbon.Bind(&f.keyboardType, "LMGetKbdLast"); err != nil {
			return err
		}
	}
	for _, c := range []struct {
		dst    *uintptr
		symbol string
	}{{&f.layoutDataKey, "kTISPropertyUnicodeKeyLayoutData"}, {&f.inputSourceChanged, "kTISNotifySelectedKeyboardInputSourceChanged"}} {
		addr, err := carbon.Symbol(c.symbol)
		if err != nil {
			return err
		}
		*c.dst = *(*uintptr)(dl.Pointer(addr)) // the CFString that the exported variable holds
	}
	f.onChanged = dl.Callback(func(_, observer, _, _, _ uintptr) {
		if p, ok := objcx.LookupAs[*Platform](objcx.Handle(observer)); ok {
			p.guard(p.kb.layoutChanged)
		}
	})
	return nil
}

// keyboard is the keyboard of the platform: the responder, the layout of the keyboard and what the
// redispatch needs.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterKeyboardManager.mm FlutterKeyboardManager
type keyboard struct {
	p    *Platform
	resp *keyResponder

	// layoutData is the CFData of the current layout (UCKeyboardLayout), retained; 0 when it is not read.
	layoutData cf.Ref
	// dispatching is the event that is being given to the next responder: its key equivalent must go to the
	// menus, not to the view again (isDispatchingKeyEvent).
	dispatching objc.ID
	// upMonitor is the local monitor of the key ups that AppKit does not send while Command is down.
	upMonitor objc.ID
	upBlock   objc.Block
	observing bool

	// equivalent is the event that performKeyEquivalent: is handling: the input context must know that a key
	// came as a key equivalent (Flutter's isKeyEquivalent mark of the event).
	equivalent objc.ID
	// produced says that the input context made output (text or a command) for the event it is handling;
	// prevFlags are the modifier flags of the last event that went to it.
	produced  bool
	prevFlags uint64
}

// newKeyboard makes the keyboard of the platform: it reads the layout and listens to the input source.
func newKeyboard(p *Platform) *keyboard {
	k := &keyboard{p: p}
	k.resp = newKeyResponder(p.sendKey)
	k.rebuildLayout()
	f := &p.s.kb
	f.addObserver(f.distributedCenter(), uintptr(p.handle), f.onChanged, f.inputSourceChanged, 0, cfNotificationSuspensionDeliverImmediately)
	k.observing = true
	k.installUpMonitor()
	return k
}

// close stops the listening and lets go of the layout.
func (k *keyboard) close() {
	f := &k.p.s.kb
	if k.observing {
		f.removeObservers(f.distributedCenter(), uintptr(k.p.handle))
		k.observing = false
	}
	if k.upMonitor != 0 {
		class("NSEvent").Send(sel("removeMonitor:"), k.upMonitor)
		k.upMonitor = 0
		k.upBlock.Release()
	}
	k.dropLayout()
}

func (k *keyboard) dropLayout() {
	if k.layoutData != 0 {
		cf.Release(k.layoutData)
		k.layoutData = 0
	}
}

// layoutChanged is the notification that the user selected another input source: the layout is read again.
//
// Flutter: FlutterKeyboardLayout.mm onKeyboardLayoutChanged
func (k *keyboard) layoutChanged() { k.rebuildLayout() }

// rebuildLayout reads the layout of the keyboard and makes the logical keys of the typewriter keys from it
// (buildLayout).
//
// Flutter: FlutterKeyboardManager.mm buildLayout
func (k *keyboard) rebuildLayout() {
	k.dropLayout()
	k.readLayoutData()
	k.resp.layout = buildLayout(layoutGoals, k.clueFor)
}

// readLayoutData reads the Unicode layout data of the current input source. The input source of an input
// method (Japanese) has none, so the layout input source is asked then.
//
// Flutter: FlutterKeyboardLayout.mm CurrentKeyboardLayoutData
func (k *keyboard) readLayoutData() {
	f := &k.p.s.kb
	for _, copySource := range []func() uintptr{f.copyInputSource, f.copyLayoutInputSource} {
		source := copySource()
		if source == 0 {
			continue
		}
		data := f.inputSourceProperty(source, f.layoutDataKey)
		if data != 0 {
			k.layoutData = cf.Retain(cf.Ref(data))
		}
		cf.Release(cf.Ref(source))
		if k.layoutData != 0 {
			return
		}
	}
}

// clueFor is the character that a key makes in the current layout, without or with Shift: a dead key is
// pressed twice to get the character that stands for it, and a control character or a string of other
// than one UTF-16 unit is no clue.
//
// Flutter: FlutterKeyboardLayout.mm lookUpLayoutForKeyCode:shift:
func (k *keyboard) clueFor(keyCode uint16, shift bool) layoutClue {
	if k.layoutData == 0 {
		return layoutClue{}
	}
	f := &k.p.s.kb
	layout := f.dataBytePtr(uintptr(k.layoutData))
	var modifiers uint32
	if shift {
		modifiers = (carbonShiftKey >> 8) & 0xff
	}
	var kbType uint32
	if f.keyboardType != nil {
		kbType = uint32(f.keyboardType())
	}
	var dead uint32
	var n uint64
	var out uint16
	isDead := false
	status := f.ucKeyTranslate(layout, keyCode, ucKeyActionDown, modifiers, kbType, 0, &dead, 1, &n, &out)
	if status == 0 && n == 0 && dead != 0 {
		isDead = true
		status = f.ucKeyTranslate(layout, keyCode, ucKeyActionDown, modifiers, kbType, 0, &dead, 1, &n, &out)
	}
	if status == 0 && n == 1 && !isControl(out) {
		return layoutClue{character: uint32(out), dead: isDead}
	}
	return layoutClue{}
}

// isControl is std::iscntrl in the C locale.
func isControl(c uint16) bool { return c < 0x20 || c == 0x7f }

// sendKey gives an event to the handler and returns whether the framework handled it. The synthesized
// events and the key events of a handler that cannot answer at once are not waited for.
func (p *Platform) sendKey(d ui.KeyData) bool {
	h := p.handler
	if h == nil {
		return false
	}
	if sync, ok := h.(platform.SyncInputHandler); ok {
		return sync.OnKeySync(d)
	}
	h.OnKey(d)
	return false
}

// keyMethods are the key methods of GlutterView.
func keyMethods() []objcx.Method {
	keyMethod := func(name string, kind int) objcx.Method {
		return objcx.Method{Name: name, Fn: func(self objc.ID, _ objc.SEL, ev objc.ID) {
			if w := windowOf(self); w != nil && !w.closed {
				w.p.guard(func() { w.keyEvent(ev, kind, true) })
			}
		}}
	}
	return []objcx.Method{
		keyMethod("keyDown:", keyKindDown),
		keyMethod("keyUp:", keyKindUp),
		keyMethod("flagsChanged:", keyKindFlagsChanged),
	}
}

// keyEventOf reads an NSEvent of a key. Only a key down and a key up have characters and a repeat flag;
// asking a flags-changed event for them raises an exception.
func (w *window) keyEventOf(ev objc.ID, kind int) keyEvent {
	ke := keyEvent{
		kind:          kind,
		keyCode:       send[uint16](ev, "keyCode"),
		modifierFlags: send[uint64](ev, "modifierFlags"),
		timestamp:     w.p.frameTime(send[float64](ev, "timestamp")),
	}
	if kind != keyKindFlagsChanged {
		ke.isARepeat = send[bool](ev, "isARepeat")
		ke.characters = nsString(send[objc.ID](ev, "characters"))
		ke.ignoringModifiers = nsString(send[objc.ID](ev, "charactersIgnoringModifiers"))
	}
	return ke
}

// nsString is the Go string of an NSString that may be nil.
func nsString(id objc.ID) string {
	if id == 0 {
		return ""
	}
	return objcx.String(id)
}

// keyEvent handles a key event of the view: the responder converts it and the framework answers; a key that
// the framework did not handle goes to the next responder, where the input method and the menus take it.
//
// Flutter: FlutterKeyboardManager.mm handleEvent, performProcessEvent, dispatchTextEvent
func (w *window) keyEvent(ev objc.ID, kind int, redispatch bool) {
	k := w.p.kb
	// While an input method composes, it has the first look at the keys (Backspace edits the composition,
	// Enter and Escape end it, the arrows move inside it): the shortcuts of the framework see them when it
	// does not take them. Without a composition the framework goes first, as Flutter's does.
	if redispatch && kind == keyKindDown && w.text.hasMarked() && k.textKey(w, ev, kind) {
		return
	}
	ke := w.keyEventOf(ev, kind)
	if w.text.enabled && kind == keyKindDown {
		// A text field takes its text from the input context (the key may be a dead key, or a start of a
		// composition), not from the character of the key, which the framework would type itself.
		ke.characters = ""
	}
	if k.resp.handle(ke) {
		return
	}
	if redispatch {
		k.dispatchText(w, ev, kind)
	}
}

// dispatchText gives a key event that the framework did not handle to the next responder, which for a text
// field is the text input (step 33.5) and otherwise the window and its menus. While it is given, a key
// equivalent of the same event is not taken by the view again.
//
// Flutter: FlutterKeyboardManager.mm dispatchTextEvent
func (k *keyboard) dispatchText(w *window, ev objc.ID, kind int) {
	if k.textKey(w, ev, kind) {
		return
	}
	next := send[objc.ID](w.view, "nextResponder")
	if next == 0 || k.dispatching != 0 {
		return
	}
	name := [...]string{keyKindDown: "keyDown:", keyKindUp: "keyUp:", keyKindFlagsChanged: "flagsChanged:"}[kind]
	if !send[bool](next, "respondsToSelector:", sel(name)) {
		return
	}
	k.dispatching = ev
	defer func() { k.dispatching = 0 }()
	next.Send(sel(name), ev)
}

// performKeyEquivalent is the view's answer to a key equivalent (Command+key, and every key that AppKit
// offers to the views before it keys): when the view is the first responder it takes the event as a key
// down, and when it is not, or the event is the one that is being redispatched, the menus get it.
//
// Flutter: FlutterViewController.mm performKeyEquivalent:
func (w *window) performKeyEquivalent(self, ev objc.ID) bool {
	k := w.p.kb
	if send[objc.ID](w.ns, "firstResponder") != w.view || k.dispatching == ev {
		return objc.SendSuper[bool](self, sel("performKeyEquivalent:"), ev)
	}
	k.equivalent = ev
	defer func() { k.equivalent = 0 }()
	w.keyEvent(ev, keyKindDown, true)
	return true
}

// installUpMonitor listens for the key ups that AppKit does not send to the view while Command is down (it
// consumes them as key equivalents), so that the framework does not keep the key pressed. It never changes
// the event.
//
// Flutter: FlutterViewController.mm listenForMetaModifiedKeyUpEvents
func (k *keyboard) installUpMonitor() {
	k.upBlock = objcx.NewBlock(func(_ objc.Block, ev objc.ID) objc.ID {
		k.p.guard(func() { k.monitoredKeyUp(ev) })
		return ev
	})
	k.upMonitor = objcx.Retain(class("NSEvent").Send(sel("addLocalMonitorForEventsMatchingMask:handler:"), eventMaskKeyUp, k.upBlock))
}

func (k *keyboard) monitoredKeyUp(ev objc.ID) {
	if send[uint64](ev, "modifierFlags")&flagCommand == 0 {
		return
	}
	win := send[objc.ID](ev, "window")
	if win == 0 {
		return
	}
	first := send[objc.ID](win, "firstResponder")
	for _, w := range k.p.sortedWindows() {
		if w.view == first && !w.closed {
			w.keyEvent(ev, keyKindUp, false)
			return
		}
	}
}

// syncModifiers gives the framework the modifier keys that changed without an event of this window: the
// pointer events carry the flags of the moment.
//
// Flutter: FlutterKeyboardManager.mm syncModifiersIfNeeded:timestamp:
func (k *keyboard) syncModifiers(w *window, ev objc.ID) {
	k.resp.syncModifiers(send[uint64](ev, "modifierFlags"), w.p.frameTime(send[float64](ev, "timestamp")))
}
