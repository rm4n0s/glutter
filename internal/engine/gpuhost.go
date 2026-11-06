package engine

import (
	"log/slog"
	"slices"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// gpuHub is what the engine gives to package gpu (SPEC §27.2): the device of the explicit renderer that draws
// the windows, and the changes of it. The renderers tell it (RendererOpts.OnDevice) from Draw and Close, on the UI
// thread, and package gpu asks and watches through the handler, which implements explicit.DeviceHost by
// delegating to it. It is used on the UI thread only.
type gpuHub struct {
	log *slog.Logger
	// dev is the device of the renderer, nil before the first frame and between a loss and the next frame.
	dev explicit.Device
	// watchers are the functions that watch the device, with the number WatchGPUDevice gave them.
	watchers []gpuWatcher
	next     int
	// renderer says whether the renderer in use has a device at all (see hasDevice).
	renderer func() (chosen bool, name string)
}

type gpuWatcher struct {
	id int
	fn func(explicit.DeviceEvent)
}

// explicitRenderers are the renderers whose device package gpu can use.
var explicitRenderers = []string{"vulkan", "webgpu", "d3d12"}

// hasDevice says whether the renderer that is in use is one with a device: false when the engine chose GLES,
// WebGL or the software renderer.
func (g *gpuHub) hasDevice() bool {
	if g.renderer == nil {
		return true
	}
	chosen, name := g.renderer()
	return !chosen || slices.Contains(explicitRenderers, name)
}

// notify is what the renderers call: it keeps the device and tells the watchers, in the order they were added.
func (g *gpuHub) notify(e explicit.DeviceEvent) {
	switch e.Kind {
	case explicit.DeviceOpened:
		g.dev = e.Device
	case explicit.DeviceLost, explicit.DeviceClosed:
		if g.dev == e.Device {
			g.dev = nil
		}
	}
	for _, w := range slices.Clone(g.watchers) {
		w.fn(e)
	}
}

// device implements explicit.DeviceHost.GPUDevice.
func (g *gpuHub) device() (explicit.Device, explicit.HostState) {
	switch {
	case g.dev != nil:
		return g.dev, explicit.HostReady
	case !g.hasDevice():
		return nil, explicit.HostUnavailable
	}
	return nil, explicit.HostPending
}

// watch implements explicit.DeviceHost.WatchGPUDevice.
func (g *gpuHub) watch(fn func(explicit.DeviceEvent)) func() {
	g.next++
	id := g.next
	g.watchers = append(g.watchers, gpuWatcher{id, fn})
	return func() {
		g.watchers = slices.DeleteFunc(g.watchers, func(w gpuWatcher) bool { return w.id == id })
	}
}

// GPUDevice implements explicit.DeviceHost: the device of the renderer of the windows, when it has one.
func (h *handler) GPUDevice() (explicit.Device, explicit.HostState) { return h.gpu.device() }

// WatchGPUDevice implements explicit.DeviceHost.
func (h *handler) WatchGPUDevice(fn func(explicit.DeviceEvent)) func() { return h.gpu.watch(fn) }

// GPULogger implements explicit.DeviceHost.
func (h *handler) GPULogger() *slog.Logger { return h.log }

var _ explicit.DeviceHost = (*handler)(nil)

// chosenRenderer says which renderer the chooser settled on: chosen is false before the first frame, and
// name is the renderer's name ("vulkan", "gles", ...). A renderer that is not the chooser (a test's) is
// reported as not chosen yet.
func (h *handler) chosenRenderer() (chosen bool, name string) {
	c, ok := h.r.(*chooser)
	if !ok {
		return false, "" // a renderer of a test: it tells by its events
	}
	return c.cur != nil, c.name
}
