//go:build windows && amd64

package d3d12

import (
	"math"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// clearDepthStencil is ID3D12GraphicsCommandList::ClearDepthStencilView of the whole view. Its depth is
// the fourth argument, which the Windows x64 convention passes in XMM3; the runtime's call to the
// standard library's SyscallN copies each of the first four integer arguments into the XMM register of
// the same number too, so the bits of the float travel in the fourth one.
func clearDepthStencil(list win32.Object, view uintptr, flags uint32, depth float32, stencil uint8) {
	list.Call(listClearDepthStencilView, view, uintptr(flags), uintptr(math.Float32bits(depth)), uintptr(stencil), 0, 0)
}
