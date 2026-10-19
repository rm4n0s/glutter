package widgets

import (
	"reflect"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
)

// semanticsClipper is Flutter's _SemanticsClipper: it clips the semantics of
// its child by the insets of a notifier.
type semanticsClipper struct {
	SingleChildRenderObjectWidget
	clipDetailsNotifier *foundation.ValueNotifier[painting.EdgeInsets]
}

func (w *semanticsClipper) CreateRenderObject(BuildContext) rendering.RenderObject {
	return newRenderSemanticsClipper(w.clipDetailsNotifier)
}

func (w *semanticsClipper) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*renderSemanticsClipper).setClipDetailsNotifier(w.clipDetailsNotifier)
}

// renderSemanticsClipper is Flutter's _RenderSemanticsClipper.
type renderSemanticsClipper struct {
	rendering.RenderProxyBox
	clipDetailsNotifier *foundation.ValueNotifier[painting.EdgeInsets]
	handle              foundation.ListenerHandle
}

func newRenderSemanticsClipper(notifier *foundation.ValueNotifier[painting.EdgeInsets]) *renderSemanticsClipper {
	r := &renderSemanticsClipper{clipDetailsNotifier: notifier}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

func (r *renderSemanticsClipper) setClipDetailsNotifier(value *foundation.ValueNotifier[painting.EdgeInsets]) {
	if r.clipDetailsNotifier == value {
		return
	}
	if r.Attached() {
		r.clipDetailsNotifier.RemoveListener(r.handle)
	}
	r.clipDetailsNotifier = value
	r.handle = r.clipDetailsNotifier.AddListener(r.MarkNeedsSemanticsUpdate)
	r.MarkNeedsSemanticsUpdate()
}

func (r *renderSemanticsClipper) SemanticBounds() ui.Rect {
	clip := r.clipDetailsNotifier.Value()
	original := r.RenderProxyBox.SemanticBounds()
	return ui.NewRectFromLTRB(original.Left+clip.Left, original.Top+clip.Top, original.Right-clip.Right, original.Bottom-clip.Bottom)
}

func (r *renderSemanticsClipper) Attach(owner *rendering.PipelineOwner) {
	r.RenderProxyBox.Attach(owner)
	r.handle = r.clipDetailsNotifier.AddListener(r.MarkNeedsSemanticsUpdate)
}

func (r *renderSemanticsClipper) Detach() {
	r.clipDetailsNotifier.RemoveListener(r.handle)
	r.RenderProxyBox.Detach()
}

func (r *renderSemanticsClipper) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBox.DescribeSemanticsConfiguration(config)
	config.SetIsSemanticBoundary(true)
}

// ModalBarrierOpts are the optional parameters of NewModalBarrier.
//
// Flutter: widgets/modal_barrier.dart ModalBarrier.new
type ModalBarrierOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Color is the color to fill the barrier with; none for transparent.
	Color foundation.Opt[ui.Color]
	// Dismissible is whether tapping the barrier calls OnDismiss; the default is
	// true.
	Dismissible foundation.Opt[bool]
	// OnDismiss is called when the barrier is tapped and Dismissible is true; the
	// default pops the route of the Navigator.
	OnDismiss func()
	// SemanticsLabel is the semantic label of the barrier.
	SemanticsLabel string
	// BarrierSemanticsDismissible is whether the barrier is a semantic target
	// that assistive technology can dismiss; the default follows Dismissible
	// on the platforms that support it.
	BarrierSemanticsDismissible foundation.Opt[bool]
	// ClipDetailsNotifier notifies the insets that clip the semantics of the
	// barrier.
	ClipDetailsNotifier *foundation.ValueNotifier[painting.EdgeInsets]
	// SemanticsOnTapHint is the hint of the tap action of the barrier.
	SemanticsOnTapHint string
}

// ModalBarrier is a widget that prevents the user from interacting with widgets
// behind itself.
//
// Flutter: widgets/modal_barrier.dart ModalBarrier
type ModalBarrier struct {
	StatelessWidget
	o ModalBarrierOpts
}

// NewModalBarrier creates the widget.
//
// Flutter: widgets/modal_barrier.dart ModalBarrier.new
func NewModalBarrier(opts ...ModalBarrierOpts) *ModalBarrier {
	o := firstOpts(opts)
	return &ModalBarrier{NewStatelessWidgetBase(WidgetOpts{Key: o.Key}), o}
}

// Color is the color of the barrier, or none.
//
// Flutter: widgets/modal_barrier.dart ModalBarrier.color
func (w *ModalBarrier) Color() foundation.Opt[ui.Color] { return w.o.Color }

// Dismissible is whether tapping the barrier dismisses it.
//
// Flutter: widgets/modal_barrier.dart ModalBarrier.dismissible
func (w *ModalBarrier) Dismissible() bool { return w.o.Dismissible.OrElse(true) }

// OnDismiss is called when the barrier is tapped, or nil.
//
// Flutter: widgets/modal_barrier.dart ModalBarrier.onDismiss
func (w *ModalBarrier) OnDismiss() func() { return w.o.OnDismiss }

// SemanticsLabel is the semantic label of the barrier.
//
// Flutter: widgets/modal_barrier.dart ModalBarrier.semanticsLabel
func (w *ModalBarrier) SemanticsLabel() string { return w.o.SemanticsLabel }

// BarrierSemanticsDismissible is whether assistive technology can dismiss the
// barrier.
//
// Flutter: widgets/modal_barrier.dart ModalBarrier.barrierSemanticsDismissible
func (w *ModalBarrier) BarrierSemanticsDismissible() foundation.Opt[bool] {
	return w.o.BarrierSemanticsDismissible
}

// ClipDetailsNotifier notifies the insets that clip the semantics, or nil.
//
// Flutter: widgets/modal_barrier.dart ModalBarrier.clipDetailsNotifier
func (w *ModalBarrier) ClipDetailsNotifier() *foundation.ValueNotifier[painting.EdgeInsets] {
	return w.o.ClipDetailsNotifier
}

// SemanticsOnTapHint is the hint of the tap action of the barrier.
//
// Flutter: widgets/modal_barrier.dart ModalBarrier.semanticsOnTapHint
func (w *ModalBarrier) SemanticsOnTapHint() string { return w.o.SemanticsOnTapHint }

// navigatorMaybePop is Navigator.maybePop. TODO(step 10.2): pop the route of the
// Navigator.
func navigatorMaybePop(BuildContext) {}

// Build builds the barrier: a semantics node, an opaque area that is filled with
// the color, and a detector of taps of any button.
//
// Flutter: widgets/modal_barrier.dart ModalBarrier.build
func (w *ModalBarrier) Build(bc BuildContext) Widget {
	o := w.o
	dismissible := o.Dismissible.OrElse(true)
	if foundation.DebugMode && dismissible && o.SemanticsLabel != "" {
		DebugCheckHasDirectionality(bc)
	}
	var platformSupportsDismissingBarrier bool
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		platformSupportsDismissingBarrier = true
	}
	semanticsDismissible := dismissible && platformSupportsDismissingBarrier
	modalBarrierSemanticsDismissible := o.BarrierSemanticsDismissible.OrElse(semanticsDismissible)

	handleDismiss := func() {
		if dismissible {
			if o.OnDismiss != nil {
				o.OnDismiss()
			} else {
				navigatorMaybePop(bc)
			}
		}
		// TODO(no step yet): SystemSound.play(SystemSoundType.alert) when not
		// dismissible.
	}

	semanticsOpts := SemanticsOpts{}
	if o.SemanticsOnTapHint != "" {
		semanticsOpts.HintOverrides = &semantics.SemanticsHintOverrides{OnTapHint: o.SemanticsOnTapHint}
	}
	if semanticsDismissible && o.SemanticsLabel != "" {
		semanticsOpts.OnTap = handleDismiss
		semanticsOpts.OnDismiss = handleDismiss
		semanticsOpts.TextDirection = DirectionalityOf(bc)
	}
	if semanticsDismissible {
		semanticsOpts.Label = foundation.Some(o.SemanticsLabel)
	}
	var fill Widget
	if c, ok := o.Color.Get(); ok {
		fill = NewColoredBox(c, nil)
	}
	// TODO(step 10.4): MouseRegion(cursor: SystemMouseCursors.basic).
	var barrier Widget = NewSemantics(
		NewMouseRegion(NewConstrainedBox(rendering.NewBoxConstraintsExpand(), fill)),
		semanticsOpts)

	// Developers can set dismissible and barrierSemanticsDismissible to let
	// assistive technology users dismiss a modal bottom sheet by tapping the
	// scrim. On iOS, some modal barriers are not dismissible in accessibility
	// mode.
	excluding := !semanticsDismissible || !modalBarrierSemanticsDismissible
	if !excluding && o.ClipDetailsNotifier != nil {
		barrier = &semanticsClipper{NewSingleChildRenderObjectWidgetBase(barrier), o.ClipDetailsNotifier}
	}
	return NewBlockSemantics(NewExcludeSemantics(
		&modalBarrierGestureDetector{NewStatelessWidgetBase(), barrier, handleDismiss},
		ExcludeSemanticsOpts{Excluding: foundation.Some(excluding)}))
}

// AnimatedModalBarrierOpts are the optional parameters of NewAnimatedModalBarrier.
//
// Flutter: widgets/modal_barrier.dart AnimatedModalBarrier.new
type AnimatedModalBarrierOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Dismissible is whether tapping the barrier calls OnDismiss; the default is
	// true.
	Dismissible foundation.Opt[bool]
	// SemanticsLabel is the semantic label of the barrier.
	SemanticsLabel string
	// BarrierSemanticsDismissible is whether assistive technology can dismiss the
	// barrier.
	BarrierSemanticsDismissible foundation.Opt[bool]
	// OnDismiss is called when the barrier is tapped and Dismissible is true.
	OnDismiss func()
	// ClipDetailsNotifier notifies the insets that clip the semantics.
	ClipDetailsNotifier *foundation.ValueNotifier[painting.EdgeInsets]
	// SemanticsOnTapHint is the hint of the tap action of the barrier.
	SemanticsOnTapHint string
}

// AnimatedModalBarrier is a widget that prevents the user from interacting with
// widgets behind itself, and can be configured with an animated color value.
//
// Flutter: widgets/modal_barrier.dart AnimatedModalBarrier
type AnimatedModalBarrier struct {
	AnimatedWidget
	o AnimatedModalBarrierOpts
}

// NewAnimatedModalBarrier creates the widget; the animation is of the color,
// none for transparent.
//
// Flutter: widgets/modal_barrier.dart AnimatedModalBarrier.new
func NewAnimatedModalBarrier(color animation.Animation[foundation.Opt[ui.Color]], opts ...AnimatedModalBarrierOpts) *AnimatedModalBarrier {
	o := firstOpts(opts)
	return &AnimatedModalBarrier{NewAnimatedWidgetBase(color, WidgetOpts{Key: o.Key}), o}
}

// Color is the animation of the color of the barrier.
//
// Flutter: widgets/modal_barrier.dart AnimatedModalBarrier.color
func (w *AnimatedModalBarrier) Color() animation.Animation[foundation.Opt[ui.Color]] {
	return w.listenable.(animation.Animation[foundation.Opt[ui.Color]])
}

// Build builds a ModalBarrier with the current color.
//
// Flutter: widgets/modal_barrier.dart AnimatedModalBarrier.build
func (w *AnimatedModalBarrier) Build(BuildContext) Widget {
	return NewModalBarrier(ModalBarrierOpts{
		Color: w.Color().Value(), Dismissible: w.o.Dismissible, SemanticsLabel: w.o.SemanticsLabel,
		BarrierSemanticsDismissible: w.o.BarrierSemanticsDismissible, OnDismiss: w.o.OnDismiss,
		ClipDetailsNotifier: w.o.ClipDetailsNotifier, SemanticsOnTapHint: w.o.SemanticsOnTapHint})
}

// anyTapGestureRecognizer is Flutter's _AnyTapGestureRecognizer: it recognizes
// the tap up of any single button.
type anyTapGestureRecognizer struct {
	gestures.BaseTapGestureRecognizer
	onAnyTapUp func()
}

func newAnyTapGestureRecognizer(binding *gestures.GestureBinding) *anyTapGestureRecognizer {
	r := &anyTapGestureRecognizer{}
	r.InitBaseTap(binding, r, gestures.BaseTapOpts{})
	return r
}

func (r *anyTapGestureRecognizer) IsPointerAllowed(event *gestures.PointerDownEvent) bool {
	if r.onAnyTapUp == nil {
		return false
	}
	return r.BaseTapGestureRecognizer.IsPointerAllowed(event)
}

func (r *anyTapGestureRecognizer) HandleTapDown(*gestures.PointerDownEvent) {}

func (r *anyTapGestureRecognizer) HandleTapUp(_ *gestures.PointerDownEvent, _ *gestures.PointerUpEvent) {
	if r.onAnyTapUp != nil {
		r.InvokeCallback("onAnyTapUp", r.onAnyTapUp)
	}
}

func (r *anyTapGestureRecognizer) HandleTapCancel(*gestures.PointerDownEvent, *gestures.PointerCancelEvent, string) {
}

func (r *anyTapGestureRecognizer) DebugDescription() string { return "any tap" }

// modalBarrierGestureDetector is Flutter's _ModalBarrierGestureDetector: a
// detector with one callback, which recognizes the tap of any button.
type modalBarrierGestureDetector struct {
	StatelessWidget
	child     Widget
	onDismiss func()
}

func (w *modalBarrierGestureDetector) Build(bc BuildContext) Widget {
	binding := WidgetsBindingOf(bc).Gestures()
	factory := NewGestureRecognizerFactoryWithHandlers(
		func() *anyTapGestureRecognizer { return newAnyTapGestureRecognizer(binding) },
		func(r *anyTapGestureRecognizer) { r.onAnyTapUp = w.onDismiss })
	return NewRawGestureDetector(w.child, RawGestureDetectorOpts{
		Gestures: map[reflect.Type]GestureRecognizerFactory{factory.RecognizerType(): factory},
		Behavior: rendering.HitTestBehaviorOpaque,
	})
}
