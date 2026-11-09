package macos

import (
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// Cocoa puts the origin of the screen coordinates at the bottom left of the primary display and grows
// upward; ui puts it at the top left of the primary display and grows downward (SPEC §29.4). These
// functions turn one into the other, in plain Go, so that two displays of different heights and scales
// are tested on every host.

// cocoaRect is a rectangle in Cocoa's screen coordinates: the origin is the bottom-left corner.
type cocoaRect struct{ X, Y, W, H float64 }

// flipRect is the rectangle in the coordinates of ui: the top-left origin of the primary display, whose
// height is primaryHeight.
func flipRect(r cocoaRect, primaryHeight float64) ui.Rect {
	top := primaryHeight - (r.Y + r.H)
	return ui.NewRectFromLTWH(r.X, top, r.W, r.H)
}

// unflipRect is the Cocoa rectangle of a rectangle of ui.
func unflipRect(r ui.Rect, primaryHeight float64) cocoaRect {
	return cocoaRect{X: r.Left, Y: primaryHeight - r.Bottom, W: r.Right - r.Left, H: r.Bottom - r.Top}
}

// flipPoint is the point of ui for a Cocoa screen point.
func flipPoint(x, y, primaryHeight float64) ui.Offset {
	return ui.Offset{Dx: x, Dy: primaryHeight - y}
}

// physicalSize is the size in physical pixels of a size in points at the scale. The size of the buffer
// that a view draws into is whole pixels, rounded to the nearest (a fractional scale gives a half pixel
// at most).
func physicalSize(points ui.Size, scale float64) ui.Size {
	if scale <= 0 {
		scale = 1
	}
	return ui.Size{Width: math.Round(points.Width * scale), Height: math.Round(points.Height * scale)}
}

// logicalSize is the size in points of a size in physical pixels.
func logicalSize(px ui.Size, scale float64) ui.Size {
	if scale <= 0 {
		scale = 1
	}
	return ui.Size{Width: px.Width / scale, Height: px.Height / scale}
}
