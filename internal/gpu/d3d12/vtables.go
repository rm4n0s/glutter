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
)

// The methods of ID3D12Device.
const (
	deviceCreateCommandQueue         = 8
	deviceCreateCommandAllocator     = 9
	deviceCreateCommandList          = 12
	deviceCheckFeatureSupport        = 13
	deviceCreateDescriptorHeap       = 14
	deviceGetDescriptorIncrementSize = 15 // GetDescriptorHandleIncrementSize
	deviceCreateShaderResourceView   = 18
	deviceCreateRenderTargetView     = 20
	deviceCreateSampler              = 22
	deviceCreateCommittedResource    = 27
	deviceCreateFence                = 36
	deviceGetDeviceRemovedReason     = 37
	deviceGetCopyableFootprints      = 38
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
	heapGetCPUHandleStart    = 9 // GetCPUDescriptorHandleForHeapStart
)

// The methods of ID3D12GraphicsCommandList.
const (
	listClose                 = 9
	listReset                 = 10
	listCopyBufferRegion      = 15
	listCopyTextureRegion     = 16
	listResourceBarrier       = 26
	listClearRenderTargetView = 48
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
)
