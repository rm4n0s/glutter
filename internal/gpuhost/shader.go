package gpuhost

import (
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

var storageNames = map[explicit.TextureFormat]string{
	explicit.TextureFormatRGBA8Unorm: "rgba8unorm", explicit.TextureFormatRGBA16Float: "rgba16float",
	explicit.TextureFormatR32Float: "r32float", explicit.TextureFormatR32Uint: "r32uint",
}

// ShaderOf makes a gpu.Shader of a shader that glutter-gen made for the explicit device, such as the shaders of the
// device tests, so that the tests of package gpu on a real GPU can use them. The vertex inputs are not in the explicit
// form: the entry points of a test have none.
//
// Flutter: no counterpart
func ShaderOf(s *explicit.Shader) *gpu.Shader {
	out := &gpu.Shader{Name: s.Name, WGSL: s.WGSL, SPIRV: s.SPIRV, HLSL: s.HLSL}
	for _, e := range s.Entries {
		ge := gpu.EntryPoint{Name: e.Name, Stage: gpu.ShaderStage(e.Stage), WorkgroupSize: e.WorkgroupSize,
			ReadsInstanceIndex: e.ReadsInstanceIndex, ReadsVertexIndex: e.ReadsVertexIndex}
		for _, b := range e.Bindings {
			gb := gpu.ShaderBinding{Name: b.Name, Group: b.Group, Binding: b.Binding, Kind: gpu.BindingKind(b.Kind), Dimension: gpu.TextureDimension(b.Dimension),
				Sample: gpu.SampleType(b.Sample), Register: gpu.ShaderRegister{Class: b.Register.Class, Space: b.Register.Space, Index: b.Register.Index}}
			if b.Kind == explicit.BindingStorageTexture {
				gb.StorageFormat = storageNames[b.StorageFormat]
			}
			ge.Bindings = append(ge.Bindings, gb)
		}
		out.Entries = append(out.Entries, ge)
	}
	return out
}
