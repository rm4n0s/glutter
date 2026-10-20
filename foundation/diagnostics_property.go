package foundation

import (
	"reflect"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dartfmt"
)

// noDefault is the type of the value that stands for "no default value".
type noDefault struct{}

// DefaultNull is the DefaultValue that means Dart's `defaultValue: null`: a
// property whose value is nil has its default value. Leave DefaultValue unset
// (nil) for a property that has no default value.
//
// Flutter: foundation/diagnostics.dart kNoDefaultValue (the opposite: Go's nil is the absence)
var DefaultNull = noDefault{}

// propertyConfig gathers the parameters every property constructor passes to
// newDiagnosticsProperty.
type propertyConfig struct {
	description                    Opt[string]
	ifNull, ifEmpty, tooltip       Opt[string]
	showName, showSeparator        Opt[bool]
	allowWrap, allowNameWrap       Opt[bool]
	hasDefault                     bool
	defaultValue                   any
	missingIfNull, expandableValue bool
	linePrefix                     Opt[string]
	style                          DiagnosticsTreeStyle
	level                          DiagnosticLevel
	valueToString                  func(p *DiagnosticsProperty, parent *TextTreeConfiguration) string
	levelFn                        func(p *DiagnosticsProperty) DiagnosticLevel
	showNameFn                     func(p *DiagnosticsProperty) bool
}

// DiagnosticsProperty describes a property of an object: a name and a value,
// drawn as "name: value". The constructors NewStringProperty,
// NewDoubleProperty, NewFlagProperty and the others make properties that
// describe their value in their own way.
//
// Deviation: all the property kinds are this one type, which the constructors
// configure; Flutter has a class for each (SPEC §12.10).
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty
type DiagnosticsProperty struct {
	DiagnosticsNodeBase
	// typeName is the name of Flutter's class, such as "StringProperty".
	typeName string
	// propertyType is the name of the type of the value.
	propertyType string

	description    Opt[string]
	ifNull         Opt[string]
	ifEmpty        Opt[string]
	tooltip        Opt[string]
	missingIfNull  bool
	expandable     bool
	allowWrap      bool
	allowNameWrap  bool
	hasDefault     bool
	defaultValue   any
	defaultLevel   DiagnosticLevel
	value          any
	compute        func() any
	valueComputed  bool
	exception      any
	hasException   bool
	valueToStringF func(p *DiagnosticsProperty, parent *TextTreeConfiguration) string
	levelF         func(p *DiagnosticsProperty) DiagnosticLevel
	showNameF      func(p *DiagnosticsProperty) bool
	baseShowName   bool
}

// newDiagnosticsProperty makes a property. A nil compute means the value is
// given.
func newDiagnosticsProperty(typeName, propertyType, name string, value any, compute func() any, cfg propertyConfig) *DiagnosticsProperty {
	p := &DiagnosticsProperty{
		typeName:       typeName,
		propertyType:   propertyType,
		description:    cfg.description,
		ifNull:         cfg.ifNull,
		ifEmpty:        cfg.ifEmpty,
		tooltip:        cfg.tooltip,
		missingIfNull:  cfg.missingIfNull,
		expandable:     cfg.expandableValue,
		allowWrap:      cfg.allowWrap.OrElse(true),
		allowNameWrap:  cfg.allowNameWrap.OrElse(true),
		hasDefault:     cfg.hasDefault,
		defaultValue:   cfg.defaultValue,
		defaultLevel:   cfg.level.orDefault(DiagnosticLevelInfo),
		value:          value,
		compute:        compute,
		valueComputed:  compute == nil,
		valueToStringF: cfg.valueToString,
		levelF:         cfg.levelFn,
		showNameF:      cfg.showNameFn,
	}
	if cfg.missingIfNull && !p.ifNull.IsSome() {
		p.ifNull = Some("MISSING")
	}
	InitDiagnosticsNodeBase(&p.DiagnosticsNodeBase, p, DiagnosticsNodeBaseOpts{
		Name:          Some(name),
		Style:         cfg.style.orDefault(DiagnosticsTreeStyleSingleLine),
		ShowName:      cfg.showName,
		ShowSeparator: cfg.showSeparator,
		LinePrefix:    cfg.linePrefix,
	})
	p.baseShowName = p.showName
	return p
}

// DiagnosticsPropertyOpts are the optional parameters of the property
// constructors that describe any value.
//
// A string that is empty means null in Dart: it is not used.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.new
type DiagnosticsPropertyOpts struct {
	// Description replaces the description made from the value.
	Description string
	// IfNull is the description of a nil value.
	IfNull string
	// IfEmpty is the description of a value that describes as an empty string.
	IfEmpty string
	// ShowName is whether the name is drawn; the default is true.
	ShowName Opt[bool]
	// ShowSeparator is whether a colon follows the name; the default is true.
	ShowSeparator Opt[bool]
	// DefaultValue is the value that is not interesting: the property is
	// hidden from most dumps when it has this value. Leave it nil for none, and
	// use DefaultNull when the default is Dart's null.
	DefaultValue any
	// Tooltip is added after the description in parentheses.
	Tooltip string
	// MissingIfNull makes a nil value a warning that reads "MISSING".
	MissingIfNull bool
	// LinePrefix is text added before every line of the node.
	LinePrefix string
	// ExpandableValue shows the properties and children of the value under
	// this property.
	ExpandableValue bool
	// NoWrap stops the description from wrapping; Flutter's allowWrap is true
	// by default.
	NoWrap bool
	// NoNameWrap stops the name from wrapping; Flutter's allowNameWrap is
	// true by default.
	NoNameWrap bool
	// Style is how the property is drawn; the default is single line.
	Style DiagnosticsTreeStyle
	// Level is the importance of the property; the default is info.
	Level DiagnosticLevel
	// ValueToString replaces the description of the value, as an override of
	// DiagnosticsProperty.valueToString does in Dart. It gets the configuration
	// of the parent node.
	ValueToString func(parent *TextTreeConfiguration) string
}

func optString(s string) Opt[string] {
	if s == "" {
		return None[string]()
	}
	return Some(s)
}

func (o DiagnosticsPropertyOpts) config() propertyConfig {
	c := propertyConfig{
		description:     optString(o.Description),
		ifNull:          optString(o.IfNull),
		ifEmpty:         optString(o.IfEmpty),
		tooltip:         optString(o.Tooltip),
		showName:        o.ShowName,
		showSeparator:   o.ShowSeparator,
		missingIfNull:   o.MissingIfNull,
		linePrefix:      optString(o.LinePrefix),
		expandableValue: o.ExpandableValue,
		style:           o.Style,
		level:           o.Level,
	}
	if o.NoWrap {
		c.allowWrap = Some(false)
	}
	if o.NoNameWrap {
		c.allowNameWrap = Some(false)
	}
	if o.ValueToString != nil {
		fn := o.ValueToString
		c.valueToString = func(_ *DiagnosticsProperty, parent *TextTreeConfiguration) string { return fn(parent) }
	}
	setDefault(&c, o.DefaultValue)
	return c
}

// setDefault records the default value of a property.
func setDefault(c *propertyConfig, v any) {
	switch v.(type) {
	case nil:
	case noDefault:
		c.hasDefault, c.defaultValue = true, nil
	default:
		c.hasDefault, c.defaultValue = true, v
	}
}

// NewDiagnosticsProperty creates a property with a name and a value of any
// type. A value that is nil (or a nil pointer, map, slice or function) is
// Dart's null.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.new
func NewDiagnosticsProperty[T any](name string, value T, opts ...DiagnosticsPropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	return newDiagnosticsProperty("DiagnosticsProperty<"+typeArg[T]()+">", typeArg[T](), name, any(value), nil, o.config())
}

// NewDiagnosticsPropertyOpt creates a property whose value is None for Dart's
// null, where the type of the value has no nil of its own (a bool, a number or
// a struct).
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.new
func NewDiagnosticsPropertyOpt[T any](name string, value Opt[T], opts ...DiagnosticsPropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	return newDiagnosticsProperty("DiagnosticsProperty<"+typeArg[T]()+">", typeArg[T](), name, optAny(value), nil, o.config())
}

// NewDiagnosticsPropertyLazy creates a property whose value is computed when
// it is first needed. If computing panics, the property shows the panic as
// its exception instead of a value, and has the error level.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.lazy
func NewDiagnosticsPropertyLazy[T any](name string, computeValue func() T, opts ...DiagnosticsPropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	return newDiagnosticsProperty("DiagnosticsProperty<"+typeArg[T]()+">", typeArg[T](), name, nil,
		func() any { return any(computeValue()) }, o.config())
}

// typeArg is the Dart name of the type T.
func typeArg[T any]() string {
	return RuntimeTypeName(reflect.New(reflect.TypeFor[T]()).Elem().Interface())
}

// MessagePropertyOpts are the optional parameters of NewMessageProperty.
//
// Flutter: foundation/diagnostics.dart MessageProperty.new
type MessagePropertyOpts struct {
	// Style is how the property is drawn; the default is single line.
	Style DiagnosticsTreeStyle
	// Level is the importance of the property; the default is info.
	Level DiagnosticLevel
}

// NewMessageProperty creates a property whose description is a message.
//
// Flutter: foundation/diagnostics.dart MessageProperty.new
func NewMessageProperty(name, message string, opts ...MessagePropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	return newDiagnosticsProperty("MessageProperty", "void", name, nil, nil, propertyConfig{
		description: Some(message),
		style:       o.Style.orDefault(DiagnosticsTreeStyleSingleLine),
		level:       o.Level.orDefault(DiagnosticLevelInfo),
	})
}

// StringPropertyOpts are the optional parameters of NewStringProperty.
//
// Flutter: foundation/diagnostics.dart StringProperty.new
type StringPropertyOpts struct {
	// Description replaces the description made from the value.
	Description string
	// Tooltip is added after the description in parentheses.
	Tooltip string
	// ShowName is whether the name is drawn; the default is true.
	ShowName Opt[bool]
	// DefaultValue is the value that is not interesting; see
	// DiagnosticsPropertyOpts.
	DefaultValue any
	// Unquoted draws the string without quotes; Flutter's quoted is true by
	// default.
	Unquoted bool
	// IfEmpty is the description of an empty string.
	IfEmpty string
	// Style is how the property is drawn; the default is single line.
	Style DiagnosticsTreeStyle
	// Level is the importance of the property; the default is info.
	Level DiagnosticLevel
}

// NewStringProperty creates a property that describes a string, in quotes.
// The value is None for Dart's null.
//
// Flutter: foundation/diagnostics.dart StringProperty.new
func NewStringProperty(name string, value Opt[string], opts ...StringPropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	cfg := propertyConfig{
		description: optString(o.Description),
		tooltip:     optString(o.Tooltip),
		showName:    o.ShowName,
		ifEmpty:     optString(o.IfEmpty),
		style:       o.Style,
		level:       o.Level,
	}
	setDefault(&cfg, o.DefaultValue)
	quoted := !o.Unquoted
	cfg.valueToString = func(p *DiagnosticsProperty, parent *TextTreeConfiguration) string {
		text, hasText := p.description.Get()
		if !hasText && p.value != nil {
			text, hasText = p.value.(string), true
		}
		if parent != nil && !parent.LineBreakProperties && hasText {
			// Escape linebreaks in multiline strings to avoid confusing output
			// when the parent of this node is trying to display all properties
			// on the same line.
			text = strings.ReplaceAll(text, "\n", `\n`)
		}
		if quoted && hasText {
			// An empty value would not appear empty after being surrounded
			// with quotes so we have to handle this case separately.
			if ifEmpty, ok := p.ifEmpty.Get(); ok && text == "" {
				return ifEmpty
			}
			return `"` + text + `"`
		}
		if !hasText {
			return "null"
		}
		return text
	}
	var v any
	if s, ok := value.Get(); ok {
		v = s
	}
	return newDiagnosticsProperty("StringProperty", "String", name, v, nil, cfg)
}

// NumberPropertyOpts are the optional parameters of the number properties.
//
// Flutter: foundation/diagnostics.dart DoubleProperty.new
type NumberPropertyOpts struct {
	// IfNull is the description of a null value.
	IfNull string
	// Unit is added after the number, such as "px".
	Unit string
	// Tooltip is added after the description in parentheses.
	Tooltip string
	// DefaultValue is the value that is not interesting; see
	// DiagnosticsPropertyOpts.
	DefaultValue any
	// ShowName is whether the name is drawn; the default is true.
	ShowName Opt[bool]
	// Style is how the property is drawn; the default is single line.
	Style DiagnosticsTreeStyle
	// Level is the importance of the property; the default is info.
	Level DiagnosticLevel
}

func (o NumberPropertyOpts) config(numberToString func(v any) string, unitSep string) propertyConfig {
	cfg := propertyConfig{
		ifNull:   optString(o.IfNull),
		tooltip:  optString(o.Tooltip),
		showName: o.ShowName,
		style:    o.Style,
		level:    o.Level,
	}
	setDefault(&cfg, o.DefaultValue)
	unit := o.Unit
	cfg.valueToString = func(p *DiagnosticsProperty, _ *TextTreeConfiguration) string {
		if p.Value() == nil {
			return "null"
		}
		if unit != "" {
			return numberToString(p.value) + unitSep + unit
		}
		return numberToString(p.value)
	}
	return cfg
}

// DebugFormatDouble formats a number for diagnostics: one decimal, as Dart's
// toStringAsFixed(1).
//
// Flutter: foundation/debug.dart debugFormatDouble
func DebugFormatDouble(value Opt[float64]) string {
	v, ok := value.Get()
	if !ok {
		return "null"
	}
	return dartfmt.ToStringAsFixed(v, 1)
}

// NewDoubleProperty creates a property that describes a number with one
// decimal. The value is None for Dart's null.
//
// Flutter: foundation/diagnostics.dart DoubleProperty.new
func NewDoubleProperty(name string, value Opt[float64], opts ...NumberPropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	cfg := o.config(func(v any) string { return DebugFormatDouble(Some(v.(float64))) }, "")
	return newDiagnosticsProperty("DoubleProperty", "double", name, optAny(value), nil, cfg)
}

// NewDoublePropertyLazy creates a DoubleProperty whose value is computed when
// it is first needed.
//
// Flutter: foundation/diagnostics.dart DoubleProperty.lazy
func NewDoublePropertyLazy(name string, computeValue func() Opt[float64], opts ...NumberPropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	cfg := o.config(func(v any) string { return DebugFormatDouble(Some(v.(float64))) }, "")
	return newDiagnosticsProperty("DoubleProperty", "double", name, nil, func() any { return optAny(computeValue()) }, cfg)
}

// NewIntProperty creates a property that describes an integer. The value is
// None for Dart's null.
//
// Flutter: foundation/diagnostics.dart IntProperty.new
func NewIntProperty(name string, value Opt[int], opts ...NumberPropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	cfg := o.config(func(v any) string { return itoa(v.(int)) }, "")
	return newDiagnosticsProperty("IntProperty", "int", name, optAny(value), nil, cfg)
}

// NewPercentProperty creates a property that describes a fraction as a
// percentage with one decimal ("37.5%"), clamped to 0 through 100. The value
// is None for Dart's null.
//
// Flutter: foundation/diagnostics.dart PercentProperty.new
func NewPercentProperty(name string, fraction Opt[float64], opts ...NumberPropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	cfg := o.config(func(v any) string {
		return dartfmt.ToStringAsFixed(min(max(v.(float64), 0), 1)*100, 1) + "%"
	}, " ")
	return newDiagnosticsProperty("PercentProperty", "double", name, optAny(fraction), nil, cfg)
}

// optAny returns the value of o, or nil.
func optAny[T any](o Opt[T]) any {
	if v, ok := o.Get(); ok {
		return v
	}
	return nil
}

// FlagPropertyOpts are the optional parameters of NewFlagProperty.
//
// Flutter: foundation/diagnostics.dart FlagProperty.new
type FlagPropertyOpts struct {
	// IfTrue is the description when the flag is true.
	IfTrue string
	// IfFalse is the description when the flag is false.
	IfFalse string
	// ShowName is whether the name is drawn next to the description.
	ShowName bool
	// DefaultValue is the value that is not interesting; see
	// DiagnosticsPropertyOpts.
	DefaultValue any
	// Level is the importance of the property; the default is info.
	Level DiagnosticLevel
}

// NewFlagProperty creates a property that describes a boolean flag by a
// phrase for each value, such as "has focus". The value is None for Dart's
// null. At least one of IfTrue and IfFalse must be set, or it panics with the
// tag FlagPropertyNoDescription in debug builds.
//
// Flutter: foundation/diagnostics.dart FlagProperty.new
func NewFlagProperty(name string, value Opt[bool], opts ...FlagPropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	if DebugMode && o.IfTrue == "" && o.IfFalse == "" {
		panic(errors.New("FlagPropertyNoDescription", "a FlagProperty needs IfTrue or IfFalse", "name", name))
	}
	cfg := propertyConfig{showName: Some(o.ShowName), level: o.Level}
	setDefault(&cfg, o.DefaultValue)
	cfg.valueToString = func(p *DiagnosticsProperty, parent *TextTreeConfiguration) string {
		switch v := p.value.(type) {
		case bool:
			if v && o.IfTrue != "" {
				return o.IfTrue
			}
			if !v && o.IfFalse != "" {
				return o.IfFalse
			}
		}
		return p.defaultValueToString(parent)
	}
	cfg.showNameFn = func(p *DiagnosticsProperty) bool {
		v, ok := p.value.(bool)
		if !ok || (v && o.IfTrue == "") || (!v && o.IfFalse == "") {
			// We are missing a description for the flag value so we need to
			// show the flag name. The property will have
			// DiagnosticLevelHidden for this case so users will not see this
			// property in this case unless they are displaying hidden
			// properties.
			return true
		}
		return p.baseShowName
	}
	cfg.levelFn = func(p *DiagnosticsProperty) DiagnosticLevel {
		if v, ok := p.value.(bool); ok {
			if (v && o.IfTrue == "") || (!v && o.IfFalse == "") {
				return DiagnosticLevelHidden
			}
		}
		return p.baseLevel()
	}
	return newDiagnosticsProperty("FlagProperty", "bool", name, optAny(value), nil, cfg)
}

// IterablePropertyOpts are the optional parameters of NewIterableProperty.
//
// Flutter: foundation/diagnostics.dart IterableProperty.new
type IterablePropertyOpts struct {
	// DefaultValue is the value that is not interesting; see
	// DiagnosticsPropertyOpts.
	DefaultValue any
	// IfNull is the description of a nil value.
	IfNull string
	// IfEmpty is the description of an empty list; the default is "[]". Set
	// NoIfEmpty to have none.
	IfEmpty string
	// NoIfEmpty removes the description of an empty list: the property then
	// has the fine level when it is empty.
	NoIfEmpty bool
	// Style is how the property is drawn; the default is single line.
	Style DiagnosticsTreeStyle
	// ShowName is whether the name is drawn; the default is true.
	ShowName Opt[bool]
	// ShowSeparator is whether a colon follows the name; the default is true.
	ShowSeparator Opt[bool]
	// Level is the importance of the property; the default is info.
	Level DiagnosticLevel
}

// NewIterableProperty creates a property that describes a list of values.
// Float values are formatted with DebugFormatDouble. The value is a slice;
// nil is Dart's null.
//
// Flutter: foundation/diagnostics.dart IterableProperty.new
func NewIterableProperty[T any](name string, value []T, opts ...IterablePropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	cfg := propertyConfig{
		ifNull:        optString(o.IfNull),
		style:         o.Style,
		showName:      o.ShowName,
		showSeparator: o.ShowSeparator,
		level:         o.Level,
	}
	if !o.NoIfEmpty {
		if o.IfEmpty == "" {
			cfg.ifEmpty = Some("[]")
		} else {
			cfg.ifEmpty = Some(o.IfEmpty)
		}
	}
	setDefault(&cfg, o.DefaultValue)
	cfg.valueToString = func(p *DiagnosticsProperty, parent *TextTreeConfiguration) string {
		if value == nil {
			return "null"
		}
		if len(value) == 0 {
			return p.ifEmpty.OrElse("[]")
		}
		formatted := make([]string, len(value))
		for i, v := range value {
			if f, ok := any(v).(float64); ok {
				formatted[i] = DebugFormatDouble(Some(f))
			} else {
				formatted[i] = describeValue(v)
			}
		}
		if parent != nil && !parent.LineBreakProperties {
			// Always display the value as a single line and enclose the
			// iterable value in brackets to avoid ambiguity.
			return "[" + strings.Join(formatted, ", ") + "]"
		}
		if isSingleLineStyle(p.Style()) {
			return strings.Join(formatted, ", ")
		}
		return strings.Join(formatted, "\n")
	}
	cfg.levelFn = func(p *DiagnosticsProperty) DiagnosticLevel {
		if !p.ifEmpty.IsSome() && value != nil && len(value) == 0 && p.baseLevel() != DiagnosticLevelHidden {
			return DiagnosticLevelFine
		}
		return p.baseLevel()
	}
	var v any
	if value != nil {
		v = value
	}
	return newDiagnosticsProperty("IterableProperty<"+typeArg[T]()+">", "Iterable<"+typeArg[T]()+">", name, v, nil, cfg)
}

// EnumPropertyOpts are the optional parameters of NewEnumProperty.
//
// Flutter: foundation/diagnostics.dart EnumProperty.new
type EnumPropertyOpts struct {
	// DefaultValue is the value that is not interesting; see
	// DiagnosticsPropertyOpts.
	DefaultValue any
	// Level is the importance of the property; the default is info.
	Level DiagnosticLevel
}

// EnumValue is a value of an enum type that can name itself, as Dart's
// Enum.name does. The enum types of Glutter have a Name method.
//
// Flutter: no counterpart
type EnumValue interface {
	comparable
	Name() string
}

// NewEnumProperty creates a property that describes an enum value by its
// name. The value is None for Dart's null.
//
// Flutter: foundation/diagnostics.dart EnumProperty.new
func NewEnumProperty[T EnumValue](name string, value Opt[T], opts ...EnumPropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	cfg := propertyConfig{level: o.Level}
	setDefault(&cfg, o.DefaultValue)
	cfg.valueToString = func(p *DiagnosticsProperty, _ *TextTreeConfiguration) string {
		if v, ok := p.value.(T); ok {
			return v.Name()
		}
		return "null"
	}
	return newDiagnosticsProperty("EnumProperty<"+typeArg[T]()+">", typeArg[T](), name, optAny(value), nil, cfg)
}

// ObjectFlagPropertyOpts are the optional parameters of the object flag
// properties.
//
// Flutter: foundation/diagnostics.dart ObjectFlagProperty.new
type ObjectFlagPropertyOpts struct {
	// IfPresent is the description of a value that is not nil.
	IfPresent string
	// IfNull is the description of a nil value.
	IfNull string
	// ShowName is whether the name is drawn next to the description.
	ShowName bool
	// Level is the importance of the property; the default is info.
	Level DiagnosticLevel
}

// NewObjectFlagProperty creates a property that describes whether an object
// is there, by a phrase for each case, such as "has a callback". At least one
// of IfPresent and IfNull must be set, or it panics with the tag
// ObjectFlagPropertyNoDescription in debug builds.
//
// Flutter: foundation/diagnostics.dart ObjectFlagProperty.new
func NewObjectFlagProperty[T any](name string, value T, opts ...ObjectFlagPropertyOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	if DebugMode && o.IfPresent == "" && o.IfNull == "" {
		panic(errors.New("ObjectFlagPropertyNoDescription", "an ObjectFlagProperty needs IfPresent or IfNull", "name", name))
	}
	return newObjectFlagProperty(name, any(value), typeArg[T](), o)
}

// NewObjectFlagPropertyHas creates an ObjectFlagProperty that reads "has
// <name>" when the value is not nil and shows nothing otherwise.
//
// Flutter: foundation/diagnostics.dart ObjectFlagProperty.has
func NewObjectFlagPropertyHas[T any](name string, value T, level ...DiagnosticLevel) *DiagnosticsProperty {
	var l DiagnosticLevel
	if len(level) > 0 {
		l = level[0]
	}
	return newObjectFlagProperty(name, any(value), typeArg[T](), ObjectFlagPropertyOpts{IfPresent: "has " + name, Level: l})
}

func newObjectFlagProperty(name string, value any, propertyType string, o ObjectFlagPropertyOpts) *DiagnosticsProperty {
	cfg := propertyConfig{
		ifNull:   optString(o.IfNull),
		showName: Some(o.ShowName),
		level:    o.Level,
	}
	cfg.valueToString = func(p *DiagnosticsProperty, parent *TextTreeConfiguration) string {
		if !isNil(p.value) {
			if o.IfPresent != "" {
				return o.IfPresent
			}
		} else if o.IfNull != "" {
			return o.IfNull
		}
		return p.defaultValueToString(parent)
	}
	cfg.showNameFn = func(p *DiagnosticsProperty) bool {
		if (!isNil(p.value) && o.IfPresent == "") || (isNil(p.value) && o.IfNull == "") {
			return true
		}
		return p.baseShowName
	}
	cfg.levelFn = func(p *DiagnosticsProperty) DiagnosticLevel {
		if !isNil(p.value) {
			if o.IfPresent == "" {
				return DiagnosticLevelHidden
			}
		} else if o.IfNull == "" {
			return DiagnosticLevelHidden
		}
		return p.baseLevel()
	}
	if isNil(value) {
		value = nil
	}
	return newDiagnosticsProperty("ObjectFlagProperty<"+propertyType+">", propertyType, name, value, nil, cfg)
}

// FlagsSummaryOpts are the optional parameters of NewFlagsSummary.
//
// Flutter: foundation/diagnostics.dart FlagsSummary.new
type FlagsSummaryOpts struct {
	// IfEmpty is the description when no entry has a value.
	IfEmpty string
	// ShowName is whether the name is drawn; the default is true.
	ShowName Opt[bool]
	// ShowSeparator is whether a colon follows the name; the default is true.
	ShowSeparator Opt[bool]
	// Level is the importance of the property; the default is info.
	Level DiagnosticLevel
}

// FlagEntry is one entry of a FlagsSummary: a description and whether it
// applies (Flutter's non-null value).
//
// Flutter: no counterpart
type FlagEntry struct {
	// Description is shown when Present is true.
	Description string
	// Present is whether the entry has a value.
	Present bool
}

// NewFlagsSummary creates a property that lists the descriptions of the
// entries that are present, such as the states of a button. The entries keep
// their order (Flutter's map keeps its insertion order).
//
// Flutter: foundation/diagnostics.dart FlagsSummary.new
func NewFlagsSummary(name string, entries []FlagEntry, opts ...FlagsSummaryOpts) *DiagnosticsProperty {
	o := firstOpts(opts)
	cfg := propertyConfig{
		ifEmpty:       optString(o.IfEmpty),
		showName:      o.ShowName,
		showSeparator: o.ShowSeparator,
		level:         o.Level,
	}
	hasNonNull := func() bool {
		for _, e := range entries {
			if e.Present {
				return true
			}
		}
		return false
	}
	formatted := func() []string {
		var out []string
		for _, e := range entries {
			if e.Present {
				out = append(out, e.Description)
			}
		}
		return out
	}
	cfg.valueToString = func(p *DiagnosticsProperty, parent *TextTreeConfiguration) string {
		if ifEmpty, ok := p.ifEmpty.Get(); !hasNonNull() && ok {
			return ifEmpty
		}
		values := formatted()
		if parent != nil && !parent.LineBreakProperties {
			// Always display the value as a single line and enclose the
			// iterable value in brackets to avoid ambiguity.
			return "[" + strings.Join(values, ", ") + "]"
		}
		if isSingleLineStyle(p.Style()) {
			return strings.Join(values, ", ")
		}
		return strings.Join(values, "\n")
	}
	cfg.levelFn = func(p *DiagnosticsProperty) DiagnosticLevel {
		if !hasNonNull() && !p.ifEmpty.IsSome() {
			return DiagnosticLevelHidden
		}
		return p.baseLevel()
	}
	return newDiagnosticsProperty("FlagsSummary", "Map<String, Object?>", name, entries, nil, cfg)
}

// Exception is what computing the value panicked with, or nil.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.exception
func (p *DiagnosticsProperty) Exception() any {
	p.maybeCacheValue()
	return p.exception
}

// Value is the value of the property, computed on first use when the
// property is lazy. A nil result is Dart's null.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.value
func (p *DiagnosticsProperty) Value() any {
	p.maybeCacheValue()
	return p.value
}

func (p *DiagnosticsProperty) maybeCacheValue() {
	if p.valueComputed {
		return
	}
	p.valueComputed = true
	defer func() {
		if r := recover(); r != nil {
			// The error is shown in the dump; panicking again would destroy
			// the debugging experience.
			p.exception, p.hasException = r, true
			p.value = nil
		}
	}()
	p.value = p.compute()
	if isNil(p.value) {
		p.value = nil
	}
}

// PropertyType is the name of the type of the value.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.propertyType
func (p *DiagnosticsProperty) PropertyType() string { return p.propertyType }

// TypeName is the name of Flutter's class that this property stands for, such
// as "StringProperty".
func (p *DiagnosticsProperty) TypeName() string { return p.typeName }

// IfNull is the description of a null value, if any.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.ifNull
func (p *DiagnosticsProperty) IfNull() Opt[string] { return p.ifNull }

// IfEmpty is the description of a value that describes as an empty string, if
// any.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.ifEmpty
func (p *DiagnosticsProperty) IfEmpty() Opt[string] { return p.ifEmpty }

// Tooltip is shown after the description in parentheses, if any.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.tooltip
func (p *DiagnosticsProperty) Tooltip() Opt[string] { return p.tooltip }

// MissingIfNull is whether a nil value is a warning.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.missingIfNull
func (p *DiagnosticsProperty) MissingIfNull() bool { return p.missingIfNull }

// DefaultValue is the value that is not interesting, and whether there is
// one.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.defaultValue
func (p *DiagnosticsProperty) DefaultValue() (any, bool) { return p.defaultValue, p.hasDefault }

// IsInteresting is whether the value differs from the default value, or there
// is none.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.isInteresting
func (p *DiagnosticsProperty) IsInteresting() bool {
	return !p.hasDefault || !equalValues(p.Value(), p.defaultValue)
}

// ShowName is whether the name is drawn.
func (p *DiagnosticsProperty) ShowName() bool {
	if p.showNameF != nil {
		return p.showNameF(p)
	}
	return p.baseShowName
}

// AllowWrap is whether the description may wrap.
func (p *DiagnosticsProperty) AllowWrap() bool { return p.allowWrap }

// AllowNameWrap is whether the name may wrap.
func (p *DiagnosticsProperty) AllowNameWrap() bool { return p.allowNameWrap }

// baseLevel is the level of an ordinary property.
func (p *DiagnosticsProperty) baseLevel() DiagnosticLevel {
	if p.defaultLevel == DiagnosticLevelHidden {
		return p.defaultLevel
	}
	if p.Exception() != nil {
		return DiagnosticLevelError
	}
	if p.Value() == nil && p.missingIfNull {
		return DiagnosticLevelWarning
	}
	if !p.IsInteresting() {
		return DiagnosticLevelFine
	}
	return p.defaultLevel
}

// Level is the importance of the property: hidden when it was created hidden,
// error when computing its value panicked, warning for a missing value, fine
// for a value that is not interesting and its own level otherwise.
func (p *DiagnosticsProperty) Level() DiagnosticLevel {
	if p.levelF != nil {
		return p.levelF(p)
	}
	return p.baseLevel()
}

// defaultValueToString describes the value of an ordinary property.
func (p *DiagnosticsProperty) defaultValueToString(parent *TextTreeConfiguration) string {
	v := p.Value()
	if tree, ok := v.(DiagnosticableTree); ok {
		// DiagnosticableTree values are shown using the shorter
		// toStringShort() instead of the longer toString() because the
		// toString() for a DiagnosticableTree value is likely too large to be
		// useful.
		return ToStringShort(tree)
	}
	return describeValue(v)
}

// ValueToString describes the value of the property.
//
// Flutter: foundation/diagnostics.dart DiagnosticsProperty.valueToString
func (p *DiagnosticsProperty) ValueToString(parent *TextTreeConfiguration) string {
	if p.valueToStringF != nil {
		return p.valueToStringF(p, parent)
	}
	return p.defaultValueToString(parent)
}

// ToDescription is the description of the property, without its name.
func (p *DiagnosticsProperty) ToDescription(parent *TextTreeConfiguration) string {
	if d, ok := p.description.Get(); ok {
		return p.addTooltip(d)
	}
	if p.Exception() != nil {
		return "EXCEPTION (" + RuntimeTypeName(p.exception) + ")"
	}
	if ifNull, ok := p.ifNull.Get(); ok && p.Value() == nil {
		return p.addTooltip(ifNull)
	}
	result := p.ValueToString(parent)
	if ifEmpty, ok := p.ifEmpty.Get(); ok && result == "" {
		result = ifEmpty
	}
	return p.addTooltip(result)
}

func (p *DiagnosticsProperty) addTooltip(text string) string {
	if t, ok := p.tooltip.Get(); ok {
		return text + " (" + t + ")"
	}
	return text
}

// GetProperties returns the properties of the value when ExpandableValue was
// set and the value can describe itself.
func (p *DiagnosticsProperty) GetProperties() []DiagnosticsNode {
	if node := p.expandedNode(); node != nil {
		return node.GetProperties()
	}
	return nil
}

// GetChildren returns the children of the value when ExpandableValue was set
// and the value can describe itself.
func (p *DiagnosticsProperty) GetChildren() []DiagnosticsNode {
	if node := p.expandedNode(); node != nil {
		return node.GetChildren()
	}
	return nil
}

func (p *DiagnosticsProperty) expandedNode() DiagnosticsNode {
	if !p.expandable {
		return nil
	}
	switch v := p.Value().(type) {
	case DiagnosticsNode:
		return v
	case Diagnosticable:
		return ToDiagnosticsNode(v, DiagnosticsNodeOpts{Style: p.Style()})
	}
	return nil
}
