//go:build darwin && !ios

package iokit

import (
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/darwin/cf"
)

// The framework is there and every symbol binds (the Mac that runs the tests has the Force Feedback framework).
func TestLoadFFBindsForceFeedback(t *testing.T) {
	f, err := loadFF()
	if err != nil {
		t.Fatal(err)
	}
	if f.sineUUID == 0 || f.service == nil || f.isForceFeedback == nil || f.createEffect == nil || f.start == nil {
		t.Errorf("the table is not complete: %+v", f)
	}
}

// fakeFFFuncs makes an ffFuncs table whose calls are recorded and answer from the fields.
type fakeFFTable struct {
	*ffFuncs
	calls      []string
	isFFResult int32
	createDev  int32
	devHandle  uintptr
	effectHR   int32
	service    uint32
	lastEffect ffEffect
}

func newFakeFFTable() *fakeFFTable {
	t := &fakeFFTable{ffFuncs: &ffFuncs{}, devHandle: 0xD00D, service: 77}
	f := t.ffFuncs
	f.sineUUID = cf.Ref(0x1234)
	f.service = func(cf.Ref) uint32 { t.calls = append(t.calls, "service"); return t.service }
	f.isForceFeedback = func(s uint32) int32 { t.calls = append(t.calls, "is ff"); return t.isFFResult }
	f.createDevice = func(s uint32, out *uintptr) int32 {
		t.calls = append(t.calls, "create device")
		*out = t.devHandle
		return t.createDev
	}
	f.releaseDevice = func(dev uintptr) int32 { t.calls = append(t.calls, "release device"); return 0 }
	f.command = func(dev uintptr, cmd uint32) int32 { t.calls = append(t.calls, "command"); return 0 }
	f.createEffect = func(dev uintptr, typ cf.Ref, e unsafe.Pointer, out *uintptr) int32 {
		t.calls = append(t.calls, "create effect")
		t.lastEffect = *(*ffEffect)(e)
		*out = 0xEFFEC7
		return t.effectHR
	}
	f.releaseEffect = func(dev, effect uintptr) int32 { t.calls = append(t.calls, "release effect"); return 0 }
	f.setParameters = func(effect uintptr, e unsafe.Pointer, flags uint32) int32 { t.calls = append(t.calls, "set"); return 0 }
	f.start = func(effect uintptr, iterations, flags uint32) int32 {
		t.calls = append(t.calls, "start")
		if iterations != 1 || flags != 0 {
			t.calls = append(t.calls, "bad start arguments")
		}
		return 0
	}
	f.stop = func(effect uintptr) int32 { t.calls = append(t.calls, "stop"); return 0 }
	return t
}

func withFF(t *testing.T, tab *fakeFFTable) {
	t.Helper()
	old := ffLoader
	ffLoader = func() (*ffFuncs, error) { return tab.ffFuncs, nil }
	t.Cleanup(func() { ffLoader = old })
}

func TestDeviceForceFeedbackOverTheFramework(t *testing.T) {
	tab := newFakeFFTable()
	withFF(t, tab)
	d := &device{ref: 5}
	if !d.HasForceFeedback() {
		t.Fatal("a device whose service is force feedback says it has none")
	}
	tab.isFFResult = -1
	if d.HasForceFeedback() {
		t.Error("a device whose service is not force feedback says it has")
	}
	tab.isFFResult, tab.calls = 0, nil

	rumble, err := d.OpenForceFeedback()
	if err != nil {
		t.Fatal(err)
	}
	if err := rumble.Play(65535, 65535, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := rumble.Play(0, 0, 0); err != nil {
		t.Fatal(err)
	}
	rumble.Close()
	want := []string{"service", "create device", "command", "command", "create effect", "start", "stop", "release effect", "release device"}
	if len(tab.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", tab.calls, want)
	}
	for i := range want {
		if tab.calls[i] != want[i] {
			t.Fatalf("calls = %v, want %v", tab.calls, want)
		}
	}
	if tab.lastEffect.Size != 80 || tab.lastEffect.Duration != 1_000_000 {
		t.Errorf("effect = %+v", tab.lastEffect)
	}
}

func TestDeviceForceFeedbackFailures(t *testing.T) {
	tab := newFakeFFTable()
	withFF(t, tab)
	d := &device{ref: 5}
	tab.createDev = -1
	_, err := d.OpenForceFeedback()
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagForceFeedbackDevice || !appErr.HasRoute("device.OpenForceFeedback."+tagForceFeedbackDevice) {
		t.Errorf("error %v, want ForceFeedbackDeviceFailed", err)
	}
	d.gone.Store(true)
	if d.HasForceFeedback() {
		t.Error("a device that is gone has force feedback")
	}
	_, err = d.OpenForceFeedback()
	appErr, ok = errors.FromError(err)
	if !ok || appErr.Tag != tagForceFeedbackGone || !appErr.HasRoute("device.OpenForceFeedback."+tagForceFeedbackGone) {
		t.Errorf("error %v, want ForceFeedbackDeviceGone", err)
	}
	// A machine without the framework has no force feedback and no error to show.
	old := ffLoader
	defer func() { ffLoader = old }()
	ffLoader = func() (*ffFuncs, error) { return nil, errors.New(tagIOKitUnavailable, "no framework") }
	if (&device{ref: 5}).HasForceFeedback() {
		t.Error("a machine without the framework says it has force feedback")
	}
	if _, err := (&device{ref: 5}).OpenForceFeedback(); err == nil {
		t.Error("opening force feedback worked without the framework")
	}
}
