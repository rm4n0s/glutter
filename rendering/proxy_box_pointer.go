package rendering

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// PointerDownEventListener is called when a pointer comes into contact with the
// screen.
//
// Flutter: rendering/proxy_box.dart PointerDownEventListener
type PointerDownEventListener func(event *gestures.PointerDownEvent)

// PointerMoveEventListener is called when a pointer that is in contact with the
// screen moves.
//
// Flutter: rendering/proxy_box.dart PointerMoveEventListener
type PointerMoveEventListener func(event *gestures.PointerMoveEvent)

// PointerUpEventListener is called when a pointer stops being in contact with
// the screen.
//
// Flutter: rendering/proxy_box.dart PointerUpEventListener
type PointerUpEventListener func(event *gestures.PointerUpEvent)

// PointerCancelEventListener is called when the input from a pointer is no
// longer directed towards this receiver.
//
// Flutter: rendering/proxy_box.dart PointerCancelEventListener
type PointerCancelEventListener func(event *gestures.PointerCancelEvent)

// PointerPanZoomStartEventListener is called when a pan/zoom begins.
//
// Flutter: rendering/proxy_box.dart PointerPanZoomStartEventListener
type PointerPanZoomStartEventListener func(event *gestures.PointerPanZoomStartEvent)

// PointerPanZoomUpdateEventListener is called when a pan/zoom is updated.
//
// Flutter: rendering/proxy_box.dart PointerPanZoomUpdateEventListener
type PointerPanZoomUpdateEventListener func(event *gestures.PointerPanZoomUpdateEvent)

// PointerPanZoomEndEventListener is called when a pan/zoom ends.
//
// Flutter: rendering/proxy_box.dart PointerPanZoomEndEventListener
type PointerPanZoomEndEventListener func(event *gestures.PointerPanZoomEndEvent)

// PointerSignalEventListener is called when a pointer signal occurs, such as a
// scroll.
//
// Flutter: rendering/proxy_box.dart PointerSignalEventListener
type PointerSignalEventListener func(event gestures.PointerSignalEvent)

// RenderPointerListener calls callbacks in response to common pointer events.
//
// Flutter: rendering/proxy_box.dart RenderPointerListener
type RenderPointerListener struct {
	RenderProxyBoxWithHitTestBehavior
	// OnPointerDown is called when a pointer comes into contact with the screen
	// at this box.
	//
	// Flutter: rendering/proxy_box.dart RenderPointerListener.onPointerDown
	OnPointerDown PointerDownEventListener
	// OnPointerMove is called when a pointer that is in contact with the screen
	// and started at this box moves.
	//
	// Flutter: rendering/proxy_box.dart RenderPointerListener.onPointerMove
	OnPointerMove PointerMoveEventListener
	// OnPointerUp is called when a pointer that started at this box stops being
	// in contact with the screen.
	//
	// Flutter: rendering/proxy_box.dart RenderPointerListener.onPointerUp
	OnPointerUp PointerUpEventListener
	// OnPointerHover is called when a pointer that has not triggered an
	// on-pointer-down event changes.
	//
	// Flutter: rendering/proxy_box.dart RenderPointerListener.onPointerHover
	OnPointerHover services.PointerHoverEventListener
	// OnPointerCancel is called when the input from a pointer is no longer
	// directed towards this box.
	//
	// Flutter: rendering/proxy_box.dart RenderPointerListener.onPointerCancel
	OnPointerCancel PointerCancelEventListener
	// OnPointerPanZoomStart is called when a pan/zoom begins.
	//
	// Flutter: rendering/proxy_box.dart RenderPointerListener.onPointerPanZoomStart
	OnPointerPanZoomStart PointerPanZoomStartEventListener
	// OnPointerPanZoomUpdate is called when a pan/zoom is updated.
	//
	// Flutter: rendering/proxy_box.dart RenderPointerListener.onPointerPanZoomUpdate
	OnPointerPanZoomUpdate PointerPanZoomUpdateEventListener
	// OnPointerPanZoomEnd is called when a pan/zoom ends.
	//
	// Flutter: rendering/proxy_box.dart RenderPointerListener.onPointerPanZoomEnd
	OnPointerPanZoomEnd PointerPanZoomEndEventListener
	// OnPointerSignal is called when a pointer signal occurs over this box.
	//
	// Flutter: rendering/proxy_box.dart RenderPointerListener.onPointerSignal
	OnPointerSignal PointerSignalEventListener
}

// RenderPointerListenerOpts are the optional parameters of
// NewRenderPointerListener. The listeners are fields of the render object, and
// can be changed at any time.
//
// Flutter: rendering/proxy_box.dart RenderPointerListener.new
type RenderPointerListenerOpts struct {
	// Behavior defaults to HitTestBehaviorDeferToChild.
	Behavior HitTestBehavior
}

// NewRenderPointerListener creates the box.
//
// Flutter: rendering/proxy_box.dart RenderPointerListener.new
func NewRenderPointerListener(child Box, opts ...RenderPointerListenerOpts) *RenderPointerListener {
	r := &RenderPointerListener{}
	InitRenderProxyBoxWithHitTestBehavior(&r.RenderProxyBoxWithHitTestBehavior, r, firstOpts(opts).Behavior, child)
	return r
}

// ComputeSizeForNoChild is the biggest size: a listener without a child fills
// the space.
//
// Flutter: rendering/proxy_box.dart RenderPointerListener.computeSizeForNoChild
func (r *RenderPointerListener) ComputeSizeForNoChild(constraints BoxConstraints) ui.Size {
	return constraints.Biggest()
}

// HandleEvent calls the listener for the event.
//
// Flutter: rendering/proxy_box.dart RenderPointerListener.handleEvent
func (r *RenderPointerListener) HandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) {
	r.DebugHandleEvent(event, entry)
	switch e := event.(type) {
	case *gestures.PointerDownEvent:
		if r.OnPointerDown != nil {
			r.OnPointerDown(e)
		}
	case *gestures.PointerMoveEvent:
		if r.OnPointerMove != nil {
			r.OnPointerMove(e)
		}
	case *gestures.PointerUpEvent:
		if r.OnPointerUp != nil {
			r.OnPointerUp(e)
		}
	case *gestures.PointerHoverEvent:
		if r.OnPointerHover != nil {
			r.OnPointerHover(e)
		}
	case *gestures.PointerCancelEvent:
		if r.OnPointerCancel != nil {
			r.OnPointerCancel(e)
		}
	case *gestures.PointerPanZoomStartEvent:
		if r.OnPointerPanZoomStart != nil {
			r.OnPointerPanZoomStart(e)
		}
	case *gestures.PointerPanZoomUpdateEvent:
		if r.OnPointerPanZoomUpdate != nil {
			r.OnPointerPanZoomUpdate(e)
		}
	case *gestures.PointerPanZoomEndEvent:
		if r.OnPointerPanZoomEnd != nil {
			r.OnPointerPanZoomEnd(e)
		}
	case gestures.PointerSignalEvent:
		if r.OnPointerSignal != nil {
			r.OnPointerSignal(e)
		}
	}
}

// DebugFillProperties adds the listeners.
//
// Flutter: rendering/proxy_box.dart RenderPointerListener.debugFillProperties
func (r *RenderPointerListener) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBoxWithHitTestBehavior.DebugFillProperties(b)
	b.Add(foundation.NewFlagsSummary("listeners", []foundation.FlagEntry{
		{Description: "down", Present: r.OnPointerDown != nil},
		{Description: "move", Present: r.OnPointerMove != nil},
		{Description: "up", Present: r.OnPointerUp != nil},
		{Description: "hover", Present: r.OnPointerHover != nil},
		{Description: "cancel", Present: r.OnPointerCancel != nil},
		{Description: "panZoomStart", Present: r.OnPointerPanZoomStart != nil},
		{Description: "panZoomUpdate", Present: r.OnPointerPanZoomUpdate != nil},
		{Description: "panZoomEnd", Present: r.OnPointerPanZoomEnd != nil},
		{Description: "signal", Present: r.OnPointerSignal != nil},
	}, foundation.FlagsSummaryOpts{IfEmpty: "<none>"}))
}

// RenderMouseRegion tracks mouse pointers entering, hovering over and leaving
// the region. It is a services.MouseTrackerAnnotation.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion
type RenderMouseRegion struct {
	RenderProxyBoxWithHitTestBehavior
	onEnter              services.PointerEnterEventListener
	onHover              services.PointerHoverEventListener
	onExit               services.PointerExitEventListener
	cursor               services.MouseCursor
	validForMouseTracker bool
	opaque               bool
}

// RenderMouseRegionOpts are the optional parameters of NewRenderMouseRegion.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.new
type RenderMouseRegionOpts struct {
	// OnEnter is called when a mouse pointer enters the region.
	OnEnter services.PointerEnterEventListener
	// OnHover is called when a mouse pointer moves over the region.
	OnHover services.PointerHoverEventListener
	// OnExit is called when a mouse pointer leaves the region.
	OnExit services.PointerExitEventListener
	// Cursor defaults to services.MouseCursorDefer.
	Cursor services.MouseCursor
	// ValidForMouseTracker defaults to true.
	ValidForMouseTracker foundation.Opt[bool]
	// Opaque defaults to true.
	Opaque foundation.Opt[bool]
	// HitTestBehavior defaults to HitTestBehaviorOpaque.
	HitTestBehavior HitTestBehavior
}

// NewRenderMouseRegion creates the box.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.new
func NewRenderMouseRegion(child Box, opts ...RenderMouseRegionOpts) *RenderMouseRegion {
	o := firstOpts(opts)
	if o.Cursor == nil {
		o.Cursor = services.MouseCursorDefer
	}
	if o.HitTestBehavior == HitTestBehaviorUnspecified {
		o.HitTestBehavior = HitTestBehaviorOpaque
	}
	r := &RenderMouseRegion{onEnter: o.OnEnter, onHover: o.OnHover, onExit: o.OnExit, cursor: o.Cursor,
		validForMouseTracker: o.ValidForMouseTracker.OrElse(true), opaque: o.Opaque.OrElse(true)}
	InitRenderProxyBoxWithHitTestBehavior(&r.RenderProxyBoxWithHitTestBehavior, r, o.HitTestBehavior, child)
	return r
}

// HitTest also requires the region to be opaque.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.hitTest
func (r *RenderMouseRegion) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	return r.RenderProxyBoxWithHitTestBehavior.HitTest(result, position) && r.opaque
}

// HandleEvent calls the hover listener.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.handleEvent
func (r *RenderMouseRegion) HandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) {
	r.DebugHandleEvent(event, entry)
	if e, ok := event.(*gestures.PointerHoverEvent); ok && r.onHover != nil {
		r.onHover(e)
	}
}

// Opaque is whether this region prevents other mouse regions behind it from
// detecting the mouse.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.opaque
func (r *RenderMouseRegion) Opaque() bool { return r.opaque }

// SetOpaque changes the flag and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.opaque
func (r *RenderMouseRegion) SetOpaque(value bool) {
	if r.opaque != value {
		r.opaque = value
		r.MarkNeedsPaint()
	}
}

// HitTestBehavior is the hit test behavior of the region.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.hitTestBehavior
func (r *RenderMouseRegion) HitTestBehavior() HitTestBehavior { return r.Behavior }

// SetHitTestBehavior changes the behavior (the zero value is opaque) and marks
// the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.hitTestBehavior
func (r *RenderMouseRegion) SetHitTestBehavior(value HitTestBehavior) {
	if value == HitTestBehaviorUnspecified {
		value = HitTestBehaviorOpaque
	}
	if r.Behavior != value {
		r.Behavior = value
		r.MarkNeedsPaint()
	}
}

// OnEnter is called when a mouse pointer enters the region.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.onEnter
func (r *RenderMouseRegion) OnEnter() services.PointerEnterEventListener { return r.onEnter }

// SetOnEnter changes the enter listener.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.onEnter
func (r *RenderMouseRegion) SetOnEnter(v services.PointerEnterEventListener) { r.onEnter = v }

// OnHover is called when a mouse pointer moves over the region.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.onHover
func (r *RenderMouseRegion) OnHover() services.PointerHoverEventListener { return r.onHover }

// SetOnHover changes the hover listener.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.onHover
func (r *RenderMouseRegion) SetOnHover(v services.PointerHoverEventListener) { r.onHover = v }

// OnExit is called when a mouse pointer leaves the region.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.onExit
func (r *RenderMouseRegion) OnExit() services.PointerExitEventListener { return r.onExit }

// SetOnExit changes the exit listener.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.onExit
func (r *RenderMouseRegion) SetOnExit(v services.PointerExitEventListener) { r.onExit = v }

// Cursor is the mouse cursor over the region.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.cursor
func (r *RenderMouseRegion) Cursor() services.MouseCursor { return r.cursor }

// SetCursor changes the cursor and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.cursor
func (r *RenderMouseRegion) SetCursor(value services.MouseCursor) {
	if r.cursor != value {
		r.cursor = value
		r.MarkNeedsPaint()
	}
}

// ValidForMouseTracker is false for a detached region.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.validForMouseTracker
func (r *RenderMouseRegion) ValidForMouseTracker() bool { return r.validForMouseTracker }

// Attach makes the region valid for the mouse tracker.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.attach
func (r *RenderMouseRegion) Attach(owner *PipelineOwner) {
	r.RenderProxyBoxWithHitTestBehavior.Attach(owner)
	r.validForMouseTracker = true
}

// Detach makes the region invalid for the mouse tracker.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.detach
func (r *RenderMouseRegion) Detach() {
	r.validForMouseTracker = false
	r.RenderProxyBoxWithHitTestBehavior.Detach()
}

// ComputeSizeForNoChild is the biggest size.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.computeSizeForNoChild
func (r *RenderMouseRegion) ComputeSizeForNoChild(constraints BoxConstraints) ui.Size {
	return constraints.Biggest()
}

// DebugFillProperties adds the listeners, the cursor and the flags.
//
// Flutter: rendering/proxy_box.dart RenderMouseRegion.debugFillProperties
func (r *RenderMouseRegion) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBoxWithHitTestBehavior.DebugFillProperties(b)
	b.Add(foundation.NewFlagsSummary("listeners", []foundation.FlagEntry{
		{Description: "enter", Present: r.onEnter != nil},
		{Description: "hover", Present: r.onHover != nil},
		{Description: "exit", Present: r.onExit != nil},
	}, foundation.FlagsSummaryOpts{IfEmpty: "<none>"}))
	b.Add(foundation.NewDiagnosticsProperty("cursor", r.cursor, foundation.DiagnosticsPropertyOpts{DefaultValue: services.MouseCursorDefer}))
	b.Add(foundation.NewDiagnosticsProperty("opaque", r.opaque, foundation.DiagnosticsPropertyOpts{DefaultValue: true}))
	b.Add(foundation.NewFlagProperty("validForMouseTracker", foundation.Some(r.validForMouseTracker),
		foundation.FlagPropertyOpts{DefaultValue: true, IfTrue: "", IfFalse: "invalid for MouseTracker"}))
}

// RenderRepaintBoundary creates a separate display list for its child.
//
// Flutter: rendering/proxy_box.dart RenderRepaintBoundary
type RenderRepaintBoundary struct {
	RenderProxyBox
	symmetricPaintCount  int
	asymmetricPaintCount int
}

// NewRenderRepaintBoundary creates the box.
//
// Flutter: rendering/proxy_box.dart RenderRepaintBoundary.new
func NewRenderRepaintBoundary(child Box) *RenderRepaintBoundary {
	r := &RenderRepaintBoundary{}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// IsRepaintBoundary is true.
//
// Flutter: rendering/proxy_box.dart RenderRepaintBoundary.isRepaintBoundary
func (r *RenderRepaintBoundary) IsRepaintBoundary() bool { return true }

// ToImage creates an image from the contents of this render object and its
// children. The box must have been painted.
//
// Flutter: rendering/proxy_box.dart RenderRepaintBoundary.toImage
func (r *RenderRepaintBoundary) ToImage(pixelRatio float64, opts ...ui.ToImageOpts) (*ui.Image, error) {
	return r.offsetLayer().ToImage(ui.OffsetZero.And(r.Size()), pixelRatio, opts...)
}

// ToImageSync is like ToImage but does not wait for the rasterization.
//
// Flutter: rendering/proxy_box.dart RenderRepaintBoundary.toImageSync
func (r *RenderRepaintBoundary) ToImageSync(pixelRatio float64, opts ...ui.ToImageOpts) (*ui.Image, error) {
	return r.offsetLayer().ToImageSync(ui.OffsetZero.And(r.Size()), pixelRatio, opts...)
}

func (r *RenderRepaintBoundary) offsetLayer() *OffsetLayer {
	if foundation.DebugMode && r.DebugNeedsPaint() {
		panic(errors.New("RepaintBoundaryNotPainted", "ToImage was called on a repaint boundary that needs paint",
			"object", foundation.DescribeIdentity(r.self)))
	}
	return r.Layer().(OffsetLayerLike).AsOffsetLayer()
}

// DebugSymmetricPaintCount is the number of times that this repaint boundary
// repainted at the same time as its parent (debug builds only).
//
// Flutter: rendering/proxy_box.dart RenderRepaintBoundary.debugSymmetricPaintCount
func (r *RenderRepaintBoundary) DebugSymmetricPaintCount() int { return r.symmetricPaintCount }

// DebugAsymmetricPaintCount is the number of times that this repaint boundary
// repainted without its parent (debug builds only).
//
// Flutter: rendering/proxy_box.dart RenderRepaintBoundary.debugAsymmetricPaintCount
func (r *RenderRepaintBoundary) DebugAsymmetricPaintCount() int { return r.asymmetricPaintCount }

// DebugResetMetrics clears the paint counts.
//
// Flutter: rendering/proxy_box.dart RenderRepaintBoundary.debugResetMetrics
func (r *RenderRepaintBoundary) DebugResetMetrics() {
	if foundation.DebugMode {
		r.symmetricPaintCount, r.asymmetricPaintCount = 0, 0
	}
}

// DebugRegisterRepaintBoundaryPaint counts a paint of the boundary.
//
// Flutter: rendering/proxy_box.dart RenderRepaintBoundary.debugRegisterRepaintBoundaryPaint
func (r *RenderRepaintBoundary) DebugRegisterRepaintBoundaryPaint(includedParent, includedChild bool) {
	if !foundation.DebugMode {
		return
	}
	if includedParent && includedChild {
		r.symmetricPaintCount++
	} else {
		r.asymmetricPaintCount++
	}
}

// DebugFillProperties adds the usefulness of the boundary.
//
// Flutter: rendering/proxy_box.dart RenderRepaintBoundary.debugFillProperties
func (r *RenderRepaintBoundary) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	if !foundation.DebugMode {
		b.Add(foundation.NewDiagnosticsNodeMessage("(run in debug mode to collect repaint boundary statistics)"))
		return
	}
	total := r.symmetricPaintCount + r.asymmetricPaintCount
	if total == 0 {
		b.Add(foundation.NewMessageProperty("usefulness ratio", "no metrics collected yet (never painted)"))
		return
	}
	fraction := float64(r.asymmetricPaintCount) / float64(total)
	var diagnosis string
	switch {
	case total < 5:
		diagnosis = "insufficient data to draw conclusion (less than five repaints)"
	case fraction > 0.9:
		diagnosis = "this is an outstandingly useful repaint boundary and should definitely be kept"
	case fraction > 0.5:
		diagnosis = "this is a useful repaint boundary and should be kept"
	case fraction > 0.3:
		diagnosis = "this repaint boundary is probably useful, but maybe it would be more useful in tandem with adding more repaint boundaries elsewhere"
	case fraction > 0.1:
		diagnosis = "this repaint boundary does sometimes show value, though currently not that often"
	case r.asymmetricPaintCount > 0:
		diagnosis = "this repaint boundary is not very effective and should probably be removed"
	default:
		diagnosis = "this repaint boundary is astoundingly ineffectual and should be removed"
	}
	b.Add(foundation.NewPercentProperty("metrics", foundation.Some(fraction), foundation.NumberPropertyOpts{
		Unit: "useful", Tooltip: itoa(r.symmetricPaintCount) + " bad vs " + itoa(r.asymmetricPaintCount) + " good"}))
	b.Add(foundation.NewMessageProperty("diagnosis", diagnosis))
}

// RenderIgnorePointer is invisible to hit testing when ignoring.
//
// Deviation: the deprecated ignoringSemantics parameter is not ported, so it is
// always null: the pointer-ignoring subtree blocks user actions and keeps its
// semantics (SPEC §12.10).
//
// Flutter: rendering/proxy_box.dart RenderIgnorePointer
type RenderIgnorePointer struct {
	RenderProxyBox
	ignoring bool
}

// RenderIgnorePointerOpts are the optional parameters of NewRenderIgnorePointer.
//
// Flutter: rendering/proxy_box.dart RenderIgnorePointer.new
type RenderIgnorePointerOpts struct {
	// Ignoring defaults to true.
	Ignoring foundation.Opt[bool]
}

// NewRenderIgnorePointer creates the box.
//
// Flutter: rendering/proxy_box.dart RenderIgnorePointer.new
func NewRenderIgnorePointer(child Box, opts ...RenderIgnorePointerOpts) *RenderIgnorePointer {
	r := &RenderIgnorePointer{ignoring: firstOpts(opts).Ignoring.OrElse(true)}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Ignoring is whether this render object is ignored during hit testing.
//
// Flutter: rendering/proxy_box.dart RenderIgnorePointer.ignoring
func (r *RenderIgnorePointer) Ignoring() bool { return r.ignoring }

// SetIgnoring changes the flag.
//
// Flutter: rendering/proxy_box.dart RenderIgnorePointer.ignoring
func (r *RenderIgnorePointer) SetIgnoring(value bool) {
	if value == r.ignoring {
		return
	}
	r.ignoring = value
	r.MarkNeedsSemanticsUpdate()
}

// DescribeSemanticsConfiguration blocks the user actions of the subtree when
// ignoring.
//
// Flutter: rendering/proxy_box.dart RenderIgnorePointer.describeSemanticsConfiguration
func (r *RenderIgnorePointer) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBox.DescribeSemanticsConfiguration(config)
	config.IsBlockingUserActions = r.ignoring
}

// HitTest is false when ignoring.
//
// Flutter: rendering/proxy_box.dart RenderIgnorePointer.hitTest
func (r *RenderIgnorePointer) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	return !r.ignoring && r.RenderProxyBox.HitTest(result, position)
}

// DebugFillProperties adds the flag.
//
// Flutter: rendering/proxy_box.dart RenderIgnorePointer.debugFillProperties
func (r *RenderIgnorePointer) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("ignoring", r.ignoring))
}

// RenderOffstage lays out its child as if it was in the tree, but does not
// paint it and does not let it hit test.
//
// Flutter: rendering/proxy_box.dart RenderOffstage
type RenderOffstage struct {
	RenderProxyBox
	offstage bool
}

// RenderOffstageOpts are the optional parameters of NewRenderOffstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.new
type RenderOffstageOpts struct {
	// Offstage defaults to true.
	Offstage foundation.Opt[bool]
}

// NewRenderOffstage creates the box.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.new
func NewRenderOffstage(child Box, opts ...RenderOffstageOpts) *RenderOffstage {
	r := &RenderOffstage{offstage: firstOpts(opts).Offstage.OrElse(true)}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Offstage is whether the child is hidden from the rest of the tree.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.offstage
func (r *RenderOffstage) Offstage() bool { return r.offstage }

// SetOffstage changes the flag; the box is sized by its parent when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.offstage
func (r *RenderOffstage) SetOffstage(value bool) {
	if value == r.offstage {
		return
	}
	r.offstage = value
	r.MarkNeedsLayoutForSizedByParentChange()
}

// ComputeMinIntrinsicWidth is 0 when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.computeMinIntrinsicWidth
func (r *RenderOffstage) ComputeMinIntrinsicWidth(height float64) float64 {
	if r.offstage {
		return 0
	}
	return r.RenderProxyBox.ComputeMinIntrinsicWidth(height)
}

// ComputeMaxIntrinsicWidth is 0 when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.computeMaxIntrinsicWidth
func (r *RenderOffstage) ComputeMaxIntrinsicWidth(height float64) float64 {
	if r.offstage {
		return 0
	}
	return r.RenderProxyBox.ComputeMaxIntrinsicWidth(height)
}

// ComputeMinIntrinsicHeight is 0 when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.computeMinIntrinsicHeight
func (r *RenderOffstage) ComputeMinIntrinsicHeight(width float64) float64 {
	if r.offstage {
		return 0
	}
	return r.RenderProxyBox.ComputeMinIntrinsicHeight(width)
}

// ComputeMaxIntrinsicHeight is 0 when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.computeMaxIntrinsicHeight
func (r *RenderOffstage) ComputeMaxIntrinsicHeight(width float64) float64 {
	if r.offstage {
		return 0
	}
	return r.RenderProxyBox.ComputeMaxIntrinsicHeight(width)
}

// ComputeDistanceToActualBaseline is none when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.computeDistanceToActualBaseline
func (r *RenderOffstage) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	if r.offstage {
		return foundation.None[float64]()
	}
	return r.RenderProxyBox.ComputeDistanceToActualBaseline(baseline)
}

// SizedByParent is true when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.sizedByParent
func (r *RenderOffstage) SizedByParent() bool { return r.offstage }

// ComputeDryBaseline is none when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.computeDryBaseline
func (r *RenderOffstage) ComputeDryBaseline(constraints BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if r.offstage {
		return foundation.None[float64]()
	}
	return r.RenderProxyBox.ComputeDryBaseline(constraints, baseline)
}

// ComputeDryLayout is the smallest size when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.computeDryLayout
func (r *RenderOffstage) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if r.offstage {
		return constraints.Smallest()
	}
	return r.RenderProxyBox.ComputeDryLayout(constraints)
}

// PerformResize takes the smallest size; only an offstage box is sized by its
// parent.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.performResize
func (r *RenderOffstage) PerformResize() {
	if foundation.DebugMode && !r.offstage {
		panic(errors.New("OffstagePerformResize", "PerformResize was called on a RenderOffstage that is not offstage",
			"object", foundation.DescribeIdentity(r.self)))
	}
	r.RenderProxyBox.PerformResize()
}

// PerformLayout lays the child out with the constraints even when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.performLayout
func (r *RenderOffstage) PerformLayout() {
	if r.offstage {
		if r.hasChild() {
			r.Child().Layout(r.BoxConstraints())
		}
		return
	}
	r.RenderProxyBox.PerformLayout()
}

// HitTest is false when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.hitTest
func (r *RenderOffstage) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	return !r.offstage && r.RenderProxyBox.HitTest(result, position)
}

// PaintsChild is false when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.paintsChild
func (r *RenderOffstage) PaintsChild(child RenderObject) bool {
	if foundation.DebugMode && child.Parent() != RenderObject(r.self) {
		panic(errors.New("RenderObjectNotChild", "PaintsChild was called with a render object that is not a child",
			"object", foundation.DescribeIdentity(r.self), "child", foundation.DescribeIdentity(child)))
	}
	return !r.offstage
}

// Paint paints nothing when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.paint
func (r *RenderOffstage) Paint(context *PaintingContext, offset ui.Offset) {
	if r.offstage {
		return
	}
	r.RenderProxyBox.Paint(context, offset)
}

// DebugFillProperties adds the flag.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.debugFillProperties
func (r *RenderOffstage) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("offstage", r.offstage))
}

// VisitChildrenForSemantics visits no child when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.visitChildrenForSemantics
func (r *RenderOffstage) VisitChildrenForSemantics(visitor RenderObjectVisitor) {
	if r.offstage {
		return
	}
	r.RenderProxyBox.VisitChildrenForSemantics(visitor)
}

// DebugDescribeChildren describes the child in the offstage style when offstage.
//
// Flutter: rendering/proxy_box.dart RenderOffstage.debugDescribeChildren
func (r *RenderOffstage) DebugDescribeChildren() []foundation.DiagnosticsNode {
	if !r.hasChild() {
		return nil
	}
	style := foundation.DiagnosticsTreeStyleSparse
	if r.offstage {
		style = foundation.DiagnosticsTreeStyleOffstage
	}
	return []foundation.DiagnosticsNode{foundation.ToDiagnosticsNode(r.Child(),
		foundation.DiagnosticsNodeOpts{Name: foundation.Some("child"), Style: style})}
}

// RenderAbsorbPointer absorbs pointer events when absorbing.
//
// Deviation: the deprecated ignoringSemantics parameter is not ported, so it is
// always null (SPEC §12.10).
//
// Flutter: rendering/proxy_box.dart RenderAbsorbPointer
type RenderAbsorbPointer struct {
	RenderProxyBox
	absorbing bool
}

// RenderAbsorbPointerOpts are the optional parameters of NewRenderAbsorbPointer.
//
// Flutter: rendering/proxy_box.dart RenderAbsorbPointer.new
type RenderAbsorbPointerOpts struct {
	// Absorbing defaults to true.
	Absorbing foundation.Opt[bool]
}

// NewRenderAbsorbPointer creates the box.
//
// Flutter: rendering/proxy_box.dart RenderAbsorbPointer.new
func NewRenderAbsorbPointer(child Box, opts ...RenderAbsorbPointerOpts) *RenderAbsorbPointer {
	r := &RenderAbsorbPointer{absorbing: firstOpts(opts).Absorbing.OrElse(true)}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Absorbing is whether this render object absorbs pointers during hit testing.
//
// Flutter: rendering/proxy_box.dart RenderAbsorbPointer.absorbing
func (r *RenderAbsorbPointer) Absorbing() bool { return r.absorbing }

// SetAbsorbing changes the flag.
//
// Flutter: rendering/proxy_box.dart RenderAbsorbPointer.absorbing
func (r *RenderAbsorbPointer) SetAbsorbing(value bool) {
	if r.absorbing == value {
		return
	}
	r.absorbing = value
	r.MarkNeedsSemanticsUpdate()
}

// DescribeSemanticsConfiguration blocks the user actions of the subtree when
// absorbing.
//
// Flutter: rendering/proxy_box.dart RenderAbsorbPointer.describeSemanticsConfiguration
func (r *RenderAbsorbPointer) DescribeSemanticsConfiguration(config *semantics.SemanticsConfiguration) {
	r.RenderProxyBox.DescribeSemanticsConfiguration(config)
	config.IsBlockingUserActions = r.absorbing
}

// HitTest is true inside the box when absorbing, without testing the child.
//
// Flutter: rendering/proxy_box.dart RenderAbsorbPointer.hitTest
func (r *RenderAbsorbPointer) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	if r.absorbing {
		return r.Size().Contains(position)
	}
	return r.RenderProxyBox.HitTest(result, position)
}

// DebugFillProperties adds the flag.
//
// Flutter: rendering/proxy_box.dart RenderAbsorbPointer.debugFillProperties
func (r *RenderAbsorbPointer) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("absorbing", r.absorbing))
}

// RenderMetaData holds opaque meta data in the render tree.
//
// Flutter: rendering/proxy_box.dart RenderMetaData
type RenderMetaData struct {
	RenderProxyBoxWithHitTestBehavior
	// MetaData is opaque meta data ignored by the render tree.
	//
	// Flutter: rendering/proxy_box.dart RenderMetaData.metaData
	MetaData any
}

// RenderMetaDataOpts are the optional parameters of NewRenderMetaData.
//
// Flutter: rendering/proxy_box.dart RenderMetaData.new
type RenderMetaDataOpts struct {
	// MetaData is the meta data.
	MetaData any
	// Behavior defaults to HitTestBehaviorDeferToChild.
	Behavior HitTestBehavior
}

// NewRenderMetaData creates the box.
//
// Flutter: rendering/proxy_box.dart RenderMetaData.new
func NewRenderMetaData(child Box, opts ...RenderMetaDataOpts) *RenderMetaData {
	o := firstOpts(opts)
	r := &RenderMetaData{MetaData: o.MetaData}
	InitRenderProxyBoxWithHitTestBehavior(&r.RenderProxyBoxWithHitTestBehavior, r, o.Behavior, child)
	return r
}

// DebugFillProperties adds the meta data.
//
// Flutter: rendering/proxy_box.dart RenderMetaData.debugFillProperties
func (r *RenderMetaData) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBoxWithHitTestBehavior.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("metaData", r.MetaData))
}

// RenderLeaderLayer provides an anchor for a RenderFollowerLayer.
//
// Flutter: rendering/proxy_box.dart RenderLeaderLayer
type RenderLeaderLayer struct {
	RenderProxyBox
	link               *LayerLink
	previousLayoutSize foundation.Opt[ui.Size]
}

// NewRenderLeaderLayer creates the box.
//
// Flutter: rendering/proxy_box.dart RenderLeaderLayer.new
func NewRenderLeaderLayer(link *LayerLink, child Box) *RenderLeaderLayer {
	r := &RenderLeaderLayer{link: link}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Link is the link object that connects this leader with followers.
//
// Flutter: rendering/proxy_box.dart RenderLeaderLayer.link
func (r *RenderLeaderLayer) Link() *LayerLink { return r.link }

// SetLink changes the link and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderLeaderLayer.link
func (r *RenderLeaderLayer) SetLink(value *LayerLink) {
	if r.link == value {
		return
	}
	r.link.LeaderSize = foundation.None[ui.Size]()
	r.link = value
	if r.previousLayoutSize.IsSome() {
		r.link.LeaderSize = r.previousLayoutSize
	}
	r.MarkNeedsPaint()
}

// AlwaysNeedsCompositing is true.
//
// Flutter: rendering/proxy_box.dart RenderLeaderLayer.alwaysNeedsCompositing
func (r *RenderLeaderLayer) AlwaysNeedsCompositing() bool { return true }

// PerformLayout lays out as a proxy box and tells the link the size.
//
// Flutter: rendering/proxy_box.dart RenderLeaderLayer.performLayout
func (r *RenderLeaderLayer) PerformLayout() {
	r.RenderProxyBox.PerformLayout()
	r.previousLayoutSize = foundation.Some(r.Size())
	r.link.LeaderSize = r.previousLayoutSize
}

// Paint pushes a leader layer.
//
// Flutter: rendering/proxy_box.dart RenderLeaderLayer.paint
func (r *RenderLeaderLayer) Paint(context *PaintingContext, offset ui.Offset) {
	layer := layerAs[*LeaderLayer](r.Layer())
	if layer == nil {
		layer = NewLeaderLayer(r.link, LeaderLayerOpts{Offset: offset})
		r.SetLayer(layer)
	} else {
		layer.SetLink(r.link)
		layer.SetOffset(offset)
	}
	context.PushLayer(layer, r.RenderProxyBox.Paint, ui.OffsetZero, foundation.None[ui.Rect]())
}

// DebugFillProperties adds the link.
//
// Flutter: rendering/proxy_box.dart RenderLeaderLayer.debugFillProperties
func (r *RenderLeaderLayer) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("link", r.link))
}

// RenderFollowerLayer transforms its child so that it follows a
// RenderLeaderLayer.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer
type RenderFollowerLayer struct {
	RenderProxyBox
	link             *LayerLink
	showWhenUnlinked bool
	offset           ui.Offset
	leaderAnchor     painting.Alignment
	followerAnchor   painting.Alignment
}

// RenderFollowerLayerOpts are the optional parameters of NewRenderFollowerLayer.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.new
type RenderFollowerLayerOpts struct {
	// ShowWhenUnlinked defaults to true.
	ShowWhenUnlinked foundation.Opt[bool]
	// Offset is the additional offset to apply to the follower.
	Offset ui.Offset
	// LeaderAnchor is the anchor on the leader; the default is the top left.
	LeaderAnchor foundation.Opt[painting.Alignment]
	// FollowerAnchor is the anchor on the follower; the default is the top left.
	FollowerAnchor foundation.Opt[painting.Alignment]
}

// NewRenderFollowerLayer creates the box.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.new
func NewRenderFollowerLayer(link *LayerLink, child Box, opts ...RenderFollowerLayerOpts) *RenderFollowerLayer {
	o := firstOpts(opts)
	r := &RenderFollowerLayer{link: link, showWhenUnlinked: o.ShowWhenUnlinked.OrElse(true), offset: o.Offset,
		leaderAnchor: o.LeaderAnchor.OrElse(painting.AlignmentTopLeft), followerAnchor: o.FollowerAnchor.OrElse(painting.AlignmentTopLeft)}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Link is the link object that connects this follower with a leader.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.link
func (r *RenderFollowerLayer) Link() *LayerLink { return r.link }

// SetLink changes the link and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.link
func (r *RenderFollowerLayer) SetLink(value *LayerLink) {
	if r.link == value {
		return
	}
	r.link = value
	r.MarkNeedsPaint()
}

// ShowWhenUnlinked is whether the child is shown when the link has no leader.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.showWhenUnlinked
func (r *RenderFollowerLayer) ShowWhenUnlinked() bool { return r.showWhenUnlinked }

// SetShowWhenUnlinked changes the flag and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.showWhenUnlinked
func (r *RenderFollowerLayer) SetShowWhenUnlinked(value bool) {
	if r.showWhenUnlinked == value {
		return
	}
	r.showWhenUnlinked = value
	r.MarkNeedsPaint()
}

// Offset is the offset to apply to the follower relative to the anchors.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.offset
func (r *RenderFollowerLayer) Offset() ui.Offset { return r.offset }

// SetOffset changes the offset and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.offset
func (r *RenderFollowerLayer) SetOffset(value ui.Offset) {
	if r.offset == value {
		return
	}
	r.offset = value
	r.MarkNeedsPaint()
}

// LeaderAnchor is the anchor point on the leader that is linked to the follower.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.leaderAnchor
func (r *RenderFollowerLayer) LeaderAnchor() painting.Alignment { return r.leaderAnchor }

// SetLeaderAnchor changes the anchor and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.leaderAnchor
func (r *RenderFollowerLayer) SetLeaderAnchor(value painting.Alignment) {
	if r.leaderAnchor == value {
		return
	}
	r.leaderAnchor = value
	r.MarkNeedsPaint()
}

// FollowerAnchor is the anchor point on the follower that is linked to the
// leader.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.followerAnchor
func (r *RenderFollowerLayer) FollowerAnchor() painting.Alignment { return r.followerAnchor }

// SetFollowerAnchor changes the anchor and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.followerAnchor
func (r *RenderFollowerLayer) SetFollowerAnchor(value painting.Alignment) {
	if r.followerAnchor == value {
		return
	}
	r.followerAnchor = value
	r.MarkNeedsPaint()
}

// Detach lets the layer go.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.detach
func (r *RenderFollowerLayer) Detach() {
	r.SetLayer(nil)
	r.RenderProxyBox.Detach()
}

// AlwaysNeedsCompositing is true.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.alwaysNeedsCompositing
func (r *RenderFollowerLayer) AlwaysNeedsCompositing() bool { return true }

// GetCurrentTransform is the transform that was applied the last time the layer
// was composited, or the identity.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.getCurrentTransform
func (r *RenderFollowerLayer) GetCurrentTransform() *ui.Matrix4 {
	if layer := layerAs[*FollowerLayer](r.Layer()); layer != nil {
		if t := layer.LastTransform(); t != nil {
			return t
		}
	}
	return ui.NewMatrix4Identity()
}

// HitTest is false for an unlinked follower that is not shown.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.hitTest
func (r *RenderFollowerLayer) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	if r.link.Leader() == nil && !r.showWhenUnlinked {
		return false
	}
	return r.self.(Box).HitTestChildren(result, position)
}

// HitTestChildren hit tests the child in the followed coordinates.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.hitTestChildren
func (r *RenderFollowerLayer) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	return result.AddWithPaintTransform(r.GetCurrentTransform(), position, func(result *BoxHitTestResult, position ui.Offset) bool {
		return r.RenderProxyBox.HitTestChildren(result, position)
	})
}

// Paint pushes a follower layer.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.paint
func (r *RenderFollowerLayer) Paint(context *PaintingContext, offset ui.Offset) {
	leaderSize := r.link.LeaderSize
	if foundation.DebugMode && leaderSize.IsNone() && r.link.Leader() != nil && r.leaderAnchor != painting.AlignmentTopLeft {
		panic(errors.New("FollowerLeaderSizeMissing", "the layer is linked to a leader but a valid leader size is not set; "+
			"the leader size is required when the leader anchor is not the top left",
			"link", r.link.String(), "leaderAnchor", r.leaderAnchor))
	}
	effectiveLinked := r.offset
	if size, ok := leaderSize.Get(); ok {
		effectiveLinked = r.leaderAnchor.AlongSize(size).Sub(r.followerAnchor.AlongSize(r.Size())).Add(r.offset)
	}
	layer := layerAs[*FollowerLayer](r.Layer())
	if layer == nil {
		layer = NewFollowerLayer(r.link, FollowerLayerOpts{HideWhenUnlinked: !r.showWhenUnlinked,
			LinkedOffset: effectiveLinked, UnlinkedOffset: offset})
		r.SetLayer(layer)
	} else {
		layer.Link = r.link
		layer.ShowWhenUnlinked = r.showWhenUnlinked
		layer.LinkedOffset = effectiveLinked
		layer.UnlinkedOffset = offset
	}
	context.PushLayer(layer, r.RenderProxyBox.Paint, ui.OffsetZero,
		foundation.Some(ui.NewRectFromLTRB(math.Inf(-1), math.Inf(-1), math.Inf(1), math.Inf(1))))
}

// ApplyPaintTransform applies the followed transform.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.applyPaintTransform
func (r *RenderFollowerLayer) ApplyPaintTransform(_ RenderObject, transform *ui.Matrix4) {
	transform.MultiplyBy(r.GetCurrentTransform())
}

// DebugFillProperties adds the link, the flag, the offset and the transform.
//
// Flutter: rendering/proxy_box.dart RenderFollowerLayer.debugFillProperties
func (r *RenderFollowerLayer) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("link", r.link))
	b.Add(foundation.NewDiagnosticsProperty("showWhenUnlinked", r.showWhenUnlinked))
	b.Add(foundation.NewDiagnosticsProperty("offset", r.offset))
	b.Add(painting.NewTransformProperty("current transform matrix", r.GetCurrentTransform()))
}

// RenderAnnotatedRegion annotates a region of the layer tree with a value.
//
// Deviation: the value is an any, not a type parameter (SPEC §12.10, like
// AnnotatedRegionLayer).
//
// Flutter: rendering/proxy_box.dart RenderAnnotatedRegion
type RenderAnnotatedRegion struct {
	RenderProxyBox
	value       any
	sized       bool
	layerHandle *LayerHandle[*AnnotatedRegionLayer]
}

// NewRenderAnnotatedRegion creates the box. A sized region only covers the box
// itself.
//
// Flutter: rendering/proxy_box.dart RenderAnnotatedRegion.new
func NewRenderAnnotatedRegion(value any, sized bool, child Box) *RenderAnnotatedRegion {
	r := &RenderAnnotatedRegion{value: value, sized: sized, layerHandle: NewLayerHandle[*AnnotatedRegionLayer](nil)}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Value is the value that is provided for the region.
//
// Flutter: rendering/proxy_box.dart RenderAnnotatedRegion.value
func (r *RenderAnnotatedRegion) Value() any { return r.value }

// SetValue changes the value and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderAnnotatedRegion.value
func (r *RenderAnnotatedRegion) SetValue(v any) {
	if r.value == v {
		return
	}
	r.value = v
	r.MarkNeedsPaint()
}

// Sized is whether the region is limited to the size of the box.
//
// Flutter: rendering/proxy_box.dart RenderAnnotatedRegion.sized
func (r *RenderAnnotatedRegion) Sized() bool { return r.sized }

// SetSized changes the flag and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderAnnotatedRegion.sized
func (r *RenderAnnotatedRegion) SetSized(v bool) {
	if r.sized == v {
		return
	}
	r.sized = v
	r.MarkNeedsPaint()
}

// AlwaysNeedsCompositing is true.
//
// Flutter: rendering/proxy_box.dart RenderAnnotatedRegion.alwaysNeedsCompositing
func (r *RenderAnnotatedRegion) AlwaysNeedsCompositing() bool { return true }

// Paint pushes an annotated region layer.
//
// Flutter: rendering/proxy_box.dart RenderAnnotatedRegion.paint
func (r *RenderAnnotatedRegion) Paint(context *PaintingContext, offset ui.Offset) {
	var o AnnotatedRegionLayerOpts
	if r.sized {
		o.Size = foundation.Some(r.Size())
		o.Offset = offset
	}
	layer := NewAnnotatedRegionLayer(r.value, o)
	r.layerHandle.SetLayer(layer)
	context.PushLayer(layer, r.RenderProxyBox.Paint, offset, foundation.None[ui.Rect]())
}

// Dispose lets the layer go.
//
// Flutter: rendering/proxy_box.dart RenderAnnotatedRegion.dispose
func (r *RenderAnnotatedRegion) Dispose() {
	r.layerHandle.SetLayer(nil)
	r.RenderProxyBox.Dispose()
}
