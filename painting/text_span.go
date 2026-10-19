package painting

import (
	"strings"
	"unicode/utf8"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// TextSpanOpts are the optional parameters of NewTextSpan.
//
// Flutter: painting/text_span.dart TextSpan
type TextSpanOpts struct {
	// Text is the text contained in the span; None is null.
	Text foundation.Opt[string]
	// Children are the spans that follow the text; nil is null.
	Children []InlineSpan
	// Style is the style to apply to the text and the children.
	Style *TextStyle
	// Recognizer receives the pointer down events on the span.
	Recognizer gestures.GestureRecognizer
	// MouseCursor is the cursor over the span; the default is MouseCursorDefer,
	// and the click cursor when there is a recognizer.
	MouseCursor services.MouseCursor
	// OnEnter is called when a mouse pointer enters the span.
	OnEnter services.PointerEnterEventListener
	// OnExit is called when a mouse pointer leaves the span.
	OnExit services.PointerExitEventListener
	// SemanticsLabel replaces the text for accessibility; it needs Text.
	SemanticsLabel foundation.Opt[string]
	// SemanticsIdentifier is the identifier of the semantics node.
	SemanticsIdentifier foundation.Opt[string]
	// Locale is the locale of the text, for accessibility.
	Locale foundation.Opt[ui.Locale]
	// SpellOut says that the text is spoken character by character.
	SpellOut foundation.Opt[bool]
}

// TextSpan is an immutable span of text.
//
// Flutter: painting/text_span.dart TextSpan
type TextSpan struct {
	InlineSpanBase
	// Text is the text contained in the span.
	Text foundation.Opt[string]
	// Children are the spans that follow the text.
	Children []InlineSpan
	// Recognizer receives the pointer down events on the span.
	Recognizer gestures.GestureRecognizer
	// MouseCursor is the cursor over the span.
	MouseCursor services.MouseCursor
	// SemanticsLabel replaces the text for accessibility.
	SemanticsLabel foundation.Opt[string]
	// SemanticsIdentifier is the identifier of the semantics node.
	SemanticsIdentifier foundation.Opt[string]
	// Locale is the locale of the text, for accessibility.
	Locale foundation.Opt[ui.Locale]
	// SpellOut says that the text is spoken character by character.
	SpellOut foundation.Opt[bool]

	onEnter services.PointerEnterEventListener
	onExit  services.PointerExitEventListener
}

var (
	_ services.MouseTrackerAnnotation = (*TextSpan)(nil)
	_ gestures.HitTestTarget          = (*TextSpan)(nil)
	_ InlineSpan                      = (*TextSpan)(nil)
)

// NewTextSpan creates a span of text. In debug builds a semantics label without
// a text panics with the tag TextSpanLabelWithoutText.
//
// Flutter: painting/text_span.dart TextSpan.new
func NewTextSpan(opts ...TextSpanOpts) *TextSpan {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Text.IsNone() && o.SemanticsLabel.IsSome() {
		panic(errors.New("TextSpanLabelWithoutText", "a TextSpan with a semanticsLabel needs a text"))
	}
	cursor := o.MouseCursor
	if cursor == nil {
		if o.Recognizer == nil {
			cursor = services.MouseCursorDefer
		} else {
			cursor = services.SystemMouseCursors.Click
		}
	}
	return &TextSpan{
		InlineSpanBase: NewInlineSpanBase(o.Style), Text: o.Text, Children: o.Children, Recognizer: o.Recognizer,
		MouseCursor: cursor, onEnter: o.OnEnter, onExit: o.OnExit, SemanticsLabel: o.SemanticsLabel,
		SemanticsIdentifier: o.SemanticsIdentifier, Locale: o.Locale, SpellOut: o.SpellOut,
	}
}

// Cursor is the cursor over the span.
//
// Flutter: painting/text_span.dart TextSpan.cursor
func (s *TextSpan) Cursor() services.MouseCursor { return s.MouseCursor }

// OnEnter is the callback for a mouse pointer entering the span.
//
// Flutter: painting/text_span.dart TextSpan.onEnter
func (s *TextSpan) OnEnter() services.PointerEnterEventListener { return s.onEnter }

// OnExit is the callback for a mouse pointer leaving the span.
//
// Flutter: painting/text_span.dart TextSpan.onExit
func (s *TextSpan) OnExit() services.PointerExitEventListener { return s.onExit }

// ValidForMouseTracker is always true for a text span.
//
// Flutter: painting/text_span.dart TextSpan.validForMouseTracker
func (s *TextSpan) ValidForMouseTracker() bool { return true }

// HandleEvent gives a pointer down event to the recognizer.
//
// Flutter: painting/text_span.dart TextSpan.handleEvent
func (s *TextSpan) HandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) {
	if down, ok := event.(*gestures.PointerDownEvent); ok && s.Recognizer != nil {
		s.Recognizer.AddPointer(down)
	}
}

// Build adds the span, with its style and its children, to the builder.
//
// Flutter: painting/text_span.dart TextSpan.build
func (s *TextSpan) Build(builder *ui.ParagraphBuilder, opts ...SpanBuildOpts) {
	s.DebugAssertIsValid()
	scaler, _ := spanBuildScaler(opts)
	hasStyle := s.style != nil
	if hasStyle {
		builder.PushStyle(s.style.GetTextStyle(GetTextStyleOpts{TextScaler: scaler}))
	}
	if text, ok := s.Text.Get(); ok {
		// Flutter replaces text it can't encode (lone surrogates) with U+FFFD;
		// Go strings can hold invalid UTF-8, so do the same for those.
		if !utf8.ValidString(text) {
			text = strings.ToValidUTF8(text, "�")
		}
		builder.AddText(text)
	}
	for _, child := range s.Children {
		child.Build(builder, opts...)
	}
	if hasStyle {
		builder.Pop()
	}
}

// VisitChildren calls the visitor on this span, if it has text, then on the
// descendants.
//
// Flutter: painting/text_span.dart TextSpan.visitChildren
func (s *TextSpan) VisitChildren(visitor InlineSpanVisitor) bool {
	if s.Text.IsSome() && !visitor(s) {
		return false
	}
	for _, child := range s.Children {
		if !child.VisitChildren(visitor) {
			return false
		}
	}
	return true
}

// VisitDirectChildren calls the visitor on the direct children.
//
// Flutter: painting/text_span.dart TextSpan.visitDirectChildren
func (s *TextSpan) VisitDirectChildren(visitor InlineSpanVisitor) bool {
	for _, child := range s.Children {
		if !visitor(child) {
			return false
		}
	}
	return true
}

// GetSpanForPositionVisitor returns this span if the position is in its text.
//
// Flutter: painting/text_span.dart TextSpan.getSpanForPositionVisitor
func (s *TextSpan) GetSpanForPositionVisitor(position ui.TextPosition, offset *Accumulator) InlineSpan {
	text, ok := s.Text.Get()
	if !ok || text == "" {
		return nil
	}
	targetOffset := position.Offset
	endOffset := offset.Value() + len(text)
	if (offset.Value() == targetOffset && position.Affinity == ui.TextAffinityDownstream) ||
		(offset.Value() < targetOffset && targetOffset < endOffset) ||
		(endOffset == targetOffset && position.Affinity == ui.TextAffinityUpstream) {
		return s
	}
	offset.Increment(len(text))
	return nil
}

// ComputeToPlainText writes the text of the span and its descendants.
//
// Flutter: painting/text_span.dart TextSpan.computeToPlainText
func (s *TextSpan) ComputeToPlainText(buffer *strings.Builder, includeSemanticsLabels, includePlaceholders bool) {
	s.DebugAssertIsValid()
	if label, ok := s.SemanticsLabel.Get(); ok && includeSemanticsLabels {
		buffer.WriteString(label)
	} else if text, ok := s.Text.Get(); ok {
		buffer.WriteString(text)
	}
	for _, child := range s.Children {
		child.ComputeToPlainText(buffer, includeSemanticsLabels, includePlaceholders)
	}
}

// ComputeSemanticsInformation adds the semantics information of the span and its
// descendants.
//
// Flutter: painting/text_span.dart TextSpan.computeSemanticsInformation
func (s *TextSpan) ComputeSemanticsInformation(collector *[]InlineSpanSemanticsInformation) {
	s.computeSemantics(collector, foundation.None[ui.Locale](), false)
}

func (s *TextSpan) computeSemantics(collector *[]InlineSpanSemanticsInformation, inheritedLocale foundation.Opt[ui.Locale], inheritedSpellOut bool) {
	s.DebugAssertIsValid()
	effectiveLocale := orOpt(s.Locale, inheritedLocale)
	effectiveSpellOut := s.SpellOut.OrElse(inheritedSpellOut)
	if text, ok := s.Text.Get(); ok {
		textLength := len(text)
		if label, ok := s.SemanticsLabel.Get(); ok {
			textLength = len(label)
		}
		var attrs []ui.StringAttribute
		if effectiveSpellOut && textLength > 0 {
			attrs = append(attrs, ui.NewSpellOutStringAttribute(ui.NewTextRange(0, textLength)))
		}
		if loc, ok := effectiveLocale.Get(); ok && textLength > 0 {
			attrs = append(attrs, ui.NewLocaleStringAttribute(ui.NewTextRange(0, textLength), loc))
		}
		info := InlineSpanSemanticsInformation{StringAttributes: attrs, Recognizer: s.Recognizer}
		if label, ok := s.SemanticsLabel.Get(); ok {
			info.SemanticsLabel, info.SemanticsLabelSet = label, true
		}
		info.SemanticsIdentifier = s.SemanticsIdentifier.OrElse("")
		*collector = append(*collector, NewInlineSpanSemanticsInformation(text, info))
	}
	for _, child := range s.Children {
		if ts, ok := child.(*TextSpan); ok {
			ts.computeSemantics(collector, effectiveLocale, effectiveSpellOut)
		} else {
			child.ComputeSemanticsInformation(collector)
		}
	}
}

// CodeUnitAtVisitor returns the code unit at the index of the span's text.
//
// Flutter: painting/text_span.dart TextSpan.codeUnitAtVisitor
func (s *TextSpan) CodeUnitAtVisitor(index int, offset *Accumulator) (byte, bool) {
	text, ok := s.Text.Get()
	if !ok {
		return 0, false
	}
	local := index - offset.Value()
	if foundation.DebugMode && local < 0 {
		panic(errors.New("TextSpanOffsetNegative", "the index is before the span", "index", index, "offset", offset.Value()))
	}
	offset.Increment(len(text))
	if local < len(text) {
		return text[local], true
	}
	return 0, false
}

// DebugAssertIsValid checks the children, in debug builds, and returns true.
//
// Flutter: painting/text_span.dart TextSpan.debugAssertIsValid
func (s *TextSpan) DebugAssertIsValid() bool {
	if foundation.DebugMode {
		for _, child := range s.Children {
			child.DebugAssertIsValid()
		}
	}
	return true
}

// CompareTo describes the difference between this span and another.
//
// Flutter: painting/text_span.dart TextSpan.compareTo
func (s *TextSpan) CompareTo(other InlineSpan) RenderComparison {
	if InlineSpan(s) == other {
		return RenderComparisonIdentical
	}
	o, ok := other.(*TextSpan)
	if !ok {
		return RenderComparisonLayout
	}
	if o.Text != s.Text || len(s.Children) != len(o.Children) || (s.style == nil) != (o.style == nil) {
		return RenderComparisonLayout
	}
	result := RenderComparisonIdentical
	if !sameValue(s.Recognizer, o.Recognizer) {
		result = RenderComparisonMetadata
	}
	if s.style != nil {
		if c := s.style.CompareTo(o.style); c > result {
			result = c
		}
		if result == RenderComparisonLayout {
			return result
		}
	}
	for i, child := range s.Children {
		if c := child.CompareTo(o.Children[i]); c > result {
			result = c
		}
		if result == RenderComparisonLayout {
			return result
		}
	}
	return result
}

// Equal is Dart's == on spans.
//
// Flutter: painting/text_span.dart TextSpan.operator ==
func (s *TextSpan) Equal(other InlineSpan) bool {
	if InlineSpan(s) == other {
		return true
	}
	o, ok := other.(*TextSpan)
	if !ok || o == nil {
		return false
	}
	if !s.EqualBase(o) {
		return false
	}
	if o.Text != s.Text || !sameValue(o.Recognizer, s.Recognizer) || o.SemanticsLabel != s.SemanticsLabel ||
		o.SemanticsIdentifier != s.SemanticsIdentifier || o.MouseCursor != s.MouseCursor || len(o.Children) != len(s.Children) {
		return false
	}
	// Function values can't be compared in Go; two spans with callbacks are equal
	// only if both have none or they are the same span.
	if o.onEnter != nil || s.onEnter != nil || o.onExit != nil || s.onExit != nil {
		return false
	}
	for i, c := range s.Children {
		if !c.Equal(o.Children[i]) {
			return false
		}
	}
	return true
}

// ToStringShort names the span.
//
// Flutter: painting/text_span.dart TextSpan.toStringShort
func (s *TextSpan) ToStringShort() string { return "TextSpan" }

// String describes the span.
//
// Flutter: painting/text_span.dart TextSpan.toString
func (s *TextSpan) String() string { return foundation.DiagnosticableString(s) }

// DebugFillProperties adds the style, the text, the recognizer, the callbacks,
// the cursor and the semantics label.
//
// Flutter: painting/text_span.dart TextSpan.debugFillProperties
func (s *TextSpan) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	s.DebugFillBase(p)
	p.Add(foundation.NewStringProperty("text", s.Text, foundation.StringPropertyOpts{
		ShowName: foundation.Some(false), DefaultValue: foundation.DefaultNull}))
	if s.style == nil && s.Text.IsNone() && s.Children == nil {
		p.Add(foundation.NewDiagnosticsNodeMessage("(empty)"))
	}
	recOpts := foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}
	if s.Recognizer != nil {
		recOpts.Description = foundation.RuntimeTypeName(s.Recognizer)
	}
	p.Add(foundation.NewDiagnosticsProperty("recognizer", s.Recognizer, recOpts))
	p.Add(foundation.NewFlagsSummary("callbacks", []foundation.FlagEntry{
		{Description: "enter", Present: s.onEnter != nil}, {Description: "exit", Present: s.onExit != nil}}))
	p.Add(foundation.NewDiagnosticsProperty("mouseCursor", s.MouseCursor, foundation.DiagnosticsPropertyOpts{DefaultValue: services.MouseCursorDefer}))
	if s.SemanticsLabel.IsSome() {
		p.Add(foundation.NewStringProperty("semanticsLabel", s.SemanticsLabel))
	}
	if s.SemanticsIdentifier.IsSome() {
		p.Add(foundation.NewStringProperty("semanticsIdentifier", s.SemanticsIdentifier))
	}
}

// DebugDescribeChildren describes the child spans.
//
// Flutter: painting/text_span.dart TextSpan.debugDescribeChildren
func (s *TextSpan) DebugDescribeChildren() []foundation.DiagnosticsNode {
	out := make([]foundation.DiagnosticsNode, 0, len(s.Children))
	for _, c := range s.Children {
		out = append(out, foundation.ToDiagnosticsNode(c))
	}
	return out
}
