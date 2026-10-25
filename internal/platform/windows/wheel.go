package windows

import (
	"math"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// This file is the plain-Go part of the wheel (SPEC §25.6): WM_MOUSEWHEEL and WM_MOUSEHWHEEL turned
// into scroll deltas in physical pixels.

// defaultWheelLines is the lines a notch scrolls when the system's setting cannot be read.
//
// Flutter: shell/platform/windows/flutter_window.cc kLinesPerScrollWindowsDefault
const defaultWheelLines = 3

// wheelNotchPixels is the amount that one notch of the wheel scrolls, in physical pixels: the lines
// of the system's setting times 100/3 (so 100 pixels for the default 3 lines; the rule of Chromium
// that Flutter's embedder follows). WHEEL_PAGESCROLL scrolls a page, the size of the view along the
// axis of the wheel.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::UpdateScrollOffsetMultiplier
func wheelNotchPixels(lines uint32, page float64) float64 {
	if lines == win32.WheelPageScroll {
		return page
	}
	return float64(lines) * 100.0 / 3.0
}

// wheelDelta reads the signed wheel delta of the high word of the wParam of a wheel message.
func wheelDelta(wparam uintptr) int16 { return int16((wparam >> 16) & 0xFFFF) }

// wheelScroll turns a wheel message into the scroll delta of the pointer data: the delta divided by
// WHEEL_DELTA (120, one notch; a precision touchpad sends small ones, which scroll in proportion)
// times the amount of a notch. A vertical wheel is positive away from the user, which scrolls up,
// so its scroll delta is the negative; a horizontal one is positive to the right, as the scroll
// delta is. width and height are the size of the view in physical pixels, for WHEEL_PAGESCROLL.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_MOUSEWHEEL, WM_MOUSEHWHEEL)
func wheelScroll(horizontal bool, wparam uintptr, lines uint32, width, height float64) (dx, dy float64) {
	notches := float64(wheelDelta(wparam)) / win32.WheelDelta
	if horizontal {
		return notches * wheelNotchPixels(lines, width), 0
	}
	return 0, -notches * wheelNotchPixels(lines, height)
}

// wheelParam makes the wParam of a wheel message of the pixels to scroll, for the scripted events
// and the tests: the delta of the nearest whole unit of 1/120 of a notch (at least 1 for a scroll
// that is not zero), in the high word, with no key states. dx or dy are in physical pixels, and
// have the sign of the scroll delta.
func wheelParam(scroll float64, lines uint32, page float64) uintptr {
	notch := wheelNotchPixels(lines, page)
	if scroll == 0 || notch == 0 {
		return 0
	}
	units := scroll / notch * win32.WheelDelta
	n := int(math.Round(units))
	if n == 0 {
		if scroll > 0 {
			n = 1
		} else {
			n = -1
		}
	}
	if n > 32767 {
		n = 32767
	}
	if n < -32768 {
		n = -32768
	}
	return uintptr(uint16(int16(n))) << 16
}
