//go:build windows && glutter_wine

package gdiwindow

import (
	"image"
	"image/color"
	"runtime"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/d3d12"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests draw into a real window under Wine on a headless X server and read the pixels back
// (SPEC §25.13): go test -tags glutter_wine -exec scripts/wine-exec.sh -p 1 ./internal/gpu/gdiwindow

// makeWindow makes a visible window of the size that has no painting of its own.
func makeWindow(t *testing.T, w, h int32) uintptr {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	instance, err := win32.GetModuleHandle("")
	if err != nil {
		t.Fatal(err)
	}
	class := "GdiWindowTest"
	name, err := win32.UTF16PtrFromString(class)
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = win32.UnregisterClass(class, instance) })
	// A popup, so that the client area is the window.
	hwnd, err := win32.CreateWindowEx(0, class, "t", win32.WSPopup|win32.WSVisible, 10, 10, w, h, 0, 0, instance, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = win32.DestroyWindow(hwnd) })
	return hwnd
}

// pixel reads a pixel of the window as RGBA.
func pixel(t *testing.T, hwnd uintptr, x, y int32) color.RGBA {
	t.Helper()
	dc, err := win32.GetDC(hwnd)
	if err != nil {
		t.Fatal(err)
	}
	defer win32.ReleaseDC(hwnd, dc)
	c := win32.GetPixel(dc, x, y)
	if c == 0xFFFFFFFF {
		t.Fatalf("pixel %d,%d is outside the window", x, y)
	}
	return color.RGBA{R: uint8(c), G: uint8(c >> 8), B: uint8(c >> 16), A: 255}
}

func TestPresentShowsTheFrameAndTheDamageInTheWindow(t *testing.T) {
	hwnd := makeWindow(t, 120, 90)
	p := NewPresenter()
	h := gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Window: hwnd}
	defer p.Release(h)

	red := color.RGBA{R: 255, A: 255}
	blue := color.RGBA{B: 255, G: 128, A: 255}
	img := solid(120, 90, red)
	if err := p.Present(h, img, gpu.Damage{Full: true}); err != nil {
		t.Fatal(err)
	}
	for _, pt := range [][2]int32{{0, 0}, {119, 0}, {0, 89}, {60, 45}, {119, 89}} {
		if got := pixel(t, hwnd, pt[0], pt[1]); got != red {
			t.Errorf("after the whole frame, pixel %v = %v, want red", pt, got)
		}
	}

	// A damage rectangle in the middle: its pixels change, and the rows above and below it, and the
	// columns beside it, stay (the rectangle's rows are sent as a block of their own).
	for y := 30; y < 50; y++ {
		for x := 40; x < 80; x++ {
			img.SetRGBA(x, y, blue)
		}
	}
	if err := p.Present(h, img, gpu.Damage{Rects: []image.Rectangle{image.Rect(40, 30, 80, 50)}}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		x, y int32
		want color.RGBA
	}{
		{"inside, top left", 40, 30, blue}, {"inside, bottom right", 79, 49, blue}, {"inside", 60, 40, blue},
		{"left of it", 39, 40, red}, {"right of it", 80, 40, red}, {"above it", 60, 29, red}, {"below it", 60, 50, red}, {"far away", 5, 80, red},
	}
	for _, tt := range tests {
		if got := pixel(t, hwnd, tt.x, tt.y); got != tt.want {
			t.Errorf("%s: pixel %d,%d = %v, want %v", tt.name, tt.x, tt.y, got, tt.want)
		}
	}

	// A new size shows all of the new frame.
	if err := win32.SetWindowPos(hwnd, 0, 0, 0, 60, 40, win32.SWPNoMove|win32.SWPNoZOrder|win32.SWPNoActivate); err != nil {
		t.Fatal(err)
	}
	green := color.RGBA{G: 255, A: 255}
	if err := p.Present(h, solid(60, 40, green), gpu.Damage{}); err != nil {
		t.Fatal(err)
	}
	if got := pixel(t, hwnd, 59, 39); got != green {
		t.Errorf("after a new size, pixel 59,39 = %v, want green", got)
	}
}

func TestExposureRepaintsTheKeptFrameInTheWindow(t *testing.T) {
	hwnd := makeWindow(t, 80, 60)
	p := NewPresenter()
	h := gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Window: hwnd}
	defer p.Release(h)
	if err := p.Present(h, solid(80, 60, color.RGBA{G: 200, A: 255}), gpu.Damage{Full: true}); err != nil {
		t.Fatal(err)
	}
	paint := win32.PaintHandler(hwnd)
	if paint == nil {
		t.Fatal("the presenter did not set the paint handler")
	}
	// Paint something else over the window, as another window that was uncovered would leave.
	dc, _ := win32.GetDC(hwnd)
	info := win32.BitmapInfo{Header: win32.BitmapInfoHeader{Size: 40, Width: 1, Height: -1, Planes: 1, BitCount: 32}}
	for y := int32(0); y < 60; y++ {
		for x := int32(0); x < 80; x++ {
			win32.StretchDIBits(dc, x, y, 1, 1, 0, 0, 1, 1, []byte{255, 0, 0, 0}, &info, win32.DIBRGBColors, win32.SrcCopy)
		}
	}
	win32.ReleaseDC(hwnd, dc)
	if got := pixel(t, hwnd, 10, 10); got != (color.RGBA{B: 255, A: 255}) {
		t.Fatalf("the window was not painted over: %v", got)
	}
	// An exposure: the invalid region is the whole window.
	user32, err := win32.LoadSystemDLL("user32.dll")
	if err != nil {
		t.Fatal(err)
	}
	invalidate, err := user32.Proc("InvalidateRect")
	if err != nil {
		t.Fatal(err)
	}
	if r, _ := invalidate.Call(hwnd, 0, 0); r == 0 {
		t.Fatal("InvalidateRect failed")
	}
	paint(hwnd) // what the window procedure does on WM_PAINT
	if got := pixel(t, hwnd, 10, 10); got != (color.RGBA{G: 200, A: 255}) {
		t.Errorf("after the exposure the pixel is %v, want the kept frame", got)
	}
}

// TestGDIDrawsIntoAWindowThatHadASwapChain is the check of SPEC §22: after d3d12 made a flip-model swap chain
// for a window and released it, `soft` draws into the same window with GDI, which is what auto does when d3d12
// fails after its swap chain was made. Under Wine the pixels of GDI are there. Whether they are on Windows, where
// DWM may keep showing nothing for a window that had a flip-model swap chain, is the manual check of step 28.10:
// if not, the surface becomes a child window that is made again (platform.SurfaceRenewer, SPEC §25.5).
func TestGDIDrawsIntoAWindowThatHadASwapChain(t *testing.T) {
	tests := []struct {
		name        string
		closeDevice bool
	}{
		{"after the device was closed, as the chooser leaves it", true},
		{"with the device still open", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hwnd := makeWindow(t, 120, 90)
			h := gpu.SurfaceHandle{Kind: gpu.SurfaceKindWin32, Window: hwnd}
			dev, err := d3d12.Open(d3d12.Options{Getenv: func(string) string { return "" }})
			if err != nil {
				t.Fatalf("d3d12.Open: %+v", err)
			}
			sc, err := dev.NewSwapchain(h)
			if err != nil {
				_ = dev.Close()
				t.Fatalf("NewSwapchain: %+v", err)
			}
			for i := 0; i < 3; i++ { // presented a few times, as a first frame and the frames after it are
				e, err := dev.Begin()
				if err != nil {
					t.Fatal(err)
				}
				tex, err := sc.Acquire()
				if err != nil {
					t.Fatal(err)
				}
				e.BeginPass(tex, explicit.LoadClear, ui.Color{G: 1, A: 1}).End()
				if err := e.Submit(); err != nil {
					t.Fatal(err)
				}
				if err := sc.Present(nil); err != nil {
					t.Fatal(err)
				}
			}
			sc.Destroy()
			if tt.closeDevice {
				if err := dev.Close(); err != nil {
					t.Fatal(err)
				}
			} else {
				defer func() { _ = dev.Close() }()
			}
			p := NewPresenter()
			defer p.Release(h)
			red := color.RGBA{R: 255, A: 255}
			if err := p.Present(h, solid(120, 90, red), gpu.Damage{Full: true}); err != nil {
				t.Fatal(err)
			}
			for _, pt := range [][2]int32{{0, 0}, {119, 0}, {60, 45}, {0, 89}, {119, 89}} {
				if got := pixel(t, hwnd, pt[0], pt[1]); got != red {
					t.Errorf("GDI after a swap chain: pixel %v = %v, want red", pt, got)
				}
			}
		})
	}
}
