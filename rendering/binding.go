package rendering

import (
	"time"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// RendererBindingOpts are the optional parameters of NewRendererBinding.
//
// Flutter: rendering/binding.dart RendererBinding
type RendererBindingOpts struct {
	// Debug are the debug flags that the render objects read; the default is a
	// fresh set with every flag off.
	Debug *DebugOptions
	// FallbackCursor is the cursor of the mouse tracker when no region asks for
	// one. Without it the binding has no mouse tracker.
	//
	// TODO(step 10.4): default to SystemMouseCursors.basic.
	FallbackCursor services.MouseCursor
	// ReportError is called with the error of a frame that could not be sent to
	// the engine; the default panics with the error.
	//
	// TODO(step 7.3): report through the framework's error reporter.
	ReportError func(err error)
}

// RendererBinding is the glue between the render trees and the engine: it owns
// the root pipeline owner, the render views and the mouse tracker, draws the
// frame (layout, compositing bits, paint, composite, semantics) and hit tests
// the views.
//
// Deviation: Flutter's is a mixin on BindingBase with a static instance; this is
// a struct made from the scheduler and gesture bindings (SPEC §12.10).
//
// Flutter: rendering/binding.dart RendererBinding
type RendererBinding struct {
	d         *ui.PlatformDispatcher
	scheduler *scheduler.SchedulerBinding
	gesture   *gestures.GestureBinding

	debug          *DebugOptions
	reportError    func(error)
	manifold       *bindingPipelineManifold
	rootOwner      *PipelineOwner
	mouseTracker   *MouseTracker
	semantics      *semantics.SemanticsBinding
	views          []*RenderView
	firstFrameDefs int
	firstFrameSent bool
}

// NewRendererBinding creates the binding. It sets the dispatcher's metrics
// callback (calling the one that was set), adds the persistent frame callback to
// the scheduler, and plugs the hit test of the views and the mouse tracking into
// the gesture binding.
//
// Flutter: rendering/binding.dart RendererBinding.initInstances
func NewRendererBinding(d *ui.PlatformDispatcher, s *scheduler.SchedulerBinding, g *gestures.GestureBinding, opts ...RendererBindingOpts) *RendererBinding {
	o := firstOpts(opts)
	if o.Debug == nil {
		o.Debug = NewDebugOptions()
	}
	b := &RendererBinding{d: d, scheduler: s, gesture: g, debug: o.Debug, reportError: o.ReportError}
	if o.ReportError != nil && o.Debug.ReportError == nil {
		o.Debug.ReportError = o.ReportError
	}
	if b.reportError == nil {
		b.reportError = func(err error) { panic(err) }
	}
	b.semantics = semantics.NewSemanticsBinding(d, s, semantics.SemanticsBindingOpts{PerformSemanticsAction: b.performSemanticsAction})
	b.manifold = &bindingPipelineManifold{binding: b}
	b.semantics.AddSemanticsEnabledListener(b.manifold.NotifyListeners)
	b.rootOwner = b.CreateRootPipelineOwner()
	previous := d.OnMetricsChanged
	d.OnMetricsChanged = func() {
		if previous != nil {
			previous()
		}
		b.handleMetricsChanged()
	}
	s.AddPersistentFrameCallback(b.handlePersistentFrame)
	if o.FallbackCursor != nil {
		b.InitMouseTracker(NewMouseTracker(func(position ui.Offset, viewID ui.ViewID) *gestures.HitTestResult {
			result := gestures.NewHitTestResult()
			b.HitTestInView(result, position, viewID)
			return result
		}, o.FallbackCursor))
	}
	g.OnHitTestInView = b.hitTestRenderView
	g.OnDispatchEvent = b.onDispatchEvent
	b.rootOwner.Attach(b.manifold)
	return b
}

// DebugOptions are the debug flags of the rendering library.
//
// Flutter: rendering/debug.dart
func (b *RendererBinding) DebugOptions() *DebugOptions { return b.debug }

// CreateRootPipelineOwner creates the root pipeline owner: one that manages
// child owners, not a root node. SetRootNode on it panics with RootNodeNotAllowed.
//
// Flutter: rendering/binding.dart RendererBinding.createRootPipelineOwner
func (b *RendererBinding) CreateRootPipelineOwner() *PipelineOwner {
	return NewPipelineOwner(PipelineOwnerOpts{
		DisallowRootNode: true, Debug: b.debug,
		// Never called: the root owner has no root node.
		OnSemanticsUpdate: func(*ui.SemanticsUpdate) {},
	})
}

// RootPipelineOwner is the root of the tree of pipeline owners; render views'
// owners are its children.
//
// Flutter: rendering/binding.dart RendererBinding.rootPipelineOwner
func (b *RendererBinding) RootPipelineOwner() *PipelineOwner { return b.rootOwner }

// SemanticsBinding is the semantics binding that the renderer binding made: it
// says whether semantics are enabled, and passes the actions of assistive
// technologies to the semantics owners of the render views.
//
// Deviation: Flutter's RendererBinding is mixed with SemanticsBinding; this one
// owns it (SPEC §12.10).
//
// Flutter: semantics/binding.dart SemanticsBinding.instance
func (b *RendererBinding) SemanticsBinding() *semantics.SemanticsBinding { return b.semantics }

// Scheduler is the scheduler binding.
func (b *RendererBinding) Scheduler() *scheduler.SchedulerBinding { return b.scheduler }

// MouseTracker is the mouse tracker, or nil if the binding was made without a
// fallback cursor.
//
// Flutter: rendering/binding.dart RendererBinding.mouseTracker
func (b *RendererBinding) MouseTracker() *MouseTracker { return b.mouseTracker }

// InitMouseTracker replaces the mouse tracker, disposing the old one.
//
// Flutter: rendering/binding.dart RendererBinding.initMouseTracker
func (b *RendererBinding) InitMouseTracker(tracker *MouseTracker) {
	if b.mouseTracker != nil {
		b.mouseTracker.Dispose()
	}
	b.mouseTracker = tracker
}

// RenderViews are the render views that were added, in order.
//
// Flutter: rendering/binding.dart RendererBinding.renderViews
func (b *RendererBinding) RenderViews() []*RenderView { return append([]*RenderView(nil), b.views...) }

func (b *RendererBinding) renderViewFor(id ui.ViewID) *RenderView {
	for _, v := range b.views {
		if v.view.ViewID() == id {
			return v
		}
	}
	return nil
}

// AddRenderView adds the render view and gives it its configuration. Adding a
// view twice, or two views of one view ID, panics with RenderViewAlreadyAdded in
// debug builds.
//
// Flutter: rendering/binding.dart RendererBinding.addRenderView
func (b *RendererBinding) AddRenderView(view *RenderView) {
	if foundation.DebugMode && b.renderViewFor(view.view.ViewID()) != nil {
		panic(errors.New("RenderViewAlreadyAdded", "a render view for this view was added already",
			"view", int64(view.view.ViewID())))
	}
	b.views = append(b.views, view)
	view.SetConfiguration(b.CreateViewConfigurationFor(view))
}

// RemoveRenderView removes the render view.
//
// Flutter: rendering/binding.dart RendererBinding.removeRenderView
func (b *RendererBinding) RemoveRenderView(view *RenderView) {
	for i, v := range b.views {
		if v == view {
			b.views = append(b.views[:i], b.views[i+1:]...)
			return
		}
	}
	if foundation.DebugMode {
		panic(errors.New("RenderViewNotAdded", "the render view was not added to the binding",
			"view", int64(view.view.ViewID())))
	}
}

// CreateViewConfigurationFor makes the configuration of the render view from
// its view's metrics.
//
// Flutter: rendering/binding.dart RendererBinding.createViewConfigurationFor
func (b *RendererBinding) CreateViewConfigurationFor(view *RenderView) ViewConfiguration {
	return NewViewConfigurationFromView(view.FlutterView())
}

func (b *RendererBinding) handleMetricsChanged() {
	forceFrame := false
	for _, v := range b.views {
		forceFrame = forceFrame || !isNil(v.Child())
		v.SetConfiguration(b.CreateViewConfigurationFor(v))
	}
	if forceFrame {
		b.scheduler.ScheduleForcedFrame()
	}
}

func (b *RendererBinding) onDispatchEvent(event gestures.PointerEvent, hitTestResult *gestures.HitTestResult) {
	if b.mouseTracker == nil {
		return
	}
	if _, ok := event.(*gestures.PointerMoveEvent); ok {
		hitTestResult = nil
	}
	b.mouseTracker.UpdateWithEvent(event, hitTestResult)
}

func (b *RendererBinding) hitTestRenderView(result *gestures.HitTestResult, position ui.Offset, viewID ui.ViewID) {
	if v := b.renderViewFor(viewID); v != nil {
		v.HitTest(result, position)
	}
}

// HitTestInView hit tests the render view of the view ID, then adds the gesture
// binding, as the last target.
//
// Flutter: rendering/binding.dart RendererBinding.hitTestInView
func (b *RendererBinding) HitTestInView(result *gestures.HitTestResult, position ui.Offset, viewID ui.ViewID) {
	b.gesture.HitTestInView(result, position, viewID)
}

func (b *RendererBinding) handlePersistentFrame(_ time.Duration) {
	if err := b.DrawFrame(); err != nil {
		b.reportError(err)
	}
	b.scheduleMouseTrackerUpdate()
}

func (b *RendererBinding) scheduleMouseTrackerUpdate() {
	if b.mouseTracker == nil {
		return
	}
	b.scheduler.AddPostFrameCallback(func(time.Duration) {
		b.mouseTracker.UpdateAllDevices()
	})
}

// SendFramesToEngine reports whether the frames go to the engine: they do once
// the first frame was sent, and when no widget deferred the first frame.
//
// Flutter: rendering/binding.dart RendererBinding.sendFramesToEngine
func (b *RendererBinding) SendFramesToEngine() bool {
	return b.firstFrameSent || b.firstFrameDefs == 0
}

// DeferFirstFrame stops the frames from going to the engine until
// AllowFirstFrame is called as often.
//
// Flutter: rendering/binding.dart RendererBinding.deferFirstFrame
func (b *RendererBinding) DeferFirstFrame() { b.firstFrameDefs++ }

// AllowFirstFrame undoes a DeferFirstFrame; when the last one is undone and no
// frame was sent, it schedules a frame.
//
// Deviation: Flutter schedules a warm-up frame, which the scheduler does not
// have (SPEC §12.10).
//
// Flutter: rendering/binding.dart RendererBinding.allowFirstFrame
func (b *RendererBinding) AllowFirstFrame() {
	if foundation.DebugMode && b.firstFrameDefs <= 0 {
		panic(errors.New("AllowFirstFrameUnbalanced", "AllowFirstFrame was called more often than DeferFirstFrame"))
	}
	b.firstFrameDefs--
	if !b.firstFrameSent {
		b.scheduler.ScheduleFrame()
	}
}

// ResetFirstFrameSent lets the binding defer the first frame again.
//
// Flutter: rendering/binding.dart RendererBinding.resetFirstFrameSent
func (b *RendererBinding) ResetFirstFrameSent() { b.firstFrameSent = false }

// DrawFrame lays out, updates the compositing bits, paints, composites the views
// (if frames go to the engine) and flushes the semantics. It returns the first
// error of a view's compositing.
//
// Flutter: rendering/binding.dart RendererBinding.drawFrame
func (b *RendererBinding) DrawFrame() error {
	b.rootOwner.FlushLayout()
	b.rootOwner.FlushCompositingBits()
	b.rootOwner.FlushPaint()
	var firstErr error
	if b.SendFramesToEngine() {
		for _, v := range b.views {
			if err := v.CompositeFrame(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		b.rootOwner.FlushSemantics()
		b.firstFrameSent = true
	}
	return firstErr
}

// Reassemble marks every render view and its subtree for relayout and repaint,
// then schedules a frame (Flutter schedules a warm-up frame).
//
// Flutter: rendering/binding.dart RendererBinding.performReassemble
func (b *RendererBinding) Reassemble() {
	for _, v := range b.views {
		v.Reassemble()
	}
	b.scheduler.ScheduleFrame()
}

// ForceRepaint marks every render object of every view as needing paint, and
// returns the channel that is closed at the end of the next frame.
//
// Flutter: rendering/binding.dart RendererBinding._forceRepaint
func (b *RendererBinding) ForceRepaint() <-chan struct{} {
	var visitor RenderObjectVisitor
	visitor = func(child RenderObject) {
		child.MarkNeedsPaint()
		child.VisitChildren(visitor)
	}
	for _, v := range b.views {
		v.VisitChildren(visitor)
	}
	return b.scheduler.EndOfFrame()
}

// DebugDumpRenderTree is the text dump of every render view's tree.
//
// Flutter: rendering/binding.dart debugDumpRenderTree
func (b *RendererBinding) DebugDumpRenderTree() string {
	if len(b.views) == 0 {
		return "No render tree root was added to the binding."
	}
	out := ""
	for i, v := range b.views {
		if i > 0 {
			out += "\n\n"
		}
		out += v.ToStringDeep()
	}
	return out
}

// DebugDumpLayerTree is the text dump of every render view's layer tree.
//
// Flutter: rendering/binding.dart debugDumpLayerTree
func (b *RendererBinding) DebugDumpLayerTree() string {
	if len(b.views) == 0 {
		return "No render tree root was added to the binding."
	}
	out := ""
	for i, v := range b.views {
		if i > 0 {
			out += "\n\n"
		}
		if l := v.DebugLayer(); l != nil {
			out += foundation.ToStringDeep(l)
		} else {
			out += "Layer tree unavailable for " + v.ToStringShort() + "."
		}
	}
	return out
}

// DebugDumpPipelineOwnerTree is the text dump of the tree of pipeline owners.
//
// Flutter: rendering/binding.dart debugDumpPipelineOwnerTree
func (b *RendererBinding) DebugDumpPipelineOwnerTree() string {
	return foundation.ToStringDeep(b.rootOwner)
}

// bindingPipelineManifold connects the pipeline owners to the scheduler.
//
// Flutter: rendering/binding.dart _BindingPipelineManifold
type bindingPipelineManifold struct {
	foundation.ChangeNotifier
	binding *RendererBinding
}

func (m *bindingPipelineManifold) RequestVisualUpdate() { m.binding.scheduler.EnsureVisualUpdate() }

func (m *bindingPipelineManifold) SemanticsEnabled() bool {
	return m.binding.semantics.SemanticsEnabled()
}

func (b *RendererBinding) viewByID(id ui.ViewID) *RenderView {
	for _, v := range b.views {
		if v.FlutterView().ViewID() == id {
			return v
		}
	}
	return nil
}

// performSemanticsAction performs the action on the semantics node of a view.
// Actions for views or nodes that no longer exist are ignored, since a screen
// reader may not have processed the last semantics update yet.
//
// Flutter: rendering/binding.dart RendererBinding.performSemanticsAction
func (b *RendererBinding) performSemanticsAction(action ui.SemanticsActionEvent) {
	v := b.viewByID(action.ViewID)
	if v == nil || v.Owner() == nil || v.Owner().SemanticsOwner() == nil {
		return
	}
	v.Owner().SemanticsOwner().PerformAction(action.NodeID, action.Type, action.Arguments)
}

// GetRectOfSemanticsNodeInViewCoordinates is the rectangle of the semantics node
// in the logical pixels of the view. The second result is false for an unknown
// view or node, or if the view has no semantics owner (the tag
// SemanticsLookupFailed in debug builds).
//
// Flutter: rendering/binding.dart RendererBinding.getRectOfSemanticsNodeInViewCoordinates
func (b *RendererBinding) GetRectOfSemanticsNodeInViewCoordinates(viewID ui.ViewID, nodeID int) (ui.Rect, bool) {
	fail := func(msg string) (ui.Rect, bool) {
		if foundation.DebugMode {
			panic(errors.New("SemanticsLookupFailed", msg, "view", int(viewID), "node", nodeID))
		}
		return ui.Rect{}, false
	}
	v := b.viewByID(viewID)
	if v == nil {
		return fail("GetRectOfSemanticsNodeInViewCoordinates was called for an unknown view")
	}
	if v.Owner() == nil || v.Owner().SemanticsOwner() == nil {
		return fail("GetRectOfSemanticsNodeInViewCoordinates was called for a view that has no SemanticsOwner: semantics must be enabled for the lookup to succeed")
	}
	node := v.Owner().SemanticsOwner().GetSemanticsNode(nodeID)
	if node == nil {
		return fail("GetRectOfSemanticsNodeInViewCoordinates was called for an unknown node")
	}
	transform := ui.NewMatrix4Identity()
	for current := node; current != nil; current = current.Parent() {
		if t := current.Transform(); t != nil {
			transform = t.Multiply(transform)
		}
	}
	// The walk accumulated the root transform of the view, which scales from
	// logical to physical pixels: undo it with the matrix the framework applied.
	rootInverse := v.Configuration().ToMatrix().Clone()
	rootInverse.Invert()
	transform = rootInverse.Multiply(transform)
	return transform.TransformRect(node.Rect()), true
}

// DebugDumpSemanticsTree is the text dump of the semantics tree of every render
// view, children in the given order; a view without semantics gets a notice.
//
// Flutter: rendering/binding.dart debugDumpSemanticsTree
func (b *RendererBinding) DebugDumpSemanticsTree(order semantics.DebugSemanticsDumpOrder) string {
	if len(b.views) == 0 {
		return "No render tree root was added to the binding."
	}
	const explanation = "For performance reasons, the framework only generates semantics when asked to do so by the platform.\n" +
		"Usually, platforms only ask for semantics when assistive technologies (like screen readers) are running.\n" +
		"To generate semantics, try turning on an assistive technology (like VoiceOver or TalkBack) on your device."
	out := ""
	printedExplanation := false
	for i, v := range b.views {
		if i > 0 {
			out += "\n\n"
		}
		if node := v.DebugSemantics(); node != nil {
			out += node.ToStringDeep(semantics.SemanticsToStringDeepOpts{ChildOrder: order})
			continue
		}
		msg := "Semantics not generated for " + v.ToStringShort() + "."
		if !printedExplanation {
			printedExplanation = true
			msg += "\n" + explanation
		}
		out += msg
	}
	return out
}
