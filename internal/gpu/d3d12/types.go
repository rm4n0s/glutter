package d3d12

// The Go mirrors of the structures of d3d12shader.h that the shader reflection returns, the positions
// of the methods of the interfaces the backend calls, and the constants of d3dcommon.h it reads. The
// layout test (layout_test.go) checks each against the mingw-w64 headers.

// ShaderDesc is the Go mirror of D3D12_SHADER_DESC (ID3D12ShaderReflection::GetDesc).
//
// Flutter: no counterpart
type ShaderDesc struct {
	Version                   uint32  `c:"Version"`
	Creator                   uintptr `c:"Creator"`
	Flags                     uint32  `c:"Flags"`
	ConstantBuffers           uint32  `c:"ConstantBuffers"`
	BoundResources            uint32  `c:"BoundResources"`
	InputParameters           uint32  `c:"InputParameters"`
	OutputParameters          uint32  `c:"OutputParameters"`
	InstructionCount          uint32  `c:"InstructionCount"`
	TempRegisterCount         uint32  `c:"TempRegisterCount"`
	TempArrayCount            uint32  `c:"TempArrayCount"`
	DefCount                  uint32  `c:"DefCount"`
	DclCount                  uint32  `c:"DclCount"`
	TextureNormalInstructions uint32  `c:"TextureNormalInstructions"`
	TextureLoadInstructions   uint32  `c:"TextureLoadInstructions"`
	TextureCompInstructions   uint32  `c:"TextureCompInstructions"`
	TextureBiasInstructions   uint32  `c:"TextureBiasInstructions"`
	TextureGradientInstrs     uint32  `c:"TextureGradientInstructions"`
	FloatInstructionCount     uint32  `c:"FloatInstructionCount"`
	IntInstructionCount       uint32  `c:"IntInstructionCount"`
	UintInstructionCount      uint32  `c:"UintInstructionCount"`
	StaticFlowControlCount    uint32  `c:"StaticFlowControlCount"`
	DynamicFlowControlCount   uint32  `c:"DynamicFlowControlCount"`
	MacroInstructionCount     uint32  `c:"MacroInstructionCount"`
	ArrayInstructionCount     uint32  `c:"ArrayInstructionCount"`
	CutInstructionCount       uint32  `c:"CutInstructionCount"`
	EmitInstructionCount      uint32  `c:"EmitInstructionCount"`
	GSOutputTopology          uint32  `c:"GSOutputTopology"`
	GSMaxOutputVertexCount    uint32  `c:"GSMaxOutputVertexCount"`
	InputPrimitive            uint32  `c:"InputPrimitive"`
	PatchConstantParameters   uint32  `c:"PatchConstantParameters"`
	GSInstanceCount           uint32  `c:"cGSInstanceCount"`
	ControlPoints             uint32  `c:"cControlPoints"`
	HSOutputPrimitive         uint32  `c:"HSOutputPrimitive"`
	HSPartitioning            uint32  `c:"HSPartitioning"`
	TessellatorDomain         uint32  `c:"TessellatorDomain"`
	BarrierInstructions       uint32  `c:"cBarrierInstructions"`
	InterlockedInstructions   uint32  `c:"cInterlockedInstructions"`
	TextureStoreInstructions  uint32  `c:"cTextureStoreInstructions"`
}

// ShaderInputBindDesc is the Go mirror of D3D12_SHADER_INPUT_BIND_DESC
// (ID3D12ShaderReflection::GetResourceBindingDesc): one resource that the shader binds.
//
// Flutter: no counterpart
type ShaderInputBindDesc struct {
	Name       uintptr `c:"Name"`
	Type       uint32  `c:"Type"`
	BindPoint  uint32  `c:"BindPoint"`
	BindCount  uint32  `c:"BindCount"`
	Flags      uint32  `c:"uFlags"`
	ReturnType uint32  `c:"ReturnType"`
	Dimension  uint32  `c:"Dimension"`
	NumSamples uint32  `c:"NumSamples"`
	Space      uint32  `c:"Space"`
	ID         uint32  `c:"uID"`
}

// SignatureParameterDesc is the Go mirror of D3D12_SIGNATURE_PARAMETER_DESC
// (ID3D12ShaderReflection::GetInputParameterDesc): one input or output of a shader stage.
//
// Flutter: no counterpart
type SignatureParameterDesc struct {
	SemanticName  uintptr `c:"SemanticName"`
	SemanticIndex uint32  `c:"SemanticIndex"`
	Register      uint32  `c:"Register"`
	SystemValue   uint32  `c:"SystemValueType"`
	ComponentType uint32  `c:"ComponentType"`
	Mask          uint8   `c:"Mask"`
	ReadWriteMask uint8   `c:"ReadWriteMask"`
	_             [2]byte
	Stream        uint32 `c:"Stream"`
	MinPrecision  uint32 `c:"MinPrecision"`
}

// The kinds of resource that D3D_SHADER_INPUT_TYPE names.
const (
	// SITConstantBuffer is D3D_SIT_CBUFFER.
	//
	// Flutter: no counterpart
	SITConstantBuffer uint32 = 0
	// SITTexture is D3D_SIT_TEXTURE.
	//
	// Flutter: no counterpart
	SITTexture uint32 = 2
	// SITSampler is D3D_SIT_SAMPLER.
	//
	// Flutter: no counterpart
	SITSampler uint32 = 3
)

// The positions of the methods in the vtables of the interfaces the shaders use; the layout test checks each.
const (
	// blobGetBufferPointer and blobGetBufferSize are ID3DBlob::GetBufferPointer and GetBufferSize.
	blobGetBufferPointer = 3
	blobGetBufferSize    = 4
	// reflectGetDesc, reflectGetResourceBindingDesc and reflectGetInputParameterDesc are
	// ID3D12ShaderReflection::GetDesc, GetResourceBindingDesc and GetInputParameterDesc.
	reflectGetDesc                = 3
	reflectGetResourceBindingDesc = 6
	reflectGetInputParameterDesc  = 7
)
