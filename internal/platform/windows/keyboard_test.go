package windows

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests are ported from the unit tests of Flutter's Windows embedder
// (shell/platform/windows/keyboard_unittests.cc, keyboard_key_embedder_handler_unittests.cc and
// keyboard_key_handler_unittests.cc) and run the converter and the manager over a fake Windows:
// the table of key states, the layout and the queue of character messages are the test's.
//
// Not ported: the cases of the redispatching (SlowFrameworkResponse*, RestartClearsKeyboardState,
// the channel handler), which Flutter needs because the framework answers later and which do not
// exist here (the framework answers at once), the slow-response Vietnamese Telex cases for the same
// reason. The Telex case with the fast response and the end of the text input submit case are in
// textinput_test.go (step 28.5), with the text input.

// The ids of the keys that the tests expect.
const (
	physKeyA         = 0x70004
	physKeyE         = 0x70008
	physKeyQ         = 0x70014
	physKeyY         = 0x7001c
	physKeyZ         = 0x7001d
	physDigit1       = 0x7001e
	physBracketLeft  = 0x7002f
	physArrowLeft    = 0x70050
	physNumpad1      = 0x70059
	physNumpad2      = 0x7005a
	physCapsLock     = 0x70039
	physNumLock      = 0x70053
	physShiftLeft    = 0x700e1
	physShiftRight   = 0x700e5
	physControlLeft  = 0x700e0
	physControlRight = 0x700e4
	physAltLeft      = 0x700e2
	physAltRight     = 0x700e6
	physF4           = 0x7003d

	logKeyA         = 0x61
	logKeyE         = 0x65
	logKeyQ         = 0x71
	logKeyY         = 0x79
	logKeyZ         = 0x7a
	logDigit1       = 0x31
	logBracketRight = 0x5d
	logArrowLeft    = 0x100000302
	logNumpad1      = 0x200000231
	logNumpad2      = 0x200000232
	logCapsLock     = 0x100000104
	logNumLock      = 0x10000010a
	logShiftLeft    = 0x200000102
	logShiftRight   = 0x200000103
	logControlLeft  = 0x200000100
	logControlRight = 0x200000101
	logAltLeft      = 0x200000104
	logAltRight     = 0x200000105
	logF4           = 0x100000804
)

// The scan codes of the keys.
const (
	scanKeyA         = 0x1e
	scanKeyE         = 0x12
	scanKeyQ         = 0x10
	scanKeyY         = 0x15
	scanKeyZ         = 0x2c
	scanDigit1       = 0x02
	scanBracketLeft  = 0x1a
	scanArrowLeft    = 0x4b // extended
	scanNumpad1      = 0x4f
	scanNumpad2      = 0x50
	scanCapsLock     = 0x3a
	scanNumLock      = 0x45 // extended
	scanLShift       = 0x2a
	scanRShift       = 0x36
	scanLControl     = 0x1d
	scanLAlt         = 0x38
	scanF4           = 0x3e
	extended         = true
	notExtended      = false
	wasUp, wasDown   = false, true
	vkE              = 'E'
	vkQ              = 'Q'
	vkA              = 'A'
	vkY              = 'Y'
	vkZ              = 'Z'
	vk1              = '1'
	vkBracketRightUS = 0xdd // VK_OEM_6, the key of "^" on a French layout
)

// fakeKeyHost is Windows for the keyboard tests.
type fakeKeyHost struct {
	state  map[uint32]int16
	layout uintptr
	// chars is the character of a key at its first level on the layout, by scan code.
	chars map[uint32]uint32
	// vkOfScan is MAPVK_VSC_TO_VK_EX; the shifts have it by default.
	vkOfScan map[uint32]uint32
}

func newFakeKeyHost() *fakeKeyHost {
	return &fakeKeyHost{
		state: map[uint32]int16{}, layout: 1, chars: map[uint32]uint32{},
		vkOfScan: map[uint32]uint32{scanLShift: win32.VKLShift, scanRShift: win32.VKRShift},
	}
}

func (h *fakeKeyHost) KeyState(vk uint32) int16 { return h.state[vk] }

// ScanCode is MapVirtualKey for the virtual keys the embedder asks for at the start.
func (h *fakeKeyHost) ScanCode(vk uint32, extended bool) uint32 {
	scan := map[uint32]uint32{
		win32.VKLShift: 0x2a, win32.VKRShift: 0x36, win32.VKLControl: 0x1d, win32.VKRControl: 0xe01d,
		win32.VKLMenu: 0x38, win32.VKRMenu: 0xe038, win32.VKLWin: 0xe05b, win32.VKRWin: 0xe05c,
		win32.VKCapital: 0x3a, win32.VKScroll: 0x46, win32.VKNumLock: 0xe045,
	}[vk]
	if !extended {
		scan &= 0xff
	}
	return scan
}
func (h *fakeKeyHost) VirtualKeyOfScan(scan uint32) uint32 { return h.vkOfScan[scan] }
func (h *fakeKeyHost) Layout() uintptr                     { return h.layout }
func (h *fakeKeyHost) CharOfKey(scan uint32) uint32        { return h.chars[scan] }

// set puts keys down (the high bit) or up in the table of key states; the lock bit stays.
func (h *fakeKeyHost) set(down bool, vks ...uint32) {
	for _, vk := range vks {
		s := h.state[vk] &^ stateMaskPressed
		if down {
			s |= stateMaskPressed - 0x100 // the int16 with only the high bit set
		}
		h.state[vk] = s
	}
}

// toggle sets the lock bit.
func (h *fakeKeyHost) toggle(on bool, vk uint32) {
	if on {
		h.state[vk] |= stateMaskToggled
	} else {
		h.state[vk] &^= stateMaskToggled
	}
}

// fakeKeyQueue is the queue of key messages of a window.
type fakeKeyQueue struct{ msgs []keyMsg }

func (q *fakeKeyQueue) Peek() (keyMsg, bool) {
	if len(q.msgs) == 0 {
		return keyMsg{}, false
	}
	return q.msgs[0], true
}
func (q *fakeKeyQueue) Take() {
	if len(q.msgs) > 0 {
		q.msgs = q.msgs[1:]
	}
}

// keyTester drives a manager.
type keyTester struct {
	t    *testing.T
	host *fakeKeyHost
	q    *fakeKeyQueue
	m    *keyboardManager
	got  []ui.KeyData
	// handled is the framework's answer to the main events.
	handled bool
	clock   time.Duration
}

func newKeyTester(t *testing.T, setup func(*fakeKeyHost)) *keyTester {
	t.Helper()
	k := &keyTester{t: t, host: newFakeKeyHost(), q: &fakeKeyQueue{}}
	if setup != nil {
		setup(k.host)
	}
	k.m = newKeyboardManager(k.host, func() time.Duration { k.clock += time.Millisecond; return k.clock },
		func(d ui.KeyData) bool {
			k.got = append(k.got, d)
			return k.handled && !d.Synthesized
		})
	return k
}

// kmsg is a message of a step.
type kmsg struct {
	msg uint32
	// vk is the virtual key of a key message, the character of a character message.
	vk   uint32
	scan uint32 // the scan code, with 0xE000 for an extended key
	was  bool
	// time is the time of the message.
	time uint32
}

func (m kmsg) lparam() uintptr {
	l := uintptr(1) | uintptr(m.scan&0xff)<<16
	if m.scan&0xe000 != 0 {
		l |= 1 << 24
	}
	if m.was {
		l |= 1 << 30
	}
	if m.msg == win32.WMKeyUp || m.msg == win32.WMSysKeyUp {
		l |= 1<<30 | 1<<31
	}
	return l
}

func (m kmsg) queued() keyMsg {
	return keyMsg{Message: m.msg, WParam: uintptr(m.vk), LParam: m.lparam(), Time: m.time}
}

func chr(msg uint32, c rune) kmsg { return kmsg{msg: msg, vk: uint32(c)} }

// deliver handles a key message with the messages that follow it already in the queue.
func (k *keyTester) deliver(key kmsg, after ...kmsg) keyAnswer {
	k.t.Helper()
	k.q.msgs = nil
	for _, a := range after {
		k.q.msgs = append(k.q.msgs, a.queued())
	}
	return k.m.handleKey(k.q, key.msg, uintptr(key.vk), key.lparam(), key.time)
}

// we is a key data that a test expects: the type, the keys, the character, whether synthesized.
type we struct {
	typ   ui.KeyEventType
	phys  uint64
	log   uint64
	text  string
	synth bool
}

const (
	down   = ui.KeyEventTypeDown
	up     = ui.KeyEventTypeUp
	repeat = ui.KeyEventTypeRepeat
)

func (k *keyTester) expect(want ...we) {
	k.t.Helper()
	var got []we
	for _, d := range k.got {
		got = append(got, we{d.Type, d.Physical, d.Logical, d.Character, d.Synthesized})
	}
	k.got = nil
	if !slices.Equal(got, want) {
		k.t.Errorf("key data:\n got  %s\n want %s", showWe(got), showWe(want))
	}
}

func showWe(w []we) string {
	s := ""
	for _, e := range w {
		s += fmt.Sprintf("\n    {%s phys %#x log %#x %q synth %v}", e.typ.Label(), e.phys, e.log, e.text, e.synth)
	}
	return s
}

func TestKeyboardSingleKeys(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*fakeKeyHost)
		handled bool
		// steps run in order; each checks its events and answer.
		steps []kstep
	}{
		{"a plain letter, handled", nil, true, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 'a')},
				want: []we{{down, physKeyA, logKeyA, "a", false}}, answer: keyConsumed, taken: 1},
			{key: kmsg{msg: win32.WMKeyUp, vk: vkA, scan: scanKeyA},
				want: []we{{up, physKeyA, logKeyA, "", false}}, answer: keyConsumed},
		}},
		{"a plain letter, unhandled goes to the system", nil, false, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 'a')},
				want: []we{{down, physKeyA, logKeyA, "a", false}}, answer: keyToSystem, taken: 1},
			{key: kmsg{msg: win32.WMKeyUp, vk: vkA, scan: scanKeyA},
				want: []we{{up, physKeyA, logKeyA, "", false}}, answer: keyToSystem},
		}},
		{"an arrow key is extended and has no character", nil, true, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKLeft, scan: 0xe000 | scanArrowLeft},
				want: []we{{down, physArrowLeft, logArrowLeft, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyUp, vk: win32.VKLeft, scan: 0xe000 | scanArrowLeft},
				want: []we{{up, physArrowLeft, logArrowLeft, "", false}}, answer: keyConsumed},
		}},
		{"Control+A makes a control character, which is no text", nil, true, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 1)},
				want: []we{{down, physKeyA, logKeyA, "", false}}, answer: keyConsumed, taken: 1},
		}},
		{"a repeat keeps the logical key of the press", nil, true, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 'a')},
				want: []we{{down, physKeyA, logKeyA, "a", false}}, answer: keyConsumed, taken: 1},
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA, was: wasDown}, after: []kmsg{chr(win32.WMChar, 'a')},
				want: []we{{repeat, physKeyA, logKeyA, "a", false}}, answer: keyConsumed, taken: 1},
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA, was: wasDown}, after: []kmsg{chr(win32.WMChar, 'a')},
				want: []we{{repeat, physKeyA, logKeyA, "a", false}}, answer: keyConsumed, taken: 1},
			{key: kmsg{msg: win32.WMKeyUp, vk: vkA, scan: scanKeyA},
				want: []we{{up, physKeyA, logKeyA, "", false}}, answer: keyConsumed},
		}},
		{"a down after a down, bit 30 clear, is a press after a lost up", nil, true, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 'a')},
				want: []we{{down, physKeyA, logKeyA, "a", false}}, answer: keyConsumed, taken: 1},
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 'a')},
				want: []we{{up, physKeyA, logKeyA, "", true}, {down, physKeyA, logKeyA, "a", false}}, answer: keyConsumed, taken: 1},
		}},
		{"an up of a key that was not down is ignored and handled", nil, false, []kstep{
			{key: kmsg{msg: win32.WMKeyUp, vk: vkA, scan: scanKeyA},
				want: []we{{down, 0, 0, "", false}}, answer: keyConsumed},
		}},
		{"a key that an input method took makes nothing", nil, false, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKProcessKey, scan: scanKeyA},
				want: []we{{down, 0, 0, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyUp, vk: vkA, scan: scanKeyA},
				want: []we{{down, 0, 0, "", false}}, answer: keyConsumed},
		}},
		{"a character that is a surrogate pair is one character", nil, true, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA},
				after: []kmsg{{msg: win32.WMChar, vk: 0xD83D}, {msg: win32.WMChar, vk: 0xDE00}},
				want:  []we{{down, physKeyA, logKeyA, "😀", false}}, answer: keyConsumed, taken: 2},
		}},
		{"a lone high surrogate is no character", nil, true, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{{msg: win32.WMChar, vk: 0xD83D}},
				want: []we{{down, physKeyA, logKeyA, "", false}}, answer: keyConsumed, taken: 1},
		}},
		{"a letter after a dead key that does not combine has both characters", nil, true, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkQ, scan: scanKeyQ}, after: []kmsg{chr(win32.WMChar, '^'), chr(win32.WMChar, 'q')},
				want: []we{{down, physKeyQ, logKeyQ, "^q", false}}, answer: keyConsumed, taken: 2},
		}},
		{"VK_PACKET is left to the text input", nil, true, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKPacket}, want: nil, answer: keyToSystem},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := newKeyTester(t, tt.setup)
			k.handled = tt.handled
			runKeySteps(k, tt.steps)
		})
	}
}

// kstep is a key message of a test with what it must give.
type kstep struct {
	key   kmsg
	after []kmsg
	// setup runs before the message (key states, the layout).
	setup  func(*fakeKeyHost)
	want   []we
	answer keyAnswer
	// taken is how many of the messages after the key must have been taken off the queue.
	taken int
}

func runKeySteps(k *keyTester, steps []kstep) {
	k.t.Helper()
	for i, s := range steps {
		if s.setup != nil {
			s.setup(k.host)
		}
		got := k.deliver(s.key, s.after...)
		if got != s.answer {
			k.t.Errorf("step %d: answer %v, want %v", i, got, s.answer)
		}
		if left := len(k.q.msgs); left != len(s.after)-s.taken {
			k.t.Errorf("step %d: %d messages are left in the queue, want %d", i, left, len(s.after)-s.taken)
		}
		k.expect(s.want...)
	}
}

func TestKeyboardModifiersAndLocks(t *testing.T) {
	shiftDown := func(h *fakeKeyHost) { h.set(true, win32.VKLShift) }
	shiftUp := func(h *fakeKeyHost) { h.set(false, win32.VKLShift) }
	ctrlDown := func(h *fakeKeyHost) { h.set(true, win32.VKLControl) }
	capsOn := func(h *fakeKeyHost) { h.set(true, win32.VKCapital); h.toggle(true, win32.VKCapital) }
	capsReleased := func(h *fakeKeyHost) { h.set(false, win32.VKCapital) }
	capsOff := func(h *fakeKeyHost) { h.set(true, win32.VKCapital); h.toggle(false, win32.VKCapital) }
	numLockOff := func(h *fakeKeyHost) { h.toggle(false, win32.VKNumLock) }
	tests := []struct {
		name  string
		setup func(*fakeKeyHost)
		steps []kstep
	}{
		{"Shift then a letter", nil, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKShift, scan: scanLShift}, setup: shiftDown,
				want: []we{{down, physShiftLeft, logShiftLeft, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 'A')},
				want: []we{{down, physKeyA, logKeyA, "A", false}}, answer: keyConsumed, taken: 1},
			{key: kmsg{msg: win32.WMKeyUp, vk: vkA, scan: scanKeyA},
				want: []we{{up, physKeyA, logKeyA, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyUp, vk: win32.VKShift, scan: scanLShift}, setup: shiftUp,
				want: []we{{up, physShiftLeft, logShiftLeft, "", false}}, answer: keyConsumed},
		}},
		{"the right Shift is told by its scan code", nil, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKShift, scan: scanRShift}, setup: func(h *fakeKeyHost) { h.set(true, win32.VKRShift) },
				want: []we{{down, physShiftRight, logShiftRight, "", false}}, answer: keyConsumed},
		}},
		{"Control+A", nil, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKControl, scan: scanLControl}, setup: ctrlDown,
				want: []we{{down, physControlLeft, logControlLeft, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 1)},
				want: []we{{down, physKeyA, logKeyA, "", false}}, answer: keyConsumed, taken: 1},
		}},
		{"the extended bit tells the right Control", nil, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKControl, scan: 0xe000 | scanLControl}, setup: func(h *fakeKeyHost) { h.set(true, win32.VKRControl) },
				want: []we{{down, physControlRight, logControlRight, "", false}}, answer: keyConsumed},
		}},
		{"a modifier pressed while the window had no focus is synthesized before the next key", nil, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, setup: shiftDown, after: []kmsg{chr(win32.WMChar, 'A')},
				want: []we{{down, physShiftLeft, logShiftLeft, "", true}, {down, physKeyA, logKeyA, "A", false}}, answer: keyConsumed, taken: 1},
		}},
		{"a modifier released while the window had no focus is released before the next key", nil, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKShift, scan: scanLShift}, setup: shiftDown,
				want: []we{{down, physShiftLeft, logShiftLeft, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, setup: shiftUp, after: []kmsg{chr(win32.WMChar, 'a')},
				want: []we{{up, physShiftLeft, logShiftLeft, "", true}, {down, physKeyA, logKeyA, "a", false}}, answer: keyConsumed, taken: 1},
		}},
		{"CapsLock: the press toggles and the release does not", nil, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKCapital, scan: scanCapsLock}, setup: capsOn,
				want: []we{{down, physCapsLock, logCapsLock, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyUp, vk: win32.VKCapital, scan: scanCapsLock}, setup: capsReleased,
				want: []we{{up, physCapsLock, logCapsLock, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKCapital, scan: scanCapsLock}, setup: capsOff,
				want: []we{{down, physCapsLock, logCapsLock, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyUp, vk: win32.VKCapital, scan: scanCapsLock}, setup: capsReleased,
				want: []we{{up, physCapsLock, logCapsLock, "", false}}, answer: keyConsumed},
		}},
		{"CapsLock toggled while the window had no focus is a press and a release before the next key", nil, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, setup: func(h *fakeKeyHost) { h.toggle(true, win32.VKCapital) }, after: []kmsg{chr(win32.WMChar, 'A')},
				want: []we{
					{down, physCapsLock, logCapsLock, "", true},
					{up, physCapsLock, logCapsLock, "", true},
					{down, physKeyA, logKeyA, "A", false}}, answer: keyConsumed, taken: 1},
		}},
		{"Num Lock on at the start is the state the framework knows", func(h *fakeKeyHost) { h.toggle(true, win32.VKNumLock) }, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 'a')},
				want: []we{{down, physKeyA, logKeyA, "a", false}}, answer: keyConsumed, taken: 1},
		}},
		{"Num Lock turned off while the window had no focus", func(h *fakeKeyHost) { h.toggle(true, win32.VKNumLock) }, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, setup: numLockOff, after: []kmsg{chr(win32.WMChar, 'a')},
				want: []we{
					{down, physNumLock, logNumLock, "", true},
					{up, physNumLock, logNumLock, "", true},
					{down, physKeyA, logKeyA, "a", false}}, answer: keyConsumed, taken: 1},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := newKeyTester(t, tt.setup)
			k.handled = true
			runKeySteps(k, tt.steps)
		})
	}
}

func TestKeyboardNumpad(t *testing.T) {
	tests := []struct {
		name      string
		vk        uint32
		scan      uint32
		numLockOn bool
		phys, log uint64
		text      string
	}{
		{"Numpad 1 with Num Lock on", win32.VKNumpad1, scanNumpad1, true, physNumpad1, logNumpad1, "1"},
		{"the same key with Num Lock off is End but still Numpad 1", 0x23, scanNumpad1, false, physNumpad1, logNumpad1, ""},
		{"Numpad 2 with Num Lock off is the arrow down of the numpad", 0x28, scanNumpad2, false, physNumpad2, logNumpad2, ""},
		{"the arrow down key, extended, is not the numpad", 0x28, 0xe000 | 0x50, false, 0x70051, 0x100000301, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := newKeyTester(t, func(h *fakeKeyHost) { h.toggle(tt.numLockOn, win32.VKNumLock) })
			k.handled = true
			var after []kmsg
			if tt.text != "" {
				after = append(after, chr(win32.WMChar, rune(tt.text[0])))
			}
			if got := k.deliver(kmsg{msg: win32.WMKeyDown, vk: tt.vk, scan: tt.scan}, after...); got != keyConsumed {
				t.Errorf("answer %v", got)
			}
			k.expect(we{down, tt.phys, tt.log, tt.text, false})
		})
	}
}

func TestKeyboardLayouts(t *testing.T) {
	german := func(h *fakeKeyHost) {
		h.chars = map[uint32]uint32{scanKeyY: 'Z', scanKeyZ: 'Y', scanKeyQ: 'Q', scanKeyA: 'A'}
	}
	russian := func(h *fakeKeyHost) {
		// The key of A makes "ф", but the virtual key is still VK_A.
		h.chars = map[uint32]uint32{scanKeyA: 'ф', scanKeyQ: 'й', scanKeyE: 'у'}
	}
	russianNoLatinVK := func(h *fakeKeyHost) {
		h.chars = map[uint32]uint32{scanKeyA: 'ф', 0x27: 'ж' /* the key of the semicolon */}
	}
	dvorak := func(h *fakeKeyHost) {
		h.chars = map[uint32]uint32{scanKeyQ: '\'', scanKeyA: 'A'}
	}
	french := func(h *fakeKeyHost) {
		h.chars = map[uint32]uint32{scanBracketLeft: deadKeyCharMask | '^', scanKeyE: 'E', scanDigit1: '1'}
	}
	tests := []struct {
		name  string
		setup func(*fakeKeyHost)
		steps []kstep
	}{
		{"US: the key of Y and the key of Z", nil, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkY, scan: scanKeyY}, after: []kmsg{chr(win32.WMChar, 'y')},
				want: []we{{down, physKeyY, logKeyY, "y", false}}, answer: keyConsumed, taken: 1},
			{key: kmsg{msg: win32.WMKeyDown, vk: vkZ, scan: scanKeyZ}, after: []kmsg{chr(win32.WMChar, 'z')},
				want: []we{{down, physKeyZ, logKeyZ, "z", false}}, answer: keyConsumed, taken: 1},
		}},
		{"German: the physical key of Y makes Z and the physical key of Z makes Y", german, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkZ, scan: scanKeyY}, after: []kmsg{chr(win32.WMChar, 'z')},
				want: []we{{down, physKeyY, logKeyZ, "z", false}}, answer: keyConsumed, taken: 1},
			{key: kmsg{msg: win32.WMKeyDown, vk: vkY, scan: scanKeyZ}, after: []kmsg{chr(win32.WMChar, 'y')},
				want: []we{{down, physKeyZ, logKeyY, "y", false}}, answer: keyConsumed, taken: 1},
		}},
		{"AltGr+Q on a German layout types @ and drops the Control press of AltGr", german, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKControl, scan: scanLControl, time: 100}, setup: func(h *fakeKeyHost) { h.set(true, win32.VKLControl) },
				after: []kmsg{{msg: win32.WMSysKeyDown, vk: win32.VKMenu, scan: 0xe000 | scanLAlt, time: 100}},
				want:  nil, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKMenu, scan: 0xe000 | scanLAlt, time: 100}, setup: func(h *fakeKeyHost) { h.set(true, win32.VKRMenu) },
				want: []we{{down, physAltRight, logAltRight, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyDown, vk: vkQ, scan: scanKeyQ, time: 200}, after: []kmsg{chr(win32.WMChar, '@')},
				want: []we{{down, physKeyQ, logKeyQ, "@", false}}, answer: keyConsumed, taken: 1},
			{key: kmsg{msg: win32.WMKeyUp, vk: vkQ, scan: scanKeyQ, time: 300},
				want: []we{{up, physKeyQ, logKeyQ, "", false}}, answer: keyConsumed},
			// The release: the fake Control goes up with the Alt key.
			{key: kmsg{msg: win32.WMKeyUp, vk: win32.VKControl, scan: scanLControl, time: 400}, setup: func(h *fakeKeyHost) { h.set(false, win32.VKLControl) },
				want: []we{}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMSysKeyUp, vk: win32.VKMenu, scan: 0xe000 | scanLAlt, time: 400}, setup: func(h *fakeKeyHost) { h.set(false, win32.VKRMenu) },
				want: []we{{up, physAltRight, logAltRight, "", false}}, answer: keyConsumed},
		}},
		{"AltGr twice, the Control up never comes", german, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKControl, scan: scanLControl, time: 100}, setup: func(h *fakeKeyHost) { h.set(true, win32.VKLControl) },
				after: []kmsg{{msg: win32.WMKeyDown, vk: win32.VKMenu, scan: 0xe000 | scanLAlt, time: 100}},
				want:  nil, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKMenu, scan: 0xe000 | scanLAlt, time: 100}, setup: func(h *fakeKeyHost) { h.set(true, win32.VKRMenu) },
				want: []we{{down, physAltRight, logAltRight, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMSysKeyUp, vk: win32.VKMenu, scan: 0xe000 | scanLAlt, time: 200}, setup: func(h *fakeKeyHost) { h.set(false, win32.VKRMenu); h.set(false, win32.VKLControl) },
				want: []we{{up, physAltRight, logAltRight, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKControl, scan: scanLControl, time: 300}, setup: func(h *fakeKeyHost) { h.set(true, win32.VKLControl) },
				after: []kmsg{{msg: win32.WMKeyDown, vk: win32.VKMenu, scan: 0xe000 | scanLAlt, time: 300}},
				want:  nil, answer: keyConsumed},
		}},
		{"a real Control press followed by an Alt key at another time is kept", nil, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKControl, scan: scanLControl, time: 100}, setup: func(h *fakeKeyHost) { h.set(true, win32.VKLControl) },
				after: []kmsg{{msg: win32.WMKeyDown, vk: win32.VKMenu, scan: 0xe000 | scanLAlt, time: 160}},
				want:  []we{{down, physControlLeft, logControlLeft, "", false}}, answer: keyConsumed},
		}},
		{"Russian: the key of A makes the character of the US layout", russian, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 'ф')},
				want: []we{{down, physKeyA, logKeyA, "ф", false}}, answer: keyConsumed, taken: 1},
		}},
		{"a layout that gives no usable virtual key to a key of the goals: the goal's key", russianNoLatinVK, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: 0xff, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 'ф')},
				want: []we{{down, physKeyA, logKeyA, "ф", false}}, answer: keyConsumed, taken: 1},
		}},
		{"a layout key that is not Latin and is not a mandatory goal keeps the virtual key", russianNoLatinVK, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: 0xba, scan: 0x27}, // VK_OEM_1, Semicolon
				want: []we{{down, 0x70033, 0x3b, "", false}}, answer: keyConsumed},
		}},
		{"Dvorak: a punctuation mark on the key of Q is Latin, so the virtual key decides", dvorak, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: 0xde, scan: scanKeyQ}, after: []kmsg{chr(win32.WMChar, '\'')},
				want: []we{{down, physKeyQ, 0x22, "'", false}}, answer: keyConsumed, taken: 1},
		}},
		{"French: a dead key makes no text, the next key the composed character", french, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vkBracketRightUS, scan: scanBracketLeft}, after: []kmsg{chr(win32.WMDeadChar, '^')},
				want: []we{{down, physBracketLeft, logBracketRight, "", false}}, answer: keyConsumed, taken: 1},
			{key: kmsg{msg: win32.WMKeyUp, vk: vkBracketRightUS, scan: scanBracketLeft},
				want: []we{{up, physBracketLeft, logBracketRight, "", false}}, answer: keyConsumed},
			{key: kmsg{msg: win32.WMKeyDown, vk: vkE, scan: scanKeyE}, after: []kmsg{chr(win32.WMChar, 'ê')},
				want: []we{{down, physKeyE, logKeyE, "ê", false}}, answer: keyConsumed, taken: 1},
			{key: kmsg{msg: win32.WMKeyUp, vk: vkE, scan: scanKeyE},
				want: []we{{up, physKeyE, logKeyE, "", false}}, answer: keyConsumed},
		}},
		{"French: the digit row keeps the digit (Control+1)", french, []kstep{
			{key: kmsg{msg: win32.WMKeyDown, vk: vk1, scan: scanDigit1}, after: []kmsg{chr(win32.WMChar, '&')},
				want: []we{{down, physDigit1, logDigit1, "&", false}}, answer: keyConsumed, taken: 1},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := newKeyTester(t, tt.setup)
			k.handled = true
			runKeySteps(k, tt.steps)
		})
	}
}

func TestKeyboardSystemKeys(t *testing.T) {
	sysHandled := map[bool]string{true: "handled", false: "unhandled"}
	tests := []struct {
		name    string
		handled bool
		steps   []kstep
		// chars are the answers to the WM_SYSCHAR that reaches the window procedure afterwards.
		sysChar keyAnswer
	}{
		{"Alt+A handled: the key and its character are consumed", true, []kstep{
			{key: kmsg{msg: win32.WMSysKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMSysChar, 'a')},
				want: []we{{down, physKeyA, logKeyA, "a", false}}, answer: keyConsumed},
		}, keyConsumed},
		{"Alt+A unhandled: the system gets both", false, []kstep{
			{key: kmsg{msg: win32.WMSysKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMSysChar, 'a')},
				want: []we{{down, physKeyA, logKeyA, "a", false}}, answer: keyToSystem},
		}, keyToSystem},
		{"Alt+F4 goes to the system even when handled", true, []kstep{
			{key: kmsg{msg: win32.WMSysKeyDown, vk: win32.VKF4, scan: scanF4},
				want: []we{{down, physF4, logF4, "", false}}, answer: keyToSystem},
		}, keyToSystem},
		{"Alt+Space goes to the system even when handled", true, []kstep{
			{key: kmsg{msg: win32.WMSysKeyDown, vk: win32.VKSpace, scan: 0x39}, after: []kmsg{chr(win32.WMSysChar, ' ')},
				want: []we{{down, 0x7002c, 0x20, " ", false}}, answer: keyToSystem},
		}, keyToSystem},
		{"F10 handled is consumed", true, []kstep{
			{key: kmsg{msg: win32.WMSysKeyDown, vk: win32.VKF10, scan: 0x44},
				want: []we{{down, 0x70043, 0x10000080a, "", false}}, answer: keyConsumed},
		}, keyConsumed},
		{"the release of a system key is never passed on", false, []kstep{
			{key: kmsg{msg: win32.WMSysKeyDown, vk: win32.VKMenu, scan: scanLAlt}, setup: func(h *fakeKeyHost) { h.set(true, win32.VKLMenu) },
				want: []we{{down, physAltLeft, logAltLeft, "", false}}, answer: keyToSystem},
			{key: kmsg{msg: win32.WMSysKeyUp, vk: win32.VKMenu, scan: scanLAlt}, setup: func(h *fakeKeyHost) { h.set(false, win32.VKLMenu) },
				want: []we{{up, physAltLeft, logAltLeft, "", false}}, answer: keyConsumed},
		}, keyToSystem},
	}
	for _, tt := range tests {
		t.Run(tt.name+" ("+sysHandled[tt.handled]+")", func(t *testing.T) {
			k := newKeyTester(t, nil)
			k.handled = tt.handled
			runKeySteps(k, tt.steps)
			if got := k.m.handleChar(win32.WMSysChar); got != tt.sysChar {
				t.Errorf("the WM_SYSCHAR after it is answered %v, want %v", got, tt.sysChar)
			}
		})
	}
}

func TestKeyboardFocus(t *testing.T) {
	k := newKeyTester(t, nil)
	k.handled = true
	k.host.set(true, win32.VKLShift)
	runKeySteps(k, []kstep{
		{key: kmsg{msg: win32.WMKeyDown, vk: win32.VKShift, scan: scanLShift},
			want: []we{{down, physShiftLeft, logShiftLeft, "", false}}, answer: keyConsumed},
		{key: kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, after: []kmsg{chr(win32.WMChar, 'A')},
			want: []we{{down, physKeyA, logKeyA, "A", false}}, answer: keyConsumed, taken: 1},
	})
	// The window loses the focus with Shift and A down: both are released, in the order of the
	// physical keys.
	k.m.releaseAll()
	k.expect(we{up, physKeyA, logKeyA, "", true}, we{up, physShiftLeft, logShiftLeft, "", true})
	if got := k.m.emb.pressedState(); len(got) != 0 {
		t.Errorf("keys are still pressed after the loss of focus: %v", got)
	}
	// The user lets go of the keys while the window has no focus, so the focus returns with nothing
	// down, and the up messages that come later are ignored.
	k.host.set(false, win32.VKLShift)
	k.m.focusGained()
	k.expect()
	runKeySteps(k, []kstep{
		{key: kmsg{msg: win32.WMKeyUp, vk: vkA, scan: scanKeyA}, want: []we{{down, 0, 0, "", false}}, answer: keyConsumed},
	})
	// Num Lock and Caps Lock changed while the window had no focus: the focus returning says so.
	k.host.toggle(true, win32.VKCapital)
	k.m.focusGained()
	k.expect(we{down, physCapsLock, logCapsLock, "", true}, we{up, physCapsLock, logCapsLock, "", true})
	k.host.toggle(true, win32.VKNumLock)
	k.m.focusGained()
	k.expect(we{down, physNumLock, logNumLock, "", true}, we{up, physNumLock, logNumLock, "", true})
	// Shift held while the window gets the focus is pressed.
	k.host.set(true, win32.VKRShift)
	k.m.focusGained()
	k.expect(we{down, physShiftRight, logShiftRight, "", true})
}

func TestKeyboardSyncModifiers(t *testing.T) {
	tests := []struct {
		name           string
		first, then    [2]bool // shift, control
		wantFirst, end []we
	}{
		{"both pressed", [2]bool{true, true}, [2]bool{false, false},
			[]we{{down, physShiftLeft, logShiftLeft, "", true}, {down, physControlLeft, logControlLeft, "", true}},
			[]we{{up, physShiftLeft, logShiftLeft, "", true}, {up, physControlLeft, logControlLeft, "", true}}},
		{"Shift only", [2]bool{true, false}, [2]bool{true, false},
			[]we{{down, physShiftLeft, logShiftLeft, "", true}}, nil},
		{"nothing pressed", [2]bool{false, false}, [2]bool{false, false}, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := newKeyTester(t, nil)
			k.m.syncModifiers(tt.first[0], tt.first[1])
			k.expect(tt.wantFirst...)
			k.m.syncModifiers(tt.then[0], tt.then[1])
			k.expect(tt.end...)
		})
	}
	t.Run("the right Shift is released by the sync", func(t *testing.T) {
		k := newKeyTester(t, nil)
		k.handled = true
		k.host.set(true, win32.VKRShift)
		k.deliver(kmsg{msg: win32.WMKeyDown, vk: win32.VKShift, scan: scanRShift})
		k.expect(we{down, physShiftRight, logShiftRight, "", false})
		k.host.set(false, win32.VKRShift)
		k.m.syncModifiers(false, false)
		k.expect(we{up, physShiftRight, logShiftRight, "", true})
	})
}

func TestKeyboardUnansweredHandler(t *testing.T) {
	// A manager with no way to send (a handler that is not there) still answers.
	m := newKeyboardManager(newFakeKeyHost(), func() time.Duration { return 0 }, nil)
	q := &fakeKeyQueue{}
	if got := m.handleKey(q, win32.WMKeyDown, vkA, kmsg{scan: scanKeyA}.lparam(), 0); got != keyToSystem {
		t.Errorf("answer %v, want the system", got)
	}
	m.releaseAll()
	m.focusGained()
	m.syncModifiers(true, true)
	m.layoutChanged()
}

func TestResolveKeyCode(t *testing.T) {
	h := newFakeKeyHost()
	m := newKeyboardManager(h, func() time.Duration { return 0 }, nil)
	tests := []struct {
		name     string
		vk, scan uint32
		ext      bool
		want     uint32
	}{
		{"shift by the left scan code", win32.VKShift, scanLShift, false, win32.VKLShift},
		{"shift by the right scan code", win32.VKShift, scanRShift, false, win32.VKRShift},
		{"left shift virtual key by the right scan code", win32.VKLShift, scanRShift, false, win32.VKRShift},
		{"menu", win32.VKMenu, scanLAlt, false, win32.VKLMenu},
		{"menu extended", win32.VKMenu, scanLAlt, true, win32.VKRMenu},
		{"control", win32.VKControl, scanLControl, false, win32.VKLControl},
		{"control extended", win32.VKControl, scanLControl, true, win32.VKRControl},
		{"another key", vkA, scanKeyA, false, vkA},
		{"shift with an unknown scan code", win32.VKShift, 0x77, false, win32.VKShift},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.resolveKeyCode(tt.vk, tt.ext, tt.scan); got != tt.want {
				t.Errorf("resolveKeyCode = %#x, want %#x", got, tt.want)
			}
		})
	}
}

func TestCharCollector(t *testing.T) {
	tests := []struct {
		name string
		msgs []kmsg
		want string
	}{
		{"a letter", []kmsg{chr(win32.WMChar, 'a')}, "a"},
		{"a dead character is no text", []kmsg{chr(win32.WMDeadChar, '^')}, ""},
		{"a control character is no text", []kmsg{chr(win32.WMChar, 0x1b)}, ""},
		{"DEL is no text", []kmsg{chr(win32.WMChar, 0x7f)}, ""},
		{"Tab and Enter are control characters", []kmsg{chr(win32.WMChar, '\t'), chr(win32.WMChar, '\r')}, ""},
		{"a space is text", []kmsg{chr(win32.WMChar, ' ')}, " "},
		{"a surrogate pair", []kmsg{{msg: win32.WMChar, vk: 0xD83D}, {msg: win32.WMChar, vk: 0xDE00}}, "😀"},
		{"a low surrogate alone", []kmsg{{msg: win32.WMChar, vk: 0xDE00}}, ""},
		{"a high surrogate that a letter follows", []kmsg{{msg: win32.WMChar, vk: 0xD83D}, chr(win32.WMChar, 'x')}, "x"},
		{"an accent and a letter", []kmsg{chr(win32.WMChar, '´'), chr(win32.WMChar, 'x')}, "´x"},
		{"a dead character between", []kmsg{chr(win32.WMChar, 'a'), chr(win32.WMDeadChar, '`')}, "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c charCollector
			for _, m := range tt.msgs {
				c.add(m.msg, uintptr(m.vk))
			}
			if got := c.String(); got != tt.want {
				t.Errorf("text %q, want %q", got, tt.want)
			}
		})
	}
	if got := undeadChar(deadKeyCharMask | '^'); got != '^' {
		t.Errorf("undeadChar = %#x", got)
	}
}

func TestKeyboardIsExtendedKey(t *testing.T) {
	for _, tt := range []struct {
		vk   uint32
		want bool
	}{
		{win32.VKLeft, true}, {win32.VKDelete, true}, {win32.VKRControl, true}, {win32.VKRMenu, true},
		{win32.VKDivide, true}, {win32.VKLControl, false}, {vkA, false}, {win32.VKNumpad1, false}, {win32.VKReturn, false},
	} {
		if got := isExtendedKey(tt.vk); got != tt.want {
			t.Errorf("isExtendedKey(%#x) = %v", tt.vk, got)
		}
	}
}
