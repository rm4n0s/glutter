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

// The Go mirrors of the structures of dxgi.h, d3d12.h and d3d12sdklayers.h that the device (SPEC §9.8)
// passes to Direct3D 12 and DXGI or reads back, and their constants. The types are not exported: the
// device is the only user. Each field carries the name of the member in the header, and the layout
// test (layout_test.go) checks the size and the offset of each, and the value of each constant,
// against the mingw-w64 headers. A union is the variant that the device uses, followed by padding up
// to the size of the largest variant.

// luid is the Go mirror of LUID: the locally unique identifier of an adapter.
type luid struct {
	LowPart  uint32 `c:"LowPart"`
	HighPart int32  `c:"HighPart"`
}

// dxgiAdapterDesc1 is the Go mirror of DXGI_ADAPTER_DESC1 (IDXGIAdapter1::GetDesc1).
type dxgiAdapterDesc1 struct {
	Description           [128]uint16 `c:"Description"`
	VendorID              uint32      `c:"VendorId"`
	DeviceID              uint32      `c:"DeviceId"`
	SubSysID              uint32      `c:"SubSysId"`
	Revision              uint32      `c:"Revision"`
	DedicatedVideoMemory  uintptr     `c:"DedicatedVideoMemory"`
	DedicatedSystemMemory uintptr     `c:"DedicatedSystemMemory"`
	SharedSystemMemory    uintptr     `c:"SharedSystemMemory"`
	AdapterLuid           luid        `c:"AdapterLuid"`
	Flags                 uint32      `c:"Flags"`
}

// dxgiSampleDesc is the Go mirror of DXGI_SAMPLE_DESC.
type dxgiSampleDesc struct {
	Count   uint32 `c:"Count"`
	Quality uint32 `c:"Quality"`
}

// d3d12CommandQueueDesc is the Go mirror of D3D12_COMMAND_QUEUE_DESC.
type d3d12CommandQueueDesc struct {
	Type     uint32 `c:"Type"`
	Priority int32  `c:"Priority"`
	Flags    uint32 `c:"Flags"`
	NodeMask uint32 `c:"NodeMask"`
}

// d3d12HeapProperties is the Go mirror of D3D12_HEAP_PROPERTIES.
type d3d12HeapProperties struct {
	Type                 uint32 `c:"Type"`
	CPUPageProperty      uint32 `c:"CPUPageProperty"`
	MemoryPoolPreference uint32 `c:"MemoryPoolPreference"`
	CreationNodeMask     uint32 `c:"CreationNodeMask"`
	VisibleNodeMask      uint32 `c:"VisibleNodeMask"`
}

// d3d12ResourceDesc is the Go mirror of D3D12_RESOURCE_DESC.
type d3d12ResourceDesc struct {
	Dimension        uint32         `c:"Dimension"`
	Alignment        uint64         `c:"Alignment"`
	Width            uint64         `c:"Width"`
	Height           uint32         `c:"Height"`
	DepthOrArraySize uint16         `c:"DepthOrArraySize"`
	MipLevels        uint16         `c:"MipLevels"`
	Format           uint32         `c:"Format"`
	SampleDesc       dxgiSampleDesc `c:"SampleDesc"`
	Layout           uint32         `c:"Layout"`
	Flags            uint32         `c:"Flags"`
}

// d3d12Range is the Go mirror of D3D12_RANGE: the bytes of a resource that the CPU reads or wrote.
type d3d12Range struct {
	Begin uintptr `c:"Begin"`
	End   uintptr `c:"End"`
}

// d3d12Box is the Go mirror of D3D12_BOX: the region of a texture that a copy reads.
type d3d12Box struct {
	Left   uint32 `c:"left"`
	Top    uint32 `c:"top"`
	Front  uint32 `c:"front"`
	Right  uint32 `c:"right"`
	Bottom uint32 `c:"bottom"`
	Back   uint32 `c:"back"`
}

// d3d12SubresourceFootprint is the Go mirror of D3D12_SUBRESOURCE_FOOTPRINT: the layout of one
// subresource in a buffer.
type d3d12SubresourceFootprint struct {
	Format   uint32 `c:"Format"`
	Width    uint32 `c:"Width"`
	Height   uint32 `c:"Height"`
	Depth    uint32 `c:"Depth"`
	RowPitch uint32 `c:"RowPitch"`
}

// d3d12PlacedFootprint is the Go mirror of D3D12_PLACED_SUBRESOURCE_FOOTPRINT.
type d3d12PlacedFootprint struct {
	Offset    uint64                    `c:"Offset"`
	Footprint d3d12SubresourceFootprint `c:"Footprint"`
}

// d3d12TextureCopyLocation is the Go mirror of D3D12_TEXTURE_COPY_LOCATION with the placed footprint
// of a buffer (Type PLACED_FOOTPRINT): the union's larger variant.
type d3d12TextureCopyLocation struct {
	Resource uintptr              `c:"pResource"`
	Type     uint32               `c:"Type"`
	Placed   d3d12PlacedFootprint `c:"PlacedFootprint"`
}

// d3d12TextureCopyIndex is the Go mirror of D3D12_TEXTURE_COPY_LOCATION with the index of a
// subresource of a texture (Type SUBRESOURCE_INDEX), padded to the size of the union.
type d3d12TextureCopyIndex struct {
	Resource uintptr `c:"pResource"`
	Type     uint32  `c:"Type"`
	_        uint32
	Index    uint32 `c:"SubresourceIndex"`
	_        [28]byte
}

// d3d12TransitionBarrier is the Go mirror of D3D12_RESOURCE_TRANSITION_BARRIER.
type d3d12TransitionBarrier struct {
	Resource    uintptr `c:"pResource"`
	Subresource uint32  `c:"Subresource"`
	StateBefore uint32  `c:"StateBefore"`
	StateAfter  uint32  `c:"StateAfter"`
}

// d3d12ResourceBarrier is the Go mirror of D3D12_RESOURCE_BARRIER with the transition variant, which
// is the largest of the union.
type d3d12ResourceBarrier struct {
	Type       uint32                 `c:"Type"`
	Flags      uint32                 `c:"Flags"`
	Transition d3d12TransitionBarrier `c:"Transition"`
}

// d3d12DescriptorHeapDesc is the Go mirror of D3D12_DESCRIPTOR_HEAP_DESC.
type d3d12DescriptorHeapDesc struct {
	Type           uint32 `c:"Type"`
	NumDescriptors uint32 `c:"NumDescriptors"`
	Flags          uint32 `c:"Flags"`
	NodeMask       uint32 `c:"NodeMask"`
}

// d3d12SamplerDesc is the Go mirror of D3D12_SAMPLER_DESC.
type d3d12SamplerDesc struct {
	Filter         uint32     `c:"Filter"`
	AddressU       uint32     `c:"AddressU"`
	AddressV       uint32     `c:"AddressV"`
	AddressW       uint32     `c:"AddressW"`
	MipLODBias     float32    `c:"MipLODBias"`
	MaxAnisotropy  uint32     `c:"MaxAnisotropy"`
	ComparisonFunc uint32     `c:"ComparisonFunc"`
	BorderColor    [4]float32 `c:"BorderColor"`
	MinLOD         float32    `c:"MinLOD"`
	MaxLOD         float32    `c:"MaxLOD"`
}

// d3d12Tex2DSRV is the Go mirror of D3D12_TEX2D_SRV.
type d3d12Tex2DSRV struct {
	MostDetailedMip     uint32  `c:"MostDetailedMip"`
	MipLevels           uint32  `c:"MipLevels"`
	PlaneSlice          uint32  `c:"PlaneSlice"`
	ResourceMinLODClamp float32 `c:"ResourceMinLODClamp"`
}

// d3d12ShaderResourceViewDesc is the Go mirror of D3D12_SHADER_RESOURCE_VIEW_DESC with the
// Texture2D variant, padded to the size of the union.
type d3d12ShaderResourceViewDesc struct {
	Format                  uint32 `c:"Format"`
	ViewDimension           uint32 `c:"ViewDimension"`
	Shader4ComponentMapping uint32 `c:"Shader4ComponentMapping"`
	_                       uint32
	Texture2D               d3d12Tex2DSRV `c:"Texture2D"`
	_                       [8]byte
}

// d3d12Tex2DRTV is the Go mirror of D3D12_TEX2D_RTV.
type d3d12Tex2DRTV struct {
	MipSlice   uint32 `c:"MipSlice"`
	PlaneSlice uint32 `c:"PlaneSlice"`
}

// d3d12RenderTargetViewDesc is the Go mirror of D3D12_RENDER_TARGET_VIEW_DESC with the Texture2D
// variant, padded to the size of the union.
type d3d12RenderTargetViewDesc struct {
	Format        uint32        `c:"Format"`
	ViewDimension uint32        `c:"ViewDimension"`
	Texture2D     d3d12Tex2DRTV `c:"Texture2D"`
	_             [8]byte
}

// d3d12FeatureLevels is the Go mirror of D3D12_FEATURE_DATA_FEATURE_LEVELS.
type d3d12FeatureLevels struct {
	NumFeatureLevels         uint32  `c:"NumFeatureLevels"`
	FeatureLevelsRequested   uintptr `c:"pFeatureLevelsRequested"`
	MaxSupportedFeatureLevel uint32  `c:"MaxSupportedFeatureLevel"`
}

// d3d12Options is the Go mirror of D3D12_FEATURE_DATA_D3D12_OPTIONS.
type d3d12Options struct {
	DoublePrecisionFloatShaderOps                                         int32  `c:"DoublePrecisionFloatShaderOps"`
	OutputMergerLogicOp                                                   int32  `c:"OutputMergerLogicOp"`
	MinPrecisionSupport                                                   uint32 `c:"MinPrecisionSupport"`
	TiledResourcesTier                                                    uint32 `c:"TiledResourcesTier"`
	ResourceBindingTier                                                   uint32 `c:"ResourceBindingTier"`
	PSSpecifiedStencilRefSupported                                        int32  `c:"PSSpecifiedStencilRefSupported"`
	TypedUAVLoadAdditionalFormats                                         int32  `c:"TypedUAVLoadAdditionalFormats"`
	ROVsSupported                                                         int32  `c:"ROVsSupported"`
	ConservativeRasterizationTier                                         uint32 `c:"ConservativeRasterizationTier"`
	MaxGPUVirtualAddressBitsPerResource                                   uint32 `c:"MaxGPUVirtualAddressBitsPerResource"`
	StandardSwizzle64KBSupported                                          int32  `c:"StandardSwizzle64KBSupported"`
	CrossNodeSharingTier                                                  uint32 `c:"CrossNodeSharingTier"`
	CrossAdapterRowMajorTextureSupported                                  int32  `c:"CrossAdapterRowMajorTextureSupported"`
	VPAndRTArrayIndexFromAnyShaderFeedingRasterizerSupportedWithoutGSEmul int32  `c:"VPAndRTArrayIndexFromAnyShaderFeedingRasterizerSupportedWithoutGSEmulation"`
	ResourceHeapTier                                                      uint32 `c:"ResourceHeapTier"`
}

// d3d12Message is the Go mirror of D3D12_MESSAGE: one message of the debug layer's info queue.
type d3d12Message struct {
	Category          uint32  `c:"Category"`
	Severity          uint32  `c:"Severity"`
	ID                uint32  `c:"ID"`
	Description       uintptr `c:"pDescription"`
	DescriptionLength uintptr `c:"DescriptionByteLength"`
}

// The values of the enumerations and flags of the headers that the device uses; the layout test
// checks each one.
const (
	// DXGI_FORMAT.
	dxgiFormatUnknown   uint32 = 0
	dxgiFormatRGBA8     uint32 = 28 // DXGI_FORMAT_R8G8B8A8_UNORM
	dxgiFormatBGRA8     uint32 = 87 // DXGI_FORMAT_B8G8R8A8_UNORM
	dxgiFormatR8        uint32 = 61 // DXGI_FORMAT_R8_UNORM
	dxgiAdapterSoftware uint32 = 2  // DXGI_ADAPTER_FLAG_SOFTWARE

	// DXGI_CREATE_FACTORY_DEBUG and DXGI_GPU_PREFERENCE_HIGH_PERFORMANCE.
	dxgiCreateFactoryDebug    uint32 = 1
	dxgiGPUPreferenceHighPerf uint32 = 2

	// D3D_FEATURE_LEVEL.
	featureLevel11_0 uint32 = 0xb000
	featureLevel11_1 uint32 = 0xb100
	featureLevel12_0 uint32 = 0xc000
	featureLevel12_1 uint32 = 0xc100
	featureLevel12_2 uint32 = 0xc200

	// D3D12_FEATURE.
	featureD3D12Options  uint32 = 0
	featureFeatureLevels uint32 = 2

	// D3D12_COMMAND_LIST_TYPE_DIRECT.
	commandListDirect uint32 = 0

	// D3D12_HEAP_TYPE.
	heapTypeDefault  uint32 = 1
	heapTypeUpload   uint32 = 2
	heapTypeReadback uint32 = 3

	// D3D12_RESOURCE_DIMENSION, D3D12_TEXTURE_LAYOUT and D3D12_RESOURCE_FLAGS.
	resourceDimensionBuffer    uint32 = 1
	resourceDimensionTexture2D uint32 = 3
	textureLayoutUnknown       uint32 = 0
	textureLayoutRowMajor      uint32 = 1
	resourceFlagAllowRT        uint32 = 1

	// D3D12_RESOURCE_STATES.
	resourceStateCommon        uint32 = 0
	resourceStateRenderTarget  uint32 = 0x4
	resourceStatePixelShaderRS uint32 = 0x80
	resourceStateCopyDest      uint32 = 0x400
	resourceStateCopySource    uint32 = 0x800
	resourceStateGenericRead   uint32 = 0xAC3

	// D3D12_RESOURCE_BARRIER_TYPE_TRANSITION and D3D12_RESOURCE_BARRIER_ALL_SUBRESOURCES.
	barrierTypeTransition uint32 = 0
	allSubresources       uint32 = 0xFFFFFFFF

	// D3D12_TEXTURE_COPY_TYPE.
	textureCopySubresourceIndex uint32 = 0
	textureCopyPlacedFootprint  uint32 = 1

	// D3D12_DESCRIPTOR_HEAP_TYPE.
	descriptorHeapSRV     uint32 = 0
	descriptorHeapSampler uint32 = 1
	descriptorHeapRTV     uint32 = 2

	// D3D12_SRV_DIMENSION_TEXTURE2D and D3D12_RTV_DIMENSION_TEXTURE2D.
	srvDimensionTexture2D uint32 = 4
	rtvDimensionTexture2D uint32 = 4

	// D3D12_DEFAULT_SHADER_4_COMPONENT_MAPPING.
	defaultComponentMapping uint32 = 0x1688

	// D3D12_FILTER: bit 0 is the mip filter, bit 2 the magnification and bit 4 the minification; set is linear.
	filterMinMagMipPoint  uint32 = 0x00
	filterMipLinear       uint32 = 0x01
	filterMagLinear       uint32 = 0x04
	filterMinLinear       uint32 = 0x10
	filterMinMagMipLinear uint32 = 0x15

	// D3D12_TEXTURE_ADDRESS_MODE.
	addressModeWrap   uint32 = 1
	addressModeMirror uint32 = 2
	addressModeClamp  uint32 = 3

	// D3D12_COMPARISON_FUNC_NEVER and D3D12_FLOAT32_MAX as the headers define them.
	comparisonNever uint32  = 1
	float32Max      float32 = 3.402823466e+38

	// The alignments of copies (d3d12.h).
	textureDataPitchAlignment     = 256 // D3D12_TEXTURE_DATA_PITCH_ALIGNMENT
	textureDataPlacementAlignment = 512 // D3D12_TEXTURE_DATA_PLACEMENT_ALIGNMENT

	// DXGI_ERROR_DRIVER_INTERNAL_ERROR, DXGI_ERROR_NOT_FOUND (the end of an enumeration) and
	// DXGI_ERROR_SDK_COMPONENT_MISSING (the debug layer is not installed), as the unsigned codes.
	dxgiErrorDriverInternalError uint32 = 0x887A0020
	dxgiErrorNotFound            uint32 = 0x887A0002
	dxgiErrorSDKComponentMissing uint32 = 0x887A002D

	// D3D12_MESSAGE_SEVERITY.
	messageSeverityCorruption uint32 = 0
	messageSeverityError      uint32 = 1
	messageSeverityWarning    uint32 = 2

	// D3D12_RESOURCE_BINDING_TIER_1 is the tier that the device needs (SPEC §9.8).
	resourceBindingTier1 uint32 = 1
)
