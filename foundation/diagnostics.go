package foundation

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// DiagnosticLevel is the severity of a diagnostic: how interesting it is, and
// so whether a given dump shows it. The levels are ordered from the least to
// the most important; a dump shows the nodes at or above its minimum level.
//
// Flutter: foundation/diagnostics.dart DiagnosticLevel
type DiagnosticLevel int

// The values of DiagnosticLevel follow Flutter's declaration order.
const (
	// DiagnosticLevelUnspecified is the zero value: the default (info).
	//
	// Flutter: no counterpart
	DiagnosticLevelUnspecified DiagnosticLevel = iota
	// DiagnosticLevelHidden is a diagnostic that is not shown, unless a dump
	// asks for hidden ones.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticLevel.hidden
	DiagnosticLevelHidden
	// DiagnosticLevelFine is a diagnostic that is only shown in the most
	// verbose dumps, such as a property that has its default value.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticLevel.fine
	DiagnosticLevelFine
	// DiagnosticLevelDebug is a diagnostic useful when debugging.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticLevel.debug
	DiagnosticLevelDebug
	// DiagnosticLevelInfo is a diagnostic with ordinary information.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticLevel.info
	DiagnosticLevelInfo
	// DiagnosticLevelWarning is a diagnostic that something may be wrong.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticLevel.warning
	DiagnosticLevelWarning
	// DiagnosticLevelHint is a diagnostic that tells how to fix a problem.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticLevel.hint
	DiagnosticLevelHint
	// DiagnosticLevelSummary is a short summary of a problem.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticLevel.summary
	DiagnosticLevelSummary
	// DiagnosticLevelError is a diagnostic for an error.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticLevel.error
	DiagnosticLevelError
	// DiagnosticLevelOff is a level above every diagnostic: a dump at this
	// level shows nothing.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticLevel.off
	DiagnosticLevelOff
)

var diagnosticLevelNames = [...]string{"unspecified", "hidden", "fine", "debug", "info", "warning", "hint", "summary", "error", "off"}

// Name returns the name Dart's Enum.name gives the value, such as "info".
func (l DiagnosticLevel) Name() string {
	if l >= 0 && int(l) < len(diagnosticLevelNames) {
		return diagnosticLevelNames[l]
	}
	return "DiagnosticLevel(" + strconv.Itoa(int(l)) + ")"
}

// String returns the name Dart prints for the value, such as
// "DiagnosticLevel.info".
func (l DiagnosticLevel) String() string {
	if l >= 0 && int(l) < len(diagnosticLevelNames) {
		return "DiagnosticLevel." + diagnosticLevelNames[l]
	}
	return l.Name()
}

// orDefault returns def when l is the zero value.
func (l DiagnosticLevel) orDefault(def DiagnosticLevel) DiagnosticLevel {
	if l == DiagnosticLevelUnspecified {
		return def
	}
	return l
}

// DiagnosticsTreeStyle is the style in which a tree of diagnostics nodes is
// drawn as text: how its branches, properties and children are laid out.
//
// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle
type DiagnosticsTreeStyle int

// The values of DiagnosticsTreeStyle follow Flutter's declaration order.
const (
	// DiagnosticsTreeStyleUnspecified is the zero value: the node's default
	// style.
	//
	// Flutter: no counterpart (Dart's null)
	DiagnosticsTreeStyleUnspecified DiagnosticsTreeStyle = iota
	// DiagnosticsTreeStyleNone does not draw the node as text.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.none
	DiagnosticsTreeStyleNone
	// DiagnosticsTreeStyleSparse draws a tree with lines and a blank line
	// after a node with properties and no children.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.sparse
	DiagnosticsTreeStyleSparse
	// DiagnosticsTreeStyleOffstage draws a node that is not on the screen
	// with dashes.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.offstage
	DiagnosticsTreeStyleOffstage
	// DiagnosticsTreeStyleDense draws a tree without blank lines and with the
	// properties on the line of the description.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.dense
	DiagnosticsTreeStyleDense
	// DiagnosticsTreeStyleTransition draws a subtree in a box, for a node
	// that is in transition.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.transition
	DiagnosticsTreeStyleTransition
	// DiagnosticsTreeStyleError draws an error message in a box.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.error
	DiagnosticsTreeStyleError
	// DiagnosticsTreeStyleWhitespace draws a tree with indentation instead of
	// lines.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.whitespace
	DiagnosticsTreeStyleWhitespace
	// DiagnosticsTreeStyleFlat draws a tree without indentation.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.flat
	DiagnosticsTreeStyleFlat
	// DiagnosticsTreeStyleSingleLine draws the node on one line, as a
	// property usually is.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.singleLine
	DiagnosticsTreeStyleSingleLine
	// DiagnosticsTreeStyleErrorProperty draws a property of an error message.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.errorProperty
	DiagnosticsTreeStyleErrorProperty
	// DiagnosticsTreeStyleShallow draws a node without its children.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.shallow
	DiagnosticsTreeStyleShallow
	// DiagnosticsTreeStyleTruncateChildren draws a short list of the
	// descendants of the node.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsTreeStyle.truncateChildren
	DiagnosticsTreeStyleTruncateChildren
)

var diagnosticsTreeStyleNames = [...]string{
	"unspecified", "none", "sparse", "offstage", "dense", "transition", "error",
	"whitespace", "flat", "singleLine", "errorProperty", "shallow", "truncateChildren",
}

// Name returns the name Dart's Enum.name gives the value, such as "sparse".
func (s DiagnosticsTreeStyle) Name() string {
	if s >= 0 && int(s) < len(diagnosticsTreeStyleNames) {
		return diagnosticsTreeStyleNames[s]
	}
	return "DiagnosticsTreeStyle(" + strconv.Itoa(int(s)) + ")"
}

// String returns the name Dart prints for the value, such as
// "DiagnosticsTreeStyle.sparse".
func (s DiagnosticsTreeStyle) String() string {
	if s >= 0 && int(s) < len(diagnosticsTreeStyleNames) {
		return "DiagnosticsTreeStyle." + diagnosticsTreeStyleNames[s]
	}
	return s.Name()
}

// orDefault returns def when s is the zero value.
func (s DiagnosticsTreeStyle) orDefault(def DiagnosticsTreeStyle) DiagnosticsTreeStyle {
	if s == DiagnosticsTreeStyleUnspecified {
		return def
	}
	return s
}

// TextTreeConfiguration configures how a DiagnosticsTreeStyle draws a tree as
// text. The configurations of the styles are the package values
// SparseTextConfiguration and the like; do not change them.
//
// Flutter: foundation/diagnostics.dart TextTreeConfiguration
type TextTreeConfiguration struct {
	// PrefixLineOne is the prefix of the first line of a child.
	PrefixLineOne string
	// SuffixLineOne is the suffix of the first line of the node.
	SuffixLineOne string
	// PrefixOtherLines is the prefix of the other lines of a child.
	PrefixOtherLines string
	// PrefixLastChildLineOne is the prefix of the first line of the last
	// child.
	PrefixLastChildLineOne string
	// PrefixOtherLinesRootNode is the prefix of the other lines of the root
	// node.
	PrefixOtherLinesRootNode string
	// PropertyPrefixIfChildren is the prefix of the properties of a node that
	// has children.
	PropertyPrefixIfChildren string
	// PropertyPrefixNoChildren is the prefix of the properties of a node
	// without children.
	PropertyPrefixNoChildren string
	// LinkCharacter is the string that links a node to its siblings below.
	LinkCharacter string
	// ChildLinkSpace is as much space as LinkCharacter takes.
	ChildLinkSpace string
	// LineBreak is what ends a line.
	LineBreak string
	// LineBreakProperties is whether each property is on its own line.
	LineBreakProperties bool
	// BeforeName is written before the name.
	BeforeName string
	// AfterName is written after the name.
	AfterName string
	// AfterDescriptionIfBody is written after the description of a node that
	// has properties or children.
	AfterDescriptionIfBody string
	// AfterDescription is written after the description.
	AfterDescription string
	// BeforeProperties is written before the properties.
	BeforeProperties string
	// AfterProperties is written after the properties.
	AfterProperties string
	// MandatoryAfterProperties is written after the properties, even when
	// there are none.
	MandatoryAfterProperties string
	// PropertySeparator separates the properties.
	PropertySeparator string
	// BodyIndent indents the properties and the children.
	BodyIndent string
	// ShowChildren is whether the children are drawn.
	ShowChildren bool
	// AddBlankLineIfNoChildren is whether a node without children is followed
	// by a blank line.
	AddBlankLineIfNoChildren bool
	// IsNameOnOwnLine is whether the name has a line of its own.
	IsNameOnOwnLine bool
	// Footer is the line that ends a node's body.
	Footer string
	// MandatoryFooter is a line that ends the root node and is stretched to
	// the width of the text.
	MandatoryFooter string
	// IsBlankLineBetweenPropertiesAndChildren is whether a blank line
	// separates the properties from the children.
	IsBlankLineBetweenPropertiesAndChildren bool
}

// newTextTreeConfiguration builds a configuration from Flutter's defaults and
// the changes the function makes to them.
func newTextTreeConfiguration(set func(c *TextTreeConfiguration)) *TextTreeConfiguration {
	c := &TextTreeConfiguration{
		LineBreak:                               "\n",
		LineBreakProperties:                     true,
		AfterName:                               ":",
		ShowChildren:                            true,
		AddBlankLineIfNoChildren:                true,
		IsBlankLineBetweenPropertiesAndChildren: true,
	}
	set(c)
	c.ChildLinkSpace = strings.Repeat(" ", strLen(c.LinkCharacter))
	return c
}

// strLen is the length of s the way Dart counts it, in UTF-16 code units.
func strLen(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return len(utf16.Encode([]rune(s)))
		}
	}
	return len(s)
}

// trimRight removes white space from the end of s, as Dart's trimRight does.
func trimRight(s string) string { return strings.TrimRightFunc(s, unicode.IsSpace) }

// SparseTextConfiguration draws a tree with lines; it is the default style.
// It never changes.
//
// Flutter: foundation/diagnostics.dart sparseTextConfiguration
var SparseTextConfiguration = newTextTreeConfiguration(func(c *TextTreeConfiguration) {
	c.PrefixLineOne = "├─"
	c.PrefixOtherLines = " "
	c.PrefixLastChildLineOne = "└─"
	c.LinkCharacter = "│"
	c.PropertyPrefixIfChildren = "│ "
	c.PropertyPrefixNoChildren = "  "
	c.PrefixOtherLinesRootNode = " "
})

// DashedTextConfiguration draws a tree with dashed lines, for a node that is
// not on the screen. It never changes.
//
// Flutter: foundation/diagnostics.dart dashedTextConfiguration
var DashedTextConfiguration = newTextTreeConfiguration(func(c *TextTreeConfiguration) {
	c.PrefixLineOne = "╎╌"
	c.PrefixLastChildLineOne = "└╌"
	c.PrefixOtherLines = " "
	c.LinkCharacter = "╎"
	// Intentionally not set as a dashed line as that would make the properties
	// look like they were disabled.
	c.PropertyPrefixIfChildren = "│ "
	c.PropertyPrefixNoChildren = "  "
	c.PrefixOtherLinesRootNode = " "
})

// DenseTextConfiguration draws a tree without blank lines and with the
// properties on one line. It never changes.
//
// Flutter: foundation/diagnostics.dart denseTextConfiguration
var DenseTextConfiguration = newTextTreeConfiguration(func(c *TextTreeConfiguration) {
	c.PropertySeparator = ", "
	c.BeforeProperties = "("
	c.AfterProperties = ")"
	c.LineBreakProperties = false
	c.PrefixLineOne = "├"
	c.PrefixOtherLines = ""
	c.PrefixLastChildLineOne = "└"
	c.LinkCharacter = "│"
	c.PropertyPrefixIfChildren = "│"
	c.PropertyPrefixNoChildren = " "
	c.PrefixOtherLinesRootNode = ""
	c.AddBlankLineIfNoChildren = false
	c.IsBlankLineBetweenPropertiesAndChildren = false
})

// TransitionTextConfiguration draws a subtree in a box. It never changes.
//
// Flutter: foundation/diagnostics.dart transitionTextConfiguration
var TransitionTextConfiguration = newTextTreeConfiguration(func(c *TextTreeConfiguration) {
	c.PrefixLineOne = "╞═╦══ "
	c.PrefixLastChildLineOne = "╘═╦══ "
	c.PrefixOtherLines = " ║ "
	c.Footer = " ╚═══════════"
	c.LinkCharacter = "│"
	// Subtree boundaries are clear due to the border around the node so omit
	// the property prefix.
	c.PropertyPrefixIfChildren = ""
	c.PropertyPrefixNoChildren = ""
	c.PrefixOtherLinesRootNode = ""
	c.AfterName = " ═══"
	// Add a colon after the description if the node has a body to make the
	// connection between the description and the body clearer.
	c.AfterDescriptionIfBody = ":"
	// Members are indented an extra two spaces to disambiguate as the
	// description is placed within the box instead of along side the name as
	// is the case for other styles.
	c.BodyIndent = "  "
	c.IsNameOnOwnLine = true
	// No need to add a blank line as the footer makes the boundary of this
	// subtree unambiguous.
	c.AddBlankLineIfNoChildren = false
	c.IsBlankLineBetweenPropertiesAndChildren = false
})

// ErrorTextConfiguration draws an error message in a box. It never changes.
//
// Flutter: foundation/diagnostics.dart errorTextConfiguration
var ErrorTextConfiguration = newTextTreeConfiguration(func(c *TextTreeConfiguration) {
	c.PrefixLineOne = "╞═╦"
	c.PrefixLastChildLineOne = "╘═╦"
	c.PrefixOtherLines = " ║ "
	c.Footer = " ╚═══════════"
	c.LinkCharacter = "│"
	// Subtree boundaries are clear due to the border around the node so omit
	// the property prefix.
	c.PropertyPrefixIfChildren = ""
	c.PropertyPrefixNoChildren = ""
	c.PrefixOtherLinesRootNode = ""
	c.BeforeName = "══╡ "
	c.SuffixLineOne = " ╞══"
	c.MandatoryFooter = "═════"
	// No need to add a blank line as the footer makes the boundary of this
	// subtree unambiguous.
	c.AddBlankLineIfNoChildren = false
	c.IsBlankLineBetweenPropertiesAndChildren = false
})

// WhitespaceTextConfiguration draws a tree with indentation instead of lines.
// It never changes.
//
// Flutter: foundation/diagnostics.dart whitespaceTextConfiguration
var WhitespaceTextConfiguration = newTextTreeConfiguration(func(c *TextTreeConfiguration) {
	c.PrefixLineOne = ""
	c.PrefixLastChildLineOne = ""
	c.PrefixOtherLines = " "
	c.PrefixOtherLinesRootNode = "  "
	c.PropertyPrefixIfChildren = ""
	c.PropertyPrefixNoChildren = ""
	c.LinkCharacter = " "
	c.AddBlankLineIfNoChildren = false
	// Add a colon after the description and before the properties to link the
	// properties to the description line.
	c.AfterDescriptionIfBody = ":"
	c.IsBlankLineBetweenPropertiesAndChildren = false
})

// FlatTextConfiguration draws a tree without indentation. It never changes.
//
// Flutter: foundation/diagnostics.dart flatTextConfiguration
var FlatTextConfiguration = newTextTreeConfiguration(func(c *TextTreeConfiguration) {
	c.PrefixLineOne = ""
	c.PrefixLastChildLineOne = ""
	c.PrefixOtherLines = ""
	c.PrefixOtherLinesRootNode = ""
	c.PropertyPrefixIfChildren = ""
	c.PropertyPrefixNoChildren = ""
	c.LinkCharacter = ""
	c.AddBlankLineIfNoChildren = false
	// Add a colon after the description and before the properties to link the
	// properties to the description line.
	c.AfterDescriptionIfBody = ":"
	c.IsBlankLineBetweenPropertiesAndChildren = false
})

// SingleLineTextConfiguration draws a node on one line, with its properties in
// parentheses. It never changes.
//
// Flutter: foundation/diagnostics.dart singleLineTextConfiguration
var SingleLineTextConfiguration = newTextTreeConfiguration(func(c *TextTreeConfiguration) {
	c.PropertySeparator = ", "
	c.BeforeProperties = "("
	c.AfterProperties = ")"
	c.PrefixLineOne = ""
	c.PrefixOtherLines = ""
	c.PrefixLastChildLineOne = ""
	c.LineBreak = ""
	c.LineBreakProperties = false
	c.AddBlankLineIfNoChildren = false
	c.ShowChildren = false
	c.PropertyPrefixIfChildren = "  "
	c.PropertyPrefixNoChildren = "  "
	c.LinkCharacter = ""
	c.PrefixOtherLinesRootNode = ""
})

// ErrorPropertyTextConfiguration draws a property of an error message. It
// never changes.
//
// Flutter: foundation/diagnostics.dart errorPropertyTextConfiguration
var ErrorPropertyTextConfiguration = newTextTreeConfiguration(func(c *TextTreeConfiguration) {
	c.PropertySeparator = ", "
	c.BeforeProperties = "("
	c.AfterProperties = ")"
	c.PrefixLineOne = ""
	c.PrefixOtherLines = ""
	c.PrefixLastChildLineOne = ""
	c.LineBreakProperties = false
	c.AddBlankLineIfNoChildren = false
	c.ShowChildren = false
	c.PropertyPrefixIfChildren = "  "
	c.PropertyPrefixNoChildren = "  "
	c.LinkCharacter = ""
	c.PrefixOtherLinesRootNode = ""
	c.IsNameOnOwnLine = true
})

// ShallowTextConfiguration draws a node without its children. It never
// changes.
//
// Flutter: foundation/diagnostics.dart shallowTextConfiguration
var ShallowTextConfiguration = newTextTreeConfiguration(func(c *TextTreeConfiguration) {
	c.PrefixLineOne = ""
	c.PrefixLastChildLineOne = ""
	c.PrefixOtherLines = " "
	c.PrefixOtherLinesRootNode = "  "
	c.PropertyPrefixIfChildren = ""
	c.PropertyPrefixNoChildren = ""
	c.LinkCharacter = " "
	c.AddBlankLineIfNoChildren = false
	// Add a colon after the description and before the properties to link the
	// properties to the description line.
	c.AfterDescriptionIfBody = ":"
	c.IsBlankLineBetweenPropertiesAndChildren = false
	c.ShowChildren = false
})
