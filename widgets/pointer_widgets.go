package widgets

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
)

// ListenerOpts are the optional parameters of NewListener.
//
// Flutter: widgets/basic.dart Listener.new
type ListenerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnPointerDown is called when a pointer comes into contact with the screen
	// at this widget.
	OnPointerDown rendering.PointerDownEventListener
	// OnPointerMove is called when a pointer that is in contact with the screen
	// and started at this widget moves.
	OnPointerMove rendering.PointerMoveEventListener
	// OnPointerUp is called when a pointer that started at this widget stops
	// being in contact with the screen.
	OnPointerUp rendering.PointerUpEventListener
	// OnPointerHover is called when a pointer that has not triggered a pointer
	// down changes.
	OnPointerHover services.PointerHoverEventListener
	// OnPointerCancel is called when the input from a pointer is no longer
	// directed towards this widget.
	OnPointerCancel rendering.PointerCancelEventListener
	// OnPointerPanZoomStart is called when a pan/zoom begins.
	OnPointerPanZoomStart rendering.PointerPanZoomStartEventListener
	// OnPointerPanZoomUpdate is called when a pan/zoom is updated.
	OnPointerPanZoomUpdate rendering.PointerPanZoomUpdateEventListener
	// OnPointerPanZoomEnd is called when a pan/zoom ends.
	OnPointerPanZoomEnd rendering.PointerPanZoomEndEventListener
	// OnPointerSignal is called when a pointer signal occurs over this widget.
	OnPointerSignal rendering.PointerSignalEventListener
	// Behavior is how to behave during hit testing; the default is
	// HitTestBehaviorDeferToChild.
	Behavior rendering.HitTestBehavior
}

// Listener calls callbacks in response to common pointer events. It listens to
// the events that reach it; it does not take part in the gesture arena.
//
// Flutter: widgets/basic.dart Listener
type Listener struct {
	SingleChildRenderObjectWidget
	opts ListenerOpts
}

// NewListener creates a widget that calls the callbacks for the pointer events
// that hit its child.
//
// Flutter: widgets/basic.dart Listener.new
func NewListener(child Widget, opts ...ListenerOpts) *Listener {
	o := firstOpts(opts)
	if o.Behavior == rendering.HitTestBehaviorUnspecified {
		o.Behavior = rendering.HitTestBehaviorDeferToChild
	}
	return &Listener{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), o}
}

// OnPointerDown is called when a pointer comes into contact with the screen.
//
// Flutter: widgets/basic.dart Listener.onPointerDown
func (w *Listener) OnPointerDown() rendering.PointerDownEventListener { return w.opts.OnPointerDown }

// OnPointerMove is called when a pointer in contact with the screen moves.
//
// Flutter: widgets/basic.dart Listener.onPointerMove
func (w *Listener) OnPointerMove() rendering.PointerMoveEventListener { return w.opts.OnPointerMove }

// OnPointerUp is called when a pointer stops being in contact with the screen.
//
// Flutter: widgets/basic.dart Listener.onPointerUp
func (w *Listener) OnPointerUp() rendering.PointerUpEventListener { return w.opts.OnPointerUp }

// OnPointerHover is called when a pointer that is not in contact changes.
//
// Flutter: widgets/basic.dart Listener.onPointerHover
func (w *Listener) OnPointerHover() services.PointerHoverEventListener { return w.opts.OnPointerHover }

// OnPointerCancel is called when the input from a pointer is cancelled.
//
// Flutter: widgets/basic.dart Listener.onPointerCancel
func (w *Listener) OnPointerCancel() rendering.PointerCancelEventListener {
	return w.opts.OnPointerCancel
}

// OnPointerSignal is called when a pointer signal occurs over the widget.
//
// Flutter: widgets/basic.dart Listener.onPointerSignal
func (w *Listener) OnPointerSignal() rendering.PointerSignalEventListener {
	return w.opts.OnPointerSignal
}

// Behavior is how the widget behaves during hit testing.
//
// Flutter: widgets/basic.dart Listener.behavior
func (w *Listener) Behavior() rendering.HitTestBehavior { return w.opts.Behavior }

func (w *Listener) configure(r *rendering.RenderPointerListener) {
	r.OnPointerDown = w.opts.OnPointerDown
	r.OnPointerMove = w.opts.OnPointerMove
	r.OnPointerUp = w.opts.OnPointerUp
	r.OnPointerHover = w.opts.OnPointerHover
	r.OnPointerCancel = w.opts.OnPointerCancel
	r.OnPointerPanZoomStart = w.opts.OnPointerPanZoomStart
	r.OnPointerPanZoomUpdate = w.opts.OnPointerPanZoomUpdate
	r.OnPointerPanZoomEnd = w.opts.OnPointerPanZoomEnd
	r.OnPointerSignal = w.opts.OnPointerSignal
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Listener.createRenderObject
func (w *Listener) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := rendering.NewRenderPointerListener(nil, rendering.RenderPointerListenerOpts{Behavior: w.opts.Behavior})
	w.configure(r)
	return r
}

// UpdateRenderObject copies the callbacks and the behavior to the render
// object.
//
// Flutter: widgets/basic.dart Listener.updateRenderObject
func (w *Listener) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderPointerListener)
	w.configure(r)
	r.Behavior = w.opts.Behavior
}

// MouseRegionOpts are the optional parameters of NewMouseRegion.
//
// Flutter: widgets/basic.dart MouseRegion.new
type MouseRegionOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnEnter is called when a mouse pointer enters the region.
	OnEnter services.PointerEnterEventListener
	// OnExit is called when a mouse pointer leaves the region.
	OnExit services.PointerExitEventListener
	// OnHover is called when a mouse pointer moves inside the region.
	OnHover services.PointerHoverEventListener
	// Cursor is the mouse cursor while a mouse pointer is over the region; the
	// default is MouseCursorDefer.
	Cursor services.MouseCursor
	// Opaque is whether this region hides the regions behind it from mouse
	// pointers; the default is true.
	Opaque foundation.Opt[bool]
	// HitTestBehavior is how to behave during hit testing; the default is
	// HitTestBehaviorDeferToChild.
	HitTestBehavior rendering.HitTestBehavior
}

// MouseRegion tracks mouse pointers entering, hovering over and leaving its
// child, and sets the cursor over it.
//
// Flutter: widgets/basic.dart MouseRegion
type MouseRegion struct {
	SingleChildRenderObjectWidget
	opts MouseRegionOpts
}

// NewMouseRegion creates a widget that tracks mouse pointers over its child.
//
// Flutter: widgets/basic.dart MouseRegion.new
func NewMouseRegion(child Widget, opts ...MouseRegionOpts) *MouseRegion {
	o := firstOpts(opts)
	if o.Cursor == nil {
		o.Cursor = services.MouseCursorDefer
	}
	if o.HitTestBehavior == rendering.HitTestBehaviorUnspecified {
		o.HitTestBehavior = rendering.HitTestBehaviorDeferToChild
	}
	return &MouseRegion{NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), o}
}

// OnEnter is called when a mouse pointer enters the region.
//
// Flutter: widgets/basic.dart MouseRegion.onEnter
func (w *MouseRegion) OnEnter() services.PointerEnterEventListener { return w.opts.OnEnter }

// OnExit is called when a mouse pointer leaves the region.
//
// Flutter: widgets/basic.dart MouseRegion.onExit
func (w *MouseRegion) OnExit() services.PointerExitEventListener { return w.opts.OnExit }

// OnHover is called when a mouse pointer moves inside the region.
//
// Flutter: widgets/basic.dart MouseRegion.onHover
func (w *MouseRegion) OnHover() services.PointerHoverEventListener { return w.opts.OnHover }

// Cursor is the mouse cursor over the region.
//
// Flutter: widgets/basic.dart MouseRegion.cursor
func (w *MouseRegion) Cursor() services.MouseCursor { return w.opts.Cursor }

// Opaque is whether the region hides the regions behind it.
//
// Flutter: widgets/basic.dart MouseRegion.opaque
func (w *MouseRegion) Opaque() bool { return w.opts.Opaque.OrElse(true) }

// HitTestBehavior is how the region behaves during hit testing.
//
// Flutter: widgets/basic.dart MouseRegion.hitTestBehavior
func (w *MouseRegion) HitTestBehavior() rendering.HitTestBehavior { return w.opts.HitTestBehavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart MouseRegion.createRenderObject
func (w *MouseRegion) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderMouseRegion(nil, rendering.RenderMouseRegionOpts{OnEnter: w.opts.OnEnter, OnHover: w.opts.OnHover,
		OnExit: w.opts.OnExit, Cursor: w.opts.Cursor, Opaque: w.opts.Opaque, HitTestBehavior: w.opts.HitTestBehavior})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart MouseRegion.updateRenderObject
func (w *MouseRegion) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderMouseRegion)
	r.SetOnEnter(w.opts.OnEnter)
	r.SetOnHover(w.opts.OnHover)
	r.SetOnExit(w.opts.OnExit)
	r.SetCursor(w.opts.Cursor)
	r.SetOpaque(w.opts.Opaque.OrElse(true))
	r.SetHitTestBehavior(w.opts.HitTestBehavior)
}
