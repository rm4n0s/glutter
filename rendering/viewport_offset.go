package rendering

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
)

// ScrollDirection is the direction in which the user is scrolling.
//
// Flutter: rendering/viewport_offset.dart ScrollDirection
type ScrollDirection int

// The values of ScrollDirection follow Flutter's declaration order.
const (
	// ScrollDirectionUnspecified is the zero value; it is not a real direction.
	//
	// Flutter: no counterpart
	ScrollDirectionUnspecified ScrollDirection = iota
	// ScrollDirectionIdle means no user is scrolling the viewport.
	//
	// Flutter: rendering/viewport_offset.dart ScrollDirection.idle
	ScrollDirectionIdle
	// ScrollDirectionForward means the user is scrolling the content toward its
	// start, so the content moves in the direction of the axis (the offset goes
	// toward the start of the scroll extent).
	//
	// Flutter: rendering/viewport_offset.dart ScrollDirection.forward
	ScrollDirectionForward
	// ScrollDirectionReverse means the user is scrolling opposite to
	// ScrollDirectionForward.
	//
	// Flutter: rendering/viewport_offset.dart ScrollDirection.reverse
	ScrollDirectionReverse
)

// String is the name with the enum, such as "ScrollDirection.forward".
//
// Flutter: rendering/viewport_offset.dart ScrollDirection
func (d ScrollDirection) String() string {
	switch d {
	case ScrollDirectionIdle:
		return "ScrollDirection.idle"
	case ScrollDirectionForward:
		return "ScrollDirection.forward"
	case ScrollDirectionReverse:
		return "ScrollDirection.reverse"
	}
	return "ScrollDirection.unspecified"
}

// FlipScrollDirection returns the opposite direction; idle stays idle.
//
// Flutter: rendering/viewport_offset.dart flipScrollDirection
func FlipScrollDirection(direction ScrollDirection) ScrollDirection {
	switch direction {
	case ScrollDirectionForward:
		return ScrollDirectionReverse
	case ScrollDirectionReverse:
		return ScrollDirectionForward
	}
	return direction
}

// ViewportOffset is the interface for the offset of a viewport: how far the
// content is scrolled. A viewport listens to it and tells it the dimensions
// of the viewport and of the content. A scroll position implements it.
//
// Flutter: rendering/viewport_offset.dart ViewportOffset
type ViewportOffset interface {
	foundation.Listenable
	// Pixels is the number of pixels to offset the children in the opposite of
	// the axis direction. It is only valid when HasPixels is true.
	//
	// Flutter: rendering/viewport_offset.dart ViewportOffset.pixels
	Pixels() float64
	// HasPixels is whether Pixels has a value. It is false before the viewport
	// calls ApplyContentDimensions the first time.
	//
	// Flutter: rendering/viewport_offset.dart ViewportOffset.hasPixels
	HasPixels() bool
	// ApplyViewportDimension tells the offset the extent of the viewport in the
	// main axis. It returns whether the dimension was accepted (false asks for
	// another layout).
	//
	// Flutter: rendering/viewport_offset.dart ViewportOffset.applyViewportDimension
	ApplyViewportDimension(viewportDimension float64) bool
	// ApplyContentDimensions tells the offset the scroll extents of the content.
	// It returns whether the dimensions were accepted (false asks for another
	// layout).
	//
	// Flutter: rendering/viewport_offset.dart ViewportOffset.applyContentDimensions
	ApplyContentDimensions(minScrollExtent, maxScrollExtent float64) bool
	// CorrectBy changes Pixels by the correction without notifying the listeners;
	// the viewport uses it when a sliver asked for a scroll offset correction.
	//
	// Flutter: rendering/viewport_offset.dart ViewportOffset.correctBy
	CorrectBy(correction float64)
	// JumpTo changes Pixels to the value without animation, and notifies the
	// listeners.
	//
	// Flutter: rendering/viewport_offset.dart ViewportOffset.jumpTo
	JumpTo(pixels float64)
	// AnimateTo animates Pixels to the value over the duration with the curve.
	// The future completes when the animation does.
	//
	// Flutter: rendering/viewport_offset.dart ViewportOffset.animateTo
	AnimateTo(to float64, duration time.Duration, curve animation.Curve) *scheduler.TickerFuture
	// UserScrollDirection is the direction in which the user is scrolling.
	//
	// Flutter: rendering/viewport_offset.dart ViewportOffset.userScrollDirection
	UserScrollDirection() ScrollDirection
	// AllowImplicitScrolling is whether a viewport is allowed to expand its
	// cache extent when the semantics ask for it (for a PageView).
	//
	// Flutter: rendering/viewport_offset.dart ViewportOffset.allowImplicitScrolling
	AllowImplicitScrolling() bool
}

// ViewportOffsetMoveToOpts are the optional parameters of ViewportOffsetMoveTo.
//
// Flutter: rendering/viewport_offset.dart ViewportOffset.moveTo
type ViewportOffsetMoveToOpts struct {
	// Duration, when it is none or zero, jumps instead of animating.
	Duration foundation.Opt[time.Duration]
	// Curve is the curve of the animation; the default is Curves.Ease.
	Curve animation.Curve
	// Clamp is whether the offset clamps to the extents; only a scroll position
	// uses it.
	Clamp foundation.Opt[bool]
}

// ViewportOffsetMoveTo animates the offset to the value, or jumps if there is
// no duration.
//
// Flutter: rendering/viewport_offset.dart ViewportOffset.moveTo
func ViewportOffsetMoveTo(offset ViewportOffset, to float64, opts ...ViewportOffsetMoveToOpts) *scheduler.TickerFuture {
	o := firstOpts(opts)
	if d, ok := o.Duration.Get(); !ok || d == 0 {
		offset.JumpTo(to)
		return scheduler.NewTickerFutureComplete()
	}
	curve := o.Curve
	if curve == nil {
		curve = animation.Curves.Ease
	}
	return offset.AnimateTo(to, o.Duration.Value(), curve)
}

// ViewportOffsetString describes the offset as ViewportOffset.toString does: its
// identity, and its pixels if it has some. A type that implements ViewportOffset
// can use it as its String method.
//
// Flutter: rendering/viewport_offset.dart ViewportOffset.toString
func ViewportOffsetString(o ViewportOffset) string {
	description := ""
	if o.HasPixels() {
		description = "offset: " + fixed1(o.Pixels())
	}
	return foundation.DescribeIdentity(o) + "(" + description + ")"
}

// fixedViewportOffset is a ViewportOffset with a value that only the viewport
// changes, with CorrectBy.
type fixedViewportOffset struct {
	*foundation.ChangeNotifier
	pixels float64
}

// NewViewportOffsetFixed creates an offset that stays at the value: JumpTo and
// AnimateTo do nothing, and it accepts any dimension.
//
// Flutter: rendering/viewport_offset.dart ViewportOffset.fixed
func NewViewportOffsetFixed(value float64) ViewportOffset {
	return &fixedViewportOffset{ChangeNotifier: foundation.NewChangeNotifier(), pixels: value}
}

// NewViewportOffsetZero creates a fixed offset at zero.
//
// Flutter: rendering/viewport_offset.dart ViewportOffset.zero
func NewViewportOffsetZero() ViewportOffset { return NewViewportOffsetFixed(0) }

func (o *fixedViewportOffset) Pixels() float64                          { return o.pixels }
func (o *fixedViewportOffset) HasPixels() bool                          { return true }
func (o *fixedViewportOffset) ApplyViewportDimension(float64) bool      { return true }
func (o *fixedViewportOffset) ApplyContentDimensions(_, _ float64) bool { return true }
func (o *fixedViewportOffset) CorrectBy(correction float64)             { o.pixels += correction }
func (o *fixedViewportOffset) JumpTo(float64)                           {}
func (o *fixedViewportOffset) AnimateTo(float64, time.Duration, animation.Curve) *scheduler.TickerFuture {
	return scheduler.NewTickerFutureComplete()
}

// String is the identity of the offset and its pixels.
//
// Flutter: rendering/viewport_offset.dart ViewportOffset.toString
func (o *fixedViewportOffset) String() string { return ViewportOffsetString(o) }

func (o *fixedViewportOffset) UserScrollDirection() ScrollDirection { return ScrollDirectionIdle }
func (o *fixedViewportOffset) AllowImplicitScrolling() bool         { return false }
