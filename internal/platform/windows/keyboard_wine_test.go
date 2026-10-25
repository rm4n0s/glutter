//go:build windows && glutter_wine

package windows

import (
	"context"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests send keys to a window under Wine the way a keyboard does (SendInput) and read the key
// data the handler gets (SPEC §25.6, §25.13).

// keyRec records the key data that the platform gives at once (platform.SyncInputHandler).
type keyRec struct {
	*rec
	keys []ui.KeyData
	// answer is what OnKeySync says for the key data that is not synthesized.
	answer bool
}

func (r *keyRec) OnKeySync(d ui.KeyData) bool {
	r.keys = append(r.keys, d)
	return r.answer && !d.Synthesized
}
func (r *keyRec) OnPointerSync([]ui.PointerData)    {}
func (r *keyRec) OnTextInputSync(ui.TextInputEvent) {}

// kd is the part of a key data that the tests compare.
type kd struct {
	typ   ui.KeyEventType
	log   uint64
	text  string
	synth bool
}

func (r *keyRec) brief() []kd {
	out := make([]kd, len(r.keys))
	for i, d := range r.keys {
		out[i] = kd{d.Type, d.Logical, d.Character, d.Synthesized}
	}
	return out
}

func (r *keyRec) mainOnly() []kd {
	var out []kd
	for _, k := range r.brief() {
		if !k.synth {
			out = append(out, k)
		}
	}
	return out
}

// keyRun runs the platform with a window, puts the window in front, and runs the steps one after
// the other on the UI thread, gap apart.
func keyRun(t *testing.T, answer bool, cfg platform.WindowConfig, steps ...func(w *window, k *keyRec)) (*Platform, *keyRec, *window) {
	t.Helper()
	p, r := newTestPlatform(t)
	k := &keyRec{rec: r, answer: answer}
	if cfg.Size == (ui.Size{}) {
		cfg.Size = ui.Size{Width: 320, Height: 240}
	}
	w := newWin(t, p, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	p.Post(func() {
		var run func(i int)
		run = func(i int) {
			if i == len(steps) {
				cancel()
				return
			}
			steps[i](w, k)
			after(p, 150*time.Millisecond, func() { run(i + 1) })
		}
		focusWindow(w)
		after(p, 200*time.Millisecond, func() { run(0) })
	})
	if err := p.Run(ctx, k); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatal("the test did not finish in time")
	}
	return p, k, w
}

func focusWindow(w *window) {
	win32.SetForegroundWindow(w.hwnd)
	win32.SetFocus(w.hwnd)
}

// keys sends the keys of scripted events (key:, keydown:, keyup:) with SendInput, one step after
// the other in the same call.
func keys(scripts ...string) func(*window, *keyRec) {
	return func(w *window, _ *keyRec) {
		for _, script := range scripts {
			events, err := ParseDevEvents(script)
			if err != nil {
				panic(err)
			}
			for _, ev := range events {
				w.p.runDevKey(ev)
			}
		}
	}
}

// sendScan sends a key by its scan code, as a keyboard does; the layout of the thread makes its
// virtual key.
func sendScan(scan uint16, up bool) {
	in := win32.KeyInput{Type: win32.InputKeyboard, Key: win32.KeybdInput{Scan: scan, Flags: win32.KeyEventFScanCode}}
	if up {
		in.Key.Flags |= win32.KeyEventFKeyUp
	}
	win32.SendKeyInput([]win32.KeyInput{in})
}

// releaseKeys lets go of the keys a test may have left down: the X server of the run is shared.
func releaseKeys(t *testing.T) {
	t.Cleanup(func() {
		var in []win32.KeyInput
		for _, vk := range []uint32{win32.VKShift, win32.VKControl, win32.VKMenu, win32.VKLWin} {
			in = append(in, win32.KeyInput{Type: win32.InputKeyboard, Key: win32.KeybdInput{VK: uint16(vk), Flags: win32.KeyEventFKeyUp}})
		}
		win32.SendKeyInput(in)
	})
}

func TestSendInputKeysTypeCharactersAndNavigate(t *testing.T) {
	releaseKeys(t)
	// The framework does not handle the keys: their characters are what it would type.
	_, k, _ := keyRun(t, false, platform.WindowConfig{},
		keys("key:H"), keys("key:SHIFT+I"), keys("key:BACK"), keys("key:DELETE"), keys("key:LEFT"),
		keys("key:HOME"), keys("key:END"), keys("key:PAGEDOWN"), keys("key:RETURN"), keys("key:SPACE"),
		keys("key:CONTROL+A"), keys("key:ESC"),
	)
	const (
		logH, logI, logA = 0x68, 0x69, 0x61
		logBack          = 0x100000008
		logDelete        = 0x10000007f
		logLeft          = 0x100000302
		logHome          = 0x100000306
		logEnd           = 0x100000305
		logPageDown      = 0x100000307
		logReturn        = 0x10000000d
		logSpace         = 0x20
		logEscape        = 0x10000001b
		logShiftLeft     = 0x200000102
		logControlLeft   = 0x200000100
	)
	want := []kd{
		{down, logH, "h", false}, {up, logH, "", false},
		{down, logShiftLeft, "", false}, {down, logI, "I", false}, {up, logI, "", false}, {up, logShiftLeft, "", false},
		{down, logBack, "", false}, {up, logBack, "", false},
		{down, logDelete, "", false}, {up, logDelete, "", false},
		{down, logLeft, "", false}, {up, logLeft, "", false},
		{down, logHome, "", false}, {up, logHome, "", false},
		{down, logEnd, "", false}, {up, logEnd, "", false},
		{down, logPageDown, "", false}, {up, logPageDown, "", false},
		{down, logReturn, "", false}, {up, logReturn, "", false},
		{down, logSpace, " ", false}, {up, logSpace, "", false},
		{down, logControlLeft, "", false}, {down, logA, "", false}, {up, logA, "", false}, {up, logControlLeft, "", false},
		{down, logEscape, "", false}, {up, logEscape, "", false},
	}
	got := k.mainOnly()
	if len(got) != len(want) {
		t.Fatalf("%d key data, want %d:\n got  %v\n want %v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("key data %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	for _, d := range k.keys {
		if d.Synthesized {
			t.Errorf("a key event was synthesized: %v", d)
		}
	}
}

func TestSendInputKeyRepeatAndHandledKeys(t *testing.T) {
	releaseKeys(t)
	// Two key downs with no key up between them are an auto-repeat: the second one has bit 30 set.
	_, k, _ := keyRun(t, true, platform.WindowConfig{},
		keys("keydown:A"), keys("keydown:A"), keys("keydown:A"), keys("keyup:A"),
	)
	got := k.mainOnly()
	want := []kd{{down, 0x61, "a", false}, {repeat, 0x61, "a", false}, {repeat, 0x61, "a", false}, {up, 0x61, "", false}}
	if len(got) != len(want) {
		t.Fatalf("key data %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("key data %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestSendInputLayoutGoalsUSAndGerman(t *testing.T) {
	releaseKeys(t)
	const scanY, scanZ = 0x15, 0x2c // the keys of Y and Z on a US keyboard
	type result struct {
		y, z []kd
	}
	var us, de result
	var loaded uintptr
	var previous uintptr
	_, k, _ := keyRun(t, false, platform.WindowConfig{},
		func(w *window, k *keyRec) {
			sendScan(scanY, false)
			sendScan(scanY, true)
			sendScan(scanZ, false)
			sendScan(scanZ, true)
		},
		func(w *window, k *keyRec) {
			us = result{y: k.mainOnly()[:2], z: k.mainOnly()[2:4]}
			k.keys = nil
			// 00000407 is German (Germany).
			loaded = win32.LoadKeyboardLayout("00000407", win32.KLFActivate)
			if loaded != 0 {
				previous = win32.ActivateKeyboardLayout(loaded, 0)
			}
		},
		func(w *window, k *keyRec) {
			if loaded == 0 {
				return
			}
			sendScan(scanY, false)
			sendScan(scanY, true)
			sendScan(scanZ, false)
			sendScan(scanZ, true)
		},
		func(w *window, k *keyRec) {
			if loaded != 0 && len(k.mainOnly()) >= 4 {
				de = result{y: k.mainOnly()[:2], z: k.mainOnly()[2:4]}
			}
			if previous != 0 {
				win32.ActivateKeyboardLayout(previous, 0)
			}
		},
	)
	_ = k
	// On the US layout the key at the place of Y makes y, the one at the place of Z makes z.
	if us.y[0].log != 0x79 || us.y[0].text != "y" || us.z[0].log != 0x7a || us.z[0].text != "z" {
		t.Errorf("US layout: the key of Y = %+v, the key of Z = %+v; want y and z", us.y, us.z)
	}
	if loaded == 0 {
		t.Skip("Wine could not load the German keyboard layout (LoadKeyboardLayoutW 00000407 returned 0): the layout goals on German are tested over recorded answers in keyboard_test.go")
	}
	if len(de.y) == 0 {
		t.Skipf("the German layout loaded (%#x) but the keys did not arrive", loaded)
	}
	t.Logf("MapVirtualKeyEx(0x15, VSC_TO_VK_EX, German) = %#x; MapVirtualKeyEx(0x2c, ...) = %#x", win32.MapVirtualKeyEx(0x15, win32.MapVKVSCToVKEx, loaded), win32.MapVirtualKeyEx(0x2c, win32.MapVKVSCToVKEx, loaded))
	if de.y[0].log == 0x79 && de.z[0].log == 0x7a {
		t.Skipf("Wine loaded the German layout (%#x) but its keys still make the US characters (the key of Y gives %+v): the German logical keys are tested over recorded answers in keyboard_test.go", loaded, de.y)
	}
	if de.y[0].log != 0x7a || de.y[0].text != "z" || de.z[0].log != 0x79 || de.z[0].text != "y" {
		t.Errorf("German layout: the key of Y = %+v, the key of Z = %+v; want z and y", de.y, de.z)
	}
}

func TestAltAloneDoesNotOpenTheSystemMenuAndAltF4Closes(t *testing.T) {
	releaseKeys(t)
	var inMenuAfterAlt, inMenuAfterAltSpace bool
	p, k, w := keyRun(t, true, platform.WindowConfig{},
		// Alt alone, with the framework handling it.
		keys("keydown:MENU"), keys("keyup:MENU"),
		func(w *window, k *keyRec) { inMenuAfterAlt = w.modalTimer },
		// Alt alone with nobody handling it does not open the menu either: the release is never passed on.
		func(w *window, k *keyRec) { k.answer = false },
		keys("keydown:MENU"), keys("keyup:MENU"),
		func(w *window, k *keyRec) { inMenuAfterAlt = inMenuAfterAlt || w.modalTimer },
		// Alt+Space, unhandled, opens the system menu (Windows' own behavior): the control for the test above.
		keys("keydown:MENU", "key:SPACE"),
		func(w *window, k *keyRec) { inMenuAfterAltSpace = w.modalTimer },
		// Escape closes the menu and the Alt key goes up.
		keys("key:ESC", "keyup:MENU"),
		// Alt+F4 closes the window even when the framework says it handled the key.
		func(w *window, k *keyRec) { k.answer = true },
		keys("keydown:MENU", "key:F4", "keyup:MENU"),
	)
	_ = p
	if inMenuAfterAlt {
		t.Error("Alt alone opened the system menu")
	}
	if !inMenuAfterAltSpace {
		t.Log("Alt+Space did not open the system menu under Wine, so the control for \"Alt alone\" shows nothing")
	}
	if len(k.closes) != 1 || k.closes[0] != w.id {
		t.Errorf("Alt+F4 made the close requests %v, want one for the window", k.closes)
	}
	var sawF4 bool
	for _, d := range k.keys {
		if d.Logical == 0x100000804 && d.Type == ui.KeyEventTypeDown {
			sawF4 = true
		}
	}
	if !sawF4 {
		t.Error("the framework did not get the key down of F4")
	}
}

func TestKeysHeldAreReleasedWhenTheFocusIsLost(t *testing.T) {
	releaseKeys(t)
	_, k, _ := keyRun(t, false, platform.WindowConfig{},
		keys("keydown:SHIFT", "keydown:A"),
		func(w *window, k *keyRec) { k.keys = nil; win32.SetFocus(0) },
	)
	got := k.brief()
	want := []kd{{up, 0x61, "", true}, {up, 0x200000102, "", true}}
	if len(got) != len(want) {
		t.Fatalf("after the loss of focus: %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestLockStatesAreSeenWhenTheFocusReturns(t *testing.T) {
	releaseKeys(t)
	// A key pressed while the window has no focus never reaches it. Wine delivers such keys to the
	// active window anyway, so the change of the lock states is made in the state table of the
	// thread (SetKeyboardState), which is what GetKeyState reads.
	flip := func(vks ...uint32) {
		state, ok := win32.GetKeyboardState()
		if !ok {
			t.Fatal("GetKeyboardState failed")
		}
		for _, vk := range vks {
			state[vk] ^= stateMaskToggled
		}
		if !win32.SetKeyboardState(state) {
			t.Fatal("SetKeyboardState failed")
		}
	}
	_, k, _ := keyRun(t, false, platform.WindowConfig{},
		func(w *window, k *keyRec) { win32.SetFocus(0); k.keys = nil },
		func(w *window, k *keyRec) { flip(win32.VKCapital, win32.VKNumLock) },
		func(w *window, k *keyRec) {
			if k.keys != nil {
				t.Errorf("a window without focus got key data: %v", k.brief())
			}
			win32.SetFocus(w.hwnd)
		},
		func(w *window, k *keyRec) { flip(win32.VKCapital, win32.VKNumLock) }, // put them back
	)
	got := k.brief()
	const logCaps, logNum = 0x100000104, 0x10000010a
	// The toggled states are synchronized first, then the pressed ones, with the keys in the order of
	// their virtual keys: Caps Lock (0x14) before Num Lock (0x90).
	want := []kd{{down, logCaps, "", true}, {down, logNum, "", true}, {up, logCaps, "", true}, {up, logNum, "", true}}
	if len(got) != len(want) {
		t.Fatalf("after the focus returned: %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestDevEventsKeysReachTheFramework(t *testing.T) {
	releaseKeys(t)
	script := "wait:100ms;key:SHIFT+A;key:ctrl+left;keydown:SHIFT;key:RIGHT;keyup:SHIFT"
	p, r := newTestPlatform(t, Options{Getenv: func(k string) string {
		if k == DevEventsEnv {
			return script
		}
		return ""
	}})
	k := &keyRec{rec: r, answer: false}
	newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r.onFrame = func(ui.ViewID) { p.startDevEvents() }
	go func() {
		select {
		case <-p.devDone:
			time.Sleep(500 * time.Millisecond)
			cancel()
		case <-ctx.Done():
		}
	}()
	if err := p.Run(ctx, k); err != nil {
		t.Fatal(err)
	}
	var typed string
	for _, d := range k.keys {
		if d.Type == ui.KeyEventTypeDown && !d.Synthesized {
			typed += d.Character
		}
	}
	if typed != "A" {
		t.Errorf("the characters of the scripted keys are %q, want \"A\"", typed)
	}
	if len(k.mainOnly()) != 12 {
		t.Errorf("%d key data from the script, want 12 (6 downs and 6 ups): %v", len(k.mainOnly()), k.mainOnly())
	}
}
