package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// ScriptCategory is a family of writing systems that share text geometry.
//
// Flutter: material/typography.dart ScriptCategory
type ScriptCategory int

const (
	// ScriptCategoryEnglishLike is the category of scripts such as English,
	// French, Russian and Greek. It is the zero value.
	//
	// Flutter: material/typography.dart ScriptCategory.englishLike
	ScriptCategoryEnglishLike ScriptCategory = iota
	// ScriptCategoryDense is the category of scripts such as Chinese, Japanese
	// and Korean.
	//
	// Flutter: material/typography.dart ScriptCategory.dense
	ScriptCategoryDense
	// ScriptCategoryTall is the category of scripts such as Farsi, Hindi and
	// Thai.
	//
	// Flutter: material/typography.dart ScriptCategory.tall
	ScriptCategoryTall
)

// Typography is the color and geometry TextTheme values used by a Theme: the
// black and white themes give the colors and font families of the platform,
// the englishLike, dense and tall themes give the geometry of a script.
//
// Flutter: material/typography.dart Typography
type Typography struct {
	// Black is a Material Design text theme with dark glyphs.
	Black *TextTheme
	// White is a Material Design text theme with light glyphs.
	White *TextTheme
	// EnglishLike is the text geometry for English-like scripts.
	EnglishLike *TextTheme
	// Dense is the text geometry for dense scripts, such as Chinese, Japanese
	// and Korean.
	Dense *TextTheme
	// Tall is the text geometry for tall scripts, such as Farsi, Hindi and
	// Thai.
	Tall *TextTheme
}

// TypographyOpts are the optional parameters of the Typography constructors.
// Platform defaults to Android; when it is unset, Black and White must both be
// given.
//
// Flutter: material/typography.dart Typography.new
type TypographyOpts struct {
	// Platform picks the color themes (and so the font families).
	Platform foundation.Opt[foundation.TargetPlatform]
	// Black replaces the black theme of the platform.
	Black *TextTheme
	// White replaces the white theme of the platform.
	White *TextTheme
	// EnglishLike replaces the geometry of English-like scripts.
	EnglishLike *TextTheme
	// Dense replaces the geometry of dense scripts.
	Dense *TextTheme
	// Tall replaces the geometry of tall scripts.
	Tall *TextTheme
}

// Typography2021Opts are the optional parameters of NewTypographyMaterial2021:
// those of the other constructors and the color scheme that gives the colors of
// the black and white themes.
//
// Flutter: material/typography.dart Typography.material2021
type Typography2021Opts struct {
	TypographyOpts
	// ColorScheme gives the colors of the themes; the default is the light
	// scheme.
	ColorScheme *ColorScheme
}

// NewTypography creates the typography of Material 2018 (Typography.material2018).
//
// Flutter: material/typography.dart Typography.new
func NewTypography(opts ...TypographyOpts) *Typography { return NewTypographyMaterial2018(opts...) }

// NewTypographyMaterial2014 creates the typography of the 2014 Material
// Design specification.
//
// Flutter: material/typography.dart Typography.material2014
func NewTypographyMaterial2014(opts ...TypographyOpts) *Typography {
	o := firstOpts(opts)
	return typographyWithPlatform(o, TypographyEnglishLike2014, TypographyDense2014, TypographyTall2014)
}

// NewTypographyMaterial2018 creates the typography of the 2018 Material Design
// specification.
//
// Flutter: material/typography.dart Typography.material2018
func NewTypographyMaterial2018(opts ...TypographyOpts) *Typography {
	o := firstOpts(opts)
	return typographyWithPlatform(o, TypographyEnglishLike2018, TypographyDense2018, TypographyTall2018)
}

// NewTypographyMaterial2021 creates the typography of the Material 3
// specification. The black and white themes get a uniform color from the
// color scheme.
//
// Flutter: material/typography.dart Typography.material2021
func NewTypographyMaterial2021(opts ...Typography2021Opts) *Typography {
	o := firstOpts(opts)
	scheme := o.ColorScheme
	if scheme == nil {
		scheme = NewColorSchemeLight()
	}
	base := typographyWithPlatform(o.TypographyOpts, TypographyEnglishLike2021, TypographyDense2021, TypographyTall2021)
	dark, light := scheme.Surface(), scheme.OnSurface()
	if scheme.Brightness() == ui.BrightnessLight {
		dark, light = scheme.OnSurface(), scheme.Surface()
	}
	return base.CopyWith(TypographyOpts{
		Black: base.Black.Apply(TextThemeApply{DisplayColor: foundation.Some(dark), BodyColor: foundation.Some(dark), DecorationColor: foundation.Some(dark)}),
		White: base.White.Apply(TextThemeApply{DisplayColor: foundation.Some(light), BodyColor: foundation.Some(light), DecorationColor: foundation.Some(light)}),
	})
}

func typographyWithPlatform(o TypographyOpts, englishLike, dense, tall *TextTheme) *Typography {
	// Flutter's default platform is Android; a null platform needs both themes,
	// and then the themes win anyway.
	platform := o.Platform.OrElse(foundation.TargetPlatformAndroid)
	black, white := o.Black, o.White
	if black == nil || white == nil {
		var b, w *TextTheme
		switch platform {
		case foundation.TargetPlatformIOS:
			b, w = TypographyBlackCupertino, TypographyWhiteCupertino
		case foundation.TargetPlatformAndroid, foundation.TargetPlatformFuchsia:
			b, w = TypographyBlackMountainView, TypographyWhiteMountainView
		case foundation.TargetPlatformWindows:
			b, w = TypographyBlackRedmond, TypographyWhiteRedmond
		case foundation.TargetPlatformMacOS:
			b, w = TypographyBlackRedwoodCity, TypographyWhiteRedwoodCity
		case foundation.TargetPlatformLinux:
			b, w = TypographyBlackHelsinki, TypographyWhiteHelsinki
		default:
			panic(errors.New("TypographyUnknownPlatform", "the platform has no typography", "platform", platform.Name()))
		}
		if black == nil {
			black = b
		}
		if white == nil {
			white = w
		}
	}
	pick := func(given, def *TextTheme) *TextTheme {
		if given != nil {
			return given
		}
		return def
	}
	return &Typography{
		Black: black, White: white,
		EnglishLike: pick(o.EnglishLike, englishLike), Dense: pick(o.Dense, dense), Tall: pick(o.Tall, tall),
	}
}

// GeometryThemeFor returns the text geometry theme for the script category.
//
// Flutter: material/typography.dart Typography.geometryThemeFor
func (t *Typography) GeometryThemeFor(category ScriptCategory) *TextTheme {
	switch category {
	case ScriptCategoryDense:
		return t.Dense
	case ScriptCategoryTall:
		return t.Tall
	default:
		return t.EnglishLike
	}
}

// CopyWith creates a copy of this typography but with the themes of c that
// are not nil replaced (Platform is ignored).
//
// Flutter: material/typography.dart Typography.copyWith
func (t *Typography) CopyWith(c TypographyOpts) *Typography {
	pick := func(given, def *TextTheme) *TextTheme {
		if given != nil {
			return given
		}
		return def
	}
	return &Typography{
		Black: pick(c.Black, t.Black), White: pick(c.White, t.White),
		EnglishLike: pick(c.EnglishLike, t.EnglishLike), Dense: pick(c.Dense, t.Dense), Tall: pick(c.Tall, t.Tall),
	}
}

// TypographyLerp linearly interpolates between two typographies.
//
// Flutter: material/typography.dart Typography.lerp
func TypographyLerp(a, b *Typography, t float64) *Typography {
	if a == b {
		return a
	}
	return &Typography{
		Black: TextThemeLerp(a.Black, b.Black, t), White: TextThemeLerp(a.White, b.White, t),
		EnglishLike: TextThemeLerp(a.EnglishLike, b.EnglishLike, t),
		Dense:       TextThemeLerp(a.Dense, b.Dense, t),
		Tall:        TextThemeLerp(a.Tall, b.Tall, t),
	}
}

// Equal reports whether the two typographies have equal themes.
//
// Flutter: material/typography.dart Typography.==
func (t *Typography) Equal(other *Typography) bool {
	if t == other {
		return true
	}
	if t == nil || other == nil {
		return false
	}
	return t.Black.Equal(other.Black) && t.White.Equal(other.White) && t.EnglishLike.Equal(other.EnglishLike) &&
		t.Dense.Equal(other.Dense) && t.Tall.Equal(other.Tall)
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (t *Typography) ToStringShort() string { return "Typography" }

// String returns the description of the typography.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (t *Typography) String() string { return foundation.DiagnosticableString(t) }

// DebugFillProperties adds the themes that differ from those of
// Typography.material2014.
//
// Flutter: material/typography.dart Typography.debugFillProperties
func (t *Typography) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	def := NewTypographyMaterial2014()
	add := func(name string, v, d *TextTheme) {
		properties.Add(foundation.NewDiagnosticsProperty(name, v, foundation.DiagnosticsPropertyOpts{DefaultValue: d}))
	}
	add("black", t.Black, def.Black)
	add("white", t.White, def.White)
	add("englishLike", t.EnglishLike, def.EnglishLike)
	add("dense", t.Dense, def.Dense)
	add("tall", t.Tall, def.Tall)
}
