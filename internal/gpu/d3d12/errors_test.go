package d3d12

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

func TestD3D12ErrorTable(t *testing.T) {
	tests := []struct {
		name  string
		hr    win32.HRESULT
		tag   string
		route string
	}{
		{"out of memory", win32.EOutOfMemory, "GPUOutOfMemory", "check->d3d12Error.GPUOutOfMemory"},
		{"device removed", win32.DXGIErrorDeviceRemoved, "GPUDeviceLost", "check->d3d12Error.GPUDeviceLost"},
		{"device reset", win32.DXGIErrorDeviceReset, "GPUDeviceLost", "check->d3d12Error.GPUDeviceLost"},
		{"device hung", win32.DXGIErrorDeviceHung, "GPUDeviceLost", "check->d3d12Error.GPUDeviceLost"},
		{"driver internal error", win32.HRESULT(int32(-0x7785FFE0)), "GPUDeviceLost", "check->d3d12Error.GPUDeviceLost"},
		{"invalid argument", win32.EInvalidArg, "D3D12CallFailed", "check->d3d12Error.D3D12CallFailed"},
		{"fail", win32.EFail, "D3D12CallFailed", "check->d3d12Error.D3D12CallFailed"},
		{"not implemented", win32.ENotImpl, "D3D12CallFailed", "check->d3d12Error.D3D12CallFailed"},
		{"a code that has no name", win32.HRESULT(int32(-0x7785FFD3)), "D3D12CallFailed", "check->d3d12Error.D3D12CallFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := check("ID3D12Device::CreateFence", tt.hr)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tt.tag || !appErr.HasRoute(tt.route) {
				t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tt.tag, tt.route)
			}
			if appErr.Metadata["method"] != "ID3D12Device::CreateFence" || appErr.Metadata["hresult"] != tt.hr.String() {
				t.Errorf("metadata = %v", appErr.Metadata)
			}
		})
	}
}

func TestCheckLetsSuccessesThrough(t *testing.T) {
	for _, hr := range []win32.HRESULT{win32.SOK, 1 /* S_FALSE */, 0x087A0001 /* DXGI_STATUS_OCCLUDED */} {
		if err := check("IDXGISwapChain::Present", hr); err != nil {
			t.Errorf("%v gave %v", hr, err)
		}
	}
}
