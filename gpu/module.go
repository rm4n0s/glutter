package gpu

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// storageFormats maps the spelling of a storage texture format in WGSL to the format.
var storageFormats = map[string]explicit.TextureFormat{
	"rgba8unorm":  explicit.TextureFormatRGBA8Unorm,
	"rgba16float": explicit.TextureFormatRGBA16Float,
	"r32float":    explicit.TextureFormatR32Float,
	"r32uint":     explicit.TextureFormatR32Uint,
}

// ShaderModule is a shader that glutter-gen gpushaders made, ready for the pipelines of a device (WebGPU's
// GPUShaderModule). The backend compiles it when a pipeline that uses one of its entry points is made.
//
// Flutter: no counterpart
type ShaderModule struct {
	resource
	shader *Shader
	exp    *explicit.Shader
}

// toExplicit makes the explicit form of a shader. The names of the storage formats are WGSL's here; a name that
// is not one of the four fails with ok false.
func toExplicit(s *Shader) (exp *explicit.Shader, bad string) {
	exp = &explicit.Shader{Name: s.Name, WGSL: s.WGSL, SPIRV: s.SPIRV, HLSL: s.HLSL, Entries: make([]explicit.EntryPoint, len(s.Entries))}
	for i, e := range s.Entries {
		ee := explicit.EntryPoint{Name: e.Name, Stage: explicit.ShaderStage(e.Stage), WorkgroupSize: e.WorkgroupSize,
			ReadsInstanceIndex: e.ReadsInstanceIndex, ReadsVertexIndex: e.ReadsVertexIndex, Bindings: make([]explicit.ShaderBinding, len(e.Bindings))}
		for j, b := range e.Bindings {
			sb := explicit.ShaderBinding{Name: b.Name, Group: b.Group, Binding: b.Binding, Kind: explicit.BindingKind(b.Kind),
				Dimension: explicit.TextureDimension(b.Dimension), Sample: explicit.SampleType(b.Sample),
				Register: explicit.ShaderRegister{Class: b.Register.Class, Space: b.Register.Space, Index: b.Register.Index}}
			if b.Kind == BindingStorageTexture {
				f, ok := storageFormats[b.StorageFormat]
				if !ok {
					return nil, b.StorageFormat
				}
				sb.StorageFormat = f
			}
			ee.Bindings[j] = sb
		}
		exp.Entries[i] = ee
	}
	return exp, ""
}

// NewShaderModule makes a module from a shader that glutter-gen gpushaders made (WebGPU's
// GPUDevice.createShaderModule). It fails with GPUInvalidDescriptor for a shader that is not whole (nil, no entry
// point, an entry point twice, a binding of a kind or a storage format that does not exist, a binding twice in one
// group) and with GPULimitExceeded for an entry point that has more resources than a stage may. The backend
// compiles the shader later, at the first pipeline: a failure there is GPUShaderCompileFailed.
//
// Flutter: no counterpart
func (d *Device) NewShaderModule(s *Shader) (*ShaderModule, error) {
	d.assertThread("NewShaderModule")
	if d.lostErr != nil {
		return nil, d.lostErr
	}
	if s == nil || len(s.Entries) == 0 {
		return nil, errors.New(TagGPUInvalidDescriptor, "a shader module is made from a shader with at least one entry point", "field", "Shader")
	}
	seen := map[string]bool{}
	for _, e := range s.Entries {
		if e.Stage < ShaderStageVertex || e.Stage > ShaderStageCompute || e.Name == "" || seen[e.Name] {
			return nil, errors.New(TagGPUInvalidDescriptor, "an entry point has a name that is not used twice and a stage", "field", "Entries", "entry", e.Name, "stage", e.Stage.String(), "shader", s.Name)
		}
		seen[e.Name] = true
		type slot struct{ group, binding int }
		used := map[slot]bool{}
		var textures, samplers, uniforms, storageBuffers, storageTextures int
		for _, b := range e.Bindings {
			if b.Kind < BindingUniformBuffer || b.Kind > BindingStorageTexture || b.Group < 0 || b.Binding < 0 || used[slot{b.Group, b.Binding}] {
				return nil, errors.New(TagGPUInvalidDescriptor, "a binding has a kind that exists and a place that no other binding of the entry point has",
					"field", "Bindings", "entry", e.Name, "binding", b.Name, "shader", s.Name)
			}
			used[slot{b.Group, b.Binding}] = true
			if b.Group >= d.limits.MaxBindGroups {
				return nil, errors.New(TagGPULimitExceeded, "a binding is in a bind group past Limits.MaxBindGroups", "entry", e.Name, "binding", b.Name, "group", b.Group, "max", d.limits.MaxBindGroups)
			}
			if b.Kind == BindingStorageTexture {
				if _, ok := storageFormats[b.StorageFormat]; !ok {
					return nil, errors.New(TagGPUInvalidDescriptor, "a storage texture binding has one of the storage formats", "field", "StorageFormat",
						"entry", e.Name, "binding", b.Name, "value", b.StorageFormat)
				}
			}
			switch b.Kind {
			case BindingSampledTexture:
				textures++
			case BindingSampler:
				samplers++
			case BindingUniformBuffer:
				uniforms++
			case BindingStorageBufferReadOnly, BindingStorageBuffer:
				storageBuffers++
			case BindingStorageTexture:
				storageTextures++
			}
		}
		l := d.limits
		for _, c := range []struct {
			what       string
			n, allowed int
		}{
			{"sampled textures per stage", textures, l.MaxSampledTexturesPerStage}, {"samplers per stage", samplers, l.MaxSamplersPerStage},
			{"uniform buffers per stage", uniforms, l.MaxUniformBuffersPerStage}, {"storage buffers per stage", storageBuffers, l.MaxStorageBuffersPerStage},
			{"storage textures per stage", storageTextures, l.MaxStorageTexturesPerStage},
		} {
			if c.n > c.allowed {
				return nil, errors.New(TagGPULimitExceeded, "an entry point has more resources of a kind than a stage may", "what", c.what, "entry", e.Name, "value", c.n, "max", c.allowed)
			}
		}
	}
	exp, ok := d.modules[s]
	if !ok {
		var bad string
		exp, bad = toExplicit(s)
		if exp == nil {
			return nil, errors.New(TagGPUInvalidDescriptor, "a storage texture binding has one of the storage formats", "field", "StorageFormat", "value", bad)
		}
		d.modules[s] = exp
	}
	return &ShaderModule{resource: d.track(kindShaderModule, s.Name, 1), shader: s, exp: exp}, nil
}

// Release forgets the module. The pipelines made from it keep working. It may be called twice.
//
// Flutter: no counterpart
func (m *ShaderModule) Release() {
	m.d.assertThread("ShaderModule.Release")
	m.untrack()
}
