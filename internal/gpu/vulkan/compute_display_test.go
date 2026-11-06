//go:build linux && glutter_display

package vulkan

import (
	"image"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
)

// These tests need Vulkan but no display: step 30.2 on lavapipe, with the validation layer on. The scenes
// of internal/gpu/explicit/devicetest cover what the device draws and computes; these cover what only
// the Vulkan device does: the features it enables, the debug panics of its calls, a storage image
// with mip levels, and the reads that are empty.

func TestComputeFeaturesOfTheRealDevice(t *testing.T) {
	d := openLavapipe(t)
	f := d.Features()
	if !f.TimestampQuery || f.TimestampPeriod <= 0 {
		t.Errorf("lavapipe has timestamp queries, but Features is %+v", f)
	}
	if !d.info.DrawIndirectFirstInstance || !d.info.FullDrawIndexUint32 {
		t.Errorf("lavapipe has drawIndirectFirstInstance and fullDrawIndexUint32, but the info says %v and %v", d.info.DrawIndirectFirstInstance, d.info.FullDrawIndexUint32)
	}
	set, err := d.NewQuerySet(3)
	if err != nil {
		t.Fatal(err)
	}
	if set.Count() != 3 {
		t.Errorf("the query set has %d queries, want 3", set.Count())
	}
	set.Destroy()
	set.Destroy()
}

func TestComputeCallsPanicOnTheRealDevice(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the design limits are checked in debug builds only")
	}
	d := openLavapipe(t)
	rgba := explicit.TextureFormatRGBA8Unorm
	storage, err := d.NewBuffer(1024, explicit.BufferStorage)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Destroy()
	plain, err := d.NewBuffer(1024, explicit.BufferCopySource)
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Destroy()
	target, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 4, Format: rgba, Usage: explicit.TextureRenderTarget})
	if err != nil {
		t.Fatal(err)
	}
	defer target.Destroy()
	cp, err := d.NewComputePipeline(explicit.ComputePipelineDesc{Shader: devicetest.ShaderCompute, Entry: "cs_square"})
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Destroy()
	ip, err := d.NewPipeline(explicit.PipelineDesc{Shader: devicetest.ShaderInstance, VertexEntry: "vs_instance", FragmentEntry: "fs_instance",
		Topology: explicit.TopologyTriangleStrip, ColorTargets: []explicit.ColorTargetState{{Format: rgba}}})
	if err != nil {
		t.Fatal(err)
	}
	defer ip.Destroy()

	enc, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		tag   string
		route string
		do    func()
	}{
		{"a buffer larger than the limit", "GPULimitExceeded", "Device.NewBuffer->Limits.CheckBuffer->exceeded.GPULimitExceeded",
			func() { _, _ = d.NewBuffer(300<<20, explicit.BufferStorage) }},
		{"a storage offset that is not aligned", "GPULimitExceeded",
			"Device.NewComputeBindGroup->Device.makeBindGroup->Limits.CheckBinding->misaligned.GPULimitExceeded",
			func() {
				_, _ = d.NewComputeBindGroup(cp, 0, []explicit.Binding{{Binding: 1, Buffer: storage, Offset: 4, Size: 64}})
			}},
		{"a storage binding with a buffer that is not storage", "GPUDescriptorInvalid",
			"Device.NewComputeBindGroup->Device.makeBindGroup->Limits.CheckBinding->invalid.GPUDescriptorInvalid",
			func() { _, _ = d.NewComputeBindGroup(cp, 0, []explicit.Binding{{Binding: 1, Buffer: plain}}) }},
		{"a compute pipeline from a vertex entry point", "GPUDescriptorInvalid",
			"Device.NewComputePipeline->Limits.CheckComputePipeline->invalid.GPUDescriptorInvalid",
			func() {
				_, _ = d.NewComputePipeline(explicit.ComputePipelineDesc{Shader: devicetest.ShaderInstance, Entry: "vs_instance"})
			}},
		{"a read outside the buffer", "GPUDescriptorInvalid", "encoder.ReadBuffer->Limits.CheckReadBuffer->invalid.GPUDescriptorInvalid",
			func() { enc.ReadBuffer(plain, 1020, 8, func([]byte, error) {}) }},
		{"a copy onto the same buffer", "GPUDescriptorInvalid", "encoder.CopyBuffer->Limits.CheckCopyBuffer->invalid.GPUDescriptorInvalid",
			func() { enc.CopyBuffer(plain, 0, plain, 4, 8) }},
	}
	for _, tt := range tests {
		err := panicError(tt.do)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
			t.Errorf("%s: got %v, want %s via %s", tt.name, err, tt.tag, tt.route)
		}
	}

	cpass := enc.BeginComputePass(explicit.ComputePassDesc{})
	err = panicError(func() { cpass.Dispatch(70000, 1, 1) })
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "GPULimitExceeded" || !appErr.HasRoute("computePass.Dispatch->Limits.CheckDispatch->exceeded.GPULimitExceeded") {
		t.Errorf("a dispatch of 70000 workgroups: got %v, want GPULimitExceeded via computePass.Dispatch", err)
	}
	err = panicError(func() { cpass.DispatchIndirect(plain, 0) })
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "GPUDescriptorInvalid" || !appErr.HasRoute("computePass.DispatchIndirect->Limits.CheckIndirect->invalid.GPUDescriptorInvalid") {
		t.Errorf("an indirect dispatch from a buffer without BufferIndirect: got %v, want GPUDescriptorInvalid via computePass.DispatchIndirect", err)
	}
	err = panicError(func() { enc.BeginComputePass(explicit.ComputePassDesc{}) })
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagVulkanPassNotEnded || !appErr.HasRoute("encoder.BeginComputePass.VulkanPassNotEnded") {
		t.Errorf("a compute pass inside a compute pass: got %v, want VulkanPassNotEnded", err)
	}
	cpass.End()

	pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target}}})
	pass.SetPipeline(ip)
	err = panicError(func() { pass.Draw(4, 2, 0, 1) })
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "GPUFirstInstanceRead" || !appErr.HasRoute("pass.Draw->pass.checkFirstIndices->CheckFirstIndices.GPUFirstInstanceRead") {
		t.Errorf("a first instance of 1 for a shader that reads instance_index: got %v, want GPUFirstInstanceRead", err)
	}
	err = panicError(func() { pass.Draw(4, 1, 1, 0) })
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "GPUFirstInstanceRead" {
		t.Errorf("a first vertex of 1 for a shader that reads vertex_index: got %v, want GPUFirstInstanceRead", err)
	}
	err = panicError(func() { pass.DrawIndirect(plain, 0) })
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != "GPUDescriptorInvalid" || !appErr.HasRoute("pass.DrawIndirect->Limits.CheckIndirect->invalid.GPUDescriptorInvalid") {
		t.Errorf("an indirect draw from a buffer without BufferIndirect: got %v, want GPUDescriptorInvalid via pass.DrawIndirect", err)
	}
	err = panicError(func() { enc.BeginComputePass(explicit.ComputePassDesc{}) })
	if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagVulkanPassNotEnded || !appErr.HasRoute("encoder.BeginComputePass.VulkanPassNotEnded") {
		t.Errorf("a compute pass inside a render pass: got %v, want VulkanPassNotEnded", err)
	}
	pass.End()
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
}

// TestAStorageImageWithMipLevels binds a texture with several levels as a storage image: the descriptor
// takes the view of level 0, which the validation layer would refuse for all of them.
func TestAStorageImageWithMipLevels(t *testing.T) {
	d := openLavapipe(t)
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, MipLevels: 4, Format: explicit.TextureFormatRGBA8Unorm,
		Usage: explicit.TextureStorage | explicit.TextureSampled | explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	defer tex.Destroy()
	cp, err := d.NewComputePipeline(explicit.ComputePipelineDesc{Shader: devicetest.ShaderCompute, Entry: "cs_paint"})
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Destroy()
	enc, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	g, err := d.NewComputeBindGroup(cp, 1, []explicit.Binding{{Binding: 0, Texture: tex}})
	if err != nil {
		t.Fatal(err)
	}
	pass := enc.BeginComputePass(explicit.ComputePassDesc{})
	pass.SetPipeline(cp)
	pass.SetBindGroup(1, g)
	pass.Dispatch(2, 2, 1)
	pass.End()
	var got []byte
	enc.ReadTextureBytes(tex, image.Rect(0, 0, 4, 4), func(data []byte, _ int, err error) {
		if err != nil {
			t.Error(err)
		}
		got = data
	})
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
	if want := []byte{60, 0, 128, 255}; len(got) < 8 || string(got[4:8]) != string(want) {
		t.Errorf("the second texel of the first row is %v, want %v", got[min(4, len(got)):min(8, len(got))], want)
	}
}

// TestAnEmptyReadAndAnOnCompleteAlone reads no bytes from a buffer, and waits for an encoder that does
// nothing else.
func TestAnEmptyReadAndAnOnCompleteAlone(t *testing.T) {
	d := openLavapipe(t)
	b, err := d.NewBuffer(64, explicit.BufferCopySource)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Destroy()
	enc, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	done, completed := false, false
	enc.ReadBuffer(b, 8, 0, func(data []byte, err error) {
		done = err == nil && len(data) == 0
	})
	enc.OnComplete(func() { completed = true })
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
	if !done || !completed {
		t.Errorf("the empty read was delivered: %v, OnComplete ran: %v; want both", done, completed)
	}
}
