//go:build windows && glutter_wine

package d3d12

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"testing"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests run the real Direct3D 12 of the Wine prefix: vkd3d on the host's Vulkan (lavapipe on a
// headless server), SPEC §25.13. It is not Windows' Direct3D 12, so they check what holds in both: the
// device at 11_0, the layouts and the alignments, the barriers, the data that goes in and comes
// back exactly. What differs (the adapter's name, the debug layer, WARP, device removal) is said where
// the test depends on it. A device is used from the OS thread that made it (runtime.LockOSThread).

// logRecorder is a slog handler that keeps the records.
type logRecorder struct {
	mu      sync.Mutex
	records []slog.Record
}

func (l *logRecorder) Enabled(context.Context, slog.Level) bool { return true }
func (l *logRecorder) Handle(_ context.Context, r slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, r.Clone())
	return nil
}
func (l *logRecorder) WithAttrs([]slog.Attr) slog.Handler { return l }
func (l *logRecorder) WithGroup(string) slog.Handler      { return l }

// find returns the first record whose message contains the text.
func (l *logRecorder) find(text string) (slog.Record, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range l.records {
		if strings.Contains(r.Message, text) {
			return r, true
		}
	}
	return slog.Record{}, false
}

// attr reads an attribute of a record.
func attr(r slog.Record, key string) (v slog.Value, ok bool) {
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			v, ok = a.Value, true
			return false
		}
		return true
	})
	return v, ok
}

// env makes a Getenv from a map.
func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

// openDevice opens a device on the calling thread, which the test keeps for itself until the end.
func openTestDevice(t *testing.T, o Options) (*Device, *logRecorder) {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	rec := &logRecorder{}
	if o.Logger == nil {
		o.Logger = slog.New(rec)
	}
	if o.Getenv == nil {
		o.Getenv = env(nil)
	}
	d, err := Open(o)
	if err != nil {
		t.Fatalf("Open: %+v", err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return d, rec
}

// expectPanic runs fn and returns the tagged error it panicked with.
func expectPanic(t *testing.T, fn func()) (appErr *errors.Error) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("no panic")
		}
		err, ok := r.(error)
		if !ok {
			t.Fatalf("panic(%v), not an error", r)
		}
		appErr, ok = errors.FromError(err)
		if !ok {
			t.Fatalf("panic(%T), not *errors.Error", err)
		}
	}()
	fn()
	return nil
}

func wantRoute(t *testing.T, err error, tag, route string) *errors.Error {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(route) {
		t.Fatalf("got %s via %s; want %s via %s", appErr.Tag, appErr.Route(), tag, route)
	}
	return appErr
}

func TestWineDeviceIsMadeAtFeatureLevel11(t *testing.T) {
	d, rec := openTestDevice(t, Options{AllowWarp: true})
	adapters := d.Adapters()
	for _, a := range adapters {
		t.Logf("adapter %d: %q vendor %04X device %04X software %v video memory %d MiB", a.Index, a.Name, a.VendorID, a.DeviceID, a.Software, a.DedicatedVideoMemory>>20)
	}
	if len(adapters) == 0 {
		t.Fatal("DXGI lists no adapter")
	}
	info := d.Adapter()
	t.Logf("chosen: %s; highest feature level %s; binding tier %d", d.Description(), d.FeatureLevel(), d.BindingTier())
	if info.Software {
		t.Errorf("auto chose the software adapter %q while hardware was listed", info.Name)
	}
	if d.featureLevel < featureLevel11_0 {
		t.Errorf("the highest feature level is %#x", d.featureLevel)
	}
	if d.BindingTier() < int(resourceBindingTier1) {
		t.Errorf("resource binding tier %d, the device needs at least 1", d.BindingTier())
	}
	if !strings.HasPrefix(d.Description(), "d3d12: "+info.Name) {
		t.Errorf("Description = %q", d.Description())
	}
	if got := fmt.Sprintf("(%04X:%04X)", info.VendorID, info.DeviceID); !strings.HasSuffix(d.Description(), got) {
		t.Errorf("Description = %q, want it to end with %s", d.Description(), got)
	}
	// The report of SPEC §9.7: the adapter, the feature level and the binding tier.
	r, ok := rec.find("Direct3D 12 device")
	if !ok {
		t.Fatal("the device logged no startup line")
	}
	for _, k := range []string{"device", "feature_level", "max_feature_level", "binding_tier", "opened"} {
		if _, ok := attr(r, k); !ok {
			t.Errorf("the startup line has no %q", k)
		}
	}
	if v, _ := attr(r, "feature_level"); v.String() != "11_0" {
		t.Errorf("feature_level = %v, want 11_0", v)
	}
	if d.StartupReport().Context <= 0 {
		t.Error("the startup report has no time for the context")
	}
	if got := d.Limits(); got != explicit.DefaultLimits() {
		t.Errorf("Limits = %+v, want the design limits %+v", got, explicit.DefaultLimits())
	}
}

func TestWineAdapterChoiceByIndexAndName(t *testing.T) {
	first, _ := openTestDevice(t, Options{})
	adapters := first.Adapters()
	want := first.Adapter()
	name := strings.ToUpper(want.Name[:min(len(want.Name), 6)])
	for _, sel := range []string{fmt.Sprint(want.Index), name, strings.ToLower(want.Name)} {
		t.Run(sel, func(t *testing.T) {
			d, _ := openTestDevice(t, Options{Getenv: env(map[string]string{AdapterEnvVar: sel})})
			if got := d.Adapter(); got.Name != want.Name || got.VendorID != want.VendorID || got.DeviceID != want.DeviceID {
				t.Errorf("GLUTTER_D3D12_ADAPTER=%q chose %+v, auto chose %+v", sel, got, want)
			}
		})
	}
	t.Logf("%d adapters listed", len(adapters))
}

func TestWineAdapterChoiceFailsWithNoAdapter(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	rec := &logRecorder{}
	tests := []struct {
		name, selector string
	}{
		{"an index out of range", "99"},
		{"a name no adapter has", "no such adapter anywhere"},
		{"warp, which Wine does not have", "warp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Open(Options{Logger: slog.New(rec), Getenv: env(map[string]string{AdapterEnvVar: tt.selector})})
			if err == nil {
				t.Skipf("%q worked: this Direct3D has such an adapter", tt.selector)
			}
			appErr := wantRoute(t, err, TagD3D12NoAdapter, "Open->openWith->Device.openDevice")
			if tt.selector != "warp" && !appErr.HasRoute("chooseAdapters.D3D12NoAdapter") {
				t.Errorf("route = %s", appErr.Route())
			}
			if appErr.Metadata["selector"] != tt.selector {
				t.Errorf("metadata = %v", appErr.Metadata)
			}
		})
	}
}

func TestWineUnsupportedFeatureLevel(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// A feature level that no Direct3D 12 knows cannot be made on any adapter.
	_, err := openWith(Options{Logger: slog.New(&logRecorder{}), Getenv: env(nil)}, defaultEntryNames(), 0xFFFF)
	appErr := wantRoute(t, err, TagD3D12Unsupported, "openWith->Device.openDevice.D3D12Unsupported")
	if tried, _ := appErr.Metadata["tried"].([]string); len(tried) == 0 {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
}

func TestWineLoadFailed(t *testing.T) {
	tests := []struct {
		name    string
		mod     func(*entryNames)
		library string
		symbol  string
		cause   string
	}{
		{"d3d12.dll is missing", func(n *entryNames) { n.d3d12 = "glutter-no-such-d3d12.dll" }, "glutter-no-such-d3d12.dll", "", "LibraryLoadFailed"},
		{"dxgi.dll is missing", func(n *entryNames) { n.dxgi = "glutter-no-such-dxgi.dll" }, "glutter-no-such-dxgi.dll", "", "LibraryLoadFailed"},
		{"a function of d3d12.dll is missing", func(n *entryNames) { n.createDevice = "D3D12NoSuchFunction" }, "d3d12.dll", "D3D12NoSuchFunction", "SymbolMissing"},
		{"a function of dxgi.dll is missing", func(n *entryNames) { n.createFactory = "CreateNoSuchFactory" }, "dxgi.dll", "CreateNoSuchFactory", "SymbolMissing"},
		{"a library with a path is refused", func(n *entryNames) { n.dxgi = `C:\dxgi.dll` }, `C:\dxgi.dll`, "", "LibraryLoadFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := defaultEntryNames()
			tt.mod(&names)
			_, err := openWith(Options{Logger: slog.New(&logRecorder{}), Getenv: env(nil)}, names, featureLevel11_0)
			appErr := wantRoute(t, err, TagD3D12LoadFailed, "openWith->loadLibraries.D3D12LoadFailed")
			if appErr.Metadata["library"] != tt.library || appErr.Metadata["symbol"] != tt.symbol || appErr.Metadata["cause"] != tt.cause {
				t.Errorf("metadata = %v; want library %q symbol %q cause %q", appErr.Metadata, tt.library, tt.symbol, tt.cause)
			}
		})
	}
}

// GLUTTER_D3D12_DEBUG=1 turns the layer on when it is installed, and logs that it is missing and goes on
// when it is not (Wine has no d3d12sdklayers.dll).
func TestWineDebugLayerMissingIsLoggedAndIgnored(t *testing.T) {
	d, rec := openTestDevice(t, Options{Getenv: env(map[string]string{DebugEnvVar: "1"})})
	r, warned := rec.find("debug layer")
	if d.DebugLayer() {
		t.Logf("this Direct3D has the debug layer: the missing-layer path is not covered here")
		if warned {
			t.Errorf("the layer is on and still a warning was logged: %v", r.Message)
		}
		return
	}
	if !warned {
		t.Fatal("the layer is missing and nothing was logged")
	}
	if r.Level != slog.LevelWarn {
		t.Errorf("logged at %v, want Warn", r.Level)
	}
	if v, ok := attr(r, "hresult"); !ok || v.String() == "" {
		t.Errorf("the warning has no hresult")
	} else {
		missing, _ := attr(r, "missing")
		t.Logf("D3D12GetDebugInterface said %s (DXGI_ERROR_SDK_COMPONENT_MISSING: %v)", v, missing)
	}
	if _, ok := rec.find("Direct3D 12 device"); !ok {
		t.Error("the device did not go on to make itself")
	}
	if got := len(d.DebugErrors()); got != 0 {
		t.Errorf("%d debug messages without a layer", got)
	}
	// The same through Options.Debug.
	d2, rec2 := openTestDevice(t, Options{Debug: true})
	if _, ok := rec2.find("debug layer"); !ok || d2.DebugLayer() {
		t.Errorf("Options.Debug: warned %v, layer %v", ok, d2.DebugLayer())
	}
}

// A device that is lost on demand: the next Begin fails with GPUDeviceLost and the reason, and Close
// still leaves nothing. (A removed device cannot be made here: the test hook stands in for
// GetDeviceRemovedReason.)
func TestWineDeviceRemovalIsGPUDeviceLost(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	enc, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
	d.removed = func() win32.HRESULT { return win32.DXGIErrorDeviceRemoved }
	_, err = d.Begin()
	appErr := wantRoute(t, err, "GPUDeviceLost", "Device.Begin->Device.checkRemoved.GPUDeviceLost")
	if appErr.Metadata["reason"] != win32.DXGIErrorDeviceRemoved.String() {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
	if !d.lost {
		t.Error("the device does not know it was lost")
	}
	// The reason is added to an error that came from a failed call too.
	err = d.annotate(check("IDXGISwapChain1::Present1", win32.DXGIErrorDeviceReset))
	appErr = wantRoute(t, err, "GPUDeviceLost", "check->d3d12Error.GPUDeviceLost")
	if appErr.Metadata["reason"] != win32.DXGIErrorDeviceRemoved.String() {
		t.Errorf("the reason was not added: %v", appErr.Metadata)
	}
	// A reason other than a removal code is a loss too.
	d.removed = func() win32.HRESULT { return win32.EFail }
	_, err = d.Begin()
	_ = wantRoute(t, err, "GPUDeviceLost", "Device.Begin->Device.checkRemoved.GPUDeviceLost")
	d.removed = d.deviceRemovedReason
	if _, err := d.Begin(); err != nil {
		t.Errorf("the real device is fine and Begin failed: %v", err)
	}
}

// ---- resources ----

// frame runs fn in a frame and submits it.
func frame(t *testing.T, d *Device, fn func(e explicit.Encoder)) {
	t.Helper()
	e, err := d.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	fn(e)
	if err := e.Submit(); err != nil {
		t.Fatalf("Submit: %v", err)
	}
}

// readTexture reads a rectangle back, in a frame of its own.
func readTexture(t *testing.T, d *Device, tex explicit.Texture, r image.Rectangle) *image.RGBA {
	t.Helper()
	var got *image.RGBA
	calls := 0
	frame(t, d, func(e explicit.Encoder) {
		e.ReadTexture(tex, r, func(img *image.RGBA, err error) {
			calls++
			if err != nil {
				t.Errorf("ReadTexture: %v", err)
			}
			got = img
		})
		if calls != 0 {
			t.Error("done ran inside the frame that asked")
		}
	})
	if calls != 1 {
		t.Fatalf("done ran %d times after Submit", calls)
	}
	return got
}

// pattern fills width x height pixels of bpp bytes with values that tell the pixel and the byte
// apart, in rows stride bytes apart (the bytes between rows are 0xEE, which must never reach the
// texture).
func pattern(width, height, bpp, stride int, seed byte) []byte {
	data := make([]byte, stride*height)
	for i := range data {
		data[i] = 0xEE
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			for c := 0; c < bpp; c++ {
				data[y*stride+x*bpp+c] = seed + byte(x*7+y*13+c*31)
			}
		}
	}
	return data
}

// expected is the RGBA image that ReadTexture must give for pattern bytes of the format.
func expected(format explicit.TextureFormat, data []byte, width, height, stride int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	bpp := format.BytesPerPixel()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			p := data[y*stride+x*bpp:]
			var c color.RGBA
			switch format {
			case explicit.TextureFormatBGRA8Unorm:
				c = color.RGBA{p[2], p[1], p[0], p[3]}
			case explicit.TextureFormatR8Unorm:
				c = color.RGBA{p[0], p[0], p[0], 255}
			default:
				c = color.RGBA{p[0], p[1], p[2], p[3]}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func samePixels(t *testing.T, what string, got, want *image.RGBA) {
	t.Helper()
	if got == nil || got.Bounds() != want.Bounds() {
		t.Fatalf("%s: bounds %v, want %v", what, got.Bounds(), want.Bounds())
	}
	for y := 0; y < want.Bounds().Dy(); y++ {
		for x := 0; x < want.Bounds().Dx(); x++ {
			if g, w := got.RGBAAt(x, y), want.RGBAAt(x, y); g != w {
				t.Fatalf("%s: pixel (%d,%d) = %v, want %v", what, x, y, g, w)
			}
		}
	}
}

var formats = []struct {
	name   string
	format explicit.TextureFormat
}{
	{"rgba8", explicit.TextureFormatRGBA8Unorm},
	{"bgra8", explicit.TextureFormatBGRA8Unorm},
	{"r8", explicit.TextureFormatR8Unorm},
}

// gpuReadBuffer copies a buffer to a readback buffer on the GPU and returns what arrived: what the
// GPU saw of the bytes the CPU wrote.
func gpuReadBuffer(t *testing.T, d *Device, b explicit.Buffer, n int) []byte {
	t.Helper()
	rb, err := d.createBuffer(heapTypeReadback, n)
	if err != nil {
		t.Fatal(err)
	}
	frame(t, d, func(e explicit.Encoder) {
		e.(*encoder).list.Call(listCopyBufferRegion, rb.Ptr(), 0, b.(*buffer).res.Ptr(), 0, uintptr(n))
	})
	if err := d.waitIdle(); err != nil {
		t.Fatal(err)
	}
	ptr, err := d.mapBuffer(rb, true, n)
	if err != nil {
		t.Fatal(err)
	}
	got := append([]byte(nil), win32.Bytes(ptr, n)...)
	unmapBuffer(rb, 0)
	d.releaseBuffer(rb, n)
	return got
}

func TestWineBufferIsWrittenAndReadBackExactly(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	for _, size := range []int{1, 4, 100, 256, 4096, 64 << 10} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			b, err := d.NewBuffer(size, explicit.BufferUniform|explicit.BufferVertex)
			if err != nil {
				t.Fatal(err)
			}
			defer b.Destroy()
			if b.Size() != size {
				t.Errorf("Size = %d", b.Size())
			}
			data := pattern(size, 1, 1, size, 3)
			d.WriteBuffer(b, 0, data)
			n := max(size, 4)
			if got := gpuReadBuffer(t, d, b, n); string(got[:size]) != string(data) {
				t.Errorf("the GPU saw other bytes than the CPU wrote")
			}
			// A write in the middle changes only its bytes.
			if size >= 8 {
				d.WriteBuffer(b, size/2, []byte{1, 2, 3, 4})
				copy(data[size/2:], []byte{1, 2, 3, 4})
				if got := gpuReadBuffer(t, d, b, n); string(got[:size]) != string(data) {
					t.Errorf("a write at an offset changed other bytes")
				}
			}
		})
	}
	b, _ := d.NewBuffer(16, explicit.BufferUniform)
	for _, tt := range []struct{ offset, n int }{{-1, 4}, {13, 4}, {0, 17}, {16, 1}} {
		appErr := expectPanic(t, func() { d.WriteBuffer(b, tt.offset, make([]byte, tt.n)) })
		if appErr.Tag != tagD3D12WriteOutOfRange || !appErr.HasRoute("Device.WriteBuffer.D3D12WriteOutOfRange") {
			t.Errorf("offset %d, %d bytes: %s via %s", tt.offset, tt.n, appErr.Tag, appErr.Route())
		}
	}
	d.WriteBuffer(b, 16, nil) // an empty write at the end is fine
	b.Destroy()
	b.Destroy() // twice is fine
}

func TestWineTexturesAreWrittenAndReadBackExactly(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	sizes := []struct{ w, h int }{{1, 1}, {37, 23}, {64, 4}, {65, 3}, {256, 2}, {257, 5}, {600, 3}}
	for _, f := range formats {
		for _, sz := range sizes {
			t.Run(fmt.Sprintf("%s/%dx%d", f.name, sz.w, sz.h), func(t *testing.T) {
				tex, err := d.NewTexture(explicit.TextureDesc{Width: sz.w, Height: sz.h, Format: f.format,
					Usage: explicit.TextureSampled | explicit.TextureCopySource})
				if err != nil {
					t.Fatal(err)
				}
				defer tex.Destroy()
				bpp := f.format.BytesPerPixel()
				// The source rows are tightly packed: a pitch that is not a multiple of 256 in most sizes.
				stride := sz.w * bpp
				data := pattern(sz.w, sz.h, bpp, stride, 17)
				d.WriteTexture(tex, 0, image.Rect(0, 0, sz.w, sz.h), data, stride)
				samePixels(t, "whole texture", readTexture(t, d, tex, image.Rect(0, 0, sz.w, sz.h)), expected(f.format, data, sz.w, sz.h, stride))
			})
		}
	}
}

// A source with padding between rows and a rectangle inside the texture: only the rectangle changes.
func TestWineTextureWritesOfARectangleWithPaddedRows(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	for _, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			const w, h = 50, 40
			bpp := f.format.BytesPerPixel()
			tex, err := d.NewTexture(explicit.TextureDesc{Width: w, Height: h, Format: f.format, Usage: explicit.TextureSampled | explicit.TextureCopySource})
			if err != nil {
				t.Fatal(err)
			}
			defer tex.Destroy()
			base := pattern(w, h, bpp, w*bpp, 1)
			d.WriteTexture(tex, 0, image.Rect(0, 0, w, h), base, w*bpp)
			rect := image.Rect(7, 11, 7+19, 11+9)
			stride := rect.Dx()*bpp + 13 // padding between rows that must not be copied
			patch := pattern(rect.Dx(), rect.Dy(), bpp, stride, 99)
			d.WriteTexture(tex, 0, rect, patch, stride)
			want := expected(f.format, base, w, h, w*bpp)
			patchImg := expected(f.format, patch, rect.Dx(), rect.Dy(), stride)
			for y := 0; y < rect.Dy(); y++ {
				for x := 0; x < rect.Dx(); x++ {
					want.SetRGBA(rect.Min.X+x, rect.Min.Y+y, patchImg.RGBAAt(x, y))
				}
			}
			samePixels(t, "after the patch", readTexture(t, d, tex, image.Rect(0, 0, w, h)), want)
			// And a part of the texture can be read.
			sub := image.Rect(5, 9, 30, 25)
			got := readTexture(t, d, tex, sub)
			cut := want.SubImage(sub).(*image.RGBA)
			cutCopy := image.NewRGBA(image.Rect(0, 0, sub.Dx(), sub.Dy()))
			for y := 0; y < sub.Dy(); y++ {
				for x := 0; x < sub.Dx(); x++ {
					cutCopy.SetRGBA(x, y, cut.RGBAAt(sub.Min.X+x, sub.Min.Y+y))
				}
			}
			samePixels(t, "a rectangle read back", got, cutCopy)
		})
	}
}

// Writes made before a frame begins are recorded when it does; writes made in an open frame between
// passes are recorded at once; a texture destroyed with a write waiting is released after the frame
// that uses it.
func TestWineWritesBeforeAndInsideFrames(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	mk := func() explicit.Texture {
		tex, err := d.NewTexture(explicit.TextureDesc{Width: 16, Height: 16, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureSampled | explicit.TextureCopySource})
		if err != nil {
			t.Fatal(err)
		}
		return tex
	}
	full := image.Rect(0, 0, 16, 16)
	a, b := mk(), mk()
	dataA, dataB := pattern(16, 16, 4, 64, 5), pattern(16, 16, 4, 64, 77)
	d.WriteTexture(a, 0, full, dataA, 64) // before any frame
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	d.WriteTexture(b, 0, full, dataB, 64) // inside the open frame
	var gotA, gotB *image.RGBA
	e.ReadTexture(a, full, func(i *image.RGBA, err error) { gotA = i })
	e.ReadTexture(b, full, func(i *image.RGBA, err error) { gotB = i })
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
	samePixels(t, "write before the frame", gotA, expected(explicit.TextureFormatRGBA8Unorm, dataA, 16, 16, 64))
	samePixels(t, "write inside the frame", gotB, expected(explicit.TextureFormatRGBA8Unorm, dataB, 16, 16, 64))

	// A write is queued and the texture destroyed before the frame: nothing is released too early.
	c := mk()
	d.WriteTexture(c, 0, full, dataA, 64)
	c.Destroy()
	frame(t, d, func(e explicit.Encoder) {})
	if err := d.waitIdle(); err != nil {
		t.Fatal(err)
	}
	a.Destroy()
	b.Destroy()
	if s := d.Stats(); s.Textures != 0 || s.Descriptors != 0 {
		t.Errorf("stats after the destroys: %+v", s)
	}
}

// A write made while a pass is open is recorded in the next frame, and its staging memory is the next
// slot's, not the open frame's: the open frame is done on the GPU before the next one starts, but its
// half could be reused by the frame after it.
func TestWineWriteInsideAPassGoesToTheNextFrame(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	rt, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Destroy()
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 16, Height: 16, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	defer tex.Destroy()
	data := pattern(16, 16, 4, 64, 33)
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	open := e.(*encoder).slot
	pass := e.BeginPass(rt, explicit.LoadClear, ui.Color{})
	d.WriteTexture(tex, 0, image.Rect(0, 0, 16, 16), data, 64)
	pass.End()
	if got := d.Stats().StagingUsed; got[open] != 0 || got[(open+1)%framesInFlight] == 0 {
		t.Errorf("staging used %v: the write should be in the half of slot %d", got, (open+1)%framesInFlight)
	}
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
	samePixels(t, "write made inside a pass", readTexture(t, d, tex, image.Rect(0, 0, 16, 16)), expected(explicit.TextureFormatRGBA8Unorm, data, 16, 16, 64))
}

// A write larger than the half of the staging ring gets an upload buffer of its own, released when its
// frame is done.
func TestWineLargeWriteUsesAnUploadBufferOfItsOwn(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	const w, h = 2100, 1000 // 8.4 MB of pixels and more with the pitch: more than the 8 MiB of a half
	tex, err := d.NewTexture(explicit.TextureDesc{Width: w, Height: h, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	defer tex.Destroy()
	data := pattern(w, h, 4, w*4, 9)
	d.WriteTexture(tex, 0, image.Rect(0, 0, w, h), data, w*4)
	if got := d.Stats().OneOffUploads; got != 1 {
		t.Errorf("OneOffUploads = %d before the frame, want 1", got)
	}
	samePixels(t, "large texture", readTexture(t, d, tex, image.Rect(0, 0, w, h)), expected(explicit.TextureFormatRGBA8Unorm, data, w, h, w*4))
	if got := d.Stats().OneOffUploads; got != 0 {
		t.Errorf("OneOffUploads = %d after the frame, want 0", got)
	}
}

func TestWineMipLevelsAndLevelViews(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	const w, h, levels = 64, 32, 6
	for _, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			bpp := f.format.BytesPerPixel()
			tex, err := d.NewTexture(explicit.TextureDesc{Width: w, Height: h, Format: f.format, MipLevels: levels,
				Usage: explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource})
			if err != nil {
				t.Fatal(err)
			}
			defer tex.Destroy()
			if tex.Desc().MipLevels != levels {
				t.Fatalf("MipLevels = %d", tex.Desc().MipLevels)
			}
			datas := make([][]byte, levels)
			for l := range levels {
				lw, lh := max(w>>l, 1), max(h>>l, 1)
				datas[l] = pattern(lw, lh, bpp, lw*bpp, byte(40*l+1))
				d.WriteTexture(tex, l, image.Rect(0, 0, lw, lh), datas[l], lw*bpp)
			}
			for l := range levels {
				lw, lh := max(w>>l, 1), max(h>>l, 1)
				view := tex.Level(l)
				if got := view.Desc(); got.Width != lw || got.Height != lh || got.MipLevels != 1 || got.Format != f.format {
					t.Errorf("Level(%d).Desc() = %+v", l, got)
				}
				if tex.Level(l) != view {
					t.Errorf("Level(%d) made a second view", l)
				}
				samePixels(t, fmt.Sprintf("level %d through its view", l), readTexture(t, d, view, image.Rect(0, 0, lw, lh)), expected(f.format, datas[l], lw, lh, lw*bpp))
			}
			// The parent reads its first level.
			samePixels(t, "level 0 through the parent", readTexture(t, d, tex, image.Rect(0, 0, w, h)), expected(f.format, datas[0], w, h, w*bpp))
			// The view of a level has no levels of its own.
			if tex.Level(2).Level(0) != tex.Level(2) {
				t.Error("Level(0) of a level view is not the view")
			}
			appErr := expectPanic(t, func() { tex.Level(levels) })
			if appErr.Tag != tagD3D12LevelInvalid || !appErr.HasRoute("texture.Level.D3D12LevelInvalid") {
				t.Errorf("%s via %s", appErr.Tag, appErr.Route())
			}
			// The views have descriptors, which the parent frees: the shader resource view of the texture
			// and of each level, and the render target view of the texture and of each level.
			if got := d.Stats().Descriptors; got != 2*(levels+1) {
				t.Errorf("descriptors = %d, want %d", got, 2*(levels+1))
			}
		})
	}
	if got := d.Stats().Descriptors; got != 0 {
		t.Errorf("descriptors after the textures went = %d, want 0", got)
	}
}

func TestWineSinglelevelTextureIsItsOwnLevel(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	defer tex.Destroy()
	if tex.Level(0) != tex {
		t.Error("Level(0) of a texture with one level is not the texture")
	}
	appErr := expectPanic(t, func() { tex.Level(1) })
	if appErr.Tag != tagD3D12LevelInvalid {
		t.Errorf("tag = %s", appErr.Tag)
	}
}

func TestWineCopyTexture(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	for _, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			bpp := f.format.BytesPerPixel()
			mk := func(w, h, levels int) explicit.Texture {
				tex, err := d.NewTexture(explicit.TextureDesc{Width: w, Height: h, Format: f.format, MipLevels: levels,
					Usage: explicit.TextureSampled | explicit.TextureCopySource | explicit.TextureRenderTarget})
				if err != nil {
					t.Fatal(err)
				}
				return tex
			}
			src, dst := mk(40, 30, 1), mk(50, 50, 1)
			defer src.Destroy()
			defer dst.Destroy()
			srcData, dstData := pattern(40, 30, bpp, 40*bpp, 11), pattern(50, 50, bpp, 50*bpp, 222)
			d.WriteTexture(src, 0, image.Rect(0, 0, 40, 30), srcData, 40*bpp)
			d.WriteTexture(dst, 0, image.Rect(0, 0, 50, 50), dstData, 50*bpp)
			rect, at := image.Rect(3, 4, 3+21, 4+17), image.Pt(12, 20)
			frame(t, d, func(e explicit.Encoder) { e.CopyTexture(src, rect, dst, at) })

			want := expected(f.format, dstData, 50, 50, 50*bpp)
			srcImg := expected(f.format, srcData, 40, 30, 40*bpp)
			for y := 0; y < rect.Dy(); y++ {
				for x := 0; x < rect.Dx(); x++ {
					want.SetRGBA(at.X+x, at.Y+y, srcImg.RGBAAt(rect.Min.X+x, rect.Min.Y+y))
				}
			}
			samePixels(t, "destination", readTexture(t, d, dst, image.Rect(0, 0, 50, 50)), want)
			samePixels(t, "source is unchanged", readTexture(t, d, src, image.Rect(0, 0, 40, 30)), srcImg)

			// Between two levels of one texture: different subresources, so it is allowed.
			m := mk(32, 32, 3)
			defer m.Destroy()
			l1 := pattern(16, 16, bpp, 16*bpp, 60)
			d.WriteTexture(m, 1, image.Rect(0, 0, 16, 16), l1, 16*bpp)
			d.WriteTexture(m, 2, image.Rect(0, 0, 8, 8), pattern(8, 8, bpp, 8*bpp, 5), 8*bpp)
			frame(t, d, func(e explicit.Encoder) { e.CopyTexture(m.Level(1), image.Rect(0, 0, 8, 8), m.Level(2), image.Point{}) })
			l1Img := expected(f.format, l1, 16, 16, 16*bpp)
			wantL2 := image.NewRGBA(image.Rect(0, 0, 8, 8))
			for y := range 8 {
				for x := range 8 {
					wantL2.SetRGBA(x, y, l1Img.RGBAAt(x, y))
				}
			}
			samePixels(t, "level 2 after a copy from level 1", readTexture(t, d, m.Level(2), image.Rect(0, 0, 8, 8)), wantL2)

			// An empty rectangle copies nothing.
			frame(t, d, func(e explicit.Encoder) { e.CopyTexture(src, image.Rectangle{}, dst, image.Point{}) })
		})
	}
}

func TestWineCopyOntoItselfIsAStateConflict(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	defer tex.Destroy()
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	appErr := expectPanic(t, func() { e.CopyTexture(tex, image.Rect(0, 0, 4, 4), tex, image.Pt(4, 4)) })
	if appErr.Tag != tagD3D12StateConflict || !appErr.HasRoute("encoder.CopyTexture->stateTracker[...].need.D3D12StateConflict") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
}

// A render target is cleared, kept by LoadLoad, and read back: the barriers between the states
// render target, copy source and copy destination work.
func TestWinePassClearsAndKeepsTheTarget(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	for _, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			tex, err := d.NewTexture(explicit.TextureDesc{Width: 20, Height: 10, Format: f.format, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
			if err != nil {
				t.Fatal(err)
			}
			defer tex.Destroy()
			var want color.RGBA
			var clear ui.Color
			switch f.format {
			case explicit.TextureFormatBGRA8Unorm, explicit.TextureFormatRGBA8Unorm:
				clear, want = ui.Color{R: 1, G: 0, B: 0, A: 1}, color.RGBA{255, 0, 0, 255}
			default:
				clear, want = ui.Color{R: 1}, color.RGBA{255, 255, 255, 255}
			}
			frame(t, d, func(e explicit.Encoder) {
				e.BeginPass(tex, explicit.LoadClear, clear).End()
			})
			wantImg := image.NewRGBA(image.Rect(0, 0, 20, 10))
			for y := range 10 {
				for x := range 20 {
					wantImg.SetRGBA(x, y, want)
				}
			}
			samePixels(t, "cleared", readTexture(t, d, tex, image.Rect(0, 0, 20, 10)), wantImg)
			// LoadLoad does not clear: a pass with another clear color changes nothing.
			frame(t, d, func(e explicit.Encoder) {
				e.BeginPass(tex, explicit.LoadLoad, ui.Color{G: 1, A: 1}).End()
			})
			samePixels(t, "loaded", readTexture(t, d, tex, image.Rect(0, 0, 20, 10)), wantImg)
		})
	}
}

func TestWinePassErrors(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	rt, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Destroy()
	plain, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	defer plain.Destroy()
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	// Not a render target.
	appErr := expectPanic(t, func() { e.BeginPass(plain, explicit.LoadClear, ui.Color{}) })
	if appErr.Tag != tagD3D12TextureInvalid || !appErr.HasRoute("encoder.BeginPass.D3D12TextureInvalid") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
	pass := e.BeginPass(rt, explicit.LoadClear, ui.Color{})
	// A second pass, a copy, a read and a Submit before the pass ends.
	for name, fn := range map[string]func(){
		"BeginPass":   func() { e.BeginPass(rt, explicit.LoadClear, ui.Color{}) },
		"CopyTexture": func() { e.CopyTexture(rt, image.Rect(0, 0, 1, 1), plain, image.Point{}) },
		"ReadTexture": func() { e.ReadTexture(rt, image.Rect(0, 0, 1, 1), func(*image.RGBA, error) {}) },
		"Submit":      func() { _ = e.Submit() },
	} {
		appErr := expectPanic(t, fn)
		if appErr.Tag != tagD3D12PassNotEnded || !appErr.HasRoute("encoder."+name+".D3D12PassNotEnded") {
			t.Errorf("%s: %s via %s", name, appErr.Tag, appErr.Route())
		}
	}
	pass.End()
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
	// Submit on a frame that is not open, and Begin on a frame that is.
	err = e.Submit()
	_ = wantRoute(t, err, tagD3D12FrameNotOpen, "encoder.Submit.D3D12FrameNotOpen")
	e1, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Begin()
	_ = wantRoute(t, err, tagD3D12FrameOpen, "Device.Begin.D3D12FrameOpen")
	if err := e1.Submit(); err != nil {
		t.Fatal(err)
	}
	// A handle that is not a window of Windows cannot have a swap chain.
	_, err = d.NewSwapchain(gpuHandle())
	_ = wantRoute(t, err, TagD3D12SwapchainFailed, "Device.NewSwapchain.D3D12SwapchainFailed")
}

func TestWineResourceErrors(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	other, _ := openTestDevice(t, Options{})
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	defer tex.Destroy()
	otherTex, err := other.NewTexture(explicit.TextureDesc{Width: 8, Height: 8, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	defer otherTex.Destroy()
	otherBuf, _ := other.NewBuffer(16, explicit.BufferUniform)
	defer otherBuf.Destroy()

	t.Run("invalid textures", func(t *testing.T) {
		for name, desc := range map[string]explicit.TextureDesc{
			"no width":             {Width: 0, Height: 4, Format: explicit.TextureFormatRGBA8Unorm},
			"no height":            {Width: 4, Height: 0, Format: explicit.TextureFormatRGBA8Unorm},
			"no format":            {Width: 4, Height: 4},
			"more levels than fit": {Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, MipLevels: 4},
		} {
			_, err := d.NewTexture(desc)
			if appErr := wantRoute(t, err, tagD3D12TextureInvalid, "Device.NewTexture.D3D12TextureInvalid"); appErr.Metadata["levels"] == nil {
				t.Errorf("%s: metadata = %v", name, appErr.Metadata)
			}
		}
		// The most levels a size has is down to 1x1.
		ok, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, MipLevels: 3, Usage: explicit.TextureSampled})
		if err != nil {
			t.Fatalf("4x4 with 3 levels: %v", err)
		}
		ok.Destroy()
	})
	t.Run("a design limit", func(t *testing.T) {
		if !foundation.DebugMode {
			t.Skip("the limits panic in debug builds only")
		}
		appErr := expectPanic(t, func() {
			_, _ = d.NewTexture(explicit.TextureDesc{Width: 8193, Height: 1, Format: explicit.TextureFormatRGBA8Unorm})
		})
		if appErr.Tag != "GPULimitExceeded" {
			t.Errorf("tag = %s", appErr.Tag)
		}
	})
	t.Run("resources of another device", func(t *testing.T) {
		for name, fn := range map[string]func(){
			"WriteBuffer":  func() { d.WriteBuffer(otherBuf, 0, []byte{1}) },
			"WriteTexture": func() { d.WriteTexture(otherTex, 0, image.Rect(0, 0, 1, 1), []byte{1, 2, 3, 4}, 4) },
		} {
			appErr := expectPanic(t, fn)
			if appErr.Tag != tagD3D12ForeignResource || !appErr.HasRoute("Device."+name) {
				t.Errorf("%s: %s via %s", name, appErr.Tag, appErr.Route())
			}
		}
		e, _ := d.Begin()
		for name, fn := range map[string]func(){
			"CopyTexture": func() { e.CopyTexture(otherTex, image.Rect(0, 0, 1, 1), tex, image.Point{}) },
			"ReadTexture": func() { e.ReadTexture(otherTex, image.Rect(0, 0, 1, 1), func(*image.RGBA, error) {}) },
			"BeginPass":   func() { e.BeginPass(otherTex, explicit.LoadClear, ui.Color{}) },
		} {
			appErr := expectPanic(t, fn)
			if appErr.Tag != tagD3D12ForeignResource || !appErr.HasRoute("encoder."+name+"->Device.own2texture.D3D12ForeignResource") {
				t.Errorf("%s: %s via %s", name, appErr.Tag, appErr.Route())
			}
		}
		if err := e.Submit(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("writes out of range", func(t *testing.T) {
		full := image.Rect(0, 0, 8, 8)
		tests := []struct {
			name   string
			level  int
			r      image.Rectangle
			data   []byte
			stride int
			tag    string
		}{
			{"a rectangle past the edge", 0, image.Rect(4, 4, 12, 8), make([]byte, 8*4*4), 32, tagD3D12WriteOutOfRange},
			{"a negative corner", 0, image.Rect(-1, 0, 4, 4), make([]byte, 5*4*4), 20, tagD3D12WriteOutOfRange},
			{"data too short", 0, full, make([]byte, 8*4*8-1), 32, tagD3D12WriteOutOfRange},
			{"a stride smaller than a row", 0, full, make([]byte, 8*4*8), 31, tagD3D12WriteOutOfRange},
			{"a level that does not exist", 1, full, make([]byte, 8*4*8), 32, tagD3D12LevelInvalid},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				appErr := expectPanic(t, func() { d.WriteTexture(tex, tt.level, tt.r, tt.data, tt.stride) })
				if appErr.Tag != tt.tag || !appErr.HasRoute("Device.WriteTexture."+tt.tag) {
					t.Errorf("%s via %s", appErr.Tag, appErr.Route())
				}
			})
		}
	})
	t.Run("copies and reads out of range", func(t *testing.T) {
		e, _ := d.Begin()
		for name, fn := range map[string]func(){
			"CopyTexture": func() { e.CopyTexture(tex, image.Rect(0, 0, 9, 8), tex, image.Point{}) },
			"ReadTexture": func() { e.ReadTexture(tex, image.Rect(0, 0, 8, 9), func(*image.RGBA, error) {}) },
		} {
			appErr := expectPanic(t, fn)
			if appErr.Tag != tagD3D12RectOutOfRange || !appErr.HasRoute("encoder."+name+".D3D12RectOutOfRange") {
				t.Errorf("%s: %s via %s", name, appErr.Tag, appErr.Route())
			}
		}
		if err := e.Submit(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("a failing call is D3D12CallFailed", func(t *testing.T) {
		// A resource of an unknown dimension is an invalid argument of CreateCommittedResource.
		hp := heapProps(heapTypeDefault)
		rd := d3d12ResourceDesc{Width: 4, Height: 4, DepthOrArraySize: 1, MipLevels: 1, Format: dxgiFormatRGBA8, SampleDesc: dxgiSampleDesc{Count: 1}}
		var out uintptr
		err := d.hr(d.dev, "ID3D12Device::CreateCommittedResource", deviceCreateCommittedResource,
			uintptr(unsafe.Pointer(&hp)), 0, uintptr(unsafe.Pointer(&rd)), uintptr(resourceStateCommon), 0,
			uintptr(unsafe.Pointer(&iidResource)), uintptr(unsafe.Pointer(&out)))
		if err == nil {
			t.Fatal("a resource of an unknown dimension was made")
		}
		appErr := wantRoute(t, err, TagD3D12CallFailed, "Device.hr->callHR->check->d3d12Error.D3D12CallFailed")
		if appErr.Metadata["method"] != "ID3D12Device::CreateCommittedResource" {
			t.Errorf("metadata = %v", appErr.Metadata)
		}
	})
	t.Run("a buffer too large is GPUOutOfMemory or an invalid call", func(t *testing.T) {
		before := d.Stats()
		_, err := d.createBuffer(heapTypeUpload, 1<<44)
		if err == nil {
			t.Fatal("a buffer of 16 TiB was made")
		}
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("%T", err)
		}
		t.Logf("a buffer of 16 TiB: %s (%v)", appErr.Tag, appErr.Metadata["hresult"])
		if appErr.Tag != "GPUOutOfMemory" && appErr.Tag != TagD3D12CallFailed {
			t.Errorf("tag = %s", appErr.Tag)
		}
		if after := d.Stats(); after != before {
			t.Errorf("a failed buffer changed the stats: %+v -> %+v", before, after)
		}
	})
}

func TestWineSamplers(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	var all []explicit.Sampler
	filters := []explicit.FilterMode{explicit.FilterUnspecified, explicit.FilterNearest, explicit.FilterLinear}
	addresses := []explicit.AddressMode{explicit.AddressUnspecified, explicit.AddressClamp, explicit.AddressRepeat, explicit.AddressMirror}
	for _, min := range filters {
		for _, mag := range filters {
			for _, mip := range filters {
				for _, a := range addresses {
					s, err := d.NewSampler(explicit.SamplerDesc{MinFilter: min, MagFilter: mag, MipmapFilter: mip, AddressU: a, AddressV: addresses[(int(a)+1)%len(addresses)]})
					if err != nil {
						t.Fatal(err)
					}
					all = append(all, s)
				}
			}
		}
	}
	if got := d.Stats(); got.Samplers != len(all) || got.Descriptors != len(all) {
		t.Errorf("stats after %d samplers: %+v", len(all), got)
	}
	// More samplers than a heap holds: a second heap is added.
	for len(all) < cpuDescriptorsPerHeap+100 {
		s, err := d.NewSampler(explicit.SamplerDesc{})
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, s)
	}
	if got := len(d.cpu[2].heaps); got != 2 {
		t.Errorf("sampler heaps = %d, want 2", got)
	}
	for _, s := range all {
		s.Destroy()
		s.Destroy() // twice is fine
	}
	if got := d.Stats(); got.Samplers != 0 || got.Descriptors != 0 {
		t.Errorf("stats after the destroys: %+v", got)
	}
	// The freed numbers are reused, no heap is added.
	for range 50 {
		if _, err := d.NewSampler(explicit.SamplerDesc{}); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(d.cpu[2].heaps); got != 2 {
		t.Errorf("sampler heaps after reuse = %d, want 2", got)
	}
}

// The layout the device computes for a copy is the one Direct3D 12 reports for a placed footprint.
func TestWineCopyLayoutMatchesGetCopyableFootprints(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	for _, f := range formats {
		for _, sz := range []struct{ w, h int }{{1, 1}, {37, 23}, {64, 4}, {65, 3}, {255, 2}, {257, 9}, {1000, 5}} {
			rd := d3d12ResourceDesc{
				Dimension: resourceDimensionTexture2D, Width: uint64(sz.w), Height: uint32(sz.h), DepthOrArraySize: 1, MipLevels: 1,
				Format: dxgiFormat(f.format), SampleDesc: dxgiSampleDesc{Count: 1}, Layout: textureLayoutUnknown,
			}
			var fp d3d12PlacedFootprint
			var rows uint32
			var rowSize, total uint64
			d.dev.Call(deviceGetCopyableFootprints, uintptr(unsafe.Pointer(&rd)), 0, 1, 0,
				uintptr(unsafe.Pointer(&fp)), uintptr(unsafe.Pointer(&rows)), uintptr(unsafe.Pointer(&rowSize)), uintptr(unsafe.Pointer(&total)))
			l := layoutOf(d.limits, sz.w, sz.h, f.format.BytesPerPixel())
			if int(fp.Footprint.RowPitch) != l.RowPitch || int(rows) != l.Rows || int(rowSize) != l.RowBytes || int(total) != l.Size {
				t.Errorf("%s %dx%d: device says pitch %d rows %d row bytes %d total %d; layoutOf says %+v",
					f.name, sz.w, sz.h, fp.Footprint.RowPitch, rows, rowSize, total, l)
			}
		}
	}
}

// ---- frames and memory ----

// 100 frames in a row that upload, copy and read back do not grow memory: the resources, the committed
// bytes, the descriptors, the staging ring's heads and the Go heap are the same at the end as after the
// first frames.
func TestWine100FramesDoNotGrowMemory(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	const w, h = 96, 64
	mk := func() explicit.Texture {
		tex, err := d.NewTexture(explicit.TextureDesc{Width: w, Height: h, Format: explicit.TextureFormatRGBA8Unorm, MipLevels: 3,
			Usage: explicit.TextureSampled | explicit.TextureCopySource | explicit.TextureRenderTarget})
		if err != nil {
			t.Fatal(err)
		}
		return tex
	}
	a, b := mk(), mk()
	defer a.Destroy()
	defer b.Destroy()
	data := pattern(w, h, 4, w*4, 1)
	full := image.Rect(0, 0, w, h)
	runFrame := func(i int) {
		d.WriteTexture(a, 0, full, data, w*4)
		e, err := d.Begin()
		if err != nil {
			t.Fatalf("frame %d: Begin: %v", i, err)
		}
		d.WriteTexture(a, 1, image.Rect(0, 0, w/2, h/2), data, w*4)
		e.CopyTexture(a, full, b, image.Point{})
		e.BeginPass(b, explicit.LoadClear, ui.Color{R: 1, A: 1}).End()
		if i%10 == 0 {
			e.ReadTexture(b, image.Rect(0, 0, 8, 8), func(img *image.RGBA, err error) {
				if err != nil || img.RGBAAt(0, 0) != (color.RGBA{255, 0, 0, 255}) {
					t.Errorf("frame %d read back %v, %v", i, img.RGBAAt(0, 0), err)
				}
			})
		}
		if err := e.Submit(); err != nil {
			t.Fatalf("frame %d: Submit: %v", i, err)
		}
	}
	for i := range 4 { // warm-up: the staging buffer, the lists and the allocators reach their sizes
		runFrame(i)
	}
	if err := d.waitIdle(); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	// The device's own reference count: every resource, heap, allocator and list that is alive holds
	// one, so a COM object that a frame leaves behind shows here whatever the counters say.
	refs := func() uint32 { n := d.dev.AddRef(); d.dev.Release(); return n }
	refsBefore := refs()
	base := d.Stats()
	if base.CommittedBytes < stagingSize || base.Objects == 0 {
		t.Fatalf("the baseline measures nothing: %+v", base)
	}
	t.Logf("baseline: %+v", base)
	for i := 4; i < 104; i++ {
		runFrame(i)
		got := d.Stats()
		// The ring's heads count the uploads of the frames since the slot was last reused; they never
		// pass the half.
		for slot, used := range got.StagingUsed {
			if used > stagingSize/framesInFlight {
				t.Fatalf("frame %d: slot %d took %d bytes of its half", i, slot, used)
			}
		}
		got.StagingUsed = base.StagingUsed
		if got != base {
			t.Fatalf("frame %d: stats %+v, baseline %+v", i, got, base)
		}
	}
	if err := d.waitIdle(); err != nil {
		t.Fatal(err)
	}
	if got := refs(); got != refsBefore {
		t.Errorf("the device holds %d references after 100 frames, %d before", got, refsBefore)
	}
	if got := d.fence.Call(fenceGetCompletedValue); uint64(got) != d.fenceValue {
		t.Errorf("after waitIdle the fence is at %d, the last value is %d", got, d.fenceValue)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	if grew := int64(after.HeapAlloc) - int64(before.HeapAlloc); grew > 4<<20 {
		t.Errorf("the Go heap grew by %d bytes in 100 frames", grew)
	}
	for slot := range framesInFlight {
		if used := d.staging.ring.used(slot); used > stagingSize/framesInFlight {
			t.Errorf("slot %d took %d bytes", slot, used)
		}
	}
}

// The staging ring is released by frame: after a slot is waited for, its half is empty again.
func TestWineStagingIsReleasedByFrame(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 100, Height: 100, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	defer tex.Destroy()
	data := pattern(100, 100, 4, 400, 0)
	d.WriteTexture(tex, 0, image.Rect(0, 0, 100, 100), data, 400) // for the slot of the next frame
	used := d.Stats().StagingUsed[d.next]
	if want := layoutOf(d.limits, 100, 100, 4).Size; used != want {
		t.Errorf("the slot took %d bytes, want %d", used, want)
	}
	slot := d.next
	frame(t, d, func(e explicit.Encoder) {})
	frame(t, d, func(e explicit.Encoder) {})
	frame(t, d, func(e explicit.Encoder) {}) // the third frame reuses the slot
	if got := d.Stats().StagingUsed[slot]; got != 0 {
		t.Errorf("the slot's half is not free after its frame: %d bytes", got)
	}
}

// The fence wait returns once the GPU is done: the completed value reaches the slot's value.
func TestWineFenceWait(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	slot := e.(*encoder).slot
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
	value := d.slots[slot].fence
	if value == 0 || d.slots[slot].state != slotInflight {
		t.Fatalf("slot %d after Submit: fence %d state %v", slot, value, d.slots[slot].state)
	}
	if err := d.waitFor(value); err != nil {
		t.Fatal(err)
	}
	if got := uint64(d.fence.Call(fenceGetCompletedValue)); got < value {
		t.Errorf("the fence is at %d after waiting for %d", got, value)
	}
	// Waiting for a value in the past returns at once, and a slot of the past is ready after prepare.
	if err := d.waitFor(1); err != nil {
		t.Fatal(err)
	}
	if err := d.prepare(slot); err != nil {
		t.Fatal(err)
	}
	if d.slots[slot].state != slotReady {
		t.Errorf("state = %v after prepare", d.slots[slot].state)
	}
	// A frame whose reads wait inside Submit leaves its slot ready.
	tex, _ := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureCopySource})
	defer tex.Destroy()
	e, _ = d.Begin()
	slot = e.(*encoder).slot
	e.ReadTexture(tex, image.Rect(0, 0, 4, 4), func(*image.RGBA, error) {})
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
	if d.slots[slot].state != slotReady {
		t.Errorf("a frame with a read left its slot in state %v", d.slots[slot].state)
	}
}

// ReadTexture of an empty rectangle gives an empty image after Submit.
func TestWineReadOfAnEmptyRectangle(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	tex, _ := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureCopySource})
	defer tex.Destroy()
	img := readTexture(t, d, tex, image.Rectangle{})
	if img == nil || !img.Bounds().Empty() {
		t.Errorf("image = %v", img)
	}
}

// Close leaves nothing: every COM object the device made is released, and the device itself has no
// reference left (Leaks), also with resources that were not destroyed and frames that were not
// waited for.
func TestWineCloseLeavesNothing(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	d, err := Open(Options{Logger: slog.New(&logRecorder{}), Getenv: env(nil)})
	if err != nil {
		t.Fatal(err)
	}
	if got := d.Stats().Objects; got == 0 {
		t.Fatalf("a new device counts no objects")
	}
	var textures []explicit.Texture
	for i := range 5 {
		tex, err := d.NewTexture(explicit.TextureDesc{Width: 32, Height: 32, Format: explicit.TextureFormatBGRA8Unorm, MipLevels: 3,
			Usage: explicit.TextureSampled | explicit.TextureRenderTarget | explicit.TextureCopySource})
		if err != nil {
			t.Fatal(err)
		}
		tex.Level(1)
		textures = append(textures, tex)
		if i%2 == 0 {
			tex.Destroy()
		}
	}
	b, _ := d.NewBuffer(1000, explicit.BufferUniform)
	if _, err := d.NewSampler(explicit.SamplerDesc{}); err != nil {
		t.Fatal(err)
	}
	_ = b
	d.WriteTexture(textures[1], 0, image.Rect(0, 0, 32, 32), make([]byte, 32*32*4), 128)
	frame(t, d, func(e explicit.Encoder) {
		e.ReadTexture(textures[1], image.Rect(0, 0, 4, 4), func(*image.RGBA, error) {})
	})
	d.WriteTexture(textures[3], 0, image.Rect(0, 0, 8, 8), make([]byte, 8*8*4), 32) // waiting for a frame that never comes
	before := d.Stats()
	t.Logf("before Close: %+v", before)
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil { // twice is fine
		t.Fatal(err)
	}
	if d.Leaks() != 0 || d.objects != 0 {
		t.Errorf("Close left %d leaks and %d objects", d.Leaks(), d.objects)
	}
	for _, tex := range textures {
		tex.Destroy() // after Close: a no-op
	}
}

// Close after a failed Open leaves nothing either.
func TestWineFailedOpenLeavesNothing(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, level := range []uint32{0xFFFF, featureLevel11_0} {
		dev, err := openWith(Options{Logger: slog.New(&logRecorder{}), Getenv: env(nil)}, defaultEntryNames(), level)
		if err != nil {
			continue
		}
		if err := dev.Close(); err != nil || dev.Leaks() != 0 {
			t.Errorf("Close: %v, leaks %d", err, dev.Leaks())
		}
	}
}

// A second device in the same process, alongside the first and after it.
func TestWineSecondDeviceInTheSameProcess(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	open := func() *Device {
		d, err := Open(Options{Logger: slog.New(&logRecorder{}), Getenv: env(nil)})
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		return d
	}
	roundTrip := func(d *Device, seed byte) {
		tex, err := d.NewTexture(explicit.TextureDesc{Width: 9, Height: 9, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureCopySource})
		if err != nil {
			t.Fatal(err)
		}
		defer tex.Destroy()
		data := pattern(9, 9, 4, 36, seed)
		d.WriteTexture(tex, 0, image.Rect(0, 0, 9, 9), data, 36)
		samePixels(t, "round trip", readTexture(t, d, tex, image.Rect(0, 0, 9, 9)), expected(explicit.TextureFormatRGBA8Unorm, data, 9, 9, 36))
	}
	a, b := open(), open()
	roundTrip(a, 1)
	roundTrip(b, 2)
	if err := a.Close(); err != nil || a.Leaks() != 0 {
		t.Errorf("first Close: %v, leaks %d", err, a.Leaks())
	}
	roundTrip(b, 3) // the second one is not affected by the first one going
	c := open()
	roundTrip(c, 4)
	for _, d := range []*Device{b, c} {
		if err := d.Close(); err != nil || d.Leaks() != 0 {
			t.Errorf("Close: %v, leaks %d", err, d.Leaks())
		}
	}
}

// gpuHandle is a surface handle for the call that is not implemented yet.
func gpuHandle() gpu.SurfaceHandle { return gpu.SurfaceHandle{} }
