package widgets

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
)

// LayoutWidgetBuilder builds a widget for the constraints that the parent gives.
//
// Flutter: widgets/layout_builder.dart LayoutWidgetBuilder
type LayoutWidgetBuilder func(bc BuildContext, constraints rendering.BoxConstraints) Widget

// layoutBuilderWidget is a widget whose child is built during layout, from the
// constraints (or the other layout information) of its render object.
//
// Flutter: widgets/layout_builder.dart AbstractLayoutBuilder
type layoutBuilderWidget interface {
	renderObjectWidget
	// buildLayout builds the child for the layout information.
	buildLayout(bc BuildContext, info any) Widget
	// updateShouldRebuild is whether the child must be built again for the new
	// widget.
	updateShouldRebuild(old Widget) bool
}

// layoutBuilderRender is a render object that runs a layout callback.
//
// Flutter: widgets/layout_builder.dart RenderAbstractLayoutBuilderMixin
type layoutBuilderRender interface {
	rendering.RenderObject
	updateCallback(func(rendering.Constraints))
	clearCallback()
	scheduleLayoutCallback()
	layoutInfo() any
}

// LayoutBuilderOpts are the optional parameters of NewLayoutBuilder.
//
// Flutter: widgets/layout_builder.dart LayoutBuilder.new
type LayoutBuilderOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Extra, when set, is called at layout time, and the builder runs again when
	// its value (a comparable value) differs from the one of the last build, even
	// if the constraints did not change. It takes the place of the extra fields of
	// the BoxConstraints subclass that Flutter's Scaffold passes to its body
	// builder, which Go's BoxConstraints cannot carry (SPEC §12.10). The function
	// of the first widget that creates the render object stays.
	Extra func() any
}

// LayoutBuilder builds a widget tree that can depend on the parent widget's
// size: the builder is called with the constraints the parent gives, at layout
// time, whenever they change or the widget is updated.
//
// Flutter: widgets/layout_builder.dart LayoutBuilder
type LayoutBuilder struct {
	RenderObjectWidget
	builder LayoutWidgetBuilder
	extra   func() any
}

// NewLayoutBuilder creates a widget that builds its child with the constraints
// of its parent.
//
// Flutter: widgets/layout_builder.dart LayoutBuilder.new
func NewLayoutBuilder(builder LayoutWidgetBuilder, opts ...LayoutBuilderOpts) *LayoutBuilder {
	o := firstOpts(opts)
	return &LayoutBuilder{RenderObjectWidget: NewRenderObjectWidgetBase(WidgetOpts{Key: o.Key}), builder: builder, extra: o.Extra}
}

// Builder is the function that builds the child.
//
// Flutter: widgets/layout_builder.dart ConstrainedLayoutBuilder.builder
func (w *LayoutBuilder) Builder() LayoutWidgetBuilder { return w.builder }

func (w *LayoutBuilder) buildLayout(bc BuildContext, info any) Widget {
	if x, ok := info.(layoutBuilderExtraInfo); ok {
		return w.builder(bc, x.constraints)
	}
	return w.builder(bc, info.(rendering.BoxConstraints))
}

// layoutBuilderExtraInfo is the layout information of a LayoutBuilder with an
// Extra function: the constraints and the extra value.
type layoutBuilderExtraInfo struct {
	constraints rendering.BoxConstraints
	extra       any
}

func (w *LayoutBuilder) updateShouldRebuild(Widget) bool { return true }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/layout_builder.dart LayoutBuilder.createRenderObject
func (w *LayoutBuilder) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := &renderLayoutBuilder{extra: w.extra}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	rendering.InitLayoutCallbackMixin(&r.LayoutCallbackMixin, r, r.layoutCallback)
	return r
}

// CreateElement creates the element, which builds the child during layout.
//
// Flutter: widgets/layout_builder.dart AbstractLayoutBuilder.createElement
func (w *LayoutBuilder) CreateElement() Element {
	e := &layoutBuilderElement{}
	InitRenderObjectElement(&e.RenderObjectElement, e, w)
	return e
}

// renderLayoutBuilder is the render object of a LayoutBuilder.
//
// Flutter: widgets/layout_builder.dart _RenderLayoutBuilder
type renderLayoutBuilder struct {
	rendering.RenderProxyBox
	rendering.LayoutCallbackMixin
	callback func(rendering.Constraints)
	extra    func() any
}

func (r *renderLayoutBuilder) updateCallback(f func(rendering.Constraints)) {
	hadCallback := r.callback != nil
	r.callback = f
	if !hadCallback {
		r.ScheduleLayoutCallback()
	}
}

func (r *renderLayoutBuilder) clearCallback()          { r.callback = nil }
func (r *renderLayoutBuilder) scheduleLayoutCallback() { r.ScheduleLayoutCallback() }
func (r *renderLayoutBuilder) layoutInfo() any {
	if r.extra != nil {
		return layoutBuilderExtraInfo{r.BoxConstraints(), r.extra()}
	}
	return r.BoxConstraints()
}

func (r *renderLayoutBuilder) layoutCallback() {
	if r.callback != nil {
		r.callback(r.Constraints())
	}
}

// The intrinsic dimensions of a LayoutBuilder are zero: calculating them would
// require running the layout callback speculatively, which might mutate the live
// render object tree.
//
// Deviation: Flutter throws in debug builds when they are asked for outside of a
// check of the intrinsics; Glutter has no such check (SPEC §12.10).
func (r *renderLayoutBuilder) ComputeMinIntrinsicWidth(float64) float64  { return 0 }
func (r *renderLayoutBuilder) ComputeMaxIntrinsicWidth(float64) float64  { return 0 }
func (r *renderLayoutBuilder) ComputeMinIntrinsicHeight(float64) float64 { return 0 }
func (r *renderLayoutBuilder) ComputeMaxIntrinsicHeight(float64) float64 { return 0 }

func (r *renderLayoutBuilder) ComputeDryLayout(rendering.BoxConstraints) ui.Size {
	r.DebugCannotComputeDryLayout("Calculating the dry layout would require running the layout callback speculatively, which might mutate the live render object tree.")
	return ui.Size{}
}

func (r *renderLayoutBuilder) ComputeDryBaseline(rendering.BoxConstraints, ui.TextBaseline) foundation.Opt[float64] {
	r.DebugCannotComputeDryLayout("Calculating the dry baseline would require running the layout callback speculatively, which might mutate the live render object tree.")
	return foundation.None[float64]()
}

// PerformLayout runs the layout callback, which builds the child, then lays the
// child out with the same constraints.
func (r *renderLayoutBuilder) PerformLayout() {
	constraints := r.BoxConstraints()
	r.RunLayoutCallback()
	if child := r.Child(); child != nil {
		child.Layout(constraints, rendering.LayoutOpts{ParentUsesSize: true})
		r.SetSize(constraints.Constrain(child.Size()))
		return
	}
	r.SetSize(constraints.Biggest())
}

// layoutBuilderElement is the element of a layout builder: it builds its child
// in its own build scope, during the layout of its render object.
//
// Flutter: widgets/layout_builder.dart _LayoutBuilderElement
type layoutBuilderElement struct {
	RenderObjectElement
	child                     Element
	scope                     *BuildScope
	deferredCallbackScheduled bool
	previousLayoutInfo        any
	needsBuild                bool
	rebuildFunc               func(rendering.Constraints)
}

func (e *layoutBuilderElement) render() layoutBuilderRender {
	return e.renderObject.(layoutBuilderRender)
}

// BuildScope is the scope of this element, which builds what is below it.
func (e *layoutBuilderElement) BuildScope() *BuildScope {
	if e.scope == nil {
		e.scope = NewBuildScope(BuildScopeOpts{ScheduleRebuild: e.scheduleRebuild})
	}
	return e.scope
}

// scheduleRebuild marks the render object as needing layout, as the rebuilding
// is done in its layout. The render tree is kept clean while the post frame
// callbacks and the idle phase run, so the work is deferred to the next frame.
func (e *layoutBuilderElement) scheduleRebuild() {
	if e.deferredCallbackScheduled {
		return
	}
	sched := e.owner.binding.Scheduler()
	switch sched.SchedulerPhase() {
	case scheduler.SchedulerPhaseIdle, scheduler.SchedulerPhasePostFrameCallbacks:
		e.deferredCallbackScheduled = true
		sched.ScheduleFrameCallback(e.frameCallback)
	default:
		e.render().scheduleLayoutCallback()
	}
}

func (e *layoutBuilderElement) frameCallback(time.Duration) {
	e.deferredCallbackScheduled = false
	if e.Mounted() {
		e.render().scheduleLayoutCallback()
	}
}

// VisitChildren visits the child.
func (e *layoutBuilderElement) VisitChildren(visitor ElementVisitor) {
	if e.child != nil {
		visitor(e.child)
	}
}

// ForgetChild drops the child.
func (e *layoutBuilderElement) ForgetChild(child Element) {
	e.child = nil
	e.RenderObjectElement.ForgetChild(child)
}

// Mount creates the render object and gives it the callback.
func (e *layoutBuilderElement) Mount(parent Element, newSlot any) {
	e.RenderObjectElement.Mount(parent, newSlot)
	e.rebuildFunc = e.rebuildWithConstraints
	e.render().updateCallback(e.rebuildFunc)
}

// Update updates the widget and builds the child again when the widget asks for
// it.
func (e *layoutBuilderElement) Update(newWidget Widget) {
	old := e.widget.(layoutBuilderWidget)
	e.RenderObjectElement.Update(newWidget)
	e.render().updateCallback(e.rebuildFunc)
	if newWidget.(layoutBuilderWidget).updateShouldRebuild(old) {
		e.needsBuild = true
		e.render().scheduleLayoutCallback()
	}
}

// MarkNeedsBuild schedules the layout callback instead of the rebuild: the
// callback does what PerformRebuild does, so the element stays clean.
func (e *layoutBuilderElement) MarkNeedsBuild() {
	e.render().scheduleLayoutCallback()
	e.needsBuild = true
}

// PerformRebuild forces the callback to be called, even if the constraints are
// the same, because it may depend on the updated widget or an inherited widget.
func (e *layoutBuilderElement) PerformRebuild() {
	e.render().scheduleLayoutCallback()
	e.needsBuild = true
	e.RenderObjectElement.PerformRebuild()
}

// Unmount takes the callback away from the render object.
func (e *layoutBuilderElement) Unmount() {
	e.render().clearCallback()
	e.RenderObjectElement.Unmount()
}

func (e *layoutBuilderElement) panicInfo() []any {
	return []any{"library", "widgets library", "element", e.String(), "widget", foundation.RuntimeTypeName(e.widget),
		"creatorChain", e.DebugGetCreatorChain(12)}
}

// rebuildWithConstraints is the layout callback: it builds the child if the
// constraints changed or a build was asked for.
func (e *layoutBuilderElement) rebuildWithConstraints(rendering.Constraints) {
	info := e.render().layoutInfo()
	update := func() {
		var built Widget
		var buildErr error
		report := func(err error) { buildErr = err }
		if e.owner.OnError == nil {
			report = nil
		}
		foundation.Guard(report, foundation.BuildPanicked, "while building "+e.String(), e.panicInfo, func() {
			built = e.widget.(layoutBuilderWidget).buildLayout(e, info)
			if foundation.DebugMode && built == nil {
				panic(errors.New("BuildReturnedNil", "a layout builder returned nil; builders must never return nil",
					"widget", foundation.RuntimeTypeName(e.widget)))
			}
		})
		if buildErr != nil {
			e.owner.OnError(buildErr)
			built = e.owner.errorWidgetFor(buildErr)
		}
		var updateErr error
		report = func(err error) { updateErr = err }
		if e.owner.OnError == nil {
			report = nil
		}
		foundation.Guard(report, foundation.BuildPanicked, "while building "+e.String(), e.panicInfo, func() {
			e.child = e.UpdateChild(e.child, built, nil)
		})
		if updateErr != nil {
			e.owner.OnError(updateErr)
			e.child = e.UpdateChild(nil, e.owner.errorWidgetFor(updateErr), e.slot)
		}
		e.needsBuild = false
		e.previousLayoutInfo = info
	}
	if e.needsBuild || info != e.previousLayoutInfo {
		e.owner.BuildScope(e, update)
	} else {
		e.owner.BuildScope(e, nil)
	}
}

// InsertRenderObjectChild sets the child of the render object.
func (e *layoutBuilderElement) InsertRenderObjectChild(child rendering.RenderObject, _ any) {
	e.renderObject.(singleChildHolder).SetChildObject(child)
}

// MoveRenderObjectChild does nothing: the single child has no other slot.
func (e *layoutBuilderElement) MoveRenderObjectChild(rendering.RenderObject, any, any) {}

// RemoveRenderObjectChild empties the render object.
func (e *layoutBuilderElement) RemoveRenderObjectChild(rendering.RenderObject, any) {
	e.renderObject.(singleChildHolder).SetChildObject(nil)
}

// OrientationWidgetBuilder builds a widget for the orientation of the parent.
//
// Flutter: widgets/layout_builder.dart OrientationWidgetBuilder
type OrientationWidgetBuilder func(bc BuildContext, orientation Orientation) Widget

// OrientationBuilder builds a widget tree that can depend on the parent widget's
// orientation (distinct from the device orientation): the width against the
// height of the constraints that it gets. Unbounded constraints are portrait.
//
// Flutter: widgets/layout_builder.dart OrientationBuilder
type OrientationBuilder struct {
	StatelessWidget
	builder OrientationWidgetBuilder
}

// NewOrientationBuilder creates a widget that builds with the orientation of the
// space it gets.
//
// Flutter: widgets/layout_builder.dart OrientationBuilder.new
func NewOrientationBuilder(builder OrientationWidgetBuilder, opts ...WidgetOpts) *OrientationBuilder {
	return &OrientationBuilder{NewStatelessWidgetBase(firstOpts(opts)), builder}
}

// Builder is the function that builds the child.
//
// Flutter: widgets/layout_builder.dart OrientationBuilder.builder
func (w *OrientationBuilder) Builder() OrientationWidgetBuilder { return w.builder }

// Build builds a LayoutBuilder that finds the orientation.
//
// Flutter: widgets/layout_builder.dart OrientationBuilder.build
func (w *OrientationBuilder) Build(BuildContext) Widget {
	return NewLayoutBuilder(func(bc BuildContext, c rendering.BoxConstraints) Widget {
		orientation := OrientationPortrait
		if c.MaxWidth > c.MaxHeight {
			orientation = OrientationLandscape
		}
		return w.builder(bc, orientation)
	})
}
