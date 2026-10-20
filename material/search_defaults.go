// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// searchViewFullScreenBarHeight is the height of the search field of a full screen
// search view.
//
// Flutter: material/search_anchor.dart _SearchViewDefaultsM3.fullScreenBarHeight
const searchViewFullScreenBarHeight = 72.0

// searchBarDefaultsM3 are the defaults of the Material 3 search bar.
//
// Flutter: material/search_anchor.dart _SearchBarDefaultsM3
func searchBarDefaultsM3(bc widgets.BuildContext) SearchBarThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	textTheme := theme.TextTheme
	return SearchBarThemeData{
		BackgroundColor:  widgets.NewWidgetStatePropertyAll(foundation.Some(colors.SurfaceContainerHigh())),
		Elevation:        widgets.NewWidgetStatePropertyAll(foundation.Some(6.0)),
		ShadowColor:      widgets.NewWidgetStatePropertyAll(foundation.Some(colors.Shadow())),
		SurfaceTintColor: widgets.NewWidgetStatePropertyAll(foundation.Some(Colors.Transparent)),
		OverlayColor: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
			switch {
			case states.Has(widgets.WidgetStatePressed):
				return foundation.Some(colors.OnSurface().WithOpacity(0.1))
			case states.Has(widgets.WidgetStateHovered):
				return foundation.Some(colors.OnSurface().WithOpacity(0.08))
			}
			return foundation.Some(Colors.Transparent)
		}),
		Shape:   widgets.NewWidgetStatePropertyAll[painting.OutlinedBorder](painting.NewStadiumBorder()),
		Padding: widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](painting.NewEdgeInsetsSymmetric(0, 8.0)),
		TextStyle: widgets.NewWidgetStatePropertyAll(
			textTheme.BodyLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurface())})),
		HintStyle: widgets.NewWidgetStatePropertyAll(
			textTheme.BodyLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurfaceVariant())})),
		Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{
			MinWidth: foundation.Some(360.0), MaxWidth: foundation.Some(800.0), MinHeight: foundation.Some(56.0)})),
		TextCapitalization: 0,
	}
}

// searchViewDefaultsM3 are the defaults of the Material 3 search view.
//
// Flutter: material/search_anchor.dart _SearchViewDefaultsM3
func searchViewDefaultsM3(bc widgets.BuildContext, isFullScreen bool) SearchViewThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	textTheme := theme.TextTheme
	var shape painting.OutlinedBorder = painting.NewRoundedRectangleBorder()
	if !isFullScreen {
		shape = painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(28.0))})
	}
	return SearchViewThemeData{
		BackgroundColor:  foundation.Some(colors.SurfaceContainerHigh()),
		Elevation:        foundation.Some(6.0),
		SurfaceTintColor: foundation.Some(Colors.Transparent),
		Shape:            shape,
		HeaderTextStyle:  textTheme.BodyLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurface())}),
		HeaderHintStyle:  textTheme.BodyLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurfaceVariant())}),
		Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{
			MinWidth: foundation.Some(360.0), MinHeight: foundation.Some(240.0)})),
		BarPadding:   painting.NewEdgeInsetsSymmetric(0, 8.0),
		ShrinkWrap:   foundation.Some(false),
		DividerColor: foundation.Some(colors.Outline()),
	}
}
