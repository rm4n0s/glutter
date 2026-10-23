package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// DragBoundaryDelegate tells whether a dragged object is within the boundary,
// and where the nearest position within it is.
//
// Flutter: widgets/drag_boundary.dart DragBoundaryDelegate
type DragBoundaryDelegate[T any] interface {
	// IsWithinBoundary is whether the dragged object is within the boundary.
	//
	// Flutter: widgets/drag_boundary.dart DragBoundaryDelegate.isWithinBoundary
	IsWithinBoundary(draggedObject T) bool
	// NearestPositionWithinBoundary is the dragged object moved to the nearest
	// position within the boundary.
	//
	// Flutter: widgets/drag_boundary.dart DragBoundaryDelegate.nearestPositionWithinBoundary
	NearestPositionWithinBoundary(draggedObject T) T
}

// dragBoundaryDelegateForRect is the delegate of a rectangular boundary, or of
// none.
//
// Flutter: widgets/drag_boundary.dart _DragBoundaryDelegateForRect
type dragBoundaryDelegateForRect struct {
	boundary foundation.Opt[ui.Rect]
}

func (d dragBoundaryDelegateForRect) IsWithinBoundary(draggedObject ui.Rect) bool {
	boundary, ok := d.boundary.Get()
	if !ok {
		return true
	}
	return boundary.Contains(draggedObject.TopLeft()) && boundary.Contains(draggedObject.BottomRight())
}

// NearestPositionWithinBoundary moves the rectangle inside the boundary. In debug
// builds it panics with the tag DragBoundaryTooSmall when the rectangle is larger
// than the boundary.
func (d dragBoundaryDelegateForRect) NearestPositionWithinBoundary(draggedObject ui.Rect) ui.Rect {
	boundary, ok := d.boundary.Get()
	if !ok {
		return draggedObject
	}
	if boundary.Right-draggedObject.Width() < boundary.Left || boundary.Bottom-draggedObject.Height() < boundary.Top {
		panic(errors.New(tagDragBoundaryTooSmall, "the rect is larger than the boundary; the rect width must be less than the boundary width, and the rect height must be less than the boundary height"))
	}
	left := ui.ClampDouble(draggedObject.Left, boundary.Left, boundary.Right-draggedObject.Width())
	top := ui.ClampDouble(draggedObject.Top, boundary.Top, boundary.Bottom-draggedObject.Height())
	return ui.NewRectFromLTWH(left, top, draggedObject.Width(), draggedObject.Height())
}

// DragBoundary is a widget that provides a boundary for its descendants, which
// they use to keep what they drag inside it.
//
// Flutter: widgets/drag_boundary.dart DragBoundary
type DragBoundary struct {
	InheritedWidget
}

// NewDragBoundary creates a boundary around the child.
//
// Flutter: widgets/drag_boundary.dart DragBoundary.new
func NewDragBoundary(child Widget, opts ...WidgetOpts) *DragBoundary {
	return &DragBoundary{NewInheritedWidgetBase(child, opts...)}
}

// UpdateShouldNotify is true.
//
// Flutter: widgets/drag_boundary.dart DragBoundary.updateShouldNotify
func (w *DragBoundary) UpdateShouldNotify(Widget) bool { return true }

// DragBoundaryForRectOfOpts are the optional parameters of the DragBoundaryForRect
// functions.
//
// Flutter: widgets/drag_boundary.dart DragBoundary.forRectOf
type DragBoundaryForRectOfOpts struct {
	// UseLocalPosition makes the boundary be in the local coordinates of the
	// DragBoundary instead of the global ones (Flutter's useGlobalPosition: false).
	UseLocalPosition bool
}

// DragBoundaryForRectMaybeOf is the delegate for dragged rectangles of the
// closest DragBoundary above the context, and nil if there is none.
//
// Flutter: widgets/drag_boundary.dart DragBoundary.forRectMaybeOf
func DragBoundaryForRectMaybeOf(bc BuildContext, opts ...DragBoundaryForRectOfOpts) DragBoundaryDelegate[ui.Rect] {
	element := GetElementForInheritedWidget[*DragBoundary](bc)
	if element == nil {
		return nil
	}
	rb, ok := element.FindRenderObject().(rendering.Box)
	if !ok || !rb.HasSize() {
		panic(errors.New(tagDragBoundaryUnavailable, "DragBoundary is not available"))
	}
	var boundary ui.Rect
	if firstOpts(opts).UseLocalPosition {
		boundary = ui.OffsetZero.And(rb.Size())
	} else {
		a := rb.LocalToGlobal(ui.OffsetZero, nil)
		b := rb.LocalToGlobal(rb.Size().BottomRight(ui.OffsetZero), nil)
		boundary = ui.NewRectFromLTRB(minF(a.Dx, b.Dx), minF(a.Dy, b.Dy), maxF(a.Dx, b.Dx), maxF(a.Dy, b.Dy))
	}
	return dragBoundaryDelegateForRect{foundation.Some(boundary)}
}

// DragBoundaryForRectOf is the delegate for dragged rectangles of the closest
// DragBoundary above the context; without one it is a delegate with no boundary.
//
// Flutter: widgets/drag_boundary.dart DragBoundary.forRectOf
func DragBoundaryForRectOf(bc BuildContext, opts ...DragBoundaryForRectOfOpts) DragBoundaryDelegate[ui.Rect] {
	if d := DragBoundaryForRectMaybeOf(bc, opts...); d != nil {
		return d
	}
	return dragBoundaryDelegateForRect{}
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDragBoundaryTooSmall    = "DragBoundaryTooSmall"
	tagDragBoundaryUnavailable = "DragBoundaryUnavailable"
)
