package painting

import (
	"slices"
	"strconv"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// StrutStyleOpts are the optional parameters of NewStrutStyle.
//
// Flutter: painting/strut_style.dart StrutStyle
type StrutStyleOpts struct {
	// FontFamily is the name of the font to use when painting the strut.
	FontFamily string
	// FontFamilyFallback is the ordered list of font families to fall back on.
	FontFamilyFallback []string
	// FontSize is the size of the strut's font; it must be greater than zero.
	FontSize foundation.Opt[float64]
	// Height is the height of the strut as a multiple of the font size.
	Height foundation.Opt[float64]
	// LeadingDistribution is how the extra height is divided above and below.
	LeadingDistribution ui.TextLeadingDistribution
	// Leading is the additional leading as a multiple of the font size; it must
	// not be negative.
	Leading foundation.Opt[float64]
	// FontWeight is the typeface thickness of the strut.
	FontWeight ui.FontWeight
	// FontStyle is the typeface variant of the strut.
	FontStyle ui.FontStyle
	// ForceStrutHeight makes every line use the strut's height when true.
	ForceStrutHeight foundation.Opt[bool]
	// DebugLabel is a human-readable description of the style.
	DebugLabel string
	// Package is the package that holds the font; the font family names are
	// prefixed with "packages/<package>/".
	Package string
}

// StrutStyle defines the strut, which sets the minimum vertical layout metrics
// of the lines of a paragraph.
//
// Flutter: painting/strut_style.dart StrutStyle
type StrutStyle struct {
	// FontFamily is the name of the font to use when painting the strut,
	// including any package prefix.
	FontFamily string
	// FontSize is the size of the strut's font.
	FontSize foundation.Opt[float64]
	// Height is the height of the strut as a multiple of the font size.
	Height foundation.Opt[float64]
	// LeadingDistribution is how the extra height is divided.
	LeadingDistribution ui.TextLeadingDistribution
	// FontWeight is the typeface thickness of the strut.
	FontWeight ui.FontWeight
	// FontStyle is the typeface variant of the strut.
	FontStyle ui.FontStyle
	// Leading is the additional leading as a multiple of the font size.
	Leading foundation.Opt[float64]
	// ForceStrutHeight makes every line use the strut's height when true.
	ForceStrutHeight foundation.Opt[bool]
	// DebugLabel is a human-readable description of the style.
	DebugLabel string

	fontFamilyFallback []string
	pkg                string
}

// NewStrutStyle creates a strut style. In debug builds a font size that is not
// greater than zero, a negative leading, or a package without a font family
// panics with the tags StrutStyleFontSizeInvalid, StrutStyleLeadingInvalid and
// StrutStylePackageWithoutFont.
//
// Flutter: painting/strut_style.dart StrutStyle.new
func NewStrutStyle(opts ...StrutStyleOpts) *StrutStyle {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.FontSize.IsSome() && !(o.FontSize.Value() > 0) {
			panic(errors.New("StrutStyleFontSizeInvalid", "the font size of a strut must be greater than zero", "fontSize", o.FontSize.Value()))
		}
		if o.Leading.IsSome() && !(o.Leading.Value() >= 0) {
			panic(errors.New("StrutStyleLeadingInvalid", "the leading of a strut must not be negative", "leading", o.Leading.Value()))
		}
		if o.Package != "" && o.FontFamily == "" && o.FontFamilyFallback == nil {
			panic(errors.New("StrutStylePackageWithoutFont", "a package needs a fontFamily or a fontFamilyFallback", "package", o.Package))
		}
	}
	family := o.FontFamily
	if o.Package != "" {
		family = "packages/" + o.Package + "/" + o.FontFamily
	}
	return &StrutStyle{
		FontFamily: family, FontSize: o.FontSize, Height: o.Height, LeadingDistribution: o.LeadingDistribution,
		FontWeight: o.FontWeight, FontStyle: o.FontStyle, Leading: o.Leading, ForceStrutHeight: o.ForceStrutHeight,
		DebugLabel: o.DebugLabel, fontFamilyFallback: slices.Clone(o.FontFamilyFallback), pkg: o.Package,
	}
}

// NewStrutStyleFromTextStyle creates a strut style from a text style, with the
// options overriding it.
//
// Flutter: painting/strut_style.dart StrutStyle.fromTextStyle
func NewStrutStyleFromTextStyle(textStyle *TextStyle, opts ...StrutStyleOpts) *StrutStyle {
	o := firstOpts(opts)
	family := textStyle.FontFamily
	if o.FontFamily != "" {
		family = o.FontFamily
		if o.Package != "" {
			family = "packages/" + o.Package + "/" + o.FontFamily
		}
	}
	// Like Flutter, the package prefixes the family a second time, because the
	// constructor it forwards to prefixes it again.
	if o.Package != "" {
		family = "packages/" + o.Package + "/" + family
	}
	fallback := o.FontFamilyFallback
	if fallback == nil {
		fallback = textStyle.FontFamilyFallback()
	}
	return &StrutStyle{
		FontFamily:          family,
		FontSize:            orOpt(o.FontSize, textStyle.FontSize),
		Height:              orOpt(o.Height, textStyle.Height),
		LeadingDistribution: orEnum(o.LeadingDistribution, textStyle.LeadingDistribution),
		Leading:             o.Leading,
		FontWeight:          orEnum(o.FontWeight, textStyle.FontWeight),
		FontStyle:           orEnum(o.FontStyle, textStyle.FontStyle),
		ForceStrutHeight:    o.ForceStrutHeight,
		DebugLabel:          orString(o.DebugLabel, textStyle.DebugLabel),
		fontFamilyFallback:  slices.Clone(fallback),
		pkg:                 o.Package,
	}
}

// StrutStyleDisabled is the strut that has no height and no leading, so it
// takes no space and only the text sets the line height.
//
// Flutter: painting/strut_style.dart StrutStyle.disabled
func StrutStyleDisabled() *StrutStyle {
	return NewStrutStyle(StrutStyleOpts{Height: foundation.Some(0.0), Leading: foundation.Some(0.0)})
}

// FontFamilyFallback is the ordered list of font families to fall back on,
// with the package prefix applied.
//
// Flutter: painting/strut_style.dart StrutStyle.fontFamilyFallback
func (s *StrutStyle) FontFamilyFallback() []string {
	return prefixFamilies(s.fontFamilyFallback, s.pkg)
}

func prefixFamilies(families []string, pkg string) []string {
	if families == nil || pkg == "" {
		return families
	}
	out := make([]string, len(families))
	for i, f := range families {
		out[i] = "packages/" + pkg + "/" + f
	}
	return out
}

func orOpt[T any](a, b foundation.Opt[T]) foundation.Opt[T] {
	if a.IsSome() {
		return a
	}
	return b
}

func orEnum[T comparable](a, b T) T {
	var zero T
	if a != zero {
		return a
	}
	return b
}

func orString(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// CompareTo describes the difference between two strut styles: layout when one
// changes the layout, else identical.
//
// Flutter: painting/strut_style.dart StrutStyle.compareTo
func (s *StrutStyle) CompareTo(other *StrutStyle) RenderComparison {
	if s == other {
		return RenderComparisonIdentical
	}
	if s.FontFamily != other.FontFamily || s.FontSize != other.FontSize || s.FontWeight != other.FontWeight ||
		s.FontStyle != other.FontStyle || s.Height != other.Height || s.Leading != other.Leading ||
		s.ForceStrutHeight != other.ForceStrutHeight ||
		!slices.Equal(s.FontFamilyFallback(), other.FontFamilyFallback()) ||
		(s.Height.IsSome() && s.LeadingDistribution != other.LeadingDistribution) {
		return RenderComparisonLayout
	}
	return RenderComparisonIdentical
}

// InheritFromTextStyle returns a strut style with the unset fields taken from
// the text style.
//
// Flutter: painting/strut_style.dart StrutStyle.inheritFromTextStyle
func (s *StrutStyle) InheritFromTextStyle(other *TextStyle) *StrutStyle {
	if other == nil {
		return s
	}
	effectiveHeight := orOpt(s.Height, other.Height)
	out := &StrutStyle{
		FontFamily:         orString(s.FontFamily, other.FontFamily),
		FontSize:           orOpt(s.FontSize, other.FontSize),
		Height:             effectiveHeight,
		FontWeight:         orEnum(s.FontWeight, other.FontWeight),
		FontStyle:          orEnum(s.FontStyle, other.FontStyle),
		Leading:            s.Leading,
		ForceStrutHeight:   s.ForceStrutHeight,
		DebugLabel:         orString(s.DebugLabel, other.DebugLabel),
		fontFamilyFallback: s.FontFamilyFallback(),
	}
	if s.fontFamilyFallback == nil {
		out.fontFamilyFallback = other.FontFamilyFallback()
	}
	if effectiveHeight.IsSome() {
		out.LeadingDistribution = orEnum(s.LeadingDistribution, other.LeadingDistribution)
	}
	return out
}

// Merge returns a strut style where the fields of other replace the ones of
// this strut style, where other has them.
//
// Flutter: painting/strut_style.dart StrutStyle.merge
func (s *StrutStyle) Merge(other *StrutStyle) *StrutStyle {
	if other == nil {
		return s
	}
	out := &StrutStyle{
		FontFamily:          orString(other.FontFamily, s.FontFamily),
		FontSize:            orOpt(other.FontSize, s.FontSize),
		Height:              orOpt(other.Height, s.Height),
		LeadingDistribution: orEnum(other.LeadingDistribution, s.LeadingDistribution),
		Leading:             orOpt(other.Leading, s.Leading),
		FontWeight:          orEnum(other.FontWeight, s.FontWeight),
		FontStyle:           orEnum(other.FontStyle, s.FontStyle),
		ForceStrutHeight:    orOpt(other.ForceStrutHeight, s.ForceStrutHeight),
		DebugLabel:          orString(other.DebugLabel, s.DebugLabel),
	}
	if other.fontFamilyFallback != nil {
		out.fontFamilyFallback, out.pkg = other.fontFamilyFallback, other.pkg
	} else {
		out.fontFamilyFallback, out.pkg = s.fontFamilyFallback, s.pkg
	}
	return out
}

// Equal is Dart's == on strut styles.
//
// Flutter: painting/strut_style.dart StrutStyle.operator ==
func (s *StrutStyle) Equal(other *StrutStyle) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil {
		return false
	}
	return s.FontFamily == other.FontFamily && s.FontSize == other.FontSize && s.FontWeight == other.FontWeight &&
		s.FontStyle == other.FontStyle && s.Height == other.Height && s.Leading == other.Leading &&
		s.ForceStrutHeight == other.ForceStrutHeight &&
		(s.Height.IsNone() || s.LeadingDistribution == other.LeadingDistribution) &&
		slices.Equal(s.FontFamilyFallback(), other.FontFamilyFallback())
}

// ToUI converts the strut style to the engine's, scaling the font size.
func (s *StrutStyle) toUI(scaler TextScaler) *ui.StrutStyle {
	o := ui.StrutStyleOpts{
		FontFamily:          s.FontFamily,
		FontFamilyFallback:  s.FontFamilyFallback(),
		LeadingDistribution: s.LeadingDistribution,
		FontWeight:          s.FontWeight,
		FontStyle:           s.FontStyle,
		Height:              optPtr(s.Height),
		Leading:             optPtr(s.Leading),
		ForceStrutHeight:    optPtr(s.ForceStrutHeight),
	}
	if s.FontSize.IsSome() {
		o.FontSize = ui.Ptr(scaler.Scale(s.FontSize.Value()))
	}
	return ui.NewStrutStyle(o)
}

func optPtr[T any](o foundation.Opt[T]) *T {
	if v, ok := o.Get(); ok {
		return &v
	}
	return nil
}

// ToStringShort names the style.
//
// Flutter: painting/strut_style.dart StrutStyle.toStringShort
func (s *StrutStyle) ToStringShort() string { return "StrutStyle" }

// String describes the style and its properties.
//
// Flutter: painting/strut_style.dart StrutStyle.toString
func (s *StrutStyle) String() string { return foundation.DiagnosticableString(s) }

// DebugFillProperties adds the properties of the style.
//
// Flutter: painting/strut_style.dart StrutStyle.debugFillProperties
func (s *StrutStyle) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	s.DebugFillPropertiesWithPrefix(p, "")
}

// DebugFillPropertiesWithPrefix is DebugFillProperties with a prefix before
// each property name, which TextSpan uses for the "style: " properties.
//
// Flutter: painting/strut_style.dart StrutStyle.debugFillProperties
func (s *StrutStyle) DebugFillPropertiesWithPrefix(p *foundation.DiagnosticPropertiesBuilder, prefix string) {
	null := foundation.DiagnosticsPropertyOpts{DefaultValue: foundation.DefaultNull}
	if s.DebugLabel != "" {
		p.Add(foundation.NewMessageProperty(prefix+"debugLabel", s.DebugLabel))
	}
	var family foundation.Opt[string]
	if s.FontFamily != "" {
		family = foundation.Some(s.FontFamily)
	}
	var fallback []string
	if s.fontFamilyFallback != nil {
		fallback = s.FontFamilyFallback()
	}
	styles := []foundation.DiagnosticsNode{
		foundation.NewStringProperty(prefix+"family", family, foundation.StringPropertyOpts{DefaultValue: foundation.DefaultNull, Unquoted: true}),
		foundation.NewIterableProperty(prefix+"familyFallback", fallback, foundation.IterablePropertyOpts{DefaultValue: foundation.DefaultNull}),
		foundation.NewDoubleProperty(prefix+"size", s.FontSize, foundation.NumberPropertyOpts{DefaultValue: foundation.DefaultNull}),
	}
	weight := foundation.None[ui.FontWeight]()
	weightOpts := null
	if s.FontWeight != 0 {
		weight = foundation.Some(s.FontWeight)
		weightOpts.Description = "w" + strconv.Itoa(int(s.FontWeight))
	}
	styles = append(styles,
		foundation.NewDiagnosticsPropertyOpt(prefix+"weight", weight, weightOpts),
		foundation.NewEnumProperty(prefix+"style", optEnum(s.FontStyle), foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultNull}),
		foundation.NewDoubleProperty(prefix+"height", s.Height, foundation.NumberPropertyOpts{Unit: "x", DefaultValue: foundation.DefaultNull}),
		foundation.NewFlagProperty(prefix+"forceStrutHeight", s.ForceStrutHeight,
			foundation.FlagPropertyOpts{IfTrue: prefix + "<strut height forced>", IfFalse: prefix + "<strut height normal>"}),
	)
	if s.Height.IsSome() {
		styles = append(styles, foundation.NewEnumProperty(prefix+"leadingDistribution", optEnum(s.LeadingDistribution),
			foundation.EnumPropertyOpts{DefaultValue: foundation.DefaultNull}))
	}
	styleSpecified := false
	for _, n := range styles {
		if !n.IsFiltered(foundation.DiagnosticLevelInfo) {
			styleSpecified = true
		}
	}
	for _, n := range styles {
		p.Add(n)
	}
	if !styleSpecified {
		p.Add(foundation.NewFlagProperty("forceStrutHeight", s.ForceStrutHeight,
			foundation.FlagPropertyOpts{IfTrue: prefix + "<strut height forced>", IfFalse: prefix + "<strut height normal>"}))
	}
}

// optEnum is None for the zero (unspecified) value of an enum.
func optEnum[T interface {
	comparable
	Name() string
}](v T) foundation.Opt[T] {
	var zero T
	if v == zero {
		return foundation.None[T]()
	}
	return foundation.Some(v)
}
