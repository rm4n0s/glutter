package engine

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/pads"
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

// padsRun runs the engine in a gamepad mode, with the dispatcher kept so that the test can say that the app uses the pads, and
// returns what the service saw. use is called on the UI thread once the engine has started.
type padsRun struct {
	h    *harness
	disp chan *ui.PlatformDispatcher
	done <-chan error
}

func startPadsRun(t *testing.T, option string, env map[string]string) *padsRun {
	t.Helper()
	e := map[string]string{"DISPLAY": ":0"}
	for k, v := range env {
		e[k] = v
	}
	r := &padsRun{h: newHarness(t, e), disp: make(chan *ui.PlatformDispatcher, 1)}
	r.h.cfg.Gamepads = option
	r.h.cfg.Start = func(d *ui.PlatformDispatcher) error { r.disp <- d; return nil }
	r.done = r.h.run(t)
	return r
}

// use says, on the UI thread, that the app uses package gamepad: what gamepad.ForBinding does.
func (r *padsRun) use(t *testing.T) {
	t.Helper()
	d := <-r.disp
	r.disp <- d
	ran := make(chan struct{})
	r.h.c.Post(func() {
		var host pads.Host
		if d.HostAs(&host) {
			host.UsePads()
		}
		close(ran)
	})
	<-ran
}

func (r *padsRun) managerIsNil(t *testing.T) bool {
	t.Helper()
	d := <-r.disp
	r.disp <- d
	var isNil bool
	ran := make(chan struct{})
	r.h.c.Post(func() {
		var host pads.Host
		isNil = !d.HostAs(&host) || host.PadManager() == nil
		close(ran)
	})
	<-ran
	return isNil
}

func (r *padsRun) stop(t *testing.T) {
	t.Helper()
	r.h.stopEngine(t)
	if err := result(t, r.done); err != nil {
		t.Fatal(err)
	}
}

func TestTheGamepadModes(t *testing.T) {
	tests := []struct {
		name            string
		option          string
		env             map[string]string
		startsAtOnce    int  // the starts of the service before the app uses the pads
		managerAtOnce   bool // there is a manager before the app uses the pads
		startsAfterUse  int
		extrasAfterUse  int
		extrasAtStart   int // how many wake-ups there were when the service started, after the use
		managerAfterUse bool
	}{
		{"auto by default: it starts, asleep, and wakes at the first use", "", nil, 1, true, 1, 1, 0, true},
		{"auto by name", "auto", nil, 1, true, 1, 1, 0, true},
		{"on demand starts nothing until the use, and then starts awake", "ondemand", nil, 0, true, 1, 1, 1, true},
		{"off never starts it", "off", nil, 0, false, 0, 0, 0, false},
		{"the variable wins: off", "auto", map[string]string{GamepadsEnv: "off"}, 0, false, 0, 0, 0, false},
		{"the variable wins: on demand", "", map[string]string{GamepadsEnv: "ondemand"}, 0, true, 1, 1, 1, true},
		{"the variable wins: auto", "off", map[string]string{GamepadsEnv: "auto"}, 1, true, 1, 1, 0, true},
		{"a variable that is no mode is ignored", "off", map[string]string{GamepadsEnv: "yes"}, 0, false, 0, 0, 0, false},
		{"a variable that is no mode leaves auto", "", map[string]string{GamepadsEnv: "yes"}, 1, true, 1, 1, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := startPadsRun(t, tc.option, tc.env)
			pads := r.h.c.Gamepads()
			// The engine starts the gamepads after the framework starts: wait for the framework, then give it time to not start them.
			d := <-r.disp
			r.disp <- d
			if tc.startsAtOnce > 0 {
				waitFor(t, func() bool { return pads.Starts() == tc.startsAtOnce })
			} else {
				time.Sleep(100 * time.Millisecond)
			}
			if got := pads.Starts(); got != tc.startsAtOnce {
				t.Errorf("before the use the service was started %d times, want %d", got, tc.startsAtOnce)
			}
			if got := pads.Extras(); got != 0 {
				t.Errorf("before the use the extras were woken %d times, want 0", got)
			}
			r.use(t)
			r.use(t) // the second use changes nothing
			if got := pads.Starts(); got != tc.startsAfterUse {
				t.Errorf("after the use the service was started %d times, want %d", got, tc.startsAfterUse)
			}
			if got := pads.Extras(); got != tc.extrasAfterUse {
				t.Errorf("after the use the extras were woken %d times, want %d", got, tc.extrasAfterUse)
			}
			if tc.startsAfterUse > 0 && pads.ExtrasAtStart() != tc.extrasAtStart {
				t.Errorf("the service started after %d wake-ups, want %d", pads.ExtrasAtStart(), tc.extrasAtStart)
			}
			if nilAfter := r.managerIsNil(t); nilAfter == tc.managerAfterUse {
				t.Errorf("a manager after the use is %v, want %v", !nilAfter, tc.managerAfterUse)
			}
			r.stop(t)
		})
	}
}

func TestAnUnknownGamepadModeIsAnError(t *testing.T) {
	h := newHarness(t, map[string]string{"DISPLAY": ":0"})
	h.cfg.Gamepads = "sometimes"
	err := <-h.run(t)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "GamepadsModeUnknown" || !appErr.HasRoute("gamepadModeOf.GamepadsModeUnknown") {
		t.Fatalf("error %v, want GamepadsModeUnknown from gamepadModeOf", err)
	}
	if h.c.Gamepads().Starts() != 0 {
		t.Error("the service started although the mode is unknown")
	}
}

func TestGamepadModeOfLogsAnUnknownVariable(t *testing.T) {
	var logged string
	log := slog.New(slog.NewTextHandler(writerFunc(func(p []byte) { logged += string(p) }), nil))
	mode, err := gamepadModeOf("off", func(string) string { return "banana" }, log)
	if err != nil || mode != GamepadsOff {
		t.Fatalf("mode %q, %v; want off", mode, err)
	}
	if logged == "" {
		t.Error("an unknown value of the variable was not logged")
	}
}

type writerFunc func([]byte)

func (f writerFunc) Write(p []byte) (int, error) { f(p); return len(p), nil }
