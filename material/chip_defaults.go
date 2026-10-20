// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// chipM3LabelPadding is the label padding of the Material 3 chips: it goes from
// 8 to 4 on each side as the scaled font size goes from 14 to 28.
func chipM3LabelPadding(bc widgets.BuildContext, labelStyle *painting.TextStyle) painting.EdgeInsetsGeometry {
	fontSize := 14.0
	if labelStyle != nil {
		fontSize = labelStyle.FontSize.OrElse(14.0)
	}
	return chipLabelPaddingForScale(widgets.MediaQueryTextScalerOf(bc).Scale(fontSize) / 14.0)
}

// chipDefaultsM3 are the defaults of the Material 3 chip, when the variants do not
// have their own.
//
// Flutter: material/chip.dart _ChipDefaultsM3
func chipDefaultsM3(bc widgets.BuildContext, isEnabled bool) ChipThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	labelColor := colors.OnSurface()
	deleteIconColor := colors.OnSurface()
	side := painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.OnSurface().WithOpacity(0.12))})
	iconColor := colors.OnSurface()
	if isEnabled {
		labelColor = colors.OnSurfaceVariant()
		deleteIconColor = colors.OnSurfaceVariant()
		side = painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.OutlineVariant())})
		iconColor = colors.Primary()
	}
	labelStyle := theme.TextTheme.LabelLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(labelColor)})
	return ChipThemeData{
		Elevation:        foundation.Some(0.0),
		Shape:            painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(8.0))}),
		ShowCheckmark:    foundation.Some(true),
		LabelStyle:       labelStyle,
		ShadowColor:      foundation.Some(Colors.Transparent),
		SurfaceTintColor: foundation.Some(Colors.Transparent),
		DeleteIconColor:  foundation.Some(deleteIconColor),
		Side:             foundation.Some(side),
		IconTheme:        foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(iconColor), Size: foundation.Some(18.0)})),
		Padding:          painting.NewEdgeInsetsAll(8.0),
		LabelPadding:     chipM3LabelPadding(bc, labelStyle),
	}
}
