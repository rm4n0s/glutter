//go:build android

package nativewindow

import (
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// libAPI is the windowAPI over libandroid.so (SPEC §24.4).
type libAPI struct {
	lib *dl.Lib

	setGeometry  func(window uintptr, width, height, format int32) int32
	lockFn       func(window, out, inOutDirty uintptr) int32
	unlockPostFn func(window uintptr) int32
	// buf is C memory for the ANativeWindow_Buffer that lock fills in.
	buf uintptr
}

var _ windowAPI = (*libAPI)(nil)

// NewPresenter loads libandroid.so and binds the window functions. It fails with
// LibraryLoadFailed when the library cannot be loaded and with SymbolMissing when it lacks a
// function.
//
// Flutter: no counterpart
func NewPresenter() (Presenter, error) {
	lib, err := dl.Open("libandroid.so")
	if err != nil {
		return nil, err
	}
	a := &libAPI{lib: lib}
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&a.setGeometry, "ANativeWindow_setBuffersGeometry"},
		{&a.lockFn, "ANativeWindow_lock"},
		{&a.unlockPostFn, "ANativeWindow_unlockAndPost"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	a.buf = dl.Calloc(1, bufferSize)
	return newWindowPresenter(a), nil
}

func (a *libAPI) setBuffersGeometry(window uintptr, width, height int) int {
	return int(a.setGeometry(window, int32(width), int32(height), windowFormatRGBA8888))
}

func (a *libAPI) lock(window uintptr) (int, lockedBuffer) {
	if rc := a.lockFn(window, a.buf, 0); rc != 0 {
		return int(rc), lockedBuffer{}
	}
	b := (*windowBuffer)(dl.Pointer(a.buf))
	return 0, lockedBuffer{
		Mem:    dl.Bytes(b.Bits, int(b.Stride)*int(b.Height)*4),
		Stride: int(b.Stride), Width: int(b.Width), Height: int(b.Height), Format: int(b.Format),
	}
}

func (a *libAPI) unlockAndPost(window uintptr) int { return int(a.unlockPostFn(window)) }
