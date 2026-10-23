//go:build js && wasm

package webgpu

import (
	"image"
	"syscall/js"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/ui"
)

// panicError runs fn and returns the error it panicked with; nil when it did not panic.
func panicError(t *testing.T, fn func()) (err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(error)
			if !ok {
				t.Fatalf("panicked with %T, want an error", r)
			}
			err = e
		}
	}()
	fn()
	return nil
}

// TestDeviceErrorRoutes checks the failures that a Device finds before it calls the browser: each
// has a tag of its own, and its route ends in the method that found it (R4 of the errors guide).
func TestDeviceErrorRoutes(t *testing.T) {
	d := &Device{limits: explicit.DefaultLimits()}
	other := &Device{limits: explicit.DefaultLimits()}
	shader := explicit.ShaderBlit
	unknown := &explicit.Shader{Name: "unknown", Entries: shader.Entries}
	tests := []struct {
		name  string
		run   func() error
		tag   string
		route string
	}{
		{"write of a foreign buffer", func() error {
			return panicError(t, func() { d.WriteBuffer(&buffer{d: other, size: 8}, 0, nil) })
		}, "WebGPUForeignResource", "Device.WriteBuffer.WebGPUForeignResource"},
		{"write outside the buffer", func() error {
			return panicError(t, func() { d.WriteBuffer(&buffer{d: d, size: 8}, 8, make([]byte, 16)) })
		}, "WebGPUWriteOutOfRange", "Device.WriteBuffer.WebGPUWriteOutOfRange"},
		{"level the texture lacks", func() error {
			return panicError(t, func() { (&texture{d: d, desc: explicit.TextureDesc{MipLevels: 1}}).Level(3) })
		}, "WebGPULevelInvalid", "texture.Level.WebGPULevelInvalid"},
		{"texture without a size", func() error {
			_, err := d.NewTexture(explicit.TextureDesc{Width: 0, Height: 4})
			return err
		}, "WebGPUTextureInvalid", "Device.NewTexture.WebGPUTextureInvalid"},
		{"write to a foreign texture", func() error {
			return panicError(t, func() { d.WriteTexture(&texture{d: other}, 0, image.Rect(0, 0, 1, 1), nil, 4) })
		}, "WebGPUForeignResource", "Device.WriteTexture.WebGPUForeignResource"},
		{"pipeline without a shader", func() error {
			_, err := d.NewPipeline(explicit.PipelineDesc{})
			return err
		}, "WebGPUPipelineInvalid", "Device.NewPipeline.WebGPUPipelineInvalid"},
		{"pipeline with an entry point the shader lacks", func() error {
			_, err := d.NewPipeline(explicit.PipelineDesc{Shader: shader, VertexEntry: "nope", FragmentEntry: "nope"})
			return err
		}, "WebGPUPipelineInvalid", "Device.NewPipeline.WebGPUPipelineInvalid"},
		{"pipeline with a shader that is not the renderer's", func() error {
			_, err := d.NewPipeline(explicit.PipelineDesc{
				Shader: unknown, VertexEntry: shader.Entries[0].Name, FragmentEntry: shader.Entries[1].Name,
			})
			return err
		}, "WebGPUPipelineInvalid", "Device.NewPipeline.WebGPUPipelineInvalid"},
		{"bind group of a foreign pipeline", func() error {
			return panicError(t, func() { _, _ = d.NewBindGroup(&pipeline{d: other}, 0, nil) })
		}, "WebGPUForeignResource", "Device.NewBindGroup.WebGPUForeignResource"},
		{"bind group the pipeline lacks", func() error {
			_, err := d.NewBindGroup(&pipeline{d: d}, 0, nil)
			return err
		}, "WebGPUBindGroupInvalid", "Device.NewBindGroup.WebGPUBindGroupInvalid"},
		{"binding the shader lacks", func() error {
			_, err := d.NewBindGroup(&pipeline{d: d, layouts: []js.Value{{}}, kinds: []map[int]explicit.BindingKind{{}}}, 0,
				[]explicit.Binding{{Binding: 3}})
			return err
		}, "WebGPUBindGroupInvalid", "Device.NewBindGroup.WebGPUBindGroupInvalid"},
		{"swapchain on a handle of another kind", func() error {
			_, err := d.newSwapchain(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWayland})
			return err
		}, "WebGPUSurfaceKindUnsupported", "Device.newSwapchain.WebGPUSurfaceKindUnsupported"},
		{"frame begun twice", func() error {
			e := &encoder{d: d}
			d.open = e
			defer func() { d.open = nil }()
			_, err := d.Begin()
			return err
		}, "WebGPUFrameOpen", "Device.Begin.WebGPUFrameOpen"},
		{"submit of a frame that is not open", func() error { return (&encoder{d: d}).Submit() },
			"WebGPUFrameNotOpen", "encoder.Submit.WebGPUFrameNotOpen"},
		{"pass on a foreign target", func() error {
			return panicError(t, func() { (&encoder{d: d}).BeginPass(&texture{d: other}, explicit.LoadLoad, ui.Color{}) })
		}, "WebGPUForeignResource", "encoder.BeginPass.WebGPUForeignResource"},
		{"pass begun before the last one ended", func() error {
			return panicError(t, func() { (&encoder{d: d, pass: &pass{}}).BeginPass(&texture{d: d}, explicit.LoadLoad, ui.Color{}) })
		}, "WebGPUPassNotEnded", "encoder.BeginPass.WebGPUPassNotEnded"},
		{"copy of a foreign texture", func() error {
			return panicError(t, func() {
				(&encoder{d: d}).CopyTexture(&texture{d: other}, image.Rectangle{}, &texture{d: d}, image.Point{})
			})
		}, "WebGPUForeignResource", "encoder.CopyTexture.WebGPUForeignResource"},
		{"read of a foreign texture", func() error {
			return panicError(t, func() { (&encoder{d: d}).ReadTexture(&texture{d: other}, image.Rectangle{}, nil) })
		}, "WebGPUForeignResource", "encoder.ReadTexture.WebGPUForeignResource"},
		{"foreign pipeline in a pass", func() error {
			return panicError(t, func() { (&pass{e: &encoder{d: d}}).SetPipeline(&pipeline{d: other}) })
		}, "WebGPUForeignResource", "pass.SetPipeline.WebGPUForeignResource"},
		{"foreign bind group in a pass", func() error {
			return panicError(t, func() { (&pass{e: &encoder{d: d}}).SetBindGroup(0, nil) })
		}, "WebGPUForeignResource", "pass.SetBindGroup.WebGPUForeignResource"},
		{"foreign vertex buffer in a pass", func() error {
			return panicError(t, func() { (&pass{e: &encoder{d: d}}).SetVertexBuffer(0, &buffer{d: other}, 0) })
		}, "WebGPUForeignResource", "pass.SetVertexBuffer.WebGPUForeignResource"},
		{"foreign index buffer in a pass", func() error {
			return panicError(t, func() { (&pass{e: &encoder{d: d}}).SetIndexBuffer(&buffer{d: other}, 0) })
		}, "WebGPUForeignResource", "pass.SetIndexBuffer.WebGPUForeignResource"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tt.tag, tt.route)
			}
		})
	}
}

// TestCallThatThrows: tryCall returns what the browser threw as WebGPUCallFailed, with the name of
// the call, and configureContext returns it as WebGPUConfigureFailed.
func TestCallThatThrows(t *testing.T) {
	throw := func() js.Value { return js.Global().Get("Function").New("throw new Error('refused')").Invoke() }
	_, err := tryCall("createTexture", throw)
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if appErr.Tag != tagWebGPUCallFailed || !appErr.HasRoute("tryCall.func1.WebGPUCallFailed") || appErr.Metadata["call"] != "createTexture" {
		t.Errorf("got %s via %s %v", appErr.Tag, appErr.Route(), appErr.Metadata)
	}

	ctx := js.Global().Get("Object").New()
	ctx.Set("configure", js.Global().Get("Function").New("throw new Error('refused')"))
	s := &swapchain{d: &Device{}, ctx: ctx}
	err = s.configureContext()
	appErr, ok = errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if appErr.Tag != tagWebGPUConfigureFailed || !appErr.HasRoute("swapchain.configureContext.func1.WebGPUConfigureFailed") {
		t.Errorf("got %s via %s", appErr.Tag, appErr.Route())
	}
}

// TestReadBackThatFailsToMap: a read-back the browser could not map ends with WebGPUReadFailed.
func TestReadBackThatFailsToMap(t *testing.T) {
	p := &Provider{reads: map[int]*readJob{}}
	var got error
	p.reads[1] = &readJob{id: 1, d: &Device{}, done: func(_ *image.RGBA, err error) { got = err }}
	p.onMapFailed(1, "Invalid buffer")
	appErr, ok := errors.FromError(got)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T: %v", got, got)
	}
	if appErr.Tag != tagWebGPUReadFailed || !appErr.HasRoute("Provider.onMapFailed.WebGPUReadFailed") || appErr.Metadata["message"] != "Invalid buffer" {
		t.Errorf("got %s via %s %v", appErr.Tag, appErr.Route(), appErr.Metadata)
	}
}

// TestOpenOfAProviderWithoutADevice: Open says why the device is not there: it failed (by the tag
// that the browser's helper gave), or it is still coming (GPUDevicePending).
func TestOpenOfAProviderWithoutADevice(t *testing.T) {
	tests := []struct {
		name    string
		failure *failure
		tag     string
	}{
		{"no navigator.gpu", &failure{tag: "unavailable", message: "no adapter"}, "WebGPUUnavailable"},
		{"a shader the browser rejects", &failure{tag: "shader", message: "blit:3:1: expected"}, "ShaderModuleInvalid"},
		{"a device the adapter does not give", &failure{tag: "device", message: "requestDevice rejected"}, "WebGPUDeviceFailed"},
		{"a device that is on its way", nil, "GPUDevicePending"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{sup: &supply{request: func() {}}}
			if tt.failure != nil {
				p.sup.deliver(nil, tt.failure)
			}
			_, err := p.Open(gpu.SurfaceHandle{})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute("Provider.Open."+tt.tag) {
				t.Errorf("got %s via %s, want %s via Provider.Open", appErr.Tag, appErr.Route(), tt.tag)
			}
			if got := explicit.IsDevicePending(err); got != (tt.tag == "GPUDevicePending") {
				t.Errorf("IsDevicePending = %v", got)
			}
		})
	}
}
