package d3d12

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// d3d12Error turns the HRESULT of a Direct3D 12 or DXGI method that failed into the tagged error of
// SPEC §9.8: GPUOutOfMemory for E_OUTOFMEMORY, GPUDeviceLost for DXGI_ERROR_DEVICE_REMOVED, _RESET,
// _HUNG and _DRIVER_INTERNAL_ERROR, and D3D12CallFailed for any other code; the metadata are "method"
// (such as "ID3D12Device::CreateFence") and "hresult". It is the one function that turns a code into a
// tag, like vkError, and is listed in errcheck.allow (SPEC §14.4, R4). A success gives nil.
func d3d12Error(method string, hr win32.HRESULT) error {
	if hr.Succeeded() {
		return nil
	}
	switch hr.Code() {
	case win32.EOutOfMemory.Code():
		return errors.New(tagGPUOutOfMemory, "the GPU or the system is out of memory", "method", method, "hresult", hr.String())
	case win32.DXGIErrorDeviceRemoved.Code(), win32.DXGIErrorDeviceReset.Code(), win32.DXGIErrorDeviceHung.Code(), dxgiErrorDriverInternalError:
		return errors.New(tagGPUDeviceLost, "the Direct3D 12 device was removed or reset", "method", method, "hresult", hr.String())
	}
	return errors.New(TagD3D12CallFailed, "a Direct3D 12 or DXGI method failed", "method", method, "hresult", hr.String())
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagD3D12CallFailed is the tag of the error for a method of Direct3D 12 or DXGI that returned a
	// failed HRESULT that has no tag of its own; the metadata are "method" and "hresult".
	TagD3D12CallFailed = "D3D12CallFailed"
	// TagD3D12SwapchainFailed is the tag of the error for a swap chain that cannot be made: the surface is not
	// a window of Windows, DXGI accepted no format, or a resize left it without buffers (SPEC §9.8); the
	// metadata are the "window" and, for a refused format, "refused" with each format and the answer.
	TagD3D12SwapchainFailed = "D3D12SwapchainFailed"
	// tagGPUOutOfMemory and tagGPUDeviceLost are the tags of SPEC §9.3, which every explicit backend
	// makes; explicit.Renderer recovers from the second (SPEC §9.3, "Decided in step 20.7").
	tagGPUOutOfMemory = "GPUOutOfMemory"
	tagGPUDeviceLost  = "GPUDeviceLost"
)
