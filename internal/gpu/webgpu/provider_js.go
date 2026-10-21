//go:build js && wasm

package webgpu

import (
	"context"
	"log/slog"
	"sync"
	"syscall/js"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// Provider hands out the devices of the browser's WebGPU to the renderer. Acquire makes it and
// waits for the first device; a device that is lost is replaced by one asked for without waiting.
//
// Flutter: no counterpart
type Provider struct {
	o      Options
	log    *slog.Logger
	helper js.Value
	funcs  []js.Func
	sup    *supply
	codes  js.Value // the WGSL of every shader, by name

	mu      sync.Mutex
	devices map[int]*Device
	nextID  int
	reads   map[int]*readJob
	handle  gpu.SurfaceHandle // the window the last device was opened for
	// delivered is closed over by Acquire to hear that the first request is answered.
	delivered func()
	job       int
	// validations are the errors the browser reported for calls that no error scope caught.
	validations []string
}

// defaultTimeout is how long Acquire waits when Options.Timeout is zero.
const defaultTimeout = 20 * time.Second

// acquired is what a request for a device gives: the browser's objects.
type acquired struct {
	adapter, device, modules js.Value
	description              string
	fallback                 bool
}

// Acquire asks the browser for an adapter and a device, compiles every shader of the renderer on
// it, and waits for all of it. It is the one place where the UI goroutine waits for promises, and
// the engine calls it before the platform installs its event listeners (SPEC §23.3). It fails
// with WebGPUUnavailable (no navigator.gpu, or no adapter), WebGPUDeviceFailed (the adapter gave
// no device, or the wait ended: the context ended or Options.Timeout passed) and ShaderModuleInvalid
// (the browser rejected a WGSL module, with its messages).
//
// Flutter: no counterpart
func Acquire(ctx context.Context, o Options) (*Provider, error) {
	log := o.Logger
	if log == nil {
		log = slog.Default()
	}
	p := &Provider{o: o, log: log, devices: map[int]*Device{}, reads: map[int]*readJob{}}
	p.sup = &supply{request: p.request}
	p.helper = js.Global().Get("Function").New(helperScript).Invoke()
	callbacks := js.Global().Get("Object").New()
	callbacks.Set("acquired", p.keep(func(args []js.Value) { p.onAcquired(args[0].Int(), args[1]) }))
	callbacks.Set("failed", p.keep(func(args []js.Value) { p.onFailed(args[0].Int(), args[1].String(), args[2].String()) }))
	callbacks.Set("lost", p.keep(func(args []js.Value) { p.onLost(args[0].Int(), args[1].String(), args[2].String()) }))
	callbacks.Set("uncaptured", p.keep(func(args []js.Value) { p.onUncaptured(args[0].Int(), args[1].String()) }))
	callbacks.Set("mapped", p.keep(func(args []js.Value) { p.onMapped(args[0].Int()) }))
	callbacks.Set("mapFailed", p.keep(func(args []js.Value) { p.onMapFailed(args[0].Int(), args[1].String()) }))
	p.helper.Call("init", callbacks)
	p.codes = js.Global().Get("Object").New()
	for _, s := range explicit.Shaders() {
		p.codes.Set(s.Name, s.WGSL)
	}

	done := make(chan struct{}, 1)
	p.mu.Lock()
	p.delivered = func() {
		select {
		case done <- struct{}{}:
		default:
		}
	}
	p.mu.Unlock()
	p.sup.start()
	timeout := o.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var waitErr error
	select {
	case <-done:
	case <-ctx.Done():
		waitErr = errDeviceFailed("the wait for the adapter and the device ended: " + ctx.Err().Error())
	case <-timer.C:
		waitErr = errDeviceFailed("the browser gave no adapter and device in " + timeout.String())
	}
	p.mu.Lock()
	p.delivered = nil
	p.mu.Unlock()
	if waitErr != nil {
		return nil, waitErr
	}
	state, f := p.sup.peek()
	if state == supplyFailed {
		return nil, failureError(f)
	}
	return p, nil
}

// failureError makes the error of a failed request, in the goroutine that reports it.
func failureError(f *failure) error {
	switch f.tag {
	case "unavailable":
		return errUnavailable(f.message)
	case "shader":
		return errShaderInvalid(f.message)
	}
	return errDeviceFailed(f.message)
}

// keep makes a callback for the browser. They are made once, here, and kept for the life of the
// program (SPEC §23.3); each calls fn with the arguments and returns nothing.
func (p *Provider) keep(fn func(args []js.Value)) js.Func {
	f := js.FuncOf(func(_ js.Value, args []js.Value) any {
		fn(args)
		return nil
	})
	p.funcs = append(p.funcs, f)
	return f
}

// post runs fn on the UI goroutine.
func (p *Provider) post(fn func()) {
	if p.o.Post != nil {
		p.o.Post(fn)
		return
	}
	fn()
}

// request asks the browser for a device: the supply calls it. The answer arrives in onAcquired or
// onFailed.
func (p *Provider) request() {
	p.mu.Lock()
	p.job++
	job := p.job
	p.mu.Unlock()
	p.helper.Call("request", p.codes, job)
}

func (p *Provider) onAcquired(_ int, r js.Value) {
	p.sup.deliver(&acquired{
		adapter: r.Get("adapter"), device: r.Get("device"), modules: r.Get("modules"),
		description: r.Get("description").String(), fallback: r.Get("fallback").Truthy(),
	}, nil)
	p.answered()
}

func (p *Provider) onFailed(_ int, tag, message string) {
	p.sup.deliver(nil, &failure{tag: tag, message: message})
	p.answered()
}

// answered tells Acquire that its request was answered, or, for a device that was asked for
// after a loss, tells the owner on the UI goroutine.
func (p *Provider) answered() {
	p.mu.Lock()
	delivered, h := p.delivered, p.handle
	p.mu.Unlock()
	if delivered != nil {
		delivered()
		return
	}
	state, f := p.sup.peek()
	if state == supplyFailed {
		p.log.Warn("the WebGPU device asked for after a loss did not come", "tag", failureError(f).Error())
	}
	p.post(func() {
		if p.o.OnDeviceReady != nil {
			p.o.OnDeviceReady(h)
		}
	})
}

// Open is the open function of the explicit renderer: it makes the Device from the device that
// Acquire got, or from the one that was asked for after a loss. It returns the error of a request
// that failed (WebGPUUnavailable, WebGPUDeviceFailed, ShaderModuleInvalid), and
// explicit.GPUDevicePending while a new device is still coming. A nil Provider fails with
// WebGPUUnavailable.
//
// Flutter: no counterpart
func (p *Provider) Open(h gpu.SurfaceHandle) (explicit.Device, error) {
	if p == nil {
		return nil, errUnavailable("the engine did not start WebGPU")
	}
	p.mu.Lock()
	p.handle = h
	p.mu.Unlock()
	got, state, f := p.sup.take()
	switch state {
	case supplyReady:
		return p.newDevice(got.(*acquired)), nil
	case supplyFailed:
		return nil, failureError(f)
	}
	return nil, devicePending()
}

// Close destroys the device that Acquire got when nobody opened it (the platform failed to start);
// the devices it gave are the renderer's, which closes them.
//
// Flutter: no counterpart
func (p *Provider) Close() {
	if p == nil {
		return
	}
	if a, ok := p.sup.release().(*acquired); ok {
		_, _ = tryCall("destroy", func() js.Value { return a.device.Call("destroy") })
	}
}

// register gives a device its number, which the browser's callbacks name it by.
func (p *Provider) register(d *Device) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nextID++
	p.devices[p.nextID] = d
	return p.nextID
}

func (p *Provider) forget(id int) {
	p.mu.Lock()
	delete(p.devices, id)
	p.mu.Unlock()
}

func (p *Provider) device(id int) *Device {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.devices[id]
}

// onLost is the browser telling that a device was lost. A device that its owner destroyed reports
// the reason "destroyed" too, and is not news. A real loss makes the device fail every call with
// GPUDeviceLost and starts the request for the device that replaces it.
func (p *Provider) onLost(id int, reason, message string) {
	d := p.device(id)
	if d == nil || !d.loss.markLost(reason, message) {
		return
	}
	p.log.Info("the WebGPU device was lost", "reason", reason, "message", message)
	p.sup.start()
}

func (p *Provider) onUncaptured(id int, message string) {
	d := p.device(id)
	if d == nil {
		return
	}
	d.noteValidation(message)
}

// noteValidation records an error of the browser and logs the first few.
func (p *Provider) noteValidation(message string) {
	p.mu.Lock()
	p.validations = append(p.validations, message)
	n := len(p.validations)
	p.mu.Unlock()
	if n <= 20 {
		p.log.Warn("WebGPU validation error", "message", message)
	}
}

// ValidationErrors returns the errors that the browser reported so far for calls that no error
// scope caught: a call that breaks a WebGPU rule is reported to the page in a later task and does
// not stop the frame. The browser tests fail on any.
//
// Flutter: no counterpart
func (p *Provider) ValidationErrors() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.validations...)
}
