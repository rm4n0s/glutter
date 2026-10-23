package nativewindow

import (
	"image"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
)

// windowAPI is the part of the NDK's ANativeWindow that the presenter calls, so that the sequence
// of calls is plain Go that a test runs with a fake (the Android build binds it to libandroid.so,
// present_android.go).
type windowAPI interface {
	// setBuffersGeometry is ANativeWindow_setBuffersGeometry with the RGBA_8888 format; it returns
	// the status, 0 on success.
	setBuffersGeometry(window uintptr, width, height int) int
	// lock is ANativeWindow_lock: it returns the status and, on success, the buffer.
	lock(window uintptr) (status int, buf lockedBuffer)
	// unlockAndPost is ANativeWindow_unlockAndPost; it returns the status.
	unlockAndPost(window uintptr) int
}

// lockedBuffer is a window's buffer while it is locked: Mem holds Height rows of Stride pixels of
// four bytes.
type lockedBuffer struct {
	Mem                           []byte
	Stride, Width, Height, Format int
}

// windowPresenter is the Presenter over a windowAPI.
type windowPresenter struct {
	api windowAPI
	mu  sync.Mutex
	// size is the buffer size last set for each window.
	size map[uintptr][2]int
}

var _ Presenter = (*windowPresenter)(nil)

func newWindowPresenter(api windowAPI) *windowPresenter {
	return &windowPresenter{api: api, size: map[uintptr][2]int{}}
}

// Present sets the window's buffer size when it differs from the image's, locks the buffer, copies
// the frame into it and posts it. It fails with NativeWindowFailed, with the status in the
// metadata, when the window refuses a call: the window may be destroyed, or the buffer is not
// RGBA_8888.
func (p *windowPresenter) Present(h gpu.SurfaceHandle, img *image.RGBA) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	w, ht := img.Rect.Dx(), img.Rect.Dy()
	if p.size[h.Window] != [2]int{w, ht} {
		if rc := p.api.setBuffersGeometry(h.Window, w, ht); rc != 0 {
			return errors.New(tagNativeWindowFailed, "ANativeWindow_setBuffersGeometry failed", "status", rc, "width", w, "height", ht)
		}
		p.size[h.Window] = [2]int{w, ht}
	}
	rc, buf := p.api.lock(h.Window)
	if rc != 0 {
		return errors.New(tagNativeWindowFailed, "ANativeWindow_lock failed: the window may be destroyed", "status", rc)
	}
	if buf.Format != windowFormatRGBA8888 && buf.Format != windowFormatRGBX8888 {
		// Hand the buffer back untouched: drawing in another format would show garbage.
		_ = p.api.unlockAndPost(h.Window) // the format is the error to report
		return errors.New(tagNativeWindowFailed, "the window's buffer is not RGBA_8888", "format", buf.Format)
	}
	copyFrame(buf.Mem, buf.Stride, buf.Width, buf.Height, img)
	if rc := p.api.unlockAndPost(h.Window); rc != 0 {
		return errors.New(tagNativeWindowFailed, "ANativeWindow_unlockAndPost failed", "status", rc)
	}
	return nil
}

// Release forgets the buffer size of the window.
func (p *windowPresenter) Release(h gpu.SurfaceHandle) {
	p.mu.Lock()
	delete(p.size, h.Window)
	p.mu.Unlock()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagNativeWindowFailed = "NativeWindowFailed"
)
