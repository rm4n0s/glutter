package d3d12

import "github.com/rm4n0s/glutter/internal/platform/windows/win32"

// check returns nil for an HRESULT that succeeded (S_OK, and the positive status codes such as
// DXGI_STATUS_OCCLUDED) and d3d12Error otherwise. method names the Direct3D method.
func check(method string, hr win32.HRESULT) error {
	if hr.Succeeded() {
		return nil
	}
	return d3d12Error(method, hr)
}
