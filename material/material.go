package material

import (
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// RectCallback is the signature of a function that returns a Rect.
//
// Flutter: material/material.dart RectCallback
type RectCallback func() ui.Rect

// MaterialType is the kind of material: canvas, card, circle, button or
// transparency. It selects the shape, the default color and the elevation
// behavior of Material.
//
// Flutter: material/material.dart MaterialType
type MaterialType int

const (
	// MaterialTypeUnspecified means the default of the widget: canvas.
	MaterialTypeUnspecified MaterialType = iota
	// MaterialTypeCanvas is a rectangle with the canvas color of the theme.
	//
	// Flutter: material/material.dart MaterialType.canvas
	MaterialTypeCanvas
	// MaterialTypeCard is a rectangle with the card color of the theme and slightly rounded corners.
	//
	// Flutter: material/material.dart MaterialType.card
	MaterialTypeCard
	// MaterialTypeCircle is a circle. It has no default color.
	//
	// Flutter: material/material.dart MaterialType.circle
	MaterialTypeCircle
	// MaterialTypeButton is a rectangle with slightly rounded corners and no default color.
	//
	// Flutter: material/material.dart MaterialType.button
	MaterialTypeButton
	// MaterialTypeTransparency is a transparent piece of material that draws ink splashes and highlights.
	//
	// Flutter: material/material.dart MaterialType.transparency
	MaterialTypeTransparency
)

// Name returns the name of the type, as Dart's Enum.name.
func (t MaterialType) Name() string {
	switch t {
	case MaterialTypeCard:
		return "card"
	case MaterialTypeCircle:
		return "circle"
	case MaterialTypeButton:
		return "button"
	case MaterialTypeTransparency:
		return "transparency"
	}
	return "canvas"
}

// String returns "MaterialType.canvas"...
func (t MaterialType) String() string { return "MaterialType." + t.Name() }

// effective is the type that the zero value stands for.
func (t MaterialType) effective() MaterialType {
	if t == MaterialTypeUnspecified {
		return MaterialTypeCanvas
	}
	return t
}

// MaterialEdges maps a type of material to the border radius of its edges; a
// type that is missing has no rounded edges.
//
// Flutter: material/material.dart kMaterialEdges
var MaterialEdges = map[MaterialType]painting.BorderRadius{
	MaterialTypeCard:   painting.NewBorderRadiusAll(ui.NewRadiusCircular(2.0)),
	MaterialTypeButton: painting.NewBorderRadiusAll(ui.NewRadiusCircular(2.0)),
}

// MaterialInkController is an interface for widgets that paint ink on a Material
// below them: it is what InkFeature draws on.
//
// Flutter: material/material.dart MaterialInkController
type MaterialInkController interface {
	// Color is the color of the material, or none.
	//
	// Flutter: material/material.dart MaterialInkController.color
	Color() foundation.Opt[ui.Color]
	// Vsync is the ticker provider used by the controller; ink features that
	// animate use it to make their animation controllers.
	//
	// Flutter: material/material.dart MaterialInkController.vsync
	Vsync() scheduler.TickerProvider
	// AddInkFeature adds an ink feature that renders on the material.
	//
	// Flutter: material/material.dart MaterialInkController.addInkFeature
	AddInkFeature(feature InkFeature)
	// MarkNeedsPaint informs the controller that one of its ink features needs
	// to repaint.
	//
	// Flutter: material/material.dart MaterialInkController.markNeedsPaint
	MarkNeedsPaint()
}

// MaterialOpts are the optional parameters of NewMaterial.
//
// Flutter: material/material.dart Material.new
type MaterialOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Type is the kind of material; the default is canvas.
	Type MaterialType
	// Elevation is the z-coordinate at which to place this material relative to
	// its parent; it must not be negative.
	Elevation float64
	// Color is the background color; the default is the canvas or card color of
	// the theme for those types, and none for the others.
	Color foundation.Opt[ui.Color]
	// ShadowColor is the color to paint the shadow below the material; the
	// default is the shadow color of the color scheme (Material 3) or of the
	// theme.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the color of the surface tint overlay applied to the
	// material color to indicate elevation (Material 3).
	SurfaceTintColor foundation.Opt[ui.Color]
	// TextStyle is the typographic style to use for text within this material;
	// the default is the body medium style of the text theme.
	TextStyle *painting.TextStyle
	// BorderRadius, if set, clips the material to a rounded rectangle; it must
	// not be given with Shape or the circle type.
	BorderRadius painting.BorderRadiusGeometry
	// Shape defines the material's shape as well as its shadow; it must not be
	// given with BorderRadius or the circle type.
	Shape painting.ShapeBorder
	// NoBorderOnForeground says to paint the border of the shape behind the
	// child instead of on top of it.
	NoBorderOnForeground bool
	// ClipBehavior is whether the content is clipped and how; the default is
	// none.
	ClipBehavior ui.Clip
	// AnimationDuration is the duration over which to animate the changes of the
	// shape and the elevation; the default is ThemeChangeDuration.
	AnimationDuration time.Duration
	// AnimateColor is whether the color animates when it changes (only for the
	// canvas type with the default shape).
	AnimateColor bool
}

// Material is a piece of material: a widget that is the surface that ink
// features paint on, with an elevation and a shape. It is the origin of the
// Material Design layout model.
//
// Flutter: material/material.dart Material
type Material struct {
	widgets.StatefulWidget
	child widgets.Widget
	o     MaterialOpts
}

// NewMaterial creates a piece of material. In debug builds it panics with the
// tag MaterialElevationNegative when the elevation is negative, with
// MaterialShapeAndBorderRadius when both a shape and a border radius are given,
// and with MaterialCircleWithShape when the type is circle and a border radius
// or a shape is given.
//
// Flutter: material/material.dart Material.new
func NewMaterial(child widgets.Widget, opts ...MaterialOpts) *Material {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if !(o.Elevation >= 0.0) {
			panic(errors.New("MaterialElevationNegative", "the elevation of a Material must not be negative", "elevation", o.Elevation))
		}
		if o.Shape != nil && o.BorderRadius != nil {
			panic(errors.New("MaterialShapeAndBorderRadius", "a Material cannot have both a shape and a border radius"))
		}
		if o.Type == MaterialTypeCircle && (o.BorderRadius != nil || o.Shape != nil) {
			panic(errors.New("MaterialCircleWithShape", "a Material of type circle cannot have a shape or a border radius"))
		}
	}
	if o.AnimationDuration == 0 {
		o.AnimationDuration = ThemeChangeDuration
	}
	o.Type = o.Type.effective()
	return &Material{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child, o}
}

// Child is the widget below this widget in the tree.
//
// Flutter: material/material.dart Material.child
func (w *Material) Child() widgets.Widget { return w.child }

// Type is the kind of material.
//
// Flutter: material/material.dart Material.type
func (w *Material) Type() MaterialType { return w.o.Type }

// Elevation is the z-coordinate at which to place this material.
//
// Flutter: material/material.dart Material.elevation
func (w *Material) Elevation() float64 { return w.o.Elevation }

// Color is the background color, or none.
//
// Flutter: material/material.dart Material.color
func (w *Material) Color() foundation.Opt[ui.Color] { return w.o.Color }

// ShadowColor is the color of the shadow, or none.
//
// Flutter: material/material.dart Material.shadowColor
func (w *Material) ShadowColor() foundation.Opt[ui.Color] { return w.o.ShadowColor }

// SurfaceTintColor is the color of the surface tint overlay, or none.
//
// Flutter: material/material.dart Material.surfaceTintColor
func (w *Material) SurfaceTintColor() foundation.Opt[ui.Color] { return w.o.SurfaceTintColor }

// TextStyle is the style of the text of the material, or nil for the default.
//
// Flutter: material/material.dart Material.textStyle
func (w *Material) TextStyle() *painting.TextStyle { return w.o.TextStyle }

// Shape is the shape of the material, or nil.
//
// Flutter: material/material.dart Material.shape
func (w *Material) Shape() painting.ShapeBorder { return w.o.Shape }

// BorderOnForeground is whether the border of the shape is painted on top of the child.
//
// Flutter: material/material.dart Material.borderOnForeground
func (w *Material) BorderOnForeground() bool { return !w.o.NoBorderOnForeground }

// ClipBehavior is whether the content is clipped and how.
//
// Flutter: material/material.dart Material.clipBehavior
func (w *Material) ClipBehavior() ui.Clip { return w.o.ClipBehavior }

// AnimationDuration is the duration of the animation of the shape and the elevation.
//
// Flutter: material/material.dart Material.animationDuration
func (w *Material) AnimationDuration() time.Duration { return w.o.AnimationDuration }

// BorderRadius is the border radius of the material's rounded rectangle, or nil.
//
// Flutter: material/material.dart Material.borderRadius
func (w *Material) BorderRadius() painting.BorderRadiusGeometry { return w.o.BorderRadius }

// MaterialDefaultSplashRadius is the default radius of an ink splash in logical pixels.
//
// Flutter: material/material.dart Material.defaultSplashRadius
const MaterialDefaultSplashRadius = 35.0

// MaterialMaybeOf returns the controller of the closest Material ancestor of
// bc that is inside the closest LookupBoundary, if any.
//
// Flutter: material/material.dart Material.maybeOf
func MaterialMaybeOf(bc widgets.BuildContext) (MaterialInkController, bool) {
	r, ok := widgets.LookupBoundaryFindAncestorRenderObjectOfType[*renderInkFeatures](bc)
	if !ok {
		return nil, false
	}
	return r, true
}

// MaterialOf returns the controller of the closest Material ancestor of bc. In
// debug builds it panics with the tag MaterialNotFound when there is none, and
// with MaterialHiddenByLookupBoundary when there is one that a LookupBoundary
// hides.
//
// Flutter: material/material.dart Material.of
func MaterialOf(bc widgets.BuildContext) MaterialInkController {
	controller, ok := MaterialMaybeOf(bc)
	if !ok && foundation.DebugMode {
		if widgets.LookupBoundaryDebugIsHidingAncestorRenderObjectOfType[*renderInkFeatures](bc) {
			panic(errors.New("MaterialHiddenByLookupBoundary", "Material.of() was called with a context that does not have access to a Material widget.\n"+
				"The context provided to Material.of() does have a Material widget ancestor, but it is hidden by a LookupBoundary. "+
				"This can happen because you are using a widget that looks for a Material ancestor, but no such ancestor exists within the closest LookupBoundary.",
				"context", foundation.DescribeIdentity(bc)))
		}
		panic(errors.New("MaterialNotFound", "Material.of() was called with a context that does not contain a Material widget.\n"+
			"No Material widget ancestor could be found starting from the context that was passed to Material.of(). "+
			"This can happen because you are using a widget that looks for a Material ancestor, but no such ancestor exists.",
			"context", foundation.DescribeIdentity(bc)))
	}
	return controller
}

// CreateState creates the state.
//
// Flutter: material/material.dart Material.createState
func (w *Material) CreateState() widgets.State {
	return &materialState{inkFeatureRenderer: widgets.NewGlobalKey[widgets.State](widgets.GlobalKeyOpts{DebugLabel: "ink renderer"})}
}

// DebugFillProperties adds the type, the elevation, the colors, the text style
// and the shape.
//
// Flutter: material/material.dart Material.debugFillProperties
func (w *Material) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	o := w.o
	p.Add(foundation.NewEnumProperty("type", foundation.Some(o.Type)))
	p.Add(foundation.NewDoubleProperty("elevation", foundation.Some(o.Elevation), foundation.NumberPropertyOpts{DefaultValue: 0.0}))
	p.Add(painting.NewColorProperty("color", o.Color, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	p.Add(painting.NewColorProperty("shadowColor", o.ShadowColor, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	p.Add(painting.NewColorProperty("surfaceTintColor", o.SurfaceTintColor, foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	if o.TextStyle != nil {
		o.TextStyle.DebugFillPropertiesWithPrefix(p, "textStyle.")
	}
	p.Add(foundation.NewDiagnosticsPropertyOpt("shape", optShape(o.Shape), foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	p.Add(foundation.NewDiagnosticsProperty("borderOnForeground", !o.NoBorderOnForeground, foundation.DiagnosticsPropertyOpts{DefaultValue: true}))
	p.Add(foundation.NewDiagnosticsPropertyOpt("borderRadius", optBorderRadius(o.BorderRadius), foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
}

func optShape(s painting.ShapeBorder) foundation.Opt[painting.ShapeBorder] {
	if s == nil {
		return foundation.None[painting.ShapeBorder]()
	}
	return foundation.Some(s)
}

func optBorderRadius(r painting.BorderRadiusGeometry) foundation.Opt[painting.BorderRadiusGeometry] {
	if r == nil {
		return foundation.None[painting.BorderRadiusGeometry]()
	}
	return foundation.Some(r)
}

type materialState struct {
	widgets.StateOf[*Material]
	widgets.TickerProvider
	inkFeatureRenderer *widgets.GlobalKey[widgets.State]
}

func (s *materialState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	theme := ThemeOf(bc)
	var backgroundColor foundation.Opt[ui.Color]
	switch {
	case w.o.Color.IsSome():
		backgroundColor = w.o.Color
	case w.o.Type == MaterialTypeCanvas:
		backgroundColor = foundation.Some(theme.CanvasColor)
	case w.o.Type == MaterialTypeCard:
		backgroundColor = foundation.Some(theme.CardColor)
	}
	modelShadowColor := w.o.ShadowColor.OrElse(theme.ShadowColor)
	if !w.o.ShadowColor.IsSome() && theme.UseMaterial3 {
		modelShadowColor = theme.ColorScheme.Shadow()
	}
	if foundation.DebugMode && !backgroundColor.IsSome() && w.o.Type != MaterialTypeTransparency {
		panic(errors.New("MaterialColorMissing", "If Material type is not MaterialType.transparency, a color must either be passed in through the `color` property, "+
			"or be defined in the theme (ex. canvasColor != null if type is set to MaterialType.canvas)", "type", w.o.Type.Name()))
	}
	contents := w.child
	if contents != nil {
		style := w.o.TextStyle
		if style == nil {
			style = ThemeOf(bc).TextTheme.BodyMedium
		}
		contents = widgets.NewAnimatedDefaultTextStyle(style, w.o.AnimationDuration, contents)
	}
	contents = widgets.NewNotificationListener(func(widgets.LayoutChangedNotification) bool {
		if ctx := s.inkFeatureRenderer.CurrentContext(); ctx != nil {
			ctx.FindRenderObject().(*renderInkFeatures).didChangeLayout()
		}
		return false
	}, widgets.Widget(&inkFeatures{
		SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(contents, widgets.WidgetOpts{Key: s.inkFeatureRenderer}),
		color:                         backgroundColor,
		vsync:                         s,
		absorbHitTest:                 w.o.Type != MaterialTypeTransparency,
	}))

	shape := w.o.Shape
	if w.o.BorderRadius != nil {
		shape = painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: w.o.BorderRadius})
	}
	// PhysicalModel has a temporary workaround for a performance issue that
	// speeds up rectangular non transparent material (the workaround is to skip
	// the call to ui.Canvas.saveLayer if the border radius is 0). Until the
	// saveLayer performance issue is resolved, we're keeping this special case
	// here for canvas material type that is using the default shape (rectangle).
	// We could go down this fast path for explicitly specified rectangles (e.g
	// shape RoundedRectangleBorder with radius 0, but we choose not to as we want
	// the change from the fast-path to the slow-path to be noticeable in the
	// construction site of Material.
	if w.o.Type == MaterialTypeCanvas && shape == nil {
		var color ui.Color
		if theme.UseMaterial3 {
			color = ElevationOverlayApplySurfaceTint(backgroundColor.Value(), optColorPtr(w.o.SurfaceTintColor), w.o.Elevation)
		} else {
			color = ElevationOverlayApplyOverlay(bc, backgroundColor.Value(), w.o.Elevation)
		}
		return widgets.NewAnimatedPhysicalModel(color, modelShadowColor, w.o.AnimationDuration, contents, widgets.AnimatedPhysicalModelOpts{
			Curve:        animation.Curves.FastOutSlowIn,
			ClipBehavior: w.o.ClipBehavior,
			Elevation:    w.o.Elevation,
			AnimateColor: foundation.Some(w.o.AnimateColor),
		})
	}
	if shape == nil {
		switch w.o.Type {
		case MaterialTypeCircle:
			shape = painting.NewCircleBorder()
		case MaterialTypeCard, MaterialTypeButton:
			shape = painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusCircular(2.0)})
		default:
			shape = painting.NewRoundedRectangleBorder()
		}
	}
	if w.o.Type == MaterialTypeTransparency {
		return widgets.NewClipPath(shapeBorderPaint(shape, contents, true, bc), widgets.ClipPathOpts{
			Clipper:      rendering.NewShapeBorderClipper(shape, rendering.ShapeBorderClipperOpts{TextDirection: maybeDirection(bc)}),
			ClipBehavior: w.o.ClipBehavior,
		})
	}
	return &materialInterior{
		ImplicitlyAnimatedWidget: widgets.NewImplicitlyAnimatedWidgetBase(w.o.AnimationDuration, widgets.ImplicitlyAnimatedWidgetOpts{Curve: animation.Curves.FastOutSlowIn}),
		child:                    contents, shape: shape, borderOnForeground: !w.o.NoBorderOnForeground, clipBehavior: w.o.ClipBehavior,
		elevation: w.o.Elevation, color: backgroundColor.Value(), shadowColor: modelShadowColor, surfaceTintColor: w.o.SurfaceTintColor,
	}
}

// maybeDirection is Directionality.maybeOf, with the unspecified direction for
// null.
func maybeDirection(bc widgets.BuildContext) ui.TextDirection {
	d, _ := widgets.MaybeDirectionalityOf(bc)
	return d
}

func optColorPtr(c foundation.Opt[ui.Color]) *ui.Color {
	if v, ok := c.Get(); ok {
		return &v
	}
	return nil
}

// inkFeatures is Flutter's _InkFeatures: the render object widget that holds
// the ink features of a Material.
//
// Flutter: material/material.dart _InkFeatures
type inkFeatures struct {
	widgets.SingleChildRenderObjectWidget
	color         foundation.Opt[ui.Color]
	vsync         scheduler.TickerProvider
	absorbHitTest bool
}

func (w *inkFeatures) CreateRenderObject(widgets.BuildContext) rendering.RenderObject {
	return newRenderInkFeatures(w.vsync, w.absorbHitTest, w.color)
}

func (w *inkFeatures) UpdateRenderObject(_ widgets.BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderInkFeatures)
	r.color = w.color
	r.absorbHitTest = w.absorbHitTest
}

// renderInkFeatures is Flutter's _RenderInkFeatures: the render object that
// paints the ink features of a Material above its own background and below its
// child. It is the MaterialInkController.
//
// Flutter: material/material.dart _RenderInkFeatures
type renderInkFeatures struct {
	rendering.RenderProxyBox
	vsync         scheduler.TickerProvider
	color         foundation.Opt[ui.Color]
	absorbHitTest bool
	inkFeatures   []InkFeature
}

func newRenderInkFeatures(vsync scheduler.TickerProvider, absorbHitTest bool, color foundation.Opt[ui.Color]) *renderInkFeatures {
	r := &renderInkFeatures{vsync: vsync, absorbHitTest: absorbHitTest, color: color}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

// Color is the color of the material; the painting of it is done by the
// Material's AnimatedPhysicalModel or PhysicalShape.
func (r *renderInkFeatures) Color() foundation.Opt[ui.Color] { return r.color }

// Vsync is the ticker provider of the state of the Material.
func (r *renderInkFeatures) Vsync() scheduler.TickerProvider { return r.vsync }

// DebugInkFeatures lists the ink features, in debug builds.
//
// Flutter: material/material.dart _RenderInkFeatures.debugInkFeatures
func (r *renderInkFeatures) DebugInkFeatures() []InkFeature {
	if foundation.DebugMode {
		return r.inkFeatures
	}
	return nil
}

// AddInkFeature adds the feature and repaints.
func (r *renderInkFeatures) AddInkFeature(feature InkFeature) {
	base := feature.AsInkFeature()
	if foundation.DebugMode {
		if base.debugDisposed {
			panic(errors.New("InkFeatureDisposed", "an ink feature that was disposed cannot be added to a Material"))
		}
		if base.controller != r {
			panic(errors.New("InkFeatureWrongController", "an ink feature was added to a controller other than its own"))
		}
		if slices.Contains(r.inkFeatures, feature) {
			panic(errors.New("InkFeatureAddedTwice", "an ink feature was added to its controller twice"))
		}
	}
	r.inkFeatures = append(r.inkFeatures, feature)
	r.MarkNeedsPaint()
}

func (r *renderInkFeatures) removeFeature(feature InkFeature) {
	if i := slices.Index(r.inkFeatures, feature); i >= 0 {
		r.inkFeatures = slices.Delete(r.inkFeatures, i, i+1)
	}
	r.MarkNeedsPaint()
}

func (r *renderInkFeatures) didChangeLayout() {
	if len(r.inkFeatures) > 0 {
		r.MarkNeedsPaint()
	}
}

// HitTestSelf absorbs the hits unless the material is transparent.
func (r *renderInkFeatures) HitTestSelf(ui.Offset) bool { return r.absorbHitTest }

// Paint paints the ink features, clipped to the box, then the child.
func (r *renderInkFeatures) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	features := r.inkFeatures
	if len(features) > 0 {
		canvas := context.Canvas()
		canvas.Save()
		canvas.Translate(offset.Dx, offset.Dy)
		canvas.ClipRect(ui.NewRectFromLTWH(0, 0, r.Size().Width, r.Size().Height))
		for _, f := range features {
			f.AsInkFeature().paint(canvas)
		}
		canvas.Restore()
	}
	r.RenderProxyBox.Paint(context, offset)
}

// InkFeature is a visual reaction on a Material that can be rendered with a
// Canvas: a splash, a highlight, or a decoration. Embed InkFeatureBase and
// implement PaintFeature; the feature gets painted until it is disposed.
//
// Flutter: material/material.dart InkFeature
type InkFeature interface {
	// AsInkFeature returns the base of the feature.
	AsInkFeature() *InkFeatureBase
	// Dispose frees the resources of the feature and removes it from the
	// controller. A feature that owns more than the base, like the animations of a
	// splash, overrides it.
	//
	// Flutter: material/material.dart InkFeature.dispose
	Dispose()
	// PaintFeature paints the feature onto the canvas. The transform takes a
	// point in the coordinate space of the reference box to the coordinate space
	// of the material, as the canvas of the material has the origin at its
	// top left corner.
	//
	// Flutter: material/material.dart InkFeature.paintFeature
	PaintFeature(canvas *ui.Canvas, transform *ui.Matrix4)
}

// InkFeatureBase is the embedded base of an InkFeature.
//
// Flutter: material/material.dart InkFeature
type InkFeatureBase struct {
	self          InkFeature
	controller    *renderInkFeatures
	referenceBox  rendering.Box
	onRemoved     func()
	debugDisposed bool
}

// InkFeatureOpts are the parameters of InitInkFeature.
//
// Flutter: material/material.dart InkFeature.new
type InkFeatureOpts struct {
	// OnRemoved is called when the feature is disposed.
	OnRemoved func()
}

// InitInkFeature initializes the base of the feature self. The controller must
// be one that MaterialOf returned, and referenceBox is the render box whose
// coordinate system the feature is positioned in.
//
// Flutter: material/material.dart InkFeature.new
func InitInkFeature(f *InkFeatureBase, self InkFeature, controller MaterialInkController, referenceBox rendering.Box, opts ...InkFeatureOpts) {
	f.self = self
	f.controller = controller.(*renderInkFeatures)
	f.referenceBox = referenceBox
	f.onRemoved = firstOpts(opts).OnRemoved
}

// AsInkFeature returns the base.
func (f *InkFeatureBase) AsInkFeature() *InkFeatureBase { return f }

// Controller is the render object that is told to paint the feature.
//
// Flutter: material/material.dart InkFeature.controller
func (f *InkFeatureBase) Controller() MaterialInkController { return f.controller }

// ReferenceBox is the render box whose coordinate system the feature is
// positioned in.
//
// Flutter: material/material.dart InkFeature.referenceBox
func (f *InkFeatureBase) ReferenceBox() rendering.Box { return f.referenceBox }

// Dispose free up the resources associated with this feature and removes it
// from the controller. In debug builds disposing twice panics with the tag
// InkFeatureDisposed.
//
// Flutter: material/material.dart InkFeature.dispose
func (f *InkFeatureBase) Dispose() {
	if foundation.DebugMode {
		if f.debugDisposed {
			panic(errors.New("InkFeatureDisposed", "an ink feature was disposed twice"))
		}
		f.debugDisposed = true
	}
	f.controller.removeFeature(f.self)
	if f.onRemoved != nil {
		f.onRemoved()
	}
}

// inkFeaturePaintTransform returns the paint transform that allows
// fromRenderObject to perform paint in toRenderObject's coordinate space. It
// returns nil if either is not in the same render tree, or either of them is in
// an offscreen subtree (see RenderObject.PaintsChild).
//
// Flutter: material/material.dart InkFeature._getPaintTransform
func inkFeaturePaintTransform(fromRenderObject, toRenderObject rendering.RenderObject) *ui.Matrix4 {
	// The paths to fromRenderObject and toRenderObject's common ancestor.
	fromPath := []rendering.RenderObject{fromRenderObject}
	toPath := []rendering.RenderObject{toRenderObject}
	from, to := fromRenderObject, toRenderObject
	for from != to {
		fromDepth, toDepth := from.Depth(), to.Depth()
		if fromDepth >= toDepth {
			fromParent := from.Parent()
			// Return early if the 2 render objects are not in the same render
			// tree, or either of them is offscreen and thus won't get painted.
			if fromParent == nil || !fromParent.PaintsChild(from) {
				return nil
			}
			fromPath = append(fromPath, fromParent)
			from = fromParent
		}
		if fromDepth <= toDepth {
			toParent := to.Parent()
			if toParent == nil || !toParent.PaintsChild(to) {
				return nil
			}
			toPath = append(toPath, toParent)
			to = toParent
		}
	}
	transform := ui.NewMatrix4Identity()
	inverseTransform := ui.NewMatrix4Identity()
	for index := len(toPath) - 1; index > 0; index-- {
		toPath[index].ApplyPaintTransform(toPath[index-1], transform)
	}
	for index := len(fromPath) - 1; index > 0; index-- {
		fromPath[index].ApplyPaintTransform(fromPath[index-1], inverseTransform)
	}
	if det := inverseTransform.Invert(); det != 0 {
		inverseTransform.MultiplyBy(transform)
		return inverseTransform
	}
	return nil
}

func (f *InkFeatureBase) paint(canvas *ui.Canvas) {
	if foundation.DebugMode {
		if !f.referenceBox.Attached() {
			panic(errors.New("InkFeatureDetached", "an ink feature was painted with a reference box that is not attached"))
		}
		if f.debugDisposed {
			panic(errors.New("InkFeatureDisposed", "an ink feature was painted after it was disposed"))
		}
	}
	// Determine the transform that gets our coordinate system to be like theirs.
	if transform := inkFeaturePaintTransform(f.controller, f.referenceBox); transform != nil {
		f.self.PaintFeature(canvas, transform)
	}
}

// NewShapeBorderTween creates a tween between two shapes (either may be nil).
//
// Flutter: material/material.dart ShapeBorderTween
func NewShapeBorderTween(begin, end painting.ShapeBorder) *animation.Tween[painting.ShapeBorder] {
	return animation.NewTweenFunc(begin, end, painting.ShapeBorderLerp).SetName("ShapeBorderTween")
}

// materialInterior is Flutter's _MaterialInterior: the animated shape,
// elevation and colors of a Material that is not the plain canvas.
//
// Flutter: material/material.dart _MaterialInterior
type materialInterior struct {
	widgets.ImplicitlyAnimatedWidget
	child              widgets.Widget
	shape              painting.ShapeBorder
	borderOnForeground bool
	clipBehavior       ui.Clip
	elevation          float64
	color              ui.Color
	shadowColor        ui.Color
	surfaceTintColor   foundation.Opt[ui.Color]
}

func (w *materialInterior) CreateState() widgets.State { return &materialInteriorState{} }

func (w *materialInterior) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewDiagnosticsProperty("shape", w.shape))
	p.Add(foundation.NewDoubleProperty("elevation", foundation.Some(w.elevation)))
	p.Add(painting.NewColorProperty("color", foundation.Some(w.color)))
	p.Add(painting.NewColorProperty("shadowColor", foundation.Some(w.shadowColor)))
}

type materialInteriorState struct {
	widgets.AnimatedWidgetBaseState[*materialInterior]
	elevation        *animation.Tween[float64]
	surfaceTintColor *animation.Tween[foundation.Opt[ui.Color]]
	shadowColor      *animation.Tween[foundation.Opt[ui.Color]]
	border           *animation.Tween[painting.ShapeBorder]
}

func (s *materialInteriorState) ForEachTween(v *widgets.TweenVisitor) {
	w := s.Widget()
	colorTween := func(x foundation.Opt[ui.Color]) *animation.Tween[foundation.Opt[ui.Color]] {
		return animation.NewColorTween(x, x)
	}
	colorEqual := func(a, b foundation.Opt[ui.Color]) bool { return a == b }
	widgets.VisitTween(v, &s.elevation, foundation.Some(w.elevation), func(a, b float64) bool { return a == b },
		func(x float64) *animation.Tween[float64] { return animation.NewTween(x, x) })
	widgets.VisitTween(v, &s.shadowColor, foundation.Some(foundation.Some(w.shadowColor)), colorEqual, colorTween)
	if w.surfaceTintColor.IsSome() {
		widgets.VisitTween(v, &s.surfaceTintColor, foundation.Some(w.surfaceTintColor), colorEqual, colorTween)
	} else {
		s.surfaceTintColor = nil
	}
	widgets.VisitTween(v, &s.border, foundation.Some(w.shape), func(a, b painting.ShapeBorder) bool { return foundation.ValuesEqual(a, b) },
		func(x painting.ShapeBorder) *animation.Tween[painting.ShapeBorder] { return NewShapeBorderTween(x, x) })
}

func (s *materialInteriorState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	shape := s.border.Evaluate(s.Animation())
	elevation := s.elevation.Evaluate(s.Animation())
	var color ui.Color
	if ThemeOf(bc).UseMaterial3 {
		var tint *ui.Color
		if s.surfaceTintColor != nil {
			tint = optColorPtr(s.surfaceTintColor.Evaluate(s.Animation()))
		}
		color = ElevationOverlayApplySurfaceTint(w.color, tint, elevation)
	} else {
		color = ElevationOverlayApplyOverlay(bc, w.color, elevation)
	}
	shadowColor := s.shadowColor.Evaluate(s.Animation()).Value()
	return widgets.NewPhysicalShape(rendering.NewShapeBorderClipper(shape, rendering.ShapeBorderClipperOpts{TextDirection: maybeDirection(bc)}),
		color, shapeBorderPaint(shape, w.child, w.borderOnForeground, bc), widgets.PhysicalShapeOpts{
			ClipBehavior: w.clipBehavior, Elevation: elevation, ShadowColor: foundation.Some(shadowColor)})
}

// shapeBorderPaint is Flutter's _ShapeBorderPaint: a CustomPaint that paints the
// border of the shape over (or under) the child.
//
// Flutter: material/material.dart _ShapeBorderPaint
func shapeBorderPaint(shape painting.ShapeBorder, child widgets.Widget, borderOnForeground bool, bc widgets.BuildContext) widgets.Widget {
	painter := newShapeBorderPainter(shape, maybeDirection(bc))
	if borderOnForeground {
		return widgets.NewCustomPaint(child, widgets.CustomPaintOpts{ForegroundPainter: painter})
	}
	return widgets.NewCustomPaint(child, widgets.CustomPaintOpts{Painter: painter})
}

// shapeBorderPainter is Flutter's _ShapeBorderPainter.
//
// Flutter: material/material.dart _ShapeBorderPainter
type shapeBorderPainter struct {
	rendering.CustomPainterBase
	border        painting.ShapeBorder
	textDirection ui.TextDirection
}

func newShapeBorderPainter(border painting.ShapeBorder, textDirection ui.TextDirection) *shapeBorderPainter {
	return &shapeBorderPainter{border: border, textDirection: textDirection}
}

func (p *shapeBorderPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	p.border.Paint(canvas, ui.NewRectFromLTWH(0, 0, size.Width, size.Height), p.textDirection)
}

func (p *shapeBorderPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*shapeBorderPainter)
	return !ok || !foundation.ValuesEqual(o.border, p.border)
}
