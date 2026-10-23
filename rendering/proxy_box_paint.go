package rendering

import (
	"math"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// DecorationPosition is where a decoration is painted relative to the child.
//
// Flutter: rendering/proxy_box.dart DecorationPosition
type DecorationPosition int

// The values of DecorationPosition follow Flutter's declaration order.
const (
	// DecorationPositionUnspecified is the zero value: the widget's default
	// (background).
	//
	// Flutter: no counterpart
	DecorationPositionUnspecified DecorationPosition = iota
	// DecorationPositionBackground paints the decoration behind the child.
	//
	// Flutter: rendering/proxy_box.dart DecorationPosition.background
	DecorationPositionBackground
	// DecorationPositionForeground paints the decoration in front of the child.
	//
	// Flutter: rendering/proxy_box.dart DecorationPosition.foreground
	DecorationPositionForeground
)

// Name returns the Dart enum name, such as "background".
//
// Flutter: rendering/proxy_box.dart DecorationPosition.name
func (p DecorationPosition) Name() string {
	switch p {
	case DecorationPositionBackground:
		return "background"
	case DecorationPositionForeground:
		return "foreground"
	}
	return "unspecified"
}

// String returns the name Dart prints, such as "DecorationPosition.background".
//
// Flutter: rendering/proxy_box.dart DecorationPosition.toString
func (p DecorationPosition) String() string { return "DecorationPosition." + p.Name() }

// RenderDecoratedBox paints a decoration before or after its child.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox
type RenderDecoratedBox struct {
	RenderProxyBox
	painter       painting.BoxPainter
	decoration    painting.Decoration
	position      DecorationPosition
	configuration painting.ImageConfiguration
}

// RenderDecoratedBoxOpts are the optional parameters of NewRenderDecoratedBox.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.new
type RenderDecoratedBoxOpts struct {
	// Position defaults to DecorationPositionBackground.
	Position DecorationPosition
	// Configuration defaults to painting.ImageConfigurationEmpty.
	Configuration painting.ImageConfiguration
}

// NewRenderDecoratedBox creates the box.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.new
func NewRenderDecoratedBox(decoration painting.Decoration, child Box, opts ...RenderDecoratedBoxOpts) *RenderDecoratedBox {
	o := firstOpts(opts)
	if o.Position == DecorationPositionUnspecified {
		o.Position = DecorationPositionBackground
	}
	r := &RenderDecoratedBox{decoration: decoration, position: o.Position, configuration: o.Configuration}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Decoration is what to paint.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.decoration
func (r *RenderDecoratedBox) Decoration() painting.Decoration { return r.decoration }

// SetDecoration changes the decoration and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.decoration
func (r *RenderDecoratedBox) SetDecoration(value painting.Decoration) {
	if value == r.decoration {
		return
	}
	r.disposePainter()
	r.decoration = value
	r.MarkNeedsPaint()
}

func (r *RenderDecoratedBox) disposePainter() {
	if r.painter != nil {
		r.painter.Dispose()
		r.painter = nil
	}
}

// Position is whether the decoration is painted behind or in front of the child.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.position
func (r *RenderDecoratedBox) Position() DecorationPosition { return r.position }

// SetPosition changes the position and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.position
func (r *RenderDecoratedBox) SetPosition(value DecorationPosition) {
	if value == r.position {
		return
	}
	r.position = value
	r.MarkNeedsPaint()
}

// Configuration is the settings to pass to the decoration when painting.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.configuration
func (r *RenderDecoratedBox) Configuration() painting.ImageConfiguration { return r.configuration }

// SetConfiguration changes the configuration and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.configuration
func (r *RenderDecoratedBox) SetConfiguration(value painting.ImageConfiguration) {
	if value == r.configuration {
		return
	}
	r.configuration = value
	r.MarkNeedsPaint()
}

// Detach disposes the painter.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.detach
func (r *RenderDecoratedBox) Detach() {
	r.disposePainter()
	r.RenderProxyBox.Detach()
	r.MarkNeedsPaint()
}

// Dispose disposes the painter.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.dispose
func (r *RenderDecoratedBox) Dispose() {
	r.disposePainter()
	r.RenderProxyBox.Dispose()
}

// HitTestSelf asks the decoration.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.hitTestSelf
func (r *RenderDecoratedBox) HitTestSelf(position ui.Offset) bool {
	return r.decoration.HitTest(r.Size(), position, r.configuration.TextDirection)
}

func (r *RenderDecoratedBox) paintDecoration(context *PaintingContext, offset ui.Offset, filled painting.ImageConfiguration) {
	saveCount := context.Canvas().GetSaveCount()
	r.painter.Paint(context.Canvas(), offset, filled)
	if foundation.DebugMode && saveCount != context.Canvas().GetSaveCount() {
		panic(errors.New(tagDecorationPainterSaveMismatch, foundation.RuntimeTypeName(r.decoration)+
			" painter had mismatching save and restore calls; every call to save or saveLayer must be matched by a call to restore",
			"saveCountBefore", saveCount, "saveCountAfter", context.Canvas().GetSaveCount(), "decoration", r.decoration))
	}
	if r.decoration.IsComplex() {
		context.SetIsComplexHint()
	}
}

// Paint paints the decoration and the child.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.paint
func (r *RenderDecoratedBox) Paint(context *PaintingContext, offset ui.Offset) {
	if r.painter == nil {
		r.painter = r.decoration.CreateBoxPainter(r.MarkNeedsPaint)
	}
	filled := r.configuration.CopyWith(painting.ImageConfigurationCopy{Size: foundation.Some(r.Size())})
	if r.position == DecorationPositionBackground {
		r.paintDecoration(context, offset, filled)
	}
	r.RenderProxyBox.Paint(context, offset)
	if r.position == DecorationPositionForeground {
		r.paintDecoration(context, offset, filled)
	}
}

// DebugFillProperties adds the decoration and the configuration.
//
// Flutter: rendering/proxy_box.dart RenderDecoratedBox.debugFillProperties
func (r *RenderDecoratedBox) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.ToDiagnosticsNode(r.decoration, foundation.DiagnosticsNodeOpts{Name: foundation.Some("decoration")}))
	b.Add(foundation.NewDiagnosticsProperty("configuration", r.configuration))
}

// RenderTransform transforms its child using a matrix before painting it.
//
// Flutter: rendering/proxy_box.dart RenderTransform
type RenderTransform struct {
	RenderProxyBox
	// TransformHitTests says whether hit tests are transformed too.
	//
	// Flutter: rendering/proxy_box.dart RenderTransform.transformHitTests
	TransformHitTests bool

	origin        foundation.Opt[ui.Offset]
	alignment     painting.AlignmentGeometry
	textDirection ui.TextDirection
	transform     *ui.Matrix4
	filterQuality foundation.Opt[ui.FilterQuality]
}

// RenderTransformOpts are the optional parameters of NewRenderTransform.
//
// Flutter: rendering/proxy_box.dart RenderTransform.new
type RenderTransformOpts struct {
	// Origin is the point about which to rotate or scale the transform, in the
	// coordinates of the box.
	Origin foundation.Opt[ui.Offset]
	// Alignment is the same as Origin, but aligned in the box.
	Alignment painting.AlignmentGeometry
	// TextDirection resolves a directional Alignment.
	TextDirection ui.TextDirection
	// TransformHitTests defaults to true.
	TransformHitTests foundation.Opt[bool]
	// FilterQuality applies the transform as an image filter of this quality;
	// none (the default) transforms the canvas.
	FilterQuality foundation.Opt[ui.FilterQuality]
}

// NewRenderTransform creates the box with a copy of the matrix.
//
// Flutter: rendering/proxy_box.dart RenderTransform.new
func NewRenderTransform(transform *ui.Matrix4, child Box, opts ...RenderTransformOpts) *RenderTransform {
	o := firstOpts(opts)
	r := &RenderTransform{TransformHitTests: o.TransformHitTests.OrElse(true), origin: o.Origin, alignment: o.Alignment,
		textDirection: o.TextDirection, transform: transform.Clone(), filterQuality: o.FilterQuality}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Origin is the origin of the coordinate system for the transform.
//
// Flutter: rendering/proxy_box.dart RenderTransform.origin
func (r *RenderTransform) Origin() foundation.Opt[ui.Offset] { return r.origin }

// SetOrigin changes the origin and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderTransform.origin
func (r *RenderTransform) SetOrigin(value foundation.Opt[ui.Offset]) {
	if r.origin == value {
		return
	}
	r.origin = value
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

// Alignment is the alignment of the origin in the box; nil for none.
//
// Flutter: rendering/proxy_box.dart RenderTransform.alignment
func (r *RenderTransform) Alignment() painting.AlignmentGeometry { return r.alignment }

// SetAlignment changes the alignment and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderTransform.alignment
func (r *RenderTransform) SetAlignment(value painting.AlignmentGeometry) {
	if (r.alignment == nil && value == nil) || (r.alignment != nil && value != nil && r.alignment.Equal(value)) {
		return
	}
	r.alignment = value
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

// TextDirection resolves a directional alignment.
//
// Flutter: rendering/proxy_box.dart RenderTransform.textDirection
func (r *RenderTransform) TextDirection() ui.TextDirection { return r.textDirection }

// SetTextDirection changes the direction and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderTransform.textDirection
func (r *RenderTransform) SetTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

// AlwaysNeedsCompositing is true if there is a child and a filter quality.
//
// Flutter: rendering/proxy_box.dart RenderTransform.alwaysNeedsCompositing
func (r *RenderTransform) AlwaysNeedsCompositing() bool {
	return r.hasChild() && r.filterQuality.IsSome()
}

// SetTransform changes the matrix (a copy is kept) and marks the box as needing
// paint.
//
// Flutter: rendering/proxy_box.dart RenderTransform.transform
func (r *RenderTransform) SetTransform(value *ui.Matrix4) {
	if r.transform.Equals(value) {
		return
	}
	r.transform = value.Clone()
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

// FilterQuality is the quality of the filter that applies the transform.
//
// Flutter: rendering/proxy_box.dart RenderTransform.filterQuality
func (r *RenderTransform) FilterQuality() foundation.Opt[ui.FilterQuality] { return r.filterQuality }

// SetFilterQuality changes the quality and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderTransform.filterQuality
func (r *RenderTransform) SetFilterQuality(value foundation.Opt[ui.FilterQuality]) {
	if r.filterQuality == value {
		return
	}
	didNeedCompositing := r.AlwaysNeedsCompositing()
	r.filterQuality = value
	if didNeedCompositing != r.AlwaysNeedsCompositing() {
		r.MarkNeedsCompositingBitsUpdate()
	}
	r.MarkNeedsPaint()
}

func (r *RenderTransform) changed() {
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

// SetIdentity sets the transform to the identity matrix.
//
// Flutter: rendering/proxy_box.dart RenderTransform.setIdentity
func (r *RenderTransform) SetIdentity() { r.transform.SetIdentity(); r.changed() }

// RotateX concatenates a rotation about the x axis.
//
// Flutter: rendering/proxy_box.dart RenderTransform.rotateX
func (r *RenderTransform) RotateX(radians float64) { r.transform.RotateX(radians); r.changed() }

// RotateY concatenates a rotation about the y axis.
//
// Flutter: rendering/proxy_box.dart RenderTransform.rotateY
func (r *RenderTransform) RotateY(radians float64) { r.transform.RotateY(radians); r.changed() }

// RotateZ concatenates a rotation about the z axis.
//
// Flutter: rendering/proxy_box.dart RenderTransform.rotateZ
func (r *RenderTransform) RotateZ(radians float64) { r.transform.RotateZ(radians); r.changed() }

// Translate concatenates a translation by (x, y, z).
//
// Flutter: rendering/proxy_box.dart RenderTransform.translate
func (r *RenderTransform) Translate(x, y, z float64) {
	r.transform.Translate(float32(x), float32(y), float32(z))
	r.changed()
}

// Scale concatenates a scale by (x, y, z).
//
// Flutter: rendering/proxy_box.dart RenderTransform.scale
func (r *RenderTransform) Scale(x, y, z float64) {
	r.transform.Scale(float32(x), float32(y), float32(z))
	r.changed()
}

// effectiveTransform is the matrix, moved to the origin and the alignment.
func (r *RenderTransform) effectiveTransform() *ui.Matrix4 {
	var resolved foundation.Opt[painting.Alignment]
	if r.alignment != nil {
		resolved = foundation.Some(r.alignment.Resolve(r.textDirection))
	}
	if r.origin.IsNone() && resolved.IsNone() {
		return r.transform
	}
	result := ui.NewMatrix4Identity()
	if o, ok := r.origin.Get(); ok {
		result.Translate(float32(o.Dx), float32(o.Dy), 0)
	}
	var translation ui.Offset
	if a, ok := resolved.Get(); ok {
		translation = a.AlongSize(r.Size())
		result.Translate(float32(translation.Dx), float32(translation.Dy), 0)
	}
	result.MultiplyBy(r.transform)
	if resolved.IsSome() {
		result.Translate(float32(-translation.Dx), float32(-translation.Dy), 0)
	}
	if o, ok := r.origin.Get(); ok {
		result.Translate(float32(-o.Dx), float32(-o.Dy), 0)
	}
	return result
}

// HitTest hit tests the children without checking the box's own size.
//
// Flutter: rendering/proxy_box.dart RenderTransform.hitTest
func (r *RenderTransform) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	return r.self.(Box).HitTestChildren(result, position)
}

// HitTestChildren hit tests the child in the transformed coordinates.
//
// Flutter: rendering/proxy_box.dart RenderTransform.hitTestChildren
func (r *RenderTransform) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	var transform *ui.Matrix4
	if r.TransformHitTests {
		transform = r.effectiveTransform()
	}
	return result.AddWithPaintTransform(transform, position, func(result *BoxHitTestResult, position ui.Offset) bool {
		return r.RenderProxyBox.HitTestChildren(result, position)
	})
}

// Paint paints the child transformed.
//
// Flutter: rendering/proxy_box.dart RenderTransform.paint
func (r *RenderTransform) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() {
		return
	}
	transform := r.effectiveTransform()
	quality, hasQuality := r.filterQuality.Get()
	if !hasQuality {
		if childOffset, ok := transform.AsTranslation(); ok {
			r.RenderProxyBox.Paint(context, offset.Add(childOffset))
			r.SetLayer(nil)
			return
		}
		det := transform.Determinant()
		if det == 0 || math.IsInf(det, 0) || math.IsNaN(det) {
			r.SetLayer(nil)
			return
		}
		setObjectLayer(&r.RenderObjectBase, context.PushTransform(r.NeedsCompositing(), offset, transform, r.RenderProxyBox.Paint,
			layerAs[*TransformLayer](r.Layer())))
		return
	}
	effective := ui.NewMatrix4TranslationValues(float32(offset.Dx), float32(offset.Dy), 0)
	effective.MultiplyBy(transform)
	effective.Translate(float32(-offset.Dx), float32(-offset.Dy), 0)
	filter := ui.NewImageFilterMatrix(effective, ui.ImageFilterMatrixOpts{FilterQuality: quality})
	if layer := layerAs[*ImageFilterLayer](r.Layer()); layer != nil {
		layer.SetImageFilter(filter)
	} else {
		r.SetLayer(NewImageFilterLayer(ImageFilterLayerOpts{ImageFilter: filter}))
	}
	context.PushLayer(r.Layer(), r.RenderProxyBox.Paint, offset, foundation.None[ui.Rect]())
}

// ApplyPaintTransform applies the effective transform.
//
// Flutter: rendering/proxy_box.dart RenderTransform.applyPaintTransform
func (r *RenderTransform) ApplyPaintTransform(_ RenderObject, transform *ui.Matrix4) {
	transform.MultiplyBy(r.effectiveTransform())
}

// DebugFillProperties adds the transform and its settings.
//
// Flutter: rendering/proxy_box.dart RenderTransform.debugFillProperties
func (r *RenderTransform) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(painting.NewTransformProperty("transform matrix", r.transform))
	b.Add(foundation.NewDiagnosticsPropertyOpt("origin", r.origin))
	if r.alignment != nil {
		b.Add(foundation.NewDiagnosticsProperty("alignment", r.alignment))
	} else {
		b.Add(foundation.NewDiagnosticsProperty[painting.AlignmentGeometry]("alignment", nil))
	}
	b.Add(foundation.NewEnumProperty("textDirection", foundation.Some(r.textDirection), foundation.EnumPropertyOpts{DefaultValue: ui.TextDirectionUnspecified}))
	b.Add(foundation.NewDiagnosticsProperty("transformHitTests", r.TransformHitTests))
}

// RenderFittedBox scales and positions its child within itself according to a
// fit.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox
type RenderFittedBox struct {
	RenderProxyBox
	fit               painting.BoxFit
	alignment         painting.AlignmentGeometry
	textDirection     ui.TextDirection
	clipBehavior      ui.Clip
	resolvedAlignment foundation.Opt[painting.Alignment]
	hasVisualOverflow bool
	transform         *ui.Matrix4
}

// RenderFittedBoxOpts are the optional parameters of NewRenderFittedBox.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.new
type RenderFittedBoxOpts struct {
	// Fit defaults to BoxFitContain.
	Fit painting.BoxFit
	// Alignment defaults to painting.AlignmentCenter.
	Alignment painting.AlignmentGeometry
	// TextDirection resolves a directional alignment.
	TextDirection ui.TextDirection
	// ClipBehavior defaults to ClipNone.
	ClipBehavior ui.Clip
}

// NewRenderFittedBox creates the box.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.new
func NewRenderFittedBox(child Box, opts ...RenderFittedBoxOpts) *RenderFittedBox {
	o := firstOpts(opts)
	if o.Fit == painting.BoxFitUnspecified {
		o.Fit = painting.BoxFitContain
	}
	if o.Alignment == nil {
		o.Alignment = painting.AlignmentCenter
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	r := &RenderFittedBox{fit: o.Fit, alignment: o.Alignment, textDirection: o.TextDirection, clipBehavior: o.ClipBehavior}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

func (r *RenderFittedBox) resolve() painting.Alignment {
	if !r.resolvedAlignment.IsSome() {
		r.resolvedAlignment = foundation.Some(r.alignment.Resolve(r.textDirection))
	}
	return r.resolvedAlignment.Value()
}

func (r *RenderFittedBox) markNeedResolution() {
	r.resolvedAlignment = foundation.None[painting.Alignment]()
	r.MarkNeedsPaint()
}

func fitAffectsLayout(fit painting.BoxFit) bool { return fit == painting.BoxFitScaleDown }

// Fit is how the child is inscribed into the available space.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.fit
func (r *RenderFittedBox) Fit() painting.BoxFit { return r.fit }

// SetFit changes the fit; layout is needed if either fit is scaleDown.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.fit
func (r *RenderFittedBox) SetFit(value painting.BoxFit) {
	if r.fit == value {
		return
	}
	last := r.fit
	r.fit = value
	if fitAffectsLayout(last) || fitAffectsLayout(value) {
		r.MarkNeedsLayout()
	} else {
		r.clearPaintData()
		r.MarkNeedsPaint()
	}
}

// Alignment is how to align the child within its parent's bounds.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.alignment
func (r *RenderFittedBox) Alignment() painting.AlignmentGeometry { return r.alignment }

// SetAlignment changes the alignment and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.alignment
func (r *RenderFittedBox) SetAlignment(value painting.AlignmentGeometry) {
	if r.alignment.Equal(value) {
		return
	}
	r.alignment = value
	r.clearPaintData()
	r.markNeedResolution()
}

// TextDirection resolves a directional alignment.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.textDirection
func (r *RenderFittedBox) TextDirection() ui.TextDirection { return r.textDirection }

// SetTextDirection changes the direction and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.textDirection
func (r *RenderFittedBox) SetTextDirection(value ui.TextDirection) {
	if r.textDirection == value {
		return
	}
	r.textDirection = value
	r.clearPaintData()
	r.markNeedResolution()
}

// ClipBehavior is how the overflow is clipped.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.clipBehavior
func (r *RenderFittedBox) ClipBehavior() ui.Clip { return r.clipBehavior }

// SetClipBehavior changes the clip behavior and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.clipBehavior
func (r *RenderFittedBox) SetClipBehavior(value ui.Clip) {
	if value != r.clipBehavior {
		r.clipBehavior = value
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	}
}

func (r *RenderFittedBox) sizeFor(constraints BoxConstraints, childSize ui.Size) ui.Size {
	if r.fit == painting.BoxFitScaleDown {
		unconstrained := constraints.Loosen().ConstrainSizeAndAttemptToPreserveAspectRatio(childSize)
		return constraints.Constrain(unconstrained)
	}
	return constraints.ConstrainSizeAndAttemptToPreserveAspectRatio(childSize)
}

// ComputeDryLayout is the size for the child's unconstrained size.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.computeDryLayout
func (r *RenderFittedBox) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	if !r.hasChild() {
		return constraints.Smallest()
	}
	return r.sizeFor(constraints, r.Child().GetDryLayout(BoxConstraints{
		MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)}))
}

// ComputeDryBaseline is the child's baseline for unconstrained constraints.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.computeDryBaseline
func (r *RenderFittedBox) ComputeDryBaseline(_ BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	return r.Child().GetDryBaseline(BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)}, baseline)
}

// PerformLayout lays the child out unconstrained and sizes the box to fit it.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.performLayout
func (r *RenderFittedBox) PerformLayout() {
	if !r.hasChild() {
		r.SetSize(r.BoxConstraints().Smallest())
		return
	}
	c := r.Child()
	c.Layout(BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)}, LayoutOpts{ParentUsesSize: true})
	r.SetSize(r.sizeFor(r.BoxConstraints(), c.Size()))
	r.clearPaintData()
}

func (r *RenderFittedBox) clearPaintData() {
	r.hasVisualOverflow = false
	r.transform = nil
}

func (r *RenderFittedBox) updatePaintData() {
	if r.transform != nil {
		return
	}
	if !r.hasChild() {
		r.hasVisualOverflow = false
		r.transform = ui.NewMatrix4Identity()
		return
	}
	resolved := r.resolve()
	childSize := r.Child().Size()
	sizes := painting.ApplyBoxFit(r.fit, childSize, r.Size())
	scaleX := sizes.Destination.Width / sizes.Source.Width
	scaleY := sizes.Destination.Height / sizes.Source.Height
	sourceRect := resolved.Inscribe(sizes.Source, ui.OffsetZero.And(childSize))
	destinationRect := resolved.Inscribe(sizes.Destination, ui.OffsetZero.And(r.Size()))
	r.hasVisualOverflow = sourceRect.Width() < childSize.Width || sourceRect.Height() < childSize.Height
	t := ui.NewMatrix4TranslationValues(float32(destinationRect.Left), float32(destinationRect.Top), 0)
	t.Scale(float32(scaleX), float32(scaleY), 1)
	t.Translate(float32(-sourceRect.Left), float32(-sourceRect.Top), 0)
	r.transform = t
}

func (r *RenderFittedBox) paintChildWithTransform(context *PaintingContext, offset ui.Offset) *TransformLayer {
	if childOffset, ok := r.transform.AsTranslation(); ok {
		r.RenderProxyBox.Paint(context, offset.Add(childOffset))
		return nil
	}
	return context.PushTransform(r.NeedsCompositing(), offset, r.transform, r.RenderProxyBox.Paint,
		layerAs[*TransformLayer](r.Layer()))
}

// Paint paints the child scaled and positioned.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.paint
func (r *RenderFittedBox) Paint(context *PaintingContext, offset ui.Offset) {
	if !r.hasChild() || r.Size().IsEmpty() || r.Child().Size().IsEmpty() {
		return
	}
	r.updatePaintData()
	if r.hasVisualOverflow && r.clipBehavior != ui.ClipNone {
		setObjectLayer(&r.RenderObjectBase, context.PushClipRect(r.NeedsCompositing(), offset, ui.OffsetZero.And(r.Size()),
			func(context *PaintingContext, offset ui.Offset) { r.paintChildWithTransform(context, offset) },
			PushClipRectOpts{ClipBehavior: r.clipBehavior, OldLayer: layerAs[*ClipRectLayer](r.Layer())}))
		return
	}
	setObjectLayer(&r.RenderObjectBase, r.paintChildWithTransform(context, offset))
}

// HitTestChildren hit tests the child in the fitted coordinates.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.hitTestChildren
func (r *RenderFittedBox) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	if r.Size().IsEmpty() || (r.hasChild() && r.Child().Size().IsEmpty()) {
		return false
	}
	r.updatePaintData()
	return result.AddWithPaintTransform(r.transform, position, func(result *BoxHitTestResult, position ui.Offset) bool {
		return r.RenderProxyBox.HitTestChildren(result, position)
	})
}

// PaintsChild is true unless the box or the child is empty.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.paintsChild
func (r *RenderFittedBox) PaintsChild(child RenderObject) bool {
	if foundation.DebugMode && child.Parent() != RenderObject(r.self) {
		panic(errors.New(tagRenderObjectNotChild, "PaintsChild was called with a render object that is not a child",
			"object", foundation.DescribeIdentity(r.self), "child", foundation.DescribeIdentity(child)))
	}
	return !r.Size().IsEmpty() && !child.(Box).Size().IsEmpty()
}

// ApplyPaintTransform applies the fitting transform.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.applyPaintTransform
func (r *RenderFittedBox) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	if !r.PaintsChild(child) {
		transform.SetZero()
		return
	}
	r.updatePaintData()
	transform.MultiplyBy(r.transform)
}

// DebugFillProperties adds the fit and the alignment.
//
// Flutter: rendering/proxy_box.dart RenderFittedBox.debugFillProperties
func (r *RenderFittedBox) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewEnumProperty("fit", foundation.Some(r.fit)))
	b.Add(foundation.NewDiagnosticsProperty("alignment", r.alignment))
	b.Add(foundation.NewEnumProperty("textDirection", foundation.Some(r.textDirection), foundation.EnumPropertyOpts{DefaultValue: ui.TextDirectionUnspecified}))
}

// RenderFractionalTranslation translates its child's painting by a fraction of
// its size.
//
// Flutter: rendering/proxy_box.dart RenderFractionalTranslation
type RenderFractionalTranslation struct {
	RenderProxyBox
	// TransformHitTests says whether hit tests are translated too.
	//
	// Flutter: rendering/proxy_box.dart RenderFractionalTranslation.transformHitTests
	TransformHitTests bool

	translation ui.Offset
}

// RenderFractionalTranslationOpts are the optional parameters of
// NewRenderFractionalTranslation.
//
// Flutter: rendering/proxy_box.dart RenderFractionalTranslation.new
type RenderFractionalTranslationOpts struct {
	// TransformHitTests defaults to true.
	TransformHitTests foundation.Opt[bool]
}

// NewRenderFractionalTranslation creates the box.
//
// Flutter: rendering/proxy_box.dart RenderFractionalTranslation.new
func NewRenderFractionalTranslation(translation ui.Offset, child Box, opts ...RenderFractionalTranslationOpts) *RenderFractionalTranslation {
	o := firstOpts(opts)
	r := &RenderFractionalTranslation{translation: translation, TransformHitTests: o.TransformHitTests.OrElse(true)}
	InitRenderProxyBox(&r.RenderProxyBox, r, child)
	return r
}

// Translation is the translation, as a fraction of the size of the box.
//
// Flutter: rendering/proxy_box.dart RenderFractionalTranslation.translation
func (r *RenderFractionalTranslation) Translation() ui.Offset { return r.translation }

// SetTranslation changes the translation and marks the box as needing paint.
//
// Flutter: rendering/proxy_box.dart RenderFractionalTranslation.translation
func (r *RenderFractionalTranslation) SetTranslation(value ui.Offset) {
	if r.translation == value {
		return
	}
	r.translation = value
	r.MarkNeedsPaint()
	r.MarkNeedsSemanticsUpdate()
}

// HitTest hit tests the children without checking the box's own size.
//
// Flutter: rendering/proxy_box.dart RenderFractionalTranslation.hitTest
func (r *RenderFractionalTranslation) HitTest(result *BoxHitTestResult, position ui.Offset) bool {
	return r.self.(Box).HitTestChildren(result, position)
}

// HitTestChildren hit tests the child in the translated coordinates.
//
// Flutter: rendering/proxy_box.dart RenderFractionalTranslation.hitTestChildren
func (r *RenderFractionalTranslation) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	var offset *ui.Offset
	if r.TransformHitTests {
		o := ui.Offset{Dx: r.translation.Dx * r.Size().Width, Dy: r.translation.Dy * r.Size().Height}
		offset = &o
	}
	return result.AddWithPaintOffset(offset, position, func(result *BoxHitTestResult, position ui.Offset) bool {
		return r.RenderProxyBox.HitTestChildren(result, position)
	})
}

// Paint paints the child translated.
//
// Flutter: rendering/proxy_box.dart RenderFractionalTranslation.paint
func (r *RenderFractionalTranslation) Paint(context *PaintingContext, offset ui.Offset) {
	if r.hasChild() {
		r.RenderProxyBox.Paint(context, ui.Offset{
			Dx: offset.Dx + r.translation.Dx*r.Size().Width, Dy: offset.Dy + r.translation.Dy*r.Size().Height})
	}
}

// ApplyPaintTransform translates by the fraction of the size.
//
// Flutter: rendering/proxy_box.dart RenderFractionalTranslation.applyPaintTransform
func (r *RenderFractionalTranslation) ApplyPaintTransform(_ RenderObject, transform *ui.Matrix4) {
	transform.Translate(float32(r.translation.Dx*r.Size().Width), float32(r.translation.Dy*r.Size().Height), 0)
}

// DebugFillProperties adds the translation.
//
// Flutter: rendering/proxy_box.dart RenderFractionalTranslation.debugFillProperties
func (r *RenderFractionalTranslation) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderProxyBox.DebugFillProperties(b)
	b.Add(foundation.NewDiagnosticsProperty("translation", r.translation))
	b.Add(foundation.NewDiagnosticsProperty("transformHitTests", r.TransformHitTests))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDecorationPainterSaveMismatch = "DecorationPainterSaveMismatch"
)
