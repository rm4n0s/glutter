package rendering

import (
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// RenderAnimatedSizeState is the state of a RenderAnimatedSize.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSizeState
type RenderAnimatedSizeState int

// The states of a RenderAnimatedSize.
const (
	// RenderAnimatedSizeStateUnspecified is the zero value; it is not a real state.
	//
	// Flutter: no counterpart
	RenderAnimatedSizeStateUnspecified RenderAnimatedSizeState = iota
	// RenderAnimatedSizeStateStart is the initial state: the box has not seen the
	// child's size yet.
	//
	// Flutter: rendering/animated_size.dart RenderAnimatedSizeState.start
	RenderAnimatedSizeStateStart
	// RenderAnimatedSizeStateStable means the child's size has not changed since
	// the last layout.
	//
	// Flutter: rendering/animated_size.dart RenderAnimatedSizeState.stable
	RenderAnimatedSizeStateStable
	// RenderAnimatedSizeStateChanged means the child's size changed, and the box
	// is animating to it.
	//
	// Flutter: rendering/animated_size.dart RenderAnimatedSizeState.changed
	RenderAnimatedSizeStateChanged
	// RenderAnimatedSizeStateUnstable means the child's size changed more than
	// once in a row; the box follows it without animating.
	//
	// Flutter: rendering/animated_size.dart RenderAnimatedSizeState.unstable
	RenderAnimatedSizeStateUnstable
)

// String is the name of the state, such as "RenderAnimatedSizeState.stable".
//
// Flutter: rendering/animated_size.dart RenderAnimatedSizeState
func (s RenderAnimatedSizeState) String() string {
	switch s {
	case RenderAnimatedSizeStateStart:
		return "RenderAnimatedSizeState.start"
	case RenderAnimatedSizeStateStable:
		return "RenderAnimatedSizeState.stable"
	case RenderAnimatedSizeStateChanged:
		return "RenderAnimatedSizeState.changed"
	case RenderAnimatedSizeStateUnstable:
		return "RenderAnimatedSizeState.unstable"
	}
	return "RenderAnimatedSizeState.unspecified"
}

// RenderAnimatedSize is a render object that animates its size to its child's
// size over a given duration and with a given curve. If the child's size
// changes in the middle of the animation, the animation is restarted from the
// current size.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize
type RenderAnimatedSize struct {
	RenderAligningShiftedBox
	vsync          scheduler.TickerProvider
	clipBehavior   ui.Clip
	onEnd          func()
	controller     *animation.AnimationController
	animation      *animation.CurvedAnimation
	sizeTween      *animation.Tween[foundation.Opt[ui.Size]]
	hasVisualOver  bool
	lastValue      foundation.Opt[float64]
	state          RenderAnimatedSizeState
	currentSize    ui.Size
	clipRectLayer  *LayerHandle[*ClipRectLayer]
	statusHandle   animation.StatusListenerHandle
	controllerHdle foundation.ListenerHandle
}

// RenderAnimatedSizeOpts are the optional parameters of NewRenderAnimatedSize.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.new
type RenderAnimatedSizeOpts struct {
	// ReverseDuration is the duration when the size shrinks; the default is the
	// duration.
	ReverseDuration foundation.Opt[time.Duration]
	// Curve is the curve of the animation; the default is Curves.Linear.
	Curve animation.Curve
	// Alignment is how to align the child while the size animates; the default is
	// the center.
	Alignment painting.AlignmentGeometry
	// TextDirection resolves a directional alignment.
	TextDirection ui.TextDirection
	// ClipBehavior is how the child is clipped while it overflows; the default is
	// ClipHardEdge.
	ClipBehavior ui.Clip
	// OnEnd is called when the animation completes.
	OnEnd func()
}

// NewRenderAnimatedSize creates the box; the child may be nil.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.new
func NewRenderAnimatedSize(vsync scheduler.TickerProvider, duration time.Duration, child Box, opts ...RenderAnimatedSizeOpts) *RenderAnimatedSize {
	o := firstOpts(opts)
	if o.Curve == nil {
		o.Curve = animation.Curves.Linear
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	r := &RenderAnimatedSize{vsync: vsync, clipBehavior: o.ClipBehavior, onEnd: o.OnEnd, state: RenderAnimatedSizeStateStart,
		sizeTween:     animation.NewSizeTween(foundation.None[ui.Size](), foundation.None[ui.Size]()),
		clipRectLayer: NewLayerHandle[*ClipRectLayer](nil)}
	InitRenderAligningShiftedBox(&r.RenderAligningShiftedBox, r, o.Alignment, o.TextDirection, child)
	r.controller = animation.NewAnimationController(vsync, animation.AnimationControllerOpts{
		Duration: foundation.Some(duration), ReverseDuration: o.ReverseDuration})
	r.controllerHdle = r.controller.AddListener(func() {
		if last, ok := r.lastValue.Get(); !ok || r.controller.Value() != last {
			r.MarkNeedsLayout()
		}
	})
	r.animation = animation.NewCurvedAnimation(r.controller, o.Curve)
	return r
}

// DebugController is the animation controller, in debug builds.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.debugController
func (r *RenderAnimatedSize) DebugController() *animation.AnimationController {
	if foundation.DebugMode {
		return r.controller
	}
	return nil
}

// DebugAnimation is the curved animation, in debug builds.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.debugAnimation
func (r *RenderAnimatedSize) DebugAnimation() *animation.CurvedAnimation {
	if foundation.DebugMode {
		return r.animation
	}
	return nil
}

// State is the state of the box.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.state
func (r *RenderAnimatedSize) State() RenderAnimatedSizeState { return r.state }

// Duration is the duration of the animation.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.duration
func (r *RenderAnimatedSize) Duration() time.Duration { return r.controller.Duration.Value() }

// SetDuration changes the duration of the animation.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.duration
func (r *RenderAnimatedSize) SetDuration(value time.Duration) {
	if r.controller.Duration == foundation.Some(value) {
		return
	}
	r.controller.Duration = foundation.Some(value)
}

// ReverseDuration is the duration of the animation when the size shrinks.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.reverseDuration
func (r *RenderAnimatedSize) ReverseDuration() foundation.Opt[time.Duration] {
	return r.controller.ReverseDuration
}

// SetReverseDuration changes the duration of the animation when the size shrinks.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.reverseDuration
func (r *RenderAnimatedSize) SetReverseDuration(value foundation.Opt[time.Duration]) {
	if value == r.controller.ReverseDuration {
		return
	}
	r.controller.ReverseDuration = value
}

// Curve is the curve of the animation.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.curve
func (r *RenderAnimatedSize) Curve() animation.Curve { return r.animation.Curve }

// SetCurve changes the curve of the animation.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.curve
func (r *RenderAnimatedSize) SetCurve(value animation.Curve) {
	if animation.CurveEqual(value, r.animation.Curve) {
		return
	}
	r.animation.Curve = value
}

// ClipBehavior is how the child is clipped while it overflows.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.clipBehavior
func (r *RenderAnimatedSize) ClipBehavior() ui.Clip { return r.clipBehavior }

// SetClipBehavior changes the clip behavior and marks the box as needing paint.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.clipBehavior
func (r *RenderAnimatedSize) SetClipBehavior(value ui.Clip) {
	if value != r.clipBehavior {
		r.clipBehavior = value
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

// IsAnimating is whether the size is animating.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.isAnimating
func (r *RenderAnimatedSize) IsAnimating() bool { return r.controller.IsAnimating() }

// Vsync is the provider of the ticker of the animation.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.vsync
func (r *RenderAnimatedSize) Vsync() scheduler.TickerProvider { return r.vsync }

// SetVsync changes the provider; the running animation moves to its ticker.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.vsync
func (r *RenderAnimatedSize) SetVsync(value scheduler.TickerProvider) {
	if value == r.vsync {
		return
	}
	r.vsync = value
	r.controller.Resync(value)
}

// OnEnd is called when the animation completes.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.onEnd
func (r *RenderAnimatedSize) OnEnd() func() { return r.onEnd }

// SetOnEnd changes the callback.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.onEnd
func (r *RenderAnimatedSize) SetOnEnd(value func()) { r.onEnd = value }

// Attach starts listening to the status of the animation, and resumes an
// interrupted resizing.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.attach
func (r *RenderAnimatedSize) Attach(owner *PipelineOwner) {
	r.RenderAligningShiftedBox.Attach(owner)
	switch r.state {
	case RenderAnimatedSizeStateChanged, RenderAnimatedSizeStateUnstable:
		// Mark the box in case it is not dirty already, to resume the
		// interrupted resizing animation.
		r.MarkNeedsLayout()
	}
	r.statusHandle = r.controller.AddStatusListener(r.animationStatusListener)
}

// Detach stops the animation.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.detach
func (r *RenderAnimatedSize) Detach() {
	r.controller.Stop()
	r.controller.RemoveStatusListener(r.statusHandle)
	r.RenderAligningShiftedBox.Detach()
}

func (r *RenderAnimatedSize) animatedSize() foundation.Opt[ui.Size] {
	return r.sizeTween.Evaluate(r.animation)
}

// PerformLayout lays the child out and works out the size of this step of the
// animation.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.performLayout
func (r *RenderAnimatedSize) PerformLayout() {
	r.lastValue = foundation.Some(r.controller.Value())
	r.hasVisualOver = false
	constraints := r.BoxConstraints()
	if !r.hasChild() || constraints.IsTight() {
		r.controller.Stop()
		smallest := constraints.Smallest()
		r.currentSize = smallest
		r.SetSize(smallest)
		r.sizeTween.Begin, r.sizeTween.End = foundation.Some(smallest), foundation.Some(smallest)
		r.state = RenderAnimatedSizeStateStart
		if r.hasChild() {
			r.Child().Layout(constraints, LayoutOpts{})
		}
		return
	}
	child := r.Child()
	child.Layout(constraints, LayoutOpts{ParentUsesSize: true})
	switch r.state {
	case RenderAnimatedSizeStateStart:
		r.layoutStart()
	case RenderAnimatedSizeStateStable:
		r.layoutStable()
	case RenderAnimatedSizeStateChanged:
		r.layoutChanged()
	case RenderAnimatedSizeStateUnstable:
		r.layoutUnstable()
	}
	r.currentSize = constraints.Constrain(r.animatedSize().Value())
	r.SetSize(r.currentSize)
	r.AlignChild()
	end := r.sizeTween.End.Value()
	if r.Size().Width < end.Width || r.Size().Height < end.Height {
		r.hasVisualOver = true
	}
}

// ComputeDryLayout is the size of the current step of the animation, without
// changing the state.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.computeDryLayout
func (r *RenderAnimatedSize) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if !r.hasChild() || constraints.IsTight() {
		return constraints.Smallest()
	}
	// This is a simplified performLayout that only works out the current size
	// without changing any state; see PerformLayout.
	childSize := r.Child().GetDryLayout(constraints)
	switch r.state {
	case RenderAnimatedSizeStateStart:
		return constraints.Constrain(childSize)
	case RenderAnimatedSizeStateStable:
		if r.sizeTween.End != foundation.Some(childSize) {
			return constraints.Constrain(r.currentSize)
		} else if r.controller.Value() == r.controller.UpperBound {
			return constraints.Constrain(childSize)
		}
	case RenderAnimatedSizeStateUnstable, RenderAnimatedSizeStateChanged:
		if r.sizeTween.End != foundation.Some(childSize) {
			return constraints.Constrain(childSize)
		}
	}
	return constraints.Constrain(r.animatedSize().Value())
}

func (r *RenderAnimatedSize) restartAnimation() {
	r.lastValue = foundation.Some(0.0)
	r.controller.Forward(0)
}

func (r *RenderAnimatedSize) layoutStart() {
	size := foundation.Some(r.DebugAdoptSize(r.Child().Size()))
	r.sizeTween.Begin, r.sizeTween.End = size, size
	r.state = RenderAnimatedSizeStateStable
}

func (r *RenderAnimatedSize) layoutStable() {
	childSize := r.Child().Size()
	if r.sizeTween.End != foundation.Some(childSize) {
		r.sizeTween.Begin = foundation.Some(r.Size())
		r.sizeTween.End = foundation.Some(r.DebugAdoptSize(childSize))
		r.restartAnimation()
		r.state = RenderAnimatedSizeStateChanged
	} else if r.controller.Value() == r.controller.UpperBound {
		// The animation finished: reset the target sizes.
		size := foundation.Some(r.DebugAdoptSize(childSize))
		r.sizeTween.Begin, r.sizeTween.End = size, size
	} else if !r.controller.IsAnimating() {
		r.controller.Forward() // resume the animation after being detached
	}
}

func (r *RenderAnimatedSize) layoutChanged() {
	childSize := r.Child().Size()
	if r.sizeTween.End != foundation.Some(childSize) {
		// The child's size changed again: match it and restart the animation.
		size := foundation.Some(r.DebugAdoptSize(childSize))
		r.sizeTween.Begin, r.sizeTween.End = size, size
		r.restartAnimation()
		r.state = RenderAnimatedSizeStateUnstable
	} else {
		// The child's size stabilized.
		r.state = RenderAnimatedSizeStateStable
		if !r.controller.IsAnimating() {
			// Resume the animation after being detached.
			r.controller.Forward()
		}
	}
}

func (r *RenderAnimatedSize) layoutUnstable() {
	childSize := r.Child().Size()
	if r.sizeTween.End != foundation.Some(childSize) {
		// Still unstable: continue tracking the child.
		size := foundation.Some(r.DebugAdoptSize(childSize))
		r.sizeTween.Begin, r.sizeTween.End = size, size
		r.restartAnimation()
	} else {
		// The child's size stabilized.
		r.controller.Stop()
		r.state = RenderAnimatedSizeStateStable
	}
}

func (r *RenderAnimatedSize) animationStatusListener(status animation.AnimationStatus) {
	if status.IsCompleted() && r.onEnd != nil {
		r.onEnd()
	}
}

// Paint clips the child while it overflows the animated size.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.paint
func (r *RenderAnimatedSize) Paint(context *PaintingContext, offset ui.Offset) {
	if r.hasChild() && r.hasVisualOver && r.clipBehavior != ui.ClipNone {
		r.clipRectLayer.SetLayer(context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()),
			r.RenderAligningShiftedBox.Paint, PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: r.clipRectLayer.Layer()}))
	} else {
		r.clipRectLayer.SetLayer(nil)
		r.RenderAligningShiftedBox.Paint(context, offset)
	}
}

// ComputeDryBaseline is the child's baseline plus the offset of the aligned
// child in the animated size.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.computeDryBaseline
func (r *RenderAnimatedSize) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	child := r.Child()
	result, ok := child.GetDryBaseline(constraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	// The animated size, not the child's, decides the offset of alignChild.
	childSize := child.GetDryLayout(constraints)
	mySize := r.GetDryLayout(constraints)
	offset := r.ResolvedAlignment().AlongOffset(sizeDiff(mySize, childSize))
	return foundation.Some(result + offset.Dy)
}

// Dispose releases the controller, the animation and the clip layer.
//
// Flutter: rendering/animated_size.dart RenderAnimatedSize.dispose
func (r *RenderAnimatedSize) Dispose() {
	r.clipRectLayer.SetLayer(nil)
	r.controller.RemoveListener(r.controllerHdle)
	r.controller.Dispose()
	r.animation.Dispose()
	r.RenderAligningShiftedBox.Dispose()
}
