// Package gpuhost is an explicit.DeviceHost for tests: it holds the device that a test gives it, tells the watchers
// when the device is opened, lost or closed, and keeps the log that package gpu writes. gluttertest's recording device
// and the display and browser tests of package gpu use it. It is imported by tests only.
//
// Flutter: no counterpart
package gpuhost

import (
	"bytes"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// Host is the device host of a test.
//
// Flutter: no counterpart
type Host struct {
	mu       sync.Mutex
	dev      explicit.Device
	watchers map[int]func(explicit.DeviceEvent)
	next     int
	logs     bytes.Buffer
	log      *slog.Logger
	ended    bool
	textures *explicit.TextureRegistry
}

var _ explicit.DeviceHost = (*Host)(nil)

// New returns a host whose device is open.
//
// Flutter: no counterpart
func New(dev explicit.Device) *Host {
	h := &Host{dev: dev, watchers: map[int]func(explicit.DeviceEvent){}, textures: explicit.NewTextureRegistry()}
	h.log = slog.New(slog.NewTextHandler(&lockedWriter{h}, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return h
}

// lockedWriter writes to the log of the host under its lock.
type lockedWriter struct{ h *Host }

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.h.mu.Lock()
	defer w.h.mu.Unlock()
	n, _ := w.h.logs.Write(p) // a bytes.Buffer never fails to write
	return n, nil
}

// GPUDevice implements explicit.DeviceHost.
func (h *Host) GPUDevice() (explicit.Device, explicit.HostState) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.dev == nil {
		return nil, explicit.HostPending
	}
	return h.dev, explicit.HostReady
}

// WatchGPUDevice implements explicit.DeviceHost.
func (h *Host) WatchGPUDevice(fn func(explicit.DeviceEvent)) func() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.next++
	id := h.next
	h.watchers[id] = fn
	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.watchers, id)
	}
}

// GPULogger implements explicit.DeviceHost: the log is kept for Logs.
func (h *Host) GPULogger() *slog.Logger { return h.log }

// GPUTextures implements explicit.DeviceHost.
func (h *Host) GPUTextures() *explicit.TextureRegistry { return h.textures }

// emit tells the watchers, in the order they were added.
func (h *Host) emit(e explicit.DeviceEvent) {
	h.mu.Lock()
	ids := make([]int, 0, len(h.watchers))
	for id := range h.watchers {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	fns := make([]func(explicit.DeviceEvent), len(ids))
	for i, id := range ids {
		fns[i] = h.watchers[id]
	}
	h.mu.Unlock()
	for _, fn := range fns {
		fn(e)
	}
}

// Device returns the device of the moment.
//
// Flutter: no counterpart
func (h *Host) Device() explicit.Device {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.dev
}

// Lose loses the device, as a driver reset does: the watchers are told, and the host has no device until Open gives it
// one (the renderer asks the system for the new device, which may take a while).
//
// Flutter: no counterpart
func (h *Host) Lose() {
	h.mu.Lock()
	old := h.dev
	h.dev = nil
	h.mu.Unlock()
	if old != nil {
		h.emit(explicit.DeviceEvent{Kind: explicit.DeviceLost, Device: old})
	}
}

// Open gives the host a device, as the renderer does when it has made one, and tells the watchers.
//
// Flutter: no counterpart
func (h *Host) Open(dev explicit.Device) {
	h.mu.Lock()
	h.dev = dev
	h.mu.Unlock()
	h.emit(explicit.DeviceEvent{Kind: explicit.DeviceOpened, Device: dev})
}

// Replace loses the device and opens fresh at once, as the renderer does when the system has the new device ready
// (SPEC §9.4, §9.5).
//
// Flutter: no counterpart
func (h *Host) Replace(fresh explicit.Device) {
	h.Lose()
	h.Open(fresh)
}

// End ends the app as the engine does when its windows are gone: the device is closed and the watchers are told that
// the host ended. The second call does nothing.
//
// Flutter: no counterpart
func (h *Host) End() {
	h.mu.Lock()
	if h.ended {
		h.mu.Unlock()
		return
	}
	h.ended = true
	dev := h.dev
	h.mu.Unlock()
	h.emit(explicit.DeviceEvent{Kind: explicit.DeviceClosed, Device: dev})
	h.emit(explicit.DeviceEvent{Kind: explicit.HostEnded})
}

// Logs returns what package gpu logged.
//
// Flutter: no counterpart
func (h *Host) Logs() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.logs.String()
}

// ClearLogs forgets the log.
//
// Flutter: no counterpart
func (h *Host) ClearLogs() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.logs.Reset()
}

// LogContains says whether the log has the text.
//
// Flutter: no counterpart
func (h *Host) LogContains(text string) bool { return strings.Contains(h.Logs(), text) }
