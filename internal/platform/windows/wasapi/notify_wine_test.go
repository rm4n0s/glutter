//go:build windows && glutter_wine && glutter_audio

package wasapi

// The notification client is a COM object that Go makes. Wine does not tell it that the default device changed (the server moves the
// stream by itself), so the test calls its methods through the vtable, as Windows does: the arguments are read right, the roles and
// the flows are told apart, and a device that follows the default is made again.

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

type silentHandler struct{ fills atomic.Int64 }

func (h *silentHandler) Fill(out []float32) {
	clear(out)
	h.fills.Add(1)
}
func (*silentHandler) Capture([]float32)                {}
func (*silentHandler) StateChanged(platform.AudioState) {}

func wait(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestWineNotificationClient(t *testing.T) {
	if os.Getenv("PULSE_SERVER") == "" {
		t.Skip("no private audio server")
	}
	s := New(nil)
	defer s.Close()
	var changes atomic.Int32
	s.SetChangeHandler(func() { changes.Add(1) })
	if _, err := s.Devices(platform.AudioPlayback); err != nil {
		t.Fatal(err)
	}
	if s.notif.IsNil() {
		t.Fatal("the notification client was not registered")
	}
	id, err := win32.UTF16PtrFromString("{0.0.0.00000000}.{another}")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("the changes of the list reach the handler", func(t *testing.T) {
		before := changes.Load()
		notify(t, s, win32.IMMNotificationClientOnDeviceAdded, uintptr(unsafe.Pointer(id)))
		notify(t, s, win32.IMMNotificationClientOnDeviceRemoved, uintptr(unsafe.Pointer(id)))
		notify(t, s, win32.IMMNotificationClientOnDeviceStateChanged, uintptr(unsafe.Pointer(id)), 4)
		if got := changes.Load() - before; got != 3 {
			t.Errorf("%d changes, want 3", got)
		}
		before = changes.Load()
		notify(t, s, win32.IMMNotificationClientOnPropertyValueChanged, uintptr(unsafe.Pointer(id)), uintptr(unsafe.Pointer(&win32.PKeyDeviceFriendlyName)))
		if got := changes.Load() - before; got != 0 {
			t.Errorf("a property that changed made %d changes, want none", got)
		}
	})

	t.Run("only the console role of the default device counts", func(t *testing.T) {
		before := changes.Load()
		notify(t, s, win32.IMMNotificationClientOnDefaultDeviceChanged, win32.ERender, 1, uintptr(unsafe.Pointer(id)))
		notify(t, s, win32.IMMNotificationClientOnDefaultDeviceChanged, win32.ERender, 2, uintptr(unsafe.Pointer(id)))
		if got := changes.Load() - before; got != 0 {
			t.Errorf("the multimedia and communications roles made %d changes", got)
		}
		notify(t, s, win32.IMMNotificationClientOnDefaultDeviceChanged, win32.ERender, win32.EConsole, uintptr(unsafe.Pointer(id)))
		if got := changes.Load() - before; got != 1 {
			t.Errorf("the console role made %d changes, want 1", got)
		}
	})

	t.Run("a device that follows the default is made again", func(t *testing.T) {
		h := &silentHandler{}
		dev, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{}, h)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := dev.Close(); err != nil {
				t.Error(err)
			}
		}()
		d := dev.(*device)
		wait(t, "the first fills", func() bool { return h.fills.Load() > 5 })
		// The flow of the recordings does not touch it.
		notify(t, s, win32.IMMNotificationClientOnDefaultDeviceChanged, win32.ECapture, win32.EConsole, uintptr(unsafe.Pointer(id)))
		time.Sleep(100 * time.Millisecond)
		if n := d.rebuilds.Load(); n != 0 {
			t.Fatalf("a change of the default recording device made the playback stream again (%d)", n)
		}
		notify(t, s, win32.IMMNotificationClientOnDefaultDeviceChanged, win32.ERender, win32.EConsole, uintptr(unsafe.Pointer(id)))
		wait(t, "the stream to be made again", func() bool { return d.rebuilds.Load() == 1 })
		fills := h.fills.Load()
		wait(t, "fills after the stream was made again", func() bool { return h.fills.Load() > fills+5 })
		// The same device again (its ID is the one the stream has) is no change.
		notify(t, s, win32.IMMNotificationClientOnDefaultDeviceChanged, win32.ERender, win32.EConsole, uintptr(unsafe.Pointer(utf16Of(t, d.curIDOnThread()))))
		time.Sleep(100 * time.Millisecond)
		if n := d.rebuilds.Load(); n != 1 {
			t.Errorf("the default device that the stream already has made it again (%d)", n)
		}
	})

	t.Run("a pinned device is not moved", func(t *testing.T) {
		list, err := s.Devices(platform.AudioPlayback)
		if err != nil || len(list) == 0 {
			t.Fatalf("devices: %v, %v", list, err)
		}
		h := &silentHandler{}
		dev, err := s.Open(context.Background(), platform.AudioPlayback, platform.AudioOpenOptions{ID: list[0].ID}, h)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := dev.Close(); err != nil {
				t.Error(err)
			}
		}()
		d := dev.(*device)
		wait(t, "the first fills", func() bool { return h.fills.Load() > 5 })
		notify(t, s, win32.IMMNotificationClientOnDefaultDeviceChanged, win32.ERender, win32.EConsole, uintptr(unsafe.Pointer(id)))
		time.Sleep(100 * time.Millisecond)
		if n := d.rebuilds.Load(); n != 0 {
			t.Errorf("a pinned device was made again (%d)", n)
		}
	})

	t.Run("the object keeps its references", func(t *testing.T) {
		n := s.notif.AddRef()
		if got := s.notif.Release(); got != n-1 {
			t.Errorf("AddRef gave %d and Release %d", n, got)
		}
	})
}

// notify calls a method of the notification client through its vtable, as Windows does.
//
//go:uintptrescapes
func notify(t *testing.T, s *Service, index int, args ...uintptr) {
	t.Helper()
	if hr := call(s.notif, index, args...); hr != win32.SOK {
		t.Fatalf("method %d answered %s", index, hr)
	}
}

// utf16Of is the UTF-16 text of a string.
func utf16Of(t *testing.T, s string) *uint16 {
	t.Helper()
	p, err := win32.UTF16PtrFromString(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// curIDOnThread is the ID of the endpoint of the stream, read on the thread of the device.
func (d *device) curIDOnThread() string {
	var id string
	d.exec(func() { id = d.curID })
	return id
}
