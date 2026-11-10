package cupertino

import (
	"reflect"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// slideTarget is what a button of a dialog or an action sheet is for the finger that slides over the buttons: it
// is told when the finger enters it and leaves it, and when the finger is lifted over it.
//
// Flutter: cupertino/dialog.dart _SlideTarget
type slideTarget interface {
	// didEnter is called when the finger enters the target; innerEnabled says whether the targets inside it that
	// were entered allow it, and the result is what the targets around it get.
	didEnter(fromPointerDown, innerEnabled bool) bool
	didLeave()
	didConfirm()
}

// slidingTapRecognizer is a drag recognizer that wins at once (the drag starts at the down) and that reports the
// moves and the end of the primary pointer before the drag does, so the finger slides over the buttons.
//
// Flutter: cupertino/dialog.dart _SlidingTapGestureRecognizer
type slidingTapRecognizer struct {
	gestures.VerticalDragGestureRecognizer
	onResponsiveUpdate func(ui.Offset)
	onResponsiveEnd    func(ui.Offset)
	primaryPointer     foundation.Opt[int]
}

func newSlidingTapRecognizer(binding *gestures.GestureBinding, debugOwner any) *slidingTapRecognizer {
	r := &slidingTapRecognizer{}
	gestures.InitVerticalDragGestureRecognizer(&r.VerticalDragGestureRecognizer, r, binding, gestures.DragGestureRecognizerOpts{DebugOwner: debugOwner})
	r.DragStartBehavior = gestures.DragStartBehaviorDown
	return r
}

// AddAllowedPointer remembers the first pointer as the primary one.
//
// Flutter: cupertino/dialog.dart _SlidingTapGestureRecognizer.addAllowedPointer
func (r *slidingTapRecognizer) AddAllowedPointer(event *gestures.PointerDownEvent) {
	if r.primaryPointer.IsNone() {
		r.primaryPointer = foundation.Some(event.Pointer())
	}
	r.VerticalDragGestureRecognizer.AddAllowedPointer(event)
}

// RejectGesture forgets the primary pointer when it is the one that is rejected.
//
// Flutter: cupertino/dialog.dart _SlidingTapGestureRecognizer.rejectGesture
func (r *slidingTapRecognizer) RejectGesture(pointer int) {
	if p, ok := r.primaryPointer.Get(); ok && p == pointer {
		r.primaryPointer = foundation.None[int]()
	}
	r.VerticalDragGestureRecognizer.RejectGesture(pointer)
}

// HandleEvent reports the moves and the end of the primary pointer.
//
// Flutter: cupertino/dialog.dart _SlidingTapGestureRecognizer.handleEvent
func (r *slidingTapRecognizer) HandleEvent(event gestures.PointerEvent) {
	if p, ok := r.primaryPointer.Get(); ok && event.Pointer() == p {
		switch e := event.(type) {
		case *gestures.PointerMoveEvent:
			if r.onResponsiveUpdate != nil {
				r.onResponsiveUpdate(e.Position())
			}
		case *gestures.PointerUpEvent:
			r.StopTrackingPointer(p)
			if r.onResponsiveEnd != nil {
				r.onResponsiveEnd(e.Position())
			}
			r.primaryPointer = foundation.None[int]()
			return
		case *gestures.PointerCancelEvent:
			r.primaryPointer = foundation.None[int]()
		}
	}
	r.VerticalDragGestureRecognizer.HandleEvent(event)
}

// DebugDescription is "tap slide".
//
// Flutter: cupertino/dialog.dart _SlidingTapGestureRecognizer.debugDescription
func (r *slidingTapRecognizer) DebugDescription() string { return "tap slide" }

// targetSelectionRecognizer lets the finger slide over the buttons: for each position of the finger it hit tests
// the view, and tells the slide targets that it finds which of them the finger entered, left and lifted over.
// It is the sliding tap recognizer, whose arena and pointer methods it has.
//
// Flutter: cupertino/dialog.dart _TargetSelectionGestureRecognizer
type targetSelectionRecognizer struct {
	*slidingTapRecognizer
	hitTest        func(ui.Offset) *gestures.HitTestResult
	currentTargets []slideTarget
}

func newTargetSelectionRecognizer(binding *gestures.GestureBinding, debugOwner any, hitTest func(ui.Offset) *gestures.HitTestResult) *targetSelectionRecognizer {
	r := &targetSelectionRecognizer{slidingTapRecognizer: newSlidingTapRecognizer(binding, debugOwner), hitTest: hitTest}
	r.OnDown = func(details gestures.DragDownDetails) { r.updateDrag(details.GlobalPosition, true) }
	r.onResponsiveUpdate = func(p ui.Offset) { r.updateDrag(p, false) }
	r.onResponsiveEnd = func(p ui.Offset) {
		r.updateDrag(p, false)
		for _, target := range r.currentTargets {
			target.didConfirm()
		}
		r.currentTargets = nil
	}
	r.OnCancel = func() {
		for _, target := range r.currentTargets {
			target.didLeave()
		}
		r.currentTargets = nil
	}
	return r
}

func (r *targetSelectionRecognizer) updateDrag(pointerPosition ui.Offset, fromPointerDown bool) {
	result := r.hitTest(pointerPosition)
	var foundTargets []slideTarget
	for _, entry := range result.Path() {
		if target, ok := entry.Target().(*rendering.RenderMetaData); ok {
			if t, ok := target.MetaData.(slideTarget); ok {
				foundTargets = append(foundTargets, t)
			}
		}
	}
	var first, now slideTarget
	if len(r.currentTargets) > 0 {
		first = r.currentTargets[0]
	}
	if len(foundTargets) > 0 {
		now = foundTargets[0]
	}
	if first != now {
		for _, target := range r.currentTargets {
			target.didLeave()
		}
		r.currentTargets = foundTargets
		enabled := true
		for _, target := range r.currentTargets {
			enabled = target.didEnter(fromPointerDown, enabled)
		}
	}
}

// DebugDescription is "target selection".
//
// Flutter: cupertino/dialog.dart _TargetSelectionGestureRecognizer.debugDescription
func (r *targetSelectionRecognizer) DebugDescription() string { return "target selection" }

// actionSheetGestureDetector wraps the buttons of a dialog or an action sheet so that a finger can slide over them.
//
// Flutter: cupertino/dialog.dart _ActionSheetGestureDetector
type actionSheetGestureDetector struct {
	widgets.StatelessWidget
	child widgets.Widget
}

func newActionSheetGestureDetector(child widgets.Widget) *actionSheetGestureDetector {
	return &actionSheetGestureDetector{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), child: child}
}

func (w *actionSheetGestureDetector) Build(bc widgets.BuildContext) widgets.Widget {
	binding := widgets.WidgetsBindingOf(bc)
	viewID := widgets.ViewOf(bc).ViewID()
	hitTest := func(globalPosition ui.Offset) *gestures.HitTestResult {
		result := gestures.NewHitTestResult()
		binding.Renderer().HitTestInView(result, globalPosition, viewID)
		return result
	}
	return widgets.NewRawGestureDetector(w.child, widgets.RawGestureDetectorOpts{
		ExcludeFromSemantics: true,
		Gestures: map[reflect.Type]widgets.GestureRecognizerFactory{
			reflect.TypeFor[*targetSelectionRecognizer](): widgets.NewGestureRecognizerFactoryWithHandlers(
				func() *targetSelectionRecognizer {
					return newTargetSelectionRecognizer(binding.Gestures(), w, hitTest)
				}, func(*targetSelectionRecognizer) {}),
		},
	})
}
