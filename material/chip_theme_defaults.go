// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// lerpBoolDefault is `t < 0.5 ? a ?? def : b ?? def`.
func lerpBoolDefault(a, b foundation.Opt[bool], t float64, def bool) foundation.Opt[bool] {
	if t < 0.5 {
		return foundation.Some(a.OrElse(def))
	}
	return foundation.Some(b.OrElse(def))
}

// lerpBrightnessDefaultLight is `t < 0.5 ? a ?? light : b ?? light`.
func lerpBrightnessDefaultLight(a, b foundation.Opt[ui.Brightness], t float64) foundation.Opt[ui.Brightness] {
	if t < 0.5 {
		return foundation.Some(a.OrElse(ui.BrightnessLight))
	}
	return foundation.Some(b.OrElse(ui.BrightnessLight))
}

// ChipThemeDataFromDefaultsOpts are the parameters of ChipThemeDataFromDefaults.
//
// Flutter: material/chip_theme.dart ChipThemeData.fromDefaults
type ChipThemeDataFromDefaultsOpts struct {
	// Brightness says whether the chip is light or dark; give either it or
	// PrimaryColor.
	Brightness ui.Brightness
	// PrimaryColor is the color that the other colors derive from; the brightness
	// is estimated from it.
	PrimaryColor foundation.Opt[ui.Color]
}

// ChipThemeDataFromDefaults creates the data of the original (Material 2) look of
// chips, from a secondary color and a label style. In debug builds it panics with
// the tag ChipThemeBrightnessAndColor unless exactly one of the brightness and
// the primary color is given.
//
// Flutter: material/chip_theme.dart ChipThemeData.fromDefaults
func ChipThemeDataFromDefaults(secondaryColor ui.Color, labelStyle *painting.TextStyle, opts ChipThemeDataFromDefaultsOpts) ChipThemeData {
	brightness := opts.Brightness
	primary, hasPrimary := opts.PrimaryColor.Get()
	if foundation.DebugMode && hasPrimary == (brightness != ui.BrightnessUnspecified) {
		panic(errors.New(tagChipThemeBrightnessAndColor, "exactly one of the primary color and the brightness of ChipThemeDataFromDefaults must be given",
			"hasPrimaryColor", hasPrimary, "brightness", brightness.String()))
	}
	if hasPrimary {
		brightness = EstimateBrightnessForColor(primary)
	} else if brightness == ui.BrightnessLight {
		primary = Colors.Black
	} else {
		primary = Colors.White
	}
	const (
		backgroundAlpha = 0x1f // 12%
		deleteIconAlpha = 0xde // 87%
		disabledAlpha   = 0x0c // 38% * 12% = 5%
		selectAlpha     = 0x3d // 12% + 12% = 24%
		textLabelAlpha  = 0xde // 87%
	)
	secondaryLabelStyle := labelStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(secondaryColor.WithAlpha(textLabelAlpha))})
	labelStyle = labelStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(primary.WithAlpha(textLabelAlpha))})
	return ChipThemeData{
		BackgroundColor:        foundation.Some(primary.WithAlpha(backgroundAlpha)),
		DeleteIconColor:        foundation.Some(primary.WithAlpha(deleteIconAlpha)),
		DisabledColor:          foundation.Some(primary.WithAlpha(disabledAlpha)),
		SelectedColor:          foundation.Some(primary.WithAlpha(selectAlpha)),
		SecondarySelectedColor: foundation.Some(secondaryColor.WithAlpha(selectAlpha)),
		ShadowColor:            foundation.Some(Colors.Black),
		SelectedShadowColor:    foundation.Some(Colors.Black),
		ShowCheckmark:          foundation.Some(true),
		Padding:                painting.NewEdgeInsetsAll(4.0),
		LabelStyle:             labelStyle,
		SecondaryLabelStyle:    secondaryLabelStyle,
		Brightness:             foundation.Some(brightness),
		Elevation:              foundation.Some(0.0),
		PressElevation:         foundation.Some(8.0),
		IconTheme:              foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(18.0)})),
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagChipThemeBrightnessAndColor = "ChipThemeBrightnessAndColor"
)
