package explicit

import "log/slog"

// DeviceEventKind says what happened to the device of a renderer.
//
// Flutter: no counterpart
type DeviceEventKind int

// The events of the device of a renderer.
const (
	// DeviceOpened is the first frame, or the first one after a loss: the renderer has a device.
	//
	// Flutter: no counterpart
	DeviceOpened DeviceEventKind = iota + 1
	// DeviceLost is the loss of the device (a driver reset, a destroyed WebGPU device): the renderer is
	// about to release it and makes a new one at its next frame (SPEC §9.4, §9.5). It is sent before the
	// device is closed, so that what holds resources made with it can let them go.
	//
	// Flutter: no counterpart
	DeviceLost
	// DeviceClosed is the end of the renderer (the app ends, or the engine goes on to another renderer):
	// the device is about to be closed and no other follows from this renderer.
	//
	// Flutter: no counterpart
	DeviceClosed
	// HostEnded is the end of the app: the host sends it after the renderer is closed, with no device. It
	// is the moment when what package gpu still holds is a leak.
	//
	// Flutter: no counterpart
	HostEnded
)

// String returns "opened", "lost", "closed" or "ended".
func (k DeviceEventKind) String() string {
	switch k {
	case DeviceOpened:
		return "opened"
	case DeviceLost:
		return "lost"
	case DeviceClosed:
		return "closed"
	case HostEnded:
		return "ended"
	}
	return "unknown"
}

// DeviceEvent is a change of the device of a renderer.
//
// Flutter: no counterpart
type DeviceEvent struct {
	// Kind says what happened.
	Kind DeviceEventKind
	// Device is the device that was opened, lost or closed.
	Device Device
	// Err is why the device was lost, for DeviceLost; nil otherwise.
	Err error
}

// HostState is whether a host has a device.
//
// Flutter: no counterpart
type HostState int

// The states of a DeviceHost.
const (
	// HostPending means that the renderer is one that has a device and has not opened it yet (it does so at
	// its first frame, with the handle of the first window), or has lost it and opens another at its next frame.
	//
	// Flutter: no counterpart
	HostPending HostState = iota
	// HostReady means that GPUDevice returns the device.
	//
	// Flutter: no counterpart
	HostReady
	// HostUnavailable means that the renderer has no device and will not have one: the GLES, WebGL and software
	// renderers.
	//
	// Flutter: no counterpart
	HostUnavailable
)

// DeviceHost is what the engine gives to package gpu (SPEC §27.2) through ui.PlatformDispatcher.HostAs: the
// device of the renderer it chose, and its changes. Every method runs on the UI thread.
//
// Flutter: no counterpart
type DeviceHost interface {
	// GPUDevice returns the device of the renderer when state is HostReady.
	GPUDevice() (dev Device, state HostState)
	// WatchGPUDevice calls fn for each change of the device from now on, on the UI thread, inside the frame
	// that caused it. It returns the function that stops the calls.
	WatchGPUDevice(fn func(DeviceEvent)) (stop func())
	// GPULogger is where package gpu logs what it finds out (a resource that was not released).
	GPULogger() *slog.Logger
}
