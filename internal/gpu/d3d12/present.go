package d3d12

import (
	"image"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// The parts of the swap chain (SPEC §9.8 "Swap chain") that need no Windows: the formats it is tried
// with, the dirty rectangles of a present, and the rule for the device-class failures. swapchain_windows.go
// calls DXGI.

// swapFormat is a format the swap chain may have: the DXGI format and the texture format of the explicit
// renderer that its buffers are seen as.
type swapFormat struct {
	Name    string
	DXGI    uint32
	Texture explicit.TextureFormat
}

// swapFormats are the formats of the swap chain in the order they are tried (SPEC §9.8): B8G8R8A8_UNORM, the
// format that DWM prefers, then R8G8B8A8_UNORM when it is refused. An sRGB format is never used: the
// renderer blends in non-linear sRGB (§9.3).
var swapFormats = []swapFormat{
	{Name: "B8G8R8A8_UNORM", DXGI: dxgiFormatBGRA8, Texture: explicit.TextureFormatBGRA8Unorm},
	{Name: "R8G8B8A8_UNORM", DXGI: dxgiFormatRGBA8, Texture: explicit.TextureFormatRGBA8Unorm},
}

// isDeviceFailure says whether an HRESULT is about the device and not about the format that was tried:
// out of memory, or a device that was removed, reset or hung. Another format would not help, and the
// error is GPUOutOfMemory or GPUDeviceLost (the table of d3d12Error), not D3D12SwapchainFailed.
func isDeviceFailure(hr win32.HRESULT) bool {
	switch hr.Code() {
	case win32.EOutOfMemory.Code(), win32.DXGIErrorDeviceRemoved.Code(), win32.DXGIErrorDeviceReset.Code(),
		win32.DXGIErrorDeviceHung.Code(), dxgiErrorDriverInternalError:
		return true
	}
	return false
}

// chooseSwapFormat tries the formats in order (swapFormats, or the list a test gives) with attempt, which makes the
// swap chain with one and says what DXGI answered. The first success wins. It stops at a failure that isDeviceFailure and returns it
// as last; it also returns the formats that were refused, each with its answer, for the error that
// NewSwapchain makes when none works. ok is whether a format was accepted.
func chooseSwapFormat(formats []swapFormat, attempt func(f swapFormat) win32.HRESULT) (chosen swapFormat, refused []string, last win32.HRESULT, ok bool) {
	for _, f := range formats {
		hr := attempt(f)
		if hr.Succeeded() {
			return f, refused, hr, true
		}
		refused = append(refused, f.Name+": "+hr.String())
		last = hr
		if isDeviceFailure(hr) {
			return swapFormat{}, refused, hr, false
		}
	}
	return swapFormat{}, refused, last, false
}

// presentRects turns the damage of a frame (SPEC §9.1: rectangles from the top left of the window, nil for
// all of it) into the dirty rectangles of DXGI_PRESENT_PARAMETERS. nil stays nil: the whole window, which
// DXGI wants as a count of 0. Each rectangle is cut to the window and an empty one is dropped; damage that
// is left with nothing (empty, or outside the window) is a pixel at the top left, the smallest damage, as
// the renderer says "nothing changed" (explicit.Renderer.damage). A window with no area has no rectangles.
// The rectangles are hints for DWM: the frame is drawn whole either way.
func presentRects(damage []image.Rectangle, width, height int) []d3d12Rect {
	if damage == nil || width <= 0 || height <= 0 {
		return nil
	}
	bounds := image.Rect(0, 0, width, height)
	out := make([]d3d12Rect, 0, len(damage))
	for _, r := range damage {
		if r = r.Canon().Intersect(bounds); r.Empty() {
			continue
		}
		out = append(out, d3d12Rect{Left: int32(r.Min.X), Top: int32(r.Min.Y), Right: int32(r.Max.X), Bottom: int32(r.Max.Y)})
	}
	if len(out) == 0 {
		return []d3d12Rect{{Right: 1, Bottom: 1}}
	}
	return out
}
