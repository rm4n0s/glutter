package win32

import (
	"testing"
	"unsafe"
)

// TestInputStructSizes checks the sizes that the Windows SDK gives the structures of the pointer
// input on a 64-bit target (amd64 and arm64), worked out from the field lists of winuser.h. It
// stands in for the layout test (TestLayoutAgainstTheHeaders) on a machine without the mingw-w64
// compiler; the layout test checks every offset against the headers when it can run.
func TestInputStructSizes(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("the sizes are those of a 64-bit target")
	}
	tests := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"TRACKMOUSEEVENT", unsafe.Sizeof(TrackMouseEventData{}), 24},
		{"POINTER_INFO", unsafe.Sizeof(PointerInfo{}), 96},
		{"POINTER_TOUCH_INFO", unsafe.Sizeof(PointerTouchInfo{}), 144},
		{"POINTER_PEN_INFO", unsafe.Sizeof(PointerPenInfo{}), 120},
		{"MOUSEINPUT", unsafe.Sizeof(MouseInput{}), 32},
		{"INPUT", unsafe.Sizeof(Input{}), 40},
		{"INPUT (keyboard)", unsafe.Sizeof(KeyInput{}), 40},
		{"KEYBDINPUT", unsafe.Sizeof(KeybdInput{}), 24},
		{"CURSORINFO", unsafe.Sizeof(CursorInfo{}), 24},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("sizeof(%s) = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
	// The offsets that the C compiler would put the members at.
	offsets := []struct {
		name      string
		got, want uintptr
	}{
		{"POINTER_INFO.sourceDevice", unsafe.Offsetof(PointerInfo{}.SourceDevice), 16},
		{"POINTER_INFO.ptPixelLocation", unsafe.Offsetof(PointerInfo{}.PixelLocation), 32},
		{"POINTER_INFO.dwTime", unsafe.Offsetof(PointerInfo{}.Time), 64},
		{"POINTER_INFO.PerformanceCount", unsafe.Offsetof(PointerInfo{}.PerformanceCount), 80},
		{"POINTER_TOUCH_INFO.rcContact", unsafe.Offsetof(PointerTouchInfo{}.Contact), 104},
		{"POINTER_TOUCH_INFO.pressure", unsafe.Offsetof(PointerTouchInfo{}.Pressure), 140},
		{"POINTER_PEN_INFO.pressure", unsafe.Offsetof(PointerPenInfo{}.Pressure), 104},
		{"POINTER_PEN_INFO.tiltY", unsafe.Offsetof(PointerPenInfo{}.TiltY), 116},
		{"INPUT.mi", unsafe.Offsetof(Input{}.Mouse), 8},
		{"MOUSEINPUT.dwExtraInfo", unsafe.Offsetof(MouseInput{}.ExtraInfo), 24},
		{"INPUT.ki", unsafe.Offsetof(KeyInput{}.Key), 8},
		{"KEYBDINPUT.dwFlags", unsafe.Offsetof(KeybdInput{}.Flags), 4},
		{"KEYBDINPUT.dwExtraInfo", unsafe.Offsetof(KeybdInput{}.ExtraInfo), 16},
		{"CURSORINFO.hCursor", unsafe.Offsetof(CursorInfo{}.Cursor), 8},
	}
	for _, tt := range offsets {
		if tt.got != tt.want {
			t.Errorf("offsetof(%s) = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}
