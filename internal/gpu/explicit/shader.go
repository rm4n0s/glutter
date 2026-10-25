package explicit

import "strconv"

// ShaderStage is the stage of a shader entry point.
//
// Flutter: no counterpart
type ShaderStage int

// The stages of the shaders of the explicit renderer. There is no compute in 1.1.
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
)

// String returns "vertex" or "fragment".
func (s ShaderStage) String() string {
	switch s {
	case ShaderStageVertex:
		return "vertex"
	case ShaderStageFragment:
		return "fragment"
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
	// BindingSampledTexture is a `texture_2d<f32>`.
	//
	// Flutter: no counterpart
	BindingSampledTexture
	// BindingSampler is a `sampler`.
	//
	// Flutter: no counterpart
	BindingSampler
)

// String returns "uniform", "texture" or "sampler".
func (k BindingKind) String() string {
	switch k {
	case BindingUniformBuffer:
		return "uniform"
	case BindingSampledTexture:
		return "texture"
	case BindingSampler:
		return "sampler"
	}
	return "unspecified"
}

// ShaderRegister is where the HLSL of a shader binds a resource (D25, SPEC §9.8): for the binding
// B of the group G, `register(<Class>B, spaceG)`. The generator makes every register of
// shaders_gen.go like that and checks the HLSL against it.
//
// Flutter: no counterpart
type ShaderRegister struct {
	// Class is the register class: "b" for a constant buffer (a uniform), "t" for a texture and "s"
	// for a sampler.
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
	// Register is where the HLSL binds the resource (Direct3D 12 only).
	Register ShaderRegister
}

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
