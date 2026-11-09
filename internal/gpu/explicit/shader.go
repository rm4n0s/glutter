package explicit

import "strconv"

// ShaderStage is the stage of a shader entry point.
//
// Flutter: no counterpart
type ShaderStage int

// The stages of the shaders of the explicit renderer. The 2D renderer's shaders have no compute entry
// point; those of glutter-gen shaders -gpu may (SPEC §27.1).
const (
	// ShaderStageUnspecified is the zero value: no stage.
	//
	// Flutter: no counterpart
	ShaderStageUnspecified ShaderStage = iota
	// ShaderStageVertex is an entry point marked @vertex.
	//
	// Flutter: no counterpart
	ShaderStageVertex
	// ShaderStageFragment is an entry point marked @fragment.
	//
	// Flutter: no counterpart
	ShaderStageFragment
	// ShaderStageCompute is an entry point marked @compute.
	//
	// Flutter: no counterpart
	ShaderStageCompute
)

// String returns "vertex" or "fragment".
func (s ShaderStage) String() string {
	switch s {
	case ShaderStageVertex:
		return "vertex"
	case ShaderStageFragment:
		return "fragment"
	case ShaderStageCompute:
		return "compute"
	}
	return "unspecified"
}

// BindingKind is what a shader binding is.
//
// Flutter: no counterpart
type BindingKind int

// The kinds of binding the design limits allow (SPEC §9.3).
const (
	// BindingUnspecified is the zero value: no binding.
	//
	// Flutter: no counterpart
	BindingUnspecified BindingKind = iota
	// BindingUniformBuffer is a `var<uniform>`. It is always bound with a dynamic
	// offset: the uniforms of a frame go into a ring buffer (SPEC §9.3).
	//
	// Flutter: no counterpart
	BindingUniformBuffer
	// BindingSampledTexture is a `texture_2d<f32>`, or a texture_2d_array, texture_cube or
	// texture_3d of f32 or u32 (ShaderBinding.Dimension and Sample).
	//
	// Flutter: no counterpart
	BindingSampledTexture
	// BindingSampler is a `sampler`.
	//
	// Flutter: no counterpart
	BindingSampler
	// BindingStorageBufferReadOnly is a `var<storage, read>`, bound at an offset that is a multiple
	// of Limits.StorageOffsetAlignment. Any stage may read it.
	//
	// Flutter: no counterpart
	BindingStorageBufferReadOnly
	// BindingStorageBuffer is a `var<storage, read_write>`: a compute entry point only.
	//
	// Flutter: no counterpart
	BindingStorageBuffer
	// BindingStorageTexture is a `texture_storage_2d<format, write>` (or _2d_array or _3d, with
	// ShaderBinding.StorageFormat): a compute entry point only.
	//
	// Flutter: no counterpart
	BindingStorageTexture
)

// String returns "uniform", "texture", "sampler", "storage-read", "storage" or "storage-texture".
func (k BindingKind) String() string {
	switch k {
	case BindingUniformBuffer:
		return "uniform"
	case BindingSampledTexture:
		return "texture"
	case BindingSampler:
		return "sampler"
	case BindingStorageBufferReadOnly:
		return "storage-read"
	case BindingStorageBuffer:
		return "storage"
	case BindingStorageTexture:
		return "storage-texture"
	}
	return "unspecified"
}

// TextureDimension is the dimension of a texture binding of a shader: the WGSL type
// texture_2d, texture_2d_array, texture_cube or texture_3d. The zero value is TextureDim2D.
//
// Flutter: no counterpart
type TextureDimension int

// The dimensions of a texture binding.
const (
	// TextureDim2D is a texture_2d.
	//
	// Flutter: no counterpart
	TextureDim2D TextureDimension = iota
	// TextureDim2DArray is a texture_2d_array.
	//
	// Flutter: no counterpart
	TextureDim2DArray
	// TextureDimCube is a texture_cube.
	//
	// Flutter: no counterpart
	TextureDimCube
	// TextureDim3D is a texture_3d.
	//
	// Flutter: no counterpart
	TextureDim3D
)

// Dimension returns the dimension of the views of a texture of this kind that a shader samples.
//
// Flutter: no counterpart
func (k TextureKind) Dimension() TextureDimension {
	switch k {
	case TextureKind2DArray:
		return TextureDim2DArray
	case TextureKindCube:
		return TextureDimCube
	case TextureKind3D:
		return TextureDim3D
	}
	return TextureDim2D
}

// ShaderRegister is where the HLSL of a shader binds a resource (D25, SPEC §9.8): for the binding
// B of the group G, `register(<Class>B, spaceG)`. The generator makes every register of
// shaders_gen.go like that and checks the HLSL against it.
//
// Flutter: no counterpart
type ShaderRegister struct {
	// Class is the register class: "b" for a constant buffer (a uniform), "t" for a texture or a
	// read-only storage buffer, "s" for a sampler and "u" for a writable storage buffer or a storage
	// texture.
	Class string
	// Space is the register space, the group of the binding.
	Space int
	// Index is the register number, the binding number.
	Index int
}

// String returns the register as HLSL writes it, such as "t3, space1".
func (r ShaderRegister) String() string {
	return r.Class + strconv.Itoa(r.Index) + ", space" + strconv.Itoa(r.Space)
}

// ShaderBinding is one resource a shader reads, from its declaration in the WGSL:
// `@group(g) @binding(b) var ...`.
//
// Flutter: no counterpart
type ShaderBinding struct {
	// Name is the name of the variable in the WGSL.
	Name string
	// Group and Binding are the numbers of the declaration.
	Group, Binding int
	// Kind says what the resource is.
	Kind BindingKind
	// Dimension is the dimension of a texture binding; the zero value is a 2D texture.
	Dimension TextureDimension
	// Sample is the sample type of a texture binding. A f32 texture that the shader reads only with
	// textureLoad, textureDimensions and the like is SampleTypeUnfilterableFloat: any float
	// texture can be bound to it, a float32 texture without float32-filterable too. A u32 texture
	// is SampleTypeUint.
	Sample SampleType
	// StorageFormat is the texel format of a storage texture binding.
	StorageFormat TextureFormat
	// Register is where the HLSL binds the resource (Direct3D 12 only).
	Register ShaderRegister
	// MetalIndex is where the MSL binds the resource (Metal only): the index in the argument table of its
	// class, buffer for uniform and storage buffers, texture for sampled and storage textures, sampler for
	// samplers. It is the place of the binding among the bindings of the module of its class, in the order
	// of group and binding (SPEC §29.5).
	MetalIndex int
}

// MetalVertexBufferBase is the index of the first vertex buffer in Metal's buffer argument table: the
// vertex buffer of slot s is bound at MetalVertexBufferBase+s, above the indexes of the buffers that a
// shader binds (a shader has at most MetalVertexBufferBase of them; SPEC §29.5).
//
// Flutter: no counterpart
const MetalVertexBufferBase = 16

// EntryPoint is a function of a shader that a pipeline stage runs.
//
// Flutter: no counterpart
type EntryPoint struct {
	// Name is the name of the function.
	Name string
	// Stage is the stage the function runs in.
	Stage ShaderStage
	// Bindings are the resources the function reads, directly or through the
	// functions it calls, sorted by group and binding.
	Bindings []ShaderBinding
	// WorkgroupSize is the @workgroup_size of a compute entry point (the missing dimensions are 1).
	WorkgroupSize [3]int
	// ReadsInstanceIndex and ReadsVertexIndex say that a vertex entry point reads
	// @builtin(instance_index) or @builtin(vertex_index): Direct3D 12's SV_InstanceID and
	// SV_VertexID leave out the first instance and the first vertex of a draw, so such a shader is
	// drawn with a first instance, a first vertex and a base vertex of 0 only (SPEC §27.1).
	ReadsInstanceIndex, ReadsVertexIndex bool
}

// Shader is one WGSL module with its SPIR-V. glutter-gen shaders makes the values
// of this type in shaders_gen.go (D20): WebGPU takes WGSL, Vulkan takes SPIR-V.
//
// Flutter: no counterpart
type Shader struct {
	// Name is the name of the .wgsl file without its extension.
	Name string
	// WGSL is the source text.
	WGSL string
	// SPIRV is what naga made from WGSL: one module with every entry point.
	SPIRV []byte
	// HLSL is what naga made from WGSL for shader model 5.1, after the generator put every
	// binding at `register(<class>B, spaceG)` (ShaderBinding.Register): one file with every entry
	// point, under the names they have in the WGSL. Direct3D 12 compiles it with D3DCompile.
	HLSL string
	// MSL is the Metal Shading Language 2.4 that naga writes for the module, with the resources bound at
	// the indexes of ShaderBinding.MetalIndex (Metal only).
	MSL string
	// Entries are the entry points of the module.
	Entries []EntryPoint
}

// Entry returns the entry point with the name; ok is false when there is none.
//
// Flutter: no counterpart
func (s *Shader) Entry(name string) (e EntryPoint, ok bool) {
	for _, e := range s.Entries {
		if e.Name == name {
			return e, true
		}
	}
	return EntryPoint{}, false
}

// Shaders returns every generated shader, in the order of their names: the modules a backend that
// checks them at startup (WebGPU) compiles.
//
// Flutter: no counterpart
func Shaders() []*Shader {
	return []*Shader{ShaderBlit, ShaderFilter, ShaderImage, ShaderLayer, ShaderMask, ShaderShape, ShaderSolid, ShaderVertices}
}
