package rendering

import (
	"math"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// ellipsis is the character that replaces the text that does not fit.
//
// Flutter: rendering/paragraph.dart _kEllipsis
const ellipsis = "…"

// TextParentData is the parent data of the inline children of a paragraph: the
// offset that the text layout gave the placeholder, and the span they are for.
//
// Flutter: rendering/paragraph.dart TextParentData
type TextParentData struct {
	ParentDataBase
	ContainerParentData[Box]
	// Offset is the offset at which to paint the child, in the paragraph's
	// coordinates, or None if the child was not laid out yet.
	//
	// Flutter: rendering/paragraph.dart TextParentData.offset
	Offset foundation.Opt[ui.Offset]
	// Span is the placeholder span of the child.
	//
	// Flutter: rendering/paragraph.dart TextParentData.span
	Span painting.PlaceholderSpan
}

// Detach clears the span and the offset.
//
// Flutter: rendering/paragraph.dart TextParentData.detach
func (d *TextParentData) Detach() {
	d.Span = nil
	d.Offset = foundation.None[ui.Offset]()
	d.checkDetach()
}

// String describes the span and the offset.
//
// Flutter: rendering/paragraph.dart TextParentData.toString
func (d *TextParentData) String() string {
	span := "null"
	if d.Span != nil {
		span = foundation.ToStringShort(d.Span)
	}
	if o, ok := d.Offset.Get(); ok {
		return "widget: " + span + ", offset: " + o.String()
	}
	return "widget: " + span + ", not laid out"
}

func textParentData(child Box) *TextParentData { return child.ParentData().(*TextParentData) }

// SetupInlineChildParentData gives the child a TextParentData if it has none.
// A render object with inline children calls it from SetupParentData.
//
// Flutter: rendering/paragraph.dart RenderInlineChildrenContainerDefaults.setupParentData
func SetupInlineChildParentData(child RenderObject) {
	if _, ok := child.ParentData().(*TextParentData); !ok {
		child.AsRenderObject().SetParentData(&TextParentData{})
	}
}

func layoutInlineChild(child Box, childConstraints BoxConstraints, layoutChild ChildLayouter, getBaseline ChildBaselineGetter) painting.PlaceholderDimensions {
	span := textParentData(child).Span
	if span == nil {
		if foundation.DebugMode {
			panic(errors.New("InlineChildWithoutSpan", "an inline child has no placeholder span in its parent data",
				"child", foundation.DescribeIdentity(child)))
		}
		return painting.PlaceholderDimensionsEmpty()
	}
	dims := painting.PlaceholderDimensions{
		Size: layoutChild(child, childConstraints), Alignment: span.Alignment(), Baseline: span.Baseline(),
	}
	if span.Alignment() == ui.PlaceholderAlignmentBaseline {
		dims.BaselineOffset = getBaseline(child, childConstraints, span.Baseline())
	}
	return dims
}

// LayoutInlineChildren lays out the children, which are the placeholders of the
// text, and returns their dimensions for the text layout.
//
// Flutter: rendering/paragraph.dart RenderInlineChildrenContainerDefaults.layoutInlineChildren
func LayoutInlineChildren(c *ContainerRenderObject[Box], maxWidth float64, layoutChild ChildLayouter, getChildBaseline ChildBaselineGetter) []painting.PlaceholderDimensions {
	constraints := BoxConstraints{MaxWidth: maxWidth, MaxHeight: math.Inf(1)}
	var out []painting.PlaceholderDimensions
	for child := c.FirstChild(); !isNil(child); child = c.ChildAfter(child) {
		out = append(out, layoutInlineChild(child, constraints, layoutChild, getChildBaseline))
	}
	return out
}

// PositionInlineChildren sets the offsets of the children from the boxes that
// the text layout gave the placeholders. In debug builds it panics with
// InlineBoxesMismatch if there are more boxes than children.
//
// Flutter: rendering/paragraph.dart RenderInlineChildrenContainerDefaults.positionInlineChildren
func PositionInlineChildren(c *ContainerRenderObject[Box], boxes []ui.TextBox) {
	child := c.FirstChild()
	for _, box := range boxes {
		if isNil(child) {
			if foundation.DebugMode {
				panic(errors.New("InlineBoxesMismatch", "Invalid number of boxes provided to positionInlineChildren. The number of boxes ("+
					itoa(len(boxes))+") exceeds the number of child render objects ("+itoa(c.ChildCount())+"). Each box corresponds "+
					"to a child, but there are not enough children to position all boxes.", "boxes", len(boxes), "children", c.ChildCount()))
			}
			return
		}
		textParentData(child).Offset = foundation.Some(ui.Offset{Dx: box.Left, Dy: box.Top})
		child = c.ChildAfter(child)
	}
	for ; !isNil(child); child = c.ChildAfter(child) {
		textParentData(child).Offset = foundation.None[ui.Offset]()
	}
}

// DefaultApplyPaintTransformInline translates the transform by the offset of the
// inline child, or zeroes it if the child was not laid out.
//
// Flutter: rendering/paragraph.dart RenderInlineChildrenContainerDefaults.defaultApplyPaintTransform
func DefaultApplyPaintTransformInline(child Box, transform *ui.Matrix4) {
	if o, ok := textParentData(child).Offset.Get(); ok {
		transform.Translate(float32(o.Dx), float32(o.Dy), 0)
	} else {
		transform.SetZero()
	}
}

// PaintInlineChildren paints the children at their offsets.
//
// Flutter: rendering/paragraph.dart RenderInlineChildrenContainerDefaults.paintInlineChildren
func PaintInlineChildren(c *ContainerRenderObject[Box], context *PaintingContext, offset ui.Offset) {
	for child := c.FirstChild(); !isNil(child); child = c.ChildAfter(child) {
		childOffset, ok := textParentData(child).Offset.Get()
		if !ok {
			return
		}
		context.PaintChild(child, childOffset.Add(offset))
	}
}

// HitTestInlineChildren hit tests the children at their offsets.
//
// Flutter: rendering/paragraph.dart RenderInlineChildrenContainerDefaults.hitTestInlineChildren
func HitTestInlineChildren(c *ContainerRenderObject[Box], result *BoxHitTestResult, position ui.Offset) bool {
	for child := c.FirstChild(); !isNil(child); child = c.ChildAfter(child) {
		child := child
		childOffset, ok := textParentData(child).Offset.Get()
		if !ok {
			return false
		}
		if result.AddWithPaintOffset(&childOffset, position, func(result *BoxHitTestResult, transformed ui.Offset) bool {
			return child.HitTest(result, transformed)
		}) {
			return true
		}
	}
	return false
}

// RenderParagraph displays a paragraph of text, with inline children for the
// placeholders of its text.
//
// Flutter: rendering/paragraph.dart RenderParagraph
type RenderParagraph struct {
	RenderBox
	ContainerRenderObject[Box]
	engine                  ui.ParagraphFactory
	textPainter             *painting.TextPainter
	textIntrinsicsCache     *painting.TextPainter
	softWrap                bool
	overflow                painting.TextOverflow
	devicePixelRatio        float64
	selectionColor          foundation.Opt[ui.Color]
	registrar               SelectionRegistrar
	lastSelectableFragments []*selectableFragment
	needsClipping           bool
	overflowShader          *ui.Gradient
	placeholderDimensions   []painting.PlaceholderDimensions
	psem                    paragraphSemantics
}

// RenderParagraphOpts are the optional parameters of NewRenderParagraph.
//
// Flutter: rendering/paragraph.dart RenderParagraph.new
type RenderParagraphOpts struct {
	// TextEngine lays out the text. Deviation: Flutter reaches the engine
	// through dart:ui; the widgets pass the engine of the PlatformDispatcher.
	TextEngine ui.ParagraphFactory
	// TextAlign defaults to start.
	TextAlign ui.TextAlign
	// SoftWrap, if None, is true.
	SoftWrap foundation.Opt[bool]
	// Overflow defaults to painting.TextOverflowClip.
	Overflow painting.TextOverflow
	// TextScaler defaults to painting.TextScalerNoScaling. Deviation: the
	// deprecated textScaleFactor is not ported.
	TextScaler painting.TextScaler
	// MaxLines, if set, limits the number of lines; it must be positive.
	MaxLines foundation.Opt[int]
	// Locale selects region-specific glyphs.
	Locale foundation.Opt[ui.Locale]
	// StrutStyle is the strut of the paragraph.
	StrutStyle *painting.StrutStyle
	// TextWidthBasis defaults to painting.TextWidthBasisParent.
	TextWidthBasis painting.TextWidthBasis
	// TextHeightBehavior says how the line height applies to the first and last
	// lines.
	TextHeightBehavior *ui.TextHeightBehavior
	// Children are the inline children, in the order of the placeholders.
	Children []Box
	// SelectionColor is the color of the selected text.
	SelectionColor foundation.Opt[ui.Color]
	// Registrar is where the selectable fragments of the text register.
	Registrar SelectionRegistrar
	// DevicePixelRatio defaults to 1.
	DevicePixelRatio foundation.Opt[float64]
}

// NewRenderParagraph creates a paragraph for the text; textDirection is required.
//
// Flutter: rendering/paragraph.dart RenderParagraph.new
func NewRenderParagraph(text painting.InlineSpan, textDirection ui.TextDirection, opts ...RenderParagraphOpts) *RenderParagraph {
	o := firstOpts(opts)
	if foundation.DebugMode {
		text.DebugAssertIsValid()
		if m, ok := o.MaxLines.Get(); ok && m <= 0 {
			panic(errors.New("ParagraphMaxLinesInvalid", "maxLines must be greater than zero", "maxLines", m))
		}
	}
	if o.Overflow == painting.TextOverflowUnspecified {
		o.Overflow = painting.TextOverflowClip
	}
	r := &RenderParagraph{engine: o.TextEngine, softWrap: o.SoftWrap.OrElse(true), overflow: o.Overflow,
		devicePixelRatio: o.DevicePixelRatio.OrElse(1), selectionColor: o.SelectionColor}
	popts := painting.TextPainterOpts{TextEngine: o.TextEngine, Text: text, TextAlign: o.TextAlign, TextDirection: textDirection,
		TextScaler: o.TextScaler, MaxLines: o.MaxLines, Locale: o.Locale, StrutStyle: o.StrutStyle,
		TextWidthBasis: o.TextWidthBasis, TextHeightBehavior: o.TextHeightBehavior}
	if o.Overflow == painting.TextOverflowEllipsis {
		popts.Ellipsis = ellipsis
	}
	r.textPainter = painting.NewTextPainter(popts)
	InitRenderBox(&r.RenderBox, r)
	InitContainerRenderObject(&r.ContainerRenderObject, r)
	r.AddAll(o.Children)
	r.SetRegistrar(o.Registrar)
	return r
}

// SetupParentData gives the child a TextParentData if it has none.
//
// Flutter: rendering/paragraph.dart RenderInlineChildrenContainerDefaults.setupParentData
func (r *RenderParagraph) SetupParentData(child RenderObject) { SetupInlineChildParentData(child) }

func (r *RenderParagraph) textIntrinsics() *painting.TextPainter {
	if r.textIntrinsicsCache == nil {
		r.textIntrinsicsCache = painting.NewTextPainter(painting.TextPainterOpts{TextEngine: r.engine})
	}
	p, q := r.textIntrinsicsCache, r.textPainter
	p.SetText(q.Text())
	p.SetTextAlign(q.TextAlign())
	p.SetTextDirection(q.TextDirection())
	p.SetTextScaler(q.TextScaler())
	p.SetMaxLines(q.MaxLines())
	p.SetEllipsis(q.Ellipsis())
	p.SetLocale(q.Locale())
	p.SetStrutStyle(q.StrutStyle())
	p.SetTextWidthBasis(q.TextWidthBasis())
	p.SetTextHeightBehavior(q.TextHeightBehavior())
	return p
}

// Text is the text to display.
//
// Flutter: rendering/paragraph.dart RenderParagraph.text
func (r *RenderParagraph) Text() painting.InlineSpan { return r.textPainter.Text() }

// SetText changes the text. What the paragraph does depends on how the new text
// compares to the old: nothing, a semantics update, a repaint or a new layout.
//
// Flutter: rendering/paragraph.dart RenderParagraph.text
func (r *RenderParagraph) SetText(value painting.InlineSpan) {
	switch r.textPainter.Text().CompareTo(value) {
	case painting.RenderComparisonIdentical:
		return
	case painting.RenderComparisonMetadata:
		r.textPainter.SetText(value)
		r.psem.haveCombinedInfos = false
		r.MarkNeedsSemanticsUpdate()
	case painting.RenderComparisonPaint:
		r.textPainter.SetText(value)
		r.psem.haveCachedLabels, r.psem.haveCombinedInfos = false, false
		r.MarkNeedsPaint()
		r.MarkNeedsSemanticsUpdate()
	case painting.RenderComparisonLayout:
		r.textPainter.SetText(value)
		r.overflowShader = nil
		r.psem.haveCachedLabels, r.psem.haveCombinedInfos = false, false
		r.MarkNeedsLayout()
		r.removeSelectionRegistrarSubscription()
		r.disposeSelectableFragments()
		r.updateSelectionRegistrarSubscription()
	}
}

// Registrar is where the selectable fragments of the text register.
//
// Flutter: rendering/paragraph.dart RenderParagraph.registrar
func (r *RenderParagraph) Registrar() SelectionRegistrar { return r.registrar }

// SetRegistrar changes the registrar; the fragments of the text register with
// the new one.
//
// Flutter: rendering/paragraph.dart RenderParagraph.registrar
func (r *RenderParagraph) SetRegistrar(value SelectionRegistrar) {
	if value == r.registrar {
		return
	}
	r.removeSelectionRegistrarSubscription()
	r.disposeSelectableFragments()
	r.registrar = value
	r.updateSelectionRegistrarSubscription()
}

func (r *RenderParagraph) updateSelectionRegistrarSubscription() {
	if r.registrar == nil {
		return
	}
	if r.lastSelectableFragments == nil {
		r.lastSelectableFragments = r.getSelectableFragments()
	}
	for _, f := range r.lastSelectableFragments {
		r.registrar.Add(f)
	}
	if len(r.lastSelectableFragments) > 0 {
		r.MarkNeedsCompositingBitsUpdate()
	}
}

func (r *RenderParagraph) removeSelectionRegistrarSubscription() {
	if r.registrar == nil || r.lastSelectableFragments == nil {
		return
	}
	for _, f := range r.lastSelectableFragments {
		r.registrar.Remove(f)
	}
}

// getSelectableFragments splits the text at the placeholders.
//
// Flutter: rendering/paragraph.dart RenderParagraph._getSelectableFragments
func (r *RenderParagraph) getSelectableFragments() []*selectableFragment {
	plainText := painting.InlineSpanToPlainText(r.Text(), painting.ToPlainTextOpts{ExcludeSemanticsLabels: true})
	var result []*selectableFragment
	placeholder := string(rune(painting.PlaceholderCodeUnit))
	for start := 0; start < len(plainText); {
		end := strings.Index(plainText[start:], placeholder)
		if end >= 0 {
			end += start
		}
		if start != end {
			if end == -1 {
				end = len(plainText)
			}
			result = append(result, newSelectableFragment(r, ui.TextRange{Start: start, End: end}, plainText))
			start = end
		}
		start += len(placeholder)
	}
	return result
}

// SelectableBelongsToParagraph says whether the selectable is one of the
// fragments of this paragraph.
//
// Flutter: rendering/paragraph.dart RenderParagraph.selectableBelongsToParagraph
func (r *RenderParagraph) SelectableBelongsToParagraph(selectable Selectable) bool {
	for _, f := range r.lastSelectableFragments {
		if Selectable(f) == selectable {
			return true
		}
	}
	return false
}

func (r *RenderParagraph) disposeSelectableFragments() {
	for _, f := range r.lastSelectableFragments {
		f.Dispose()
	}
	r.lastSelectableFragments = nil
}

// AlwaysNeedsCompositing is true while the paragraph has selectable fragments.
//
// Flutter: rendering/paragraph.dart RenderParagraph.alwaysNeedsCompositing
func (r *RenderParagraph) AlwaysNeedsCompositing() bool { return len(r.lastSelectableFragments) > 0 }

// MarkNeedsLayout tells the fragments that the layout changes.
//
// Flutter: rendering/paragraph.dart RenderParagraph.markNeedsLayout
func (r *RenderParagraph) MarkNeedsLayout() {
	for _, f := range r.lastSelectableFragments {
		f.didChangeParagraphLayout()
	}
	r.RenderBox.MarkNeedsLayout()
}

// Dispose disposes the fragments and the text painters.
//
// Flutter: rendering/paragraph.dart RenderParagraph.dispose
func (r *RenderParagraph) Dispose() {
	r.removeSelectionRegistrarSubscription()
	r.disposeSelectableFragments()
	r.textPainter.Dispose()
	if r.textIntrinsicsCache != nil {
		r.textIntrinsicsCache.Dispose()
	}
	r.RenderBox.Dispose()
}

// SystemFontsDidChange lays the paragraph out again, because the fonts of the
// system changed.
//
// Flutter: rendering/object.dart RelayoutWhenSystemFontsChangeMixin.systemFontsDidChange
func (r *RenderParagraph) SystemFontsDidChange() {
	r.MarkNeedsLayout()
	r.textPainter.MarkNeedsLayout()
}

// TextAlign is how the text is aligned horizontally.
//
// Flutter: rendering/paragraph.dart RenderParagraph.textAlign
func (r *RenderParagraph) TextAlign() ui.TextAlign { return r.textPainter.TextAlign() }

// SetTextAlign marks the paragraph as needing paint when the value changes.
//
// Flutter: rendering/paragraph.dart RenderParagraph.textAlign
func (r *RenderParagraph) SetTextAlign(value ui.TextAlign) {
	if r.textPainter.TextAlign() == value {
		return
	}
	r.textPainter.SetTextAlign(value)
	r.MarkNeedsPaint()
}

// TextDirection is the direction of the text.
//
// Flutter: rendering/paragraph.dart RenderParagraph.textDirection
func (r *RenderParagraph) TextDirection() ui.TextDirection { return r.textPainter.TextDirection() }

// SetTextDirection marks the paragraph as needing layout when the value changes.
//
// Flutter: rendering/paragraph.dart RenderParagraph.textDirection
func (r *RenderParagraph) SetTextDirection(value ui.TextDirection) {
	if r.textPainter.TextDirection() == value {
		return
	}
	r.textPainter.SetTextDirection(value)
	r.MarkNeedsLayout()
}

// SoftWrap says whether the text wraps at the width of the box.
//
// Flutter: rendering/paragraph.dart RenderParagraph.softWrap
func (r *RenderParagraph) SoftWrap() bool { return r.softWrap }

// SetSoftWrap marks the paragraph as needing layout when the value changes.
//
// Flutter: rendering/paragraph.dart RenderParagraph.softWrap
func (r *RenderParagraph) SetSoftWrap(value bool) {
	if r.softWrap == value {
		return
	}
	r.softWrap = value
	r.MarkNeedsLayout()
}

// Overflow is how the visual overflow is handled.
//
// Flutter: rendering/paragraph.dart RenderParagraph.overflow
func (r *RenderParagraph) Overflow() painting.TextOverflow { return r.overflow }

// SetOverflow marks the paragraph as needing layout when the value changes.
//
// Flutter: rendering/paragraph.dart RenderParagraph.overflow
func (r *RenderParagraph) SetOverflow(value painting.TextOverflow) {
	if value == painting.TextOverflowUnspecified {
		value = painting.TextOverflowClip
	}
	if r.overflow == value {
		return
	}
	r.overflow = value
	if value == painting.TextOverflowEllipsis {
		r.textPainter.SetEllipsis(ellipsis)
	} else {
		r.textPainter.SetEllipsis("")
	}
	r.MarkNeedsLayout()
}

// TextScaler is how the font sizes are scaled.
//
// Flutter: rendering/paragraph.dart RenderParagraph.textScaler
func (r *RenderParagraph) TextScaler() painting.TextScaler { return r.textPainter.TextScaler() }

// SetTextScaler marks the paragraph as needing layout when the value changes.
//
// Flutter: rendering/paragraph.dart RenderParagraph.textScaler
func (r *RenderParagraph) SetTextScaler(value painting.TextScaler) {
	if r.textPainter.TextScaler() == value {
		return
	}
	r.textPainter.SetTextScaler(value)
	r.overflowShader = nil
	r.MarkNeedsLayout()
}

// DevicePixelRatio is the number of device pixels per logical pixel.
//
// Flutter: rendering/paragraph.dart RenderParagraph.devicePixelRatio
func (r *RenderParagraph) DevicePixelRatio() float64 { return r.devicePixelRatio }

// SetDevicePixelRatio changes the ratio. Deviation: Flutter repaints only on
// the web, where text is positioned in device pixels; here nothing repaints.
//
// Flutter: rendering/paragraph.dart RenderParagraph.devicePixelRatio
func (r *RenderParagraph) SetDevicePixelRatio(value float64) { r.devicePixelRatio = value }

// MaxLines is the maximum number of lines, if set.
//
// Flutter: rendering/paragraph.dart RenderParagraph.maxLines
func (r *RenderParagraph) MaxLines() foundation.Opt[int] { return r.textPainter.MaxLines() }

// SetMaxLines marks the paragraph as needing layout when the value changes. In
// debug builds a value that is not positive panics with ParagraphMaxLinesInvalid.
//
// Flutter: rendering/paragraph.dart RenderParagraph.maxLines
func (r *RenderParagraph) SetMaxLines(value foundation.Opt[int]) {
	if m, ok := value.Get(); ok && m <= 0 && foundation.DebugMode {
		panic(errors.New("ParagraphMaxLinesInvalid", "maxLines must be greater than zero", "maxLines", m))
	}
	if r.textPainter.MaxLines() == value {
		return
	}
	r.textPainter.SetMaxLines(value)
	r.overflowShader = nil
	r.MarkNeedsLayout()
}

// Locale selects region-specific glyphs.
//
// Flutter: rendering/paragraph.dart RenderParagraph.locale
func (r *RenderParagraph) Locale() foundation.Opt[ui.Locale] { return r.textPainter.Locale() }

// SetLocale marks the paragraph as needing layout when the value changes.
//
// Flutter: rendering/paragraph.dart RenderParagraph.locale
func (r *RenderParagraph) SetLocale(value foundation.Opt[ui.Locale]) {
	if r.textPainter.Locale() == value {
		return
	}
	r.textPainter.SetLocale(value)
	r.overflowShader = nil
	r.MarkNeedsLayout()
}

// StrutStyle is the strut of the paragraph.
//
// Flutter: rendering/paragraph.dart RenderParagraph.strutStyle
func (r *RenderParagraph) StrutStyle() *painting.StrutStyle { return r.textPainter.StrutStyle() }

// SetStrutStyle marks the paragraph as needing layout when the value changes.
//
// Flutter: rendering/paragraph.dart RenderParagraph.strutStyle
func (r *RenderParagraph) SetStrutStyle(value *painting.StrutStyle) {
	if r.textPainter.StrutStyle() == value {
		return
	}
	r.textPainter.SetStrutStyle(value)
	r.overflowShader = nil
	r.MarkNeedsLayout()
}

// TextWidthBasis is how the width of the text is measured.
//
// Flutter: rendering/paragraph.dart RenderParagraph.textWidthBasis
func (r *RenderParagraph) TextWidthBasis() painting.TextWidthBasis {
	return r.textPainter.TextWidthBasis()
}

// SetTextWidthBasis marks the paragraph as needing layout when the value changes.
//
// Flutter: rendering/paragraph.dart RenderParagraph.textWidthBasis
func (r *RenderParagraph) SetTextWidthBasis(value painting.TextWidthBasis) {
	if r.textPainter.TextWidthBasis() == value {
		return
	}
	r.textPainter.SetTextWidthBasis(value)
	r.overflowShader = nil
	r.MarkNeedsLayout()
}

// TextHeightBehavior says how the line height applies to the first and last
// lines.
//
// Flutter: rendering/paragraph.dart RenderParagraph.textHeightBehavior
func (r *RenderParagraph) TextHeightBehavior() *ui.TextHeightBehavior {
	return r.textPainter.TextHeightBehavior()
}

// SetTextHeightBehavior marks the paragraph as needing layout when the value
// changes.
//
// Flutter: rendering/paragraph.dart RenderParagraph.textHeightBehavior
func (r *RenderParagraph) SetTextHeightBehavior(value *ui.TextHeightBehavior) {
	if r.textPainter.TextHeightBehavior() == value {
		return
	}
	r.textPainter.SetTextHeightBehavior(value)
	r.overflowShader = nil
	r.MarkNeedsLayout()
}

// SelectionColor is the color of the selected text.
//
// Flutter: rendering/paragraph.dart RenderParagraph.selectionColor
func (r *RenderParagraph) SelectionColor() foundation.Opt[ui.Color] { return r.selectionColor }

// SetSelectionColor repaints the paragraph if some of it is selected.
//
// Flutter: rendering/paragraph.dart RenderParagraph.selectionColor
func (r *RenderParagraph) SetSelectionColor(value foundation.Opt[ui.Color]) {
	if r.selectionColor == value {
		return
	}
	r.selectionColor = value
	for _, f := range r.lastSelectableFragments {
		if f.Value().HasSelection() {
			r.MarkNeedsPaint()
			return
		}
	}
}

// ComputeMinIntrinsicWidth is the width of the narrowest layout of the text.
//
// Flutter: rendering/paragraph.dart RenderParagraph.computeMinIntrinsicWidth
func (r *RenderParagraph) ComputeMinIntrinsicWidth(float64) float64 {
	dims := LayoutInlineChildren(&r.ContainerRenderObject, math.Inf(1), func(child Box, _ BoxConstraints) ui.Size {
		return ui.Size{Width: child.GetMinIntrinsicWidth(math.Inf(1))}
	}, ChildLayoutHelper.GetDryBaseline)
	p := r.textIntrinsics()
	p.SetPlaceholderDimensions(dims)
	p.Layout(0, math.Inf(1))
	return p.MinIntrinsicWidth()
}

// ComputeMaxIntrinsicWidth is the width of the text without wrapping.
//
// Flutter: rendering/paragraph.dart RenderParagraph.computeMaxIntrinsicWidth
func (r *RenderParagraph) ComputeMaxIntrinsicWidth(float64) float64 {
	dims := LayoutInlineChildren(&r.ContainerRenderObject, math.Inf(1), func(child Box, _ BoxConstraints) ui.Size {
		return ui.Size{Width: child.GetMaxIntrinsicWidth(math.Inf(1))}
	}, ChildLayoutHelper.GetDryBaseline)
	p := r.textIntrinsics()
	p.SetPlaceholderDimensions(dims)
	p.Layout(0, math.Inf(1))
	return p.MaxIntrinsicWidth()
}

// PreferredLineHeight is the height of a line of the text's first style.
//
// Flutter: rendering/paragraph.dart RenderParagraph.preferredLineHeight
func (r *RenderParagraph) PreferredLineHeight() float64 { return r.textPainter.PreferredLineHeight() }

func (r *RenderParagraph) adjustMaxWidth(maxWidth float64) float64 {
	if r.softWrap || r.overflow == painting.TextOverflowEllipsis {
		return maxWidth
	}
	return math.Inf(1)
}

func (r *RenderParagraph) computeIntrinsicHeight(width float64) float64 {
	p := r.textIntrinsics()
	p.SetPlaceholderDimensions(LayoutInlineChildren(&r.ContainerRenderObject, width, ChildLayoutHelper.DryLayoutChild, ChildLayoutHelper.GetDryBaseline))
	p.Layout(width, r.adjustMaxWidth(width))
	return p.Height()
}

// ComputeMinIntrinsicHeight is the height of the text laid out at the width.
//
// Flutter: rendering/paragraph.dart RenderParagraph.computeMinIntrinsicHeight
func (r *RenderParagraph) ComputeMinIntrinsicHeight(width float64) float64 {
	return r.computeIntrinsicHeight(width)
}

// ComputeMaxIntrinsicHeight is the height of the text laid out at the width.
//
// Flutter: rendering/paragraph.dart RenderParagraph.computeMaxIntrinsicHeight
func (r *RenderParagraph) ComputeMaxIntrinsicHeight(width float64) float64 {
	return r.computeIntrinsicHeight(width)
}

// HitTestSelf is true: the paragraph is hit anywhere.
//
// Flutter: rendering/paragraph.dart RenderParagraph.hitTestSelf
func (r *RenderParagraph) HitTestSelf(ui.Offset) bool { return true }

// HitTestChildren adds the span under the position to the result if it is a hit
// test target (for example, a TextSpan with a recognizer), else hit tests the
// inline children.
//
// Flutter: rendering/paragraph.dart RenderParagraph.hitTestChildren
func (r *RenderParagraph) HitTestChildren(result *BoxHitTestResult, position ui.Offset) bool {
	if glyph, ok := r.textPainter.GetClosestGlyphForOffset(position); ok && glyph.GraphemeClusterLayoutBounds.Contains(position) {
		span := painting.InlineSpanGetSpanForPosition(r.textPainter.Text(), ui.NewTextPosition(glyph.GraphemeClusterRange.Start))
		if target, ok := span.(gestures.HitTestTarget); ok {
			result.Add(gestures.NewHitTestEntry(target))
			return true
		}
	}
	return HitTestInlineChildren(&r.ContainerRenderObject, result, position)
}

// DebugHasOverflowShader says whether the paragraph fades its overflow.
//
// Flutter: rendering/paragraph.dart RenderParagraph.debugHasOverflowShader
func (r *RenderParagraph) DebugHasOverflowShader() bool { return r.overflowShader != nil }

func (r *RenderParagraph) layoutTextWithConstraints(constraints BoxConstraints) {
	r.textPainter.SetPlaceholderDimensions(r.placeholderDimensions)
	r.textPainter.Layout(constraints.MinWidth, r.adjustMaxWidth(constraints.MaxWidth))
}

// ComputeDryLayout is the size of the text at the constraints.
//
// Flutter: rendering/paragraph.dart RenderParagraph.computeDryLayout
func (r *RenderParagraph) ComputeDryLayout(constraints BoxConstraints) ui.Size {
	p := r.textIntrinsics()
	p.SetPlaceholderDimensions(LayoutInlineChildren(&r.ContainerRenderObject, constraints.MaxWidth, ChildLayoutHelper.DryLayoutChild, ChildLayoutHelper.GetDryBaseline))
	p.Layout(constraints.MinWidth, r.adjustMaxWidth(constraints.MaxWidth))
	return constraints.Constrain(p.Size())
}

// ComputeDistanceToActualBaseline is the alphabetic baseline of the first line.
//
// Flutter: rendering/paragraph.dart RenderParagraph.computeDistanceToActualBaseline
func (r *RenderParagraph) ComputeDistanceToActualBaseline(ui.TextBaseline) foundation.Opt[float64] {
	if foundation.DebugMode && r.DebugNeedsLayout() {
		panic(errors.New("BaselineNeedsLayout", "the baseline was read from a paragraph that needs layout",
			"object", foundation.DescribeIdentity(r)))
	}
	r.layoutTextWithConstraints(r.BoxConstraints())
	return foundation.Some(r.textPainter.ComputeDistanceToActualBaseline(ui.TextBaselineAlphabetic))
}

// ComputeDryBaseline is the alphabetic baseline of the first line at the
// constraints.
//
// Flutter: rendering/paragraph.dart RenderParagraph.computeDryBaseline
func (r *RenderParagraph) ComputeDryBaseline(constraints BoxConstraints, _ ui.TextBaseline) foundation.Opt[float64] {
	p := r.textIntrinsics()
	p.SetPlaceholderDimensions(LayoutInlineChildren(&r.ContainerRenderObject, constraints.MaxWidth, ChildLayoutHelper.DryLayoutChild, ChildLayoutHelper.GetDryBaseline))
	p.Layout(constraints.MinWidth, r.adjustMaxWidth(constraints.MaxWidth))
	return foundation.Some(p.ComputeDistanceToActualBaseline(ui.TextBaselineAlphabetic))
}

// PerformLayout lays out the inline children, then the text around them, and
// works out how the overflow is handled.
//
// Flutter: rendering/paragraph.dart RenderParagraph.performLayout
func (r *RenderParagraph) PerformLayout() {
	for _, f := range r.lastSelectableFragments {
		f.didChangeParagraphLayout()
	}
	constraints := r.BoxConstraints()
	r.placeholderDimensions = LayoutInlineChildren(&r.ContainerRenderObject, constraints.MaxWidth, ChildLayoutHelper.LayoutChild, ChildLayoutHelper.GetBaseline)
	r.layoutTextWithConstraints(constraints)
	PositionInlineChildren(&r.ContainerRenderObject, r.textPainter.InlinePlaceholderBoxes())
	textSize := r.textPainter.Size()
	r.SetSize(constraints.Constrain(textSize))
	size := r.Size()
	didOverflowHeight := size.Height < textSize.Height || r.textPainter.DidExceedMaxLines()
	didOverflowWidth := size.Width < textSize.Width
	if !didOverflowWidth && !didOverflowHeight {
		r.needsClipping, r.overflowShader = false, nil
		return
	}
	switch r.overflow {
	case painting.TextOverflowVisible:
		r.needsClipping, r.overflowShader = false, nil
	case painting.TextOverflowFade:
		r.needsClipping = true
		fade := painting.NewTextPainter(painting.TextPainterOpts{TextEngine: r.engine,
			Text:          painting.NewTextSpan(painting.TextSpanOpts{Style: r.textPainter.Text().Style(), Text: foundation.Some("…")}),
			TextDirection: r.TextDirection(), TextScaler: r.TextScaler(), Locale: r.Locale()})
		fade.Layout(0, math.Inf(1))
		white, clear := ui.NewColor(0xFFFFFFFF), ui.NewColor(0x00FFFFFF)
		if didOverflowWidth {
			start, end := size.Width-fade.Width(), size.Width
			if r.TextDirection() == ui.TextDirectionRTL {
				start, end = fade.Width(), 0
			}
			r.overflowShader = ui.NewGradientLinear(ui.Offset{Dx: start}, ui.Offset{Dx: end}, []ui.Color{white, clear})
		} else {
			end := size.Height
			start := end - fade.Height()/2
			r.overflowShader = ui.NewGradientLinear(ui.Offset{Dy: start}, ui.Offset{Dy: end}, []ui.Color{white, clear})
		}
		fade.Dispose()
	default: // clip, ellipsis
		r.needsClipping, r.overflowShader = true, nil
	}
}

// ApplyPaintTransform translates the transform by the offset of the inline child.
//
// Flutter: rendering/paragraph.dart RenderParagraph.applyPaintTransform
func (r *RenderParagraph) ApplyPaintTransform(child RenderObject, transform *ui.Matrix4) {
	DefaultApplyPaintTransformInline(child.(Box), transform)
}

// Paint paints the selection, the text and the inline children, clipped or
// faded if the text overflows.
//
// Flutter: rendering/paragraph.dart RenderParagraph.paint
func (r *RenderParagraph) Paint(context *PaintingContext, offset ui.Offset) {
	r.layoutTextWithConstraints(r.BoxConstraints())
	debug := r.DebugOptions()
	if foundation.DebugMode && debug.RepaintTextRainbowEnabled {
		paint := ui.NewPaint()
		color := debug.CurrentRepaintColor
		if color == (painting.HSVColor{}) {
			color = DefaultRepaintColor
		}
		paint.SetColor(color.ToColor())
		context.Canvas().DrawRect(offset.And(r.Size()), paint)
	}
	if r.lastSelectableFragments != nil {
		if r.needsClipping {
			context.Canvas().Save()
			context.Canvas().ClipRect(offset.And(r.Size()))
		}
		for _, f := range r.lastSelectableFragments {
			f.paintSelection(context, offset)
		}
		if r.needsClipping {
			context.Canvas().Restore()
		}
	}
	if r.needsClipping {
		bounds := offset.And(r.Size())
		if r.overflowShader != nil {
			context.Canvas().SaveLayer(&bounds, ui.NewPaint())
		} else {
			context.Canvas().Save()
		}
		context.Canvas().ClipRect(bounds)
	}
	if foundation.DebugMode {
		r.textPainter.DebugPaintTextLayoutBoxes = debug.PaintTextLayoutBoxes
	}
	r.textPainter.Paint(context.Canvas(), offset)
	PaintInlineChildren(&r.ContainerRenderObject, context, offset)
	if r.needsClipping {
		if r.overflowShader != nil {
			context.Canvas().Translate(offset.Dx, offset.Dy)
			paint := ui.NewPaint()
			paint.SetBlendMode(ui.BlendModeModulate)
			paint.SetShader(r.overflowShader)
			context.Canvas().DrawRect(ui.OffsetZero.And(r.Size()), paint)
		}
		context.Canvas().Restore()
	}
	for _, f := range r.lastSelectableFragments {
		f.paintHandles(context, offset)
	}
}

// GetOffsetForCaret is the offset at which to paint a caret at the position, in
// the paragraph's coordinates. The paragraph must be laid out.
//
// Flutter: rendering/paragraph.dart RenderParagraph.getOffsetForCaret
func (r *RenderParagraph) GetOffsetForCaret(position ui.TextPosition, caretPrototype ui.Rect) ui.Offset {
	r.checkLaidOut()
	r.layoutTextWithConstraints(r.BoxConstraints())
	return r.textPainter.GetOffsetForCaret(position, caretPrototype)
}

// GetFullHeightForCaret is the full height of the caret at the position.
//
// Flutter: rendering/paragraph.dart RenderParagraph.getFullHeightForCaret
func (r *RenderParagraph) GetFullHeightForCaret(position ui.TextPosition) float64 {
	r.checkLaidOut()
	r.layoutTextWithConstraints(r.BoxConstraints())
	return r.textPainter.GetFullHeightForCaret(position, ui.Rect{})
}

// GetBoxesForSelection are the boxes that enclose the text range.
//
// Deviation: Flutter takes a TextSelection, which is ported with the text editing
// of step 10.x; a TextRange does the same.
//
// Flutter: rendering/paragraph.dart RenderParagraph.getBoxesForSelection
func (r *RenderParagraph) GetBoxesForSelection(selection ui.TextRange, opts ...ui.BoxesForRangeOpts) []ui.TextBox {
	r.checkLaidOut()
	r.layoutTextWithConstraints(r.BoxConstraints())
	return r.textPainter.GetBoxesForSelection(selection, opts...)
}

// GetPositionForOffset is the text position closest to the offset.
//
// Flutter: rendering/paragraph.dart RenderParagraph.getPositionForOffset
func (r *RenderParagraph) GetPositionForOffset(offset ui.Offset) ui.TextPosition {
	r.checkLaidOut()
	r.layoutTextWithConstraints(r.BoxConstraints())
	return r.textPainter.GetPositionForOffset(offset)
}

// GetWordBoundary is the range of the word at the position.
//
// Flutter: rendering/paragraph.dart RenderParagraph.getWordBoundary
func (r *RenderParagraph) GetWordBoundary(position ui.TextPosition) ui.TextRange {
	r.checkLaidOut()
	r.layoutTextWithConstraints(r.BoxConstraints())
	return r.textPainter.GetWordBoundary(position)
}

func (r *RenderParagraph) checkLaidOut() {
	if foundation.DebugMode && r.DebugNeedsLayout() {
		panic(errors.New("ParagraphNeedsLayout", "the paragraph was queried before it was laid out",
			"object", foundation.DescribeIdentity(r)))
	}
}

// TextSize is the size of the text, which can differ from the size of the box.
//
// Flutter: rendering/paragraph.dart RenderParagraph.textSize
func (r *RenderParagraph) TextSize() ui.Size {
	r.checkLaidOut()
	return r.textPainter.Size()
}

// DidExceedMaxLines says whether the text had more lines than maxLines allows.
//
// Flutter: rendering/paragraph.dart RenderParagraph.didExceedMaxLines
func (r *RenderParagraph) DidExceedMaxLines() bool {
	r.checkLaidOut()
	return r.textPainter.DidExceedMaxLines()
}

// DebugDescribeChildren describes the text.
//
// Flutter: rendering/paragraph.dart RenderParagraph.debugDescribeChildren
func (r *RenderParagraph) DebugDescribeChildren() []foundation.DiagnosticsNode {
	return []foundation.DiagnosticsNode{foundation.ToDiagnosticsNode(r.Text(), foundation.DiagnosticsNodeOpts{
		Name: foundation.Some("text"), Style: foundation.DiagnosticsTreeStyleTransition})}
}

// DebugFillProperties adds the text settings.
//
// Flutter: rendering/paragraph.dart RenderParagraph.debugFillProperties
func (r *RenderParagraph) DebugFillProperties(b *foundation.DiagnosticPropertiesBuilder) {
	r.RenderBox.DebugFillProperties(b)
	b.Add(foundation.NewEnumProperty("textAlign", foundation.Some(r.TextAlign())))
	b.Add(foundation.NewEnumProperty("textDirection", foundation.Some(r.TextDirection())))
	b.Add(foundation.NewFlagProperty("softWrap", foundation.Some(r.softWrap), foundation.FlagPropertyOpts{
		IfTrue: "wrapping at box width", IfFalse: "no wrapping except at line break characters", ShowName: true}))
	b.Add(foundation.NewEnumProperty("overflow", foundation.Some(r.overflow)))
	b.Add(foundation.NewDiagnosticsProperty("textScaler", r.TextScaler(), foundation.DiagnosticsPropertyOpts{DefaultValue: painting.TextScalerNoScaling}))
	b.Add(foundation.NewDiagnosticsPropertyOpt("locale", r.Locale(), foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}))
	b.Add(foundation.NewIntProperty("maxLines", r.MaxLines(), foundation.NumberPropertyOpts{IfNull: "unlimited"}))
	b.Add(foundation.NewDoubleProperty("devicePixelRatio", foundation.Some(r.devicePixelRatio), foundation.NumberPropertyOpts{DefaultValue: 1.0}))
}

// selectableFragment is the Selectable of a run of text between two
// placeholders of a paragraph.
//
// TODO(step 11.5): the selection logic of _SelectableFragment (the events, the
// geometry, the handles and the selected content) comes with SelectableRegion.
// Until then a fragment has content and bounding boxes and takes no selection.
//
// Flutter: rendering/paragraph.dart _SelectableFragment
type selectableFragment struct {
	foundation.ChangeNotifier
	paragraph *RenderParagraph
	rng       ui.TextRange
	fullText  string
	geometry  SelectionGeometry
}

func newSelectableFragment(p *RenderParagraph, rng ui.TextRange, fullText string) *selectableFragment {
	return &selectableFragment{paragraph: p, rng: rng, fullText: fullText,
		geometry: SelectionGeometry{Status: SelectionStatusNone, HasContent: true}}
}

func (f *selectableFragment) Value() SelectionGeometry { return f.geometry }

func (f *selectableFragment) PushHandleLayers(_, _ *LayerLink) {}

func (f *selectableFragment) GetSelectedContent() *SelectedContent { return nil }

func (f *selectableFragment) GetSelection() *SelectedContentRange { return nil }

func (f *selectableFragment) DispatchSelectionEvent(SelectionEvent) SelectionResult {
	return SelectionResultNone
}

func (f *selectableFragment) ContentLength() int { return f.rng.End - f.rng.Start }

func (f *selectableFragment) GetTransformTo(ancestor RenderObject) *ui.Matrix4 {
	return f.paragraph.GetTransformTo(ancestor)
}

func (f *selectableFragment) Size() ui.Size { return f.paragraph.TextSize() }

func (f *selectableFragment) BoundingBoxes() []ui.Rect {
	var rects []ui.Rect
	for _, b := range f.paragraph.GetBoxesForSelection(f.rng) {
		rects = append(rects, b.ToRect())
	}
	return rects
}

func (f *selectableFragment) Dispose() { f.ChangeNotifier.Dispose() }

func (f *selectableFragment) didChangeParagraphLayout() {}

func (f *selectableFragment) paintSelection(*PaintingContext, ui.Offset) {}

func (f *selectableFragment) paintHandles(*PaintingContext, ui.Offset) {}
