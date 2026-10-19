package rendering

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
)

// BoxHitTest hit tests a position in the coordinates of a box, adding to the
// result. It reports whether the position hit.
//
// Flutter: rendering/box.dart BoxHitTest
type BoxHitTest func(result *BoxHitTestResult, position ui.Offset) bool

// BoxHitTestWithOutOfBandPosition hit tests with a position that the caller
// knows about, not passed in.
//
// Flutter: rendering/box.dart BoxHitTestWithOutOfBandPosition
type BoxHitTestWithOutOfBandPosition func(result *BoxHitTestResult) bool

// BoxHitTestResult is a HitTestResult with the helpers for the box protocol,
// which push the transform of a child before it tests.
//
// Flutter: rendering/box.dart BoxHitTestResult
type BoxHitTestResult struct {
	*gestures.HitTestResult
}

// NewBoxHitTestResult creates an empty result.
//
// Flutter: rendering/box.dart BoxHitTestResult.new
func NewBoxHitTestResult() *BoxHitTestResult {
	return &BoxHitTestResult{gestures.NewHitTestResult()}
}

// NewBoxHitTestResultWrap makes a box result that adds to the given result.
//
// Flutter: rendering/box.dart BoxHitTestResult.wrap
func NewBoxHitTestResultWrap(result *gestures.HitTestResult) *BoxHitTestResult {
	return &BoxHitTestResult{gestures.NewHitTestResultWrap(result)}
}

// AddWithPaintTransform hit tests a child that is painted with the transform:
// the position is mapped by the inverse of the transform (with its perspective
// removed). A nil transform means none. It returns false if the transform can not
// be inverted.
//
// Flutter: rendering/box.dart BoxHitTestResult.addWithPaintTransform
func (r *BoxHitTestResult) AddWithPaintTransform(transform *ui.Matrix4, position ui.Offset, hitTest BoxHitTest) bool {
	if transform != nil {
		transform = tryInvert(gestures.RemovePerspectiveTransform(transform))
		if transform == nil {
			return false
		}
	}
	return r.AddWithRawTransform(transform, position, hitTest)
}

// AddWithPaintOffset hit tests a child that is painted at the offset. A nil
// offset means none.
//
// Flutter: rendering/box.dart BoxHitTestResult.addWithPaintOffset
func (r *BoxHitTestResult) AddWithPaintOffset(offset *ui.Offset, position ui.Offset, hitTest BoxHitTest) bool {
	transformed := position
	if offset != nil {
		transformed = position.Sub(*offset)
		r.PushOffset(offset.Neg())
	}
	isHit := hitTest(r, transformed)
	if offset != nil {
		r.PopTransform()
	}
	return isHit
}

// AddWithRawTransform hit tests a child with a transform that maps the position
// to the child's coordinates directly. A nil transform means none.
//
// Flutter: rendering/box.dart BoxHitTestResult.addWithRawTransform
func (r *BoxHitTestResult) AddWithRawTransform(transform *ui.Matrix4, position ui.Offset, hitTest BoxHitTest) bool {
	transformed := position
	if transform != nil {
		transformed = transform.TransformPoint(position)
		r.PushTransform(transform)
	}
	isHit := hitTest(r, transformed)
	if transform != nil {
		r.PopTransform()
	}
	return isHit
}

// OutOfBandOpts say how the position maps in AddWithOutOfBandPosition; exactly
// one must be set.
//
// Flutter: rendering/box.dart BoxHitTestResult.addWithOutOfBandPosition
type OutOfBandOpts struct {
	// PaintOffset is the offset the child is painted at.
	PaintOffset *ui.Offset
	// PaintTransform is the transform the child is painted with; it is inverted.
	PaintTransform *ui.Matrix4
	// RawTransform maps the position directly.
	RawTransform *ui.Matrix4
}

// AddWithOutOfBandPosition hit tests a child with a position the callback knows,
// pushing the transform of the child. Exactly one of the options must be set, or
// a debug build panics with OutOfBandOptionsInvalid.
//
// Flutter: rendering/box.dart BoxHitTestResult.addWithOutOfBandPosition
func (r *BoxHitTestResult) AddWithOutOfBandPosition(o OutOfBandOpts, hitTest BoxHitTestWithOutOfBandPosition) bool {
	n := 0
	if o.PaintOffset != nil {
		n++
	}
	if o.PaintTransform != nil {
		n++
	}
	if o.RawTransform != nil {
		n++
	}
	if foundation.DebugMode && n != 1 {
		panic(errors.New("OutOfBandOptionsInvalid", "exactly one transform or offset argument must be provided"))
	}
	switch {
	case o.PaintOffset != nil:
		r.PushOffset(o.PaintOffset.Neg())
	case o.RawTransform != nil:
		r.PushTransform(o.RawTransform)
	default:
		inv := tryInvert(gestures.RemovePerspectiveTransform(o.PaintTransform))
		if inv == nil {
			panic(errors.New("OutOfBandTransformNotInvertible", "paintTransform must be invertible"))
		}
		r.PushTransform(inv)
	}
	isHit := hitTest(r)
	r.PopTransform()
	return isHit
}

// BoxHitTestEntry is a hit test entry of a box: the target and the position of
// the hit in the box's coordinates.
//
// Flutter: rendering/box.dart BoxHitTestEntry
type BoxHitTestEntry struct {
	gestures.HitTestEntryBase
	// LocalPosition is the position of the hit, in the target's coordinates.
	//
	// Flutter: rendering/box.dart BoxHitTestEntry.localPosition
	LocalPosition ui.Offset
}

// NewBoxHitTestEntry creates an entry for the target at the position.
//
// Flutter: rendering/box.dart BoxHitTestEntry.new
func NewBoxHitTestEntry(target Box, localPosition ui.Offset) *BoxHitTestEntry {
	return &BoxHitTestEntry{HitTestEntryBase: *gestures.NewHitTestEntry(target), LocalPosition: localPosition}
}

// String describes the entry.
//
// Flutter: rendering/box.dart BoxHitTestEntry.toString
func (e *BoxHitTestEntry) String() string {
	return foundation.DescribeIdentity(e.Target()) + "@" + e.LocalPosition.String()
}
