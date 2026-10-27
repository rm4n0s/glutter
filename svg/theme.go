package svg

import (
	"github.com/rm4n0s/glutter/internal/svg"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// SvgTheme gives a document the values that its text leaves to the app: what currentColor is, and what an em and an
// ex are. A document is parsed for one theme, so the theme is part of the key of the cache.
//
// The zero value is the default theme: black, a font size of 14 and an x-height of half of it. A zero field is
// replaced by its default.
//
// flutter_svg: loaders.dart SvgTheme
type SvgTheme struct {
	// CurrentColor is the value of currentColor at the root of the document (black by default).
	CurrentColor ui.Color
	// FontSize is the font size of the root and the value of 1em there (14 by default).
	FontSize float64
	// XHeight is the value of 1ex at the root (half of FontSize by default).
	XHeight float64
}

// withDefaults fills in what the theme leaves out, so that equal themes are equal values (the cache key).
func (t SvgTheme) withDefaults() SvgTheme {
	if t.CurrentColor == (ui.Color{}) {
		t.CurrentColor = ui.NewColorFromARGB(255, 0, 0, 0)
	}
	if !(t.FontSize > 0) {
		t.FontSize = 14
	}
	if !(t.XHeight > 0) {
		t.XHeight = t.FontSize / 2
	}
	return t
}

func (t SvgTheme) internal() svg.Theme {
	return svg.Theme{CurrentColor: t.CurrentColor, FontSize: t.FontSize, XHeight: t.XHeight}
}

// DefaultSvgTheme is the theme of the SvgPicture widgets below it that were not given one.
//
// flutter_svg: default_theme.dart DefaultSvgTheme
type DefaultSvgTheme struct {
	widgets.InheritedTheme
	theme SvgTheme
}

// NewDefaultSvgTheme creates the widget.
//
// flutter_svg: default_theme.dart DefaultSvgTheme.new
func NewDefaultSvgTheme(theme SvgTheme, child widgets.Widget, o ...widgets.WidgetOpts) *DefaultSvgTheme {
	return &DefaultSvgTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, o...), theme: theme}
}

// Theme is the theme to apply.
//
// flutter_svg: default_theme.dart DefaultSvgTheme.theme
func (t *DefaultSvgTheme) Theme() SvgTheme { return t.theme }

// DefaultSvgThemeOf is the closest DefaultSvgTheme above the context, and false if there is none. The context
// depends on it.
//
// flutter_svg: default_theme.dart DefaultSvgTheme.of
func DefaultSvgThemeOf(bc widgets.BuildContext) (*DefaultSvgTheme, bool) {
	return widgets.DependOn[*DefaultSvgTheme](bc)
}

// UpdateShouldNotify is true when the theme is another one.
//
// flutter_svg: default_theme.dart DefaultSvgTheme.updateShouldNotify
func (t *DefaultSvgTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return t.theme != old.(*DefaultSvgTheme).theme
}

// Wrap copies the theme around the child, for a route that is built outside its subtree.
//
// flutter_svg: default_theme.dart DefaultSvgTheme.wrap
func (t *DefaultSvgTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewDefaultSvgTheme(t.theme, child)
}
