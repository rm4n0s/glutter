package gpu_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gpu"
)

const (
	invalid = gpu.TagGPUInvalidDescriptor
	limit   = gpu.TagGPULimitExceeded
)

// colorPipeline is a pipeline description of the test shader with the mutations of a test applied.
func (h *harness) pipelineDesc(m *gpu.ShaderModule, mutate func(*gpu.RenderPipelineDesc)) gpu.RenderPipelineDesc {
	desc := gpu.RenderPipelineDesc{
		Label: "p",
		Vertex: gpu.VertexState{Module: m, EntryPoint: "vs", Buffers: []gpu.VertexBufferLayout{{ArrayStride: 24, Attributes: []gpu.VertexAttribute{
			{ShaderLocation: 0, Format: gpu.VertexFormatFloat32x2, Offset: 0}, {ShaderLocation: 1, Format: gpu.VertexFormatFloat32x4, Offset: 8}}}}},
		Fragment: gpu.FragmentState{Module: m, EntryPoint: "fs", Targets: []gpu.ColorTargetState{{Format: gpu.TextureFormatRGBA8Unorm}}},
	}
	if mutate != nil {
		mutate(&desc)
	}
	return desc
}

func TestBuffers(t *testing.T) {
	tests := []struct {
		name  string
		desc  gpu.BufferDesc
		tag   string
		route string
	}{
		{"a vertex buffer", gpu.BufferDesc{Size: 64, Usage: gpu.BufferUsageVertex}, "", ""},
		{"every use", gpu.BufferDesc{Size: 64, Usage: gpu.BufferUsageVertex | gpu.BufferUsageIndex | gpu.BufferUsageUniform | gpu.BufferUsageStorage |
			gpu.BufferUsageIndirect | gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst | gpu.BufferUsageQueryResolve}, "", ""},
		{"no size", gpu.BufferDesc{Usage: gpu.BufferUsageVertex}, invalid, "Device.NewBuffer->Device.validateBuffer.GPUInvalidDescriptor"},
		{"no use", gpu.BufferDesc{Size: 64}, invalid, "Device.NewBuffer->Device.validateBuffer.GPUInvalidDescriptor"},
		{"a use that does not exist", gpu.BufferDesc{Size: 64, Usage: gpu.BufferUsage(1 << 20)}, invalid, "Device.NewBuffer->Device.validateBuffer.GPUInvalidDescriptor"},
		{"a size over the limit", gpu.BufferDesc{Size: 1 << 40, Usage: gpu.BufferUsageVertex}, limit, "Device.NewBuffer->Device.validateBuffer.GPULimitExceeded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			b, err := h.d.NewBuffer(tt.desc)
			if tt.tag == "" {
				if err != nil {
					t.Fatalf("NewBuffer: %v", err)
				}
				if b.Size() != tt.desc.Size || b.Usage() != tt.desc.Usage {
					t.Errorf("size, usage = %d, %d", b.Size(), b.Usage())
				}
				return
			}
			wantTag(t, err, tt.tag, tt.route)
			if b != nil {
				t.Errorf("a buffer came with the error")
			}
			if n := h.d.Resources().Total(); n != 0 {
				t.Errorf("the failed call left %d resources", n)
			}
		})
	}
}

func TestTextures(t *testing.T) {
	rgba := gpu.TextureFormatRGBA8Unorm
	use := gpu.TextureUsageTextureBinding
	tests := []struct {
		name  string
		desc  gpu.TextureDesc
		tag   string
		route string
	}{
		{"a 2D texture", gpu.TextureDesc{Size: gpu.Extent3D{Width: 64, Height: 32}, Format: rgba, Usage: use}, "", ""},
		{"with mip levels", gpu.TextureDesc{Size: gpu.Extent3D{Width: 64, Height: 32}, Format: rgba, MipLevelCount: 6, Usage: use}, "", ""},
		{"an array", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8, DepthOrArrayLayers: 4}, Kind: gpu.TextureKind2DArray, Format: rgba, Usage: use}, "", ""},
		{"a cube", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Kind: gpu.TextureKindCube, Format: rgba, Usage: use}, "", ""},
		{"a volume", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8, DepthOrArrayLayers: 8}, Kind: gpu.TextureKind3D, Format: rgba, Usage: use}, "", ""},
		{"a depth texture", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: gpu.TextureFormatDepth32Float, Usage: gpu.TextureUsageRenderAttachment}, "", ""},
		{"multisampled", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: rgba, SampleCount: 4, Usage: gpu.TextureUsageRenderAttachment}, "", ""},
		{"a storage texture", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: rgba, Usage: gpu.TextureUsageStorageBinding}, "", ""},
		{"no format", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Usage: use}, invalid, "Device.NewTexture->Device.validateTexture.GPUInvalidDescriptor"},
		{"no size", gpu.TextureDesc{Format: rgba, Usage: use}, invalid, "Device.NewTexture->Device.validateTexture.GPUInvalidDescriptor"},
		{"no use", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: rgba}, invalid, "Device.NewTexture->Device.validateTexture.GPUInvalidDescriptor"},
		{"a use the format does not allow", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: gpu.TextureFormatDepth32Float, Usage: gpu.TextureUsageStorageBinding},
			invalid, "Device.NewTexture->Device.validateTexture.GPUInvalidDescriptor"},
		{"more levels than the size has", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: rgba, MipLevelCount: 5, Usage: use},
			invalid, "Device.NewTexture->Device.validateTexture.GPUInvalidDescriptor"},
		{"layers on a 2D texture", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8, DepthOrArrayLayers: 3}, Format: rgba, Usage: use},
			invalid, "Device.NewTexture->Device.validateTexture.GPUInvalidDescriptor"},
		{"a cube that is not square", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 4}, Kind: gpu.TextureKindCube, Format: rgba, Usage: use},
			invalid, "Device.NewTexture->Device.validateTexture.GPUInvalidDescriptor"},
		{"a volume drawn into", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8, DepthOrArrayLayers: 2}, Kind: gpu.TextureKind3D, Format: rgba, Usage: gpu.TextureUsageRenderAttachment},
			invalid, "Device.NewTexture->Device.validateTexture.GPUInvalidDescriptor"},
		{"three samples", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: rgba, SampleCount: 3, Usage: gpu.TextureUsageRenderAttachment},
			invalid, "Device.NewTexture->Device.validateTexture.GPUInvalidDescriptor"},
		{"a multisampled texture that is sampled", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: rgba, SampleCount: 4, Usage: use | gpu.TextureUsageRenderAttachment},
			invalid, "Device.NewTexture->Device.validateTexture.GPUInvalidDescriptor"},
		{"a side over the limit", gpu.TextureDesc{Size: gpu.Extent3D{Width: 1 << 20, Height: 8}, Format: rgba, Usage: use},
			limit, "Device.NewTexture->Device.validateTexture.GPULimitExceeded"},
		{"layers over the limit", gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8, DepthOrArrayLayers: 1 << 20}, Kind: gpu.TextureKind2DArray, Format: rgba, Usage: use},
			limit, "Device.NewTexture->Device.validateTexture.GPULimitExceeded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			tex, err := h.d.NewTexture(tt.desc)
			if tt.tag == "" {
				if err != nil {
					t.Fatalf("NewTexture: %v", err)
				}
				if tex.Format() != tt.desc.Format || tex.Size().Width != tt.desc.Size.Width {
					t.Errorf("format, size = %v, %v", tex.Format(), tex.Size())
				}
				return
			}
			wantTag(t, err, tt.tag, tt.route)
			if tex != nil {
				t.Errorf("a texture came with the error")
			}
		})
	}
}

func TestSamplersAndQuerySets(t *testing.T) {
	t.Run("a sampler", func(t *testing.T) {
		h := newHarness(t)
		must(h.d.NewSampler(gpu.SamplerDesc{MagFilter: gpu.FilterModeLinear, AddressModeU: gpu.AddressModeRepeat}))
	})
	t.Run("a filter that does not exist", func(t *testing.T) {
		h := newHarness(t)
		_, err := h.d.NewSampler(gpu.SamplerDesc{MagFilter: 9})
		wantTag(t, err, invalid, "Device.NewSampler.GPUInvalidDescriptor")
	})
	t.Run("an address mode that does not exist", func(t *testing.T) {
		h := newHarness(t)
		_, err := h.d.NewSampler(gpu.SamplerDesc{AddressModeV: 9})
		wantTag(t, err, invalid, "Device.NewSampler.GPUInvalidDescriptor")
	})
	t.Run("a query set without the feature", func(t *testing.T) {
		h := newHarness(t)
		_, err := h.d.NewQuerySet(gpu.QuerySetDesc{Count: 4})
		wantTag(t, err, invalid, "Device.NewQuerySet.GPUInvalidDescriptor")
	})
	t.Run("a query set", func(t *testing.T) {
		ts := newTimestampHarness(t)
		q := must(ts.d.NewQuerySet(gpu.QuerySetDesc{Count: 4}))
		if q.Count() != 4 {
			t.Errorf("Count = %d", q.Count())
		}
	})
	t.Run("a query set of no timestamps", func(t *testing.T) {
		ts := newTimestampHarness(t)
		_, err := ts.d.NewQuerySet(gpu.QuerySetDesc{})
		wantTag(t, err, invalid, "Device.NewQuerySet.GPUInvalidDescriptor")
	})
}

// newTimestampHarness is a harness whose device has timestamp queries.
func newTimestampHarness(t *testing.T) *harness {
	t.Helper()
	// The recorder takes the feature before the app asks for the device.
	ts := newTester(t)
	ts.GPUDevice().SupportTimestamps(1)
	return finishHarness(t, ts)
}

func TestShaderModules(t *testing.T) {
	tests := []struct {
		name   string
		shader *gpu.Shader
		tag    string
		route  string
	}{
		{"the test shader", testShader, "", ""},
		{"no shader", nil, invalid, "Device.NewShaderModule.GPUInvalidDescriptor"},
		{"no entry point", &gpu.Shader{Name: "empty"}, invalid, "Device.NewShaderModule.GPUInvalidDescriptor"},
		{"an entry point twice", &gpu.Shader{Name: "twice", Entries: []gpu.EntryPoint{{Name: "a", Stage: gpu.ShaderStageVertex}, {Name: "a", Stage: gpu.ShaderStageFragment}}},
			invalid, "Device.NewShaderModule.GPUInvalidDescriptor"},
		{"an entry point without a stage", &gpu.Shader{Name: "nostage", Entries: []gpu.EntryPoint{{Name: "a"}}}, invalid, "Device.NewShaderModule.GPUInvalidDescriptor"},
		{"a binding of no kind", &gpu.Shader{Name: "nokind", Entries: []gpu.EntryPoint{{Name: "a", Stage: gpu.ShaderStageCompute, Bindings: []gpu.ShaderBinding{{Name: "x"}}}}},
			invalid, "Device.NewShaderModule.GPUInvalidDescriptor"},
		{"a binding twice", &gpu.Shader{Name: "twice", Entries: []gpu.EntryPoint{{Name: "a", Stage: gpu.ShaderStageCompute, Bindings: []gpu.ShaderBinding{
			{Name: "x", Kind: gpu.BindingUniformBuffer}, {Name: "y", Kind: gpu.BindingUniformBuffer}}}}}, invalid, "Device.NewShaderModule.GPUInvalidDescriptor"},
		{"a storage format that does not exist", &gpu.Shader{Name: "format", Entries: []gpu.EntryPoint{{Name: "a", Stage: gpu.ShaderStageCompute, Bindings: []gpu.ShaderBinding{
			{Name: "x", Kind: gpu.BindingStorageTexture, StorageFormat: "rgb9e5"}}}}}, invalid, "Device.NewShaderModule.GPUInvalidDescriptor"},
		{"a group past the limit", &gpu.Shader{Name: "group", Entries: []gpu.EntryPoint{{Name: "a", Stage: gpu.ShaderStageCompute, Bindings: []gpu.ShaderBinding{
			{Name: "x", Group: 9, Kind: gpu.BindingUniformBuffer}}}}}, limit, "Device.NewShaderModule.GPULimitExceeded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			m, err := h.d.NewShaderModule(tt.shader)
			if tt.tag == "" {
				if err != nil || m == nil {
					t.Fatalf("NewShaderModule: %v", err)
				}
				return
			}
			wantTag(t, err, tt.tag, tt.route)
		})
	}
}

func TestRenderPipelines(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*gpu.RenderPipelineDesc)
		tag    string
		route  string
	}{
		{"the usual pipeline", nil, "", ""},
		{"with depth and four samples", func(d *gpu.RenderPipelineDesc) {
			d.DepthStencil = &gpu.DepthStencilState{Format: gpu.TextureFormatDepth32Float, DepthWriteEnabled: true, DepthCompare: gpu.CompareFunctionLess}
			d.Multisample.Count = 4
		}, "", ""},
		{"blending, lines and no culling", func(d *gpu.RenderPipelineDesc) {
			d.Fragment.Targets[0].Blend = gpu.BlendState{Enabled: true, SrcFactor: gpu.BlendFactorSrcAlpha, DstFactor: gpu.BlendFactorOneMinusSrcAlpha}
			d.Primitive = gpu.PrimitiveState{Topology: gpu.PrimitiveTopologyLineList}
		}, "", ""},
		{"no vertex module", func(d *gpu.RenderPipelineDesc) { d.Vertex.Module = nil }, invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"an entry point the shader lacks", func(d *gpu.RenderPipelineDesc) { d.Vertex.EntryPoint = "nope" }, invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"a fragment function as the vertex one", func(d *gpu.RenderPipelineDesc) { d.Vertex.EntryPoint = "fs" }, invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"a compute function as the fragment one", func(d *gpu.RenderPipelineDesc) { d.Fragment.EntryPoint = "cs" }, invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"no targets", func(d *gpu.RenderPipelineDesc) { d.Fragment.Targets = nil }, invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"a depth format as a target", func(d *gpu.RenderPipelineDesc) { d.Fragment.Targets[0].Format = gpu.TextureFormatDepth32Float },
			invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"too many targets", func(d *gpu.RenderPipelineDesc) {
			d.Fragment.Targets = make([]gpu.ColorTargetState, 9)
			for i := range d.Fragment.Targets {
				d.Fragment.Targets[i].Format = gpu.TextureFormatRGBA8Unorm
			}
		}, limit, "Device.NewRenderPipeline.GPULimitExceeded"},
		{"a vertex input with no attribute", func(d *gpu.RenderPipelineDesc) {
			d.Vertex.Buffers[0].Attributes = d.Vertex.Buffers[0].Attributes[:1]
		}, invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"an attribute of the wrong type", func(d *gpu.RenderPipelineDesc) { d.Vertex.Buffers[0].Attributes[1].Format = gpu.VertexFormatUint32x4 },
			invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"an attribute past the stride", func(d *gpu.RenderPipelineDesc) { d.Vertex.Buffers[0].ArrayStride = 16 },
			invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"a stride that is not a multiple of 4", func(d *gpu.RenderPipelineDesc) { d.Vertex.Buffers[0].ArrayStride = 26 },
			invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"two attributes at one location", func(d *gpu.RenderPipelineDesc) { d.Vertex.Buffers[0].Attributes[1].ShaderLocation = 0 },
			invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"too many vertex buffers", func(d *gpu.RenderPipelineDesc) {
			d.Vertex.Buffers = make([]gpu.VertexBufferLayout, 9)
		}, limit, "Device.NewRenderPipeline.GPULimitExceeded"},
		{"a depth format that is a color", func(d *gpu.RenderPipelineDesc) {
			d.DepthStencil = &gpu.DepthStencilState{Format: gpu.TextureFormatRGBA8Unorm}
		}, invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"a depth bias on lines", func(d *gpu.RenderPipelineDesc) {
			d.DepthStencil = &gpu.DepthStencilState{Format: gpu.TextureFormatDepth32Float, DepthBias: 1}
			d.Primitive.Topology = gpu.PrimitiveTopologyLineList
		}, invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
		{"three samples", func(d *gpu.RenderPipelineDesc) { d.Multisample.Count = 3 }, invalid, "Device.NewRenderPipeline.GPUInvalidDescriptor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			m := h.module()
			p, err := h.d.NewRenderPipeline(h.pipelineDesc(m, tt.mutate))
			if tt.tag == "" {
				if err != nil || p == nil {
					t.Fatalf("NewRenderPipeline: %v", err)
				}
				return
			}
			wantTag(t, err, tt.tag, tt.route)
			if got := h.d.Resources().RenderPipelines; got != 0 {
				t.Errorf("the failed call left %d pipelines", got)
			}
		})
	}
}

func TestComputePipelines(t *testing.T) {
	tests := []struct {
		name  string
		desc  func(m *gpu.ShaderModule) gpu.ComputePipelineDesc
		tag   string
		route string
	}{
		{"the compute function", func(m *gpu.ShaderModule) gpu.ComputePipelineDesc {
			return gpu.ComputePipelineDesc{Module: m, EntryPoint: "cs"}
		}, "", ""},
		{"a vertex function", func(m *gpu.ShaderModule) gpu.ComputePipelineDesc {
			return gpu.ComputePipelineDesc{Module: m, EntryPoint: "vs"}
		},
			invalid, "Device.NewComputePipeline.GPUInvalidDescriptor"},
		{"no module", func(m *gpu.ShaderModule) gpu.ComputePipelineDesc { return gpu.ComputePipelineDesc{EntryPoint: "cs"} },
			invalid, "Device.NewComputePipeline.GPUInvalidDescriptor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			p, err := h.d.NewComputePipeline(tt.desc(h.module()))
			if tt.tag == "" {
				if err != nil {
					t.Fatalf("NewComputePipeline: %v", err)
				}
				if p.WorkgroupSize() != [3]int{8, 8, 1} {
					t.Errorf("WorkgroupSize = %v", p.WorkgroupSize())
				}
				return
			}
			wantTag(t, err, tt.tag, tt.route)
		})
	}
	t.Run("a workgroup over the limit", func(t *testing.T) {
		h := newHarness(t)
		m := must(h.d.NewShaderModule(&gpu.Shader{Name: "wide", Entries: []gpu.EntryPoint{{Name: "cs", Stage: gpu.ShaderStageCompute, WorkgroupSize: [3]int{512, 1, 1}}}}))
		_, err := h.d.NewComputePipeline(gpu.ComputePipelineDesc{Module: m, EntryPoint: "cs"})
		wantTag(t, err, limit, "Device.NewComputePipeline.GPULimitExceeded")
	})
	t.Run("a workgroup with no size", func(t *testing.T) {
		h := newHarness(t)
		m := must(h.d.NewShaderModule(&gpu.Shader{Name: "flat", Entries: []gpu.EntryPoint{{Name: "cs", Stage: gpu.ShaderStageCompute}}}))
		_, err := h.d.NewComputePipeline(gpu.ComputePipelineDesc{Module: m, EntryPoint: "cs"})
		wantTag(t, err, invalid, "Device.NewComputePipeline.GPUInvalidDescriptor")
	})
}

func TestBindGroups(t *testing.T) {
	type setup struct {
		h       *harness
		pipe    *gpu.RenderPipeline
		uniform *gpu.Buffer
		storage *gpu.Buffer
		view    *gpu.TextureView
		cube    *gpu.TextureView
		sampler *gpu.Sampler
		layout0 *gpu.BindGroupLayout
		layout1 *gpu.BindGroupLayout
	}
	build := func(t *testing.T) setup {
		h := newHarness(t)
		m := h.module()
		s := setup{h: h}
		s.pipe = must(h.d.NewRenderPipeline(h.pipelineDesc(m, nil)))
		s.uniform = must(h.d.NewBuffer(gpu.BufferDesc{Size: 1024, Usage: gpu.BufferUsageUniform}))
		s.storage = must(h.d.NewBuffer(gpu.BufferDesc{Size: 1024, Usage: gpu.BufferUsageStorage}))
		s.view = must(h.d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding})).View()
		s.cube = must(h.d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 8, Height: 8}, Kind: gpu.TextureKindCube, Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding})).View()
		s.sampler = must(h.d.NewSampler(gpu.SamplerDesc{}))
		s.layout0, s.layout1 = s.pipe.BindGroupLayout(0), s.pipe.BindGroupLayout(1)
		return s
	}
	tests := []struct {
		name  string
		desc  func(s setup) gpu.BindGroupDesc
		tag   string
		route string
	}{
		{"the uniform group", func(s setup) gpu.BindGroupDesc {
			return gpu.BindGroupDesc{Layout: s.layout0, Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: s.uniform, Offset: 256, Size: 64}}}
		}, "", ""},
		{"the texture group", func(s setup) gpu.BindGroupDesc {
			return gpu.BindGroupDesc{Layout: s.layout1, Entries: []gpu.BindGroupEntry{{Binding: 0, TextureView: s.view}, {Binding: 1, Sampler: s.sampler}}}
		}, "", ""},
		{"no layout", func(s setup) gpu.BindGroupDesc { return gpu.BindGroupDesc{} }, invalid, "Device.NewBindGroup.GPUInvalidDescriptor"},
		{"an entry missing", func(s setup) gpu.BindGroupDesc {
			return gpu.BindGroupDesc{Layout: s.layout1, Entries: []gpu.BindGroupEntry{{Binding: 0, TextureView: s.view}}}
		}, invalid, "Device.NewBindGroup.GPUInvalidDescriptor"},
		{"an entry for a binding the layout lacks", func(s setup) gpu.BindGroupDesc {
			return gpu.BindGroupDesc{Layout: s.layout1, Entries: []gpu.BindGroupEntry{{Binding: 0, TextureView: s.view}, {Binding: 7, Sampler: s.sampler}}}
		}, invalid, "Device.NewBindGroup.GPUInvalidDescriptor"},
		{"a binding twice", func(s setup) gpu.BindGroupDesc {
			return gpu.BindGroupDesc{Layout: s.layout1, Entries: []gpu.BindGroupEntry{{Binding: 0, TextureView: s.view}, {Binding: 0, TextureView: s.view}}}
		}, invalid, "Device.NewBindGroup.GPUInvalidDescriptor"},
		{"a sampler where a texture goes", func(s setup) gpu.BindGroupDesc {
			return gpu.BindGroupDesc{Layout: s.layout1, Entries: []gpu.BindGroupEntry{{Binding: 0, Sampler: s.sampler}, {Binding: 1, Sampler: s.sampler}}}
		}, invalid, "Device.NewBindGroup.GPUInvalidDescriptor"},
		{"a cube where a 2D texture goes", func(s setup) gpu.BindGroupDesc {
			return gpu.BindGroupDesc{Layout: s.layout1, Entries: []gpu.BindGroupEntry{{Binding: 0, TextureView: s.cube}, {Binding: 1, Sampler: s.sampler}}}
		}, invalid, "Device.NewBindGroup.GPUInvalidDescriptor"},
		{"a storage buffer where a uniform one goes", func(s setup) gpu.BindGroupDesc {
			return gpu.BindGroupDesc{Layout: s.layout0, Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: s.storage, Size: 64}}}
		}, invalid, "Device.NewBindGroup.GPUInvalidDescriptor"},
		{"a range smaller than the shader reads", func(s setup) gpu.BindGroupDesc {
			return gpu.BindGroupDesc{Layout: s.layout0, Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: s.uniform, Size: 32}}}
		}, invalid, "Device.NewBindGroup.GPUInvalidDescriptor"},
		{"a range past the buffer", func(s setup) gpu.BindGroupDesc {
			return gpu.BindGroupDesc{Layout: s.layout0, Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: s.uniform, Offset: 1024, Size: 64}}}
		}, invalid, "Device.NewBindGroup.GPUInvalidDescriptor"},
		{"an offset that is not aligned", func(s setup) gpu.BindGroupDesc {
			return gpu.BindGroupDesc{Layout: s.layout0, Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: s.uniform, Offset: 16, Size: 64}}}
		}, limit, "Device.NewBindGroup.GPULimitExceeded"},
		{"a range over the limit", func(s setup) gpu.BindGroupDesc {
			big := must(s.h.d.NewBuffer(gpu.BufferDesc{Size: 1 << 20, Usage: gpu.BufferUsageUniform}))
			return gpu.BindGroupDesc{Layout: s.layout0, Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: big}}}
		}, limit, "Device.NewBindGroup.GPULimitExceeded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := build(t)
			g, err := s.h.d.NewBindGroup(tt.desc(s))
			if tt.tag == "" {
				if err != nil || g == nil {
					t.Fatalf("NewBindGroup: %v", err)
				}
				if g.Layout() == nil {
					t.Errorf("no layout")
				}
				return
			}
			wantTag(t, err, tt.tag, tt.route)
			if s.h.d.Resources().BindGroups != 0 {
				t.Errorf("the failed call left a bind group")
			}
		})
	}
}
