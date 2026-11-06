package gpu_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gpu"
)

// scene is a pipeline with everything it needs, for the tests of the encoding.
type scene struct {
	h           *harness
	pipe        *gpu.RenderPipeline
	depthPipe   *gpu.RenderPipeline
	compute     *gpu.ComputePipeline
	color       *gpu.Texture
	small       *gpu.Texture
	depth       *gpu.Texture
	sampled     *gpu.Texture
	volume      *gpu.Texture
	vertices    *gpu.Buffer
	indices     *gpu.Buffer
	uniform     *gpu.Buffer
	storage     *gpu.Buffer
	indirect    *gpu.Buffer
	src, dst    *gpu.Buffer
	plain       *gpu.Buffer
	camera      *gpu.BindGroup
	textures    *gpu.BindGroup
	computeData *gpu.BindGroup
}

func newScene(t *testing.T) *scene { return newSceneOn(t, newHarness(t)) }

func newSceneOn(t *testing.T, h *harness) *scene {
	t.Helper()
	d := h.d
	m := h.module()
	s := &scene{h: h}
	s.pipe = must(d.NewRenderPipeline(h.pipelineDesc(m, nil)))
	s.depthPipe = must(d.NewRenderPipeline(h.pipelineDesc(m, func(p *gpu.RenderPipelineDesc) {
		p.DepthStencil = &gpu.DepthStencilState{Format: gpu.TextureFormatDepth32Float, DepthWriteEnabled: true, DepthCompare: gpu.CompareFunctionLess}
	})))
	s.compute = must(d.NewComputePipeline(gpu.ComputePipelineDesc{Module: m, EntryPoint: "cs"}))
	rt := gpu.TextureUsageRenderAttachment | gpu.TextureUsageTextureBinding | gpu.TextureUsageCopySrc
	s.color = must(d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 16, Height: 16}, Format: gpu.TextureFormatRGBA8Unorm, Usage: rt}))
	s.small = must(d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: gpu.TextureFormatRGBA8Unorm, Usage: rt}))
	s.depth = must(d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 16, Height: 16}, Format: gpu.TextureFormatDepth32Float, Usage: gpu.TextureUsageRenderAttachment}))
	s.sampled = must(d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 16, Height: 16}, Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding}))
	s.volume = must(d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 4, Height: 4, DepthOrArrayLayers: 4}, Kind: gpu.TextureKind3D,
		Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding | gpu.TextureUsageCopySrc}))
	s.vertices = must(d.NewBuffer(gpu.BufferDesc{Size: 72, Usage: gpu.BufferUsageVertex | gpu.BufferUsageCopyDst}))
	s.indices = must(d.NewBuffer(gpu.BufferDesc{Size: 12, Usage: gpu.BufferUsageIndex | gpu.BufferUsageCopyDst}))
	s.uniform = must(d.NewBuffer(gpu.BufferDesc{Size: 1024, Usage: gpu.BufferUsageUniform | gpu.BufferUsageCopyDst}))
	s.storage = must(d.NewBuffer(gpu.BufferDesc{Size: 1024, Usage: gpu.BufferUsageStorage}))
	s.indirect = must(d.NewBuffer(gpu.BufferDesc{Size: 64, Usage: gpu.BufferUsageIndirect | gpu.BufferUsageCopyDst}))
	s.src = must(d.NewBuffer(gpu.BufferDesc{Size: 4096, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst}))
	s.dst = must(d.NewBuffer(gpu.BufferDesc{Size: 4096, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst}))
	s.plain = must(d.NewBuffer(gpu.BufferDesc{Size: 64, Usage: gpu.BufferUsageStorage}))
	smp := must(d.NewSampler(gpu.SamplerDesc{}))
	s.camera = must(d.NewBindGroup(gpu.BindGroupDesc{Layout: s.pipe.BindGroupLayout(0), Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: s.uniform, Size: 64}}}))
	s.textures = must(d.NewBindGroup(gpu.BindGroupDesc{Layout: s.pipe.BindGroupLayout(1), Entries: []gpu.BindGroupEntry{
		{Binding: 0, TextureView: s.sampled.View()}, {Binding: 1, Sampler: smp}}}))
	s.computeData = must(d.NewBindGroup(gpu.BindGroupDesc{Layout: s.compute.BindGroupLayout(0), Entries: []gpu.BindGroupEntry{
		{Binding: 0, Buffer: s.uniform, Size: 16}, {Binding: 1, Buffer: s.storage, Size: 16}}}))
	return s
}

// pass begins a pass on the color texture with the pipeline and the groups set.
func (s *scene) pass(enc *gpu.CommandEncoder) *gpu.RenderPass {
	p := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.color.View()}}})
	p.SetPipeline(s.pipe)
	p.SetBindGroup(0, s.camera, 0)
	p.SetBindGroup(1, s.textures)
	p.SetVertexBuffer(0, s.vertices, 0)
	return p
}

func (s *scene) submit(enc *gpu.CommandEncoder) error { return s.h.d.Queue().Submit(enc.Finish()) }

func TestTheSceneEncodes(t *testing.T) {
	tests := []struct {
		name string
		do   func(s *scene, enc *gpu.CommandEncoder)
		ops  []string
	}{
		{"a draw", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.Draw(3, 1, 0, 0)
			p.End()
		}, []string{"Begin", "BeginRenderPass", "SetPipeline", "NewBindGroup", "SetBindGroup", "NewBindGroup", "SetBindGroup", "SetVertexBuffer", "Draw", "EndPass", "OnComplete", "Submit"}},
		{"an indexed draw", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetIndexBuffer(s.indices, 0)
			p.DrawIndexed(6, 1, 0, 0, 0)
			p.End()
		}, []string{"DrawIndexed"}},
		{"a wide index buffer", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetIndexBuffer32(s.indices, 0)
			p.DrawIndexed(3, 1, 0, 0, 0)
			p.End()
		}, []string{"SetIndexBuffer32", "DrawIndexed"}},
		{"indirect draws", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetIndexBuffer(s.indices, 0)
			p.DrawIndirect(s.indirect, 0)
			p.DrawIndexedIndirect(s.indirect, 16)
			p.End()
		}, []string{"DrawIndirect", "DrawIndexedIndirect"}},
		{"the groups set before the pipeline", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.color.View()}}})
			p.SetBindGroup(0, s.camera, 0)
			p.SetBindGroup(1, s.textures)
			p.SetPipeline(s.pipe)
			p.SetVertexBuffer(0, s.vertices, 0)
			p.Draw(3, 1, 0, 0)
			p.End()
		}, []string{"SetPipeline", "NewBindGroup", "SetBindGroup", "NewBindGroup", "SetBindGroup", "Draw"}},
		{"a depth pass with scissor and stencil", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginRenderPass(gpu.RenderPassDesc{
				ColorAttachments:       []gpu.RenderPassColorAttachment{{View: s.color.View(), LoadOp: gpu.LoadOpClear, ClearValue: gpu.Color{R: 1, A: 1}}},
				DepthStencilAttachment: &gpu.RenderPassDepthStencilAttachment{View: s.depth.View(), DepthClearValue: 1},
			})
			p.SetPipeline(s.depthPipe)
			p.SetBindGroup(0, s.camera, 256)
			p.SetBindGroup(1, s.textures)
			p.SetVertexBuffer(0, s.vertices, 0)
			p.SetScissorRect(2, 2, 8, 8)
			p.SetStencilReference(1)
			p.Draw(3, 1, 0, 0)
			p.End()
		}, []string{"BeginRenderPass", "SetScissor", "SetStencilReference", "Draw"}},
		{"a compute pass", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginComputePass(gpu.ComputePassDesc{})
			p.SetPipeline(s.compute)
			p.SetBindGroup(0, s.computeData, 0)
			p.DispatchWorkgroups(4, 4, 1)
			p.DispatchWorkgroupsIndirect(s.indirect, 0)
			p.End()
		}, []string{"BeginComputePass", "SetComputePipeline", "NewComputeBindGroup", "SetComputeBindGroup", "Dispatch", "DispatchIndirect", "EndComputePass"}},
		{"copies", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyBufferToBuffer(s.src, 0, s.dst, 64, 128)
			enc.CopyBufferToTexture(gpu.BufferCopy{Buffer: s.src, Layout: gpu.DataLayout{BytesPerRow: 256}}, gpu.TextureCopy{Texture: s.color}, gpu.Extent3D{Width: 16, Height: 4, DepthOrArrayLayers: 1})
			enc.CopyTextureToBuffer(gpu.TextureCopy{Texture: s.color}, gpu.BufferCopy{Buffer: s.dst}, gpu.Extent3D{Width: 16, Height: 16, DepthOrArrayLayers: 1})
			enc.CopyTextureToTexture(gpu.TextureCopy{Texture: s.color}, gpu.TextureCopy{Texture: s.small, Origin: gpu.Origin3D{}}, gpu.Extent3D{Width: 8, Height: 8, DepthOrArrayLayers: 1})
		}, []string{"CopyBuffer", "CopyBufferToTexture", "CopyTextureToBuffer", "CopyTexture"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newScene(t)
			s.h.rec.Reset()
			enc := s.h.d.NewCommandEncoder()
			tt.do(s, enc)
			if err := s.submit(enc); err != nil {
				t.Fatalf("Submit: %v", err)
			}
			ops := s.h.rec.Ops()
			// The ops of the test are a subsequence of the ops that were recorded.
			i := 0
			for _, op := range ops {
				if i < len(tt.ops) && op == tt.ops[i] {
					i++
				}
			}
			if i != len(tt.ops) {
				t.Errorf("ops = %v\nwant them to contain, in order, %v", ops, tt.ops)
			}
		})
	}
}

func TestEncodingMisuse(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		do   func(s *scene, enc *gpu.CommandEncoder)
		tag  string
	}{
		{"a draw with no pipeline", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.color.View()}}})
			p.Draw(3, 1, 0, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"a pipeline for other targets", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.color.View()}},
				DepthStencilAttachment: &gpu.RenderPassDepthStencilAttachment{View: s.depth.View()}})
			p.SetPipeline(s.pipe)
		}, gpu.TagGPUEncodingInvalid},
		{"a draw with a bind group missing", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.color.View()}}})
			p.SetPipeline(s.pipe)
			p.SetBindGroup(0, s.camera, 0)
			p.SetVertexBuffer(0, s.vertices, 0)
			p.Draw(3, 1, 0, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"a bind group of another layout", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetBindGroup(1, s.camera, 0) // made for group 0
		}, gpu.TagGPUEncodingInvalid},
		{"a draw with a bind group of another layout", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.color.View()}}})
			p.SetPipeline(s.pipe)
			p.SetBindGroup(0, s.computeData, 0, 0) // a compute group 0, with two uniform and storage bindings
			p.SetBindGroup(1, s.textures)
			p.SetVertexBuffer(0, s.vertices, 0)
			p.Draw(3, 1, 0, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"no dynamic offset", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetBindGroup(0, s.camera)
		}, gpu.TagGPUEncodingInvalid},
		{"a dynamic offset that is not aligned", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetBindGroup(0, s.camera, 16)
		}, gpu.TagGPUEncodingInvalid},
		{"a dynamic offset past the buffer", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetBindGroup(0, s.camera, 1024)
		}, gpu.TagGPUEncodingInvalid},
		{"a draw with no vertex buffer", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.color.View()}}})
			p.SetPipeline(s.pipe)
			p.SetBindGroup(0, s.camera, 0)
			p.SetBindGroup(1, s.textures)
			p.Draw(3, 1, 0, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"a draw past the end of the vertex buffer", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.Draw(4, 1, 0, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"a vertex buffer without the use", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetVertexBuffer(0, s.uniform, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"a vertex buffer in a slot past the limit", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetVertexBuffer(8, s.vertices, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"an indexed draw with no index buffer", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.DrawIndexed(3, 1, 0, 0, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"an indexed draw past the end of the index buffer", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetIndexBuffer(s.indices, 0)
			p.DrawIndexed(7, 1, 0, 0, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"an index buffer without the use", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetIndexBuffer(s.vertices, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"an indirect draw from a buffer without the use", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.DrawIndirect(s.vertices, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"an indirect draw past the end", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.DrawIndirect(s.indirect, 56)
		}, gpu.TagGPUEncodingInvalid},
		{"a scissor outside the attachments", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.SetScissorRect(10, 10, 10, 10)
		}, gpu.TagGPUEncodingInvalid},
		{"a command after the pass ended", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			p.End()
			p.Draw(3, 1, 0, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"a copy while a pass is open", func(s *scene, enc *gpu.CommandEncoder) {
			s.pass(enc)
			enc.CopyBufferToBuffer(s.src, 0, s.dst, 0, 4)
		}, gpu.TagGPUEncodingInvalid},
		{"finishing with a pass open", func(s *scene, enc *gpu.CommandEncoder) {
			s.pass(enc)
			enc.Finish()
		}, gpu.TagGPUEncodingInvalid},
		{"a pass with no attachment", func(s *scene, enc *gpu.CommandEncoder) {
			enc.BeginRenderPass(gpu.RenderPassDesc{})
		}, gpu.TagGPUEncodingInvalid},
		{"attachments of two sizes", func(s *scene, enc *gpu.CommandEncoder) {
			enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.color.View()}, {View: s.small.View()}}})
		}, gpu.TagGPUEncodingInvalid},
		{"an attachment that is not a render attachment", func(s *scene, enc *gpu.CommandEncoder) {
			enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.sampled.View()}}})
		}, gpu.TagGPUEncodingInvalid},
		{"a depth attachment of a color format", func(s *scene, enc *gpu.CommandEncoder) {
			enc.BeginRenderPass(gpu.RenderPassDesc{DepthStencilAttachment: &gpu.RenderPassDepthStencilAttachment{View: s.color.View()}})
		}, gpu.TagGPUEncodingInvalid},
		{"a buffer copied onto itself", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyBufferToBuffer(s.src, 0, s.src, 64, 4)
		}, gpu.TagGPUEncodingInvalid},
		{"a buffer copy that is not a multiple of 4", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyBufferToBuffer(s.src, 0, s.dst, 0, 3)
		}, gpu.TagGPUEncodingInvalid},
		{"a buffer copy past the end", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyBufferToBuffer(s.src, 4000, s.dst, 0, 200)
		}, gpu.TagGPUEncodingInvalid},
		{"a buffer copy without the use", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyBufferToBuffer(s.src, 0, s.plain, 0, 4)
		}, gpu.TagGPUEncodingInvalid},
		{"rows that are not aligned", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyBufferToTexture(gpu.BufferCopy{Buffer: s.src, Layout: gpu.DataLayout{BytesPerRow: 100}}, gpu.TextureCopy{Texture: s.color}, gpu.Extent3D{Width: 16, Height: 4, DepthOrArrayLayers: 1})
		}, gpu.TagGPUEncodingInvalid},
		{"a box past the texture", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyBufferToTexture(gpu.BufferCopy{Buffer: s.src}, gpu.TextureCopy{Texture: s.color, Origin: gpu.Origin3D{X: 8}}, gpu.Extent3D{Width: 16, Height: 4, DepthOrArrayLayers: 1})
		}, gpu.TagGPUEncodingInvalid},
		{"rows past the buffer", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyBufferToTexture(gpu.BufferCopy{Buffer: s.src, Layout: gpu.DataLayout{Offset: 4000}}, gpu.TextureCopy{Texture: s.color}, gpu.Extent3D{Width: 16, Height: 4, DepthOrArrayLayers: 1})
		}, gpu.TagGPUEncodingInvalid},
		{"a copy into a depth texture", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyBufferToTexture(gpu.BufferCopy{Buffer: s.src}, gpu.TextureCopy{Texture: s.depth}, gpu.Extent3D{Width: 4, Height: 4, DepthOrArrayLayers: 1})
		}, gpu.TagGPUEncodingInvalid},
		{"a read of a texture without the use", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyTextureToBuffer(gpu.TextureCopy{Texture: s.sampled}, gpu.BufferCopy{Buffer: s.dst}, gpu.Extent3D{Width: 4, Height: 4, DepthOrArrayLayers: 1})
		}, gpu.TagGPUEncodingInvalid},
		{"a texture copy between formats", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyTextureToTexture(gpu.TextureCopy{Texture: s.color}, gpu.TextureCopy{Texture: s.depth}, gpu.Extent3D{Width: 4, Height: 4, DepthOrArrayLayers: 1})
		}, gpu.TagGPUEncodingInvalid},
		{"a copy of a volume", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyTextureToTexture(gpu.TextureCopy{Texture: s.volume}, gpu.TextureCopy{Texture: s.volume, Origin: gpu.Origin3D{Z: 2}}, gpu.Extent3D{Width: 4, Height: 4, DepthOrArrayLayers: 1})
		}, gpu.TagGPUEncodingInvalid},
		{"overlapping boxes", func(s *scene, enc *gpu.CommandEncoder) {
			enc.CopyTextureToTexture(gpu.TextureCopy{Texture: s.color}, gpu.TextureCopy{Texture: s.color, Origin: gpu.Origin3D{X: 2}}, gpu.Extent3D{Width: 8, Height: 8, DepthOrArrayLayers: 1})
		}, gpu.TagGPUEncodingInvalid},
		{"a dispatch with no pipeline", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginComputePass(gpu.ComputePassDesc{})
			p.DispatchWorkgroups(1, 1, 1)
		}, gpu.TagGPUEncodingInvalid},
		{"a dispatch with a bind group missing", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginComputePass(gpu.ComputePassDesc{})
			p.SetPipeline(s.compute)
			p.DispatchWorkgroups(1, 1, 1)
		}, gpu.TagGPUEncodingInvalid},
		{"a dispatch over the limit", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginComputePass(gpu.ComputePassDesc{})
			p.SetPipeline(s.compute)
			p.SetBindGroup(0, s.computeData, 0)
			p.DispatchWorkgroups(1<<20, 1, 1)
		}, gpu.TagGPUEncodingInvalid},
		{"an indirect dispatch from a buffer without the use", func(s *scene, enc *gpu.CommandEncoder) {
			p := enc.BeginComputePass(gpu.ComputePassDesc{})
			p.SetPipeline(s.compute)
			p.SetBindGroup(0, s.computeData, 0)
			p.DispatchWorkgroupsIndirect(s.storage, 0)
		}, gpu.TagGPUEncodingInvalid},
		{"timestamps without the feature", func(s *scene, enc *gpu.CommandEncoder) {
			enc.BeginComputePass(gpu.ComputePassDesc{TimestampWrites: &gpu.PassTimestampWrites{}})
		}, gpu.TagGPUEncodingInvalid},
		{"a released buffer in a command", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			s.vertices.Release()
			p.SetVertexBuffer(0, s.vertices, 0)
		}, gpu.TagGPUResourceReleased},
		{"a released texture in a pass", func(s *scene, enc *gpu.CommandEncoder) {
			s.color.Release()
			enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: s.color.View()}}})
		}, gpu.TagGPUResourceReleased},
		{"a released bind group", func(s *scene, enc *gpu.CommandEncoder) {
			p := s.pass(enc)
			s.camera.Release()
			p.SetBindGroup(0, s.camera, 0)
		}, gpu.TagGPUResourceReleased},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newScene(t)
			enc := s.h.d.NewCommandEncoder()
			err := panicOf(func() { tt.do(s, enc) })
			if err == nil {
				t.Fatal("no panic")
			}
			wantTag(t, err, tt.tag, "")
		})
	}
}

func TestQueueMisuse(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		do   func(s *scene)
	}{
		{"a write into a buffer without the use", func(s *scene) { s.h.d.Queue().WriteBuffer(s.storage, 0, make([]byte, 4)) }},
		{"a write that is not a multiple of 4", func(s *scene) { s.h.d.Queue().WriteBuffer(s.vertices, 0, make([]byte, 3)) }},
		{"a write past the end", func(s *scene) { s.h.d.Queue().WriteBuffer(s.vertices, 68, make([]byte, 8)) }},
		{"a texture write past the level", func(s *scene) {
			s.h.d.Queue().WriteTexture(gpu.TextureCopy{Texture: s.color, Origin: gpu.Origin3D{X: 12}}, make([]byte, 4096), gpu.DataLayout{}, gpu.Extent3D{Width: 8, Height: 4, DepthOrArrayLayers: 1})
		}},
		{"a texture write with too little data", func(s *scene) {
			s.h.d.Queue().WriteTexture(gpu.TextureCopy{Texture: s.color}, make([]byte, 10), gpu.DataLayout{}, gpu.Extent3D{Width: 8, Height: 4, DepthOrArrayLayers: 1})
		}},
		{"a command buffer submitted twice", func(s *scene) {
			cb := s.h.d.NewCommandEncoder().Finish()
			_ = s.h.d.Queue().Submit(cb)
			_ = s.h.d.Queue().Submit(cb)
		}},
		{"a command buffer of another device", func(s *scene) {
			other := newHarness(t)
			_ = s.h.d.Queue().Submit(other.d.NewCommandEncoder().Finish())
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newScene(t)
			err := panicOf(func() { tt.do(s) })
			if err == nil {
				t.Fatal("no panic")
			}
			wantTag(t, err, gpu.TagGPUEncodingInvalid, "")
		})
	}
}
