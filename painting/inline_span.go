package painting

import (
	"slices"
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
)

// placeholderText is what a placeholder adds to the text of a paragraph: the
// object replacement character.
const placeholderText = "￼"

// PlaceholderCodeUnit is the object replacement character that stands for a
// placeholder in the text of a paragraph.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpan.placeholderCodeUnit
const PlaceholderCodeUnit = 0xFFFC

// Accumulator is a mutable counter that spans use to find the text offset of a
// span while visiting the tree. The offsets are UTF-8 byte offsets (SPEC
// §12.10).
//
// Flutter: painting/inline_span.dart Accumulator
type Accumulator struct{ value int }

// NewAccumulator creates a counter that starts at the value.
//
// Flutter: painting/inline_span.dart Accumulator.new
func NewAccumulator(value int) *Accumulator { return &Accumulator{value} }

// Value is the count.
//
// Flutter: painting/inline_span.dart Accumulator.value
func (a *Accumulator) Value() int { return a.value }

// Increment adds to the count. In debug builds a negative addend panics with
// the tag AccumulatorNegativeIncrement.
//
// Flutter: painting/inline_span.dart Accumulator.increment
func (a *Accumulator) Increment(addend int) {
	if foundation.DebugMode && addend < 0 {
		panic(errors.New("AccumulatorNegativeIncrement", "an accumulator can only grow", "addend", addend))
	}
	a.value += addend
}

// InlineSpanVisitor is called with each span of a visit. Returning false stops
// the visit.
//
// Flutter: painting/inline_span.dart InlineSpanVisitor
type InlineSpanVisitor func(span InlineSpan) bool

// InlineSpanSemanticsInformation is the information a span gives to the
// semantics tree of the paragraph.
//
// Flutter: painting/inline_span.dart InlineSpanSemanticsInformation
type InlineSpanSemanticsInformation struct {
	// Text is the text of the span.
	Text string
	// IsPlaceholder is whether the span is a placeholder.
	IsPlaceholder bool
	// SemanticsLabel replaces the text for accessibility, when SemanticsLabelSet.
	SemanticsLabel string
	// SemanticsLabelSet is whether SemanticsLabel is not null.
	SemanticsLabelSet bool
	// SemanticsIdentifier is the identifier of the semantics node, or "".
	SemanticsIdentifier string
	// Recognizer is the gesture recognizer of the span, or nil.
	Recognizer gestures.GestureRecognizer
	// RequiresOwnNode is whether the span needs its own semantics node.
	RequiresOwnNode bool
	// StringAttributes are the annotations of the text.
	StringAttributes []ui.StringAttribute
}

// NewInlineSpanSemanticsInformation creates the information of a span. In debug
// builds a placeholder that has a different text, a label or a recognizer
// panics with the tag InlineSpanSemanticsPlaceholderInvalid.
//
// Flutter: painting/inline_span.dart InlineSpanSemanticsInformation.new
func NewInlineSpanSemanticsInformation(text string, info InlineSpanSemanticsInformation) InlineSpanSemanticsInformation {
	info.Text = text
	if foundation.DebugMode && info.IsPlaceholder && (text != placeholderText || info.SemanticsLabelSet || info.Recognizer != nil) {
		panic(errors.New("InlineSpanSemanticsPlaceholderInvalid", "a placeholder has the text U+FFFC and no label or recognizer"))
	}
	info.RequiresOwnNode = info.IsPlaceholder || info.Recognizer != nil || info.SemanticsIdentifier != ""
	return info
}

// InlineSpanSemanticsPlaceholder is the information of a placeholder.
//
// Flutter: painting/inline_span.dart InlineSpanSemanticsInformation.placeholder
func InlineSpanSemanticsPlaceholder() InlineSpanSemanticsInformation {
	return NewInlineSpanSemanticsInformation(placeholderText, InlineSpanSemanticsInformation{IsPlaceholder: true})
}

// Equal is Dart's == on the information.
//
// Flutter: painting/inline_span.dart InlineSpanSemanticsInformation.operator ==
func (i InlineSpanSemanticsInformation) Equal(o InlineSpanSemanticsInformation) bool {
	return i.Text == o.Text && i.SemanticsLabel == o.SemanticsLabel && i.SemanticsLabelSet == o.SemanticsLabelSet &&
		i.SemanticsIdentifier == o.SemanticsIdentifier && sameValue(i.Recognizer, o.Recognizer) &&
		i.IsPlaceholder == o.IsPlaceholder && slices.Equal(i.StringAttributes, o.StringAttributes)
}

// String describes the information.
//
// Flutter: painting/inline_span.dart InlineSpanSemanticsInformation.toString
func (i InlineSpanSemanticsInformation) String() string {
	label := "null"
	if i.SemanticsLabelSet {
		label = i.SemanticsLabel
	}
	id := "null"
	if i.SemanticsIdentifier != "" {
		id = i.SemanticsIdentifier
	}
	rec := "null"
	if i.Recognizer != nil {
		rec = "recognizer"
	}
	return "InlineSpanSemanticsInformation{text: " + i.Text + ", semanticsLabel: " + label + ", semanticsIdentifier: " + id + ", recognizer: " + rec + "}"
}

// CombineSemanticsInfo walks the infos of spans and joins the ones that don't
// need their own node into one.
//
// Flutter: painting/inline_span.dart combineSemanticsInfo
func CombineSemanticsInfo(infoList []InlineSpanSemanticsInformation) []InlineSpanSemanticsInformation {
	var combined []InlineSpanSemanticsInformation
	var workingText, workingLabel strings.Builder
	var workingAttributes []ui.StringAttribute
	flush := func() {
		combined = append(combined, NewInlineSpanSemanticsInformation(workingText.String(), InlineSpanSemanticsInformation{
			SemanticsLabel: workingLabel.String(), SemanticsLabelSet: true, StringAttributes: workingAttributes,
		}))
	}
	for _, info := range infoList {
		if info.RequiresOwnNode {
			flush()
			workingText.Reset()
			workingLabel.Reset()
			workingAttributes = nil
			combined = append(combined, info)
			continue
		}
		workingText.WriteString(info.Text)
		effectiveLabel := info.Text
		if info.SemanticsLabelSet {
			effectiveLabel = info.SemanticsLabel
		}
		for _, attr := range info.StringAttributes {
			r := attr.Range()
			workingAttributes = append(workingAttributes,
				attr.CopyWithRange(ui.NewTextRange(r.Start+workingLabel.Len(), r.End+workingLabel.Len())))
		}
		workingLabel.WriteString(effectiveLabel)
	}
	flush()
	return combined
}

// SpanBuildOpts are the optional parameters of InlineSpan.Build.
//
// Flutter: painting/inline_span.dart InlineSpan.build
type SpanBuildOpts struct {
	// TextScaler scales the font sizes; the default is TextScalerNoScaling.
	TextScaler TextScaler
	// Dimensions are the sizes of the placeholders, in the order they appear.
	Dimensions []PlaceholderDimensions
}

// ToPlainTextOpts are the optional parameters of InlineSpan.ToPlainText. Both
// are true unless the Exclude field says otherwise.
//
// Flutter: painting/inline_span.dart InlineSpan.toPlainText
type ToPlainTextOpts struct {
	// ExcludeSemanticsLabels leaves out the semantics labels and uses the text.
	ExcludeSemanticsLabels bool
	// ExcludePlaceholders leaves out the placeholder characters.
	ExcludePlaceholders bool
}

// InlineSpan is an immutable span of inline content which forms part of a
// paragraph. TextSpan and PlaceholderSpan (WidgetSpan) implement it; embed
// InlineSpanBase for the style.
//
// Flutter: painting/inline_span.dart InlineSpan
type InlineSpan interface {
	foundation.DiagnosticableTree
	// Style is the style to apply to this span and its children, or nil.
	//
	// Flutter: painting/inline_span.dart InlineSpan.style
	Style() *TextStyle
	// Build adds the span to the builder.
	//
	// Flutter: painting/inline_span.dart InlineSpan.build
	Build(builder *ui.ParagraphBuilder, opts ...SpanBuildOpts)
	// VisitChildren calls the visitor on this span and its descendants in depth
	// first order, until it returns false; it returns false if it was stopped.
	//
	// Flutter: painting/inline_span.dart InlineSpan.visitChildren
	VisitChildren(visitor InlineSpanVisitor) bool
	// VisitDirectChildren calls the visitor on the direct children.
	//
	// Flutter: painting/inline_span.dart InlineSpan.visitDirectChildren
	VisitDirectChildren(visitor InlineSpanVisitor) bool
	// GetSpanForPositionVisitor returns this span if the position is in its
	// text, else advances the offset past it and returns nil.
	//
	// Flutter: painting/inline_span.dart InlineSpan.getSpanForPositionVisitor
	GetSpanForPositionVisitor(position ui.TextPosition, offset *Accumulator) InlineSpan
	// ComputeToPlainText writes the text of the span and its descendants.
	//
	// Flutter: painting/inline_span.dart InlineSpan.computeToPlainText
	ComputeToPlainText(buffer *strings.Builder, includeSemanticsLabels, includePlaceholders bool)
	// ComputeSemanticsInformation adds the semantics information of the span and
	// its descendants.
	//
	// Flutter: painting/inline_span.dart InlineSpan.computeSemanticsInformation
	ComputeSemanticsInformation(collector *[]InlineSpanSemanticsInformation)
	// CodeUnitAtVisitor returns the code unit at the index of the span's text,
	// advancing the offset.
	//
	// Flutter: painting/inline_span.dart InlineSpan.codeUnitAtVisitor
	CodeUnitAtVisitor(index int, offset *Accumulator) (byte, bool)
	// DebugAssertIsValid reports whether the span is valid.
	//
	// Flutter: painting/inline_span.dart InlineSpan.debugAssertIsValid
	DebugAssertIsValid() bool
	// CompareTo describes the difference to another span.
	//
	// Flutter: painting/inline_span.dart InlineSpan.compareTo
	CompareTo(other InlineSpan) RenderComparison
	// Equal is Dart's ==.
	//
	// Flutter: painting/inline_span.dart InlineSpan.operator ==
	Equal(other InlineSpan) bool
}

// InlineSpanBase holds the style that every InlineSpan has. Embed it.
//
// Flutter: painting/inline_span.dart InlineSpan
type InlineSpanBase struct{ style *TextStyle }

// NewInlineSpanBase creates the base with the style, which may be nil.
//
// Flutter: painting/inline_span.dart InlineSpan.new
func NewInlineSpanBase(style *TextStyle) InlineSpanBase { return InlineSpanBase{style} }

// Style is the style to apply to this span and its children.
//
// Flutter: painting/inline_span.dart InlineSpan.style
func (b InlineSpanBase) Style() *TextStyle { return b.style }

// EqualBase is the part of Dart's == that InlineSpan itself checks: the style.
//
// Flutter: painting/inline_span.dart InlineSpan.operator ==
func (b InlineSpanBase) EqualBase(other InlineSpan) bool {
	if other == nil {
		return false
	}
	return b.style.Equal(other.Style())
}

// DebugFillBase adds the properties of InlineSpan itself: the style.
//
// Flutter: painting/inline_span.dart InlineSpan.debugFillProperties
func (b InlineSpanBase) DebugFillBase(p *foundation.DiagnosticPropertiesBuilder) {
	p.DefaultDiagnosticsTreeStyle = foundation.DiagnosticsTreeStyleWhitespace
	if b.style != nil {
		b.style.DebugFillProperties(p)
	}
}

// InlineSpanGetSpanForPosition returns the span that contains the position, or
// nil.
//
// Flutter: painting/inline_span.dart InlineSpan.getSpanForPosition
func InlineSpanGetSpanForPosition(s InlineSpan, position ui.TextPosition) InlineSpan {
	s.DebugAssertIsValid()
	offset := NewAccumulator(0)
	var result InlineSpan
	s.VisitChildren(func(span InlineSpan) bool {
		result = span.GetSpanForPositionVisitor(position, offset)
		return result == nil
	})
	return result
}

// InlineSpanToPlainText returns the text of the span and its descendants.
//
// Flutter: painting/inline_span.dart InlineSpan.toPlainText
func InlineSpanToPlainText(s InlineSpan, opts ...ToPlainTextOpts) string {
	o := firstOpts(opts)
	var b strings.Builder
	s.ComputeToPlainText(&b, !o.ExcludeSemanticsLabels, !o.ExcludePlaceholders)
	return b.String()
}

// InlineSpanGetSemanticsInformation returns the semantics information of the
// span and its descendants.
//
// Flutter: painting/inline_span.dart InlineSpan.getSemanticsInformation
func InlineSpanGetSemanticsInformation(s InlineSpan) []InlineSpanSemanticsInformation {
	var collector []InlineSpanSemanticsInformation
	s.ComputeSemanticsInformation(&collector)
	return collector
}

// InlineSpanCodeUnitAt returns the code unit (UTF-8 byte) at the offset of the
// span's text, and false if there is none.
//
// Flutter: painting/inline_span.dart InlineSpan.codeUnitAt
func InlineSpanCodeUnitAt(s InlineSpan, index int) (byte, bool) {
	if index < 0 {
		return 0, false
	}
	offset := NewAccumulator(0)
	var result byte
	found := false
	s.VisitChildren(func(span InlineSpan) bool {
		result, found = span.CodeUnitAtVisitor(index, offset)
		return !found
	})
	return result, found
}

func spanBuildScaler(opts []SpanBuildOpts) (TextScaler, []PlaceholderDimensions) {
	o := firstOpts(opts)
	if o.TextScaler == nil {
		return TextScalerNoScaling, o.Dimensions
	}
	return o.TextScaler, o.Dimensions
}
