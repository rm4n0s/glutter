package android

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// The scan codes, key codes and key ids of KeyboardManagerTest.java.
const (
	scanKeyA         = 0x1e
	scanDigit1       = 0x2
	scanShiftLeft    = 0x2a
	scanShiftRight   = 0x36
	scanControlLeft  = 0x1d
	scanControlRight = 0x61
	scanAltLeft      = 0x38
	scanAltRight     = 0x64
	scanArrowLeft    = 0x69
	scanArrowRight   = 0x6a
	scanCapsLock     = 0x3a

	codeA          = 29
	code1          = 8
	codeDpadLeft   = 21
	codeEnter      = 66
	codeDel        = 67
	codeShiftLeft  = 59
	codeShiftRight = 60
	codeCtrlLeft   = 113
	codeCtrlRight  = 114
	codeAltLeft    = 57
	codeAltRight   = 58
	codeCapsLock   = 115
	codeButtonA    = 96

	physKeyA         uint64 = 0x70004
	physDigit1       uint64 = 0x7001e
	physShiftLeft    uint64 = 0x700e1
	physShiftRight   uint64 = 0x700e5
	physControlLeft  uint64 = 0x700e0
	physControlRight uint64 = 0x700e4
	physAltLeft      uint64 = 0x700e2
	physAltRight     uint64 = 0x700e6
	physArrowLeft    uint64 = 0x70050
	physArrowRight   uint64 = 0x7004f
	physCapsLock     uint64 = 0x70039

	logKeyA         uint64 = 0x61
	logDigit1       uint64 = 0x31
	logShiftLeft    uint64 = 0x200000102
	logShiftRight   uint64 = 0x200000103
	logControlLeft  uint64 = 0x200000100
	logControlRight uint64 = 0x200000101
	logAltLeft      uint64 = 0x200000104
	logAltRight     uint64 = 0x200000105
	logEnter        uint64 = 0x10000000d
	logBackspace    uint64 = 0x100000008
	logArrowLeft    uint64 = 0x100000302
	logCapsLock     uint64 = 0x100000104
	logGameButtonA  uint64 = 0x200000311

	metaShiftLeftOn int32 = 0x40 // META_SHIFT_LEFT_ON
)

// ev makes the event of FakeKeyEvent: the unicode char is a rune.
func ev(action, scan, code, repeat int32, char rune, meta int32) keyEvent {
	return keyEvent{Action: action, ScanCode: scan, KeyCode: code, RepeatCount: repeat, UnicodeChar: int32(char), MetaState: meta, Source: sourceKeyboard}
}

func (e keyEvent) withDevice(device, source int32) keyEvent {
	e.DeviceID, e.Source = device, source
	return e
}

// want is the key data that a test expects (the time stamp aside).
type want struct {
	typ         ui.KeyEventType
	physical    uint64
	logical     uint64
	char        string
	synthesized bool
	device      ui.KeyEventDeviceType
}

func wantData(typ ui.KeyEventType, physical, logical uint64, char string, synthesized bool) want {
	return want{typ, physical, logical, char, synthesized, ui.KeyEventDeviceTypeKeyboard}
}

func (w want) String() string {
	return fmt.Sprintf("{%v phys=%#x log=%#x char=%q synth=%v dev=%v}", w.typ, w.physical, w.logical, w.char, w.synthesized, w.device)
}

func summarize(data []ui.KeyData) []want {
	out := make([]want, 0, len(data))
	for _, d := range data {
		out = append(out, want{d.Type, d.Physical, d.Logical, d.Character, d.Synthesized, d.DeviceType})
	}
	return out
}

const (
	down   = ui.KeyEventTypeDown
	up     = ui.KeyEventTypeUp
	repeat = ui.KeyEventTypeRepeat
)

func shiftData(left, isDown, synthesized bool) want {
	typ := up
	if isDown {
		typ = down
	}
	if left {
		return wantData(typ, physShiftLeft, logShiftLeft, "", synthesized)
	}
	return wantData(typ, physShiftRight, logShiftRight, "", synthesized)
}

// step is an event and the key data that it must make.
type step struct {
	ev   keyEvent
	want []want
}

func runSteps(t *testing.T, e *keyEmbedder, steps []step) {
	t.Helper()
	for i, s := range steps {
		got := summarize(e.HandleEvent(s.ev))
		if !reflect.DeepEqual(got, s.want) {
			t.Errorf("step %d (%+v):\n got  %v\n want %v", i, s.ev, got, s.want)
		}
	}
}

// emptyEvent is the event that Flutter makes when there is nothing to send.
var emptyEvent = wantData(down, 0, 0, "", true)

func TestKeyEmbedderSequences(t *testing.T) {
	const (
		keyDown = keyActionDown
		keyUp   = keyActionUp
	)
	tests := []struct {
		name  string
		steps []step
	}{
		{"tapLowerA", []step{
			{ev(keyDown, scanKeyA, codeA, 0, 'a', 0), []want{wantData(down, physKeyA, logKeyA, "a", false)}},
			{ev(keyDown, scanKeyA, codeA, 1, 'a', 0), []want{wantData(repeat, physKeyA, logKeyA, "a", false)}},
			{ev(keyUp, scanKeyA, codeA, 0, 'a', 0), []want{wantData(up, physKeyA, logKeyA, "", false)}},
		}},
		{"virtualKeyboardShiftUpClearsState", []step{
			{ev(keyDown, 0, codeShiftLeft, 0, 0, metaShiftOn).withDevice(virtualKeyboardDevice, 0),
				[]want{wantData(down, keyAndroidPlane|codeShiftLeft, logShiftLeft, "", false)}},
			// The meta bit is still on, and no ShiftRight is synthesized.
			{ev(keyUp, 0, codeShiftLeft, 0, 0, metaShiftOn).withDevice(virtualKeyboardDevice, 0),
				[]want{wantData(up, keyAndroidPlane|codeShiftLeft, logShiftLeft, "", false)}},
			{ev(keyDown, 0, codeDel, 0, 0, metaShiftOn).withDevice(virtualKeyboardDevice, 0),
				[]want{wantData(down, keyAndroidPlane|codeDel, logBackspace, "", false)}},
		}},
		{"tapUpperA", []step{
			{ev(keyDown, scanShiftLeft, codeShiftLeft, 0, 0, 0x41), []want{shiftData(true, true, false)}},
			{ev(keyDown, scanKeyA, codeA, 0, 'A', 0x41), []want{wantData(down, physKeyA, logKeyA, "A", false)}},
			{ev(keyUp, scanKeyA, codeA, 0, 'A', 0x41), []want{wantData(up, physKeyA, logKeyA, "", false)}},
			{ev(keyUp, scanShiftLeft, codeShiftLeft, 0, 0, 0), []want{shiftData(true, false, false)}},
		}},
		{"eventsWithMintedCodes", []step{
			{ev(keyDown, 0, codeEnter, 0, '\n', 0), []want{wantData(down, 0x1100000042, logEnter, "\n", false)}},
			{ev(keyDown, 0, 0, 0, 0, 0), []want{emptyEvent}},
			{ev(keyDown, -0x21524111, 0, 0, 0, 0), []want{wantData(down, 0x11DEADBEEF, 0x1100000000, "", false)}}, // 0xDEADBEEF as an int
			{ev(keyDown, scanArrowLeft, 0, 0, 0, 0), []want{wantData(down, physArrowLeft, 0x1100000000, "", false)}},
			{ev(keyDown, scanArrowRight, -0x21524111, 0, 0, 0), []want{wantData(down, physArrowRight, 0x11DEADBEEF, "", false)}},
		}},
		{"duplicateDownEventsArePrecededBySynthesizedUpEvents", []step{
			{ev(keyDown, scanKeyA, codeA, 0, 'a', 0), []want{wantData(down, physKeyA, logKeyA, "a", false)}},
			{ev(keyDown, scanKeyA, codeA, 0, 'a', 0), []want{
				wantData(up, physKeyA, logKeyA, "", true),
				wantData(down, physKeyA, logKeyA, "a", false),
			}},
		}},
		{"abruptUpEventsAreIgnored", []step{
			{ev(keyUp, scanKeyA, codeA, 0, 'a', 0), []want{emptyEvent}},
			{ev(keyActionUp+1, scanKeyA, codeA, 0, 'a', 0), []want{emptyEvent}}, // ACTION_MULTIPLE
		}},
		{"modifierKeys", []step{
			{ev(keyDown, scanShiftLeft, codeShiftLeft, 0, 0, 0x41), []want{shiftData(true, true, false)}},
			{ev(keyUp, scanShiftLeft, codeShiftLeft, 0, 0, 0), []want{shiftData(true, false, false)}},
			{ev(keyDown, scanShiftRight, codeShiftRight, 0, 0, 0x41), []want{shiftData(false, true, false)}},
			{ev(keyUp, scanShiftRight, codeShiftRight, 0, 0, 0), []want{shiftData(false, false, false)}},
			{ev(keyDown, scanControlLeft, codeCtrlLeft, 0, 0, 0x3000), []want{wantData(down, physControlLeft, logControlLeft, "", false)}},
			{ev(keyUp, scanControlLeft, codeCtrlLeft, 0, 0, 0), []want{wantData(up, physControlLeft, logControlLeft, "", false)}},
			{ev(keyDown, scanControlRight, codeCtrlRight, 0, 0, 0x3000), []want{wantData(down, physControlRight, logControlRight, "", false)}},
			{ev(keyUp, scanControlRight, codeCtrlRight, 0, 0, 0), []want{wantData(up, physControlRight, logControlRight, "", false)}},
			{ev(keyDown, scanAltLeft, codeAltLeft, 0, 0, 0x12), []want{wantData(down, physAltLeft, logAltLeft, "", false)}},
			{ev(keyUp, scanAltLeft, codeAltLeft, 0, 0, 0), []want{wantData(up, physAltLeft, logAltLeft, "", false)}},
			{ev(keyDown, scanAltRight, codeAltRight, 0, 0, 0x12), []want{wantData(down, physAltRight, logAltRight, "", false)}},
			{ev(keyUp, scanAltRight, codeAltRight, 0, 0, 0), []want{wantData(up, physAltRight, logAltRight, "", false)}},
		}},
		{"nonUsKeys", []step{
			// French 1.
			{ev(keyDown, scanDigit1, code1, 0, '1', 0), []want{wantData(down, physDigit1, logDigit1, "1", false)}},
			{ev(keyUp, scanDigit1, code1, 0, '1', 0), []want{wantData(up, physDigit1, logDigit1, "", false)}},
			// French Shift-1.
			{ev(keyDown, scanShiftLeft, codeShiftLeft, 0, 0, 0x41), []want{shiftData(true, true, false)}},
			{ev(keyDown, scanDigit1, code1, 0, '&', 0x41), []want{wantData(down, physDigit1, logDigit1, "&", false)}},
			{ev(keyUp, scanDigit1, code1, 0, '&', 0x41), []want{wantData(up, physDigit1, logDigit1, "", false)}},
			{ev(keyUp, scanShiftLeft, codeShiftLeft, 0, 0, 0), []want{shiftData(true, false, false)}},
			// Russian lower a.
			{ev(keyDown, scanKeyA, codeA, 0, 0x444, 0), []want{wantData(down, physKeyA, logKeyA, "ф", false)}},
			{ev(keyUp, scanKeyA, codeA, 0, 0x444, 0), []want{wantData(up, physKeyA, logKeyA, "", false)}},
		}},
		{"synchronizeShiftLeftDuringForeignKeyEvents", []step{
			{ev(keyDown, scanArrowLeft, codeDpadLeft, 0, 0, metaShiftLeftOn|metaShiftOn), []want{
				shiftData(true, true, true),
				wantData(down, physArrowLeft, logArrowLeft, "", false),
			}},
			{ev(keyUp, scanArrowLeft, codeDpadLeft, 0, 0, 0), []want{
				shiftData(true, false, true),
				wantData(up, physArrowLeft, logArrowLeft, "", false),
			}},
		}},
		{"synchronizeShiftLeftDuringSelfKeyEvents", func() []step {
			on := metaShiftLeftOn | metaShiftOn
			return []step{
				// Repeat event when the current state is 0 (a down).
				{ev(keyDown, scanShiftLeft, codeShiftLeft, 1, 0, on), []want{shiftData(true, true, false)}},
				// Down event when the current state is 1.
				{ev(keyDown, scanShiftLeft, codeShiftLeft, 0, 0, on), []want{shiftData(true, false, true), shiftData(true, true, false)}},
				// Up event when the current state is 1.
				{ev(keyUp, scanShiftLeft, codeShiftLeft, 0, 0, 0), []want{shiftData(true, false, false)}},
				// Up event when the current state is 0.
				{ev(keyUp, scanShiftLeft, codeShiftLeft, 0, 0, 0), []want{emptyEvent}},
				// Down event when the current state is 0.
				{ev(keyDown, scanShiftLeft, codeShiftLeft, 0, 0, on), []want{shiftData(true, true, false)}},
				// Repeat event when the current state is 1.
				{ev(keyDown, scanShiftLeft, codeShiftLeft, 1, 0, on), []want{wantData(repeat, physShiftLeft, logShiftLeft, "", false)}},
			}
		}()},
		{"synchronizeOtherModifiers", []step{
			{ev(keyDown, scanArrowLeft, codeDpadLeft, 0, 0, metaCtrlOn), []want{
				wantData(down, physControlLeft, logControlLeft, "", true),
				wantData(down, physArrowLeft, logArrowLeft, "", false),
			}},
			{ev(keyUp, scanArrowLeft, codeDpadLeft, 0, 0, 0), []want{
				wantData(up, physControlLeft, logControlLeft, "", true),
				wantData(up, physArrowLeft, logArrowLeft, "", false),
			}},
			{ev(keyDown, scanArrowLeft, codeDpadLeft, 0, 0, metaAltOn), []want{
				wantData(down, physAltLeft, logAltLeft, "", true),
				wantData(down, physArrowLeft, logArrowLeft, "", false),
			}},
			{ev(keyUp, scanArrowLeft, codeDpadLeft, 0, 0, 0), []want{
				wantData(up, physAltLeft, logAltLeft, "", true),
				wantData(up, physArrowLeft, logArrowLeft, "", false),
			}},
		}},
		{"synchronizeModifiersForConflictingMetaState", []step{
			// Down of ShiftLeft with a meta state of 0: the main event and then a synthesized up.
			{ev(keyDown, scanShiftLeft, codeShiftLeft, 0, 0, 0), []want{shiftData(true, true, false), shiftData(true, false, true)}},
			{ev(keyDown, scanShiftLeft, codeShiftLeft, 0, 0, metaShiftLeftOn|metaShiftOn), []want{shiftData(true, true, false)}},
			// Repeat event when the current state is 0.
			{ev(keyDown, scanShiftLeft, codeShiftLeft, 1, 0, 0), []want{wantData(repeat, physShiftLeft, logShiftLeft, "", false), shiftData(true, false, true)}},
		}},
		{"synchronizeModifiersForZeroedScanCode", []step{
			{ev(keyDown, 0, codeShiftLeft, 0, 0, 0), []want{
				wantData(down, keyAndroidPlane|codeShiftLeft, logShiftLeft, "", false),
				wantData(up, keyAndroidPlane|codeShiftLeft, logShiftLeft, "", true),
			}},
		}},
		{"synchronizeModifiersForZeroedScanCodeOnRepeatEvent", []step{
			{ev(keyDown, 0, codeShiftLeft, 1, 0, 0), []want{
				wantData(down, keyAndroidPlane|codeShiftLeft, logShiftLeft, "", false),
				wantData(up, keyAndroidPlane|codeShiftLeft, logShiftLeft, "", true),
			}},
			{ev(keyDown, 0, codeAltLeft, 1, 0, 0), []want{
				wantData(down, keyAndroidPlane|codeAltLeft, logAltLeft, "", false),
				wantData(up, keyAndroidPlane|codeAltLeft, logAltLeft, "", true),
			}},
		}},
		{"normalCapsLockEvents", []step{
			// ChromeOS sets the CapsLock bit on the down and not on the up of the key.
			{ev(keyDown, scanCapsLock, codeCapsLock, 0, 0, metaCapsLockOn), []want{wantData(down, physCapsLock, logCapsLock, "", false)}},
			{ev(keyUp, scanCapsLock, codeCapsLock, 0, 0, 0), []want{wantData(up, physCapsLock, logCapsLock, "", false)}},
			{ev(keyDown, scanArrowLeft, codeDpadLeft, 0, 0, metaCapsLockOn), []want{wantData(down, physArrowLeft, logArrowLeft, "", false)}},
			{ev(keyUp, scanArrowLeft, codeDpadLeft, 0, 0, metaCapsLockOn), []want{wantData(up, physArrowLeft, logArrowLeft, "", false)}},
			{ev(keyDown, scanCapsLock, codeCapsLock, 0, 0, metaCapsLockOn), []want{wantData(down, physCapsLock, logCapsLock, "", false)}},
			{ev(keyUp, scanCapsLock, codeCapsLock, 0, 0, 0), []want{wantData(up, physCapsLock, logCapsLock, "", false)}},
			{ev(keyDown, scanArrowLeft, codeDpadLeft, 0, 0, 0), []want{wantData(down, physArrowLeft, logArrowLeft, "", false)}},
			{ev(keyUp, scanArrowLeft, codeDpadLeft, 0, 0, 0), []want{wantData(up, physArrowLeft, logArrowLeft, "", false)}},
		}},
		{"synchronizeCapsLock", []step{
			{ev(keyDown, scanArrowLeft, codeDpadLeft, 0, 0, metaCapsLockOn), []want{
				wantData(down, physCapsLock, logCapsLock, "", true),
				wantData(up, physCapsLock, logCapsLock, "", true),
				wantData(down, physArrowLeft, logArrowLeft, "", false),
			}},
			{ev(keyDown, scanCapsLock, codeCapsLock, 0, 0, metaCapsLockOn), []want{wantData(down, physCapsLock, logCapsLock, "", false)}},
			// The lock is off in the meta state while the key is down: up, down, then the main up.
			{ev(keyUp, scanArrowLeft, codeDpadLeft, 0, 0, metaCapsLockOn), []want{
				wantData(up, physCapsLock, logCapsLock, "", true),
				wantData(down, physCapsLock, logCapsLock, "", true),
				wantData(up, physArrowLeft, logArrowLeft, "", false),
			}},
			{ev(keyUp, scanCapsLock, codeCapsLock, 0, 0, 0), []want{wantData(up, physCapsLock, logCapsLock, "", false)}},
		}},
		{"deviceTypeFromInputDevice", []step{
			{ev(keyDown, scanKeyA, codeA, 0, 'a', 0).withDevice(0, sourceKeyboard),
				[]want{wantData(down, physKeyA, logKeyA, "a", false)}},
			{ev(keyDown, scanArrowLeft, codeDpadLeft, 0, 0, 0).withDevice(0, sourceDpad),
				[]want{{down, physArrowLeft, logArrowLeft, "", false, ui.KeyEventDeviceTypeDirectionalPad}}},
			// The same physical key goes down again with another logical key: an up first.
			{ev(keyDown, scanArrowLeft, codeButtonA, 0, 0, 0).withDevice(0, sourceGamepad), []want{
				wantData(up, physArrowLeft, logArrowLeft, "", true),
				{down, physArrowLeft, logGameButtonA, "", false, ui.KeyEventDeviceTypeGamepad},
			}},
			{ev(keyDown, scanArrowLeft, codeButtonA, 0, 0, 0).withDevice(0, sourceHDMI), []want{
				wantData(up, physArrowLeft, logGameButtonA, "", true),
				{down, physArrowLeft, logGameButtonA, "", false, ui.KeyEventDeviceTypeHDMI},
			}},
			{ev(keyUp, scanArrowLeft, codeButtonA, 0, 0, 0).withDevice(0, sourceJoystick),
				[]want{{up, physArrowLeft, logGameButtonA, "", false, ui.KeyEventDeviceTypeJoystick}}},
			// An unknown source is a keyboard.
			{ev(keyDown, scanArrowRight, 22, 0, 0, 0).withDevice(0, 0x4000),
				[]want{wantData(down, physArrowRight, 0x100000303, "", false)}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSteps(t, newKeyEmbedder(), tt.steps)
		})
	}
}

// TestSynchronizeShiftLeftDuringSiblingKeyEvents is synchronizeShiftLeftDuringSiblingKeyEvents:
// the events of ShiftRight with every state of both shifts before them. Even though the
// event is for ShiftRight, the meta state has the left bit (see synchronizePressingKey).
func TestSynchronizeShiftLeftDuringSiblingKeyEvents(t *testing.T) {
	const on = metaShiftLeftOn | metaShiftOn
	tests := []struct {
		name                      string
		leftPressed, rightPressed bool
		rightIsDown, truePressed  bool
		want                      []want
	}{
		{name: "up, true released, none pressed", want: nil},
		{name: "up, true released, right pressed", rightPressed: true, want: []want{shiftData(false, false, false)}},
		{name: "up, true released, left pressed", leftPressed: true, want: []want{shiftData(true, false, true)}},
		{name: "up, true released, both pressed", leftPressed: true, rightPressed: true,
			want: []want{shiftData(true, false, true), shiftData(false, false, false)}},
		{name: "up, true pressed, none pressed", truePressed: true, want: []want{shiftData(true, true, true)}},
		{name: "up, true pressed, right pressed", rightPressed: true, truePressed: true,
			want: []want{shiftData(true, true, true), shiftData(false, false, false)}},
		{name: "up, true pressed, left pressed", leftPressed: true, truePressed: true, want: nil},
		{name: "up, true pressed, both pressed", leftPressed: true, rightPressed: true, truePressed: true,
			want: []want{shiftData(false, false, false)}},
		{name: "down, true pressed, none pressed", rightIsDown: true, truePressed: true, want: []want{shiftData(false, true, false)}},
		{name: "down, true pressed, right pressed", rightPressed: true, rightIsDown: true, truePressed: true,
			want: []want{shiftData(false, false, true), shiftData(false, true, false)}},
		{name: "down, true pressed, left pressed", leftPressed: true, rightIsDown: true, truePressed: true,
			want: []want{shiftData(false, true, false)}},
		{name: "down, true pressed, both pressed", leftPressed: true, rightPressed: true, rightIsDown: true, truePressed: true,
			want: []want{shiftData(false, false, true), shiftData(false, true, false)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newKeyEmbedder()
			if tt.leftPressed {
				e.HandleEvent(ev(keyActionDown, scanShiftLeft, codeShiftLeft, 0, 0, on))
			}
			if tt.rightPressed {
				e.HandleEvent(ev(keyActionDown, scanShiftRight, codeShiftRight, 0, 0, on))
			}
			action, meta := keyActionUp, int32(0)
			if tt.rightIsDown {
				action = keyActionDown
			}
			if tt.truePressed {
				meta = on
			}
			got := summarize(e.HandleEvent(ev(action, scanShiftRight, codeShiftRight, 0, 0, meta)))
			// The test of Flutter drops the events that have no physical key.
			var filtered []want
			for _, w := range got {
				if w.physical != 0 {
					filtered = append(filtered, w)
				}
			}
			if !reflect.DeepEqual(filtered, tt.want) {
				t.Errorf("got %v, want %v", filtered, tt.want)
			}
		})
	}
}

func TestKeyEmbedderPressedState(t *testing.T) {
	e := newKeyEmbedder()
	if got := e.PressedState(); len(got) != 0 {
		t.Errorf("the initial state is %v", got)
	}
	e.HandleEvent(ev(keyActionDown, scanKeyA, codeA, 1, 'a', 0))
	if got, want := e.PressedState(), map[uint64]uint64{physKeyA: logKeyA}; !reflect.DeepEqual(got, want) {
		t.Errorf("after the down: %v, want %v", got, want)
	}
	// The state is a copy.
	e.PressedState()[1] = 2
	if len(e.PressedState()) != 1 {
		t.Error("the returned state changes the embedder's")
	}
	e.HandleEvent(ev(keyActionUp, scanKeyA, codeA, 0, 'a', 0))
	if got := e.PressedState(); len(got) != 0 {
		t.Errorf("after the up: %v", got)
	}
}

func TestKeyEmbedderTimestamps(t *testing.T) {
	e := newKeyEmbedder()
	// The synthesized ShiftLeft takes the time of the event, and the empty event is at 0.
	e1 := ev(keyActionDown, scanArrowLeft, codeDpadLeft, 0, 0, metaShiftOn)
	e1.EventTime = 1234
	data := e.HandleEvent(e1)
	if len(data) != 2 || data[0].TimeStamp != 1234*time.Millisecond || data[1].TimeStamp != 1234*time.Millisecond {
		t.Errorf("got %+v", data)
	}
	e2 := ev(keyActionUp, scanKeyA, codeA, 0, 0, metaShiftOn)
	e2.EventTime = 99
	data = e.HandleEvent(e2)
	if last := data[len(data)-1]; last.TimeStamp != 0 || !last.Synthesized || last.Physical != 0 {
		t.Errorf("the empty event is %+v", last)
	}
}

func TestCharacterCombiner(t *testing.T) {
	const dead = int32('`') | -0x80000000 // '`' | KeyCharacterMap.COMBINING_ACCENT
	tests := []struct {
		name string
		in   int32
		want rune
	}{
		{"nothing", 0, 0},
		{"a letter", 'B', 'B'},
		{"the same letter", 'B', 'B'},
		{"another letter", 'A', 'A'},
		{"nothing again", 0, 0},
		{"a dead key returns its accent", dead, '`'},
		{"the same dead key twice is the accent", dead, '`'},
		{"a letter after a dead key", 'A', 'À'},
		{"a dead key", dead, '`'},
		{"nothing removes the dead key", 0, 0},
		{"the letter after that is plain", 'A', 'A'},
		{"nothing", 0, 0},
		{"a dead key", dead, '`'},
		{"a letter that composes", 'A', 'À'},
		{"a dead key", dead, '`'},
		{"a letter that does not compose", 'x', 'x'},
		{"the dead key is gone", 'e', 'e'},
		{"acute", int32(0xb4) | -0x80000000, 0xb4},
		{"a letter with the acute", 'e', 'é'},
		{"a combining acute", 0x301 | -0x80000000, 0x301},
		{"o with the combining acute", 'o', 'ó'},
		{"circumflex", int32('^') | -0x80000000, '^'},
		{"space after a dead key is the accent", ' ', '^'},
		{"a letter outside the basic plane", 0x1f600, 0x1f600},
	}
	var c characterCombiner
	for _, tt := range tests {
		if got := c.apply(tt.in); got != tt.want {
			t.Errorf("%s: apply(%#x) = %q, want %q", tt.name, uint32(tt.in), got, tt.want)
		}
	}
}

func TestDeadChar(t *testing.T) {
	tests := []struct {
		accent, c, want rune
	}{
		{'`', 'A', 'À'},
		{0x300, 'a', 'à'},
		{0xb4, 'E', 'É'},
		{'\'', 'e', 'é'},
		{'^', 'o', 'ô'},
		{'~', 'n', 'ñ'},
		{'~', 'N', 'Ñ'},
		{0xa8, 'u', 'ü'},
		{'"', 'U', 'Ü'},
		{0xb8, 'c', 'ç'},
		{0x2da, 'a', 'å'},
		{0x2c7, 'c', 'č'},
		{0xaf, 'a', 'ā'},
		{0x2d8, 'g', 'ğ'},
		{0x2dd, 'o', 'ő'},
		{0x301, 0x3b1, 'ά'}, // Greek alpha with tonos
		{0x301, 0x435, 0},   // Cyrillic e has no precomposed form with the acute
		{'`', '`', '`'},
		{'`', ' ', '`'},
		{'`', 'x', 0},
		{'`', 0, 0},
		{'a', 'a', 'a'},
		{'a', 'b', 0},
	}
	for _, tt := range tests {
		if got := deadChar(tt.accent, tt.c); got != tt.want {
			t.Errorf("deadChar(%q, %q) = %q, want %q", tt.accent, tt.c, got, tt.want)
		}
	}
}

func TestKeyboardManager(t *testing.T) {
	// The framework answers the main event with answer and every synthesized event with
	// synthAnswer.
	type run struct {
		ev          keyEvent
		answer      bool
		synthAnswer bool
		wantSent    int
		wantHandled bool
	}
	tests := []struct {
		name string
		pre  []keyEvent
		run  run
	}{
		{name: "the framework handles the event",
			run: run{ev: ev(keyActionDown, scanKeyA, codeA, 0, 'a', 0), answer: true, wantSent: 1, wantHandled: true}},
		{name: "the framework does not handle the event: it is reported",
			run: run{ev: ev(keyActionDown, scanKeyA, codeA, 0, 'a', 0), answer: false, wantSent: 1, wantHandled: false}},
		{name: "the answers for synthesized events do not count",
			run: run{ev: ev(keyActionDown, scanArrowLeft, codeDpadLeft, 0, 0, metaShiftOn), answer: false, synthAnswer: true, wantSent: 2, wantHandled: false}},
		{name: "the answer of the main event counts after synthesized events",
			run: run{ev: ev(keyActionDown, scanArrowLeft, codeDpadLeft, 0, 0, metaShiftOn), answer: true, synthAnswer: false, wantSent: 2, wantHandled: true}},
		{name: "the answer of the main event counts before synthesized events",
			run: run{ev: ev(keyActionDown, scanShiftLeft, codeShiftLeft, 0, 0, 0), answer: false, synthAnswer: true, wantSent: 2, wantHandled: false}},
		{name: "an abrupt up has no main event: handled, and the empty event is sent",
			run: run{ev: ev(keyActionUp, scanKeyA, codeA, 0, 'a', 0), answer: false, synthAnswer: false, wantSent: 1, wantHandled: true}},
		{name: "an event without codes is handled",
			run: run{ev: ev(keyActionDown, 0, 0, 0, 0, 0), wantSent: 1, wantHandled: true}},
		{name: "an up after the down is reported when unhandled",
			pre: []keyEvent{ev(keyActionDown, scanKeyA, codeA, 0, 'a', 0)},
			run: run{ev: ev(keyActionUp, scanKeyA, codeA, 0, 'a', 0), answer: false, wantSent: 1, wantHandled: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newKeyboardManager()
			for _, p := range tt.pre {
				m.Handle(p, func(ui.KeyData) bool { return true })
			}
			var sent []ui.KeyData
			handled := m.Handle(tt.run.ev, func(d ui.KeyData) bool {
				sent = append(sent, d)
				if d.Synthesized {
					return tt.run.synthAnswer
				}
				return tt.run.answer
			})
			if handled != tt.run.wantHandled {
				t.Errorf("handled = %v, want %v", handled, tt.run.wantHandled)
			}
			if len(sent) != tt.run.wantSent {
				t.Errorf("sent %d key data, want %d: %+v", len(sent), tt.run.wantSent, sent)
			}
		})
	}
}

func TestKeyboardManagerSendsInOrderAndKeepsState(t *testing.T) {
	m := newKeyboardManager()
	var order []want
	handled := m.Handle(ev(keyActionDown, scanKeyA, codeA, 0, 'A', metaShiftOn), func(d ui.KeyData) bool {
		order = append(order, summarize([]ui.KeyData{d})...)
		return true
	})
	wantOrder := []want{
		shiftData(true, true, true),
		wantData(down, physKeyA, logKeyA, "A", false),
	}
	if !handled || !reflect.DeepEqual(order, wantOrder) {
		t.Errorf("handled=%v order=%v, want %v", handled, order, wantOrder)
	}
	if got, want := m.PressedState(), (map[uint64]uint64{physShiftLeft: logShiftLeft, physKeyA: logKeyA}); !reflect.DeepEqual(got, want) {
		t.Errorf("the pressed state is %v, want %v", got, want)
	}
	// A nil send is the same as a framework that handles nothing.
	if m.Handle(ev(keyActionUp, scanKeyA, codeA, 0, 'A', metaShiftOn), nil) {
		t.Error("an event with no one to send it to was handled")
	}
}

func TestKeyTables(t *testing.T) {
	tests := []struct {
		name  string
		got   uint64
		found bool
		want  uint64
	}{
		{"scan code 30 is KeyA", scanCodeToPhysical[30], true, physKeyA},
		{"scan code 29 is ControlLeft", scanCodeToPhysical[29], true, physControlLeft},
		{"scan code 0x7a is not a key", scanCodeToPhysical[0xffff], false, 0},
		{"KEYCODE_A is the logical key a", keyCodeToLogical[29], true, logKeyA},
		{"KEYCODE_ENTER is Enter", keyCodeToLogical[66], true, logEnter},
		{"KEYCODE_BUTTON_A is GameButtonA", keyCodeToLogical[96], true, logGameButtonA},
		{"an unknown key code", keyCodeToLogical[0xffff], false, 0},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
	if len(scanCodeToPhysical) < 150 || len(keyCodeToLogical) < 200 {
		t.Errorf("the tables are too small: %d scan codes, %d key codes", len(scanCodeToPhysical), len(keyCodeToLogical))
	}
	if len(pressingGoals) != 3 || len(togglingGoals) != 1 {
		t.Errorf("%d pressing goals and %d toggling goals", len(pressingGoals), len(togglingGoals))
	}
	for _, g := range pressingGoals {
		if len(g.keys) != 2 {
			t.Errorf("a pressing goal with %d keys", len(g.keys))
		}
	}
}
