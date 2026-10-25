package windows

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// lp makes the lParam of a mouse message from a client place.
func lp(x, y int32) uintptr { return packPoint(x, y) }

// xbutton makes the wParam of WM_XBUTTON* with the button in the high word.
func xbutton(which uintptr, keys uintptr) uintptr { return which<<16 | keys }

func TestMouseTranslator(t *testing.T) {
	type call struct {
		name   string // "move", "down", "up", "leave", "capture"
		msg    uint32
		wparam uintptr
		lparam uintptr
		other  bool // for "capture": the capture went to another window
		fx     mouseEffects
	}
	tests := []struct {
		name  string
		calls []call
		want  []ev
	}{
		{"a click: the first move asks for the leave, the button takes the capture, the up gives it back", []call{
			{name: "move", lparam: lp(10, 20), fx: mouseEffects{trackLeave: true}},
			{name: "down", msg: win32.WMLButtonDown, wparam: win32.MKLButton, lparam: lp(10, 20), fx: mouseEffects{setCapture: true}},
			{name: "up", msg: win32.WMLButtonUp, lparam: lp(10, 20), fx: mouseEffects{releaseCapture: true}},
			{name: "capture", other: true}, // the release we made
		}, []ev{
			{change: cAdd, kind: kMouse, x: 10, y: 20},
			{change: cHover, kind: kMouse, x: 10, y: 20},
			{change: cDown, kind: kMouse, x: 10, y: 20, buttons: buttonPrimary},
			{change: cUp, kind: kMouse, x: 10, y: 20},
		}},
		{"a drag that leaves the window goes on with negative places and ends outside with an up", []call{
			{name: "move", lparam: lp(5, 5), fx: mouseEffects{trackLeave: true}},
			{name: "down", msg: win32.WMLButtonDown, wparam: win32.MKLButton, lparam: lp(5, 5), fx: mouseEffects{setCapture: true}},
			{name: "move", wparam: win32.MKLButton, lparam: lp(-30, 400)},
			{name: "move", wparam: win32.MKLButton, lparam: lp(-31, 401)},
			{name: "up", msg: win32.WMLButtonUp, lparam: lp(-31, 401), fx: mouseEffects{releaseCapture: true}},
			{name: "leave"}, // Windows sends the leave now that the capture is gone and the pointer is outside
		}, []ev{
			{change: cAdd, kind: kMouse, x: 5, y: 5},
			{change: cHover, kind: kMouse, x: 5, y: 5},
			{change: cDown, kind: kMouse, x: 5, y: 5, buttons: buttonPrimary},
			{change: cMove, kind: kMouse, x: -30, y: 400, dx: -35, dy: 395, buttons: buttonPrimary},
			{change: cMove, kind: kMouse, x: -31, y: 401, dx: -1, dy: 1, buttons: buttonPrimary},
			{change: cUp, kind: kMouse, x: -31, y: 401, dx: 0, dy: 0},
			{change: cRemove, kind: kMouse, x: -31, y: 401},
		}},
		{"WM_MOUSELEAVE removes the pointer where it was (MockWindow.MouseLeave)", []call{
			{name: "move", lparam: lp(10, 20), fx: mouseEffects{trackLeave: true}},
			{name: "leave"},
			{name: "move", lparam: lp(11, 21), fx: mouseEffects{trackLeave: true}}, // asks again after the leave
		}, []ev{
			{change: cAdd, kind: kMouse, x: 10, y: 20},
			{change: cHover, kind: kMouse, x: 10, y: 20},
			{change: cRemove, kind: kMouse, x: 10, y: 20},
			{change: cAdd, kind: kMouse, x: 11, y: 21},
			{change: cHover, kind: kMouse, x: 11, y: 21},
		}},
		{"a leave while a button is down without the capture lifts it first", []call{
			{name: "move", wparam: win32.MKRButton, lparam: lp(2, 3), fx: mouseEffects{trackLeave: true}},
			{name: "leave"},
		}, []ev{
			{change: cAdd, kind: kMouse, x: 2, y: 3},
			{change: cDown, kind: kMouse, x: 2, y: 3, buttons: buttonSecondary},
			{change: cUp, kind: kMouse, x: 2, y: 3, synth: true},
			{change: cRemove, kind: kMouse, x: 2, y: 3},
		}},
		{"a leave during a drag waits for the last button up (Wine sends it at once)", []call{
			{name: "down", msg: win32.WMLButtonDown, wparam: win32.MKLButton, lparam: lp(2, 3), fx: mouseEffects{trackLeave: true, setCapture: true}},
			{name: "move", wparam: win32.MKLButton, lparam: lp(-50, 3)},
			{name: "leave"},
			{name: "move", wparam: win32.MKLButton, lparam: lp(-60, 3)},
			{name: "up", msg: win32.WMLButtonUp, lparam: lp(-60, 3), fx: mouseEffects{releaseCapture: true}},
			{name: "move", lparam: lp(4, 4), fx: mouseEffects{trackLeave: true}},
		}, []ev{
			{change: cAdd, kind: kMouse, x: 2, y: 3},
			{change: cDown, kind: kMouse, x: 2, y: 3, buttons: buttonPrimary},
			{change: cMove, kind: kMouse, x: -50, y: 3, dx: -52, buttons: buttonPrimary},
			{change: cMove, kind: kMouse, x: -60, y: 3, dx: -10, buttons: buttonPrimary},
			{change: cUp, kind: kMouse, x: -60, y: 3},
			{change: cRemove, kind: kMouse, x: -60, y: 3},
			{change: cAdd, kind: kMouse, x: 4, y: 4},
			{change: cHover, kind: kMouse, x: 4, y: 4},
		}},
		{"a leave that waited also runs when the capture goes to another window", []call{
			{name: "down", msg: win32.WMLButtonDown, wparam: win32.MKLButton, lparam: lp(2, 3), fx: mouseEffects{trackLeave: true, setCapture: true}},
			{name: "leave"},
			{name: "capture", other: true},
		}, []ev{
			{change: cAdd, kind: kMouse, x: 2, y: 3},
			{change: cDown, kind: kMouse, x: 2, y: 3, buttons: buttonPrimary},
			{change: cUp, kind: kMouse, x: 2, y: 3, synth: true},
			{change: cRemove, kind: kMouse, x: 2, y: 3},
		}},
		{"two buttons: the capture is taken once and given back with the last", []call{
			{name: "down", msg: win32.WMLButtonDown, wparam: win32.MKLButton, lparam: lp(1, 1), fx: mouseEffects{trackLeave: true, setCapture: true}},
			{name: "down", msg: win32.WMMButtonDown, wparam: win32.MKLButton | win32.MKMButton, lparam: lp(1, 1)},
			{name: "up", msg: win32.WMLButtonUp, wparam: win32.MKMButton, lparam: lp(1, 1)},
			{name: "up", msg: win32.WMMButtonUp, lparam: lp(1, 1), fx: mouseEffects{releaseCapture: true}},
		}, []ev{
			{change: cAdd, kind: kMouse, x: 1, y: 1},
			{change: cDown, kind: kMouse, x: 1, y: 1, buttons: buttonPrimary},
			{change: cMove, kind: kMouse, x: 1, y: 1, buttons: buttonPrimary | buttonMiddle},
			{change: cMove, kind: kMouse, x: 1, y: 1, buttons: buttonMiddle},
			{change: cUp, kind: kMouse, x: 1, y: 1},
		}},
		{"the back and the forward buttons", []call{
			{name: "down", msg: win32.WMXButtonDown, wparam: xbutton(win32.XButton1, win32.MKXButton1), lparam: lp(4, 4), fx: mouseEffects{trackLeave: true, setCapture: true}},
			{name: "up", msg: win32.WMXButtonUp, wparam: xbutton(win32.XButton1, 0), lparam: lp(4, 4), fx: mouseEffects{releaseCapture: true}},
			{name: "down", msg: win32.WMXButtonDown, wparam: xbutton(win32.XButton2, win32.MKXButton2), lparam: lp(4, 4), fx: mouseEffects{setCapture: true}},
			{name: "up", msg: win32.WMXButtonUp, wparam: xbutton(win32.XButton2, 0), lparam: lp(4, 4), fx: mouseEffects{releaseCapture: true}},
		}, []ev{
			{change: cAdd, kind: kMouse, x: 4, y: 4},
			{change: cDown, kind: kMouse, x: 4, y: 4, buttons: buttonBack},
			{change: cUp, kind: kMouse, x: 4, y: 4},
			{change: cDown, kind: kMouse, x: 4, y: 4, buttons: buttonForward},
			{change: cUp, kind: kMouse, x: 4, y: 4},
		}},
		{"an X button that is neither 1 nor 2 does nothing", []call{
			{name: "down", msg: win32.WMXButtonDown, wparam: xbutton(3, 0), lparam: lp(4, 4)},
			{name: "up", msg: win32.WMXButtonUp, wparam: xbutton(0, 0), lparam: lp(4, 4)},
		}, nil},
		{"the capture goes to another window: the buttons are released, the pointer stays", []call{
			{name: "down", msg: win32.WMLButtonDown, wparam: win32.MKLButton, lparam: lp(1, 1), fx: mouseEffects{trackLeave: true, setCapture: true}},
			{name: "capture", other: true},
			{name: "move", lparam: lp(2, 2)},
		}, []ev{
			{change: cAdd, kind: kMouse, x: 1, y: 1},
			{change: cDown, kind: kMouse, x: 1, y: 1, buttons: buttonPrimary},
			{change: cUp, kind: kMouse, x: 1, y: 1, synth: true},
			{change: cHover, kind: kMouse, x: 2, y: 2, dx: 1, dy: 1},
		}},
		{"the capture coming back to this window changes nothing", []call{
			{name: "down", msg: win32.WMLButtonDown, wparam: win32.MKLButton, lparam: lp(1, 1), fx: mouseEffects{trackLeave: true, setCapture: true}},
			{name: "capture", other: false},
		}, []ev{
			{change: cAdd, kind: kMouse, x: 1, y: 1},
			{change: cDown, kind: kMouse, x: 1, y: 1, buttons: buttonPrimary},
		}},
		{"a move that says no button is down gives the capture back (the up was lost)", []call{
			{name: "down", msg: win32.WMLButtonDown, wparam: win32.MKLButton, lparam: lp(1, 1), fx: mouseEffects{trackLeave: true, setCapture: true}},
			{name: "move", lparam: lp(2, 2), fx: mouseEffects{releaseCapture: true}},
		}, []ev{
			{change: cAdd, kind: kMouse, x: 1, y: 1},
			{change: cDown, kind: kMouse, x: 1, y: 1, buttons: buttonPrimary},
			{change: cUp, kind: kMouse, x: 2, y: 2, dx: 1, dy: 1},
		}},
		{"a button that is held while the pointer moves is a move with it", []call{
			{name: "move", wparam: win32.MKLButton | win32.MKRButton, lparam: lp(7, 8), fx: mouseEffects{trackLeave: true}},
		}, []ev{
			{change: cAdd, kind: kMouse, x: 7, y: 8},
			{change: cDown, kind: kMouse, x: 7, y: 8, buttons: buttonPrimary | buttonSecondary},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newPointerTracker(tickingClock())
			m := newMouseTranslator(tr, 0)
			for i, c := range tt.calls {
				var fx mouseEffects
				switch c.name {
				case "move":
					fx = m.move(c.wparam, c.lparam)
				case "down":
					fx = m.buttonDown(c.msg, c.wparam, c.lparam)
				case "up":
					fx = m.buttonUp(c.msg, c.wparam, c.lparam)
				case "leave":
					fx = m.leave()
				case "capture":
					m.captureChanged(!c.other)
				}
				if fx != c.fx {
					t.Errorf("call %d (%s): effects %+v, want %+v", i, c.name, fx, c.fx)
				}
			}
			sameEvents(t, tr.take(), tt.want)
		})
	}
}

func TestMouseButtonsOfState(t *testing.T) {
	tests := []struct {
		name   string
		wparam uintptr
		want   int
	}{
		{"none", 0, 0},
		{"left", win32.MKLButton, buttonPrimary},
		{"right", win32.MKRButton, buttonSecondary},
		{"middle", win32.MKMButton, buttonMiddle},
		{"back", win32.MKXButton1, buttonBack},
		{"forward", win32.MKXButton2, buttonForward},
		{"all five", win32.MKLButton | win32.MKRButton | win32.MKMButton | win32.MKXButton1 | win32.MKXButton2, 0x1F},
		{"the key states are not buttons", win32.MKShift | win32.MKControl, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mouseButtonsOfState(tt.wparam); got != tt.want {
				t.Errorf("mouseButtonsOfState(%#x) = %#x, want %#x", tt.wparam, got, tt.want)
			}
		})
	}
}

func TestMouseButtonOfMessage(t *testing.T) {
	tests := []struct {
		name   string
		msg    uint32
		wparam uintptr
		want   int
	}{
		{"left down", win32.WMLButtonDown, 0, buttonPrimary},
		{"left up", win32.WMLButtonUp, 0, buttonPrimary},
		{"right down", win32.WMRButtonDown, 0, buttonSecondary},
		{"right up", win32.WMRButtonUp, 0, buttonSecondary},
		{"middle down", win32.WMMButtonDown, 0, buttonMiddle},
		{"middle up", win32.WMMButtonUp, 0, buttonMiddle},
		{"x1 down", win32.WMXButtonDown, xbutton(win32.XButton1, 0), buttonBack},
		{"x2 up", win32.WMXButtonUp, xbutton(win32.XButton2, 0), buttonForward},
		{"x3", win32.WMXButtonDown, xbutton(4, 0), 0},
		{"not a button message", win32.WMMouseMove, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mouseButtonOfMessage(tt.msg, tt.wparam); got != tt.want {
				t.Errorf("mouseButtonOfMessage = %#x, want %#x", got, tt.want)
			}
		})
	}
}

func TestClientPointIsSigned16(t *testing.T) {
	tests := []struct {
		name   string
		lparam uintptr
		x, y   float64
	}{
		{"origin", 0, 0, 0},
		{"positive", 0x0014000A, 10, 20},
		{"negative x", 0x0014FFF6, -10, 20},
		{"negative y", 0xFFEC000A, 10, -20},
		{"both negative", 0xFFFFFFFF, -1, -1},
		{"the extremes", 0x80007FFF, 32767, -32768},
		{"the high bits of a 64-bit lParam are not read", 0xABCD0000_0014000A, 10, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := clientPoint(tt.lparam)
			if x != tt.x || y != tt.y {
				t.Errorf("clientPoint(%#x) = %v, %v; want %v, %v", tt.lparam, x, y, tt.x, tt.y)
			}
		})
	}
	// packPoint is the inverse.
	for _, p := range [][2]int32{{0, 0}, {10, 20}, {-10, 20}, {10, -20}, {-32768, 32767}} {
		x, y := clientPoint(packPoint(p[0], p[1]))
		if int32(x) != p[0] || int32(y) != p[1] {
			t.Errorf("packPoint(%v) came back as %v, %v", p, x, y)
		}
	}
}

func TestIsTouchOrPenMessage(t *testing.T) {
	// GetMessageExtraInfo carries 0xFF515700 for a mouse message that Windows made from pen or
	// touch (0x80 more for touch).
	tests := []struct {
		name  string
		extra uintptr
		want  bool
	}{
		{"a mouse", 0, false},
		{"a mouse with some extra information", 0x1234, false},
		{"a pen", 0xFF515700, true},
		{"a pen with a tablet number", 0xFF515705, true},
		{"a touch", 0xFF515780, true},
		{"another signature", 0xFF515800, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTouchOrPenMessage(tt.extra); got != tt.want {
				t.Errorf("isTouchOrPenMessage(%#x) = %v, want %v", tt.extra, got, tt.want)
			}
		})
	}
}
