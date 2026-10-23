package foundation

import (
	"strings"

	"github.com/rm4n0s/errors"
	intopts "github.com/rm4n0s/glutter/internal/opts"
)

// DiagnosticsNode is a description of a node of a tree of diagnostics: an
// object, a property of an object or a block of text. A tree of nodes is
// dumped as text with ToStringDeep.
//
// Embed DiagnosticsNodeBase to make a node of your own, and call
// InitDiagnosticsNodeBase in its constructor.
//
// Flutter: foundation/diagnostics.dart DiagnosticsNode
type DiagnosticsNode interface {
	// Name is the name of the node, such as the name of a property. It does
	// not end with a colon, which is added when the node is drawn.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.name
	Name() Opt[string]
	// ShowName is whether the name is drawn.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.showName
	ShowName() bool
	// ShowSeparator is whether a colon follows the name.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.showSeparator
	ShowSeparator() bool
	// LinePrefix is text added before every line of the node.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.linePrefix
	LinePrefix() Opt[string]
	// Style is how the node is drawn as text.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.style
	Style() DiagnosticsTreeStyle
	// Level is the importance of the node.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.level
	Level() DiagnosticLevel
	// Value is the object the node describes, or nil.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.value
	Value() any
	// EmptyBodyDescription is what to draw for a node that has no properties
	// and no children, if anything.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.emptyBodyDescription
	EmptyBodyDescription() Opt[string]
	// AllowWrap is whether the description may wrap over several lines.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.allowWrap
	AllowWrap() bool
	// AllowNameWrap is whether the name may wrap over several lines.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.allowNameWrap
	AllowNameWrap() bool
	// AllowTruncate is whether the properties and children may be cut short.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.allowTruncate
	AllowTruncate() bool
	// ToDescription is the description of the node, without its name.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.toDescription
	ToDescription(parentConfiguration *TextTreeConfiguration) string
	// GetProperties returns the properties of the node.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.getProperties
	GetProperties() []DiagnosticsNode
	// GetChildren returns the children of the node.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.getChildren
	GetChildren() []DiagnosticsNode
	// IsFiltered reports whether the node is less important than minLevel
	// (every node is in release builds).
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.isFiltered
	IsFiltered(minLevel DiagnosticLevel) bool
	// TextTreeConfiguration is the configuration for the node's style, or nil
	// for DiagnosticsTreeStyleNone.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.textTreeConfiguration
	TextTreeConfiguration() *TextTreeConfiguration
	// ToStringDeep dumps the node and its subtree as text.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.toStringDeep
	ToStringDeep(opts ...ToStringDeepOpts) string
	// ToStringWith describes the node on one line, or a few, at a minimum
	// level.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.toString
	ToStringWith(parentConfiguration *TextTreeConfiguration, minLevel DiagnosticLevel) string
	// String is ToStringWith with no parent and the info level.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticsNode.toString
	String() string
}

// ToStringDeepOpts are the optional parameters of ToStringDeep.
//
// Flutter: foundation/diagnostics.dart DiagnosticsNode.toStringDeep
type ToStringDeepOpts struct {
	// PrefixLineOne is added before the first line.
	PrefixLineOne string
	// PrefixOtherLines is added before the other lines; the default is
	// PrefixLineOne.
	PrefixOtherLines Opt[string]
	// ParentConfiguration is the configuration of the parent, if any.
	ParentConfiguration *TextTreeConfiguration
	// MinLevel hides the properties less important than it; the default is
	// DiagnosticLevelDebug.
	MinLevel DiagnosticLevel
	// WrapWidth is the width at which lines wrap; the default is 65.
	WrapWidth int
}

// DiagnosticsNodeBaseOpts are the optional parameters of
// InitDiagnosticsNodeBase.
//
// Flutter: foundation/diagnostics.dart DiagnosticsNode.new
type DiagnosticsNodeBaseOpts struct {
	// Name is the name of the node. It must not end with a colon, or
	// InitDiagnosticsNodeBase panics with the tag DiagnosticsNameEndsWithColon
	// in debug builds.
	Name Opt[string]
	// Style is how the node is drawn; the default is the one the node type
	// chooses.
	Style DiagnosticsTreeStyle
	// ShowName is whether the name is drawn; the default is true.
	ShowName Opt[bool]
	// ShowSeparator is whether a colon follows the name; the default is true.
	ShowSeparator Opt[bool]
	// LinePrefix is text added before every line of the node.
	LinePrefix Opt[string]
}

// DiagnosticsNodeBase holds the fields every DiagnosticsNode has and the
// methods that depend only on them. Embed it, and call InitDiagnosticsNodeBase
// with the outer node, so that the methods here reach the outer node's own
// versions of Level, Style, ToDescription and the others.
//
// Flutter: foundation/diagnostics.dart DiagnosticsNode
type DiagnosticsNodeBase struct {
	self          DiagnosticsNode
	name          Opt[string]
	style         DiagnosticsTreeStyle
	showName      bool
	showSeparator bool
	linePrefix    Opt[string]
}

// InitDiagnosticsNodeBase sets up the embedded base of the node self.
//
// Flutter: foundation/diagnostics.dart DiagnosticsNode.new
func InitDiagnosticsNodeBase(base *DiagnosticsNodeBase, self DiagnosticsNode, opts DiagnosticsNodeBaseOpts) {
	if name, ok := opts.Name.Get(); ok && DebugMode && strings.HasSuffix(name, ":") {
		// A name ending with ':' indicates that the user forgot that the ':'
		// will be automatically added for them when generating descriptions
		// of the property.
		panic(errors.New(tagDiagnosticsNameEndsWithColon, "Names of diagnostic nodes must not end with colons.", "name", name))
	}
	*base = DiagnosticsNodeBase{
		self:          self,
		name:          opts.Name,
		style:         opts.Style,
		showName:      opts.ShowName.OrElse(true),
		showSeparator: opts.ShowSeparator.OrElse(true),
		linePrefix:    opts.LinePrefix,
	}
}

// Name is the name of the node.
func (b *DiagnosticsNodeBase) Name() Opt[string] { return b.name }

// ShowName is whether the name is drawn.
func (b *DiagnosticsNodeBase) ShowName() bool { return b.showName }

// ShowSeparator is whether a colon follows the name.
func (b *DiagnosticsNodeBase) ShowSeparator() bool { return b.showSeparator }

// LinePrefix is text added before every line of the node.
func (b *DiagnosticsNodeBase) LinePrefix() Opt[string] { return b.linePrefix }

// Style is how the node is drawn.
func (b *DiagnosticsNodeBase) Style() DiagnosticsTreeStyle { return b.style }

// Level is DiagnosticLevelInfo, and hidden in release builds.
func (b *DiagnosticsNodeBase) Level() DiagnosticLevel {
	if ReleaseMode {
		return DiagnosticLevelHidden
	}
	return DiagnosticLevelInfo
}

// EmptyBodyDescription is None.
func (b *DiagnosticsNodeBase) EmptyBodyDescription() Opt[string] { return None[string]() }

// AllowWrap is false.
func (b *DiagnosticsNodeBase) AllowWrap() bool { return false }

// AllowNameWrap is false.
func (b *DiagnosticsNodeBase) AllowNameWrap() bool { return false }

// AllowTruncate is false.
func (b *DiagnosticsNodeBase) AllowTruncate() bool { return false }

// IsFiltered reports whether the node is less important than minLevel (every
// node is in release builds).
func (b *DiagnosticsNodeBase) IsFiltered(minLevel DiagnosticLevel) bool {
	return ReleaseMode || b.self.Level() < minLevel
}

// TextTreeConfiguration is the configuration for the node's style, or nil for
// DiagnosticsTreeStyleNone.
func (b *DiagnosticsNodeBase) TextTreeConfiguration() *TextTreeConfiguration {
	switch b.self.Style() {
	case DiagnosticsTreeStyleNone:
		return nil
	case DiagnosticsTreeStyleDense:
		return DenseTextConfiguration
	case DiagnosticsTreeStyleSparse:
		return SparseTextConfiguration
	case DiagnosticsTreeStyleOffstage:
		return DashedTextConfiguration
	case DiagnosticsTreeStyleWhitespace:
		return WhitespaceTextConfiguration
	case DiagnosticsTreeStyleTransition:
		return TransitionTextConfiguration
	case DiagnosticsTreeStyleSingleLine:
		return SingleLineTextConfiguration
	case DiagnosticsTreeStyleErrorProperty:
		return ErrorPropertyTextConfiguration
	case DiagnosticsTreeStyleShallow:
		return ShallowTextConfiguration
	case DiagnosticsTreeStyleError:
		return ErrorTextConfiguration
	case DiagnosticsTreeStyleFlat:
		return FlatTextConfiguration
	case DiagnosticsTreeStyleTruncateChildren:
		// Truncate children doesn't really need its own text style as the
		// rendering is quite custom.
		return WhitespaceTextConfiguration
	}
	// An unspecified style cannot be drawn.
	panic(errors.New(tagDiagnosticsStyleUnspecified, "a diagnostics node must have a style to be drawn",
		"type", RuntimeTypeName(b.self)))
}

// ToStringDeep dumps the node and its subtree as text.
func (b *DiagnosticsNodeBase) ToStringDeep(opts ...ToStringDeepOpts) string {
	o := firstOpts(opts)
	if !DebugMode {
		return ""
	}
	wrapWidth := o.WrapWidth
	if wrapWidth == 0 {
		wrapWidth = 65
	}
	return NewTextTreeRenderer(TextTreeRendererOpts{
		MinLevel:  o.MinLevel.orDefault(DiagnosticLevelDebug),
		WrapWidth: wrapWidth,
	}).Render(b.self, RenderOpts{
		PrefixLineOne:       o.PrefixLineOne,
		PrefixOtherLines:    o.PrefixOtherLines,
		ParentConfiguration: o.ParentConfiguration,
	})
}

// ToStringWith describes the node on one line, or a few, at a minimum level.
func (b *DiagnosticsNodeBase) ToStringWith(parentConfiguration *TextTreeConfiguration, minLevel DiagnosticLevel) string {
	if !DebugMode {
		return "Instance of '" + RuntimeTypeName(b.self) + "'"
	}
	if b.self.Style() == DiagnosticsTreeStyleSingleLine {
		return b.self.ToStringDeep(ToStringDeepOpts{ParentConfiguration: parentConfiguration, MinLevel: minLevel})
	}
	description := b.self.ToDescription(parentConfiguration)
	name, hasName := b.name.Get()
	if !hasName || name == "" || !b.self.ShowName() {
		return description
	}
	sep := ""
	if b.self.ShowSeparator() {
		sep = ":"
	}
	if strings.Contains(description, "\n") {
		return name + sep + "\n" + description
	}
	return name + sep + " " + description
}

// String is ToStringWith with no parent and the info level.
func (b *DiagnosticsNodeBase) String() string {
	return b.self.ToStringWith(nil, DiagnosticLevelInfo)
}

// firstOpts returns the one options value of a trailing `opts ...XOpts`
// parameter (SPEC §12.1). It wraps opts.First, because a parameter named opts
// hides the package.
func firstOpts[T any](o []T) T { return intopts.First(o) }

// NewDiagnosticsNodeMessage creates a node that shows a message. It is
// Flutter's DiagnosticsNode.message constructor.
//
// Flutter: foundation/diagnostics.dart DiagnosticsNode.message
func NewDiagnosticsNodeMessage(message string, opts ...DiagnosticsNodeMessageOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	return newDiagnosticsProperty("DiagnosticsProperty<void>", "void", "", nil, nil, propertyConfig{
		description: Some(message),
		showName:    Some(false),
		allowWrap:   Some(!o.NoWrap),
		style:       o.Style.orDefault(DiagnosticsTreeStyleSingleLine),
		level:       o.Level.orDefault(DiagnosticLevelInfo),
	})
}

// DiagnosticsNodeMessageOpts are the optional parameters of
// NewDiagnosticsNodeMessage.
//
// Flutter: foundation/diagnostics.dart DiagnosticsNode.message
type DiagnosticsNodeMessageOpts struct {
	// Style is how the message is drawn; the default is single line.
	Style DiagnosticsTreeStyle
	// Level is the importance of the message; the default is info.
	Level DiagnosticLevel
	// NoWrap stops the message from wrapping; Flutter's allowWrap is true by
	// default.
	NoWrap bool
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDiagnosticsNameEndsWithColon = "DiagnosticsNameEndsWithColon"
	tagDiagnosticsStyleUnspecified  = "DiagnosticsStyleUnspecified"
)
