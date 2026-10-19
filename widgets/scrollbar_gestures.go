package widgets

import (
	"github.com/rm4n0s/glutter/gestures"
)

// trackTapRecognizer recognizes a tap on the track of a scrollbar, and only
// there.
//
// Flutter: widgets/scrollbar.dart _TrackTapGestureRecognizer
type trackTapRecognizer struct {
	gestures.TapGestureRecognizer
	state *RawScrollbarState
}

func newTrackTapRecognizer(binding *gestures.GestureBinding, state *RawScrollbarState) *trackTapRecognizer {
	r := &trackTapRecognizer{state: state}
	r.InitBaseTap(binding, r, gestures.BaseTapOpts{GestureRecognizerOpts: gestures.GestureRecognizerOpts{DebugOwner: state}})
	return r
}

// IsPointerAllowed allows a pointer that went down on the track.
//
// Flutter: widgets/scrollbar.dart _TrackTapGestureRecognizer.isPointerAllowed
func (r *trackTapRecognizer) IsPointerAllowed(event *gestures.PointerDownEvent) bool {
	return r.state.IsPointerOverTrack(event.Position(), event.Kind()) && r.TapGestureRecognizer.IsPointerAllowed(event)
}

// verticalThumbDragRecognizer recognizes the drag of the thumb of a vertical
// scrollbar, and only that.
//
// Flutter: widgets/scrollbar.dart _VerticalThumbDragGestureRecognizer
type verticalThumbDragRecognizer struct {
	gestures.VerticalDragGestureRecognizer
	state *RawScrollbarState
}

func newVerticalThumbDragRecognizer(binding *gestures.GestureBinding, state *RawScrollbarState) *verticalThumbDragRecognizer {
	r := &verticalThumbDragRecognizer{state: state}
	gestures.InitVerticalDragGestureRecognizer(&r.VerticalDragGestureRecognizer, r, binding, gestures.DragGestureRecognizerOpts{DebugOwner: state})
	return r
}

// IsPointerPanZoomAllowed is false: the scrollbar does not take trackpad pan
// gestures.
//
// Flutter: widgets/scrollbar.dart _VerticalThumbDragGestureRecognizer.isPointerPanZoomAllowed
func (r *verticalThumbDragRecognizer) IsPointerPanZoomAllowed(*gestures.PointerPanZoomStartEvent) bool {
	return false
}

// IsPointerAllowed allows a pointer that went down on the thumb.
//
// Flutter: widgets/scrollbar.dart _VerticalThumbDragGestureRecognizer.isPointerAllowed
func (r *verticalThumbDragRecognizer) IsPointerAllowed(event *gestures.PointerDownEvent) bool {
	return r.state.IsPointerOverThumb(event.Position(), event.Kind()) && r.DragGestureRecognizer.IsPointerAllowed(event)
}

// horizontalThumbDragRecognizer recognizes the drag of the thumb of a horizontal
// scrollbar, and only that.
//
// Flutter: widgets/scrollbar.dart _HorizontalThumbDragGestureRecognizer
type horizontalThumbDragRecognizer struct {
	gestures.HorizontalDragGestureRecognizer
	state *RawScrollbarState
}

func newHorizontalThumbDragRecognizer(binding *gestures.GestureBinding, state *RawScrollbarState) *horizontalThumbDragRecognizer {
	r := &horizontalThumbDragRecognizer{state: state}
	gestures.InitHorizontalDragGestureRecognizer(&r.HorizontalDragGestureRecognizer, r, binding, gestures.DragGestureRecognizerOpts{DebugOwner: state})
	return r
}

// IsPointerPanZoomAllowed is false: the scrollbar does not take trackpad pan
// gestures.
//
// Flutter: widgets/scrollbar.dart _HorizontalThumbDragGestureRecognizer.isPointerPanZoomAllowed
func (r *horizontalThumbDragRecognizer) IsPointerPanZoomAllowed(*gestures.PointerPanZoomStartEvent) bool {
	return false
}

// IsPointerAllowed allows a pointer that went down on the thumb.
//
// Flutter: widgets/scrollbar.dart _HorizontalThumbDragGestureRecognizer.isPointerAllowed
func (r *horizontalThumbDragRecognizer) IsPointerAllowed(event *gestures.PointerDownEvent) bool {
	return r.state.IsPointerOverThumb(event.Position(), event.Kind()) && r.DragGestureRecognizer.IsPointerAllowed(event)
}
