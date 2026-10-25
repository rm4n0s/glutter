package d3d12

import "github.com/rm4n0s/glutter/internal/platform/windows/win32"

// The positions of the methods that the device calls in the vtables of the interfaces of DXGI and
// Direct3D 12 (SPEC §25.2), and the interface IDs. Every interface starts with the three methods of
// IUnknown, and an interface that derives from another has the methods of its base first. The layout
// test (layout_test.go) checks each index and each IID against the mingw-w64 headers.

// The methods of IDXGIFactory1, 4 and 6 and IDXGIAdapter1.
const (
	factoryEnumAdapters1              = 12 // IDXGIFactory1::EnumAdapters1
	factoryEnumWarpAdapter            = 27 // IDXGIFactory4::EnumWarpAdapter
	factoryEnumAdapterByGPUPreference = 29 // IDXGIFactory6::EnumAdapterByGpuPreference
	adapterGetDesc1                   = 10 // IDXGIAdapter1::GetDesc1
	adapterCheckInterfaceSupport      = 9  // IDXGIAdapter::CheckInterfaceSupport
)

// The methods of IDXGIFactory2 and of the swap chain interfaces IDXGISwapChain, 1 and 3 (SPEC §9.8
// "Swap chain").
const (
	factoryMakeWindowAssociation       = 8  // IDXGIFactory::MakeWindowAssociation
	factoryCreateSwapChainForHwnd      = 15 // IDXGIFactory2::CreateSwapChainForHwnd
	swapChainGetBuffer                 = 9  // IDXGISwapChain::GetBuffer
	swapChainResizeBuffers             = 13 // IDXGISwapChain::ResizeBuffers
	swapChainGetDesc1                  = 18 // IDXGISwapChain1::GetDesc1
	swapChainPresent1                  = 22 // IDXGISwapChain1::Present1
	swapChainGetCurrentBackBufferIndex = 36 // IDXGISwapChain3::GetCurrentBackBufferIndex
)

// The methods of ID3D12Device.
const (
	deviceCreateCommandQueue         = 8
	deviceCreateCommandAllocator     = 9
	deviceCreateGraphicsPipeline     = 10 // CreateGraphicsPipelineState
	deviceCreateCommandList          = 12
	deviceCheckFeatureSupport        = 13
	deviceCreateDescriptorHeap       = 14
	deviceGetDescriptorIncrementSize = 15 // GetDescriptorHandleIncrementSize
	deviceCreateRootSignature        = 16
	deviceCopyDescriptorsSimple      = 24
	deviceCreateShaderResourceView   = 18
	deviceCreateRenderTargetView     = 20
	deviceCreateSampler              = 22
	deviceCreateCommittedResource    = 27
	deviceCreateFence                = 36
	deviceGetDeviceRemovedReason     = 37
	deviceGetCopyableFootprints      = 38
)

// The methods of ID3D12Device1 and ID3D12PipelineLibrary.
const (
	device1CreatePipelineLibrary = 44
	libraryStorePipeline         = 8
	libraryLoadGraphicsPipeline  = 9
	librarySerializedSize        = 11 // GetSerializedSize
	librarySerialize             = 12
)

// The methods of ID3D12CommandQueue, ID3D12CommandAllocator, ID3D12Fence, ID3D12Resource and
// ID3D12DescriptorHeap.
const (
	queueExecuteCommandLists = 10
	queueSignal              = 14
	allocatorReset           = 8
	fenceGetCompletedValue   = 8
	fenceSetEventOnComplete  = 9 // SetEventOnCompletion
	resourceMap              = 8
	resourceUnmap            = 9
	resourceGPUAddress       = 11 // GetGPUVirtualAddress
	heapGetCPUHandleStart    = 9  // GetCPUDescriptorHandleForHeapStart
	heapGetGPUHandleStart    = 10 // GetGPUDescriptorHandleForHeapStart
)

// The methods of ID3D12GraphicsCommandList.
const (
	listClose                 = 9
	listReset                 = 10
	listDrawInstanced         = 12
	listDrawIndexedInstanced  = 13
	listCopyBufferRegion      = 15
	listCopyTextureRegion     = 16
	listIASetPrimitiveTopo    = 20 // IASetPrimitiveTopology
	listRSSetViewports        = 21
	listRSSetScissorRects     = 22
	listSetPipelineState      = 25
	listResourceBarrier       = 26
	listSetDescriptorHeaps    = 28
	listSetRootSignature      = 30 // SetGraphicsRootSignature
	listSetRootTable          = 32 // SetGraphicsRootDescriptorTable
	listSetRootCBV            = 38 // SetGraphicsRootConstantBufferView
	listIASetIndexBuffer      = 43
	listIASetVertexBuffers    = 44
	listOMSetRenderTargets    = 46
	listClearRenderTargetView = 48
	listDiscardResource       = 51
)

// The methods of ID3D12Debug and ID3D12InfoQueue.
const (
	debugEnableDebugLayer         = 3
	infoQueueSetMessageCountLimit = 3
	infoQueueClearStoredMessages  = 4
	infoQueueGetMessage           = 5
	infoQueueGetNumStoredMessages = 8
)

// The interface IDs.
var (
	iidDXGIFactory1        = win32.MustParseGUID("{770aae78-f26f-4dba-a829-253c83d1b387}")
	iidDXGIFactory2        = win32.MustParseGUID("{50c83a1c-e072-4c48-87b0-3630fa36a6d0}")
	iidDXGISwapChain3      = win32.MustParseGUID("{94d99bdb-f1f8-4ab0-b236-7da0170edab1}")
	iidDXGIFactory4        = win32.MustParseGUID("{1bc6ea02-ef36-464f-bf0c-21ca39e5168a}")
	iidDXGIFactory6        = win32.MustParseGUID("{c1b6694f-ff09-44a9-b03c-77900a0a1d17}")
	iidDXGIAdapter1        = win32.MustParseGUID("{29038f61-3839-4626-91fd-086879011a05}")
	iidDevice              = win32.MustParseGUID("{189819f1-1db6-4b57-be54-1821339b85f7}")
	iidCommandQueue        = win32.MustParseGUID("{0ec870a6-5d7e-4c22-8cfc-5baae07616ed}")
	iidCommandAllocator    = win32.MustParseGUID("{6102dee4-af59-4b09-b999-b44d73f09b24}")
	iidGraphicsCommandList = win32.MustParseGUID("{5b160d0f-ac1b-4185-8ba8-b3ae42a5a455}")
	iidFence               = win32.MustParseGUID("{0a753dcf-c4d8-4b91-adf6-be5a60d95a76}")
	iidResource            = win32.MustParseGUID("{696442be-a72e-4059-bc79-5b5c98040fad}")
	iidDescriptorHeap      = win32.MustParseGUID("{8efb471d-616c-4f49-90f7-127bb763fa51}")
	iidDebug               = win32.MustParseGUID("{344488b7-6846-474b-b989-f027448245e0}")
	iidInfoQueue           = win32.MustParseGUID("{0742a90b-c387-483f-b946-30a7e4e61458}")
	iidRootSignature       = win32.MustParseGUID("{c54a6b66-72df-4ee8-8be5-a946a1429214}")
	iidPipelineState       = win32.MustParseGUID("{765a30f3-f624-4c6f-a828-ace948622445}")
	iidDevice1             = win32.MustParseGUID("{77acce80-638e-4e65-8895-c1f23386863e}")
	iidPipelineLibrary     = win32.MustParseGUID("{c64226a8-9201-46af-b4cc-53fb9ff7414f}")
	iidDXGIDevice          = win32.MustParseGUID("{54ec77fa-1377-44e6-8c32-88fd5f44c84c}")
)
