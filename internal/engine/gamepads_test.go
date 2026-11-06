package engine

import (
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/fake"
	"github.com/rm4n0s/glutter/ui"
)

// padKeys runs the engine with a window and a standard pad, plays script, and returns the key events that the app got.
func padKeys(t *testing.T, focused bool, want int, script func(pads *fake.Gamepads, id platform.PadID)) []ui.KeyData {
	t.Helper()
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	var mu sync.Mutex
	var keys []ui.KeyData
	h.cfg.Hooks.Key = func(d ui.KeyData) { mu.Lock(); keys = append(keys, d); mu.Unlock() }
	h.c.Inject(metrics(640, 400))
	h.c.Inject(fake.FrameEvent{View: 0, FrameTime: time.Millisecond})
	if focused {
		h.c.Inject(fake.ViewFocusEvent{View: 0, Focused: true})
	}
	done := h.run(t)
	pads := h.c.Gamepads()
	waitFor(t, func() bool { return pads.Starts() == 1 })
	id := pads.Connect(platform.PadInfo{Name: "Fake", Standard: true, Battery: -1})
	script(pads, id)
	// The input is handled by the next frame: give frames until the keys are there (or, when none is wanted, a good number).
	for i := 0; i < 400; i++ {
		mu.Lock()
		n := len(keys)
		mu.Unlock()
		if want > 0 && n >= want {
			break
		}
		h.c.Inject(fake.FrameEvent{View: 0, FrameTime: time.Duration(i+2) * time.Millisecond})
		time.Sleep(2 * time.Millisecond)
	}
	h.stopEngine(t)
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	return append([]ui.KeyData(nil), keys...)
}

func TestAPadMakesKeyEventsForTheFocusedWindow(t *testing.T) {
	keys := padKeys(t, true, 4, func(pads *fake.Gamepads, id platform.PadID) {
		// South goes down and up, then the D-pad up (button 11 of the standard layout), then the right trigger.
		pads.Send(
			platform.PadEvent{Pad: id, Kind: platform.PadButton, Index: 0, Value: 1},
			platform.PadEvent{Pad: id, Kind: platform.PadButton, Index: 0, Value: 0},
			platform.PadEvent{Pad: id, Kind: platform.PadButton, Index: 11, Value: 1},
			platform.PadEvent{Pad: id, Kind: platform.PadAxis, Index: 5, Value: 20000},
		)
	})
	if len(keys) != 4 {
		t.Fatalf("got %d key events, want 4: %+v", len(keys), keys)
	}
	want := []struct {
		typ      ui.KeyEventType
		dev      ui.KeyEventDeviceType
		logical  uint64
		physical uint64
	}{
		{ui.KeyEventTypeDown, ui.KeyEventDeviceTypeGamepad, 0x200000311, 0x0005ff11},
		{ui.KeyEventTypeUp, ui.KeyEventDeviceTypeGamepad, 0x200000311, 0x0005ff11},
		{ui.KeyEventTypeDown, ui.KeyEventDeviceTypeDirectionalPad, 0x100000304, 0x00070052},
		{ui.KeyEventTypeDown, ui.KeyEventDeviceTypeGamepad, 0x200000318, 0x0005ff18},
	}
	for i, w := range want {
		k := keys[i]
		if k.Type != w.typ || k.DeviceType != w.dev || k.Logical != w.logical || k.Physical != w.physical {
			t.Errorf("key %d = %+v, want %+v", i, k, w)
		}
		if k.TimeStamp <= 0 {
			t.Errorf("key %d has no time stamp: %v", i, k.TimeStamp)
		}
	}
}

func TestAPadMakesNoKeyEventsForAWindowThatIsNotFocused(t *testing.T) {
	keys := padKeys(t, false, 0, func(pads *fake.Gamepads, id platform.PadID) {
		pads.Send(platform.PadEvent{Pad: id, Kind: platform.PadButton, Index: 0, Value: 1})
	})
	if len(keys) != 0 {
		t.Errorf("an app behind others got keys: %+v", keys)
	}
}

func TestThePadManagerIsFoundThroughTheHost(t *testing.T) {
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	done := h.run(t)
	waitFor(t, func() bool { return h.c.Gamepads().Starts() == 1 })
	h.stopEngine(t)
	if err := result(t, done); err != nil {
		t.Fatal(err)
	}
	if h.c.Gamepads().Starts() != 1 {
		t.Errorf("the service was started %d times", h.c.Gamepads().Starts())
	}
}
