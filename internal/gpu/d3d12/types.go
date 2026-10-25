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

// d3d12DescriptorRange is the Go mirror of D3D12_DESCRIPTOR_RANGE (version 1.0 of the root signature).
type d3d12DescriptorRange struct {
	RangeType                         uint32 `c:"RangeType"`
	NumDescriptors                    uint32 `c:"NumDescriptors"`
	BaseShaderRegister                uint32 `c:"BaseShaderRegister"`
	RegisterSpace                     uint32 `c:"RegisterSpace"`
	OffsetInDescriptorsFromTableStart uint32 `c:"OffsetInDescriptorsFromTableStart"`
}

// d3d12RootParameter is the Go mirror of D3D12_ROOT_PARAMETER: the type, the union (a table or a root
// descriptor, the larger variant is a table) and the visibility. The union is two words; rootCBV and
// rootTable write the variants.
type d3d12RootParameter struct {
	ParameterType    uint32    `c:"ParameterType"`
	Union            [2]uint64 `c:"DescriptorTable"`
	ShaderVisibility uint32    `c:"ShaderVisibility"`
}

// d3d12RootSignatureDesc is the Go mirror of D3D12_ROOT_SIGNATURE_DESC (version 1.0).
type d3d12RootSignatureDesc struct {
	NumParameters     uint32  `c:"NumParameters"`
	Parameters        uintptr `c:"pParameters"`
	NumStaticSamplers uint32  `c:"NumStaticSamplers"`
	StaticSamplers    uintptr `c:"pStaticSamplers"`
	Flags             uint32  `c:"Flags"`
}

// d3d12ShaderBytecode is the Go mirror of D3D12_SHADER_BYTECODE.
type d3d12ShaderBytecode struct {
	Code   uintptr `c:"pShaderBytecode"`
	Length uintptr `c:"BytecodeLength"`
}

// d3d12StreamOutputDesc is the Go mirror of D3D12_STREAM_OUTPUT_DESC (unused: all zero).
type d3d12StreamOutputDesc struct {
	Declaration      uintptr `c:"pSODeclaration"`
	NumEntries       uint32  `c:"NumEntries"`
	BufferStrides    uintptr `c:"pBufferStrides"`
	NumStrides       uint32  `c:"NumStrides"`
	RasterizedStream uint32  `c:"RasterizedStream"`
}

// d3d12RenderTargetBlendDesc is the Go mirror of D3D12_RENDER_TARGET_BLEND_DESC.
type d3d12RenderTargetBlendDesc struct {
	BlendEnable           uint32 `c:"BlendEnable"`
	LogicOpEnable         uint32 `c:"LogicOpEnable"`
	SrcBlend              uint32 `c:"SrcBlend"`
	DestBlend             uint32 `c:"DestBlend"`
	BlendOp               uint32 `c:"BlendOp"`
	SrcBlendAlpha         uint32 `c:"SrcBlendAlpha"`
	DestBlendAlpha        uint32 `c:"DestBlendAlpha"`
	BlendOpAlpha          uint32 `c:"BlendOpAlpha"`
	LogicOp               uint32 `c:"LogicOp"`
	RenderTargetWriteMask uint8  `c:"RenderTargetWriteMask"`
}

// d3d12BlendDesc is the Go mirror of D3D12_BLEND_DESC.
type d3d12BlendDesc struct {
	AlphaToCoverageEnable  uint32                        `c:"AlphaToCoverageEnable"`
	IndependentBlendEnable uint32                        `c:"IndependentBlendEnable"`
	RenderTarget           [8]d3d12RenderTargetBlendDesc `c:"RenderTarget"`
}

// d3d12RasterizerDesc is the Go mirror of D3D12_RASTERIZER_DESC.
type d3d12RasterizerDesc struct {
	FillMode              uint32  `c:"FillMode"`
	CullMode              uint32  `c:"CullMode"`
	FrontCounterClockwise uint32  `c:"FrontCounterClockwise"`
	DepthBias             int32   `c:"DepthBias"`
	DepthBiasClamp        float32 `c:"DepthBiasClamp"`
	SlopeScaledDepthBias  float32 `c:"SlopeScaledDepthBias"`
	DepthClipEnable       uint32  `c:"DepthClipEnable"`
	MultisampleEnable     uint32  `c:"MultisampleEnable"`
	AntialiasedLineEnable uint32  `c:"AntialiasedLineEnable"`
	ForcedSampleCount     uint32  `c:"ForcedSampleCount"`
	ConservativeRaster    uint32  `c:"ConservativeRaster"`
}

// d3d12DepthStencilOpDesc is the Go mirror of D3D12_DEPTH_STENCILOP_DESC.
type d3d12DepthStencilOpDesc struct {
	StencilFailOp      uint32 `c:"StencilFailOp"`
	StencilDepthFailOp uint32 `c:"StencilDepthFailOp"`
	StencilPassOp      uint32 `c:"StencilPassOp"`
	StencilFunc        uint32 `c:"StencilFunc"`
}

// d3d12DepthStencilDesc is the Go mirror of D3D12_DEPTH_STENCIL_DESC.
type d3d12DepthStencilDesc struct {
	DepthEnable      uint32                  `c:"DepthEnable"`
	DepthWriteMask   uint32                  `c:"DepthWriteMask"`
	DepthFunc        uint32                  `c:"DepthFunc"`
	StencilEnable    uint32                  `c:"StencilEnable"`
	StencilReadMask  uint8                   `c:"StencilReadMask"`
	StencilWriteMask uint8                   `c:"StencilWriteMask"`
	FrontFace        d3d12DepthStencilOpDesc `c:"FrontFace"`
	BackFace         d3d12DepthStencilOpDesc `c:"BackFace"`
}

// d3d12InputElementDesc is the Go mirror of D3D12_INPUT_ELEMENT_DESC.
type d3d12InputElementDesc struct {
	SemanticName         uintptr `c:"SemanticName"`
	SemanticIndex        uint32  `c:"SemanticIndex"`
	Format               uint32  `c:"Format"`
	InputSlot            uint32  `c:"InputSlot"`
	AlignedByteOffset    uint32  `c:"AlignedByteOffset"`
	InputSlotClass       uint32  `c:"InputSlotClass"`
	InstanceDataStepRate uint32  `c:"InstanceDataStepRate"`
}

// d3d12InputLayoutDesc is the Go mirror of D3D12_INPUT_LAYOUT_DESC.
type d3d12InputLayoutDesc struct {
	Elements    uintptr `c:"pInputElementDescs"`
	NumElements uint32  `c:"NumElements"`
}

// d3d12CachedPipelineState is the Go mirror of D3D12_CACHED_PIPELINE_STATE.
type d3d12CachedPipelineState struct {
	Blob uintptr `c:"pCachedBlob"`
	Size uintptr `c:"CachedBlobSizeInBytes"`
}

// d3d12GraphicsPipelineStateDesc is the Go mirror of D3D12_GRAPHICS_PIPELINE_STATE_DESC.
type d3d12GraphicsPipelineStateDesc struct {
	RootSignature         uintptr                  `c:"pRootSignature"`
	VS                    d3d12ShaderBytecode      `c:"VS"`
	PS                    d3d12ShaderBytecode      `c:"PS"`
	DS                    d3d12ShaderBytecode      `c:"DS"`
	HS                    d3d12ShaderBytecode      `c:"HS"`
	GS                    d3d12ShaderBytecode      `c:"GS"`
	StreamOutput          d3d12StreamOutputDesc    `c:"StreamOutput"`
	BlendState            d3d12BlendDesc           `c:"BlendState"`
	SampleMask            uint32                   `c:"SampleMask"`
	RasterizerState       d3d12RasterizerDesc      `c:"RasterizerState"`
	DepthStencilState     d3d12DepthStencilDesc    `c:"DepthStencilState"`
	InputLayout           d3d12InputLayoutDesc     `c:"InputLayout"`
	IBStripCutValue       uint32                   `c:"IBStripCutValue"`
	PrimitiveTopologyType uint32                   `c:"PrimitiveTopologyType"`
	NumRenderTargets      uint32                   `c:"NumRenderTargets"`
	RTVFormats            [8]uint32                `c:"RTVFormats"`
	DSVFormat             uint32                   `c:"DSVFormat"`
	SampleDesc            dxgiSampleDesc           `c:"SampleDesc"`
	NodeMask              uint32                   `c:"NodeMask"`
	CachedPSO             d3d12CachedPipelineState `c:"CachedPSO"`
	Flags                 uint32                   `c:"Flags"`
}

// d3d12Viewport is the Go mirror of D3D12_VIEWPORT.
type d3d12Viewport struct {
	TopLeftX float32 `c:"TopLeftX"`
	TopLeftY float32 `c:"TopLeftY"`
	Width    float32 `c:"Width"`
	Height   float32 `c:"Height"`
	MinDepth float32 `c:"MinDepth"`
	MaxDepth float32 `c:"MaxDepth"`
}

// d3d12Rect is the Go mirror of D3D12_RECT (RECT).
type d3d12Rect struct {
	Left   int32 `c:"left"`
	Top    int32 `c:"top"`
	Right  int32 `c:"right"`
	Bottom int32 `c:"bottom"`
}

// dxgiSwapChainDesc1 is the Go mirror of DXGI_SWAP_CHAIN_DESC1 (SPEC §9.8 "Swap chain").
type dxgiSwapChainDesc1 struct {
	Width       uint32         `c:"Width"`
	Height      uint32         `c:"Height"`
	Format      uint32         `c:"Format"`
	Stereo      int32          `c:"Stereo"`
	SampleDesc  dxgiSampleDesc `c:"SampleDesc"`
	BufferUsage uint32         `c:"BufferUsage"`
	BufferCount uint32         `c:"BufferCount"`
	Scaling     uint32         `c:"Scaling"`
	SwapEffect  uint32         `c:"SwapEffect"`
	AlphaMode   uint32         `c:"AlphaMode"`
	Flags       uint32         `c:"Flags"`
}

// dxgiPresentParameters is the Go mirror of DXGI_PRESENT_PARAMETERS (IDXGISwapChain1::Present1): the
// dirty rectangles, and the scroll rectangle and offset, which the device leaves out.
type dxgiPresentParameters struct {
	DirtyRectsCount uint32  `c:"DirtyRectsCount"`
	DirtyRects      uintptr `c:"pDirtyRects"`
	ScrollRect      uintptr `c:"pScrollRect"`
	ScrollOffset    uintptr `c:"pScrollOffset"`
}

// d3d12VertexBufferView is the Go mirror of D3D12_VERTEX_BUFFER_VIEW.
type d3d12VertexBufferView struct {
	BufferLocation uint64 `c:"BufferLocation"`
	SizeInBytes    uint32 `c:"SizeInBytes"`
	StrideInBytes  uint32 `c:"StrideInBytes"`
}

// d3d12IndexBufferView is the Go mirror of D3D12_INDEX_BUFFER_VIEW.
type d3d12IndexBufferView struct {
	BufferLocation uint64 `c:"BufferLocation"`
	SizeInBytes    uint32 `c:"SizeInBytes"`
	Format         uint32 `c:"Format"`
}

// d3d12DiscardRegion is the Go mirror of D3D12_DISCARD_REGION.
type d3d12DiscardRegion struct {
	NumRects         uint32  `c:"NumRects"`
	Rects            uintptr `c:"pRects"`
	FirstSubresource uint32  `c:"FirstSubresource"`
	NumSubresources  uint32  `c:"NumSubresources"`
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

	// The swap chain (SPEC §9.8): DXGI_USAGE_RENDER_TARGET_OUTPUT, DXGI_SCALING_NONE,
	// DXGI_SWAP_EFFECT_FLIP_DISCARD, DXGI_ALPHA_MODE_IGNORE, DXGI_MWA_NO_WINDOW_CHANGES and
	// DXGI_MWA_NO_ALT_ENTER, and DXGI_STATUS_OCCLUDED, the success code of a Present to a window that
	// cannot be seen.
	dxgiUsageRenderTargetOutput uint32 = 0x20
	dxgiScalingNone             uint32 = 1
	dxgiSwapEffectFlipDiscard   uint32 = 4
	dxgiAlphaModeIgnore         uint32 = 3
	dxgiMWANoWindowChanges      uint32 = 1
	dxgiMWANoAltEnter           uint32 = 2
	dxgiStatusOccluded          uint32 = 0x087A0001

	// D3D12_MESSAGE_SEVERITY.
	messageSeverityCorruption uint32 = 0
	messageSeverityError      uint32 = 1
	messageSeverityWarning    uint32 = 2

	// D3D12_RESOURCE_BINDING_TIER_1 is the tier that the device needs (SPEC §9.8).
	resourceBindingTier1 uint32 = 1

	// D3D12_ROOT_PARAMETER_TYPE, D3D12_DESCRIPTOR_RANGE_TYPE, D3D12_SHADER_VISIBILITY and
	// D3D_ROOT_SIGNATURE_VERSION_1 (SPEC §9.8 "Binding model").
	rootParameterTable     uint32 = 0
	rootParameterCBV       uint32 = 2
	descriptorRangeSRV     uint32 = 0
	descriptorRangeSampler uint32 = 3
	shaderVisibilityAll    uint32 = 0
	shaderVisibilityVertex uint32 = 1
	shaderVisibilityPixel  uint32 = 5
	rootSignatureVersion1  uint32 = 1
	// D3D12_ROOT_SIGNATURE_FLAG_*: the input assembler reads the vertex buffers, and the stages the
	// renderer does not use cannot read the root arguments.
	rootFlagAllowInputLayout uint32 = 0x1
	rootFlagDenyHull         uint32 = 0x4
	rootFlagDenyDomain       uint32 = 0x8
	rootFlagDenyGeometry     uint32 = 0x10
	// D3D12_DESCRIPTOR_RANGE_OFFSET_APPEND, the offset that follows the range before.
	descriptorRangeAppend uint32 = 0xFFFFFFFF
	// D3D12_DESCRIPTOR_HEAP_FLAG_SHADER_VISIBLE.
	descriptorHeapShaderVisible uint32 = 1

	// D3D12_BLEND, D3D12_BLEND_OP_ADD and D3D12_COLOR_WRITE_ENABLE_ALL.
	blendZero           uint32 = 1
	blendOne            uint32 = 2
	blendSrcAlpha       uint32 = 5
	blendInvSrcAlpha    uint32 = 6
	blendDestAlpha      uint32 = 7
	blendInvDestAlpha   uint32 = 8
	blendOpAdd          uint32 = 1
	colorWriteEnableAll uint8  = 0xF
	// D3D12_LOGIC_OP_NOOP, D3D12_FILL_MODE_SOLID, D3D12_CULL_MODE_NONE, D3D12_COMPARISON_FUNC_ALWAYS,
	// D3D12_STENCIL_OP_KEEP, D3D12_DEPTH_WRITE_MASK_ZERO and D3D12_DEFAULT_SAMPLE_MASK.
	logicOpNoop        uint32 = 4
	fillModeSolid      uint32 = 3
	cullModeNone       uint32 = 1
	comparisonAlways   uint32 = 8
	stencilOpKeep      uint32 = 1
	depthWriteMaskZero uint32 = 0
	defaultSampleMask  uint32 = 0xFFFFFFFF
	// D3D12_PRIMITIVE_TOPOLOGY_TYPE_TRIANGLE and D3D_PRIMITIVE_TOPOLOGY_TRIANGLELIST.
	topologyTypeTriangle uint32 = 3
	topologyTriangleList uint32 = 4
	// D3D12_INPUT_CLASSIFICATION.
	inputPerVertex   uint32 = 0
	inputPerInstance uint32 = 1
	// The formats of the vertex attributes and of the index buffer.
	dxgiFormatR32     uint32 = 41 // DXGI_FORMAT_R32_FLOAT
	dxgiFormatRG32    uint32 = 16 // DXGI_FORMAT_R32G32_FLOAT
	dxgiFormatRGB32   uint32 = 6  // DXGI_FORMAT_R32G32B32_FLOAT
	dxgiFormatRGBA32  uint32 = 2  // DXGI_FORMAT_R32G32B32A32_FLOAT
	dxgiFormatR16Uint uint32 = 57 // DXGI_FORMAT_R16_UINT
	// D3D12_ERROR_ADAPTER_NOT_FOUND and D3D12_ERROR_DRIVER_VERSION_MISMATCH: what a pipeline library
	// says of a blob that belongs to another adapter or driver (SPEC §9.8 "Pipelines").
	d3d12ErrorAdapterNotFound       uint32 = 0x887E0001
	d3d12ErrorDriverVersionMismatch uint32 = 0x887E0002
)
