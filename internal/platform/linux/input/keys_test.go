package input

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/ui"
)

func keyDown(physical, logical uint64, char string) ui.KeyData {
	return ui.KeyData{Type: ui.KeyEventTypeDown, DeviceType: ui.KeyEventDeviceTypeKeyboard, Physical: physical, Logical: logical, Character: char}
}

func TestKeyTrackerUp(t *testing.T) {
	var k KeyTracker
	if _, ok := k.Up(30, time.Second); ok {
		t.Fatal("an up for a key that is not down")
	}
	if !k.Down(30, keyDown(0x70004, 'a', "A")) {
		t.Fatal("Down refused a new key")
	}
	if k.Down(30, keyDown(0x70004, 'a', "A")) {
		t.Fatal("Down accepted a key that is down")
	}
	if d, ok := k.Data(30); !ok || d.Character != "A" {
		t.Errorf("Data = %+v, %v", d, ok)
	}
	up, ok := k.Up(30, 5*time.Second)
	if !ok {
		t.Fatal("no up for a key that is down")
	}
	want := ui.KeyData{TimeStamp: 5 * time.Second, Type: ui.KeyEventTypeUp, DeviceType: ui.KeyEventDeviceTypeKeyboard, Physical: 0x70004, Logical: 'a'}
	if up != want {
		t.Errorf("up = %+v, want %+v", up, want)
	}
	if _, ok := k.Data(30); ok {
		t.Error("the key is still down after its up")
	}
}

func TestKeyTrackerReleaseAllKeepsTheOrder(t *testing.T) {
	var k KeyTracker
	k.Down(42, keyDown(0x700e1, 0x200000102, ""))
	k.Down(30, keyDown(0x70004, 'a', "A"))
	k.Down(31, keyDown(0x70016, 's', "S"))
	k.Up(30, time.Second)
	got := k.ReleaseAll(2 * time.Second)
	if len(got) != 2 || got[0].Physical != 0x700e1 || got[1].Physical != 0x70016 {
		t.Fatalf("got %+v", got)
	}
	for _, d := range got {
		if d.Type != ui.KeyEventTypeUp || !d.Synthesized || d.Character != "" || d.TimeStamp != 2*time.Second {
			t.Errorf("released key: %+v", d)
		}
	}
	if len(k.ReleaseAll(3*time.Second)) != 0 {
		t.Error("keys are left after ReleaseAll")
	}
}
