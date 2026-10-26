//go:build windows

package d3d12

import (
	"image"
	"runtime"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// swapBufferCount is the number of buffers of the swap chain: two, the least a flip-model swap chain
// has (SPEC §9.8). At most two frames are in flight (§9.3) and the present never blocks.
const swapBufferCount = 2

// swapchain is the flip-model swap chain of a window (SPEC §9.8 "Swap chain"): two buffers that the
// renderer draws the frame into with its blit, and presents with dirty rectangles and a sync interval
// of 0. The buffers are textures of the device, in the state COMMON (which is PRESENT) between frames.
type swapchain struct {
	d      *Device
	hwnd   uintptr
	sc     win32.Object // IDXGISwapChain1, the one CreateSwapChainForHwnd made
	sc3    win32.Object // IDXGISwapChain3, for the index of the current buffer
	format swapFormat
	// width and height are the size of the buffers, 0 while there are none (a resize that failed).
	width, height int
	bufs          [swapBufferCount]*texture
	destroyed     bool
}

var _ explicit.Swapchain = (*swapchain)(nil)

// NewSwapchain implements explicit.Device: IDXGIFactory2::CreateSwapChainForHwnd on the command queue
// for the window of the handle, flip model with two buffers, in the first of the formats that DXGI
// accepts (swapFormats), then MakeWindowAssociation so that DXGI leaves Alt+Enter and the style of the
// window alone (SPEC §9.8). The buffers have the size of the window's client area. It fails with
// D3D12SwapchainFailed when the handle is not a window of Windows or no format is accepted, and with
// GPUDeviceLost or GPUOutOfMemory when DXGI says so.
func (d *Device) NewSwapchain(h gpu.SurfaceHandle) (explicit.Swapchain, error) {
	if h.Kind != gpu.SurfaceKindWin32 || h.Window == 0 || !win32.IsWindow(h.Window) {
		return nil, errors.New(TagD3D12SwapchainFailed, "the surface is not a window of Windows", "kind", string(h.Kind), "window", h.Window)
	}
	// A window that is minimized has an empty client area, which CreateSwapChainForHwnd refuses with
	// zero for a size: start with one pixel and let Configure give the real size.
	w, hgt := int32(1), int32(1)
	if rc, err := win32.GetClientRect(h.Window); err == nil {
		w, hgt = max(rc.Right-rc.Left, 1), max(rc.Bottom-rc.Top, 1)
	}
	f2, err := d.factory.QueryInterface(&iidDXGIFactory2)
	if err != nil {
		return nil, err
	}
	defer f2.Release()
	var out uintptr
	format, refused, last, ok := chooseSwapFormat(d.formats(), func(f swapFormat) win32.HRESULT {
		desc := dxgiSwapChainDesc1{
			Width: uint32(w), Height: uint32(hgt), Format: f.DXGI, SampleDesc: dxgiSampleDesc{Count: 1},
			BufferUsage: dxgiUsageRenderTargetOutput, BufferCount: swapBufferCount, Scaling: dxgiScalingNone,
			SwapEffect: dxgiSwapEffectFlipDiscard, AlphaMode: dxgiAlphaModeIgnore,
		}
		out = 0
		return win32.FromResult(f2.Call(factoryCreateSwapChainForHwnd, d.queue.Ptr(), h.Window, uintptr(unsafe.Pointer(&desc)), 0, 0, uintptr(unsafe.Pointer(&out))))
	})
	if !ok {
		if isDeviceFailure(last) {
			return nil, d.annotate(check("IDXGIFactory2::CreateSwapChainForHwnd", last))
		}
		return nil, errors.New(TagD3D12SwapchainFailed, "DXGI accepted no format for the swap chain of the window",
			"window", h.Window, "width", int(w), "height", int(hgt), "refused", refused)
	}
	sc := d.own(win32.ObjectFromPtr(out))
	sc3, err := sc.QueryInterface(&iidDXGISwapChain3)
	if err != nil {
		d.disown(sc)
		return nil, err
	}
	s := &swapchain{d: d, hwnd: h.Window, sc: sc, sc3: d.own(sc3), format: format}
	if err := d.hr(d.factory, "IDXGIFactory::MakeWindowAssociation", factoryMakeWindowAssociation, h.Window,
		uintptr(dxgiMWANoAltEnter|dxgiMWANoWindowChanges)); err != nil {
		// DXGI keeps its habits for this window (Alt+Enter changes it to full screen); the app still draws.
		d.log.Warn("the swap chain could not take the window away from DXGI's own handling", "err", err)
	}
	if err := s.makeBuffers(int(w), int(hgt)); err != nil {
		s.release()
		return nil, err
	}
	d.live[s] = func() { s.release() }
	return s, nil
}

// formats are the formats the swap chain is tried with: swapFormats, or the list a test gives.
func (d *Device) formats() []swapFormat {
	if d.swapFormats != nil {
		return d.swapFormats
	}
	return swapFormats
}

// makeBuffers wraps the buffers of the swap chain as textures of the device: a render target view for
// each, tracked in the state COMMON, which is the state a buffer is presented in.
func (s *swapchain) makeBuffers(width, height int) error {
	d := s.d
	for i := range s.bufs {
		var out uintptr
		if err := d.hr(s.sc, "IDXGISwapChain::GetBuffer", swapChainGetBuffer, uintptr(i),
			uintptr(unsafe.Pointer(&iidResource)), uintptr(unsafe.Pointer(&out))); err != nil {
			s.freeBuffers()
			return err
		}
		desc := explicit.TextureDesc{Width: width, Height: height, Format: s.format.Texture, MipLevels: 1, Usage: explicit.TextureRenderTarget}
		t := &texture{d: d, desc: desc, res: d.own(win32.ObjectFromPtr(out)), srv: -1, rtv: -1}
		t.root = t
		d.tracker.add(t, 1)
		d.counts.bytes += textureBytes(desc)
		if err := d.createViews(t); err != nil {
			t.release()
			s.freeBuffers()
			return err
		}
		s.bufs[i] = t
	}
	s.width, s.height = width, height
	return nil
}

// freeBuffers gives the buffers back: the references to them must be gone before ResizeBuffers.
func (s *swapchain) freeBuffers() {
	d := s.d
	for i, t := range s.bufs {
		if t == nil {
			continue
		}
		t.release()
		s.bufs[i] = nil
		for j, p := range d.toPresent {
			if p == t {
				d.toPresent = append(d.toPresent[:j], d.toPresent[j+1:]...)
				break
			}
		}
	}
	s.width, s.height = 0, 0
}

// Configure implements explicit.Swapchain: ResizeBuffers at the new size in device pixels, after the GPU
// is idle and the references to the old buffers are gone (a resize is not a hot path, SPEC §9.8). The
// size it has already, and a size of zero (a window that is minimized), do nothing.
func (s *swapchain) Configure(width, height int) error {
	if width <= 0 || height <= 0 || (width == s.width && height == s.height && s.bufs[0] != nil) {
		return nil
	}
	d := s.d
	if err := d.waitIdle(); err != nil {
		return err
	}
	s.freeBuffers()
	if err := d.hr(s.sc, "IDXGISwapChain::ResizeBuffers", swapChainResizeBuffers, 0, uintptr(width), uintptr(height), uintptr(dxgiFormatUnknown), 0); err != nil {
		return err
	}
	return s.makeBuffers(width, height)
}

// PreRotation implements explicit.Swapchain: Windows does not turn the image.
func (s *swapchain) PreRotation() explicit.Rotation { return explicit.Rotation0 }

// Stale implements explicit.Swapchain: the surface of a window does not turn.
func (s *swapchain) Stale() bool { return false }

// Acquire implements explicit.Swapchain: the current buffer (GetCurrentBackBufferIndex) in the state
// COMMON. It never reports an out-of-date swap chain. The frame's Submit puts the buffer back in
// COMMON after the pass that drew into it (a barrier before the command list is closed), which is
// what Present needs. It fails with D3D12SwapchainFailed when a resize left the swap chain without buffers.
func (s *swapchain) Acquire() (explicit.Texture, error) {
	if s.destroyed || s.bufs[0] == nil {
		return nil, errors.New(TagD3D12SwapchainFailed, "the swap chain has no buffers: a resize failed or it was destroyed", "window", s.hwnd)
	}
	d := s.d
	t := s.bufs[uint32(s.sc3.Call(swapChainGetCurrentBackBufferIndex))%swapBufferCount]
	for _, p := range d.toPresent {
		if p == t {
			return t, nil
		}
	}
	d.toPresent = append(d.toPresent, t)
	return t, nil
}

// Present implements explicit.Swapchain: Present1 with a sync interval of 0 and the damage as dirty
// rectangles (presentRects). DXGI_STATUS_OCCLUDED is a success. A removed or reset device is
// GPUDeviceLost, with the reason.
func (s *swapchain) Present(damage []image.Rectangle) error {
	rects := presentRects(damage, s.width, s.height)
	params := dxgiPresentParameters{DirtyRectsCount: uint32(len(rects))}
	if len(rects) > 0 {
		params.DirtyRects = unsafe.Pointer(&rects[0])
	}
	err := s.d.hr(s.sc, "IDXGISwapChain1::Present1", swapChainPresent1, 0, 0, uintptr(unsafe.Pointer(&params)))
	runtime.KeepAlive(rects)
	return err
}

// Destroy implements explicit.Swapchain: it waits for the GPU, gives the buffers back and releases the swap
// chain. It can be called twice.
func (s *swapchain) Destroy() {
	if s.destroyed {
		return
	}
	d := s.d
	if err := d.waitIdle(); err != nil {
		d.log.Warn("waiting for the GPU before destroying a swap chain failed", "err", err)
	}
	s.release()
}

// release gives the buffers and the swap chain back; the GPU is idle.
func (s *swapchain) release() {
	if s.destroyed {
		return
	}
	s.destroyed = true
	d := s.d
	delete(d.live, s)
	s.freeBuffers()
	d.disown(s.sc3)
	d.disown(s.sc)
}

// presentBarrier puts the swap chain buffers that were acquired for the frame back in the state COMMON,
// which Present needs, in the frame's command list, before it is closed.
func (e *encoder) presentBarrier() {
	d := e.d
	if len(d.toPresent) == 0 {
		return
	}
	for _, t := range d.toPresent {
		if err := d.tracker.need(t.root, t.base, stateCommon); err != nil {
			panic(err)
		}
	}
	d.toPresent = d.toPresent[:0]
	d.barrier(e.list)
}
