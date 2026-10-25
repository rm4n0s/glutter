package windows

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

func TestStyleOf(t *testing.T) {
	no := foundation.Some(false)
	tests := []struct {
		name string
		cfg  platform.WindowConfig
		want styleSet
	}{
		{"a regular window", platform.WindowConfig{}, styleSet{style: win32.WSOverlappedWindow}},
		{"not resizable has no thick frame and no maximize box", platform.WindowConfig{Resizable: no},
			styleSet{style: win32.WSOverlappedWindow &^ (win32.WSThickFrame | win32.WSMaximizeBox)}},
		{"not decorated is a popup", platform.WindowConfig{Decorated: no}, styleSet{style: win32.WSPopup}},
		{"borderless is a popup", platform.WindowConfig{Kind: platform.WindowKindBorderless}, styleSet{style: win32.WSPopup}},
		{"a dialog", platform.WindowConfig{Kind: platform.WindowKindDialog},
			styleSet{style: win32.WSOverlapped | win32.WSCaption | win32.WSSysMenu, exStyle: win32.WSExDlgModalFrame}},
		{"a modal dialog has the style of a dialog", platform.WindowConfig{Kind: platform.WindowKindDialog, Modal: true, HasParent: true},
			styleSet{style: win32.WSOverlapped | win32.WSCaption | win32.WSSysMenu, exStyle: win32.WSExDlgModalFrame}},
		{"a dialog that is not decorated is a popup", platform.WindowConfig{Kind: platform.WindowKindDialog, Decorated: no}, styleSet{style: win32.WSPopup}},
		{"the app's title bar keeps the frame", platform.WindowConfig{Decorations: platform.DecorationsApp}, styleSet{style: win32.WSOverlappedWindow}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := styleOf(tt.cfg); got != tt.want {
				t.Errorf("styleOf = %#x/%#x, want %#x/%#x", got.style, got.exStyle, tt.want.style, tt.want.exStyle)
			}
		})
	}
}

func TestOwnsClientArea(t *testing.T) {
	no := foundation.Some(false)
	tests := []struct {
		cfg  platform.WindowConfig
		want bool
	}{
		{platform.WindowConfig{}, false},
		{platform.WindowConfig{Decorations: platform.DecorationsApp}, true},
		{platform.WindowConfig{Decorations: platform.DecorationsApp, Decorated: no}, false},
		{platform.WindowConfig{Decorations: platform.DecorationsApp, Kind: platform.WindowKindBorderless}, false},
	}
	for _, tt := range tests {
		if got := ownsClientArea(tt.cfg); got != tt.want {
			t.Errorf("ownsClientArea(%+v) = %v, want %v", tt.cfg, got, tt.want)
		}
	}
}

func TestScaleAndPhysicalSizes(t *testing.T) {
	tests := []struct {
		dpi     uint32
		logical float64
		scale   float64
		want    int32
	}{
		{96, 480, 1, 480}, {120, 480, 1.25, 600}, {144, 480, 1.5, 720}, {168, 480, 1.75, 840}, {192, 640, 2, 1280},
		{144, 333, 1.5, 500}, {120, 101, 1.25, 126}, {0, 480, 1, 480},
	}
	for _, tt := range tests {
		if got := scaleOfDPI(tt.dpi); got != tt.scale {
			t.Errorf("scaleOfDPI(%d) = %v, want %v", tt.dpi, got, tt.scale)
		}
		if got := physical(tt.logical, scaleOfDPI(tt.dpi)); got != tt.want {
			t.Errorf("physical(%v, %v) = %d, want %d", tt.logical, tt.scale, got, tt.want)
		}
	}
}

func TestInsetsAndOuterSize(t *testing.T) {
	// What AdjustWindowRectExForDpi makes of a 100x100 client rectangle at 100% for a captioned
	// window: 8 pixels of frame at the sides and the bottom, 31 above.
	in := insetsOf(win32.Rect{Left: -8, Top: -31, Right: 108, Bottom: 108}, 100, 100)
	if in != (insets{left: 8, top: 31, right: 8, bottom: 8}) {
		t.Fatalf("insets = %+v", in)
	}
	tests := []struct {
		cw, ch int32
		w, h   int32
	}{{480, 640, 496, 679}, {720, 960, 736, 999}, {1, 1, 17, 40}}
	for _, tt := range tests {
		if w, h := in.outer(tt.cw, tt.ch); w != tt.w || h != tt.h {
			t.Errorf("outer(%d, %d) = %d, %d; want %d, %d", tt.cw, tt.ch, w, h, tt.w, tt.h)
		}
	}
	if in.width() != 16 || in.height() != 39 {
		t.Errorf("width, height = %d, %d", in.width(), in.height())
	}
}

func TestMinMaxLimits(t *testing.T) {
	in := insets{left: 8, top: 31, right: 8, bottom: 8}
	tests := []struct {
		name             string
		minSize, maxSize ui.Size
		scale            float64
		wantMin, wantMax win32.Point
		hasMin, hasMax   bool
	}{
		{"no limits leave what Windows set", ui.Size{}, ui.Size{}, 1, win32.Point{}, win32.Point{}, false, false},
		{"a minimum at 100%", ui.Size{Width: 200, Height: 100}, ui.Size{}, 1, win32.Point{X: 216, Y: 139}, win32.Point{}, true, false},
		{"a minimum at 150%", ui.Size{Width: 200, Height: 100}, ui.Size{}, 1.5, win32.Point{X: 316, Y: 189}, win32.Point{}, true, false},
		{"a maximum", ui.Size{}, ui.Size{Width: 800, Height: 600}, 1, win32.Point{}, win32.Point{X: 816, Y: 639}, false, true},
		{"a maximum with one side free is limited by the screen", ui.Size{}, ui.Size{Width: 800}, 1, win32.Point{}, win32.Point{X: 816, Y: 700}, false, true},
		{"a limit is not larger than the screen", ui.Size{Width: 5000, Height: 5000}, ui.Size{Width: 9000, Height: 9000}, 1,
			win32.Point{X: 1000, Y: 700}, win32.Point{X: 1000, Y: 700}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mn, mx, hasMin, hasMax := minMaxLimits(tt.minSize, tt.maxSize, tt.scale, in, 1000, 700)
			if hasMin != tt.hasMin || hasMax != tt.hasMax || (hasMin && mn != tt.wantMin) || (hasMax && mx != tt.wantMax) {
				t.Errorf("got min %v %v, max %v %v; want min %v %v, max %v %v", mn, hasMin, mx, hasMax, tt.wantMin, tt.hasMin, tt.wantMax, tt.hasMax)
			}
		})
	}
}

func TestFullscreenStyleAndRestore(t *testing.T) {
	saved := styleSet{style: win32.WSOverlappedWindow | win32.WSVisible | win32.WSMaximize, exStyle: win32.WSExWindowEdge | win32.WSExAppWindow}
	fs := fullscreenStyle(saved)
	if fs.style&(win32.WSCaption|win32.WSThickFrame|win32.WSSysMenu|win32.WSMinimizeBox|win32.WSMaximizeBox) != 0 {
		t.Errorf("the full-screen style %#x still has a frame", fs.style)
	}
	if fs.style&win32.WSPopup == 0 || fs.style&win32.WSVisible == 0 {
		t.Errorf("the full-screen style %#x is not a visible popup", fs.style)
	}
	if fs.exStyle != win32.WSExAppWindow {
		t.Errorf("the full-screen extended style = %#x, want only WS_EX_APPWINDOW", fs.exStyle)
	}
	rs := restoredStyle(styleSet{style: win32.WSOverlappedWindow, exStyle: win32.WSExAppWindow})
	if rs.style != win32.WSOverlappedWindow|win32.WSVisible || rs.exStyle != win32.WSExAppWindow {
		t.Errorf("restored = %#x/%#x", rs.style, rs.exStyle)
	}
}

func TestStateOfSize(t *testing.T) {
	tests := []struct {
		name       string
		wparam     uintptr
		fullscreen bool
		want       windowState
		ok         bool
	}{
		{"restored", win32.SizeRestored, false, windowState{}, true},
		{"minimized", win32.SizeMinimized, false, windowState{minimized: true}, true},
		{"maximized", win32.SizeMaximized, false, windowState{maximized: true}, true},
		{"restored in full screen", win32.SizeRestored, true, windowState{fullscreen: true}, true},
		{"maxshow changes nothing", win32.SizeMaxShow, false, windowState{}, false},
		{"maxhide changes nothing", win32.SizeMaxHide, false, windowState{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := stateOfSize(tt.wparam, tt.fullscreen)
			if got != tt.want || ok != tt.ok {
				t.Errorf("stateOfSize = %+v, %v; want %+v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestWindowInfoOf(t *testing.T) {
	no := foundation.Some(false)
	regular := platform.WindowConfig{}
	app := platform.WindowConfig{Decorations: platform.DecorationsApp}
	tests := []struct {
		name  string
		cfg   platform.WindowConfig
		style uint32
		st    windowState
		act   bool
		want  ui.WindowInfo
	}{
		{"a regular window", regular, win32.WSOverlappedWindow, windowState{}, true, ui.WindowInfo{
			Decoration: ui.DecorationServerSide, Capabilities: ui.AllWindowCapabilities, Title: "T", Activated: true}},
		{"maximized", regular, win32.WSOverlappedWindow, windowState{maximized: true}, false, ui.WindowInfo{
			Decoration: ui.DecorationServerSide, Capabilities: ui.AllWindowCapabilities, Title: "T", Maximized: true}},
		{"minimized in full screen", regular, win32.WSOverlappedWindow, windowState{minimized: true, fullscreen: true}, false, ui.WindowInfo{
			Decoration: ui.DecorationServerSide, Capabilities: ui.AllWindowCapabilities, Title: "T", Minimized: true, Fullscreen: true}},
		{"the app draws the title bar", app, win32.WSOverlappedWindow, windowState{}, false, ui.WindowInfo{
			Decoration: ui.DecorationApp, Capabilities: ui.AllWindowCapabilities, Title: "T"}},
		{"not resizable", platform.WindowConfig{Resizable: no}, win32.WSOverlappedWindow &^ (win32.WSThickFrame | win32.WSMaximizeBox), windowState{}, false, ui.WindowInfo{
			Decoration: ui.DecorationServerSide, Title: "T",
			Capabilities: ui.WindowCapabilities{WindowMenu: true, Minimize: true}}},
		{"borderless", platform.WindowConfig{Kind: platform.WindowKindBorderless}, win32.WSPopup, windowState{}, false, ui.WindowInfo{
			Decoration: ui.DecorationNone, Title: "T", Capabilities: ui.WindowCapabilities{Fullscreen: true}}},
		{"a dialog", platform.WindowConfig{Kind: platform.WindowKindDialog}, win32.WSOverlapped | win32.WSCaption | win32.WSSysMenu, windowState{}, false, ui.WindowInfo{
			Decoration: ui.DecorationServerSide, Title: "T", Capabilities: ui.WindowCapabilities{WindowMenu: true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := windowInfoOf(tt.cfg, tt.style, tt.st, tt.act, "T"); got != tt.want {
				t.Errorf("windowInfoOf = %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestHitTest(t *testing.T) {
	window := win32.Rect{Left: 100, Top: 100, Right: 500, Bottom: 400}
	const border = 8
	tests := []struct {
		name                 string
		pt                   win32.Point
		resizable, maxOrFull bool
		want                 uintptr
	}{
		{"the middle", win32.Point{X: 300, Y: 250}, true, false, win32.HTClient},
		{"left edge", win32.Point{X: 103, Y: 250}, true, false, win32.HTLeft},
		{"right edge", win32.Point{X: 497, Y: 250}, true, false, win32.HTRight},
		{"top edge", win32.Point{X: 300, Y: 103}, true, false, win32.HTTop},
		{"bottom edge", win32.Point{X: 300, Y: 397}, true, false, win32.HTBottom},
		{"top left", win32.Point{X: 101, Y: 101}, true, false, win32.HTTopLeft},
		{"top right", win32.Point{X: 499, Y: 101}, true, false, win32.HTTopRight},
		{"bottom left", win32.Point{X: 101, Y: 399}, true, false, win32.HTBottomLeft},
		{"bottom right", win32.Point{X: 499, Y: 399}, true, false, win32.HTBottomRight},
		{"just inside the band", win32.Point{X: 108, Y: 250}, true, false, win32.HTClient},
		{"just outside it", win32.Point{X: 107, Y: 250}, true, false, win32.HTLeft},
		{"maximized has no band", win32.Point{X: 101, Y: 250}, true, true, win32.HTClient},
		{"not resizable has no band", win32.Point{X: 101, Y: 250}, false, false, win32.HTClient},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hitTest(tt.pt, window, border, tt.resizable, tt.maxOrFull); got != tt.want {
				t.Errorf("hitTest = %d, want %d", got, tt.want)
			}
		})
	}
	if got := hitTest(win32.Point{X: 101, Y: 250}, window, 0, true, false); got != win32.HTClient {
		t.Errorf("a window with no frame thickness has a resize band: %d", got)
	}
}

func TestEdgeHitCoversEveryEdge(t *testing.T) {
	want := map[ui.ResizeEdge]uintptr{
		ui.ResizeEdgeTop: win32.HTTop, ui.ResizeEdgeBottom: win32.HTBottom, ui.ResizeEdgeLeft: win32.HTLeft,
		ui.ResizeEdgeRight: win32.HTRight, ui.ResizeEdgeTopLeft: win32.HTTopLeft, ui.ResizeEdgeTopRight: win32.HTTopRight,
		ui.ResizeEdgeBottomLeft: win32.HTBottomLeft, ui.ResizeEdgeBottomRight: win32.HTBottomRight,
	}
	for e, w := range want {
		if got := edgeHit(e); got != w {
			t.Errorf("edgeHit(%v) = %d, want %d", e, got, w)
		}
	}
}

func TestMaximizedInsetAndDarkTitleBar(t *testing.T) {
	if got := maximizedInset(8, 4); got != 12 {
		t.Errorf("maximizedInset = %d, want 12", got)
	}
	tests := []struct {
		value uint32
		found bool
		want  bool
	}{{0, true, true}, {1, true, false}, {0, false, false}, {1, false, false}}
	for _, tt := range tests {
		if got := darkTitleBar(tt.value, tt.found); got != tt.want {
			t.Errorf("darkTitleBar(%d, %v) = %v, want %v", tt.value, tt.found, got, tt.want)
		}
	}
}
