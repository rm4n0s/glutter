package painting

import (
	"strings"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// PlaceholderSpan is an immutable placeholder that is embedded inline within
// text. WidgetSpan implements it.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpan
type PlaceholderSpan interface {
	InlineSpan
	// Alignment is how the placeholder aligns with the text.
	//
	// Flutter: painting/placeholder_span.dart PlaceholderSpan.alignment
	Alignment() ui.PlaceholderAlignment
	// Baseline is the baseline the placeholder aligns on, for the baseline
	// alignment; unspecified is null.
	//
	// Flutter: painting/placeholder_span.dart PlaceholderSpan.baseline
	Baseline() ui.TextBaseline
}

// PlaceholderSpanBase is what a PlaceholderSpan has: the alignment, the
// baseline and the style. Embed it.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpan
type PlaceholderSpanBase struct {
	InlineSpanBase
	alignment ui.PlaceholderAlignment
	baseline  ui.TextBaseline
}

// PlaceholderSpanOpts are the optional parameters of NewPlaceholderSpanBase.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpan.new
type PlaceholderSpanOpts struct {
	// Alignment defaults to bottom.
	Alignment ui.PlaceholderAlignment
	// Baseline is the baseline to align on.
	Baseline ui.TextBaseline
	// Style is the style of the span.
	Style *TextStyle
}

// NewPlaceholderSpanBase creates the base of a placeholder span.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpan.new
func NewPlaceholderSpanBase(opts ...PlaceholderSpanOpts) PlaceholderSpanBase {
	o := firstOpts(opts)
	if o.Alignment == ui.PlaceholderAlignmentUnspecified {
		o.Alignment = ui.PlaceholderAlignmentBottom
	}
	return PlaceholderSpanBase{NewInlineSpanBase(o.Style), o.Alignment, o.Baseline}
}

// Alignment is how the placeholder aligns with the text.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpan.alignment
func (b PlaceholderSpanBase) Alignment() ui.PlaceholderAlignment { return b.alignment }

// Baseline is the baseline the placeholder aligns on.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpan.baseline
func (b PlaceholderSpanBase) Baseline() ui.TextBaseline { return b.baseline }

// ComputeToPlainText writes the placeholder character.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpan.computeToPlainText
func (b PlaceholderSpanBase) ComputeToPlainText(buffer *strings.Builder, includeSemanticsLabels, includePlaceholders bool) {
	if includePlaceholders {
		buffer.WriteString(placeholderText)
	}
}

// ComputeSemanticsInformation adds the placeholder information.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpan.computeSemanticsInformation
func (b PlaceholderSpanBase) ComputeSemanticsInformation(collector *[]InlineSpanSemanticsInformation) {
	*collector = append(*collector, InlineSpanSemanticsPlaceholder())
}

// DebugFillBase adds the style, the alignment and the baseline.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpan.debugFillProperties
func (b PlaceholderSpanBase) DebugFillBase(p *foundation.DiagnosticPropertiesBuilder) {
	b.InlineSpanBase.DebugFillBase(p)
	enumNull := foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultNull}
	p.Add(foundation.NewEnumProperty("alignment", optEnum(b.alignment), enumNull))
	p.Add(foundation.NewEnumProperty("baseline", optEnum(b.baseline), enumNull))
}

// DebugAssertIsValid panics in debug builds with the tag
// PlaceholderSpanNotImplemented: a placeholder span must be a WidgetSpan, which
// implements this itself.
//
// Flutter: painting/placeholder_span.dart PlaceholderSpan.debugAssertIsValid
func (b PlaceholderSpanBase) DebugAssertIsValid() bool {
	if foundation.DebugMode {
		panic(errors.New(tagPlaceholderSpanNotImplemented, "Consider implementing the WidgetSpan interface instead."))
	}
	return true
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagPlaceholderSpanNotImplemented = "PlaceholderSpanNotImplemented"
)
