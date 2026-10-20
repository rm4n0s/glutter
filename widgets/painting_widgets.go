package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// OpacityOpts are the optional parameters of NewOpacity.
//
// Flutter: widgets/basic.dart Opacity.new
type OpacityOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AlwaysIncludeSemantics is whether the semantics of the child are included even when it is fully transparent.
	AlwaysIncludeSemantics bool
}

// Opacity makes its child partially transparent.
//
// Flutter: widgets/basic.dart Opacity
type Opacity struct {
	SingleChildRenderObjectWidget
	opacity                float64
	alwaysIncludeSemantics bool
}

// NewOpacity creates the widget.
//
// Flutter: widgets/basic.dart Opacity.new
func NewOpacity(opacity float64, child Widget, opts ...OpacityOpts) *Opacity {
	o := firstOpts(opts)
	w := &Opacity{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), opacity: opacity, alwaysIncludeSemantics: o.AlwaysIncludeSemantics}
	if foundation.DebugMode && !(opacity >= 0 && opacity <= 1) {
		panic(errors.New("OpacityOutOfRange", "the opacity must be between 0.0 and 1.0", "opacity", opacity))
	}
	return w
}

// Opacity is the fraction to scale the child's alpha value, from 0.0 (fully transparent) to 1.0 (fully opaque).
//
// Flutter: widgets/basic.dart Opacity.opacity
func (w *Opacity) Opacity() float64 { return w.opacity }

// AlwaysIncludeSemantics is whether the semantics of the child are included even when it is fully transparent.
//
// Flutter: widgets/basic.dart Opacity.alwaysIncludeSemantics
func (w *Opacity) AlwaysIncludeSemantics() bool { return w.alwaysIncludeSemantics }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Opacity.createRenderObject
func (w *Opacity) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderOpacity(nil, rendering.RenderOpacityOpts{Opacity: foundation.Some(w.opacity), AlwaysIncludeSemantics: w.alwaysIncludeSemantics})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart Opacity.updateRenderObject
func (w *Opacity) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderOpacity)
	r.SetOpacity(w.opacity)
	r.SetAlwaysIncludeSemantics(w.alwaysIncludeSemantics)
}

// TransformOpts are the optional parameters of NewTransform.
//
// Flutter: widgets/basic.dart Transform.new
type TransformOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Origin is the origin of the coordinate system, relative to the upper left corner, in which to apply the matrix.
	Origin foundation.Opt[ui.Offset]
	// Alignment is the alignment of the origin, relative to the size of the box.
	Alignment painting.AlignmentGeometry
	// TransformHitTests is whether to apply the transformation when performing hit tests; the default is true.
	TransformHitTests foundation.Opt[bool]
	// FilterQuality is the filter quality used when the child is painted into a layer with the transform.
	FilterQuality foundation.Opt[ui.FilterQuality]
}

// Transform applies a transformation before painting its child.
//
// Flutter: widgets/basic.dart Transform
type Transform struct {
	SingleChildRenderObjectWidget
	transform         *ui.Matrix4
	origin            foundation.Opt[ui.Offset]
	alignment         painting.AlignmentGeometry
	transformHitTests foundation.Opt[bool]
	filterQuality     foundation.Opt[ui.FilterQuality]
}

// NewTransform creates the widget.
//
// Flutter: widgets/basic.dart Transform.new
func NewTransform(transform *ui.Matrix4, child Widget, opts ...TransformOpts) *Transform {
	o := firstOpts(opts)
	w := &Transform{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), transform: transform, origin: o.Origin, alignment: o.Alignment, transformHitTests: o.TransformHitTests, filterQuality: o.FilterQuality}
	return w
}

// Transform is the matrix to transform the child by during painting.
//
// Flutter: widgets/basic.dart Transform.transform
func (w *Transform) Transform() *ui.Matrix4 { return w.transform }

// Origin is the origin of the coordinate system, relative to the upper left corner, in which to apply the matrix.
//
// Flutter: widgets/basic.dart Transform.origin
func (w *Transform) Origin() foundation.Opt[ui.Offset] { return w.origin }

// Alignment is the alignment of the origin, relative to the size of the box.
//
// Flutter: widgets/basic.dart Transform.alignment
func (w *Transform) Alignment() painting.AlignmentGeometry { return w.alignment }

// TransformHitTests is whether to apply the transformation when performing hit tests; the default is true.
//
// Flutter: widgets/basic.dart Transform.transformHitTests
func (w *Transform) TransformHitTests() foundation.Opt[bool] { return w.transformHitTests }

// FilterQuality is the filter quality used when the child is painted into a layer with the transform.
//
// Flutter: widgets/basic.dart Transform.filterQuality
func (w *Transform) FilterQuality() foundation.Opt[ui.FilterQuality] { return w.filterQuality }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart Transform.createRenderObject
func (w *Transform) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderTransform(w.transform, nil, rendering.RenderTransformOpts{Origin: w.origin, Alignment: w.alignment,
		TextDirection: maybeTextDirection(bc), TransformHitTests: w.transformHitTests, FilterQuality: w.filterQuality})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart Transform.updateRenderObject
func (w *Transform) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderTransform)
	r.SetTransform(w.transform)
	r.SetOrigin(w.origin)
	r.SetAlignment(w.alignment)
	r.SetTextDirection(maybeTextDirection(bc))
	r.TransformHitTests = w.transformHitTests.OrElse(true)
	r.SetFilterQuality(w.filterQuality)
}

// ClipRectOpts are the optional parameters of NewClipRect.
//
// Flutter: widgets/basic.dart ClipRect.new
type ClipRectOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Clipper calculates the rectangle used for clipping; the default is the size of the child.
	Clipper rendering.CustomClipper[ui.Rect]
	// ClipBehavior is whether the content is clipped and how; the default is ClipHardEdge.
	ClipBehavior ui.Clip
}

// ClipRect clips its child using a rectangle.
//
// Flutter: widgets/basic.dart ClipRect
type ClipRect struct {
	SingleChildRenderObjectWidget
	clipper      rendering.CustomClipper[ui.Rect]
	clipBehavior ui.Clip
}

// NewClipRect creates the widget.
//
// Flutter: widgets/basic.dart ClipRect.new
func NewClipRect(child Widget, opts ...ClipRectOpts) *ClipRect {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipHardEdge
	}
	w := &ClipRect{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), clipper: o.Clipper, clipBehavior: o.ClipBehavior}
	return w
}

// Clipper calculates the rectangle used for clipping; the default is the size of the child.
//
// Flutter: widgets/basic.dart ClipRect.clipper
func (w *ClipRect) Clipper() rendering.CustomClipper[ui.Rect] { return w.clipper }

// ClipBehavior is whether the content is clipped and how; the default is ClipHardEdge.
//
// Flutter: widgets/basic.dart ClipRect.clipBehavior
func (w *ClipRect) ClipBehavior() ui.Clip { return w.clipBehavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart ClipRect.createRenderObject
func (w *ClipRect) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderClipRect(nil, rendering.RenderClipRectOpts{Clipper: w.clipper, ClipBehavior: w.clipBehavior})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart ClipRect.updateRenderObject
func (w *ClipRect) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderClipRect)
	r.SetClipper(w.clipper)
	r.SetClipBehavior(w.clipBehavior)
}

// DidUnmountRenderObject clears what the render object holds.
//
// Flutter: widgets/basic.dart ClipRect.didUnmountRenderObject
func (w *ClipRect) DidUnmountRenderObject(renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderClipRect).SetClipper(nil)
}

// ClipRRectOpts are the optional parameters of NewClipRRect.
//
// Flutter: widgets/basic.dart ClipRRect.new
type ClipRRectOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BorderRadius is the border radius of the rounded corners; the default is BorderRadiusZero.
	BorderRadius painting.BorderRadiusGeometry
	// Clipper calculates the rounded rectangle used for clipping; it replaces the border radius when given.
	Clipper rendering.CustomClipper[ui.RRect]
	// ClipBehavior is whether the content is clipped and how; the default is ClipAntiAlias.
	ClipBehavior ui.Clip
}

// ClipRRect clips its child using a rounded rectangle.
//
// Flutter: widgets/basic.dart ClipRRect
type ClipRRect struct {
	SingleChildRenderObjectWidget
	borderRadius painting.BorderRadiusGeometry
	clipper      rendering.CustomClipper[ui.RRect]
	clipBehavior ui.Clip
}

// NewClipRRect creates the widget.
//
// Flutter: widgets/basic.dart ClipRRect.new
func NewClipRRect(child Widget, opts ...ClipRRectOpts) *ClipRRect {
	o := firstOpts(opts)
	if o.BorderRadius == nil {
		o.BorderRadius = painting.BorderRadiusZero
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipAntiAlias
	}
	w := &ClipRRect{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), borderRadius: o.BorderRadius, clipper: o.Clipper, clipBehavior: o.ClipBehavior}
	return w
}

// BorderRadius is the border radius of the rounded corners; the default is BorderRadiusZero.
//
// Flutter: widgets/basic.dart ClipRRect.borderRadius
func (w *ClipRRect) BorderRadius() painting.BorderRadiusGeometry { return w.borderRadius }

// Clipper calculates the rounded rectangle used for clipping; it replaces the border radius when given.
//
// Flutter: widgets/basic.dart ClipRRect.clipper
func (w *ClipRRect) Clipper() rendering.CustomClipper[ui.RRect] { return w.clipper }

// ClipBehavior is whether the content is clipped and how; the default is ClipAntiAlias.
//
// Flutter: widgets/basic.dart ClipRRect.clipBehavior
func (w *ClipRRect) ClipBehavior() ui.Clip { return w.clipBehavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart ClipRRect.createRenderObject
func (w *ClipRRect) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderClipRRect(nil, rendering.RenderClipRRectOpts{BorderRadius: w.borderRadius, Clipper: w.clipper,
		ClipBehavior: w.clipBehavior, TextDirection: maybeTextDirection(bc)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart ClipRRect.updateRenderObject
func (w *ClipRRect) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderClipRRect)
	r.SetBorderRadius(w.borderRadius)
	r.SetClipper(w.clipper)
	r.SetClipBehavior(w.clipBehavior)
	r.SetTextDirection(maybeTextDirection(bc))
}

// DidUnmountRenderObject clears what the render object holds.
//
// Flutter: widgets/basic.dart ClipRRect.didUnmountRenderObject
func (w *ClipRRect) DidUnmountRenderObject(renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderClipRRect).SetClipper(nil)
}

// ClipRSuperellipseOpts are the optional parameters of NewClipRSuperellipse.
//
// Flutter: widgets/basic.dart ClipRSuperellipse.new
type ClipRSuperellipseOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BorderRadius is the border radius of the rounded corners; the default is BorderRadiusZero.
	BorderRadius painting.BorderRadiusGeometry
	// Clipper calculates the rounded superellipse used for clipping.
	Clipper rendering.CustomClipper[ui.RSuperellipse]
	// ClipBehavior is whether the content is clipped and how; the default is ClipAntiAlias.
	ClipBehavior ui.Clip
}

// ClipRSuperellipse clips its child using a rounded superellipse.
//
// Flutter: widgets/basic.dart ClipRSuperellipse
type ClipRSuperellipse struct {
	SingleChildRenderObjectWidget
	borderRadius painting.BorderRadiusGeometry
	clipper      rendering.CustomClipper[ui.RSuperellipse]
	clipBehavior ui.Clip
}

// NewClipRSuperellipse creates the widget.
//
// Flutter: widgets/basic.dart ClipRSuperellipse.new
func NewClipRSuperellipse(child Widget, opts ...ClipRSuperellipseOpts) *ClipRSuperellipse {
	o := firstOpts(opts)
	if o.BorderRadius == nil {
		o.BorderRadius = painting.BorderRadiusZero
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipAntiAlias
	}
	w := &ClipRSuperellipse{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), borderRadius: o.BorderRadius, clipper: o.Clipper, clipBehavior: o.ClipBehavior}
	return w
}

// BorderRadius is the border radius of the rounded corners; the default is BorderRadiusZero.
//
// Flutter: widgets/basic.dart ClipRSuperellipse.borderRadius
func (w *ClipRSuperellipse) BorderRadius() painting.BorderRadiusGeometry { return w.borderRadius }

// Clipper calculates the rounded superellipse used for clipping.
//
// Flutter: widgets/basic.dart ClipRSuperellipse.clipper
func (w *ClipRSuperellipse) Clipper() rendering.CustomClipper[ui.RSuperellipse] { return w.clipper }

// ClipBehavior is whether the content is clipped and how; the default is ClipAntiAlias.
//
// Flutter: widgets/basic.dart ClipRSuperellipse.clipBehavior
func (w *ClipRSuperellipse) ClipBehavior() ui.Clip { return w.clipBehavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart ClipRSuperellipse.createRenderObject
func (w *ClipRSuperellipse) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderClipRSuperellipse(nil, rendering.RenderClipRSuperellipseOpts{BorderRadius: w.borderRadius, Clipper: w.clipper,
		ClipBehavior: w.clipBehavior, TextDirection: maybeTextDirection(bc)})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart ClipRSuperellipse.updateRenderObject
func (w *ClipRSuperellipse) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderClipRSuperellipse)
	r.SetBorderRadius(w.borderRadius)
	r.SetClipper(w.clipper)
	r.SetClipBehavior(w.clipBehavior)
	r.SetTextDirection(maybeTextDirection(bc))
}

// DidUnmountRenderObject clears what the render object holds.
//
// Flutter: widgets/basic.dart ClipRSuperellipse.didUnmountRenderObject
func (w *ClipRSuperellipse) DidUnmountRenderObject(renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderClipRSuperellipse).SetClipper(nil)
}

// ClipOvalOpts are the optional parameters of NewClipOval.
//
// Flutter: widgets/basic.dart ClipOval.new
type ClipOvalOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Clipper calculates the rectangle whose inscribed oval is used for clipping; the default is the size of the child.
	Clipper rendering.CustomClipper[ui.Rect]
	// ClipBehavior is whether the content is clipped and how; the default is ClipAntiAlias.
	ClipBehavior ui.Clip
}

// ClipOval clips its child using an oval.
//
// Flutter: widgets/basic.dart ClipOval
type ClipOval struct {
	SingleChildRenderObjectWidget
	clipper      rendering.CustomClipper[ui.Rect]
	clipBehavior ui.Clip
}

// NewClipOval creates the widget.
//
// Flutter: widgets/basic.dart ClipOval.new
func NewClipOval(child Widget, opts ...ClipOvalOpts) *ClipOval {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipAntiAlias
	}
	w := &ClipOval{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), clipper: o.Clipper, clipBehavior: o.ClipBehavior}
	return w
}

// Clipper calculates the rectangle whose inscribed oval is used for clipping; the default is the size of the child.
//
// Flutter: widgets/basic.dart ClipOval.clipper
func (w *ClipOval) Clipper() rendering.CustomClipper[ui.Rect] { return w.clipper }

// ClipBehavior is whether the content is clipped and how; the default is ClipAntiAlias.
//
// Flutter: widgets/basic.dart ClipOval.clipBehavior
func (w *ClipOval) ClipBehavior() ui.Clip { return w.clipBehavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart ClipOval.createRenderObject
func (w *ClipOval) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderClipOval(nil, rendering.RenderClipRectOpts{Clipper: w.clipper, ClipBehavior: w.clipBehavior})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart ClipOval.updateRenderObject
func (w *ClipOval) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderClipOval)
	r.SetClipper(w.clipper)
	r.SetClipBehavior(w.clipBehavior)
}

// DidUnmountRenderObject clears what the render object holds.
//
// Flutter: widgets/basic.dart ClipOval.didUnmountRenderObject
func (w *ClipOval) DidUnmountRenderObject(renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderClipOval).SetClipper(nil)
}

// ClipPathOpts are the optional parameters of NewClipPath.
//
// Flutter: widgets/basic.dart ClipPath.new
type ClipPathOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Clipper calculates the path used for clipping; the default is a rectangle the size of the child.
	Clipper rendering.CustomClipper[*ui.Path]
	// ClipBehavior is whether the content is clipped and how; the default is ClipAntiAlias.
	ClipBehavior ui.Clip
}

// ClipPath clips its child using a path.
//
// Flutter: widgets/basic.dart ClipPath
type ClipPath struct {
	SingleChildRenderObjectWidget
	clipper      rendering.CustomClipper[*ui.Path]
	clipBehavior ui.Clip
}

// NewClipPath creates the widget.
//
// Flutter: widgets/basic.dart ClipPath.new
func NewClipPath(child Widget, opts ...ClipPathOpts) *ClipPath {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipAntiAlias
	}
	w := &ClipPath{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), clipper: o.Clipper, clipBehavior: o.ClipBehavior}
	return w
}

// Clipper calculates the path used for clipping; the default is a rectangle the size of the child.
//
// Flutter: widgets/basic.dart ClipPath.clipper
func (w *ClipPath) Clipper() rendering.CustomClipper[*ui.Path] { return w.clipper }

// ClipBehavior is whether the content is clipped and how; the default is ClipAntiAlias.
//
// Flutter: widgets/basic.dart ClipPath.clipBehavior
func (w *ClipPath) ClipBehavior() ui.Clip { return w.clipBehavior }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart ClipPath.createRenderObject
func (w *ClipPath) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderClipPath(nil, rendering.RenderClipPathOpts{Clipper: w.clipper, ClipBehavior: w.clipBehavior})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart ClipPath.updateRenderObject
func (w *ClipPath) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderClipPath)
	r.SetClipper(w.clipper)
	r.SetClipBehavior(w.clipBehavior)
}

// DidUnmountRenderObject clears what the render object holds.
//
// Flutter: widgets/basic.dart ClipPath.didUnmountRenderObject
func (w *ClipPath) DidUnmountRenderObject(renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderClipPath).SetClipper(nil)
}

// PhysicalModelOpts are the optional parameters of NewPhysicalModel.
//
// Flutter: widgets/basic.dart PhysicalModel.new
type PhysicalModelOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Shape is the type of shape; the default is BoxShapeRectangle.
	Shape painting.BoxShape
	// ClipBehavior is whether the content is clipped and how; the default is ClipNone.
	ClipBehavior ui.Clip
	// BorderRadius is the border radius of the rounded corners when the shape is a rectangle.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// Elevation is the z-coordinate at which to place this physical object relative to its parent.
	Elevation float64
	// ShadowColor is the color to paint the shadow below the material; the default is opaque black.
	ShadowColor foundation.Opt[ui.Color]
}

// PhysicalModel represents a physical layer that clips its children to a shape and casts a shadow.
//
// Flutter: widgets/basic.dart PhysicalModel
type PhysicalModel struct {
	SingleChildRenderObjectWidget
	color        ui.Color
	shape        painting.BoxShape
	clipBehavior ui.Clip
	borderRadius foundation.Opt[painting.BorderRadius]
	elevation    float64
	shadowColor  foundation.Opt[ui.Color]
}

// NewPhysicalModel creates the widget.
//
// Flutter: widgets/basic.dart PhysicalModel.new
func NewPhysicalModel(color ui.Color, child Widget, opts ...PhysicalModelOpts) *PhysicalModel {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	if o.Shape == painting.BoxShapeUnspecified {
		o.Shape = painting.BoxShapeRectangle
	}
	w := &PhysicalModel{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), color: color, shape: o.Shape, clipBehavior: o.ClipBehavior, borderRadius: o.BorderRadius, elevation: o.Elevation, shadowColor: o.ShadowColor}
	return w
}

// Color is the background color.
//
// Flutter: widgets/basic.dart PhysicalModel.color
func (w *PhysicalModel) Color() ui.Color { return w.color }

// Shape is the type of shape; the default is BoxShapeRectangle.
//
// Flutter: widgets/basic.dart PhysicalModel.shape
func (w *PhysicalModel) Shape() painting.BoxShape { return w.shape }

// ClipBehavior is whether the content is clipped and how; the default is ClipNone.
//
// Flutter: widgets/basic.dart PhysicalModel.clipBehavior
func (w *PhysicalModel) ClipBehavior() ui.Clip { return w.clipBehavior }

// BorderRadius is the border radius of the rounded corners when the shape is a rectangle.
//
// Flutter: widgets/basic.dart PhysicalModel.borderRadius
func (w *PhysicalModel) BorderRadius() foundation.Opt[painting.BorderRadius] { return w.borderRadius }

// Elevation is the z-coordinate at which to place this physical object relative to its parent.
//
// Flutter: widgets/basic.dart PhysicalModel.elevation
func (w *PhysicalModel) Elevation() float64 { return w.elevation }

// ShadowColor is the color to paint the shadow below the material; the default is opaque black.
//
// Flutter: widgets/basic.dart PhysicalModel.shadowColor
func (w *PhysicalModel) ShadowColor() foundation.Opt[ui.Color] { return w.shadowColor }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart PhysicalModel.createRenderObject
func (w *PhysicalModel) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderPhysicalModel(w.color, nil, rendering.RenderPhysicalModelOpts{Shape: w.shape, ClipBehavior: w.clipBehavior,
		BorderRadius: w.borderRadius, Elevation: w.elevation, ShadowColor: w.shadowColor})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart PhysicalModel.updateRenderObject
func (w *PhysicalModel) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderPhysicalModel)
	r.SetShape(w.shape)
	r.SetClipBehavior(w.clipBehavior)
	r.SetBorderRadius(w.borderRadius)
	r.SetColor(w.color)
	r.SetShadowColor(w.shadowColor.OrElse(ui.NewColor(0xFF000000)))
	r.SetElevation(w.elevation)
}

// PhysicalShapeOpts are the optional parameters of NewPhysicalShape.
//
// Flutter: widgets/basic.dart PhysicalShape.new
type PhysicalShapeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// ClipBehavior is whether the content is clipped and how; the default is ClipNone.
	ClipBehavior ui.Clip
	// Elevation is the z-coordinate at which to place this physical object relative to its parent.
	Elevation float64
	// ShadowColor is the color to paint the shadow below the material; the default is opaque black.
	ShadowColor foundation.Opt[ui.Color]
}

// PhysicalShape represents a physical layer that clips its children to a path and casts a shadow.
//
// Flutter: widgets/basic.dart PhysicalShape
type PhysicalShape struct {
	SingleChildRenderObjectWidget
	clipper      rendering.CustomClipper[*ui.Path]
	color        ui.Color
	clipBehavior ui.Clip
	elevation    float64
	shadowColor  foundation.Opt[ui.Color]
}

// NewPhysicalShape creates the widget.
//
// Flutter: widgets/basic.dart PhysicalShape.new
func NewPhysicalShape(clipper rendering.CustomClipper[*ui.Path], color ui.Color, child Widget, opts ...PhysicalShapeOpts) *PhysicalShape {
	o := firstOpts(opts)
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	w := &PhysicalShape{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), clipper: clipper, color: color, clipBehavior: o.ClipBehavior, elevation: o.Elevation, shadowColor: o.ShadowColor}
	return w
}

// Clipper calculates the path used to clip the child and to cast the shadow.
//
// Flutter: widgets/basic.dart PhysicalShape.clipper
func (w *PhysicalShape) Clipper() rendering.CustomClipper[*ui.Path] { return w.clipper }

// Color is the background color.
//
// Flutter: widgets/basic.dart PhysicalShape.color
func (w *PhysicalShape) Color() ui.Color { return w.color }

// ClipBehavior is whether the content is clipped and how; the default is ClipNone.
//
// Flutter: widgets/basic.dart PhysicalShape.clipBehavior
func (w *PhysicalShape) ClipBehavior() ui.Clip { return w.clipBehavior }

// Elevation is the z-coordinate at which to place this physical object relative to its parent.
//
// Flutter: widgets/basic.dart PhysicalShape.elevation
func (w *PhysicalShape) Elevation() float64 { return w.elevation }

// ShadowColor is the color to paint the shadow below the material; the default is opaque black.
//
// Flutter: widgets/basic.dart PhysicalShape.shadowColor
func (w *PhysicalShape) ShadowColor() foundation.Opt[ui.Color] { return w.shadowColor }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart PhysicalShape.createRenderObject
func (w *PhysicalShape) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderPhysicalShape(w.clipper, w.color, nil, rendering.RenderPhysicalModelOpts{ClipBehavior: w.clipBehavior,
		Elevation: w.elevation, ShadowColor: w.shadowColor})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart PhysicalShape.updateRenderObject
func (w *PhysicalShape) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderPhysicalShape)
	r.SetClipper(w.clipper)
	r.SetClipBehavior(w.clipBehavior)
	r.SetColor(w.color)
	r.SetShadowColor(w.shadowColor.OrElse(ui.NewColor(0xFF000000)))
	r.SetElevation(w.elevation)
}

// DidUnmountRenderObject clears what the render object holds.
//
// Flutter: widgets/basic.dart PhysicalShape.didUnmountRenderObject
func (w *PhysicalShape) DidUnmountRenderObject(renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderPhysicalShape).SetClipper(nil)
}

// CustomPaintOpts are the optional parameters of NewCustomPaint.
//
// Flutter: widgets/basic.dart CustomPaint.new
type CustomPaintOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Painter paints before the child; nil for none.
	Painter rendering.CustomPainter
	// ForegroundPainter paints after the child; nil for none.
	ForegroundPainter rendering.CustomPainter
	// Size is the size that this widget would prefer if it has no child; the default is zero.
	Size ui.Size
	// IsComplex is whether the painting is complex enough to benefit from caching.
	IsComplex bool
	// WillChange is whether the raster cache should be told that this painting is likely to change in the next frame.
	WillChange bool
}

// CustomPaint provides a canvas on which to draw during the paint phase.
//
// Flutter: widgets/basic.dart CustomPaint
type CustomPaint struct {
	SingleChildRenderObjectWidget
	painter           rendering.CustomPainter
	foregroundPainter rendering.CustomPainter
	size              ui.Size
	isComplex         bool
	willChange        bool
}

// NewCustomPaint creates the widget.
//
// Flutter: widgets/basic.dart CustomPaint.new
func NewCustomPaint(child Widget, opts ...CustomPaintOpts) *CustomPaint {
	o := firstOpts(opts)
	w := &CustomPaint{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), painter: o.Painter, foregroundPainter: o.ForegroundPainter, size: o.Size, isComplex: o.IsComplex, willChange: o.WillChange}
	return w
}

// Painter paints before the child; nil for none.
//
// Flutter: widgets/basic.dart CustomPaint.painter
func (w *CustomPaint) Painter() rendering.CustomPainter { return w.painter }

// ForegroundPainter paints after the child; nil for none.
//
// Flutter: widgets/basic.dart CustomPaint.foregroundPainter
func (w *CustomPaint) ForegroundPainter() rendering.CustomPainter { return w.foregroundPainter }

// Size is the size that this widget would prefer if it has no child; the default is zero.
//
// Flutter: widgets/basic.dart CustomPaint.size
func (w *CustomPaint) Size() ui.Size { return w.size }

// IsComplex is whether the painting is complex enough to benefit from caching.
//
// Flutter: widgets/basic.dart CustomPaint.isComplex
func (w *CustomPaint) IsComplex() bool { return w.isComplex }

// WillChange is whether the raster cache should be told that this painting is likely to change in the next frame.
//
// Flutter: widgets/basic.dart CustomPaint.willChange
func (w *CustomPaint) WillChange() bool { return w.willChange }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart CustomPaint.createRenderObject
func (w *CustomPaint) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderCustomPaint(nil, rendering.RenderCustomPaintOpts{Painter: w.painter, ForegroundPainter: w.foregroundPainter,
		PreferredSize: w.size, IsComplex: w.isComplex, WillChange: w.willChange})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart CustomPaint.updateRenderObject
func (w *CustomPaint) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderCustomPaint)
	r.SetPainter(w.painter)
	r.SetForegroundPainter(w.foregroundPainter)
	r.SetPreferredSize(w.size)
	r.IsComplex = w.isComplex
	r.WillChange = w.willChange
}

// DidUnmountRenderObject clears what the render object holds.
//
// Flutter: widgets/basic.dart CustomPaint.didUnmountRenderObject
func (w *CustomPaint) DidUnmountRenderObject(renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderCustomPaint)
	r.SetPainter(nil)
	r.SetForegroundPainter(nil)
}

// DecoratedBoxOpts are the optional parameters of NewDecoratedBox.
//
// Flutter: widgets/basic.dart DecoratedBox.new
type DecoratedBoxOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Position is whether to paint the box decoration behind or in front of the child; the default is DecorationPositionBackground.
	Position rendering.DecorationPosition
}

// DecoratedBox paints a decoration either before or after its child paints.
//
// Flutter: widgets/basic.dart DecoratedBox
type DecoratedBox struct {
	SingleChildRenderObjectWidget
	decoration painting.Decoration
	position   rendering.DecorationPosition
}

// NewDecoratedBox creates the widget.
//
// Flutter: widgets/basic.dart DecoratedBox.new
func NewDecoratedBox(decoration painting.Decoration, child Widget, opts ...DecoratedBoxOpts) *DecoratedBox {
	o := firstOpts(opts)
	if o.Position == rendering.DecorationPositionUnspecified {
		o.Position = rendering.DecorationPositionBackground
	}
	w := &DecoratedBox{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), decoration: decoration, position: o.Position}
	return w
}

// Decoration is what to paint.
//
// Flutter: widgets/basic.dart DecoratedBox.decoration
func (w *DecoratedBox) Decoration() painting.Decoration { return w.decoration }

// Position is whether to paint the box decoration behind or in front of the child; the default is DecorationPositionBackground.
//
// Flutter: widgets/basic.dart DecoratedBox.position
func (w *DecoratedBox) Position() rendering.DecorationPosition { return w.position }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart DecoratedBox.createRenderObject
func (w *DecoratedBox) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderDecoratedBox(w.decoration, nil, rendering.RenderDecoratedBoxOpts{Position: w.position,
		Configuration: CreateLocalImageConfiguration(bc, foundation.None[ui.Size]())})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart DecoratedBox.updateRenderObject
func (w *DecoratedBox) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderDecoratedBox)
	r.SetDecoration(w.decoration)
	r.SetConfiguration(CreateLocalImageConfiguration(bc, foundation.None[ui.Size]()))
	r.SetPosition(w.position)
}

// ShaderMaskOpts are the optional parameters of NewShaderMask.
//
// Flutter: widgets/basic.dart ShaderMask.new
type ShaderMaskOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BlendMode is the blend mode to use when applying the shader to the child; the default is BlendModeModulate.
	BlendMode ui.BlendMode
}

// ShaderMask applies a mask generated by a shader to its child.
//
// Flutter: widgets/basic.dart ShaderMask
type ShaderMask struct {
	SingleChildRenderObjectWidget
	shaderCallback rendering.ShaderCallback
	blendMode      ui.BlendMode
}

// NewShaderMask creates the widget.
//
// Flutter: widgets/basic.dart ShaderMask.new
func NewShaderMask(shaderCallback rendering.ShaderCallback, child Widget, opts ...ShaderMaskOpts) *ShaderMask {
	o := firstOpts(opts)
	if o.BlendMode == ui.BlendModeUnspecified {
		o.BlendMode = ui.BlendModeModulate
	}
	w := &ShaderMask{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), shaderCallback: shaderCallback, blendMode: o.BlendMode}
	return w
}

// ShaderCallback is called to create the shader for the mask, given the bounds of the child.
//
// Flutter: widgets/basic.dart ShaderMask.shaderCallback
func (w *ShaderMask) ShaderCallback() rendering.ShaderCallback { return w.shaderCallback }

// BlendMode is the blend mode to use when applying the shader to the child; the default is BlendModeModulate.
//
// Flutter: widgets/basic.dart ShaderMask.blendMode
func (w *ShaderMask) BlendMode() ui.BlendMode { return w.blendMode }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart ShaderMask.createRenderObject
func (w *ShaderMask) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderShaderMask(w.shaderCallback, nil, rendering.RenderShaderMaskOpts{BlendMode: w.blendMode})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart ShaderMask.updateRenderObject
func (w *ShaderMask) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderShaderMask)
	r.SetShaderCallback(w.shaderCallback)
	r.SetBlendMode(w.blendMode)
}

// BackdropFilterOpts are the optional parameters of NewBackdropFilter.
//
// Flutter: widgets/basic.dart BackdropFilter.new
type BackdropFilterOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BlendMode is the blend mode to use to apply the filtered background content onto the background; the default is BlendModeSrcOver.
	BlendMode ui.BlendMode
	// Enabled is whether the filter is applied; the default is true.
	Enabled foundation.Opt[bool]
	// BackdropKey is the key shared by the backdrop filters that use the same backdrop.
	BackdropKey foundation.Opt[rendering.BackdropKey]
}

// BackdropFilter applies a filter to the existing painted content and then paints its child.
//
// Flutter: widgets/basic.dart BackdropFilter
type BackdropFilter struct {
	SingleChildRenderObjectWidget
	filter      ui.ImageFilter
	blendMode   ui.BlendMode
	enabled     foundation.Opt[bool]
	backdropKey foundation.Opt[rendering.BackdropKey]
}

// NewBackdropFilter creates the widget.
//
// Flutter: widgets/basic.dart BackdropFilter.new
func NewBackdropFilter(filter ui.ImageFilter, child Widget, opts ...BackdropFilterOpts) *BackdropFilter {
	o := firstOpts(opts)
	if o.BlendMode == ui.BlendModeUnspecified {
		o.BlendMode = ui.BlendModeSrcOver
	}
	w := &BackdropFilter{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), filter: filter, blendMode: o.BlendMode, enabled: o.Enabled, backdropKey: o.BackdropKey}
	return w
}

// Filter is the image filter to apply to the existing painted content.
//
// Flutter: widgets/basic.dart BackdropFilter.filter
func (w *BackdropFilter) Filter() ui.ImageFilter { return w.filter }

// BlendMode is the blend mode to use to apply the filtered background content onto the background; the default is BlendModeSrcOver.
//
// Flutter: widgets/basic.dart BackdropFilter.blendMode
func (w *BackdropFilter) BlendMode() ui.BlendMode { return w.blendMode }

// Enabled is whether the filter is applied; the default is true.
//
// Flutter: widgets/basic.dart BackdropFilter.enabled
func (w *BackdropFilter) Enabled() foundation.Opt[bool] { return w.enabled }

// BackdropKey is the key shared by the backdrop filters that use the same backdrop.
//
// Flutter: widgets/basic.dart BackdropFilter.backdropKey
func (w *BackdropFilter) BackdropKey() foundation.Opt[rendering.BackdropKey] { return w.backdropKey }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart BackdropFilter.createRenderObject
func (w *BackdropFilter) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderBackdropFilter(w.filter, nil, rendering.RenderBackdropFilterOpts{BlendMode: w.blendMode,
		Enabled: w.enabled, BackdropKey: w.backdropKey})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart BackdropFilter.updateRenderObject
func (w *BackdropFilter) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*rendering.RenderBackdropFilter)
	r.SetFilter(w.filter)
	r.SetBlendMode(w.blendMode)
	r.SetEnabled(w.enabled.OrElse(true))
	r.SetBackdropKey(w.backdropKey)
}

// RepaintBoundaryOpts are the optional parameters of NewRepaintBoundary.
//
// Flutter: widgets/basic.dart RepaintBoundary.new
type RepaintBoundaryOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// RepaintBoundary creates a separate display list for its child.
//
// Flutter: widgets/basic.dart RepaintBoundary
type RepaintBoundary struct {
	SingleChildRenderObjectWidget
}

// NewRepaintBoundary creates the widget.
//
// Flutter: widgets/basic.dart RepaintBoundary.new
func NewRepaintBoundary(child Widget, opts ...RepaintBoundaryOpts) *RepaintBoundary {
	o := firstOpts(opts)
	w := &RepaintBoundary{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts(o))}
	return w
}

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart RepaintBoundary.createRenderObject
func (w *RepaintBoundary) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderRepaintBoundary(nil)
}

// IgnorePointerOpts are the optional parameters of NewIgnorePointer.
//
// Flutter: widgets/basic.dart IgnorePointer.new
type IgnorePointerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Ignoring is whether this widget is ignored during hit testing; the default is true.
	Ignoring foundation.Opt[bool]
}

// IgnorePointer is invisible to hit testing.
//
// Flutter: widgets/basic.dart IgnorePointer
type IgnorePointer struct {
	SingleChildRenderObjectWidget
	ignoring foundation.Opt[bool]
}

// NewIgnorePointer creates the widget.
//
// Flutter: widgets/basic.dart IgnorePointer.new
func NewIgnorePointer(child Widget, opts ...IgnorePointerOpts) *IgnorePointer {
	o := firstOpts(opts)
	w := &IgnorePointer{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), ignoring: o.Ignoring}
	return w
}

// Ignoring is whether this widget is ignored during hit testing; the default is true.
//
// Flutter: widgets/basic.dart IgnorePointer.ignoring
func (w *IgnorePointer) Ignoring() foundation.Opt[bool] { return w.ignoring }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart IgnorePointer.createRenderObject
func (w *IgnorePointer) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderIgnorePointer(nil, rendering.RenderIgnorePointerOpts{Ignoring: w.ignoring})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart IgnorePointer.updateRenderObject
func (w *IgnorePointer) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderIgnorePointer).SetIgnoring(w.ignoring.OrElse(true))
}

// AbsorbPointerOpts are the optional parameters of NewAbsorbPointer.
//
// Flutter: widgets/basic.dart AbsorbPointer.new
type AbsorbPointerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Absorbing is whether this widget absorbs pointers during hit testing; the default is true.
	Absorbing foundation.Opt[bool]
}

// AbsorbPointer absorbs pointers during hit testing.
//
// Flutter: widgets/basic.dart AbsorbPointer
type AbsorbPointer struct {
	SingleChildRenderObjectWidget
	absorbing foundation.Opt[bool]
}

// NewAbsorbPointer creates the widget.
//
// Flutter: widgets/basic.dart AbsorbPointer.new
func NewAbsorbPointer(child Widget, opts ...AbsorbPointerOpts) *AbsorbPointer {
	o := firstOpts(opts)
	w := &AbsorbPointer{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts{Key: o.Key}), absorbing: o.Absorbing}
	return w
}

// Absorbing is whether this widget absorbs pointers during hit testing; the default is true.
//
// Flutter: widgets/basic.dart AbsorbPointer.absorbing
func (w *AbsorbPointer) Absorbing() foundation.Opt[bool] { return w.absorbing }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart AbsorbPointer.createRenderObject
func (w *AbsorbPointer) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderAbsorbPointer(nil, rendering.RenderAbsorbPointerOpts{Absorbing: w.absorbing})
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart AbsorbPointer.updateRenderObject
func (w *AbsorbPointer) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderAbsorbPointer).SetAbsorbing(w.absorbing.OrElse(true))
}

// CustomSingleChildLayoutOpts are the optional parameters of NewCustomSingleChildLayout.
//
// Flutter: widgets/basic.dart CustomSingleChildLayout.new
type CustomSingleChildLayoutOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// CustomSingleChildLayout defers the layout of its single child to a delegate.
//
// Flutter: widgets/basic.dart CustomSingleChildLayout
type CustomSingleChildLayout struct {
	SingleChildRenderObjectWidget
	delegate rendering.SingleChildLayoutDelegate
}

// NewCustomSingleChildLayout creates the widget.
//
// Flutter: widgets/basic.dart CustomSingleChildLayout.new
func NewCustomSingleChildLayout(delegate rendering.SingleChildLayoutDelegate, child Widget, opts ...CustomSingleChildLayoutOpts) *CustomSingleChildLayout {
	o := firstOpts(opts)
	w := &CustomSingleChildLayout{SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(child, WidgetOpts(o)), delegate: delegate}
	return w
}

// Delegate controls the layout of the child.
//
// Flutter: widgets/basic.dart CustomSingleChildLayout.delegate
func (w *CustomSingleChildLayout) Delegate() rendering.SingleChildLayoutDelegate { return w.delegate }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart CustomSingleChildLayout.createRenderObject
func (w *CustomSingleChildLayout) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderCustomSingleChildLayoutBox(w.delegate, nil)
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart CustomSingleChildLayout.updateRenderObject
func (w *CustomSingleChildLayout) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderCustomSingleChildLayoutBox).SetDelegate(w.delegate)
}

// CustomMultiChildLayoutOpts are the optional parameters of NewCustomMultiChildLayout.
//
// Flutter: widgets/basic.dart CustomMultiChildLayout.new
type CustomMultiChildLayoutOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// CustomMultiChildLayout uses a delegate to size and position multiple children.
//
// Flutter: widgets/basic.dart CustomMultiChildLayout
type CustomMultiChildLayout struct {
	MultiChildRenderObjectWidget
	delegate rendering.MultiChildLayoutDelegate
}

// NewCustomMultiChildLayout creates the widget.
//
// Flutter: widgets/basic.dart CustomMultiChildLayout.new
func NewCustomMultiChildLayout(delegate rendering.MultiChildLayoutDelegate, children []Widget, opts ...CustomMultiChildLayoutOpts) *CustomMultiChildLayout {
	o := firstOpts(opts)
	w := &CustomMultiChildLayout{MultiChildRenderObjectWidget: NewMultiChildRenderObjectWidgetBase(children, WidgetOpts(o)), delegate: delegate}
	return w
}

// Delegate controls the layout of the children.
//
// Flutter: widgets/basic.dart CustomMultiChildLayout.delegate
func (w *CustomMultiChildLayout) Delegate() rendering.MultiChildLayoutDelegate { return w.delegate }

// CreateRenderObject creates the render object.
//
// Flutter: widgets/basic.dart CustomMultiChildLayout.createRenderObject
func (w *CustomMultiChildLayout) CreateRenderObject(bc BuildContext) rendering.RenderObject {
	return rendering.NewRenderCustomMultiChildLayoutBox(w.delegate, nil)
}

// UpdateRenderObject copies the configuration to the render object.
//
// Flutter: widgets/basic.dart CustomMultiChildLayout.updateRenderObject
func (w *CustomMultiChildLayout) UpdateRenderObject(bc BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*rendering.RenderCustomMultiChildLayoutBox).SetDelegate(w.delegate)
}
