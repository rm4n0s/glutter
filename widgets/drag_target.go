package widgets

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// DragTargetBuilder builds the contents of a DragTarget from the data of the drags
// over it: candidateData are the data of the drags that would be accepted if dropped
// now, rejectedData the data of the drags the target refuses.
//
// Flutter: widgets/drag_target.dart DragTargetBuilder
type DragTargetBuilder[T any] func(bc BuildContext, candidateData []foundation.Opt[T], rejectedData []any) Widget

// DragTargetWillAcceptWithDetails decides whether a DragTarget accepts the drag
// that is over it.
//
// Flutter: widgets/drag_target.dart DragTargetWillAcceptWithDetails
type DragTargetWillAcceptWithDetails[T any] func(details DragTargetDetails[T]) bool

// DragTargetAcceptWithDetails is called when a drag that the target accepted is
// dropped on it.
//
// Flutter: widgets/drag_target.dart DragTargetAcceptWithDetails
type DragTargetAcceptWithDetails[T any] func(details DragTargetDetails[T])

// DragTargetLeave is called when a drag leaves a DragTarget, or is dropped
// elsewhere; the data is None if the drag has none.
//
// Flutter: widgets/drag_target.dart DragTargetLeave
type DragTargetLeave[T any] func(data foundation.Opt[T])

// DragTargetMove is called when a drag moves over a DragTarget.
//
// Flutter: widgets/drag_target.dart DragTargetMove
type DragTargetMove[T any] func(details DragTargetDetails[T])

// DragUpdateCallback is called when a Draggable moves.
//
// Flutter: widgets/drag_target.dart DragUpdateCallback
type DragUpdateCallback func(details gestures.DragUpdateDetails)

// DraggableCanceledCallback is called when a Draggable is dropped where no target
// accepts it.
//
// Flutter: widgets/drag_target.dart DraggableCanceledCallback
type DraggableCanceledCallback func(velocity gestures.Velocity, offset ui.Offset)

// DragEndCallback is called when a Draggable is dropped.
//
// Flutter: widgets/drag_target.dart DragEndCallback
type DragEndCallback func(details DraggableDetails)

// DragAnchorStrategy says where the feedback of a Draggable is placed relative to
// the pointer: it returns the offset of the pointer in the feedback. The draggable
// is the widget (a *Draggable or *LongPressDraggable).
//
// Flutter: widgets/drag_target.dart DragAnchorStrategy
type DragAnchorStrategy func(draggable Widget, bc BuildContext, position ui.Offset) ui.Offset

// ChildDragAnchorStrategy puts the feedback where the child is: the pointer keeps
// its place in the child.
//
// Flutter: widgets/drag_target.dart childDragAnchorStrategy
func ChildDragAnchorStrategy(draggable Widget, bc BuildContext, position ui.Offset) ui.Offset {
	box := bc.FindRenderObject().(rendering.Box)
	return box.GlobalToLocal(position, nil)
}

// PointerDragAnchorStrategy puts the top left of the feedback at the pointer.
//
// Flutter: widgets/drag_target.dart pointerDragAnchorStrategy
func PointerDragAnchorStrategy(draggable Widget, bc BuildContext, position ui.Offset) ui.Offset {
	return ui.Offset{}
}

// DraggableDetails says how a drag of a Draggable ended.
//
// Flutter: widgets/drag_target.dart DraggableDetails
type DraggableDetails struct {
	// WasAccepted is whether a DragTarget accepted the drop.
	//
	// Flutter: widgets/drag_target.dart DraggableDetails.wasAccepted
	WasAccepted bool
	// Velocity is the velocity of the drag when it ended.
	//
	// Flutter: widgets/drag_target.dart DraggableDetails.velocity
	Velocity gestures.Velocity
	// Offset is where the feedback was when the drag ended, in global coordinates.
	//
	// Flutter: widgets/drag_target.dart DraggableDetails.offset
	Offset ui.Offset
}

// DragTargetDetails is the data of a drag over a DragTarget and where it is.
//
// Flutter: widgets/drag_target.dart DragTargetDetails
type DragTargetDetails[T any] struct {
	// Data is the data of the Draggable.
	//
	// Flutter: widgets/drag_target.dart DragTargetDetails.data
	Data T
	// Offset is the global position of the feedback, the pointer minus the anchor.
	//
	// Flutter: widgets/drag_target.dart DragTargetDetails.offset
	Offset ui.Offset
}

// DragExport is what a Draggable offers to other applications, and to other
// windows, through the platform's drag and drop (Glutter's addition; Flutter has no
// drag and drop between applications).
//
// Flutter: no counterpart
type DragExport struct {
	// Text is offered as text/plain.
	Text string
	// URIs are offered as text/uri-list; a file is a file:// URI.
	URIs []string
	// Actions are the actions the app allows; the default is copy.
	Actions ui.DropActions
}

// ExternalDragData is the data of a drag that comes from the platform: from another
// application, or from a Draggable with an Export. A DragTarget whose T accepts a
// *ExternalDragData (such as any) is offered these drags. While the drag is over
// the target, only Types is known; the content is there when it is dropped.
//
// Flutter: no counterpart
type ExternalDragData struct {
	// Types are the MIME types the source offers, such as "text/uri-list" or
	// "text/plain".
	Types []string
	// Actions are the actions the source allows.
	Actions ui.DropActions
	// Text is the text of the drag, after the drop.
	Text string
	// URIs are the URIs of the drag, after the drop.
	URIs []string
	// Dropped is whether the content is there: the drag was dropped.
	Dropped bool
}

// HasType reports whether the source offers the MIME type.
//
// Flutter: no counterpart
func (d *ExternalDragData) HasType(mime string) bool {
	for _, t := range d.Types {
		if t == mime {
			return true
		}
	}
	return false
}

// HasFiles reports whether the source offers files (text/uri-list).
//
// Flutter: no counterpart
func (d *ExternalDragData) HasFiles() bool { return d.HasType("text/uri-list") }

// Files are the paths of the file:// URIs of the drag, after the drop.
//
// Flutter: no counterpart
func (d *ExternalDragData) Files() []string {
	var out []string
	for _, u := range d.URIs {
		if p, ok := fileURIPath(u); ok {
			out = append(out, p)
		}
	}
	return out
}

// DraggableOpts are the optional parameters of NewDraggable.
//
// Flutter: widgets/drag_target.dart Draggable.new
type DraggableOpts[T any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Data is the data that a DragTarget receives when the drag is dropped on it.
	Data foundation.Opt[T]
	// Axis restricts the drag to one axis; none for any direction.
	Axis foundation.Opt[painting.Axis]
	// ChildWhenDragging is shown instead of the child while a drag is going on.
	ChildWhenDragging Widget
	// FeedbackOffset is the offset of the feedback from the pointer's hit test
	// position.
	FeedbackOffset ui.Offset
	// DragAnchorStrategy says where the feedback is relative to the pointer; the
	// default is ChildDragAnchorStrategy.
	DragAnchorStrategy DragAnchorStrategy
	// Affinity is the axis in which the drag competes with other gestures; none for
	// a drag that starts at once in any direction.
	Affinity foundation.Opt[painting.Axis]
	// MaxSimultaneousDrags is how many drags may go on at the same time; none for
	// any number.
	MaxSimultaneousDrags foundation.Opt[int]
	// OnDragStarted is called when a drag starts.
	OnDragStarted foundation.VoidCallback
	// OnDragUpdate is called when the drag moves.
	OnDragUpdate DragUpdateCallback
	// OnDraggableCanceled is called when the drag is dropped where no target
	// accepts it.
	OnDraggableCanceled DraggableCanceledCallback
	// OnDragEnd is called when the drag ends, however it ends.
	OnDragEnd DragEndCallback
	// OnDragCompleted is called when a target accepted the drop.
	OnDragCompleted foundation.VoidCallback
	// IgnoringFeedbackSemantics is whether the semantics of the feedback are left
	// out; the default is true.
	IgnoringFeedbackSemantics foundation.Opt[bool]
	// IgnoringFeedbackPointer is whether the feedback ignores the pointer; the
	// default is true.
	IgnoringFeedbackPointer foundation.Opt[bool]
	// RootOverlay is whether the feedback goes in the furthest Overlay.
	RootOverlay bool
	// HitTestBehavior is how the draggable behaves during hit testing; the default
	// is HitTestBehaviorDeferToChild.
	HitTestBehavior rendering.HitTestBehavior
	// AllowedButtonsFilter says which buttons may start a drag; the default allows
	// the primary button.
	AllowedButtonsFilter gestures.AllowedButtonsFilter
	// Slop is how far a pointer of any kind moves before the drag starts, so that a
	// click on a draggable row with a small movement stays a click; none for
	// Flutter's computeHitSlop (1 pixel for a mouse, the touch slop for the others).
	//
	// Deviation: Flutter has no such parameter (SPEC §12.10).
	Slop foundation.Opt[float64]
	// CancelOnEscape cancels a running drag of this draggable when the Escape key
	// goes down: the drag ends as if the pointer was cancelled (OnDraggableCanceled,
	// the OnLeave of the targets, the feedback removed).
	//
	// Deviation: Flutter has no such parameter (SPEC §12.10).
	CancelOnEscape bool
	// Export, when set, makes the drag a drag of the platform: other applications
	// and the other windows of the app can receive what it returns (Glutter's
	// addition; Flutter has no drag and drop between applications). The feedback is
	// then drawn by the platform, and DragTargets of this app receive the data of
	// the Draggable. Without platform support, or if the platform refuses, the drag
	// stays inside the window.
	Export func(data T) *DragExport
}

// Draggable is a widget that can be dragged from to a DragTarget. While the drag
// goes on, the feedback follows the pointer in the Overlay.
//
// Flutter: widgets/drag_target.dart Draggable
type Draggable[T any] struct {
	StatefulWidget
	child    Widget
	feedback Widget
	opts     DraggableOpts[T]
	// longPress is true for a LongPressDraggable: the drag starts after delay.
	longPress             bool
	hapticFeedbackOnStart bool
	delay                 time.Duration
}

// NewDraggable creates a draggable widget. The feedback is shown under the pointer
// during the drag.
//
// Flutter: widgets/drag_target.dart Draggable.new
func NewDraggable[T any](feedback, child Widget, opts ...DraggableOpts[T]) *Draggable[T] {
	o := firstOpts(opts)
	checkMaxSimultaneousDrags(o.MaxSimultaneousDrags)
	return &Draggable[T]{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child: child, feedback: feedback, opts: o.withDefaults()}
}

func checkMaxSimultaneousDrags(n foundation.Opt[int]) {
	if v, ok := n.Get(); ok && v < 0 && foundation.DebugMode {
		panic(errors.New(tagDraggableMaxSimultaneousDragsNegative, "maxSimultaneousDrags must not be negative", "maxSimultaneousDrags", v))
	}
}

func (o DraggableOpts[T]) withDefaults() DraggableOpts[T] {
	if o.DragAnchorStrategy == nil {
		o.DragAnchorStrategy = ChildDragAnchorStrategy
	}
	if o.HitTestBehavior == rendering.HitTestBehaviorUnspecified {
		o.HitTestBehavior = rendering.HitTestBehaviorDeferToChild
	}
	return o
}

// Child is the widget below this one in the tree, shown when no drag goes on.
//
// Flutter: widgets/drag_target.dart Draggable.child
func (w *Draggable[T]) Child() Widget { return w.child }

// Feedback is the widget shown under the pointer during the drag.
//
// Flutter: widgets/drag_target.dart Draggable.feedback
func (w *Draggable[T]) Feedback() Widget { return w.feedback }

// Data is the data that a DragTarget receives.
//
// Flutter: widgets/drag_target.dart Draggable.data
func (w *Draggable[T]) Data() foundation.Opt[T] { return w.opts.Data }

// Axis is the axis the drag is restricted to.
//
// Flutter: widgets/drag_target.dart Draggable.axis
func (w *Draggable[T]) Axis() foundation.Opt[painting.Axis] { return w.opts.Axis }

// Affinity is the axis in which the drag competes with other gestures.
//
// Flutter: widgets/drag_target.dart Draggable.affinity
func (w *Draggable[T]) Affinity() foundation.Opt[painting.Axis] { return w.opts.Affinity }

// MaxSimultaneousDrags is how many drags may go on at once.
//
// Flutter: widgets/drag_target.dart Draggable.maxSimultaneousDrags
func (w *Draggable[T]) MaxSimultaneousDrags() foundation.Opt[int] { return w.opts.MaxSimultaneousDrags }

// createRecognizer makes the recognizer that starts the drag: a long press for a
// LongPressDraggable, else one by the affinity.
//
// Flutter: widgets/drag_target.dart Draggable.createRecognizer
func (w *Draggable[T]) createRecognizer(g *gestures.GestureBinding, haptic *services.HapticFeedback, onStart gestures.GestureMultiDragStartCallback) *gestures.MultiDragGestureRecognizer {
	o := gestures.MultiDragGestureRecognizerOpts{DebugOwner: w, AllowedButtonsFilter: w.opts.AllowedButtonsFilter, Slop: w.opts.Slop}
	var r *gestures.MultiDragGestureRecognizer
	if w.longPress {
		start := onStart
		onStart = func(position ui.Offset) gestures.Drag {
			result := start(position)
			if result != nil && w.hapticFeedbackOnStart {
				haptic.SelectionClick()
			}
			return result
		}
		d := gestures.NewDelayedMultiDragGestureRecognizer(g, gestures.DelayedMultiDragGestureRecognizerOpts{
			MultiDragGestureRecognizerOpts: o, Delay: foundation.Some(w.delay)})
		r = &d.MultiDragGestureRecognizer
	} else {
		switch axis, ok := w.opts.Affinity.Get(); {
		case ok && axis == painting.AxisHorizontal:
			r = &gestures.NewHorizontalMultiDragGestureRecognizer(g, o).MultiDragGestureRecognizer
		case ok && axis == painting.AxisVertical:
			r = &gestures.NewVerticalMultiDragGestureRecognizer(g, o).MultiDragGestureRecognizer
		default:
			r = &gestures.NewImmediateMultiDragGestureRecognizer(g, o).MultiDragGestureRecognizer
		}
	}
	r.OnStart = onStart
	return r
}

// HapticFeedbackOnStart is whether a LongPressDraggable gives haptic feedback when
// the drag starts. The desktop has no haptic feedback, so it does nothing there.
//
// Flutter: widgets/drag_target.dart LongPressDraggable.hapticFeedbackOnStart
func (w *Draggable[T]) HapticFeedbackOnStart() bool { return w.hapticFeedbackOnStart }

// Delay is how long the pointer must stay down before a LongPressDraggable starts the
// drag.
//
// Flutter: widgets/drag_target.dart LongPressDraggable.delay
func (w *Draggable[T]) Delay() time.Duration { return w.delay }

// CreateState creates the state.
//
// Flutter: widgets/drag_target.dart Draggable.createState
func (w *Draggable[T]) CreateState() State { return &draggableState[T]{} }

// LongPressDraggableOpts are the optional parameters of NewLongPressDraggable.
//
// Flutter: widgets/drag_target.dart LongPressDraggable.new
type LongPressDraggableOpts[T any] struct {
	DraggableOpts[T]
	// NoHapticFeedbackOnStart turns off the haptic feedback when the drag starts.
	// The desktop has no haptic feedback, so it changes nothing there.
	NoHapticFeedbackOnStart bool
	// Delay is how long the pointer must stay down before the drag starts; the
	// default is gestures.LongPressTimeout.
	Delay foundation.Opt[time.Duration]
}

// LongPressDraggable is a Draggable that starts a drag after a long press. It is
// the same type as Draggable (Deviation: SPEC §12.10).
//
// Flutter: widgets/drag_target.dart LongPressDraggable
type LongPressDraggable[T any] = Draggable[T]

// NewLongPressDraggable creates a draggable that starts a drag after a long press.
//
// Flutter: widgets/drag_target.dart LongPressDraggable.new
func NewLongPressDraggable[T any](feedback, child Widget, opts ...LongPressDraggableOpts[T]) *LongPressDraggable[T] {
	o := firstOpts(opts)
	w := NewDraggable[T](feedback, child, o.DraggableOpts)
	w.longPress = true
	w.hapticFeedbackOnStart = !o.NoHapticFeedbackOnStart
	w.delay = o.Delay.OrElse(gestures.LongPressTimeout)
	return w
}

type draggableState[T any] struct {
	StateOf[*Draggable[T]]
	// The recognizer has an unusual lifetime: a Draggable can be removed from the
	// tree in the middle of a drag, and the recognizer, which gets the pointer
	// events that drive the drag, must stay alive. It is disposed once the state is
	// disposed and no drag is active.
	recognizer  *gestures.MultiDragGestureRecognizer
	activeCount int
}

func (s *draggableState[T]) InitState() {
	s.recognizer = s.Widget().createRecognizer(WidgetsBindingOf(s.Context()).Gestures(), WidgetsBindingOf(s.Context()).HapticFeedback(), s.startDrag)
}

func (s *draggableState[T]) Dispose() { s.disposeRecognizerIfInactive() }

func (s *draggableState[T]) DidChangeDependencies() {
	if settings, ok := MediaQueryMaybeGestureSettingsOf(s.Context()); ok {
		s.recognizer.GestureSettings = foundation.Some(settings)
	} else {
		s.recognizer.GestureSettings = foundation.None[gestures.DeviceGestureSettings]()
	}
}

func (s *draggableState[T]) disposeRecognizerIfInactive() {
	if s.activeCount > 0 {
		return
	}
	s.recognizer.Dispose()
}

func (s *draggableState[T]) routePointer(event *gestures.PointerDownEvent) {
	if max, ok := s.Widget().opts.MaxSimultaneousDrags.Get(); ok && s.activeCount >= max {
		return
	}
	s.recognizer.AddPointer(event)
}

func (s *draggableState[T]) startDrag(position ui.Offset) gestures.Drag {
	w := s.Widget()
	o := &w.opts
	if max, ok := o.MaxSimultaneousDrags.Get(); ok && s.activeCount >= max {
		return nil
	}
	bc := s.Context()
	dragStartPoint := o.DragAnchorStrategy(w, bc, position)
	s.SetState(func() { s.activeCount++ })

	var data any
	if v, ok := o.Data.Get(); ok {
		data = v
	}
	view := ViewOf(bc)
	// stopKeys removes the Escape handler of CancelOnEscape once the drag is over.
	stopKeys := func() {}
	end := func(velocity gestures.Velocity, offset ui.Offset, wasAccepted bool) {
		stopKeys()
		if s.Mounted() {
			s.SetState(func() { s.activeCount-- })
		} else {
			s.activeCount--
			s.disposeRecognizerIfInactive()
		}
		if s.Mounted() && o.OnDragEnd != nil {
			o.OnDragEnd(DraggableDetails{WasAccepted: wasAccepted, Velocity: velocity, Offset: offset})
		}
		if wasAccepted && o.OnDragCompleted != nil {
			o.OnDragCompleted()
		}
		if !wasAccepted && o.OnDraggableCanceled != nil {
			o.OnDraggableCanceled(velocity, offset)
		}
	}
	update := func(details gestures.DragUpdateDetails) {
		if s.Mounted() && o.OnDragUpdate != nil {
			o.OnDragUpdate(details)
		}
	}

	var drag gestures.Drag
	if export := o.Export; export != nil && o.Data.IsSome() {
		if exported := startExportedDrag(WidgetsBindingOf(bc), view.ViewID(), position, dragStartPoint, export(o.Data.Value()), data, end); exported != nil {
			drag = exported
		}
	}
	if drag == nil {
		overlay := OverlayOf(bc, OverlayOfOpts{DebugRequiredFor: w, RootOverlay: o.RootOverlay})
		drag = newDragAvatar(dragAvatarArgs{
			binding: WidgetsBindingOf(bc), overlayState: overlay, data: data, axis: o.Axis, initialPosition: position,
			dragStartPoint: dragStartPoint, feedback: w.feedback, feedbackOffset: o.FeedbackOffset,
			onDragUpdate: update, onDragEnd: end,
			ignoringFeedbackSemantics: o.IgnoringFeedbackSemantics.OrElse(true),
			ignoringFeedbackPointer:   o.IgnoringFeedbackPointer.OrElse(true),
			viewID:                    view.ViewID(),
		})
	}
	if o.CancelOnEscape {
		keyboard := WidgetsBindingOf(bc).Keyboard()
		handle := keyboard.AddHandler(func(event services.KeyEvent) bool {
			if _, ok := event.(*services.KeyDownEvent); ok && event.LogicalKey() == services.LogicalKeyboardKeyEscape {
				drag.Cancel()
				return true
			}
			return false
		})
		stopKeys = func() { keyboard.RemoveHandler(handle) }
	}
	if o.OnDragStarted != nil {
		o.OnDragStarted()
	}
	return drag
}

func (s *draggableState[T]) Build(bc BuildContext) Widget {
	if foundation.DebugMode {
		DebugCheckHasOverlay(bc)
	}
	w := s.Widget()
	canDrag := true
	if max, ok := w.opts.MaxSimultaneousDrags.Get(); ok {
		canDrag = s.activeCount < max
	}
	showChild := s.activeCount == 0 || w.opts.ChildWhenDragging == nil
	var onDown rendering.PointerDownEventListener
	if canDrag {
		onDown = s.routePointer
	}
	child := w.child
	if !showChild {
		child = w.opts.ChildWhenDragging
	}
	return NewListener(child, ListenerOpts{Behavior: w.opts.HitTestBehavior, OnPointerDown: onDown})
}

// DragTargetOpts are the optional parameters of NewDragTarget.
//
// Flutter: widgets/drag_target.dart DragTarget.new
type DragTargetOpts[T any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnWillAcceptWithDetails decides whether the target accepts the drag over it;
	// without it every drag with data is accepted.
	OnWillAcceptWithDetails DragTargetWillAcceptWithDetails[T]
	// OnAcceptWithDetails is called when a drag is dropped on the target and the
	// target accepted it.
	OnAcceptWithDetails DragTargetAcceptWithDetails[T]
	// OnLeave is called when a drag leaves the target or is dropped elsewhere.
	OnLeave DragTargetLeave[T]
	// OnMove is called when a drag moves over the target.
	OnMove DragTargetMove[T]
	// HitTestBehavior is how the target behaves during hit testing; the default is
	// HitTestBehaviorTranslucent.
	HitTestBehavior rendering.HitTestBehavior
	// RepeatMove calls OnMove once per frame while a drag is over the target, also
	// when the pointer does not move, so that a target can scroll or expand while the
	// pointer rests over it. A frame is scheduled only while such a drag is over such
	// a target.
	//
	// Deviation: Flutter calls onMove only when the pointer moves (SPEC §12.10).
	RepeatMove bool
}

// DragTarget is a widget that receives the data of a Draggable when it is dropped
// on it. It also receives the drags of the platform, as *ExternalDragData, when T
// accepts it.
//
// Flutter: widgets/drag_target.dart DragTarget
type DragTarget[T any] struct {
	StatefulWidget
	builder DragTargetBuilder[T]
	opts    DragTargetOpts[T]
}

// NewDragTarget creates a widget that receives drags. The builder builds its
// contents from the data of the drags over it.
//
// Flutter: widgets/drag_target.dart DragTarget.new
func NewDragTarget[T any](builder DragTargetBuilder[T], opts ...DragTargetOpts[T]) *DragTarget[T] {
	o := firstOpts(opts)
	if o.HitTestBehavior == rendering.HitTestBehaviorUnspecified {
		o.HitTestBehavior = rendering.HitTestBehaviorTranslucent
	}
	return &DragTarget[T]{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), builder: builder, opts: o}
}

// Builder builds the contents of the target.
//
// Flutter: widgets/drag_target.dart DragTarget.builder
func (w *DragTarget[T]) Builder() DragTargetBuilder[T] { return w.builder }

// HitTestBehavior is how the target behaves during hit testing.
//
// Flutter: widgets/drag_target.dart DragTarget.hitTestBehavior
func (w *DragTarget[T]) HitTestBehavior() rendering.HitTestBehavior { return w.opts.HitTestBehavior }

// CreateState creates the state.
//
// Flutter: widgets/drag_target.dart DragTarget.createState
func (w *DragTarget[T]) CreateState() State { return &dragTargetState[T]{} }

// dragTargetRef is a DragTargetState of any T: what the hit test finds in the
// MetaData and what a drag avatar talks to.
//
// Flutter: widgets/drag_target.dart _DragTargetState<Object>
type dragTargetRef interface {
	isExpectedDataType(data any) bool
	didEnter(avatar *dragAvatar) bool
	didLeave(avatar *dragAvatar)
	didDrop(avatar *dragAvatar)
	didMove(avatar *dragAvatar)
	repeatsMove() bool
}

type dragTargetState[T any] struct {
	StateOf[*DragTarget[T]]
	candidateAvatars []*dragAvatar
	rejectedAvatars  []*dragAvatar
}

// isExpectedDataType is whether the data is a T (or none).
//
// Flutter: widgets/drag_target.dart _DragTargetState.isExpectedDataType
func (s *dragTargetState[T]) isExpectedDataType(data any) bool {
	if data == nil {
		return true
	}
	_, ok := data.(T)
	return ok
}

func (s *dragTargetState[T]) dataOf(avatar *dragAvatar) (T, bool) {
	v, ok := avatar.data.(T)
	return v, ok
}

func (s *dragTargetState[T]) didEnter(avatar *dragAvatar) bool {
	willAccept := s.Widget().opts.OnWillAcceptWithDetails
	resolved := willAccept == nil
	if !resolved && avatar.data != nil {
		if v, ok := s.dataOf(avatar); ok {
			resolved = willAccept(DragTargetDetails[T]{Data: v, Offset: avatar.lastOffset})
		}
	}
	if resolved {
		s.SetState(func() { s.candidateAvatars = append(s.candidateAvatars, avatar) })
		return true
	}
	s.SetState(func() { s.rejectedAvatars = append(s.rejectedAvatars, avatar) })
	return false
}

func (s *dragTargetState[T]) didLeave(avatar *dragAvatar) {
	if !s.Mounted() {
		return
	}
	s.SetState(func() {
		s.candidateAvatars = removeAvatar(s.candidateAvatars, avatar)
		s.rejectedAvatars = removeAvatar(s.rejectedAvatars, avatar)
	})
	if leave := s.Widget().opts.OnLeave; leave != nil {
		leave(s.optData(avatar))
	}
}

func (s *dragTargetState[T]) optData(avatar *dragAvatar) foundation.Opt[T] {
	if v, ok := s.dataOf(avatar); ok {
		return foundation.Some(v)
	}
	return foundation.None[T]()
}

func (s *dragTargetState[T]) didDrop(avatar *dragAvatar) {
	if !s.Mounted() {
		return
	}
	s.SetState(func() { s.candidateAvatars = removeAvatar(s.candidateAvatars, avatar) })
	if avatar.data == nil {
		return
	}
	if v, ok := s.dataOf(avatar); ok {
		if accept := s.Widget().opts.OnAcceptWithDetails; accept != nil {
			accept(DragTargetDetails[T]{Data: v, Offset: avatar.lastOffset})
		}
	}
}

func (s *dragTargetState[T]) repeatsMove() bool { return s.Widget().opts.RepeatMove }

func (s *dragTargetState[T]) didMove(avatar *dragAvatar) {
	if !s.Mounted() || avatar.data == nil {
		return
	}
	if v, ok := s.dataOf(avatar); ok {
		if move := s.Widget().opts.OnMove; move != nil {
			move(DragTargetDetails[T]{Data: v, Offset: avatar.lastOffset})
		}
	}
}

func removeAvatar(list []*dragAvatar, avatar *dragAvatar) []*dragAvatar {
	out := list[:0:0]
	for _, a := range list {
		if a != avatar {
			out = append(out, a)
		}
	}
	return out
}

func (s *dragTargetState[T]) Build(bc BuildContext) Widget {
	w := s.Widget()
	candidates := make([]foundation.Opt[T], len(s.candidateAvatars))
	for i, a := range s.candidateAvatars {
		candidates[i] = s.optData(a)
	}
	rejected := make([]any, len(s.rejectedAvatars))
	for i, a := range s.rejectedAvatars {
		rejected[i] = a.data
	}
	return NewMetaData(w.builder(bc, candidates, rejected), MetaDataOpts{MetaData: dragTargetRef(s), Behavior: w.opts.HitTestBehavior})
}

type dragEndKind int

const (
	dragEndDropped dragEndKind = iota
	dragEndCanceled
)

type dragAvatarArgs struct {
	binding                   *WidgetsBinding
	overlayState              *OverlayState
	data                      any
	axis                      foundation.Opt[painting.Axis]
	initialPosition           ui.Offset
	dragStartPoint            ui.Offset
	feedback                  Widget
	feedbackOffset            ui.Offset
	onDragUpdate              DragUpdateCallback
	onDragEnd                 func(velocity gestures.Velocity, offset ui.Offset, wasAccepted bool)
	ignoringFeedbackSemantics bool
	ignoringFeedbackPointer   bool
	viewID                    ui.ViewID
}

// dragAvatar is the drag that a Draggable (or the platform) is going on with: it
// shows the feedback, hit tests for the targets under the pointer and tells them
// of the drag.
//
// Flutter: widgets/drag_target.dart _DragAvatar
type dragAvatar struct {
	dragAvatarArgs

	activeTarget   dragTargetRef
	enteredTargets []dragTargetRef
	position       ui.Offset
	lastOffset     ui.Offset
	overlayOffset  ui.Offset
	entry          *OverlayEntry
	// finished is whether the drag is over.
	finished bool
	// repeatID is the frame callback that reports the moves of RepeatMove targets, or 0.
	repeatID int
}

func newDragAvatar(args dragAvatarArgs) *dragAvatar {
	a := &dragAvatar{dragAvatarArgs: args, position: args.initialPosition}
	if args.overlayState != nil {
		a.entry = NewOverlayEntry(a.build)
		args.overlayState.Insert(a.entry)
	}
	a.updateDrag(args.initialPosition)
	return a
}

// Update is called when the pointer moves.
//
// Flutter: widgets/drag_target.dart _DragAvatar.update
func (a *dragAvatar) Update(details gestures.DragUpdateDetails) {
	if a.finished {
		return // cancelled while the pointer is still down (the Escape key)
	}
	oldPosition := a.position
	a.position = a.position.Add(a.restrictAxis(details.Delta))
	a.updateDrag(a.position)
	if a.onDragUpdate != nil && a.position != oldPosition {
		a.onDragUpdate(details)
	}
}

// End is called when the pointer is lifted.
//
// Flutter: widgets/drag_target.dart _DragAvatar.end
func (a *dragAvatar) End(details gestures.DragEndDetails) {
	a.finishDrag(dragEndDropped, foundation.Some(a.restrictVelocityAxis(details.Velocity)))
}

// Cancel is called when the drag is cancelled.
//
// Flutter: widgets/drag_target.dart _DragAvatar.cancel
func (a *dragAvatar) Cancel() { a.finishDrag(dragEndCanceled, foundation.None[gestures.Velocity]()) }

func (a *dragAvatar) updateDrag(globalPosition ui.Offset) {
	a.updateDragReport(globalPosition, false)
	a.scheduleRepeat()
}

// updateDragReport hits tests at the position and tells the targets. On a repeat (a frame
// with no movement) only the targets with RepeatMove get the move.
func (a *dragAvatar) updateDragReport(globalPosition ui.Offset, repeat bool) {
	a.lastOffset = globalPosition.Sub(a.dragStartPoint)
	if a.overlayState != nil && a.overlayState.Mounted() {
		box := a.overlayState.Context().FindRenderObject().(rendering.Box)
		overlaySpaceOffset := box.GlobalToLocal(globalPosition, nil)
		a.overlayOffset = overlaySpaceOffset.Sub(a.dragStartPoint)
		if a.entry != nil {
			a.entry.MarkNeedsBuild()
		}
	}

	result := gestures.NewHitTestResult()
	a.binding.Renderer().HitTestInView(result, globalPosition.Add(a.feedbackOffset), a.viewID)

	targets := a.getDragTargets(result.Path())

	listsMatch := false
	if len(targets) >= len(a.enteredTargets) && len(a.enteredTargets) > 0 {
		listsMatch = true
		for i, entered := range a.enteredTargets {
			if targets[i] != entered {
				listsMatch = false
				break
			}
		}
	}

	// If everything is the same, report moves and be done.
	if listsMatch {
		for _, target := range append([]dragTargetRef(nil), a.enteredTargets...) {
			if !repeat || target.repeatsMove() {
				target.didMove(a)
			}
		}
		return
	}

	// Leave the old targets.
	a.leaveAllEntered()

	// Enter the new targets: the first that accepts is the active one.
	var newTarget dragTargetRef
	for _, target := range targets {
		a.enteredTargets = append(a.enteredTargets, target)
		if target.didEnter(a) {
			newTarget = target
			break
		}
	}

	// Report moves to the targets.
	for _, target := range append([]dragTargetRef(nil), a.enteredTargets...) {
		target.didMove(a)
	}
	a.activeTarget = newTarget
}

// getDragTargets finds the DragTargets in the hit test path: the MetaData they build
// holds them.
//
// Flutter: widgets/drag_target.dart _DragAvatar._getDragTargets
func (a *dragAvatar) getDragTargets(path []gestures.HitTestEntry) []dragTargetRef {
	var out []dragTargetRef
	for _, entry := range path {
		meta, ok := entry.Target().(*rendering.RenderMetaData)
		if !ok {
			continue
		}
		if target, ok := meta.MetaData.(dragTargetRef); ok && target.isExpectedDataType(a.data) {
			out = append(out, target)
		}
	}
	return out
}

func (a *dragAvatar) leaveAllEntered() {
	for _, target := range a.enteredTargets {
		target.didLeave(a)
	}
	a.enteredTargets = nil
}

// wantsRepeat is whether a target the drag is over has RepeatMove.
func (a *dragAvatar) wantsRepeat() bool {
	for _, t := range a.enteredTargets {
		if t.repeatsMove() {
			return true
		}
	}
	return false
}

// scheduleRepeat schedules the frame that reports the moves of the RepeatMove targets,
// if the drag is over one and no frame is waiting. Nothing is scheduled otherwise.
func (a *dragAvatar) scheduleRepeat() {
	if a.finished || a.repeatID != 0 || a.binding == nil || !a.wantsRepeat() {
		return
	}
	a.repeatID = a.binding.Scheduler().ScheduleFrameCallback(func(time.Duration) {
		a.repeatID = 0
		if a.finished {
			return
		}
		a.updateDragReport(a.position, true)
		a.scheduleRepeat()
	})
}

func (a *dragAvatar) finishDrag(kind dragEndKind, velocity foundation.Opt[gestures.Velocity]) {
	if a.finished {
		return
	}
	a.finished = true
	if a.repeatID != 0 {
		a.binding.Scheduler().CancelFrameCallbackWithID(a.repeatID)
		a.repeatID = 0
	}
	wasAccepted := false
	if kind == dragEndDropped && a.activeTarget != nil {
		a.activeTarget.didDrop(a)
		wasAccepted = true
		a.enteredTargets = removeTarget(a.enteredTargets, a.activeTarget)
	}
	a.leaveAllEntered()
	a.activeTarget = nil
	if a.entry != nil {
		a.entry.Remove()
		a.entry.Dispose()
		a.entry = nil
	}
	if a.onDragEnd != nil {
		a.onDragEnd(velocity.OrElse(gestures.Velocity{}), a.lastOffset, wasAccepted)
	}
}

func removeTarget(list []dragTargetRef, target dragTargetRef) []dragTargetRef {
	out := list[:0:0]
	for _, t := range list {
		if t != target {
			out = append(out, t)
		}
	}
	return out
}

func (a *dragAvatar) build(bc BuildContext) Widget {
	return NewPositioned(
		NewExcludeSemantics(
			NewIgnorePointer(a.feedback, IgnorePointerOpts{Ignoring: foundation.Some(a.ignoringFeedbackPointer)}),
			ExcludeSemanticsOpts{Excluding: foundation.Some(a.ignoringFeedbackSemantics)}),
		PositionedOpts{Left: foundation.Some(a.overlayOffset.Dx), Top: foundation.Some(a.overlayOffset.Dy)})
}

func (a *dragAvatar) restrictVelocityAxis(v gestures.Velocity) gestures.Velocity {
	if !a.axis.IsSome() {
		return v
	}
	return gestures.Velocity{PixelsPerSecond: a.restrictAxis(v.PixelsPerSecond)}
}

func (a *dragAvatar) restrictAxis(offset ui.Offset) ui.Offset {
	axis, ok := a.axis.Get()
	switch {
	case ok && axis == painting.AxisHorizontal:
		return ui.Offset{Dx: offset.Dx}
	case ok && axis == painting.AxisVertical:
		return ui.Offset{Dy: offset.Dy}
	}
	return offset
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDraggableMaxSimultaneousDragsNegative = "DraggableMaxSimultaneousDragsNegative"
)
