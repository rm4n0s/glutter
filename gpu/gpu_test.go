package gpu_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/gpu"
)

// The shader of the tests, written by hand: the fields are what glutter-gen gpushaders writes.
var testShader = &gpu.Shader{Name: "test", Entries: []gpu.EntryPoint{
	{Name: "vs", Stage: gpu.ShaderStageVertex, Inputs: []gpu.VertexInput{
		{Name: "position", Location: 0, Type: gpu.ShaderTypeFloat32x2},
		{Name: "color", Location: 1, Type: gpu.ShaderTypeFloat32x4},
	}, Bindings: []gpu.ShaderBinding{
		{Name: "camera", Group: 0, Binding: 0, Kind: gpu.BindingUniformBuffer, Size: 64},
	}},
	{Name: "fs", Stage: gpu.ShaderStageFragment, Bindings: []gpu.ShaderBinding{
		{Name: "tex", Group: 1, Binding: 0, Kind: gpu.BindingSampledTexture},
		{Name: "samp", Group: 1, Binding: 1, Kind: gpu.BindingSampler},
	}},
	{Name: "cs", Stage: gpu.ShaderStageCompute, WorkgroupSize: [3]int{8, 8, 1}, Bindings: []gpu.ShaderBinding{
		{Name: "params", Group: 0, Binding: 0, Kind: gpu.BindingUniformBuffer, Size: 16},
		{Name: "data", Group: 0, Binding: 1, Kind: gpu.BindingStorageBuffer, Size: 16},
	}},
}}

func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("the checks of the encoding run in debug builds only")
	}
}

// panicOf runs fn and returns the error it panicked with, or nil.
func panicOf(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(error)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	fn()
	return nil
}

// harness is a tester with a recording GPU device and the gpu.Device of its app.
type harness struct {
	t   *testing.T
	ts  *gluttertest.Tester
	rec *gluttertest.GPURecorder
	d   *gpu.Device
}

func newTester(t *testing.T) *gluttertest.Tester {
	t.Helper()
	return gluttertest.New(t, gluttertest.Opts{GPU: true})
}

// newPlainTester is a tester with no GPU device, as with the software renderer.
func newPlainTester(t *testing.T) *gluttertest.Tester {
	t.Helper()
	return gluttertest.New(t)
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	return finishHarness(t, newTester(t))
}

func finishHarness(t *testing.T, ts *gluttertest.Tester) *harness {
	t.Helper()
	d, err := gpu.DeviceOfBinding(ts.Binding)
	if err != nil {
		t.Fatalf("DeviceOfBinding: %v", err)
	}
	return &harness{t: t, ts: ts, rec: ts.GPUDevice(), d: d}
}

// must panics for an error, which fails the test with its text.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// wantTag fails the test unless err is an *errors.Error with the tag and the route.
func wantTag(t *testing.T, err error, tag, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error with tag %s, got %T: %v", tag, err, err)
	}
	if appErr.Tag != tag {
		t.Fatalf("tag = %s, want %s (route %s)", appErr.Tag, tag, appErr.Route())
	}
	if route != "" && !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), route)
	}
}

func (h *harness) module() *gpu.ShaderModule {
	h.t.Helper()
	return must(h.mustModule())
}

func (h *harness) mustModule() (*gpu.ShaderModule, error) { return h.d.NewShaderModule(testShader) }

func TestDeviceOf(t *testing.T) {
	t.Run("without a GPU the app has none", func(t *testing.T) {
		ts := gluttertest.New(t)
		_, err := gpu.DeviceOfBinding(ts.Binding)
		wantTag(t, err, gpu.TagGPUUnavailable, "DeviceOfBinding->bindingState.current.GPUUnavailable")
	})
	t.Run("with a GPU", func(t *testing.T) {
		h := newHarness(t)
		if h.d.Limits().MaxBindGroups < 4 || h.d.Limits().MaxTextureDimension2D < 1 {
			t.Errorf("limits = %+v", h.d.Limits())
		}
		again, err := gpu.DeviceOfBinding(h.ts.Binding)
		if err != nil || again != h.d {
			t.Errorf("DeviceOfBinding again = %p, %v; want the same device %p", again, err, h.d)
		}
		if !strings.Contains(h.d.Description(), "gluttertest") {
			t.Errorf("Description = %q", h.d.Description())
		}
	})
}

func TestSmokeDraw(t *testing.T) {
	h := newHarness(t)
	m := h.module()
	pipe := must(h.d.NewRenderPipeline(gpu.RenderPipelineDesc{
		Vertex: gpu.VertexState{Module: m, EntryPoint: "vs", Buffers: []gpu.VertexBufferLayout{{ArrayStride: 24, Attributes: []gpu.VertexAttribute{
			{ShaderLocation: 0, Format: gpu.VertexFormatFloat32x2, Offset: 0}, {ShaderLocation: 1, Format: gpu.VertexFormatFloat32x4, Offset: 8}}}}},
		Fragment: gpu.FragmentState{Module: m, EntryPoint: "fs", Targets: []gpu.ColorTargetState{{Format: gpu.TextureFormatRGBA8Unorm}}},
	}))
	tex := must(h.d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 16, Height: 16}, Format: gpu.TextureFormatRGBA8Unorm,
		Usage: gpu.TextureUsageRenderAttachment | gpu.TextureUsageTextureBinding | gpu.TextureUsageCopySrc}))
	vb := must(h.d.NewBuffer(gpu.BufferDesc{Size: 72, Usage: gpu.BufferUsageVertex | gpu.BufferUsageCopyDst}))
	ub := must(h.d.NewBuffer(gpu.BufferDesc{Size: 256, Usage: gpu.BufferUsageUniform | gpu.BufferUsageCopyDst}))
	smp := must(h.d.NewSampler(gpu.SamplerDesc{}))
	g0 := must(h.d.NewBindGroup(gpu.BindGroupDesc{Layout: pipe.BindGroupLayout(0), Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: ub, Size: 64}}}))
	g1 := must(h.d.NewBindGroup(gpu.BindGroupDesc{Layout: pipe.BindGroupLayout(1), Entries: []gpu.BindGroupEntry{
		{Binding: 0, TextureView: tex.View()}, {Binding: 1, Sampler: smp}}}))
	h.d.Queue().WriteBuffer(vb, 0, make([]byte, 72))
	enc := h.d.NewCommandEncoder()
	pass := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: tex.View(), ClearValue: gpu.Color{A: 1}}}})
	pass.SetPipeline(pipe)
	pass.SetBindGroup(0, g0, 0)
	pass.SetBindGroup(1, g1)
	pass.SetVertexBuffer(0, vb, 0)
	pass.Draw(3, 1, 0, 0)
	pass.End()
	if err := h.d.Queue().Submit(enc.Finish()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	t.Log(strings.Join(h.rec.Calls(), "\n"))
}
