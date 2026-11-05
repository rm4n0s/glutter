//go:build windows && arm64

package d3d12

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// TagD3D12DepthClearUnsupported is the tag of the programmer error of a pass that clears a depth-stencil
// attachment on Windows for Arm: ClearDepthStencilView takes the depth as a float in a floating point
// register (v0), which the standard library's SyscallN does not set on arm64 (it copies the integer
// arguments into XMM registers on amd64 only), and Go assembly is not allowed here (CLAUDE.md). Load the
// depth attachment with LoadLoad, or draw a full-screen triangle to fill it.
const TagD3D12DepthClearUnsupported = "D3D12DepthClearUnsupported"

// clearDepthStencil panics: see TagD3D12DepthClearUnsupported.
func clearDepthStencil(_ win32.Object, _ uintptr, _ uint32, _ float32, _ uint8) {
	panic(errors.New(TagD3D12DepthClearUnsupported, "clearing a depth-stencil attachment is not supported on Windows for Arm"))
}
