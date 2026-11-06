package gpu_test

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/internal/gpuhost"
)

// TestTheRenderersFirstFrame: before the renderer opens its device there is none, and OnDevice is called when it comes.
func TestTheRenderersFirstFrame(t *testing.T) {
	app := gpuhost.NewBinding(t, nil)
	_, err := gpu.DeviceOfBinding(app.Binding)
	wantTag(t, err, gpu.TagGPUDeviceNotReady, "DeviceOfBinding->bindingState.current.GPUDeviceNotReady")
	var got []*gpu.Device
	gpu.OnDevice(app.Binding, func(d *gpu.Device) { got = append(got, d) })
	app.RunPosted()
	if len(got) != 0 {
		t.Fatalf("OnDevice called fn without a device")
	}
	app.Host.Open(fakedevice.New())
	app.RunPosted()
	if len(got) != 1 {
		t.Fatalf("fn was called %d times after the device came", len(got))
	}
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil || d != got[0] {
		t.Errorf("DeviceOfBinding = %p, %v; want the device of OnDevice %p", d, err, got[0])
	}
}

// TestTheBackendsErrors: what the backend returns for a resource comes back untouched, except a pipeline that the backend
// could not compile, which is GPUShaderCompileFailed with the backend's error as the cause.
func TestTheBackendsErrors(t *testing.T) {
	const backendTag = "TestBackendFailed"
	tests := []struct {
		name  string
		err   error
		do    func(h *harness) error
		tag   string
		route string
	}{
		{"memory for a buffer", errors.New(gpu.TagGPUOutOfMemory, "no memory"), func(h *harness) error {
			_, err := h.d.NewBuffer(gpu.BufferDesc{Size: 4, Usage: gpu.BufferUsageVertex})
			return err
		}, gpu.TagGPUOutOfMemory, ""},
		{"memory for a texture", errors.New(gpu.TagGPUOutOfMemory, "no memory"), func(h *harness) error {
			_, err := h.d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 4, Height: 4}, Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding})
			return err
		}, gpu.TagGPUOutOfMemory, ""},
		{"memory for a pipeline", errors.New(gpu.TagGPUOutOfMemory, "no memory"), func(h *harness) error {
			_, err := h.d.NewRenderPipeline(h.pipelineDesc(h.module(), nil))
			return err
		}, gpu.TagGPUOutOfMemory, ""},
		{"a render pipeline the backend cannot compile", errors.New(backendTag, "bad shader"), func(h *harness) error {
			_, err := h.d.NewRenderPipeline(h.pipelineDesc(h.module(), nil))
			return err
		}, gpu.TagGPUShaderCompileFailed, "Device.NewRenderPipeline.GPUShaderCompileFailed"},
		{"a compute pipeline the backend cannot compile", errors.New(backendTag, "bad shader"), func(h *harness) error {
			_, err := h.d.NewComputePipeline(gpu.ComputePipelineDesc{Module: h.module(), EntryPoint: "cs"})
			return err
		}, gpu.TagGPUShaderCompileFailed, "Device.NewComputePipeline.GPUShaderCompileFailed"},
		{"a read that cannot start", errors.New(backendTag, "no encoder"), func(h *harness) error {
			b := must(h.d.NewBuffer(gpu.BufferDesc{Size: 16, Usage: gpu.BufferUsageCopySrc}))
			var got error
			h.rec.Device().Err = errors.New(backendTag, "no encoder")
			b.Read(0, 16, func(_ []byte, err error) { got = err })
			h.ts.Pump()
			return got
		}, gpu.TagGPUReadFailed, "Buffer.Read.GPUReadFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			// The module is made before the backend is told to fail, as the pipeline is the call that fails.
			h.module()
			if tt.tag != gpu.TagGPUReadFailed {
				h.rec.Device().Err = tt.err
			}
			err := tt.do(h)
			wantTag(t, err, tt.tag, tt.route)
			if appErr, _ := errors.FromError(err); tt.tag == gpu.TagGPUShaderCompileFailed && appErr.OriginalErr == nil {
				t.Errorf("the backend's error is not the cause")
			}
		})
	}
}

// TestProgrammerErrorsHaveARoute: the panics of the encoding checks, of a resource used after Release and of another
// goroutine come from one function each.
func TestProgrammerErrorsHaveARoute(t *testing.T) {
	skipUnlessDebug(t)
	h := newHarness(t)
	s := newSceneOn(t, h)
	err := panicOf(func() {
		enc := h.d.NewCommandEncoder()
		p := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.color.View()}}})
		p.Draw(3, 1, 0, 0)
	})
	wantTag(t, err, gpu.TagGPUEncodingInvalid, "RenderPass.Draw->RenderPass.checkDraw->misuse.GPUEncodingInvalid")
	s.vertices.Release()
	err = panicOf(func() {
		p := h.d.NewCommandEncoder().BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.color.View()}}})
		p.SetVertexBuffer(0, s.vertices, 0)
	})
	wantTag(t, err, gpu.TagGPUResourceReleased, "RenderPass.SetVertexBuffer->resource.usable.GPUResourceReleased")
	done := make(chan error)
	go func() { done <- panicOf(func() { _, _ = h.d.NewBuffer(gpu.BufferDesc{}) }) }()
	wantTag(t, <-done, gpu.TagGPUWrongThread, "Device.NewBuffer->Device.assertThread->assertThread.GPUWrongThread")
}
