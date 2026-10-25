package win32

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestHRESULT(t *testing.T) {
	tests := []struct {
		name       string
		hr         HRESULT
		code       uint32
		failed     bool
		wantString string
	}{
		{"S_OK", SOK, 0, false, "0x00000000"},
		{"S_FALSE", 1, 1, false, "0x00000001"},
		{"E_FAIL", EFail, 0x80004005, true, "0x80004005"},
		{"E_NOINTERFACE", ENoInterface, 0x80004002, true, "0x80004002"},
		{"E_OUTOFMEMORY", EOutOfMemory, 0x8007000E, true, "0x8007000E"},
		{"DXGI_ERROR_DEVICE_REMOVED", DXGIErrorDeviceRemoved, 0x887A0005, true, "0x887A0005"},
		{"from a register", FromResult(0x80004003), 0x80004003, true, "0x80004003"},
		{"from a register with the high half set", FromResult(0xFFFFFFFF80004003), 0x80004003, true, "0x80004003"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hr.Code() != tt.code || tt.hr.Failed() != tt.failed || tt.hr.Succeeded() == tt.failed || tt.hr.String() != tt.wantString {
				t.Errorf("code %#x failed %v string %q, want %#x %v %q", tt.hr.Code(), tt.hr.Failed(), tt.hr, tt.code, tt.failed, tt.wantString)
			}
		})
	}
}

// TestHResultError is the table of the codes that become tags.
func TestHResultError(t *testing.T) {
	tests := []struct {
		name    string
		hr      HRESULT
		wantTag string // empty: a success
	}{
		{"S_OK", SOK, ""},
		{"S_FALSE is a success", 1, ""},
		{"E_OUTOFMEMORY", EOutOfMemory, "GPUOutOfMemory"},
		{"DXGI_ERROR_DEVICE_REMOVED", DXGIErrorDeviceRemoved, "GPUDeviceLost"},
		{"DXGI_ERROR_DEVICE_RESET", DXGIErrorDeviceReset, "GPUDeviceLost"},
		{"DXGI_ERROR_DEVICE_HUNG", DXGIErrorDeviceHung, "GPUDeviceLost"},
		{"ERROR_CANCELLED", ErrorCancelled, "Win32Cancelled"},
		{"E_FAIL", EFail, "COMCallFailed"},
		{"E_NOINTERFACE", ENoInterface, "COMCallFailed"},
		{"an unknown failure", FromResult(0x80990001), "COMCallFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HResultError("IFoo::Bar", tt.hr)
			if tt.wantTag == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T: %v", err, err)
			}
			if appErr.Tag != tt.wantTag || !appErr.HasRoute("HResultError."+tt.wantTag) {
				t.Errorf("got %s via %s, want %s via HResultError", appErr.Tag, appErr.Route(), tt.wantTag)
			}
			if appErr.Metadata["method"] != "IFoo::Bar" || appErr.Metadata["hresult"] != tt.hr.String() {
				t.Errorf("metadata = %v", appErr.Metadata)
			}
		})
	}
}

func TestWin32Error(t *testing.T) {
	tests := []struct {
		name    string
		code    uint32
		wantTag string
	}{
		{"ERROR_ACCESS_DENIED", 5, "Win32CallFailed"},
		{"no error code", 0, "Win32CallFailed"},
		{"ERROR_CANCELLED", 1223, "Win32Cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr, ok := errors.FromError(Win32Error("CreateWindowExW", tt.code))
			if !ok {
				t.Fatal("expected *errors.Error")
			}
			if appErr.Tag != tt.wantTag || !appErr.HasRoute("Win32Error."+tt.wantTag) {
				t.Errorf("got %s via %s, want %s via Win32Error", appErr.Tag, appErr.Route(), tt.wantTag)
			}
			if appErr.Metadata["function"] != "CreateWindowExW" || appErr.Metadata["code"] != int(tt.code) {
				t.Errorf("metadata = %v", appErr.Metadata)
			}
		})
	}
}
