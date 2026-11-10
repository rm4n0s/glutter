//go:build linux && !android

package evdev

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// asleepFixture is a fixture with a PlayStation 4 pad on USB (event0, hidraw3) and a pad that can rumble (event1), and a service
// whose extras are asleep.
func asleepFixture(t *testing.T) *hidFixture {
	t.Helper()
	f := newHIDFixture(t, 40*time.Millisecond)
	f.s.asleep.Store(true)
	f.s.k.writable = func(string) bool { return !f.readOnly }
	f.writeSysfs("event0", dualShock4Usb())
	f.writeNode("event0")
	f.withHidraw("event0", "hidraw3")
	f.writeSysfs("event1", rumblePad())
	f.writeNode("event1")
	return f
}

func TestAnAsleepServiceOnlyReads(t *testing.T) {
	f := asleepFixture(t)
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "two pads", func() bool { return len(f.h.added) == 2 })
	if tr := f.transport("hidraw3"); tr != nil {
		t.Error("the hidraw node was opened although the extras are asleep")
	}
	for _, info := range f.h.added {
		if info.Standard {
			t.Errorf("%s is served by a HID driver although the extras are asleep", info.Name)
		}
		if filepath.Base(info.Path) == "event1" && !info.Rumble {
			t.Error("a pad that can rumble says it cannot: the first rumble opens it for writing")
		}
	}
	if got := f.ffLog(); len(got) != 0 {
		t.Errorf("a pad was opened for writing before any rumble: %v", got)
	}
}

func TestTheFirstRumbleOpensTheDeviceForWriting(t *testing.T) {
	f := asleepFixture(t)
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "two pads", func() bool { return len(f.h.added) == 2 })
	var id platform.PadID
	for _, info := range f.h.added {
		if filepath.Base(info.Path) == "event1" {
			id = info.ID
		}
	}
	if err := f.s.Rumble(id, 0x8000, 0x4000, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := f.s.Rumble(id, 1, 2, time.Second); err != nil {
		t.Fatal(err)
	}
	want := []string{"open event1", "upload 7", "play 7 1", "upload 7", "play 7 1"}
	if got := f.ffLog(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("force feedback calls = %v, want %v (one open, at the first rumble)", got, want)
	}
}

func TestAnAsleepPadThatMayNotBeWrittenToCannotRumble(t *testing.T) {
	f := asleepFixture(t)
	f.readOnly = true
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "two pads", func() bool { return len(f.h.added) == 2 })
	for _, info := range f.h.added {
		if filepath.Base(info.Path) == "event1" {
			if info.Rumble {
				t.Error("a pad that the user may not write to says it can rumble")
			}
			err := f.s.Rumble(info.ID, 1, 1, time.Second)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != platform.TagGamepadRumbleUnavailable {
				t.Errorf("error %v, want GamepadRumbleUnavailable", err)
			}
		}
	}
	if got := f.ffLog(); len(got) != 0 {
		t.Errorf("an open for writing was tried: %v", got)
	}
}

func TestWakingTheExtrasMovesAPadToItsDriver(t *testing.T) {
	f := asleepFixture(t)
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "two pads", func() bool { return len(f.h.added) == 2 })
	var before platform.PadInfo
	for _, info := range f.h.added {
		if filepath.Base(info.Path) == "event0" {
			before = info
		}
	}
	f.s.EnableExtras()
	f.s.EnableExtras() // the second does nothing
	f.h.wait(t, "the pad moves to its driver", func() bool { return len(f.h.added) == 3 && len(f.h.gone) == 1 })
	f.h.mu.Lock()
	after := f.h.added[2]
	gone := f.h.gone[0]
	f.h.mu.Unlock()
	if gone != before.ID {
		t.Errorf("the pad that was removed is %d, want %d", gone, before.ID)
	}
	if !after.Standard || filepath.Base(after.Path) != "hidraw3" || !after.LED || after.ID == before.ID {
		t.Errorf("the pad after the wake-up = %+v", after)
	}
	if tr := f.transport("hidraw3"); tr == nil {
		t.Error("the hidraw node was not opened after the wake-up")
	}
	// The pad that no driver serves is left alone: no second removal and no second add.
	f.h.settle()
	f.h.mu.Lock()
	n := len(f.h.added)
	f.h.mu.Unlock()
	if n != 3 {
		t.Errorf("%d pads were added, want 3: the pad without a driver was touched", n)
	}
	// A driver that works takes the rumble.
	if err := f.s.Rumble(after.ID, 0xFFFF, 0, time.Minute); err != nil {
		t.Errorf("the driver does not rumble: %v", err)
	}
}

func TestWakingBeforeStartStartsAwake(t *testing.T) {
	f := asleepFixture(t)
	f.s.EnableExtras()
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "the pads", func() bool { return len(f.h.added) == 2 })
	if tr := f.transport("hidraw3"); tr == nil {
		t.Error("the driver did not start although the extras were woken before Start")
	}
	f.h.mu.Lock()
	defer f.h.mu.Unlock()
	if len(f.h.gone) != 0 {
		t.Errorf("a pad was removed: %v", f.h.gone)
	}
}

func TestWakingAnAsleepServiceThatIsNotRunningIsHarmless(t *testing.T) {
	f := asleepFixture(t)
	f.s.EnableExtras() // no run: the flag is all that changes
	f.s.Stop()
	f.s.EnableExtras()
}

func TestTheFirstRumbleThatCannotOpenTheDeviceFails(t *testing.T) {
	f := asleepFixture(t)
	if err := f.s.Start(f.h); err != nil {
		t.Fatal(err)
	}
	f.h.wait(t, "two pads", func() bool { return len(f.h.added) == 2 })
	var id platform.PadID
	for _, info := range f.h.added {
		if filepath.Base(info.Path) == "event1" {
			id = info.ID
		}
	}
	f.readOnly = true // the permission was taken away after the pad was seen
	err := f.s.Rumble(id, 1, 1, time.Second)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != platform.TagGamepadRumbleUnavailable || !appErr.HasRoute("Service.Rumble.GamepadRumbleUnavailable") {
		t.Fatalf("error %v, want GamepadRumbleUnavailable from Service.Rumble", err)
	}
	// The pad can still rumble when the permission comes back: the open is tried again at the next rumble.
	f.readOnly = false
	if err := f.s.Rumble(id, 1, 1, time.Second); err != nil {
		t.Errorf("the second rumble failed: %v", err)
	}
}
