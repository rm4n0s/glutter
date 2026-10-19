package rendering

import (
	"fmt"
	"math"

	"github.com/rm4n0s/glutter/ui"
)

// RelativeRect is a rectangle described by the distances of its edges from the
// edges of a container.
//
// Flutter: rendering/stack.dart RelativeRect
type RelativeRect struct {
	// Left is the offset of the left edge from the left edge of the container.
	//
	// Flutter: rendering/stack.dart RelativeRect.left
	Left float64
	// Top is the offset of the top edge from the top edge of the container.
	//
	// Flutter: rendering/stack.dart RelativeRect.top
	Top float64
	// Right is the offset of the right edge from the right edge of the container.
	//
	// Flutter: rendering/stack.dart RelativeRect.right
	Right float64
	// Bottom is the offset of the bottom edge from the bottom edge of the
	// container.
	//
	// Flutter: rendering/stack.dart RelativeRect.bottom
	Bottom float64
}

// RelativeRectFill is a rectangle that covers the whole container.
//
// Flutter: rendering/stack.dart RelativeRect.fill
var RelativeRectFill = RelativeRect{}

// NewRelativeRectFromLTRB creates a rectangle from the offsets of its edges.
//
// Flutter: rendering/stack.dart RelativeRect.fromLTRB
func NewRelativeRectFromLTRB(left, top, right, bottom float64) RelativeRect {
	return RelativeRect{left, top, right, bottom}
}

// NewRelativeRectFromSize creates the rectangle of a rect in a container of the
// size.
//
// Flutter: rendering/stack.dart RelativeRect.fromSize
func NewRelativeRectFromSize(rect ui.Rect, container ui.Size) RelativeRect {
	return RelativeRect{rect.Left, rect.Top, container.Width - rect.Right, container.Height - rect.Bottom}
}

// NewRelativeRectFromRect creates the rectangle of a rect in a container rect.
//
// Flutter: rendering/stack.dart RelativeRect.fromRect
func NewRelativeRectFromRect(rect, container ui.Rect) RelativeRect {
	return RelativeRect{rect.Left - container.Left, rect.Top - container.Top,
		container.Right - rect.Right, container.Bottom - rect.Bottom}
}

// NewRelativeRectFromDirectional creates the rectangle from the start and end
// offsets, which are the left and right ones in a left-to-right text direction.
//
// Flutter: rendering/stack.dart RelativeRect.fromDirectional
func NewRelativeRectFromDirectional(textDirection ui.TextDirection, start, top, end, bottom float64) RelativeRect {
	if textDirection == ui.TextDirectionRTL {
		return RelativeRect{end, top, start, bottom}
	}
	return RelativeRect{start, top, end, bottom}
}

// HasInsets reports whether any of the offsets is positive.
//
// Flutter: rendering/stack.dart RelativeRect.hasInsets
func (r RelativeRect) HasInsets() bool {
	return r.Left > 0 || r.Top > 0 || r.Right > 0 || r.Bottom > 0
}

// Shift moves the rectangle by the offset.
//
// Flutter: rendering/stack.dart RelativeRect.shift
func (r RelativeRect) Shift(offset ui.Offset) RelativeRect {
	return RelativeRect{r.Left + offset.Dx, r.Top + offset.Dy, r.Right - offset.Dx, r.Bottom - offset.Dy}
}

// Inflate grows the rectangle by the delta on every side.
//
// Flutter: rendering/stack.dart RelativeRect.inflate
func (r RelativeRect) Inflate(delta float64) RelativeRect {
	return RelativeRect{r.Left - delta, r.Top - delta, r.Right - delta, r.Bottom - delta}
}

// Deflate shrinks the rectangle by the delta on every side.
//
// Flutter: rendering/stack.dart RelativeRect.deflate
func (r RelativeRect) Deflate(delta float64) RelativeRect { return r.Inflate(-delta) }

// Intersect is the rectangle that is inside both.
//
// Flutter: rendering/stack.dart RelativeRect.intersect
func (r RelativeRect) Intersect(other RelativeRect) RelativeRect {
	return RelativeRect{math.Max(r.Left, other.Left), math.Max(r.Top, other.Top),
		math.Max(r.Right, other.Right), math.Max(r.Bottom, other.Bottom)}
}

// ToRect is the rectangle in a container rect (only its size is used).
//
// Flutter: rendering/stack.dart RelativeRect.toRect
func (r RelativeRect) ToRect(container ui.Rect) ui.Rect {
	return ui.NewRectFromLTRB(r.Left, r.Top, container.Width()-r.Right, container.Height()-r.Bottom)
}

// ToSize is the size of the rectangle in a container of the size.
//
// Flutter: rendering/stack.dart RelativeRect.toSize
func (r RelativeRect) ToSize(container ui.Size) ui.Size {
	return ui.Size{Width: container.Width - r.Left - r.Right, Height: container.Height - r.Top - r.Bottom}
}

// RelativeRectLerp interpolates between two rectangles; nil is the zero
// rectangle (the one that fills the container).
//
// Deviation: when b is nil Flutter reads b's fields and throws; this scales a,
// which is evidently what it means (SPEC §12.10).
//
// Flutter: rendering/stack.dart RelativeRect.lerp
func RelativeRectLerp(a, b *RelativeRect, t float64) *RelativeRect {
	if a == b {
		return a
	}
	if a == nil {
		return &RelativeRect{b.Left * t, b.Top * t, b.Right * t, b.Bottom * t}
	}
	if b == nil {
		k := 1 - t
		return &RelativeRect{a.Left * k, a.Top * k, a.Right * k, a.Bottom * k}
	}
	return &RelativeRect{ui.LerpDouble(a.Left, b.Left, t), ui.LerpDouble(a.Top, b.Top, t),
		ui.LerpDouble(a.Right, b.Right, t), ui.LerpDouble(a.Bottom, b.Bottom, t)}
}

// String describes the rectangle, such as "RelativeRect.fromLTRB(1.0, 2.0, 3.0,
// 4.0)".
//
// Flutter: rendering/stack.dart RelativeRect.toString
func (r RelativeRect) String() string {
	return fmt.Sprintf("RelativeRect.fromLTRB(%.1f, %.1f, %.1f, %.1f)", r.Left, r.Top, r.Right, r.Bottom)
}
