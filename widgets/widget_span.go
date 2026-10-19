package widgets

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// WidgetSpan is an immutable placeholder that is embedded inline within text: a
// widget laid out in the line of text. Use it as a child of RichText or Text.
//
// Flutter: widgets/widget_span.dart WidgetSpan
type WidgetSpan struct {
	painting.PlaceholderSpanBase
	child Widget
}

// WidgetSpanOpts are the optional parameters of NewWidgetSpan.
//
// Flutter: widgets/widget_span.dart WidgetSpan.new
type WidgetSpanOpts struct {
	// Alignment is how the widget aligns with the text; the default is
	// PlaceholderAlignmentBottom.
	Alignment ui.PlaceholderAlignment
	// Baseline is the text baseline to align to when Alignment is baseline,
	// aboveBaseline or belowBaseline; it is required then.
	Baseline ui.TextBaseline
	// Style is the style of the span.
	Style *painting.TextStyle
}

// NewWidgetSpan creates the span. In debug builds it panics with the tag
// WidgetSpanNoBaseline when the alignment is relative to the baseline but there
// is no baseline.
//
// Flutter: widgets/widget_span.dart WidgetSpan.new
func NewWidgetSpan(child Widget, opts ...WidgetSpanOpts) *WidgetSpan {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Baseline == ui.TextBaselineUnspecified &&
		(o.Alignment == ui.PlaceholderAlignmentAboveBaseline || o.Alignment == ui.PlaceholderAlignmentBelowBaseline ||
			o.Alignment == ui.PlaceholderAlignmentBaseline) {
		panic(errors.New("WidgetSpanNoBaseline", "a WidgetSpan aligned to the baseline needs a baseline", "alignment", o.Alignment.String()))
	}
	return &WidgetSpan{painting.NewPlaceholderSpanBase(painting.PlaceholderSpanOpts{Alignment: o.Alignment, Baseline: o.Baseline, Style: o.Style}), child}
}

// Child is the widget to embed.
//
// Flutter: widgets/widget_span.dart WidgetSpan.child
func (s *WidgetSpan) Child() Widget { return s.child }

// ExtractWidgetsFromInlineSpan makes the children of a RichText for the widget
// spans in the span and its descendants, in order: each is wrapped to say which
// span it belongs to and to scale it with the text.
//
// Flutter: widgets/widget_span.dart WidgetSpan.extractFromInlineSpan
func ExtractWidgetsFromInlineSpan(span painting.InlineSpan, textScaler painting.TextScaler) []Widget {
	var widgets []Widget
	fontSizeStack := []float64{painting.DefaultFontSize}
	index := 0
	var visit painting.InlineSpanVisitor
	visit = func(span painting.InlineSpan) bool {
		pushed := false
		if style := span.Style(); style != nil {
			if size, ok := style.FontSize.Get(); ok && size != fontSizeStack[len(fontSizeStack)-1] {
				fontSizeStack = append(fontSizeStack, size)
				pushed = true
			}
		}
		if ws, ok := span.(*WidgetSpan); ok {
			fontSize := fontSizeStack[len(fontSizeStack)-1]
			factor := 0.0
			if fontSize != 0 {
				factor = textScaler.Scale(fontSize) / fontSize
			}
			widgets = append(widgets, &widgetSpanParentData{
				ParentDataWidgetOf: NewParentDataWidgetBase[*rendering.TextParentData](
					NewSemantics(&autoScaleInlineWidget{
						SingleChildRenderObjectWidget: NewSingleChildRenderObjectWidgetBase(ws.child),
						span:                          ws, textScaleFactor: factor,
					}, SemanticsOpts{TagForChildren: foundation.Some(rendering.NewPlaceholderSpanIndexSemanticsTag(index))})),
				span: ws,
			})
			index++
		} else if foundation.DebugMode {
			if _, isPlaceholder := span.(painting.PlaceholderSpan); isPlaceholder {
				panic(errors.New("PlaceholderSpanNotWidgetSpan", "a PlaceholderSpan that is not a WidgetSpan is not supported"))
			}
		}
		span.VisitDirectChildren(visit)
		if pushed {
			fontSizeStack = fontSizeStack[:len(fontSizeStack)-1]
		}
		return true
	}
	visit(span)
	return widgets
}

// Build adds the placeholder, with the style of the span, to the builder. The
// dimensions in the options must have the size of every placeholder.
//
// Flutter: widgets/widget_span.dart WidgetSpan.build
func (s *WidgetSpan) Build(builder *ui.ParagraphBuilder, opts ...painting.SpanBuildOpts) {
	var o painting.SpanBuildOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.TextScaler == nil {
		o.TextScaler = painting.TextScalerNoScaling
	}
	if o.Dimensions == nil {
		panic(errors.New("WidgetSpanNoDimensions", "a WidgetSpan needs the dimensions of the placeholders to build"))
	}
	style := s.Style()
	if style != nil {
		builder.PushStyle(style.GetTextStyle(painting.GetTextStyleOpts{TextScaler: o.TextScaler}))
	}
	if builder.PlaceholderCount() >= len(o.Dimensions) {
		panic(errors.New("WidgetSpanDimensionsMissing", "there are fewer placeholder dimensions than placeholders",
			"placeholders", builder.PlaceholderCount(), "dimensions", len(o.Dimensions)))
	}
	d := o.Dimensions[builder.PlaceholderCount()]
	po := ui.AddPlaceholderOpts{Baseline: d.Baseline}
	if offset, ok := d.BaselineOffset.Get(); ok {
		po.BaselineOffset = &offset
	}
	builder.AddPlaceholder(d.Size.Width, d.Size.Height, s.Alignment(), po)
	if style != nil {
		builder.Pop()
	}
}

// VisitChildren calls the visitor on this span.
//
// Flutter: widgets/widget_span.dart WidgetSpan.visitChildren
func (s *WidgetSpan) VisitChildren(visitor painting.InlineSpanVisitor) bool { return visitor(s) }

// VisitDirectChildren does nothing: a widget span has no child spans.
//
// Flutter: widgets/widget_span.dart WidgetSpan.visitDirectChildren
func (s *WidgetSpan) VisitDirectChildren(visitor painting.InlineSpanVisitor) bool { return true }

// GetSpanForPositionVisitor returns this span if the position is at it.
//
// Flutter: widgets/widget_span.dart WidgetSpan.getSpanForPositionVisitor
func (s *WidgetSpan) GetSpanForPositionVisitor(position ui.TextPosition, offset *painting.Accumulator) painting.InlineSpan {
	if position.Offset == offset.Value() {
		return s
	}
	offset.Increment(1)
	return nil
}

// CodeUnitAtVisitor is the placeholder code unit when the index is at this span.
//
// Flutter: widgets/widget_span.dart WidgetSpan.codeUnitAtVisitor
func (s *WidgetSpan) CodeUnitAtVisitor(index int, offset *painting.Accumulator) (byte, bool) {
	local := index - offset.Value()
	offset.Increment(1)
	if local == 0 {
		// The code units of painting are UTF-8 bytes: this is the first one of U+FFFC.
		return 0xEF, true
	}
	return 0, false
}

// CompareTo describes the difference between this span and another.
//
// Flutter: widgets/widget_span.dart WidgetSpan.compareTo
func (s *WidgetSpan) CompareTo(other painting.InlineSpan) painting.RenderComparison {
	if painting.InlineSpan(s) == other {
		return painting.RenderComparisonIdentical
	}
	o, ok := other.(*WidgetSpan)
	if !ok || (s.Style() == nil) != (o.Style() == nil) || s.child != o.child || s.Alignment() != o.Alignment() {
		return painting.RenderComparisonLayout
	}
	if s.Style() != nil {
		return s.Style().CompareTo(o.Style())
	}
	return painting.RenderComparisonIdentical
}

// Equal is Dart's == on spans.
//
// Flutter: widgets/widget_span.dart WidgetSpan.operator ==
func (s *WidgetSpan) Equal(other painting.InlineSpan) bool {
	if painting.InlineSpan(s) == other {
		return true
	}
	o, ok := other.(*WidgetSpan)
	return ok && s.EqualBase(o) && o.child == s.child && o.Alignment() == s.Alignment() && o.Baseline() == s.Baseline()
}

// DebugAssertIsValid is always true: the constructor already checked the span.
//
// Flutter: widgets/widget_span.dart WidgetSpan.debugAssertIsValid
func (s *WidgetSpan) DebugAssertIsValid() bool { return true }

// ToStringShort names the span.
//
// Flutter: widgets/widget_span.dart WidgetSpan.toStringShort
func (s *WidgetSpan) ToStringShort() string { return "WidgetSpan" }

// String describes the span.
//
// Flutter: widgets/widget_span.dart WidgetSpan.toString
func (s *WidgetSpan) String() string { return foundation.DiagnosticableString(s) }

// DebugFillProperties adds the properties of the placeholder and the widget.
//
// Flutter: widgets/widget_span.dart WidgetSpan.debugFillProperties
func (s *WidgetSpan) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	s.DebugFillBase(p)
	p.Add(foundation.NewDiagnosticsProperty("widget", s.child))
}

// DebugDescribeChildren has no children.
//
// Flutter: widgets/widget_span.dart WidgetSpan.debugDescribeChildren
func (s *WidgetSpan) DebugDescribeChildren() []foundation.DiagnosticsNode { return nil }

// widgetSpanParentData sets TextParentData.span.
type widgetSpanParentData struct {
	ParentDataWidgetOf[*rendering.TextParentData]
	span *WidgetSpan
}

func (w *widgetSpanParentData) ApplyParentData(renderObject rendering.RenderObject) {
	renderObject.ParentData().(*rendering.TextParentData).Span = w.span
}

func (*widgetSpanParentData) DebugTypicalAncestorWidgetClass() string { return "RichText" }

// autoScaleInlineWidget applies text scaling to an inline widget.
type autoScaleInlineWidget struct {
	SingleChildRenderObjectWidget
	span            *WidgetSpan
	textScaleFactor float64
}

func (w *autoScaleInlineWidget) CreateRenderObject(BuildContext) rendering.RenderObject {
	return newRenderScaledInlineWidget(w.span.Alignment(), w.span.Baseline(), w.textScaleFactor)
}

func (w *autoScaleInlineWidget) UpdateRenderObject(_ BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderScaledInlineWidget)
	r.setAlignment(w.span.Alignment())
	r.setBaseline(w.span.Baseline())
	r.setScale(w.textScaleFactor)
}

// renderScaledInlineWidget scales its child with the text and lays it out with
// the width of the paragraph.
type renderScaledInlineWidget struct {
	rendering.RenderBox
	rendering.RenderObjectWithChild[rendering.Box]
	alignment      ui.PlaceholderAlignment
	baseline       ui.TextBaseline
	scale          float64
	transformLayer *rendering.LayerHandle[*rendering.TransformLayer]
}

func newRenderScaledInlineWidget(alignment ui.PlaceholderAlignment, baseline ui.TextBaseline, scale float64) *renderScaledInlineWidget {
	r := &renderScaledInlineWidget{alignment: alignment, baseline: baseline, scale: scale,
		transformLayer: rendering.NewLayerHandle[*rendering.TransformLayer](nil)}
	rendering.InitRenderBox(&r.RenderBox, r)
	rendering.InitRenderObjectWithChild(&r.RenderObjectWithChild, r)
	return r
}

func (r *renderScaledInlineWidget) setScale(value float64) {
	if value == r.scale {
		return
	}
	if foundation.DebugMode && (value <= 0 || math.IsInf(value, 0) || math.IsNaN(value)) {
		panic(errors.New("InlineScaleInvalid", "the scale of an inline widget must be positive and finite", "scale", value))
	}
	r.scale = value
	r.MarkNeedsLayout()
}

func (r *renderScaledInlineWidget) setAlignment(value ui.PlaceholderAlignment) {
	if r.alignment == value {
		return
	}
	r.alignment = value
	r.MarkNeedsLayout()
}

func (r *renderScaledInlineWidget) setBaseline(value ui.TextBaseline) {
	if r.baseline == value {
		return
	}
	r.baseline = value
	r.MarkNeedsLayout()
}

func (r *renderScaledInlineWidget) ComputeMaxIntrinsicHeight(width float64) float64 {
	if r.Child() == nil {
		return 0
	}
	return r.Child().GetMaxIntrinsicHeight(width/r.scale) * r.scale
}

func (r *renderScaledInlineWidget) ComputeMaxIntrinsicWidth(height float64) float64 {
	if r.Child() == nil {
		return 0
	}
	return r.Child().GetMaxIntrinsicWidth(height/r.scale) * r.scale
}

func (r *renderScaledInlineWidget) ComputeMinIntrinsicHeight(width float64) float64 {
	if r.Child() == nil {
		return 0
	}
	return r.Child().GetMinIntrinsicHeight(width/r.scale) * r.scale
}

func (r *renderScaledInlineWidget) ComputeMinIntrinsicWidth(height float64) float64 {
	if r.Child() == nil {
		return 0
	}
	return r.Child().GetMinIntrinsicWidth(height/r.scale) * r.scale
}

func (r *renderScaledInlineWidget) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) foundation.Opt[float64] {
	if r.Child() != nil {
		if d, ok := r.Child().GetDistanceToActualBaseline(baseline).Get(); ok {
			return foundation.Some(r.scale * d)
		}
	}
	return r.RenderBox.ComputeDistanceToActualBaseline(baseline)
}

func (r *renderScaledInlineWidget) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if r.Child() == nil {
		return foundation.None[float64]()
	}
	d, ok := r.Child().GetDryBaseline(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(constraints.MaxWidth / r.scale)}), baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	return foundation.Some(r.scale * d)
}

func (r *renderScaledInlineWidget) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	if foundation.DebugMode && constraints.HasBoundedHeight() {
		panic(errors.New("InlineScaleBoundedHeight", "an inline widget must be laid out with unbounded height"))
	}
	var unscaled ui.Size
	if r.Child() != nil {
		unscaled = r.Child().GetDryLayout(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(constraints.MaxWidth / r.scale)}))
	}
	return constraints.Constrain(unscaled.Mul(r.scale))
}

func (r *renderScaledInlineWidget) PerformLayout() {
	child := r.Child()
	if child == nil {
		return
	}
	constraints := r.BoxConstraints()
	if foundation.DebugMode && constraints.HasBoundedHeight() {
		panic(errors.New("InlineScaleBoundedHeight", "an inline widget must be laid out with unbounded height"))
	}
	child.Layout(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(constraints.MaxWidth / r.scale)}),
		rendering.LayoutOpts{ParentUsesSize: true})
	r.SetSize(constraints.Constrain(child.Size().Mul(r.scale)))
}

func (r *renderScaledInlineWidget) ApplyPaintTransform(child rendering.RenderObject, transform *ui.Matrix4) {
	transform.Scale(float32(r.scale), float32(r.scale), float32(r.scale))
}

func (r *renderScaledInlineWidget) paintChild(context *rendering.PaintingContext, offset ui.Offset) {
	context.PaintChild(r.Child(), offset)
}

func (r *renderScaledInlineWidget) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	if r.Child() == nil {
		r.transformLayer.SetLayer(nil)
		return
	}
	if r.scale == 1 {
		context.PaintChild(r.Child(), offset)
		r.transformLayer.SetLayer(nil)
		return
	}
	r.transformLayer.SetLayer(context.PushTransform(r.NeedsCompositing(), offset,
		ui.NewMatrix4Diagonal3Values(float32(r.scale), float32(r.scale), 1), r.paintChild, r.transformLayer.Layer()))
}

func (r *renderScaledInlineWidget) Dispose() {
	r.transformLayer.SetLayer(nil)
	r.RenderBox.Dispose()
}

func (r *renderScaledInlineWidget) HitTestChildren(result *rendering.BoxHitTestResult, position ui.Offset) bool {
	child := r.Child()
	if child == nil {
		return false
	}
	return result.AddWithPaintTransform(ui.NewMatrix4Diagonal3Values(float32(r.scale), float32(r.scale), 1), position,
		func(result *rendering.BoxHitTestResult, position ui.Offset) bool {
			return child.HitTest(result, position)
		})
}
