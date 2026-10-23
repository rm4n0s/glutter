package nativewindow

import (
	"image"
	"image/color"
	"testing"

	"github.com/rm4n0s/errors"
)

// fakeWindow is a windowAPI over a buffer in Go memory.
type fakeWindow struct {
	geometry       [][2]int
	locks, unlocks int
	stride         int
	format         int
	mem            []byte
	failGeometry   int
	failLock       int
	failUnlock     int
	bufW, bufH     int
}

func (f *fakeWindow) setBuffersGeometry(_ uintptr, w, h int) int {
	f.geometry = append(f.geometry, [2]int{w, h})
	f.bufW, f.bufH = w, h
	return f.failGeometry
}

func (f *fakeWindow) lock(uintptr) (int, lockedBuffer) {
	f.locks++
	if f.failLock != 0 {
		return f.failLock, lockedBuffer{}
	}
	stride := f.stride
	if stride == 0 {
		stride = f.bufW
	}
	f.mem = make([]byte, stride*f.bufH*4)
	format := f.format
	if format == 0 {
		format = windowFormatRGBA8888
	}
	return 0, lockedBuffer{Mem: f.mem, Stride: stride, Width: f.bufW, Height: f.bufH, Format: format}
}

func (f *fakeWindow) unlockAndPost(uintptr) int {
	f.unlocks++
	return f.failUnlock
}

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestPresentCopiesTheFrameRowByRow(t *testing.T) {
	// A stride wider than the width: the window pads its rows.
	fw := &fakeWindow{stride: 6}
	p := newWindowPresenter(fw)
	img := solid(4, 2, color.RGBA{1, 2, 3, 255})
	img.SetRGBA(3, 1, color.RGBA{9, 8, 7, 255})
	if err := p.Present(androidHandle, img); err != nil {
		t.Fatal(err)
	}
	if len(fw.geometry) != 1 || fw.geometry[0] != [2]int{4, 2} || fw.locks != 1 || fw.unlocks != 1 {
		t.Fatalf("geometry %v, locks %d, unlocks %d", fw.geometry, fw.locks, fw.unlocks)
	}
	row1 := fw.mem[6*4:]
	if got := [4]byte{row1[12], row1[13], row1[14], row1[15]}; got != [4]byte{9, 8, 7, 255} {
		t.Errorf("pixel (3,1) = %v", got)
	}
	if got := fw.mem[4*4 : 4*4+4]; got[0] != 0 || got[3] != 0 {
		t.Errorf("the padding of row 0 was written: %v", got)
	}
	// The same size again does not set the geometry again.
	if err := p.Present(androidHandle, img); err != nil {
		t.Fatal(err)
	}
	if len(fw.geometry) != 1 {
		t.Errorf("geometry set again: %v", fw.geometry)
	}
	// A new size does, and a released window does too.
	if err := p.Present(androidHandle, solid(5, 3, color.RGBA{})); err != nil {
		t.Fatal(err)
	}
	p.Release(androidHandle)
	if err := p.Present(androidHandle, solid(5, 3, color.RGBA{})); err != nil {
		t.Fatal(err)
	}
	if len(fw.geometry) != 3 {
		t.Errorf("geometry calls %v, want 3", fw.geometry)
	}
}

func TestPresentFailures(t *testing.T) {
	tests := []struct {
		name  string
		fw    *fakeWindow
		meta  string
		value any
		posts int
	}{
		{"geometry", &fakeWindow{failGeometry: -22}, "status", -22, 0},
		{"lock: the window is gone", &fakeWindow{failLock: -19}, "status", -19, 0},
		{"another format", &fakeWindow{format: 4}, "format", 4, 1},
		{"unlock", &fakeWindow{failUnlock: -1}, "status", -1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newWindowPresenter(tt.fw).Present(androidHandle, solid(2, 2, color.RGBA{}))
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute("windowPresenter.Present."+tagNativeWindowFailed) {
				t.Fatalf("got %v", err)
			}
			if appErr.Metadata[tt.meta] != tt.value {
				t.Errorf("metadata %v, want %s=%v", appErr.Metadata, tt.meta, tt.value)
			}
			if tt.fw.unlocks != tt.posts {
				t.Errorf("unlockAndPost called %d times, want %d (a locked buffer must be handed back)", tt.fw.unlocks, tt.posts)
			}
		})
	}
}

func TestCopyFrame(t *testing.T) {
	tests := []struct {
		name                          string
		stride, width, height, iw, ih int
		wantRows                      int
	}{
		{"same size", 4, 4, 3, 4, 3, 3},
		{"the image is bigger", 4, 4, 3, 6, 5, 3},
		{"the buffer is bigger", 8, 8, 6, 4, 3, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := solid(tt.iw, tt.ih, color.RGBA{5, 6, 7, 255})
			dst := make([]byte, tt.stride*tt.height*4)
			copyFrame(dst, tt.stride, tt.width, tt.height, img)
			rows := 0
			for y := 0; y < tt.height; y++ {
				if dst[y*tt.stride*4+3] == 255 {
					rows++
				}
			}
			if rows != tt.wantRows {
				t.Errorf("rows copied = %d, want %d", rows, tt.wantRows)
			}
		})
	}
}
