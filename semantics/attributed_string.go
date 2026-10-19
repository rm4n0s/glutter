package semantics

import (
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// The Unicode directional formatting characters that wrap a text of another
// direction.
//
// Flutter: services/text_formatter.dart (Unicode)
const (
	unicodeLRE = "\u202a"
	unicodeRLE = "\u202b"
	unicodePDF = "\u202c"
)

// AttributedString is a string with attributes on ranges of it.
//
// Flutter: semantics/semantics.dart AttributedString
type AttributedString struct {
	// String is the text.
	//
	// Flutter: semantics/semantics.dart AttributedString.string
	String string
	// Attributes are the attributes of ranges of the text.
	//
	// Flutter: semantics/semantics.dart AttributedString.attributes
	Attributes []ui.StringAttribute
}

// NewAttributedString creates a string with attributes. In debug builds it
// panics with the tag AttributedStringInvalid if the string is empty and has
// attributes, or an attribute is outside of the string.
//
// Flutter: semantics/semantics.dart AttributedString.new
func NewAttributedString(s string, attributes ...ui.StringAttribute) AttributedString {
	if foundation.DebugMode {
		if s == "" && len(attributes) > 0 {
			panic(errors.New("AttributedStringInvalid", "an empty AttributedString can not have attributes"))
		}
		for _, a := range attributes {
			if r := a.Range(); len(s) < r.Start || len(s) < r.End {
				panic(errors.New("AttributedStringInvalid", "the range of an attribute is outside of the string",
					"string", s, "range", r.String()))
			}
		}
	}
	return AttributedString{s, attributes}
}

// Add concatenates two strings; the attributes of the second move to its place.
//
// Flutter: semantics/semantics.dart AttributedString.+
func (a AttributedString) Add(other AttributedString) AttributedString {
	if a.String == "" {
		return other
	}
	if other.String == "" {
		return a
	}
	attrs := slices.Clone(a.Attributes)
	offset := len(a.String)
	for _, attr := range other.Attributes {
		r := attr.Range()
		attrs = append(attrs, attr.CopyWithRange(ui.TextRange{Start: r.Start + offset, End: r.End + offset}))
	}
	return NewAttributedString(a.String+other.String, attrs...)
}

// Equal is Dart's ==: the same text and equal attributes.
//
// Flutter: semantics/semantics.dart AttributedString.==
func (a AttributedString) Equal(o AttributedString) bool {
	return a.String == o.String && slices.EqualFunc(a.Attributes, o.Attributes, func(x, y ui.StringAttribute) bool { return x == y })
}

// GoString describes the text and its attributes.
//
// Flutter: semantics/semantics.dart AttributedString.toString
func (a AttributedString) GoString() string {
	return "AttributedString('" + a.String + "', attributes: " + attributesString(a.Attributes) + ")"
}

func attributesString(attrs []ui.StringAttribute) string {
	parts := make([]string, len(attrs))
	for i, attr := range attrs {
		parts[i] = describeAttribute(attr)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func describeAttribute(a ui.StringAttribute) string {
	r := a.Range()
	switch v := a.(type) {
	case ui.SpellOutStringAttribute:
		return "SpellOutStringAttribute(TextRange(start: " + itoa(r.Start) + ", end: " + itoa(r.End) + "))"
	case ui.LocaleStringAttribute:
		return "LocaleStringAttribute(TextRange(start: " + itoa(r.Start) + ", end: " + itoa(r.End) + "), " + v.Locale().LanguageTag() + ")"
	}
	return "StringAttribute(" + r.String() + ")"
}

// concatAttributedString appends the other string after a newline, wrapped in
// directional characters if the direction is different.
//
// Flutter: semantics/semantics.dart _concatAttributedString
func concatAttributedString(this AttributedString, thisDir ui.TextDirection, other AttributedString, otherDir ui.TextDirection) AttributedString {
	if other.String == "" {
		return this
	}
	if thisDir != otherDir && otherDir != ui.TextDirectionUnspecified {
		embedding := NewAttributedString(unicodeLRE)
		if otherDir == ui.TextDirectionRTL {
			embedding = NewAttributedString(unicodeRLE)
		}
		other = embedding.Add(other).Add(NewAttributedString(unicodePDF))
	}
	if this.String == "" {
		return other
	}
	return this.Add(NewAttributedString("\n")).Add(other)
}

// NewAttributedStringProperty creates a property that describes an attributed
// string, in quotes with its attributes. The value is nil for Dart's null. The
// property is not interesting if the string is empty, unless showWhenEmpty.
//
// Flutter: semantics/semantics.dart AttributedStringProperty
func NewAttributedStringProperty(name string, value *AttributedString, showWhenEmpty bool, opts ...foundation.DiagnosticsPropertyOpts) *foundation.DiagnosticsProperty {
	var o foundation.DiagnosticsPropertyOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	if !showWhenEmpty && (value == nil || value.String == "") {
		o.DefaultValue = value // makes the property not interesting
		if value == nil {
			o.DefaultValue = foundation.DefaultNull
		}
	}
	o.ValueToString = func(parent *foundation.TextTreeConfiguration) string {
		if value == nil {
			return "null"
		}
		text := value.String
		if parent != nil && !parent.LineBreakProperties {
			text = strings.ReplaceAll(text, "\n", `\n`)
		}
		if len(value.Attributes) == 0 {
			return `"` + text + `"`
		}
		return `"` + text + `" ` + attributesString(value.Attributes)
	}
	return foundation.NewDiagnosticsProperty(name, value, o)
}

type labelPart struct {
	text string
	dir  ui.TextDirection
}

// SemanticsLabelBuilder builds a label from parts that may have a direction
// of their own, joined by a separator.
//
// Flutter: semantics/semantics.dart SemanticsLabelBuilder
type SemanticsLabelBuilder struct {
	separator string
	direction ui.TextDirection
	parts     []labelPart
}

// SemanticsLabelBuilderOpts are the optional parameters of
// NewSemanticsLabelBuilder.
//
// Flutter: semantics/semantics.dart SemanticsLabelBuilder.new
type SemanticsLabelBuilderOpts struct {
	// Separator is put between the parts; the default is a space.
	Separator foundation.Opt[string]
	// TextDirection is the direction of the label; parts of another direction
	// are wrapped in directional characters.
	TextDirection ui.TextDirection
}

// NewSemanticsLabelBuilder creates a builder.
//
// Flutter: semantics/semantics.dart SemanticsLabelBuilder.new
func NewSemanticsLabelBuilder(opts ...SemanticsLabelBuilderOpts) *SemanticsLabelBuilder {
	var o SemanticsLabelBuilderOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	return &SemanticsLabelBuilder{separator: o.Separator.OrElse(" "), direction: o.TextDirection}
}

// AddPart adds a part; empty parts are ignored. The direction is the one of the
// part, or unspecified to use the one of the builder.
//
// Flutter: semantics/semantics.dart SemanticsLabelBuilder.addPart
func (b *SemanticsLabelBuilder) AddPart(label string, direction ...ui.TextDirection) {
	if label == "" {
		return
	}
	var d ui.TextDirection
	if len(direction) > 0 {
		d = direction[0]
	}
	b.parts = append(b.parts, labelPart{label, d})
}

// IsEmpty is whether there is no part.
//
// Flutter: semantics/semantics.dart SemanticsLabelBuilder.isEmpty
func (b *SemanticsLabelBuilder) IsEmpty() bool { return len(b.parts) == 0 }

// Length is the number of parts.
//
// Flutter: semantics/semantics.dart SemanticsLabelBuilder.length
func (b *SemanticsLabelBuilder) Length() int { return len(b.parts) }

// Build joins the parts.
//
// Flutter: semantics/semantics.dart SemanticsLabelBuilder.build
func (b *SemanticsLabelBuilder) Build() string {
	if len(b.parts) == 0 {
		return ""
	}
	if len(b.parts) == 1 {
		return b.parts[0].text
	}
	var sb strings.Builder
	sb.WriteString(b.parts[0].text)
	for _, p := range b.parts[1:] {
		dir := p.dir
		if dir == ui.TextDirectionUnspecified {
			dir = b.direction
		}
		if b.separator != "" {
			sb.WriteString(b.separator)
		}
		text := p.text
		if b.direction != ui.TextDirectionUnspecified && dir != ui.TextDirectionUnspecified && b.direction != dir {
			embedding := unicodeLRE
			if dir == ui.TextDirectionRTL {
				embedding = unicodeRLE
			}
			text = embedding + text + unicodePDF
		}
		sb.WriteString(text)
	}
	return sb.String()
}

// Clear removes the parts.
//
// Flutter: semantics/semantics.dart SemanticsLabelBuilder.clear
func (b *SemanticsLabelBuilder) Clear() { b.parts = b.parts[:0] }
