package macos

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the geometry of the title bar of a window whose app draws its own (DecorationsApp, SPEC
// §29.4): the part of the window that the content leaves for the title bar and the area of the traffic lights.
// It is plain Go over the rectangles that AppKit reports, so that it is tested at every scale on every host.

// boxRect is a rectangle of AppKit in points, with the origin at its bottom left: the plain-Go form of NSRect,
// which only the files for macOS know.
type boxRect struct {
	X, Y, W, H float64
}

// titleBarHeight is the height in points of the title bar that the content layout rectangle leaves out of
// the frame of the window, never negative.
func titleBarHeight(frame, layout boxRect) float64 {
	return math.Max(0, frame.H-layout.H)
}

// titleBarPadding is titleBarHeight in physical pixels, the top padding of ViewMetrics.
func titleBarPadding(frame, layout boxRect, scale float64) float64 {
	return titleBarHeight(frame, layout) * scale
}

// trafficLightArea is the rectangle, in physical pixels of the view with the origin at its top left, that
// holds the close, minimize and zoom buttons of a window: the union of their frames, which AppKit gives in the
// coordinates of the title bar (the origin at its bottom left), flipped into the view's. A button that is
// missing (a dialog has no zoom button) is a zero rectangle that is left out. The result is zero when no
// button exists.
func trafficLightArea(buttons []boxRect, barHeight, scale float64) ui.Rect {
	var area ui.Rect
	have := false
	for _, b := range buttons {
		if b.W <= 0 || b.H <= 0 {
			continue
		}
		r := ui.Rect{
			Left:   b.X * scale,
			Top:    (barHeight - b.Y - b.H) * scale,
			Right:  (b.X + b.W) * scale,
			Bottom: (barHeight - b.Y) * scale,
		}
		if !have {
			area, have = r, true
			continue
		}
		area.Left = math.Min(area.Left, r.Left)
		area.Top = math.Min(area.Top, r.Top)
		area.Right = math.Max(area.Right, r.Right)
		area.Bottom = math.Max(area.Bottom, r.Bottom)
	}
	return area
}

// trafficLightY is where the bottom of a window button goes in a title bar of the app's height, so that the
// button is in the middle of the bar: the buttons of AppKit sit in the middle of its own title bar (28 or 32
// points), which is rarely the height of the bar that the app draws. It is never negative (a bar lower than
// the buttons keeps them at its bottom).
func trafficLightY(barHeight, buttonHeight float64) float64 {
	return math.Max(0, (barHeight-buttonHeight)/2)
}

// The values of the system setting "Double-click a window's title bar to" (AppleActionOnDoubleClick).
const (
	doubleClickZoom     = "Maximize"
	doubleClickMinimize = "Minimize"
	doubleClickNothing  = "None"
)

// titleDoubleClick is what a double click on the title bar that the app draws does, given the system setting:
// the framework's double tap asks for the maximize or the restore, which the "Maximize" setting (the default
// and the one of an unknown value) lets through as the zoom of the window, "Minimize" turns into a minimize and
// "None" into nothing. ok is false when nothing is to be done.
func titleDoubleClick(setting string, asked ui.WindowActionKind) (kind ui.WindowActionKind, ok bool) {
	switch setting {
	case doubleClickMinimize:
		return ui.WindowActionMinimize, true
	case doubleClickNothing:
		return 0, false
	}
	return asked, true
}
