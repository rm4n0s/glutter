package rendering

import (
	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
)

// ViewConfiguration is the layout constraints and the pixel ratio of a view.
//
// Flutter: rendering/view.dart ViewConfiguration
type ViewConfiguration struct {
	// LogicalConstraints are the constraints of the root, in logical pixels.
	//
	// Flutter: rendering/view.dart ViewConfiguration.logicalConstraints
	LogicalConstraints BoxConstraints
	// PhysicalConstraints are the constraints of the view, in physical pixels.
	//
	// Flutter: rendering/view.dart ViewConfiguration.physicalConstraints
	PhysicalConstraints BoxConstraints
	// DevicePixelRatio is the number of physical pixels per logical pixel.
	//
	// Flutter: rendering/view.dart ViewConfiguration.devicePixelRatio
	DevicePixelRatio float64
}

// ViewConfigurationOpts are the optional parameters of NewViewConfiguration.
//
// Flutter: rendering/view.dart ViewConfiguration.new
type ViewConfigurationOpts struct {
	// PhysicalConstraints default to tight constraints of size zero.
	PhysicalConstraints foundation.Opt[BoxConstraints]
	// LogicalConstraints default to tight constraints of size zero.
	LogicalConstraints foundation.Opt[BoxConstraints]
	// DevicePixelRatio defaults to 1.
	DevicePixelRatio foundation.Opt[float64]
}

// NewViewConfiguration creates a view configuration.
//
// Flutter: rendering/view.dart ViewConfiguration.new
func NewViewConfiguration(opts ...ViewConfigurationOpts) ViewConfiguration {
	o := firstOpts(opts)
	return ViewConfiguration{
		PhysicalConstraints: o.PhysicalConstraints.OrElse(BoxConstraints{}),
		LogicalConstraints:  o.LogicalConstraints.OrElse(BoxConstraints{}),
		DevicePixelRatio:    o.DevicePixelRatio.OrElse(1),
	}
}

// NewViewConfigurationFromView creates the configuration for a view from its
// physical constraints and device pixel ratio.
//
// Flutter: rendering/view.dart ViewConfiguration.fromView
func NewViewConfigurationFromView(view *ui.View) ViewConfiguration {
	physical := NewBoxConstraintsFromViewConstraints(view.PhysicalConstraints())
	dpr := view.DevicePixelRatio()
	return ViewConfiguration{PhysicalConstraints: physical, LogicalConstraints: physical.Div(dpr), DevicePixelRatio: dpr}
}

// ToMatrix is the matrix from logical to physical pixels: a scale by the device
// pixel ratio.
//
// Flutter: rendering/view.dart ViewConfiguration.toMatrix
func (c ViewConfiguration) ToMatrix() *ui.Matrix4 {
	return ui.NewMatrix4Diagonal3Values(float32(c.DevicePixelRatio), float32(c.DevicePixelRatio), 1)
}

// ShouldUpdateMatrix reports whether the matrix differs from that of the old
// configuration.
//
// Flutter: rendering/view.dart ViewConfiguration.shouldUpdateMatrix
func (c ViewConfiguration) ShouldUpdateMatrix(old ViewConfiguration) bool {
	return old.DevicePixelRatio != c.DevicePixelRatio
}

// ToPhysicalSize converts a size in logical pixels to the physical size,
// constrained to the physical constraints.
//
// Flutter: rendering/view.dart ViewConfiguration.toPhysicalSize
func (c ViewConfiguration) ToPhysicalSize(logicalSize ui.Size) ui.Size {
	return c.PhysicalConstraints.Constrain(logicalSize.Mul(c.DevicePixelRatio))
}

// String describes the configuration: the logical constraints and the pixel
// ratio.
//
// Flutter: rendering/view.dart ViewConfiguration.toString
func (c ViewConfiguration) String() string {
	return c.LogicalConstraints.String() + " at " + foundation.DebugFormatDouble(foundation.Some(c.DevicePixelRatio)) + "x"
}

// RenderView is the root of the render tree of a view: it is a repaint boundary,
// lays out its child with the view's constraints, and turns the layer tree into a
// scene.
//
// Flutter: rendering/view.dart RenderView
type RenderView struct {
	RenderObjectBase
	RenderObjectWithChild[Box]

	size          ui.Size
	configuration foundation.Opt[ViewConfiguration]
	view          *ui.View
	rootTransform *ui.Matrix4

	// AutomaticSystemUIAdjustment says whether the view updates the system UI
	// (the status bar and navigation bar) from the SystemUiOverlayStyle
	// annotations of the layer tree.
	//
	// Flutter: rendering/view.dart RenderView.automaticSystemUiAdjustment
	AutomaticSystemUIAdjustment bool
}

// RenderViewOpts are the optional parameters of NewRenderView.
//
// Flutter: rendering/view.dart RenderView.new
type RenderViewOpts struct {
	// Child is the box that fills the view.
	Child Box
	// Configuration is the configuration; it must be set before the view lays
	// out.
	Configuration foundation.Opt[ViewConfiguration]
}

// NewRenderView creates the root render object for the view.
//
// Flutter: rendering/view.dart RenderView.new
func NewRenderView(view *ui.View, opts ...RenderViewOpts) *RenderView {
	o := firstOpts(opts)
	r := &RenderView{view: view, AutomaticSystemUIAdjustment: true}
	InitRenderObject(&r.RenderObjectBase, r)
	InitRenderObjectWithChild(&r.RenderObjectWithChild, r)
	if c, ok := o.Configuration.Get(); ok {
		r.SetConfiguration(c)
	}
	r.SetChild(o.Child)
	return r
}

// ToStringShort describes the object.
func (r *RenderView) ToStringShort() string { return r.RenderObjectBase.ToStringShort() }

// Size is the size of the view in logical pixels, set by layout.
//
// Flutter: rendering/view.dart RenderView.size
func (r *RenderView) Size() ui.Size { return r.size }

// Configuration is the configuration of the view. It panics with
// RenderViewNoConfiguration if there is none.
//
// Flutter: rendering/view.dart RenderView.configuration
func (r *RenderView) Configuration() ViewConfiguration {
	c, ok := r.configuration.Get()
	if !ok {
		panic(errors.New("RenderViewNoConfiguration", "RenderView has not been given a configuration yet"))
	}
	return c
}

// SetConfiguration changes the configuration. If the matrix changes it replaces
// the root layer; it marks the view as needing layout.
//
// Flutter: rendering/view.dart RenderView.configuration
func (r *RenderView) SetConfiguration(value ViewConfiguration) {
	if cur, ok := r.configuration.Get(); ok && cur == value {
		return
	}
	old, hadOld := r.configuration.Get()
	r.configuration = foundation.Some(value)
	if r.rootTransform == nil {
		return
	}
	if !hadOld || value.ShouldUpdateMatrix(old) {
		r.ReplaceRootLayer(r.updateMatricesAndCreateNewRootLayer())
	}
	r.MarkNeedsLayout()
}

// HasConfiguration reports whether the view has a configuration.
//
// Flutter: rendering/view.dart RenderView.hasConfiguration
func (r *RenderView) HasConfiguration() bool { return r.configuration.IsSome() }

// Constraints are the logical constraints of the configuration.
//
// Flutter: rendering/view.dart RenderView.constraints
func (r *RenderView) Constraints() Constraints { return r.BoxConstraints() }

// BoxConstraints are the logical constraints of the configuration.
func (r *RenderView) BoxConstraints() BoxConstraints {
	if !r.HasConfiguration() {
		panic(errors.New("RenderViewNoConfiguration",
			"constraints are not available because RenderView has not been given a configuration yet"))
	}
	return r.Configuration().LogicalConstraints
}

// FlutterView is the view that the render view renders.
//
// Flutter: rendering/view.dart RenderView.flutterView
func (r *RenderView) FlutterView() *ui.View { return r.view }

// PrepareInitialFrame schedules the first layout and paint. The view must be
// attached to a pipeline owner and have a configuration.
//
// Flutter: rendering/view.dart RenderView.prepareInitialFrame
func (r *RenderView) PrepareInitialFrame() {
	if foundation.DebugMode && (r.owner == nil || r.rootTransform != nil || !r.HasConfiguration()) {
		panic(errors.New("PrepareInitialFrameInvalid",
			"PrepareInitialFrame needs the view attached to a pipeline owner, with a configuration, and can only be called once"))
	}
	r.ScheduleInitialLayout()
	r.ScheduleInitialPaint(r.updateMatricesAndCreateNewRootLayer())
}

func (r *RenderView) updateMatricesAndCreateNewRootLayer() *TransformLayer {
	r.rootTransform = r.Configuration().ToMatrix()
	rootLayer := NewTransformLayer(TransformLayerOpts{Transform: r.rootTransform})
	rootLayer.Attach(r)
	return rootLayer
}

// DebugAssertDoesMeetConstraints is never called.
//
// Flutter: rendering/view.dart RenderView.debugAssertDoesMeetConstraints
func (r *RenderView) DebugAssertDoesMeetConstraints() {}

// PerformResize is never called; a RenderView does not size itself from its
// parent.
//
// Flutter: rendering/view.dart RenderView.performResize
func (r *RenderView) PerformResize() {
	panic(errors.New("RenderViewResize", "a RenderView is never resized"))
}

// PerformLayout lays out the child with the view's constraints and takes its size
// (or the smallest size, for tight constraints).
//
// Flutter: rendering/view.dart RenderView.performLayout
func (r *RenderView) PerformLayout() {
	if foundation.DebugMode && r.rootTransform == nil {
		panic(errors.New("RenderViewNotPrepared", "the RenderView was laid out before PrepareInitialFrame"))
	}
	c := r.BoxConstraints()
	sizedByChild := !c.IsTight()
	child := r.Child()
	if !isNil(child) {
		child.Layout(c, LayoutOpts{ParentUsesSize: sizedByChild})
	}
	if sizedByChild && !isNil(child) {
		r.size = child.Size()
	} else {
		r.size = c.Smallest()
	}
}

// HitTest hit tests the child at the position, then adds the view itself.
//
// Flutter: rendering/view.dart RenderView.hitTest
func (r *RenderView) HitTest(result *gestures.HitTestResult, position ui.Offset) bool {
	if child := r.Child(); !isNil(child) {
		child.HitTest(NewBoxHitTestResultWrap(result), position)
	}
	result.Add(gestures.NewHitTestEntry(r))
	return true
}

// IsRepaintBoundary is true.
//
// Flutter: rendering/view.dart RenderView.isRepaintBoundary
func (r *RenderView) IsRepaintBoundary() bool { return true }

// Paint paints the child, then the debug paint callbacks.
//
// Flutter: rendering/view.dart RenderView.paint
func (r *RenderView) Paint(context *PaintingContext, offset ui.Offset) {
	if child := r.Child(); !isNil(child) {
		context.PaintChild(child, offset)
	}
	if foundation.DebugMode {
		o := r.DebugOptions()
		for _, cb := range append([]DebugPaintCallback(nil), o.paintCallbacks...) {
			cb(context, offset, r)
		}
	}
}

// ApplyPaintTransform applies the root transform and the child's offset.
//
// Flutter: rendering/view.dart RenderView.applyPaintTransform
func (r *RenderView) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	if foundation.DebugMode && r.rootTransform == nil {
		panic(errors.New("RenderViewNotPrepared", "ApplyPaintTransform was called before PrepareInitialFrame"))
	}
	transform.MultiplyBy(r.rootTransform)
	r.RenderObjectBase.ApplyPaintTransform(child, transform)
}

// CompositeFrame turns the layer tree into a scene and sends it to the view. It
// returns the error of the renderer, such as a lost GPU context.
//
// Flutter: rendering/view.dart RenderView.compositeFrame
func (r *RenderView) CompositeFrame() error {
	if foundation.DebugMode && (!r.HasConfiguration() || r.rootTransform == nil || isNil(r.layerHandle.Layer())) {
		panic(errors.New("CompositeFrameNotPrepared",
			"the RenderView needs a configuration and PrepareInitialFrame before CompositeFrame"))
	}
	scene := r.layerHandle.Layer().AsContainerLayer().BuildScene(ui.NewSceneBuilder())
	// TODO(step 10.x): update the system chrome from the SystemUiOverlayStyle
	// annotations when AutomaticSystemUIAdjustment is set.
	err := r.view.Render(scene)
	scene.Dispose()
	if foundation.DebugMode {
		o := r.DebugOptions()
		if o.RepaintRainbowEnabled || o.RepaintTextRainbowEnabled {
			c := o.repaintColor()
			c.Hue = float64(int((c.Hue+2)*1000)%360000) / 1000
			o.CurrentRepaintColor = c
		}
	}
	return err
}

// PaintBounds are the view's size in physical pixels, at the origin.
//
// Flutter: rendering/view.dart RenderView.paintBounds
func (r *RenderView) PaintBounds() ui.Rect {
	return ui.OffsetZero.And(r.size.Mul(r.Configuration().DevicePixelRatio))
}

// SemanticBounds are the view's bounds in physical pixels.
//
// Flutter: rendering/view.dart RenderView.semanticBounds
func (r *RenderView) SemanticBounds() ui.Rect {
	if foundation.DebugMode && r.rootTransform == nil {
		panic(errors.New("RenderViewNotPrepared", "SemanticBounds was read before PrepareInitialFrame"))
	}
	return r.rootTransform.TransformRect(ui.OffsetZero.And(r.size))
}

// DebugFillProperties adds the view size, the pixel ratio and the configuration.
//
// Flutter: rendering/view.dart RenderView.debugFillProperties
func (r *RenderView) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	r.RenderObjectBase.DebugFillProperties(p)
	p.Add(foundation.NewDiagnosticsProperty("view size", r.view.PhysicalSize(), foundation.DiagnosticsPropertyOpts{Tooltip: "in physical pixels"}))
	p.Add(foundation.NewDoubleProperty("device pixel ratio", foundation.Some(r.view.DevicePixelRatio()),
		foundation.NumberPropertyOpts{Tooltip: "physical pixels per logical pixel"}))
	p.Add(foundation.NewDiagnosticsPropertyOpt("configuration", r.configuration, foundation.DiagnosticsPropertyOpts{Tooltip: "in logical pixels"}))
}
