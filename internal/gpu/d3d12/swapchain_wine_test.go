//go:build windows && glutter_wine

package d3d12

import (
	"encoding/binary"
	"fmt"
	"image"
	"log/slog"
	"runtime"
	"sync"
	"testing"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests make swap chains for real windows of the Wine prefix (vkd3d presents them with the host's
// Vulkan, SPEC §25.13). Wine's flip-model present is not Windows': what is checked is the use of the
// interfaces, the sizes, the states and the order of the calls; that DWM shows the frame, without
// tearing, is a manual check on a Windows PC (step 28.10).

var swapWindowClass sync.Once

// makeSwapWindow makes a visible popup window with a client area of the size, for the test that has
// called runtime.LockOSThread (openTestDevice does).
func makeSwapWindow(t *testing.T, w, h int32) gpu.SurfaceHandle {
	t.Helper()
	instance, err := win32.GetModuleHandle("")
	if err != nil {
		t.Fatal(err)
	}
	const class = "D3D12SwapchainTest"
	swapWindowClass.Do(func() {
		name, err := win32.UTF16PtrFromString(class)
		if err != nil {
			t.Error(err)
			return
		}
		proc := win32.NewCallback(func(hwnd, msg, wparam, lparam uintptr) uintptr {
			if uint32(msg) == win32.WMEraseBkgnd {
				return 1
			}
			return win32.DefWindowProc(hwnd, uint32(msg), wparam, lparam)
		})
		wc := win32.WndClassEx{WndProc: proc, Instance: instance, ClassName: name}
		wc.Size = 80
		if _, err := win32.RegisterClassEx(&wc); err != nil {
			t.Error(err)
		}
	})
	hwnd, err := win32.CreateWindowEx(0, class, "swap chain", win32.WSPopup|win32.WSVisible, 10, 10, w, h, 0, 0, instance, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = win32.DestroyWindow(hwnd) })
	return gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Display: instance, Window: hwnd}
}

// swapDesc reads the description DXGI has of the swap chain (GetDesc1).
func swapDesc(t *testing.T, s *swapchain) dxgiSwapChainDesc1 {
	t.Helper()
	var desc dxgiSwapChainDesc1
	if err := callHR(s.sc, "IDXGISwapChain1::GetDesc1", swapChainGetDesc1, uintptr(unsafe.Pointer(&desc))); err != nil {
		t.Fatal(err)
	}
	return desc
}

// clearAndPresent draws a clear pass into the buffer that Acquire gives, submits and presents.
func clearAndPresent(t *testing.T, d *Device, sc explicit.Swapchain, c ui.Color, damage []image.Rectangle) explicit.Texture {
	t.Helper()
	var tex explicit.Texture
	frame(t, d, func(e explicit.Encoder) {
		var err error
		if tex, err = sc.Acquire(); err != nil {
			t.Fatalf("Acquire: %+v", err)
		}
		p := e.BeginPass(tex, explicit.LoadClear, c)
		p.End()
	})
	if err := sc.Present(damage); err != nil {
		t.Fatalf("Present: %+v", err)
	}
	return tex
}

func TestWineSwapchainPresentsEachSizeOfAResize(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	h := makeSwapWindow(t, 200, 120)
	sc, err := d.NewSwapchain(h)
	if err != nil {
		t.Fatalf("NewSwapchain: %+v", err)
	}
	s := sc.(*swapchain)
	desc := swapDesc(t, s)
	t.Logf("swap chain: %dx%d format %d, %d buffers, effect %d, scaling %d, alpha %d", desc.Width, desc.Height, desc.Format, desc.BufferCount, desc.SwapEffect, desc.Scaling, desc.AlphaMode)
	if desc.Width != 200 || desc.Height != 120 || desc.Format != s.format.DXGI || desc.BufferCount != swapBufferCount ||
		desc.SwapEffect != dxgiSwapEffectFlipDiscard || desc.Scaling != dxgiScalingNone || desc.AlphaMode != dxgiAlphaModeIgnore {
		t.Errorf("DXGI made %+v, want 200x120 flip-discard, scaling none, 2 buffers in %s", desc, s.format.Name)
	}
	if got := sc.PreRotation(); got != explicit.Rotation0 || sc.Stale() {
		t.Errorf("PreRotation = %v, Stale = %v", got, sc.Stale())
	}
	// The sizes of the resize: the same, smaller, a thin one, larger than the window.
	sizes := []image.Point{{200, 120}, {333, 77}, {64, 64}, {500, 300}, {200, 120}}
	for _, size := range sizes {
		if err := sc.Configure(size.X, size.Y); err != nil {
			t.Fatalf("Configure(%v): %+v", size, err)
		}
		seen := map[uintptr]bool{}
		for i := 0; i < 3; i++ { // both buffers, and the first again
			tex := clearAndPresent(t, d, sc, ui.Color{R: 0.2, G: 0.4, B: 0.6, A: 1}, nil)
			if got := tex.Desc(); got.Width != size.X || got.Height != size.Y || got.Format != s.format.Texture {
				t.Errorf("size %v: the buffer is %dx%d %v", size, got.Width, got.Height, got.Format)
			}
			if st, _ := d.tracker.state(tex.(*texture), 0); st != stateCommon {
				t.Errorf("size %v: after the frame the buffer is in the state %v, want COMMON (PRESENT)", size, st)
			}
			seen[tex.(*texture).res.Ptr()] = true
		}
		if len(seen) != swapBufferCount {
			t.Errorf("size %v: %d different buffers in 3 frames, want %d", size, len(seen), swapBufferCount)
		}
		if got := swapDesc(t, s); int(got.Width) != size.X || int(got.Height) != size.Y {
			t.Errorf("size %v: DXGI says %dx%d", size, got.Width, got.Height)
		}
	}
	sc.Destroy()
	sc.Destroy() // twice is fine
	if err := d.Close(); err != nil || d.Leaks() != 0 {
		t.Errorf("Close: %v, leaks %d (stats %+v)", err, d.Leaks(), d.Stats())
	}
}

// swapProxy is an IDXGISwapChain3 made in Go over the real one: every method records its call and
// forwards it, except that a test can answer Present1 or ResizeBuffers itself, and hook into
// ResizeBuffers to look at the device at that moment.
type swapProxy struct {
	real  win32.Object
	mu    sync.Mutex
	calls []recordedCall
	// onResize runs inside ResizeBuffers, before the real call.
	onResize func(args []uintptr)
	// presentAnswer and resizeAnswer, when set, are what Present1 and ResizeBuffers return instead of the real call.
	presentAnswer, resizeAnswer *win32.HRESULT
	// rects are the dirty rectangles of each Present1 call, nil for none; presents counts the calls.
	rects    [][]d3d12Rect
	presents int
}

var (
	swapProxyOnce sync.Once
	swapProxyHost *win32.Host
	swapProxyType *win32.InterfaceType
	swapProxyErr  error
)

// swapMethods is how many methods IDXGISwapChain3 has after IUnknown (index 3 to 36).
const swapMethods = 34

// makeSwapProxy puts a proxy over the swap chain object of the swapchain and returns a function that puts
// the real one back, which the test calls before it destroys the swap chain.
func makeSwapProxy(t *testing.T, s *swapchain) (*swapProxy, func()) {
	t.Helper()
	swapProxyOnce.Do(func() {
		swapProxyHost, swapProxyErr = win32.NewHost(nil)
		if swapProxyErr != nil {
			return
		}
		methods := make([]win32.MethodDef, swapMethods)
		for i := range methods {
			index := i + 3
			methods[i] = win32.MethodDef{Name: fmt.Sprintf("swap%d", index), Args: 6, Fn: func(impl any, a []uintptr) uintptr {
				p := impl.(*swapProxy)
				p.mu.Lock()
				c := recordedCall{index: index}
				copy(c.args[:], a)
				p.calls = append(p.calls, c)
				p.mu.Unlock()
				switch index {
				case swapChainResizeBuffers:
					if p.onResize != nil {
						p.onResize(a)
					}
					if p.resizeAnswer != nil {
						return win32.Result(*p.resizeAnswer)
					}
				case swapChainPresent1:
					p.recordPresent(a)
					if p.presentAnswer != nil {
						return win32.Result(*p.presentAnswer)
					}
				}
				return p.real.Call(index, a[0], a[1], a[2], a[3], a[4], a[5])
			}}
		}
		swapProxyType, swapProxyErr = swapProxyHost.NewInterface(win32.InterfaceDef{Name: "IDXGISwapChain3", IID: iidDXGISwapChain3, Methods: methods})
	})
	if swapProxyErr != nil {
		t.Fatal(swapProxyErr)
	}
	p := &swapProxy{real: s.sc}
	obj, err := swapProxyHost.NewObject(p, swapProxyType)
	if err != nil {
		t.Fatal(err)
	}
	real := s.sc
	s.sc = obj
	return p, func() { s.sc = real; obj.Release() }
}

// recordPresent reads the dirty rectangles of the DXGI_PRESENT_PARAMETERS that Present1 got (a[2]).
func (p *swapProxy) recordPresent(a []uintptr) {
	raw := win32.Bytes(a[2], int(unsafe.Sizeof(dxgiPresentParameters{})))
	count := int(binary.LittleEndian.Uint32(raw[0:]))
	var rects []d3d12Rect
	if count > 0 {
		rb := win32.Bytes(uintptr(binary.LittleEndian.Uint64(raw[8:])), count*16)
		for i := 0; i < count; i++ {
			v := func(k int) int32 { return int32(binary.LittleEndian.Uint32(rb[i*16+4*k:])) }
			rects = append(rects, d3d12Rect{Left: v(0), Top: v(1), Right: v(2), Bottom: v(3)})
		}
	}
	p.mu.Lock()
	p.rects = append(p.rects, rects)
	p.presents++
	p.mu.Unlock()
}

// count is how many times the method at index was called.
func (p *swapProxy) count(index int) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, c := range p.calls {
		if c.index == index {
			n++
		}
	}
	return n
}

// newTestSwapchain makes a swap chain for a new window.
func newTestSwapchain(t *testing.T, d *Device, w, h int32) (*swapchain, gpu.SurfaceHandle) {
	t.Helper()
	handle := makeSwapWindow(t, w, h)
	sc, err := d.NewSwapchain(handle)
	if err != nil {
		t.Fatalf("NewSwapchain: %+v", err)
	}
	return sc.(*swapchain), handle
}

func TestWineSwapchainResizeWaitsForTheGPUAndReleasesTheBuffers(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	s, _ := newTestSwapchain(t, d, 200, 120)
	proxy, restore := makeSwapProxy(t, s)
	defer func() { restore(); s.Destroy() }()

	// A frame is in flight when the resize comes.
	clearAndPresent(t, d, s, ui.Color{R: 1, A: 1}, nil)
	submitted := d.fenceValue
	var atResize struct {
		fence, completed uint64
		buffers          int
		args             []uintptr
	}
	proxy.onResize = func(a []uintptr) {
		atResize.fence = d.fenceValue
		atResize.completed = uint64(d.fence.Call(fenceGetCompletedValue))
		for _, b := range s.bufs {
			if b != nil {
				atResize.buffers++
			}
		}
		atResize.args = append([]uintptr(nil), a[:5]...)
	}
	if err := s.Configure(300, 180); err != nil {
		t.Fatalf("Configure: %+v", err)
	}
	if proxy.count(swapChainResizeBuffers) != 1 {
		t.Fatalf("ResizeBuffers was called %d times, want 1", proxy.count(swapChainResizeBuffers))
	}
	if atResize.fence <= submitted || atResize.completed < atResize.fence {
		t.Errorf("at ResizeBuffers the fence was signalled up to %d (the frame was %d) and the GPU had reached %d: the resize did not wait for the GPU", atResize.fence, submitted, atResize.completed)
	}
	if atResize.buffers != 0 {
		t.Errorf("%d buffers still held at ResizeBuffers", atResize.buffers)
	}
	if got := atResize.args; got[0] != 0 || got[1] != 300 || got[2] != 180 || uint32(got[3]) != dxgiFormatUnknown || got[4] != 0 {
		t.Errorf("ResizeBuffers(%v), want (0, 300, 180, UNKNOWN, 0)", got)
	}
	// The same size, and sizes of nothing, do nothing.
	for _, size := range []image.Point{{300, 180}, {0, 0}, {0, 50}, {50, 0}, {-4, 9}} {
		if err := s.Configure(size.X, size.Y); err != nil {
			t.Errorf("Configure(%v): %+v", size, err)
		}
	}
	if n := proxy.count(swapChainResizeBuffers); n != 1 {
		t.Errorf("ResizeBuffers was called %d times after sizes that change nothing", n)
	}
	tex, err := s.Acquire()
	if err != nil || tex.Desc().Width != 300 || tex.Desc().Height != 180 {
		t.Errorf("Acquire = %v, %v; want a buffer of 300x180", tex, err)
	}
}

// Resizing many times gives every buffer back: the objects, the descriptors and the memory are the same
// as after the first time at that size.
func TestWineSwapchainResizesLeaveNothingBehind(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	s, _ := newTestSwapchain(t, d, 160, 100)
	sizes := []image.Point{{160, 100}, {320, 90}, {48, 48}, {160, 100}}
	var base Stats
	for i := 0; i < 40; i++ {
		size := sizes[i%len(sizes)]
		if err := s.Configure(size.X, size.Y); err != nil {
			t.Fatalf("Configure(%v): %+v", size, err)
		}
		clearAndPresent(t, d, s, ui.Color{G: 1, A: 1}, nil)
		if i == 3 {
			base = d.Stats()
		}
		if i > 3 && i%len(sizes) == 3 {
			if got := d.Stats(); got != base {
				t.Fatalf("after %d resizes the device holds %+v, after 4 it held %+v", i+1, got, base)
			}
		}
	}
	s.Destroy()
	if got := d.Stats(); got.Descriptors != 0 || got.Textures != 0 {
		t.Errorf("after the swap chain is destroyed: %+v", got)
	}
	if err := d.Close(); err != nil || d.Leaks() != 0 {
		t.Errorf("Close: %v, leaks %d", err, d.Leaks())
	}
}

func TestWineSwapchainPresentPassesTheDamageAsDirtyRectangles(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	s, _ := newTestSwapchain(t, d, 200, 120)
	proxy, restore := makeSwapProxy(t, s)
	defer func() { restore(); s.Destroy() }()
	r := func(l, t, rt, b int32) d3d12Rect { return d3d12Rect{Left: l, Top: t, Right: rt, Bottom: b} }
	tests := []struct {
		name   string
		damage []image.Rectangle
		want   []d3d12Rect
	}{
		{"the whole window", nil, nil},
		{"one rectangle", []image.Rectangle{image.Rect(10, 20, 60, 90)}, []d3d12Rect{r(10, 20, 60, 90)}},
		{"two, one of them cut", []image.Rectangle{image.Rect(0, 0, 8, 8), image.Rect(150, 100, 400, 400)}, []d3d12Rect{r(0, 0, 8, 8), r(150, 100, 200, 120)}},
		{"nothing changed", []image.Rectangle{}, []d3d12Rect{r(0, 0, 1, 1)}},
		{"all of it outside", []image.Rectangle{image.Rect(500, 500, 600, 600)}, []d3d12Rect{r(0, 0, 1, 1)}},
	}
	for _, tt := range tests {
		clearAndPresent(t, d, s, ui.Color{B: 1, A: 1}, tt.damage)
		proxy.mu.Lock()
		got := proxy.rects[len(proxy.rects)-1]
		proxy.mu.Unlock()
		if fmt.Sprint(got) != fmt.Sprint(tt.want) {
			t.Errorf("%s: DXGI got the dirty rectangles %v, want %v", tt.name, got, tt.want)
		}
	}
	// Present1 is called with a sync interval of 0 and no flags.
	proxy.mu.Lock()
	defer proxy.mu.Unlock()
	for _, c := range proxy.calls {
		if c.index == swapChainPresent1 && (c.args[0] != 0 || c.args[1] != 0) {
			t.Errorf("Present1(%d, %d, ...), want a sync interval of 0 and no flags", c.args[0], c.args[1])
		}
	}
}

func TestWineSwapchainPresentAnswers(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	s, _ := newTestSwapchain(t, d, 100, 60)
	proxy, restore := makeSwapProxy(t, s)
	defer func() { restore(); s.Destroy() }()
	answer := func(hr win32.HRESULT) { proxy.presentAnswer = &hr }

	answer(win32.HRESULT(int32(dxgiStatusOccluded)))
	if err := s.Present(nil); err != nil {
		t.Errorf("DXGI_STATUS_OCCLUDED is a success, got %+v", err)
	}
	for _, hr := range []win32.HRESULT{win32.DXGIErrorDeviceRemoved, win32.DXGIErrorDeviceReset, win32.DXGIErrorDeviceHung} {
		d.removed = func() win32.HRESULT { return hr }
		answer(hr)
		err := s.Present(nil)
		appErr := wantRoute(t, err, tagGPUDeviceLost, "swapchain.Present->Device.hr->callHR->check->d3d12Error.GPUDeviceLost")
		if appErr.Metadata["reason"] != hr.String() || appErr.Metadata["method"] != "IDXGISwapChain1::Present1" {
			t.Errorf("%v: metadata %v, want the reason and the method", hr, appErr.Metadata)
		}
	}
	d.removed = d.deviceRemovedReason
	d.lost = false
	answer(win32.EOutOfMemory)
	_ = wantRoute(t, s.Present(nil), tagGPUOutOfMemory, "swapchain.Present->Device.hr->callHR->check->d3d12Error.GPUOutOfMemory")
	answer(win32.EInvalidArg)
	appErr := wantRoute(t, s.Present(nil), TagD3D12CallFailed, "swapchain.Present->Device.hr->callHR->check->d3d12Error.D3D12CallFailed")
	if appErr.Metadata["method"] != "IDXGISwapChain1::Present1" {
		t.Errorf("metadata %v", appErr.Metadata)
	}
}

func TestWineSwapchainErrors(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	t.Run("a surface that is not a window of Windows", func(t *testing.T) {
		for _, h := range []gpu.SurfaceHandle{
			{}, {Kind: gpu.SurfaceKindWayland, Window: 5}, {Kind: gpu.SurfaceKindWin32}, {Kind: gpu.SurfaceKindWin32, Window: 0xDEAD},
		} {
			_, err := d.NewSwapchain(h)
			_ = wantRoute(t, err, TagD3D12SwapchainFailed, "Device.NewSwapchain.D3D12SwapchainFailed")
		}
	})
	t.Run("a window that was destroyed", func(t *testing.T) {
		h := makeSwapWindow(t, 50, 50)
		if err := win32.DestroyWindow(h.Window); err != nil {
			t.Fatal(err)
		}
		_, err := d.NewSwapchain(h)
		_ = wantRoute(t, err, TagD3D12SwapchainFailed, "Device.NewSwapchain.D3D12SwapchainFailed")
	})
	t.Run("no format is accepted", func(t *testing.T) {
		d.swapFormats = []swapFormat{{Name: "nonsense", DXGI: 0x7FFF, Texture: explicit.TextureFormatBGRA8Unorm}}
		defer func() { d.swapFormats = nil }()
		_, err := d.NewSwapchain(makeSwapWindow(t, 50, 50))
		appErr := wantRoute(t, err, TagD3D12SwapchainFailed, "Device.NewSwapchain.D3D12SwapchainFailed")
		refused, _ := appErr.Metadata["refused"].([]string)
		if len(refused) != 1 || appErr.Metadata["width"] != 50 {
			t.Errorf("metadata %v, want the refused format and the size", appErr.Metadata)
		}
	})
	t.Run("a format that is refused falls back to the next", func(t *testing.T) {
		d.swapFormats = []swapFormat{{Name: "nonsense", DXGI: 0x7FFF, Texture: explicit.TextureFormatBGRA8Unorm}, swapFormats[1]}
		defer func() { d.swapFormats = nil }()
		s, _ := newTestSwapchain(t, d, 50, 50)
		defer s.Destroy()
		if s.format.DXGI != dxgiFormatRGBA8 || swapDesc(t, s).Format != dxgiFormatRGBA8 {
			t.Errorf("the swap chain has the format %s, want R8G8B8A8_UNORM", s.format.Name)
		}
		if tex, err := s.Acquire(); err != nil || tex.Desc().Format != explicit.TextureFormatRGBA8Unorm {
			t.Errorf("Acquire = %v, %v; want an RGBA buffer", tex, err)
		}
	})
	t.Run("a resize that fails leaves no buffers, and the next one recovers", func(t *testing.T) {
		s, _ := newTestSwapchain(t, d, 80, 80)
		proxy, restore := makeSwapProxy(t, s)
		defer func() { restore(); s.Destroy() }()
		fail := win32.EInvalidArg
		proxy.resizeAnswer = &fail
		err := s.Configure(120, 90)
		appErr := wantRoute(t, err, TagD3D12CallFailed, "swapchain.Configure->Device.hr->callHR->check->d3d12Error.D3D12CallFailed")
		if appErr.Metadata["method"] != "IDXGISwapChain::ResizeBuffers" {
			t.Errorf("metadata %v", appErr.Metadata)
		}
		_, err = s.Acquire()
		_ = wantRoute(t, err, TagD3D12SwapchainFailed, "swapchain.Acquire.D3D12SwapchainFailed")
		proxy.resizeAnswer = nil
		// The size is the same as the failed one, but there are no buffers, so Configure makes them.
		if err := s.Configure(120, 90); err != nil {
			t.Fatalf("Configure after the failure: %+v", err)
		}
		if tex, err := s.Acquire(); err != nil || tex.Desc().Width != 120 {
			t.Errorf("Acquire = %v, %v", tex, err)
		}
	})
	t.Run("acquiring from a destroyed swap chain", func(t *testing.T) {
		s, _ := newTestSwapchain(t, d, 50, 50)
		s.Destroy()
		_, err := s.Acquire()
		_ = wantRoute(t, err, TagD3D12SwapchainFailed, "swapchain.Acquire.D3D12SwapchainFailed")
	})
}

// presentedWindow is a window of the test that has a swap chain whose presents are read: before each Present the
// test reads the buffer back, to see what a compositor would show.
type presentedWindow struct {
	h      gpu.SurfaceHandle
	frames []*image.RGBA
	rects  [][]image.Rectangle
	sizes  []image.Point
}

// readingDevice is an explicit.Device whose swap chains read the buffer back before they present it.
type readingDevice struct {
	*Device
	t       *testing.T
	windows map[gpu.SurfaceHandle]*presentedWindow
}

func (r *readingDevice) NewSwapchain(h gpu.SurfaceHandle) (explicit.Swapchain, error) {
	sc, err := r.Device.NewSwapchain(h)
	if err != nil {
		return nil, err
	}
	w := r.windows[h]
	if w == nil {
		w = &presentedWindow{h: h}
		r.windows[h] = w
	}
	return &readingSwapchain{Swapchain: sc, d: r.Device, t: r.t, w: w}, nil
}

type readingSwapchain struct {
	explicit.Swapchain
	d *Device
	t *testing.T
	w *presentedWindow
}

func (s *readingSwapchain) Present(damage []image.Rectangle) error {
	// Acquire gives the buffer that is about to be presented again (the index changes at Present); the
	// read puts it back in COMMON, which Present needs.
	tex, err := s.Acquire()
	if err != nil {
		return err
	}
	size := image.Pt(tex.Desc().Width, tex.Desc().Height)
	img := readTexture(s.t, s.d, tex, image.Rectangle{Max: size})
	s.w.frames = append(s.w.frames, img)
	s.w.rects = append(s.w.rects, append([]image.Rectangle(nil), damage...))
	s.w.sizes = append(s.w.sizes, size)
	return s.Swapchain.Present(damage)
}

// windowRig is a renderer over real swap chains, with the windows it presented.
type windowRig struct {
	t       *testing.T
	r       *explicit.Renderer
	dev     *readingDevice
	opened  int
	windows map[gpu.SurfaceHandle]*presentedWindow
}

func newWindowRig(t *testing.T, damage bool) *windowRig {
	t.Helper()
	g := &windowRig{t: t, windows: map[gpu.SurfaceHandle]*presentedWindow{}}
	rec := &logRecorder{}
	runtimeLock(t)
	g.r = explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) {
		d, err := Open(Options{Logger: slogOf(rec), Getenv: env(nil)})
		if err != nil {
			return nil, err
		}
		g.opened++
		g.dev = &readingDevice{Device: d, t: t, windows: g.windows}
		return g.dev, nil
	}, explicit.RendererOpts{Background: ui.NewColor(0xFFFFFFFF), HasBackground: true, SkipUnsupported: true, Damage: damage})
	t.Cleanup(g.r.Close)
	return g
}

// rectScene is a scene of colored rectangles that move with the step.
func rectScene(step int) *ui.Scene {
	return gputest.CanvasScene(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(10, 10, 60, 40), gputest.FillPaint(0xFF3366CC))
		c.DrawRect(ui.NewRectFromLTWH(float64(20+8*step), 60, 30, 30), gputest.FillPaint(0xFFCC3333))
		c.DrawCircle(ui.Offset{Dx: 120, Dy: 40}, 25, gputest.FillPaint(0xFF33AA55))
	})
}

func TestWineTwoWindowsShareOneDevice(t *testing.T) {
	g := newWindowRig(t, true)
	a, b := makeSwapWindow(t, 220, 140), makeSwapWindow(t, 160, 100)
	for i := 0; i < 3; i++ {
		for _, w := range []struct {
			handle gpu.SurfaceHandle
			width  int
			height int
		}{{a, 220, 140}, {b, 160, 100}} {
			img, err := g.r.Draw(w.handle, w.width, w.height, rectScene(i), true)
			if err != nil {
				t.Fatalf("Draw: %+v", err)
			}
			want, err := soft.Render(rectScene(i), w.width, w.height)
			if err != nil {
				t.Fatal(err)
			}
			gputest.Tolerance{Over8Permille: 30, Over32Permille: 3}.Check(t, fmt.Sprintf("window %d frame %d", w.handle.Window, i), gputest.DiffRect(img, want, img.Bounds()))
		}
	}
	if g.opened != 1 {
		t.Errorf("%d devices were opened for two windows, want 1", g.opened)
	}
	st := g.dev.Stats()
	if got := len(g.windows); got != 2 {
		t.Fatalf("%d windows presented", got)
	}
	for h, w := range g.windows {
		if len(w.frames) != 3 {
			t.Errorf("window %d presented %d frames, want 3", h.Window, len(w.frames))
		}
	}
	// A snapshot is drawn on the same device while the windows have their swap chains, and is the frame's
	// pixels where it is opaque (SPEC §9.3).
	snap, err := g.r.Snapshot(rectScene(2), ui.Size{Width: 160, Height: 100})
	if err != nil {
		t.Fatalf("Snapshot: %+v", err)
	}
	last := g.windows[b].frames[2]
	differ := 0
	for i := 0; i < len(snap.Pix); i += 4 {
		if snap.Pix[i+3] != 255 {
			continue
		}
		for k := 0; k < 4; k++ {
			if d := int(snap.Pix[i+k]) - int(last.Pix[i+k]); d < -1 || d > 1 {
				differ++
				break
			}
		}
	}
	if snap.Bounds() != last.Bounds() || differ != 0 {
		t.Errorf("the snapshot is %v and differs from the window's frame in %d opaque pixels (the frame is %v)", snap.Bounds(), differ, last.Bounds())
	}
	// Closing one window leaves the other drawing, on the same device.
	g.r.Release(a)
	if _, err := g.r.Draw(b, 160, 100, rectScene(4), false); err != nil {
		t.Fatalf("Draw after Release: %+v", err)
	}
	if after := g.dev.Stats(); after.Textures >= st.Textures+3 {
		t.Errorf("textures: %d with two windows, %d after one left", st.Textures, after.Textures)
	}
	g.r.Close()
	if g.opened != 1 || g.dev.Leaks() != 0 {
		t.Errorf("after Close: %d devices opened, %d leaks", g.opened, g.dev.Leaks())
	}
}

func TestWineResizingPresentsEachSizeAndDamageEqualsWhole(t *testing.T) {
	whole, damaged := newWindowRig(t, false), newWindowRig(t, true)
	hw, hd := makeSwapWindow(t, 200, 120), makeSwapWindow(t, 200, 120)
	sizes := []image.Point{{200, 120}, {321, 77}, {64, 64}, {500, 260}}
	frame := 0
	for _, size := range sizes {
		for step := 0; step < 3; step++ { // the first is a whole frame, the next two change a little
			a, err := whole.r.Draw(hw, size.X, size.Y, rectScene(step), true)
			if err != nil {
				t.Fatalf("whole: %+v", err)
			}
			b, err := damaged.r.Draw(hd, size.X, size.Y, rectScene(step), true)
			if err != nil {
				t.Fatalf("damaged: %+v", err)
			}
			wf, df := whole.windows[hw], damaged.windows[hd]
			frame++
			if len(wf.frames) != frame || len(df.frames) != frame {
				t.Fatalf("frame %d: %d whole and %d damaged presents", frame, len(wf.frames), len(df.frames))
			}
			if wf.sizes[frame-1] != size || df.sizes[frame-1] != size {
				t.Errorf("frame %d at %v: presented at %v and %v", frame, size, wf.sizes[frame-1], df.sizes[frame-1])
			}
			// What is presented is the frame, whole or damaged.
			if d := gputest.DiffRect(wf.frames[frame-1], a, a.Bounds()); d.Over8 != 0 {
				t.Errorf("frame %d at %v: the whole presented frame differs from its capture by %d pixels", frame, size, d.Over8)
			}
			if d := gputest.DiffRect(df.frames[frame-1], b, b.Bounds()); d.Over8 != 0 {
				t.Errorf("frame %d at %v: the damaged presented frame differs from its capture by %d pixels", frame, size, d.Over8)
			}
			if d := gputest.DiffRect(df.frames[frame-1], wf.frames[frame-1], a.Bounds()); d.Over8 != 0 {
				t.Errorf("frame %d at %v: a frame with damage differs from a whole one by %d pixels", frame, size, d.Over8)
			}
			if step > 0 && len(df.rects[frame-1]) == 0 {
				t.Errorf("frame %d: the damaged renderer presented the whole window for a small change", frame)
			}
			if len(wf.rects[frame-1]) != 0 {
				t.Errorf("frame %d: the renderer without damage presented rectangles %v", frame, wf.rects[frame-1])
			}
		}
	}
}

// runtimeLock keeps the test on its OS thread until it ends: a device is used from one thread.
func runtimeLock(t *testing.T) {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
}

// slogOf makes a logger of a recorder.
func slogOf(rec *logRecorder) *slog.Logger { return slog.New(rec) }

// lossySwapchain loses the device at its first Present: it returns what DXGI returns for a removed device and
// presents nothing.
type lossySwapchain struct {
	explicit.Swapchain
	lose *bool
}

func (s lossySwapchain) Present(damage []image.Rectangle) error {
	if *s.lose {
		*s.lose = false
		return check("IDXGISwapChain1::Present1", win32.DXGIErrorDeviceRemoved)
	}
	return s.Swapchain.Present(damage)
}

// lossyDevice is a device whose swap chains lose it when lose is set.
type lossyDevice struct {
	*Device
	lose *bool
}

func (l lossyDevice) NewSwapchain(h gpu.SurfaceHandle) (explicit.Swapchain, error) {
	sc, err := l.Device.NewSwapchain(h)
	if err != nil {
		return nil, err
	}
	return lossySwapchain{Swapchain: sc, lose: l.lose}, nil
}

// TestWineDeviceLostAtPresentMakesTheSwapchainAgainForTheSameWindow: a Present that says the device was removed
// destroys the swap chain and the device, opens a new device and makes a new swap chain for the same window,
// and draws the frame again; a second loss within ten seconds ends with GPUDeviceLost (SPEC §9.3 "Decided in step
// 20.7", §9.8 "Device removal").
func TestWineDeviceLostAtPresentMakesTheSwapchainAgainForTheSameWindow(t *testing.T) {
	runtimeLock(t)
	rec := &logRecorder{}
	lose := false
	var devs []*Device
	r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) {
		d, err := Open(Options{Logger: slogOf(rec), Getenv: env(nil)})
		if err != nil {
			return nil, err
		}
		devs = append(devs, d)
		return lossyDevice{Device: d, lose: &lose}, nil
	}, explicit.RendererOpts{Background: ui.NewColor(0xFFFFFFFF), HasBackground: true, SkipUnsupported: true})
	t.Cleanup(r.Close)
	h := makeSwapWindow(t, 200, 120)
	want, err := r.Draw(h, 200, 120, rectScene(0), true)
	if err != nil {
		t.Fatalf("first frame: %+v", err)
	}
	lose = true
	got, err := r.Draw(h, 200, 120, rectScene(0), true)
	if err != nil {
		t.Fatalf("the frame after the loss: %+v", err)
	}
	if len(devs) != 2 || devs[0].Leaks() != 0 {
		t.Fatalf("%d devices opened, the first left %d objects", len(devs), devs[0].Leaks())
	}
	if d := gputest.DiffRect(got, want, want.Bounds()); d.Over8 != 0 {
		t.Errorf("the frame on the new device differs from the first one in %d pixels", d.Over8)
	}
	// The new device presents to the window: another frame, at another size, with its swap chain.
	if _, err := r.Draw(h, 90, 70, rectScene(1), false); err != nil {
		t.Fatalf("a frame on the new swap chain: %+v", err)
	}
	// A second loss soon after the first is the app's end.
	lose = true
	_, err = r.Draw(h, 90, 70, rectScene(2), false)
	_ = wantRoute(t, err, tagGPUDeviceLost, "check->d3d12Error.GPUDeviceLost")
}
