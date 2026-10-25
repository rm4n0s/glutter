package windows

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// wp makes the wParam of a wheel message: the delta in the high word, the key states in the low.
func wp(delta int16, keys uintptr) uintptr { return uintptr(uint16(delta))<<16 | keys }

func TestWheelScroll(t *testing.T) {
	const w, h = 640.0, 480.0
	tests := []struct {
		name       string
		horizontal bool
		wparam     uintptr
		lines      uint32
		dx, dy     float64
	}{
		// window_unittests.cc VerticalScroll and HorizontalScroll: a delta of 10 is 10/120 of a notch,
		// and the vertical wheel is the other way round.
		{"VerticalScroll: 10 is a twelfth of a notch, up", false, wp(10, 0), 3, 0, -10.0 / 120 * 100},
		{"HorizontalScroll: 10 is a twelfth of a notch, right", true, wp(10, 0), 3, 10.0 / 120 * 100, 0},
		{"one notch away from the user scrolls up", false, wp(120, 0), 3, 0, -100},
		{"one notch towards the user scrolls down", false, wp(-120, 0), 3, 0, 100},
		{"one notch to the right", true, wp(120, 0), 3, 100, 0},
		{"one notch to the left", true, wp(-120, 0), 3, -100, 0},
		{"two notches", false, wp(-240, 0), 3, 0, 200},
		{"the key states in the low word are not read", false, wp(-120, win32.MKControl|win32.MKShift), 3, 0, 100},
		{"a precision touchpad: a delta of 12", false, wp(-12, 0), 3, 0, 10},
		{"a precision touchpad: a delta of 1", false, wp(-1, 0), 3, 0, 100.0 / 120},
		{"a free-spinning wheel: 30", true, wp(30, 0), 3, 25, 0},
		{"1 line a notch", false, wp(-120, 0), 1, 0, 100.0 / 3},
		{"10 lines a notch", false, wp(-120, 0), 10, 0, 1000.0 / 3},
		{"0 lines scrolls nothing", false, wp(-120, 0), 0, 0, 0},
		{"WHEEL_PAGESCROLL: a page of the view, vertical", false, wp(-120, 0), win32.WheelPageScroll, 0, h},
		{"WHEEL_PAGESCROLL: a page of the view, horizontal", true, wp(120, 0), win32.WheelPageScroll, w, 0},
		{"WHEEL_PAGESCROLL: half a notch is half a page", false, wp(60, 0), win32.WheelPageScroll, 0, -h / 2},
		{"the high bits of a 64-bit wParam are not read", false, 0xFFFF0000_FF88_0000, 3, 0, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dx, dy := wheelScroll(tt.horizontal, tt.wparam, tt.lines, w, h)
			if !near(dx, tt.dx) || !near(dy, tt.dy) {
				t.Errorf("wheelScroll = %v, %v; want %v, %v", dx, dy, tt.dx, tt.dy)
			}
		})
	}
}

// TestWheelNotchPixels is the multiplier of FlutterWindow::UpdateScrollOffsetMultiplier (lines
// times 100/3, the rule of Chromium), which flutter_window_unittests.cc OnScrollCallsGetScrollOffsetMultiplier
// checks through a mock.
func TestWheelNotchPixels(t *testing.T) {
	tests := []struct {
		name  string
		lines uint32
		page  float64
		want  float64
	}{
		{"the default of 3 lines is 100 pixels", defaultWheelLines, 480, 100},
		{"1 line", 1, 480, 100.0 / 3},
		{"6 lines", 6, 480, 200},
		{"a page", win32.WheelPageScroll, 480, 480},
		{"a page of an empty view", win32.WheelPageScroll, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wheelNotchPixels(tt.lines, tt.page); !near(got, tt.want) {
				t.Errorf("wheelNotchPixels = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWheelParam(t *testing.T) {
	tests := []struct {
		name   string
		scroll float64
		lines  uint32
		page   float64
		want   int16
	}{
		{"nothing", 0, 3, 480, 0},
		{"a notch", 100, 3, 480, 120},
		{"a notch the other way", -100, 3, 480, -120},
		{"three notches", 300, 3, 480, 360},
		{"a fraction is rounded", 50.4, 3, 480, 60},
		{"less than a unit is one unit", 0.2, 3, 480, 1},
		{"less than a unit the other way is minus one", -0.2, 3, 480, -1},
		{"1 line", 100.0 / 3, 1, 480, 120},
		{"a page", 480, win32.WheelPageScroll, 480, 120},
		{"too much is cut", 1e9, 3, 480, 32767},
		{"too much the other way is cut", -1e9, 3, 480, -32768},
		{"0 lines is no scrolling", 100, 0, 480, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wheelParam(tt.scroll, tt.lines, tt.page)
			if d := wheelDelta(got); d != tt.want {
				t.Errorf("wheelParam = delta %d, want %d", d, tt.want)
			}
			if got&0xFFFF != 0 {
				t.Errorf("wheelParam has key states: %#x", got)
			}
		})
	}
}
