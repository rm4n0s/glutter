//go:build glutter_android

package androidtest

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestGamepadKeysActivateAndMoveTheFocus sends the keys of a gamepad with `adb shell input gamepad keyevent` (SPEC §28.8): the
// shell's pad is a virtual device with the id of the virtual keyboard, which the Java side makes a pad of when its first key comes.
// The A button activates the button that has the focus, once (a pad's key goes to the gamepad service only, not to the keyboard as
// well, which would make it twice), the D-pad moves the focus to the next button, and A activates that one.
func TestGamepadKeysActivateAndMoveTheFocus(t *testing.T) {
	e := Shared(t, "glutter-37")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	appID := inputApp(ctx, t, e, "./examples/dev/android-gamepad", nil)

	// The first key of a session only takes the window out of touch mode on the emulator's image (no key event reaches the app,
	// whatever its source), so a first press comes before the one that is counted.
	shell(ctx, t, e, "input", "gamepad", "keyevent", "KEYCODE_BUTTON_A")
	time.Sleep(2 * time.Second)
	shell(ctx, t, e, "input", "gamepad", "keyevent", "KEYCODE_BUTTON_A")
	log, err := e.WaitLog(ctx, appID, "button pressed index=0 count=1", 20*time.Second)
	if err != nil {
		t.Fatal(Describe(err))
	}
	if !strings.Contains(log, "gamepad connected") {
		t.Errorf("the pad that the first key made was not announced:\n%s", log)
	}

	shell(ctx, t, e, "input", "gamepad", "keyevent", "KEYCODE_DPAD_DOWN")
	time.Sleep(500 * time.Millisecond)
	shell(ctx, t, e, "input", "gamepad", "keyevent", "KEYCODE_BUTTON_A")
	log, err = e.WaitLog(ctx, appID, "button pressed index=1 count=1", 20*time.Second)
	if err != nil {
		t.Fatal(Describe(err))
	}
	// Each press arrived once: the first button was activated once, the second once, and no other.
	for _, bad := range []string{"index=0 count=2", "index=1 count=2"} {
		if strings.Contains(log, "button pressed "+bad) {
			t.Errorf("a press arrived twice (%q in the log):\n%s", bad, log)
		}
	}
}
