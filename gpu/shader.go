package gpu

// ShaderStage is the stage of a shader entry point.
//
// Flutter: no counterpart
type ShaderStage int

// The stages of an entry point.
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

// String returns "vertex", "fragment", "compute" or "unspecified".
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

// The kinds of binding a shader may have (SPEC §27.3).
const (
	// BindingUnspecified is the zero value: no binding.
	//
	// Flutter: no counterpart
	BindingUnspecified BindingKind = iota
	// BindingUniformBuffer is a `var<uniform>`.
	//
	// Flutter: no counterpart
	BindingUniformBuffer
	// BindingSampledTexture is a `texture_2d<f32>`, or a texture_2d_array, texture_cube or texture_3d of f32
	// or u32 (ShaderBinding.Dimension and Sample).
	//
	// Flutter: no counterpart
	BindingSampledTexture
	// BindingSampler is a `sampler`.
	//
	// Flutter: no counterpart
	BindingSampler
	// BindingStorageBufferReadOnly is a `var<storage, read>`.
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

// String returns "uniform", "texture", "sampler", "storage-read", "storage", "storage-texture" or
// "unspecified".
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

// TextureDimension is the dimension of a texture binding of a shader. The zero value is TextureDim2D.
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

// SampleType is how a shader reads the texels of a texture (WebGPU's GPUTextureSampleType). The zero value
// is SampleTypeFloat.
//
// Flutter: no counterpart
type SampleType int

// The sample types of a texture binding.
const (
	// SampleTypeFloat is a floating point texture that a filtering sampler can read.
	//
	// Flutter: no counterpart
	SampleTypeFloat SampleType = iota
	// SampleTypeUnfilterableFloat is a floating point texture that only textureLoad and the like read:
	// any float texture can be bound to it, a float32 texture too.
	//
	// Flutter: no counterpart
	SampleTypeUnfilterableFloat
	// SampleTypeUint is an unsigned integer texture.
	//
	// Flutter: no counterpart
	SampleTypeUint
)

// ShaderType is the type of an input of a vertex entry point: a scalar or a vector of 32 bit values.
//
// Flutter: no counterpart
type ShaderType int

// The types of the inputs of a vertex entry point.
const (
	// ShaderTypeUnspecified is the zero value: no type.
	//
	// Flutter: no counterpart
	ShaderTypeUnspecified ShaderType = iota
	// ShaderTypeFloat32 is an f32.
	//
	// Flutter: no counterpart
	ShaderTypeFloat32
	// ShaderTypeFloat32x2 is a vec2<f32>.
	//
	// Flutter: no counterpart
	ShaderTypeFloat32x2
	// ShaderTypeFloat32x3 is a vec3<f32>.
	//
	// Flutter: no counterpart
	ShaderTypeFloat32x3
	// ShaderTypeFloat32x4 is a vec4<f32>.
	//
	// Flutter: no counterpart
	ShaderTypeFloat32x4
	// ShaderTypeSint32 is an i32.
	//
	// Flutter: no counterpart
	ShaderTypeSint32
	// ShaderTypeSint32x2 is a vec2<i32>.
	//
	// Flutter: no counterpart
	ShaderTypeSint32x2
	// ShaderTypeSint32x3 is a vec3<i32>.
	//
	// Flutter: no counterpart
	ShaderTypeSint32x3
	// ShaderTypeSint32x4 is a vec4<i32>.
	//
	// Flutter: no counterpart
	ShaderTypeSint32x4
	// ShaderTypeUint32 is a u32.
	//
	// Flutter: no counterpart
	ShaderTypeUint32
	// ShaderTypeUint32x2 is a vec2<u32>.
	//
	// Flutter: no counterpart
	ShaderTypeUint32x2
	// ShaderTypeUint32x3 is a vec3<u32>.
	//
	// Flutter: no counterpart
	ShaderTypeUint32x3
	// ShaderTypeUint32x4 is a vec4<u32>.
	//
	// Flutter: no counterpart
	ShaderTypeUint32x4
)

// ShaderRegister is where the HLSL of a shader binds a resource (Direct3D 12 only; SPEC §9.8, §27.3): for
// the binding B of the group G, `register(<Class>B, spaceG)`.
//
// Flutter: no counterpart
type ShaderRegister struct {
	// Class is the register class: "b" for a uniform buffer, "t" for a texture or a read-only storage
	// buffer, "s" for a sampler and "u" for a writable storage buffer or a storage texture.
	Class string
	// Space is the register space, the group of the binding.
	Space int
	// Index is the register number, the binding number.
	Index int
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
	// Sample is the sample type of a sampled texture binding.
	Sample SampleType
	// StorageFormat is the texel format of a storage texture binding, as WGSL names it ("rgba8unorm",
	// "rgba16float", "r32float" or "r32uint").
	StorageFormat string
	// Size is the size in bytes of the type of a uniform or storage buffer binding, by the layout rules of
	// WGSL. For a storage buffer that ends in an array without a length it counts one element of the
	// array, which is the smallest buffer range that WebGPU accepts for the binding.
	Size int
	// Register is where the HLSL binds the resource.
	Register ShaderRegister
}

// VertexInput is an `@location(n)` input of a vertex entry point.
//
// Flutter: no counterpart
type VertexInput struct {
	// Name is the name of the parameter or of the member of the struct.
	Name string
	// Location is the number in @location.
	Location int
	// Type is the type the shader reads the input as.
	Type ShaderType
}

// EntryPoint is a function of a shader that a pipeline stage runs.
//
// Flutter: no counterpart
type EntryPoint struct {
	// Name is the name of the function.
	Name string
	// Stage is the stage the function runs in.
	Stage ShaderStage
	// Bindings are the resources the function reads, directly or through the functions it calls, sorted by
	// group and binding.
	Bindings []ShaderBinding
	// Inputs are the @location inputs of a vertex entry point, in the order of the parameters.
	Inputs []VertexInput
	// WorkgroupSize is the @workgroup_size of a compute entry point (the missing dimensions are 1).
	WorkgroupSize [3]int
	// ReadsInstanceIndex and ReadsVertexIndex say that a vertex entry point reads
	// @builtin(instance_index) or @builtin(vertex_index). Such a shader is drawn with a first instance, a
	// first vertex and a base vertex of 0 only: Direct3D 12's SV_InstanceID and SV_VertexID leave out the
	// first instance and the first vertex of a draw (SPEC §27.1).
	ReadsInstanceIndex, ReadsVertexIndex bool
}

// Shader is one WGSL module with its SPIR-V, its HLSL and its reflection. glutter-gen gpushaders makes the
// values of this type in shaders_gen.go (D31, SPEC §27.3): WebGPU takes the WGSL, Vulkan the SPIR-V and
// Direct3D 12 the HLSL. An app does not make one by hand.
//
// Flutter: no counterpart
type Shader struct {
	// Name is the name of the .wgsl file without its extension.
	Name string
	// WGSL is the source text, with the `//#include` lines expanded.
	WGSL string
	// SPIRV is what naga made from the WGSL: one module with every entry point.
	SPIRV []byte
	// HLSL is what naga made from the WGSL for shader model 5.1, with every binding at the register of
	// ShaderBinding.Register: one file with every entry point.
	HLSL string
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
