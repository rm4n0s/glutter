//go:build !windows

package d3d12

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32/layouttest"
)

// d3d12RootDescriptorTable is the Go mirror of D3D12_ROOT_DESCRIPTOR_TABLE: the table variant of the
// union of a root parameter, which d3d12RootParameter holds as two words (only the layout test uses it).
type d3d12RootDescriptorTable struct {
	NumDescriptorRanges uint32  `c:"NumDescriptorRanges"`
	DescriptorRanges    uintptr `c:"pDescriptorRanges"`
}

// d3d12RootDescriptor is the Go mirror of D3D12_ROOT_DESCRIPTOR: the variant of a root constant buffer
// view, the first word of the union (only the layout test uses it).
type d3d12RootDescriptor struct {
	ShaderRegister uint32 `c:"ShaderRegister"`
	RegisterSpace  uint32 `c:"RegisterSpace"`
}

// TestLayoutAgainstTheHeaders checks the Go mirrors of the shader reflection and of the device, the
// positions of the methods of the interfaces the backend calls, the compile flags and the values of
// the enumerations against the mingw-w64 headers (SPEC §25.2). It runs on the Linux host and is
// skipped, with the install command, when the compiler is missing.
func TestLayoutAgainstTheHeaders(t *testing.T) {
	layouttest.Check(t, layouttest.Spec{
		Headers: []string{"windows.h", "d3dcommon.h", "d3dcompiler.h", "d3d12shader.h", "d3d12.h", "d3d12sdklayers.h", "dxgi1_6.h"},
		Structs: []layouttest.Struct{
			layouttest.StructOf("D3D12_SHADER_DESC", ShaderDesc{}),
			layouttest.StructOf("D3D12_SHADER_INPUT_BIND_DESC", ShaderInputBindDesc{}),
			layouttest.StructOf("D3D12_SIGNATURE_PARAMETER_DESC", SignatureParameterDesc{}),
			layouttest.StructOf("LUID", luid{}),
			layouttest.StructOf("DXGI_ADAPTER_DESC1", dxgiAdapterDesc1{}),
			layouttest.StructOf("DXGI_SAMPLE_DESC", dxgiSampleDesc{}),
			layouttest.StructOf("D3D12_COMMAND_QUEUE_DESC", d3d12CommandQueueDesc{}),
			layouttest.StructOf("D3D12_HEAP_PROPERTIES", d3d12HeapProperties{}),
			layouttest.StructOf("D3D12_RESOURCE_DESC", d3d12ResourceDesc{}),
			layouttest.StructOf("D3D12_RANGE", d3d12Range{}),
			layouttest.StructOf("D3D12_BOX", d3d12Box{}),
			layouttest.StructOf("D3D12_SUBRESOURCE_FOOTPRINT", d3d12SubresourceFootprint{}),
			layouttest.StructOf("D3D12_PLACED_SUBRESOURCE_FOOTPRINT", d3d12PlacedFootprint{}),
			layouttest.StructOf("D3D12_TEXTURE_COPY_LOCATION", d3d12TextureCopyLocation{}),
			layouttest.StructOf("D3D12_TEXTURE_COPY_LOCATION", d3d12TextureCopyIndex{}),
			layouttest.StructOf("D3D12_RESOURCE_TRANSITION_BARRIER", d3d12TransitionBarrier{}),
			layouttest.StructOf("D3D12_RESOURCE_BARRIER", d3d12ResourceBarrier{}),
			layouttest.StructOf("D3D12_DESCRIPTOR_HEAP_DESC", d3d12DescriptorHeapDesc{}),
			layouttest.StructOf("D3D12_SAMPLER_DESC", d3d12SamplerDesc{}),
			layouttest.StructOf("D3D12_TEX2D_SRV", d3d12Tex2DSRV{}),
			layouttest.StructOf("D3D12_SHADER_RESOURCE_VIEW_DESC", d3d12ShaderResourceViewDesc{}),
			layouttest.StructOf("D3D12_TEX2D_RTV", d3d12Tex2DRTV{}),
			layouttest.StructOf("D3D12_RENDER_TARGET_VIEW_DESC", d3d12RenderTargetViewDesc{}),
			layouttest.StructOf("D3D12_FEATURE_DATA_FEATURE_LEVELS", d3d12FeatureLevels{}),
			layouttest.StructOf("D3D12_FEATURE_DATA_D3D12_OPTIONS", d3d12Options{}),
			layouttest.StructOf("D3D12_MESSAGE", d3d12Message{}),
			layouttest.StructOf("D3D12_DESCRIPTOR_RANGE", d3d12DescriptorRange{}),
			layouttest.StructOf("D3D12_ROOT_DESCRIPTOR_TABLE", d3d12RootDescriptorTable{}),
			layouttest.StructOf("D3D12_ROOT_DESCRIPTOR", d3d12RootDescriptor{}),
			layouttest.StructOf("D3D12_ROOT_PARAMETER", d3d12RootParameter{}),
			layouttest.StructOf("D3D12_ROOT_SIGNATURE_DESC", d3d12RootSignatureDesc{}),
			layouttest.StructOf("D3D12_SHADER_BYTECODE", d3d12ShaderBytecode{}),
			layouttest.StructOf("D3D12_STREAM_OUTPUT_DESC", d3d12StreamOutputDesc{}),
			layouttest.StructOf("D3D12_RENDER_TARGET_BLEND_DESC", d3d12RenderTargetBlendDesc{}),
			layouttest.StructOf("D3D12_BLEND_DESC", d3d12BlendDesc{}),
			layouttest.StructOf("D3D12_RASTERIZER_DESC", d3d12RasterizerDesc{}),
			layouttest.StructOf("D3D12_DEPTH_STENCILOP_DESC", d3d12DepthStencilOpDesc{}),
			layouttest.StructOf("D3D12_DEPTH_STENCIL_DESC", d3d12DepthStencilDesc{}),
			layouttest.StructOf("D3D12_INPUT_ELEMENT_DESC", d3d12InputElementDesc{}),
			layouttest.StructOf("D3D12_INPUT_LAYOUT_DESC", d3d12InputLayoutDesc{}),
			layouttest.StructOf("D3D12_CACHED_PIPELINE_STATE", d3d12CachedPipelineState{}),
			layouttest.StructOf("D3D12_GRAPHICS_PIPELINE_STATE_DESC", d3d12GraphicsPipelineStateDesc{}),
			layouttest.StructOf("D3D12_VIEWPORT", d3d12Viewport{}),
			layouttest.StructOf("D3D12_RECT", d3d12Rect{}),
			layouttest.StructOf("D3D12_VERTEX_BUFFER_VIEW", d3d12VertexBufferView{}),
			layouttest.StructOf("D3D12_INDEX_BUFFER_VIEW", d3d12IndexBufferView{}),
			layouttest.StructOf("D3D12_DISCARD_REGION", d3d12DiscardRegion{}),
			layouttest.StructOf("DXGI_SWAP_CHAIN_DESC1", dxgiSwapChainDesc1{}),
			layouttest.StructOf("DXGI_PRESENT_PARAMETERS", dxgiPresentParameters{}),
		},
		VTables: []layouttest.VTable{
			{CName: "ID3D10BlobVtbl", Methods: []layouttest.Method{
				{Name: "GetBufferPointer", Index: blobGetBufferPointer},
				{Name: "GetBufferSize", Index: blobGetBufferSize},
			}},
			{CName: "ID3D12ShaderReflectionVtbl", Methods: []layouttest.Method{
				{Name: "GetDesc", Index: reflectGetDesc},
				{Name: "GetResourceBindingDesc", Index: reflectGetResourceBindingDesc},
				{Name: "GetInputParameterDesc", Index: reflectGetInputParameterDesc},
			}},
			{CName: "IDXGIFactory6Vtbl", Methods: []layouttest.Method{
				{Name: "EnumAdapters1", Index: factoryEnumAdapters1},
				{Name: "EnumWarpAdapter", Index: factoryEnumWarpAdapter},
				{Name: "EnumAdapterByGpuPreference", Index: factoryEnumAdapterByGPUPreference},
			}},
			{CName: "IDXGIFactory2Vtbl", Methods: []layouttest.Method{
				{Name: "MakeWindowAssociation", Index: factoryMakeWindowAssociation},
				{Name: "CreateSwapChainForHwnd", Index: factoryCreateSwapChainForHwnd},
			}},
			{CName: "IDXGISwapChain3Vtbl", Methods: []layouttest.Method{
				{Name: "GetBuffer", Index: swapChainGetBuffer},
				{Name: "ResizeBuffers", Index: swapChainResizeBuffers},
				{Name: "GetDesc1", Index: swapChainGetDesc1},
				{Name: "Present1", Index: swapChainPresent1},
				{Name: "GetCurrentBackBufferIndex", Index: swapChainGetCurrentBackBufferIndex},
			}},
			{CName: "IDXGIAdapter1Vtbl", Methods: []layouttest.Method{
				{Name: "GetDesc1", Index: adapterGetDesc1},
				{Name: "CheckInterfaceSupport", Index: adapterCheckInterfaceSupport},
			}},
			{CName: "ID3D12DeviceVtbl", Methods: []layouttest.Method{
				{Name: "CreateCommandQueue", Index: deviceCreateCommandQueue},
				{Name: "CreateCommandAllocator", Index: deviceCreateCommandAllocator},
				{Name: "CreateGraphicsPipelineState", Index: deviceCreateGraphicsPipeline},
				{Name: "CreateCommandList", Index: deviceCreateCommandList},
				{Name: "CreateRootSignature", Index: deviceCreateRootSignature},
				{Name: "CopyDescriptorsSimple", Index: deviceCopyDescriptorsSimple},
				{Name: "CheckFeatureSupport", Index: deviceCheckFeatureSupport},
				{Name: "CreateDescriptorHeap", Index: deviceCreateDescriptorHeap},
				{Name: "GetDescriptorHandleIncrementSize", Index: deviceGetDescriptorIncrementSize},
				{Name: "CreateShaderResourceView", Index: deviceCreateShaderResourceView},
				{Name: "CreateRenderTargetView", Index: deviceCreateRenderTargetView},
				{Name: "CreateSampler", Index: deviceCreateSampler},
				{Name: "CreateCommittedResource", Index: deviceCreateCommittedResource},
				{Name: "CreateFence", Index: deviceCreateFence},
				{Name: "GetDeviceRemovedReason", Index: deviceGetDeviceRemovedReason},
				{Name: "GetCopyableFootprints", Index: deviceGetCopyableFootprints},
			}},
			{CName: "ID3D12Device1Vtbl", Methods: []layouttest.Method{{Name: "CreatePipelineLibrary", Index: device1CreatePipelineLibrary}}},
			{CName: "ID3D12PipelineLibraryVtbl", Methods: []layouttest.Method{
				{Name: "StorePipeline", Index: libraryStorePipeline},
				{Name: "LoadGraphicsPipeline", Index: libraryLoadGraphicsPipeline},
				{Name: "GetSerializedSize", Index: librarySerializedSize},
				{Name: "Serialize", Index: librarySerialize},
			}},
			{CName: "ID3D12CommandQueueVtbl", Methods: []layouttest.Method{
				{Name: "ExecuteCommandLists", Index: queueExecuteCommandLists},
				{Name: "Signal", Index: queueSignal},
			}},
			{CName: "ID3D12CommandAllocatorVtbl", Methods: []layouttest.Method{{Name: "Reset", Index: allocatorReset}}},
			{CName: "ID3D12FenceVtbl", Methods: []layouttest.Method{
				{Name: "GetCompletedValue", Index: fenceGetCompletedValue},
				{Name: "SetEventOnCompletion", Index: fenceSetEventOnComplete},
			}},
			{CName: "ID3D12ResourceVtbl", Methods: []layouttest.Method{
				{Name: "Map", Index: resourceMap},
				{Name: "Unmap", Index: resourceUnmap},
				{Name: "GetGPUVirtualAddress", Index: resourceGPUAddress},
			}},
			{CName: "ID3D12DescriptorHeapVtbl", Methods: []layouttest.Method{
				{Name: "GetCPUDescriptorHandleForHeapStart", Index: heapGetCPUHandleStart},
				{Name: "GetGPUDescriptorHandleForHeapStart", Index: heapGetGPUHandleStart},
			}},
			{CName: "ID3D12GraphicsCommandListVtbl", Methods: []layouttest.Method{
				{Name: "Close", Index: listClose},
				{Name: "Reset", Index: listReset},
				{Name: "DrawInstanced", Index: listDrawInstanced},
				{Name: "DrawIndexedInstanced", Index: listDrawIndexedInstanced},
				{Name: "CopyBufferRegion", Index: listCopyBufferRegion},
				{Name: "CopyTextureRegion", Index: listCopyTextureRegion},
				{Name: "IASetPrimitiveTopology", Index: listIASetPrimitiveTopo},
				{Name: "RSSetViewports", Index: listRSSetViewports},
				{Name: "RSSetScissorRects", Index: listRSSetScissorRects},
				{Name: "SetPipelineState", Index: listSetPipelineState},
				{Name: "ResourceBarrier", Index: listResourceBarrier},
				{Name: "SetDescriptorHeaps", Index: listSetDescriptorHeaps},
				{Name: "SetGraphicsRootSignature", Index: listSetRootSignature},
				{Name: "SetGraphicsRootDescriptorTable", Index: listSetRootTable},
				{Name: "SetGraphicsRootConstantBufferView", Index: listSetRootCBV},
				{Name: "IASetIndexBuffer", Index: listIASetIndexBuffer},
				{Name: "IASetVertexBuffers", Index: listIASetVertexBuffers},
				{Name: "OMSetRenderTargets", Index: listOMSetRenderTargets},
				{Name: "ClearRenderTargetView", Index: listClearRenderTargetView},
				{Name: "DiscardResource", Index: listDiscardResource},
			}},
			{CName: "ID3D12DebugVtbl", Methods: []layouttest.Method{{Name: "EnableDebugLayer", Index: debugEnableDebugLayer}}},
			{CName: "ID3D12InfoQueueVtbl", Methods: []layouttest.Method{
				{Name: "SetMessageCountLimit", Index: infoQueueSetMessageCountLimit},
				{Name: "ClearStoredMessages", Index: infoQueueClearStoredMessages},
				{Name: "GetMessage", Index: infoQueueGetMessage},
				{Name: "GetNumStoredMessages", Index: infoQueueGetNumStoredMessages},
			}},
		},
		Consts: []layouttest.Const{
			{Name: "D3DCOMPILE_DEBUG", Value: CompileDebug},
			{Name: "D3DCOMPILE_SKIP_OPTIMIZATION", Value: CompileSkipOptimization},
			{Name: "D3DCOMPILE_OPTIMIZATION_LEVEL3", Value: CompileOptimizationLevel3},
			{Name: "D3D_SIT_CBUFFER", Value: SITConstantBuffer},
			{Name: "D3D_SIT_TEXTURE", Value: SITTexture},
			{Name: "D3D_SIT_SAMPLER", Value: SITSampler},
			{Name: "DXGI_FORMAT_UNKNOWN", Value: dxgiFormatUnknown},
			{Name: "DXGI_FORMAT_R8G8B8A8_UNORM", Value: dxgiFormatRGBA8},
			{Name: "DXGI_FORMAT_B8G8R8A8_UNORM", Value: dxgiFormatBGRA8},
			{Name: "DXGI_FORMAT_R8_UNORM", Value: dxgiFormatR8},
			{Name: "DXGI_ADAPTER_FLAG_SOFTWARE", Value: dxgiAdapterSoftware},
			{Name: "DXGI_CREATE_FACTORY_DEBUG", Value: dxgiCreateFactoryDebug},
			{Name: "DXGI_GPU_PREFERENCE_HIGH_PERFORMANCE", Value: dxgiGPUPreferenceHighPerf},
			{Name: "D3D_FEATURE_LEVEL_11_0", Value: featureLevel11_0},
			{Name: "D3D_FEATURE_LEVEL_11_1", Value: featureLevel11_1},
			{Name: "D3D_FEATURE_LEVEL_12_0", Value: featureLevel12_0},
			{Name: "D3D_FEATURE_LEVEL_12_1", Value: featureLevel12_1},
			{Name: "D3D_FEATURE_LEVEL_12_2", Value: featureLevel12_2},
			{Name: "D3D12_FEATURE_D3D12_OPTIONS", Value: featureD3D12Options},
			{Name: "D3D12_FEATURE_FEATURE_LEVELS", Value: featureFeatureLevels},
			{Name: "D3D12_COMMAND_LIST_TYPE_DIRECT", Value: commandListDirect},
			{Name: "D3D12_HEAP_TYPE_DEFAULT", Value: heapTypeDefault},
			{Name: "D3D12_HEAP_TYPE_UPLOAD", Value: heapTypeUpload},
			{Name: "D3D12_HEAP_TYPE_READBACK", Value: heapTypeReadback},
			{Name: "D3D12_RESOURCE_DIMENSION_BUFFER", Value: resourceDimensionBuffer},
			{Name: "D3D12_RESOURCE_DIMENSION_TEXTURE2D", Value: resourceDimensionTexture2D},
			{Name: "D3D12_TEXTURE_LAYOUT_UNKNOWN", Value: textureLayoutUnknown},
			{Name: "D3D12_TEXTURE_LAYOUT_ROW_MAJOR", Value: textureLayoutRowMajor},
			{Name: "D3D12_RESOURCE_FLAG_ALLOW_RENDER_TARGET", Value: resourceFlagAllowRT},
			{Name: "D3D12_RESOURCE_STATE_COMMON", Value: resourceStateCommon},
			{Name: "D3D12_RESOURCE_STATE_RENDER_TARGET", Value: resourceStateRenderTarget},
			{Name: "D3D12_RESOURCE_STATE_PIXEL_SHADER_RESOURCE", Value: resourceStatePixelShaderRS},
			{Name: "D3D12_RESOURCE_STATE_COPY_DEST", Value: resourceStateCopyDest},
			{Name: "D3D12_RESOURCE_STATE_COPY_SOURCE", Value: resourceStateCopySource},
			{Name: "D3D12_RESOURCE_STATE_GENERIC_READ", Value: resourceStateGenericRead},
			{Name: "D3D12_RESOURCE_BARRIER_TYPE_TRANSITION", Value: barrierTypeTransition},
			{Name: "D3D12_RESOURCE_BARRIER_ALL_SUBRESOURCES", Value: allSubresources},
			{Name: "D3D12_TEXTURE_COPY_TYPE_SUBRESOURCE_INDEX", Value: textureCopySubresourceIndex},
			{Name: "D3D12_TEXTURE_COPY_TYPE_PLACED_FOOTPRINT", Value: textureCopyPlacedFootprint},
			{Name: "D3D12_DESCRIPTOR_HEAP_TYPE_CBV_SRV_UAV", Value: descriptorHeapSRV},
			{Name: "D3D12_DESCRIPTOR_HEAP_TYPE_SAMPLER", Value: descriptorHeapSampler},
			{Name: "D3D12_DESCRIPTOR_HEAP_TYPE_RTV", Value: descriptorHeapRTV},
			{Name: "D3D12_SRV_DIMENSION_TEXTURE2D", Value: srvDimensionTexture2D},
			{Name: "D3D12_RTV_DIMENSION_TEXTURE2D", Value: rtvDimensionTexture2D},
			{Name: "D3D12_DEFAULT_SHADER_4_COMPONENT_MAPPING", Value: defaultComponentMapping},
			{Name: "D3D12_FILTER_MIN_MAG_MIP_POINT", Value: filterMinMagMipPoint},
			{Name: "D3D12_FILTER_MIN_MAG_POINT_MIP_LINEAR", Value: filterMipLinear},
			{Name: "D3D12_FILTER_MIN_POINT_MAG_LINEAR_MIP_POINT", Value: filterMagLinear},
			{Name: "D3D12_FILTER_MIN_LINEAR_MAG_MIP_POINT", Value: filterMinLinear},
			{Name: "D3D12_FILTER_MIN_MAG_MIP_LINEAR", Value: filterMinMagMipLinear},
			{Name: "D3D12_TEXTURE_ADDRESS_MODE_WRAP", Value: addressModeWrap},
			{Name: "D3D12_TEXTURE_ADDRESS_MODE_MIRROR", Value: addressModeMirror},
			{Name: "D3D12_TEXTURE_ADDRESS_MODE_CLAMP", Value: addressModeClamp},
			{Name: "D3D12_COMPARISON_FUNC_NEVER", Value: comparisonNever},
			{Name: "D3D12_TEXTURE_DATA_PITCH_ALIGNMENT", Value: textureDataPitchAlignment},
			{Name: "D3D12_TEXTURE_DATA_PLACEMENT_ALIGNMENT", Value: textureDataPlacementAlignment},
			{Name: "DXGI_USAGE_RENDER_TARGET_OUTPUT", Value: dxgiUsageRenderTargetOutput},
			{Name: "DXGI_SCALING_NONE", Value: dxgiScalingNone},
			{Name: "DXGI_SWAP_EFFECT_FLIP_DISCARD", Value: dxgiSwapEffectFlipDiscard},
			{Name: "DXGI_ALPHA_MODE_IGNORE", Value: dxgiAlphaModeIgnore},
			{Name: "DXGI_MWA_NO_WINDOW_CHANGES", Value: dxgiMWANoWindowChanges},
			{Name: "DXGI_MWA_NO_ALT_ENTER", Value: dxgiMWANoAltEnter},
			{Name: "DXGI_STATUS_OCCLUDED", Value: dxgiStatusOccluded},
			{Name: "DXGI_ERROR_DRIVER_INTERNAL_ERROR", Value: dxgiErrorDriverInternalError},
			{Name: "DXGI_ERROR_NOT_FOUND", Value: dxgiErrorNotFound},
			{Name: "DXGI_ERROR_SDK_COMPONENT_MISSING", Value: dxgiErrorSDKComponentMissing},
			{Name: "D3D12_MESSAGE_SEVERITY_CORRUPTION", Value: messageSeverityCorruption},
			{Name: "D3D12_MESSAGE_SEVERITY_ERROR", Value: messageSeverityError},
			{Name: "D3D12_MESSAGE_SEVERITY_WARNING", Value: messageSeverityWarning},
			{Name: "D3D12_RESOURCE_BINDING_TIER_1", Value: resourceBindingTier1},
			{Name: "D3D12_ROOT_PARAMETER_TYPE_DESCRIPTOR_TABLE", Value: rootParameterTable},
			{Name: "D3D12_ROOT_PARAMETER_TYPE_CBV", Value: rootParameterCBV},
			{Name: "D3D12_DESCRIPTOR_RANGE_TYPE_SRV", Value: descriptorRangeSRV},
			{Name: "D3D12_DESCRIPTOR_RANGE_TYPE_SAMPLER", Value: descriptorRangeSampler},
			{Name: "D3D12_SHADER_VISIBILITY_ALL", Value: shaderVisibilityAll},
			{Name: "D3D12_SHADER_VISIBILITY_VERTEX", Value: shaderVisibilityVertex},
			{Name: "D3D12_SHADER_VISIBILITY_PIXEL", Value: shaderVisibilityPixel},
			{Name: "D3D_ROOT_SIGNATURE_VERSION_1", Value: rootSignatureVersion1},
			{Name: "D3D12_ROOT_SIGNATURE_FLAG_ALLOW_INPUT_ASSEMBLER_INPUT_LAYOUT", Value: rootFlagAllowInputLayout},
			{Name: "D3D12_ROOT_SIGNATURE_FLAG_DENY_HULL_SHADER_ROOT_ACCESS", Value: rootFlagDenyHull},
			{Name: "D3D12_ROOT_SIGNATURE_FLAG_DENY_DOMAIN_SHADER_ROOT_ACCESS", Value: rootFlagDenyDomain},
			{Name: "D3D12_ROOT_SIGNATURE_FLAG_DENY_GEOMETRY_SHADER_ROOT_ACCESS", Value: rootFlagDenyGeometry},
			{Name: "D3D12_DESCRIPTOR_RANGE_OFFSET_APPEND", Value: descriptorRangeAppend},
			{Name: "D3D12_DESCRIPTOR_HEAP_FLAG_SHADER_VISIBLE", Value: descriptorHeapShaderVisible},
			{Name: "D3D12_BLEND_ZERO", Value: blendZero},
			{Name: "D3D12_BLEND_ONE", Value: blendOne},
			{Name: "D3D12_BLEND_SRC_ALPHA", Value: blendSrcAlpha},
			{Name: "D3D12_BLEND_INV_SRC_ALPHA", Value: blendInvSrcAlpha},
			{Name: "D3D12_BLEND_DEST_ALPHA", Value: blendDestAlpha},
			{Name: "D3D12_BLEND_INV_DEST_ALPHA", Value: blendInvDestAlpha},
			{Name: "D3D12_BLEND_OP_ADD", Value: blendOpAdd},
			{Name: "D3D12_COLOR_WRITE_ENABLE_ALL", Value: uint32(colorWriteEnableAll)},
			{Name: "D3D12_LOGIC_OP_NOOP", Value: logicOpNoop},
			{Name: "D3D12_FILL_MODE_SOLID", Value: fillModeSolid},
			{Name: "D3D12_CULL_MODE_NONE", Value: cullModeNone},
			{Name: "D3D12_COMPARISON_FUNC_ALWAYS", Value: comparisonAlways},
			{Name: "D3D12_STENCIL_OP_KEEP", Value: stencilOpKeep},
			{Name: "D3D12_DEPTH_WRITE_MASK_ZERO", Value: depthWriteMaskZero},
			{Name: "D3D12_DEFAULT_SAMPLE_MASK", Value: defaultSampleMask},
			{Name: "D3D12_PRIMITIVE_TOPOLOGY_TYPE_TRIANGLE", Value: topologyTypeTriangle},
			{Name: "D3D_PRIMITIVE_TOPOLOGY_TRIANGLELIST", Value: topologyTriangleList},
			{Name: "D3D12_INPUT_CLASSIFICATION_PER_VERTEX_DATA", Value: inputPerVertex},
			{Name: "D3D12_INPUT_CLASSIFICATION_PER_INSTANCE_DATA", Value: inputPerInstance},
			{Name: "DXGI_FORMAT_R32_FLOAT", Value: dxgiFormatR32},
			{Name: "DXGI_FORMAT_R32G32_FLOAT", Value: dxgiFormatRG32},
			{Name: "DXGI_FORMAT_R32G32B32_FLOAT", Value: dxgiFormatRGB32},
			{Name: "DXGI_FORMAT_R32G32B32A32_FLOAT", Value: dxgiFormatRGBA32},
			{Name: "DXGI_FORMAT_R16_UINT", Value: dxgiFormatR16Uint},
			{Name: "D3D12_ERROR_ADAPTER_NOT_FOUND", Value: d3d12ErrorAdapterNotFound},
			{Name: "D3D12_ERROR_DRIVER_VERSION_MISMATCH", Value: d3d12ErrorDriverVersionMismatch},
		},
	})
}

// TestIIDsAgainstTheHeaders compares the interface IDs of vtables.go with the __CRT_UUID_DECL lines of
// the mingw-w64 headers. The layout test cannot do it, since a GUID is not a constant of C.
func TestIIDsAgainstTheHeaders(t *testing.T) {
	dir := "/usr/x86_64-w64-mingw32/sys-root/mingw/include"
	var text strings.Builder
	for _, h := range []string{"d3d12.h", "d3d12sdklayers.h", "dxgi.h", "dxgi1_2.h", "dxgi1_4.h", "dxgi1_6.h"} {
		b, err := os.ReadFile(filepath.Join(dir, h))
		if err != nil {
			t.Skipf("the mingw-w64 headers are not installed (%v): %s", err, layouttest.InstallCommand)
		}
		text.Write(b)
	}
	re := regexp.MustCompile(`__CRT_UUID_DECL\((\w+),\s*0x([0-9a-fA-F]+),\s*0x([0-9a-fA-F]+),\s*0x([0-9a-fA-F]+),\s*((?:0x[0-9a-fA-F]+,?\s*){8})\)`)
	headers := map[string]win32.GUID{}
	for _, m := range re.FindAllStringSubmatch(text.String(), -1) {
		d1, _ := strconv.ParseUint(m[2], 16, 32)
		d2, _ := strconv.ParseUint(m[3], 16, 16)
		d3, _ := strconv.ParseUint(m[4], 16, 16)
		g := win32.GUID{Data1: uint32(d1), Data2: uint16(d2), Data3: uint16(d3)}
		for i, b := range strings.FieldsFunc(m[5], func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == '\t' }) {
			v, _ := strconv.ParseUint(strings.TrimPrefix(b, "0x"), 16, 8)
			g.Data4[i] = byte(v)
		}
		headers[m[1]] = g
	}
	for name, got := range namedIIDs() {
		t.Run(name, func(t *testing.T) {
			want, ok := headers[name]
			if !ok {
				t.Fatalf("the headers have no IID for %s", name)
			}
			if got != want {
				t.Errorf("IID of %s = %v, the header says %v", name, got, want)
			}
		})
	}
}

// namedIIDs lists the interface IDs with the names of their types in the headers, for the test that
// compares them.
func namedIIDs() map[string]win32.GUID {
	return map[string]win32.GUID{
		"IDXGIFactory1": iidDXGIFactory1, "IDXGIFactory2": iidDXGIFactory2, "IDXGISwapChain3": iidDXGISwapChain3, "IDXGIFactory4": iidDXGIFactory4, "IDXGIFactory6": iidDXGIFactory6,
		"IDXGIAdapter1": iidDXGIAdapter1, "ID3D12Device": iidDevice, "ID3D12CommandQueue": iidCommandQueue,
		"ID3D12CommandAllocator": iidCommandAllocator, "ID3D12GraphicsCommandList": iidGraphicsCommandList,
		"ID3D12Fence": iidFence, "ID3D12Resource": iidResource, "ID3D12DescriptorHeap": iidDescriptorHeap,
		"ID3D12Debug": iidDebug, "ID3D12InfoQueue": iidInfoQueue,
		"ID3D12RootSignature": iidRootSignature, "ID3D12PipelineState": iidPipelineState, "ID3D12Device1": iidDevice1,
		"ID3D12PipelineLibrary": iidPipelineLibrary, "IDXGIDevice": iidDXGIDevice,
	}
}
