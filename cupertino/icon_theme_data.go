package cupertino

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoIconThemeDataOpts are the optional parameters of NewCupertinoIconThemeData.
//
// Flutter: cupertino/icon_theme_data.dart CupertinoIconThemeData.new
type CupertinoIconThemeDataOpts struct {
	// Size is the size of icons in logical pixels.
	Size foundation.Opt[float64]
	// Fill is the fill of the glyph, from 0 to 1.
	Fill foundation.Opt[float64]
	// Weight is the stroke weight of the glyph; it must be greater than zero.
	Weight foundation.Opt[float64]
	// Grade is the grade of the glyph.
	Grade foundation.Opt[float64]
	// OpticalSize is the optical size of the glyph; it must be greater than zero.
	OpticalSize foundation.Opt[float64]
	// Color is the default color of icons: a ui.Color or a CupertinoDynamicColor, which
	// IconThemeData.Resolve resolves in the context; nil is unset.
	Color ui.ColorValue
	// Opacity is the opacity applied to the color, clamped to 0 to 1.
	Opacity foundation.Opt[float64]
	// Shadows are the shadows painted behind icons.
	Shadows []ui.Shadow
	// ApplyTextScaling is whether the size is scaled with the text scaler.
	ApplyTextScaling foundation.Opt[bool]
}

// NewCupertinoIconThemeData creates icon theme data whose color can be a
// CupertinoDynamicColor: IconThemeData.Resolve (which IconThemeOf calls) resolves it in
// the context of the icon.
//
// Deviation: Dart's CupertinoIconThemeData is a subclass of IconThemeData; IconThemeData
// here is a struct that keeps a color that depends on the context itself
// (IconThemeDataOpts.ColorValue), so this returns an IconThemeData (SPEC §12.10).
//
// Flutter: cupertino/icon_theme_data.dart CupertinoIconThemeData.new
func NewCupertinoIconThemeData(opts ...CupertinoIconThemeDataOpts) widgets.IconThemeData {
	var o CupertinoIconThemeDataOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	data := widgets.IconThemeDataOpts{Size: o.Size, Fill: o.Fill, Weight: o.Weight, Grade: o.Grade, OpticalSize: o.OpticalSize,
		Opacity: o.Opacity, Shadows: o.Shadows, ApplyTextScaling: o.ApplyTextScaling}
	if c, ok := o.Color.(ui.Color); ok {
		data.Color = foundation.Some(c)
	} else {
		data.ColorValue = o.Color
	}
	return widgets.NewIconThemeData(data)
}
