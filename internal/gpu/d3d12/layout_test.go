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
			{CName: "IDXGIAdapter1Vtbl", Methods: []layouttest.Method{{Name: "GetDesc1", Index: adapterGetDesc1}}},
			{CName: "ID3D12DeviceVtbl", Methods: []layouttest.Method{
				{Name: "CreateCommandQueue", Index: deviceCreateCommandQueue},
				{Name: "CreateCommandAllocator", Index: deviceCreateCommandAllocator},
				{Name: "CreateCommandList", Index: deviceCreateCommandList},
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
			}},
			{CName: "ID3D12DescriptorHeapVtbl", Methods: []layouttest.Method{
				{Name: "GetCPUDescriptorHandleForHeapStart", Index: heapGetCPUHandleStart},
			}},
			{CName: "ID3D12GraphicsCommandListVtbl", Methods: []layouttest.Method{
				{Name: "Close", Index: listClose},
				{Name: "Reset", Index: listReset},
				{Name: "CopyBufferRegion", Index: listCopyBufferRegion},
				{Name: "CopyTextureRegion", Index: listCopyTextureRegion},
				{Name: "ResourceBarrier", Index: listResourceBarrier},
				{Name: "ClearRenderTargetView", Index: listClearRenderTargetView},
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
			{Name: "DXGI_ERROR_DRIVER_INTERNAL_ERROR", Value: dxgiErrorDriverInternalError},
			{Name: "DXGI_ERROR_NOT_FOUND", Value: dxgiErrorNotFound},
			{Name: "DXGI_ERROR_SDK_COMPONENT_MISSING", Value: dxgiErrorSDKComponentMissing},
			{Name: "D3D12_MESSAGE_SEVERITY_CORRUPTION", Value: messageSeverityCorruption},
			{Name: "D3D12_MESSAGE_SEVERITY_ERROR", Value: messageSeverityError},
			{Name: "D3D12_MESSAGE_SEVERITY_WARNING", Value: messageSeverityWarning},
			{Name: "D3D12_RESOURCE_BINDING_TIER_1", Value: resourceBindingTier1},
		},
	})
}

// TestIIDsAgainstTheHeaders compares the interface IDs of vtables.go with the __CRT_UUID_DECL lines of
// the mingw-w64 headers. The layout test cannot do it, since a GUID is not a constant of C.
func TestIIDsAgainstTheHeaders(t *testing.T) {
	dir := "/usr/x86_64-w64-mingw32/sys-root/mingw/include"
	var text strings.Builder
	for _, h := range []string{"d3d12.h", "d3d12sdklayers.h", "dxgi.h", "dxgi1_4.h", "dxgi1_6.h"} {
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
		"IDXGIFactory1": iidDXGIFactory1, "IDXGIFactory4": iidDXGIFactory4, "IDXGIFactory6": iidDXGIFactory6,
		"IDXGIAdapter1": iidDXGIAdapter1, "ID3D12Device": iidDevice, "ID3D12CommandQueue": iidCommandQueue,
		"ID3D12CommandAllocator": iidCommandAllocator, "ID3D12GraphicsCommandList": iidGraphicsCommandList,
		"ID3D12Fence": iidFence, "ID3D12Resource": iidResource, "ID3D12DescriptorHeap": iidDescriptorHeap,
		"ID3D12Debug": iidDebug, "ID3D12InfoQueue": iidInfoQueue,
	}
}
