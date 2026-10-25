package win32

import "sync"

// PaintFunc paints a window that was exposed (WM_PAINT): it does BeginPaint, draws what the
// renderer kept of the last frame, and does EndPaint.
//
// Flutter: no counterpart
type PaintFunc func(hwnd uintptr)

// paintHandlers holds the PaintFunc of each window. The renderer that puts the soft frame into a
// window with GDI (internal/gpu/gdiwindow) sets its function here when it attaches to the window,
// and the platform's window procedure calls it on WM_PAINT, so that an exposure repaints the kept
// frame without a new one (SPEC §25.5). The table is process-wide, like the window procedure
// that reads it.
var paintHandlers = struct {
	mu sync.Mutex
	m  map[uintptr]PaintFunc
}{m: map[uintptr]PaintFunc{}}

// SetPaintHandler sets the function that paints the window on WM_PAINT; a nil function removes
// it.
//
// Flutter: no counterpart
func SetPaintHandler(hwnd uintptr, fn PaintFunc) {
	paintHandlers.mu.Lock()
	defer paintHandlers.mu.Unlock()
	if fn == nil {
		delete(paintHandlers.m, hwnd)
		return
	}
	paintHandlers.m[hwnd] = fn
}

// PaintHandler is the function that paints the window on WM_PAINT, or nil when there is none.
//
// Flutter: no counterpart
func PaintHandler(hwnd uintptr) PaintFunc {
	paintHandlers.mu.Lock()
	defer paintHandlers.mu.Unlock()
	return paintHandlers.m[hwnd]
}
