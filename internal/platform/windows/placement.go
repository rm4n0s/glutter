package windows

import "github.com/rm4n0s/glutter/internal/platform/windows/win32"

// The places of the windows after the first (SPEC §25.4): a dialog opens centered over its owner, and
// a window opens a step down and to the right of the one before it, as Windows' own default place
// (CW_USEDEFAULT) does on a desktop with a shell, so that the windows of an app do not pile up at one
// corner. All values are physical pixels of the screen.

// centerOn is the upper-left corner of a window of the outer size w by h whose center is the center of
// the owner's rectangle, moved to lie inside the work area of the monitor when the work area is large
// enough (a window larger than the work area keeps its upper-left corner at the work area's).
func centerOn(owner win32.Rect, w, h int32, work win32.Rect) (x, y int32) {
	x = owner.Left + (owner.Right-owner.Left-w)/2
	y = owner.Top + (owner.Bottom-owner.Top-h)/2
	return clampInto(x, w, work.Left, work.Right), clampInto(y, h, work.Top, work.Bottom)
}

// clampInto moves the start of a span of the size n so that it lies between lo and hi; a span that is
// larger than the room starts at lo.
func clampInto(start, n, lo, hi int32) int32 {
	if start+n > hi {
		start = hi - n
	}
	if start < lo {
		start = lo
	}
	return start
}

// cascadeFrom is the upper-left corner of a new window of the outer size w by h one step below and to
// the right of the previous window's corner; when the new window would not fit the work area, the
// cascade starts again at its corner.
func cascadeFrom(prev win32.Rect, step, w, h int32, work win32.Rect) (x, y int32) {
	x, y = prev.Left+step, prev.Top+step
	if x < work.Left || y < work.Top || x+w > work.Right || y+h > work.Bottom {
		return work.Left, work.Top
	}
	return x, y
}
