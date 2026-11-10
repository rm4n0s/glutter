package iokit

import (
	"fmt"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
)

// The layout of the framework's structs on a 64-bit machine, as the C compiler gave it (sizeof and offsetof of FFEFFECT and
// FFPERIODIC from ForceFeedback.h).
func TestFFStructsHaveTheFrameworksLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("the layout is the 64-bit one")
	}
	var e ffEffect
	var p ffPeriodic
	if unsafe.Sizeof(e) != 80 || unsafe.Sizeof(p) != 16 {
		t.Errorf("sizes: FFEFFECT %d (want 80), FFPERIODIC %d (want 16)", unsafe.Sizeof(e), unsafe.Sizeof(p))
	}
	for name, got := range map[string][2]uintptr{
		"dwSize": {unsafe.Offsetof(e.Size), 0}, "dwFlags": {unsafe.Offsetof(e.Flags), 4}, "dwDuration": {unsafe.Offsetof(e.Duration), 8},
		"dwSamplePeriod": {unsafe.Offsetof(e.SamplePeriod), 12}, "dwGain": {unsafe.Offsetof(e.Gain), 16},
		"dwTriggerButton": {unsafe.Offsetof(e.TriggerButton), 20}, "dwTriggerRepeatInterval": {unsafe.Offsetof(e.TriggerRepeatInterval), 24},
		"cAxes": {unsafe.Offsetof(e.Axes), 28}, "rgdwAxes": {unsafe.Offsetof(e.AxesList), 32}, "rglDirection": {unsafe.Offsetof(e.Directions), 40},
		"lpEnvelope": {unsafe.Offsetof(e.Envelope), 48}, "cbTypeSpecificParams": {unsafe.Offsetof(e.TypeSpecificSize), 56},
		"lpvTypeSpecificParams": {unsafe.Offsetof(e.TypeSpecific), 64}, "dwStartDelay": {unsafe.Offsetof(e.StartDelay), 72},
		"dwMagnitude": {unsafe.Offsetof(p.Magnitude), 0}, "lOffset": {unsafe.Offsetof(p.Offset), 4}, "dwPhase": {unsafe.Offsetof(p.Phase), 8},
		"dwPeriod": {unsafe.Offsetof(p.Period), 12},
	} {
		if got[0] != got[1] {
			t.Errorf("offset of %s is %d, the framework's is %d", name, got[0], got[1])
		}
	}
}

func TestFFMagnitudeAndDuration(t *testing.T) {
	for _, tc := range []struct {
		low, high uint16
		want      uint32
	}{{0, 0, 0}, {65535, 65535, 10000}, {65535, 0, 4999}, {0, 65535, 4999}, {1, 1, 0}, {32768, 32768, 5000}} {
		if got := ffMagnitude(tc.low, tc.high); got != tc.want {
			t.Errorf("ffMagnitude(%d, %d) = %d, want %d", tc.low, tc.high, got, tc.want)
		}
	}
	for _, tc := range []struct {
		d    time.Duration
		want uint32
	}{{time.Nanosecond, 1}, {time.Millisecond, 1000}, {2 * time.Second, 2_000_000}, {time.Hour, 65_535_000}} {
		if got := ffDuration(tc.d); got != tc.want {
			t.Errorf("ffDuration(%v) = %d, want %d", tc.d, got, tc.want)
		}
	}
}

// fakeFF is an ffAPI that records its calls and fails where the test says.
type fakeFF struct {
	calls                                                  []string
	effects                                                []ffEffect
	resetHR, actuatorsHR, createHR, setHR, startHR, stopHR int32
}

func (f *fakeFF) Command(cmd uint32) int32 {
	switch cmd {
	case ffsffcReset:
		f.calls = append(f.calls, "reset")
		return f.resetHR
	case ffsffcSetActuatorsOn:
		f.calls = append(f.calls, "actuators on")
		return f.actuatorsHR
	}
	f.calls = append(f.calls, "unknown command")
	return -1
}
func (f *fakeFF) CreateEffect(e *ffEffect) (uintptr, int32) {
	f.calls = append(f.calls, "create")
	f.effects = append(f.effects, *e)
	return 42, f.createHR
}
func (f *fakeFF) SetParameters(h uintptr, e *ffEffect, flags uint32) int32 {
	f.calls = append(f.calls, "set "+hex(flags))
	f.effects = append(f.effects, *e)
	return f.setHR
}
func (f *fakeFF) Start(uintptr) int32   { f.calls = append(f.calls, "start"); return f.startHR }
func (f *fakeFF) Stop(uintptr) int32    { f.calls = append(f.calls, "stop"); return f.stopHR }
func (f *fakeFF) ReleaseEffect(uintptr) { f.calls = append(f.calls, "release effect") }
func (f *fakeFF) ReleaseDevice()        { f.calls = append(f.calls, "release device") }

func hex(v uint32) string { return fmt.Sprintf("0x%08x", v) }

func calls(t *testing.T, f *fakeFF, want ...string) {
	t.Helper()
	if len(f.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", f.calls, want)
	}
	for i := range want {
		if f.calls[i] != want[i] {
			t.Fatalf("calls = %v, want %v", f.calls, want)
		}
	}
	f.calls = nil
}

func TestRumblerPlaysAndStops(t *testing.T) {
	f := &fakeFF{}
	r, err := newRumbler(f)
	if err != nil {
		t.Fatal(err)
	}
	calls(t, f, "reset", "actuators on")
	// Stopping before any rumble does nothing.
	if err := r.Play(0, 0, time.Second); err != nil {
		t.Fatal(err)
	}
	calls(t, f)
	// The first rumble makes the effect, with SDL's values, and plays it.
	if err := r.Play(65535, 65535, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	calls(t, f, "create", "start")
	e := f.effects[0]
	if e.Size != 80 || e.Flags != ffeffObjectOffsets|ffeffCartesian || e.Duration != 2_000_000 || e.Gain != 10000 || e.TriggerButton != ffebNoTrigger ||
		e.Axes != 2 || e.TypeSpecificSize != 16 || e.AxesList == nil || e.Directions == nil || e.TypeSpecific == nil {
		t.Errorf("effect = %+v", e)
	}
	if p := (*ffPeriodic)(e.TypeSpecific); p.Magnitude != 10000 || p.Period != 1_000_000 {
		t.Errorf("periodic = %+v", *p)
	}
	// The next rumble changes the duration and the strength of the same effect and plays it again.
	if err := r.Play(65535, 0, time.Second); err != nil {
		t.Fatal(err)
	}
	calls(t, f, "set 0x00000101", "start")
	if p := (*ffPeriodic)(f.effects[1].TypeSpecific); p.Magnitude != 4999 || f.effects[1].Duration != 1_000_000 {
		t.Errorf("the second effect = %+v, %+v", f.effects[1], *p)
	}
	// No strength, and a duration of 0, stop it.
	if err := r.Play(0, 0, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := r.Play(65535, 65535, 0); err != nil {
		t.Fatal(err)
	}
	calls(t, f, "stop", "stop")
	r.Close()
	r.Close() // twice does nothing
	calls(t, f, "release effect", "release device")
	if err := r.Play(65535, 65535, time.Second); err == nil {
		t.Error("a closed rumble played")
	}
}

func TestRumblerFailuresHaveTheirOwnTags(t *testing.T) {
	check := func(t *testing.T, err error, tag, route string) {
		t.Helper()
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tag || !appErr.HasRoute(route+"."+tag) {
			t.Errorf("error %v (route %v), want %s from %s", err, appErr.Route(), tag, route)
		}
	}
	t.Run("reset", func(t *testing.T) {
		f := &fakeFF{resetHR: -5}
		_, err := newRumbler(f)
		check(t, err, tagForceFeedbackReset, "newRumbler")
		calls(t, f, "reset", "release device")
	})
	t.Run("actuators", func(t *testing.T) {
		f := &fakeFF{actuatorsHR: -5}
		_, err := newRumbler(f)
		check(t, err, tagForceFeedbackActuators, "newRumbler")
		calls(t, f, "reset", "actuators on", "release device")
	})
	play := func(f *fakeFF) *rumbler {
		r, _ := newRumbler(f)
		f.calls = nil
		return r
	}
	t.Run("effect", func(t *testing.T) {
		f := &fakeFF{createHR: -5}
		check(t, play(f).Play(1000, 1000, time.Second), tagForceFeedbackEffect, "rumbler.Play")
	})
	t.Run("update", func(t *testing.T) {
		f := &fakeFF{}
		r := play(f)
		_ = r.Play(1000, 1000, time.Second)
		f.setHR = -5
		check(t, r.Play(2000, 2000, time.Second), tagForceFeedbackUpdate, "rumbler.Play")
	})
	t.Run("start", func(t *testing.T) {
		f := &fakeFF{startHR: -5}
		check(t, play(f).Play(1000, 1000, time.Second), tagForceFeedbackStart, "rumbler.Play")
	})
	t.Run("stop", func(t *testing.T) {
		f := &fakeFF{}
		r := play(f)
		_ = r.Play(1000, 1000, time.Second)
		f.stopHR = -5
		check(t, r.Play(0, 0, 0), tagForceFeedbackStop, "rumbler.Play")
	})
	t.Run("closed", func(t *testing.T) {
		r := play(&fakeFF{})
		r.Close()
		check(t, r.Play(1000, 1000, time.Second), tagForceFeedbackClosed, "rumbler.Play")
	})
}

// ffDevice is a fake device that has force feedback.
type ffDevice struct {
	*fakeDevice
	has     bool
	api     *fakeFF
	openErr error
	opened  int
	rumble  *rumbler
}

func (d *ffDevice) HasForceFeedback() bool { return d.has }
func (d *ffDevice) OpenForceFeedback() (ForceFeedback, error) {
	d.opened++
	if d.openErr != nil {
		return nil, d.openErr
	}
	r, err := newRumbler(d.api)
	d.rumble = r
	return r, err
}

func TestServiceRumblesAPadThroughForceFeedback(t *testing.T) {
	f := newFixture(t, true)
	api := &fakeFF{}
	dev := &ffDevice{fakeDevice: genericPad(), has: true, api: api}
	f.start()
	f.m.cb.Added(dev)
	if len(f.h.added) != 1 || !f.h.added[0].Rumble {
		t.Fatalf("added = %+v: the pad must say it can rumble", f.h.added)
	}
	id := f.h.added[0].ID
	if err := f.s.Rumble(id, 65535, 65535, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := f.s.Rumble(id, 1000, 1000, time.Second); err != nil {
		t.Fatal(err)
	}
	if dev.opened != 1 {
		t.Errorf("the device was opened %d times, want 1", dev.opened)
	}
	calls(t, api, "reset", "actuators on", "create", "start", "set 0x00000101", "start")
	// The pad that goes closes its rumble.
	f.m.cb.Removed(dev)
	calls(t, api, "release effect", "release device")
	if err := f.s.Rumble(id, 1, 1, time.Second); err == nil {
		t.Error("a pad that is gone rumbled")
	}
}

func TestServiceStopClosesTheRumble(t *testing.T) {
	f := newFixture(t, true)
	api := &fakeFF{}
	dev := &ffDevice{fakeDevice: genericPad(), has: true, api: api}
	f.start()
	f.m.cb.Added(dev)
	_ = f.s.Rumble(f.h.added[0].ID, 65535, 65535, time.Second)
	api.calls = nil
	f.s.Stop()
	calls(t, api, "release effect", "release device")
}

func TestServiceRumbleOfAPadWithoutForceFeedback(t *testing.T) {
	f := newFixture(t, true)
	for name, dev := range map[string]Device{"a device that cannot": &ffDevice{fakeDevice: genericPad()}, "a device that does not know": genericPad()} {
		f.start()
		f.m.cb.Added(dev)
		last := f.h.added[len(f.h.added)-1]
		if last.Rumble {
			t.Errorf("%s: the pad says it can rumble", name)
		}
		err := f.s.Rumble(last.ID, 65535, 65535, time.Second)
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != platform.TagGamepadRumbleUnavailable || !appErr.HasRoute("Service.Rumble."+platform.TagGamepadRumbleUnavailable) {
			t.Errorf("%s: error %v, want GamepadRumbleUnavailable", name, err)
		}
		f.s.Stop()
	}
}

func TestServiceRumbleReturnsTheErrorOfOpeningUntouched(t *testing.T) {
	f := newFixture(t, true)
	want := errors.New(tagForceFeedbackDevice, "no device")
	dev := &ffDevice{fakeDevice: genericPad(), has: true, openErr: want}
	f.start()
	f.m.cb.Added(dev)
	if err := f.s.Rumble(f.h.added[0].ID, 65535, 65535, time.Second); err != want {
		t.Errorf("error %v, want the one of the device", err)
	}
	// The next rumble tries again: no half-made rumble was kept.
	dev.openErr = nil
	dev.api = &fakeFF{}
	if err := f.s.Rumble(f.h.added[0].ID, 65535, 65535, time.Second); err != nil {
		t.Errorf("the second rumble failed: %v", err)
	}
	if dev.opened != 2 {
		t.Errorf("opened %d times, want 2", dev.opened)
	}
}
