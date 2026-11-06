package gpu

import (
	"fmt"
	"log/slog"
	"runtime"
	"sort"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/widgets"
)

// stateKey is the key under which a binding keeps the state of package gpu (WidgetsBinding.Extension).
type stateKey struct{}

// bindingState is what package gpu keeps for one binding: the host that gives it the device of the renderer, the
// Device it made for that device, and the functions that wait for a device.
type bindingState struct {
	b *widgets.WidgetsBinding
	// host is the engine's service, nil when the dispatcher has none (a binding without an engine).
	host explicit.DeviceHost
	stop func()
	// cur is the Device of the device that the host has now, nil when it has none.
	cur *Device
	// ready are the OnDevice functions, in the order they were added.
	ready  []readyFn
	nextID int
	ended  bool
}

type readyFn struct {
	id int
	fn func(*Device)
}

func stateOf(b *widgets.WidgetsBinding) *bindingState {
	return b.Extension(stateKey{}, func() any { return newBindingState(b) }).(*bindingState)
}

func newBindingState(b *widgets.WidgetsBinding) *bindingState {
	st := &bindingState{b: b}
	var host explicit.DeviceHost
	if b.PlatformDispatcher().HostAs(&host) {
		st.host = host
		st.stop = host.WatchGPUDevice(st.onEvent)
	}
	return st
}

// DeviceOf returns the GPU device of the app: the device of the renderer that draws its windows (Vulkan, Direct3D
// 12 or WebGPU), which package gpu shares with it (SPEC §27.2). It fails with GPUUnavailable when the app runs on the
// OpenGL ES, WebGL or software renderer, and with GPUDeviceNotReady before the first frame, when the renderer has
// not opened its device yet: use OnDevice to be called when there is one. Call it on the UI thread.
//
// Flutter: no counterpart
func DeviceOf(bc widgets.BuildContext) (*Device, error) {
	return DeviceOfBinding(widgets.WidgetsBindingOf(bc))
}

// DeviceOfBinding is DeviceOf for code that has the binding and no BuildContext.
//
// Flutter: no counterpart
func DeviceOfBinding(b *widgets.WidgetsBinding) (*Device, error) {
	assertThread(b.UIThread(), "DeviceOfBinding")
	return stateOf(b).current()
}

// OnDevice calls fn with the device of the app each time there is a new one: when the renderer opens its device at
// its first frame, and again after each device that was lost. fn runs on the UI thread, from the event loop, never
// inside the call to OnDevice or inside a frame, so it may make resources and submit. If a device exists when
// OnDevice is called, fn is called for it from the event loop too. It returns the function that stops the calls.
// A GPU view (step 30.5) makes its resources in such a function.
//
// Flutter: no counterpart
func OnDevice(b *widgets.WidgetsBinding, fn func(*Device)) (cancel func()) {
	assertThread(b.UIThread(), "OnDevice")
	st := stateOf(b)
	st.nextID++
	r := readyFn{st.nextID, fn}
	st.ready = append(st.ready, r)
	if d, err := st.current(); err == nil {
		st.callReady(r, d)
	}
	return func() {
		for i, e := range st.ready {
			if e.id == r.id {
				st.ready = append(st.ready[:i:i], st.ready[i+1:]...)
				return
			}
		}
	}
}

// current is the Device for the host's device, or the error that says why there is none. It fails with
// GPUUnavailable and GPUDeviceNotReady.
func (st *bindingState) current() (*Device, error) {
	if st.host == nil {
		return nil, errors.New(TagGPUUnavailable, "the app has no engine that gives a GPU device", "renderer", "none")
	}
	dev, state := st.host.GPUDevice()
	switch state {
	case explicit.HostReady:
		if st.cur == nil || st.cur.dev != dev {
			st.cur = newDevice(st, dev)
		}
		return st.cur, nil
	case explicit.HostUnavailable:
		return nil, errors.New(TagGPUUnavailable, "the renderer of the app has no GPU device: it is the OpenGL ES, WebGL or software renderer",
			"renderer", "gles, webgl or soft")
	}
	return nil, errors.New(TagGPUDeviceNotReady, "the renderer has not opened its device yet: it does at its first frame; use OnDevice")
}

// onEvent is what the host tells: the device was opened, lost or closed, or the app ended.
func (st *bindingState) onEvent(e explicit.DeviceEvent) {
	switch e.Kind {
	case explicit.DeviceOpened:
		if st.cur == nil || st.cur.dev != e.Device {
			st.cur = newDevice(st, e.Device)
		}
		d := st.cur
		for _, r := range st.ready {
			st.callReady(r, d)
		}
	case explicit.DeviceLost, explicit.DeviceClosed:
		if st.cur != nil && st.cur.dev == e.Device {
			d := st.cur
			st.cur = nil
			d.lose(e.Kind, e.Err)
		}
	case explicit.HostEnded:
		st.ended = true
		st.ready = nil
		if st.stop != nil {
			st.stop()
			st.stop = nil
		}
	}
}

// callReady runs fn for the device from the event loop, if the device is still the current one then.
func (st *bindingState) callReady(r readyFn, d *Device) {
	st.b.PlatformDispatcher().Post(func() {
		if st.ended || st.cur != d {
			return
		}
		for _, e := range st.ready {
			if e.id == r.id {
				e.fn(d)
				return
			}
		}
	})
}

// assertThread panics with GPUWrongThread in debug builds when the calling goroutine is not the UI thread (SPEC
// §13). A nil thread has no opinion.
func assertThread(t *foundation.UIThread, call string) {
	if foundation.DebugMode && !t.IsCurrent() {
		panic(errors.New(TagGPUWrongThread, "package gpu is used on the UI thread only (SPEC §13): "+call+" was called from another goroutine",
			"call", call))
	}
}

// resKind is what a resource is, for the counts and the log of leaks.
type resKind int

const (
	kindBuffer resKind = iota
	kindTexture
	kindSampler
	kindShaderModule
	kindRenderPipeline
	kindComputePipeline
	kindBindGroup
	kindQuerySet
	kindCount
)

var kindNames = [kindCount]string{"buffer", "texture", "sampler", "shader module", "render pipeline", "compute pipeline", "bind group", "query set"}

// ResourceCounts is how many resources of each kind a device made that were not released.
//
// Flutter: no counterpart
type ResourceCounts struct {
	Buffers, Textures, Samplers, ShaderModules, RenderPipelines, ComputePipelines, BindGroups, QuerySets int
}

// Total is the number of resources that are alive.
//
// Flutter: no counterpart
func (c ResourceCounts) Total() int {
	return c.Buffers + c.Textures + c.Samplers + c.ShaderModules + c.RenderPipelines + c.ComputePipelines + c.BindGroups + c.QuerySets
}

// resource is what every resource of a device holds: the device, what it is, and whether it was released. It is
// registered with the device, which counts it and, in debug builds, remembers who made it.
type resource struct {
	d        *Device
	id       uint64
	kind     resKind
	label    string
	released bool
}

// liveInfo is what a device remembers of a resource that was not released.
type liveInfo struct {
	kind    resKind
	label   string
	created string
}

// Device is a GPU device: the one of the renderer of the app, shared with it (DeviceOf). Everything made from it
// is used on the UI thread only; in debug builds a call from another goroutine panics with GPUWrongThread. When
// the device is lost, every resource made with it stops working and OnLost functions run (SPEC §27.2).
//
// Flutter: no counterpart
type Device struct {
	st     *bindingState
	dev    explicit.Device
	thread *foundation.UIThread
	limits explicit.Limits
	feat   explicit.Features
	queue  *Queue
	// lostErr is the error of the calls after the device was lost; nil while it works.
	lostErr error
	// onLost are the functions to call when it is lost, with the number that identifies them.
	onLost []lostFn
	nextFn int
	// live are the resources that were not released, by id.
	live   map[uint64]liveInfo
	nextID uint64
	counts [kindCount]int
	// modules caches the explicit form of the shaders, so that a shader made into a module twice shares the
	// pipelines' compiled code.
	modules map[*Shader]*explicit.Shader
}

type lostFn struct {
	id int
	fn func(err error)
}

func newDevice(st *bindingState, dev explicit.Device) *Device {
	d := &Device{st: st, dev: dev, thread: st.b.UIThread(), limits: dev.Limits(), feat: dev.Features(),
		live: map[uint64]liveInfo{}, modules: map[*Shader]*explicit.Shader{}}
	d.queue = &Queue{d: d}
	return d
}

// Limits returns the limits of the device.
//
// Flutter: no counterpart
func (d *Device) Limits() Limits { return limitsOf(d.limits) }

// Features returns the optional features of the device.
//
// Flutter: no counterpart
func (d *Device) Features() Features { return featuresOf(d.feat) }

// Description names the adapter, such as "AMD Radeon RX 6800M (RADV NAVI22)", when the backend can tell; "" when
// it cannot.
//
// Flutter: no counterpart
func (d *Device) Description() string {
	if x, ok := d.dev.(interface{ Description() string }); ok {
		return x.Description()
	}
	return ""
}

// Queue returns the queue of the device.
//
// Flutter: no counterpart
func (d *Device) Queue() *Queue { return d.queue }

// IsLost says whether the device was lost. A lost device does nothing, and its calls that return an error return
// GPUDeviceLost.
//
// Flutter: no counterpart
func (d *Device) IsLost() bool { return d.lostErr != nil }

// Resources returns how many resources of each kind are alive. The memory tests check that it goes back to 0.
//
// Flutter: no counterpart
func (d *Device) Resources() ResourceCounts {
	return ResourceCounts{
		Buffers: d.counts[kindBuffer], Textures: d.counts[kindTexture], Samplers: d.counts[kindSampler],
		ShaderModules: d.counts[kindShaderModule], RenderPipelines: d.counts[kindRenderPipeline],
		ComputePipelines: d.counts[kindComputePipeline], BindGroups: d.counts[kindBindGroup], QuerySets: d.counts[kindQuerySet],
	}
}

// OnLost calls fn once when the device is lost (a driver reset, a lost WebGPU device), on the UI thread from the event
// loop, with the GPUDeviceLost error. It returns the function that removes fn. A function added to a device that
// is lost already is called at once from the event loop.
//
// Flutter: no counterpart
func (d *Device) OnLost(fn func(err error)) (remove func()) {
	d.assertThread("OnLost")
	if d.lostErr != nil {
		err := d.lostErr
		d.st.b.PlatformDispatcher().Post(func() { fn(err) })
		return func() {}
	}
	d.nextFn++
	id := d.nextFn
	d.onLost = append(d.onLost, lostFn{id, fn})
	return func() {
		for i, e := range d.onLost {
			if e.id == id {
				d.onLost = append(d.onLost[:i:i], d.onLost[i+1:]...)
				return
			}
		}
	}
}

func (d *Device) assertThread(call string) { assertThread(d.thread, call) }

// lose marks the device lost and calls the OnLost functions from the event loop. A closed device (the app ends, or
// the engine went on to another renderer) reports the resources that were not released first. It makes the
// GPUDeviceLost error: this is the function that fails.
func (d *Device) lose(kind explicit.DeviceEventKind, cause error) {
	if d.lostErr != nil {
		return
	}
	if kind == explicit.DeviceClosed {
		d.reportLeaks()
	}
	text := "the GPU device was closed"
	if kind == explicit.DeviceLost {
		text = "the GPU device was lost"
	}
	attrs := []any{"event", kind.String()}
	if cause != nil {
		attrs = append(attrs, "cause", cause.Error())
	}
	d.lostErr = errors.New(TagGPUDeviceLost, text, attrs...)
	fns := d.onLost
	d.onLost = nil
	err := d.lostErr
	for _, f := range fns {
		fn := f.fn
		d.st.b.PlatformDispatcher().Post(func() { fn(err) })
	}
}

// reportLeaks logs, in debug builds, each resource that was not released, with the call that made it
// (GPUResourceLeaked, a warning).
func (d *Device) reportLeaks() {
	if !foundation.DebugMode || len(d.live) == 0 || d.st.host == nil {
		return
	}
	log := d.st.host.GPULogger()
	if log == nil {
		log = slog.Default()
	}
	ids := make([]uint64, 0, len(d.live))
	for id := range d.live {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		l := d.live[id]
		log.Warn("a GPU resource was not released", "tag", TagGPUResourceLeaked, "kind", kindNames[l.kind], "label", l.label, "created", l.created)
	}
}

// track registers a new resource: it is counted, and in debug builds the call that made it is remembered. skip is the
// number of frames between the caller of the public method and track's caller.
func (d *Device) track(kind resKind, label string, skip int) resource {
	d.nextID++
	r := resource{d: d, id: d.nextID, kind: kind, label: label}
	d.counts[kind]++
	info := liveInfo{kind: kind, label: label}
	if foundation.DebugMode {
		if _, file, line, ok := runtime.Caller(skip + 1); ok {
			info.created = fmt.Sprintf("%s:%d", file, line)
		}
	}
	d.live[r.id] = info
	return r
}

// untrack forgets a resource that was released. It reports whether this is the first release.
func (r *resource) untrack() bool {
	if r.released {
		return false
	}
	r.released = true
	delete(r.d.live, r.id)
	r.d.counts[r.kind]--
	return true
}

// Label is the name that the resource was given in its description.
//
// Flutter: no counterpart
func (r *resource) Label() string { return r.label }

// usable panics with GPUResourceReleased in debug builds when a resource that was released is used.
func (r *resource) usable(call string) {
	if foundation.DebugMode && r.released {
		panic(errors.New(TagGPUResourceReleased, "a "+kindNames[r.kind]+" was used after Release", "call", call, "kind", kindNames[r.kind], "label", r.label))
	}
}
