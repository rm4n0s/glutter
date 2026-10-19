package widgets_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
)

func TestBindingFeedsTheEnginesKeysToTheHardwareKeyboard(t *testing.T) {
	ts := gluttertest.New(t)
	k := ts.Binding.Keyboard()
	var got []string
	k.AddHandler(func(e services.KeyEvent) bool {
		got = append(got, strings.SplitN(e.(interface{ String() string }).String(), "(", 2)[0])
		return e.LogicalKey() == services.LogicalKeyboardKeyKeyA
	})

	if !ts.SendKeyDownEvent(services.LogicalKeyboardKeyKeyA, gluttertest.KeyOpts{Character: "a"}) {
		t.Error("the handler's answer did not reach the engine")
	}
	if !k.IsPhysicalKeyPressed(services.PhysicalKeyboardKeyKeyA) || !k.IsLogicalKeyPressed(services.LogicalKeyboardKeyKeyA) {
		t.Error("A should be pressed")
	}
	ts.SendKeyRepeatEvent(services.LogicalKeyboardKeyKeyA)
	ts.SendKeyDownEvent(services.LogicalKeyboardKeyCapsLock)
	if !k.LockModesEnabled().Contains(services.KeyboardLockModeCapsLock) {
		t.Error("caps lock should be on")
	}
	ts.SendKeyUpEvent(services.LogicalKeyboardKeyKeyA)
	if k.IsLogicalKeyPressed(services.LogicalKeyboardKeyKeyA) {
		t.Error("A should be released")
	}
	want := []string{"KeyDownEvent", "KeyRepeatEvent", "KeyDownEvent", "KeyUpEvent"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("events = %v, want %v", got, want)
	}
	if err := ts.TakeError(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
