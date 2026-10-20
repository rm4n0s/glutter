// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// ListTileThemeData defines default property values for descendant ListTile widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/list_tile_theme.dart ListTileThemeData
//
//glutter:copywith value
type ListTileThemeData struct {
	// Dense is whether the tile is part of a vertically dense list.
	Dense foundation.Opt[bool]
	// Shape is the shape of the tile.
	Shape painting.ShapeBorder
	// Style is the style of the tile: how the text styles and the vertical sizes are chosen.
	Style ListTileStyle
	// SelectedColor is the color of the icons and the text of a selected tile.
	SelectedColor foundation.Opt[ui.Color]
	// IconColor is the color of the icons of an unselected tile.
	IconColor foundation.Opt[ui.Color]
	// TextColor is the color of the text of an unselected tile.
	TextColor foundation.Opt[ui.Color]
	// TitleTextStyle is the style of the title.
	TitleTextStyle *painting.TextStyle
	// SubtitleTextStyle is the style of the subtitle.
	SubtitleTextStyle *painting.TextStyle
	// LeadingAndTrailingTextStyle is the style of the leading and the trailing widgets' text.
	LeadingAndTrailingTextStyle *painting.TextStyle
	// ContentPadding is the padding of the tile's content.
	ContentPadding painting.EdgeInsetsGeometry
	// TileColor is the background color of the tile.
	TileColor foundation.Opt[ui.Color]
	// SelectedTileColor is the background color of a selected tile.
	SelectedTileColor foundation.Opt[ui.Color]
	// HorizontalTitleGap is the horizontal gap between the title and the leading and trailing widgets.
	HorizontalTitleGap foundation.Opt[float64]
	// MinVerticalPadding is the minimum padding above and below the tile's content.
	MinVerticalPadding foundation.Opt[float64]
	// MinLeadingWidth is the minimum width allocated to the leading widget.
	MinLeadingWidth foundation.Opt[float64]
	// MinTileHeight is the minimum height of the tile.
	MinTileHeight foundation.Opt[float64]
	// EnableFeedback is whether a tap gives feedback.
	EnableFeedback foundation.Opt[bool]
	// MouseCursor is the cursor for the states of the tile.
	MouseCursor widgets.WidgetStateProperty[services.MouseCursor]
	// VisualDensity is how compact the tile's layout is.
	VisualDensity foundation.Opt[VisualDensity]
	// TitleAlignment is where the title aligns against the leading and the trailing widgets.
	TitleAlignment ListTileTitleAlignment
	// ControlAffinity is where the control of a checkbox, switch or radio tile goes.
	ControlAffinity ListTileControlAffinity
	// IsThreeLine is whether the tile is three lines tall.
	IsThreeLine foundation.Opt[bool]
}

// ListTileThemeDataOpts are the parameters of NewListTileThemeData: the fields of the data.
//
// Flutter: material/list_tile_theme.dart ListTileThemeData.new
type ListTileThemeDataOpts = ListTileThemeData

// NewListTileThemeData creates the data; every field is optional.
//
// Flutter: material/list_tile_theme.dart ListTileThemeData.new
func NewListTileThemeData(opts ...ListTileThemeDataOpts) ListTileThemeData {
	return firstOpts(opts)
}

// ListTileThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/list_tile_theme.dart ListTileThemeData.lerp
func ListTileThemeDataLerp(a, b ListTileThemeData, t float64) ListTileThemeData {
	return ListTileThemeData{
		Dense:                       lerpFirstOrSecond(a.Dense, b.Dense, t),
		Shape:                       painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		Style:                       lerpFirstOrSecond(a.Style, b.Style, t),
		SelectedColor:               lerpOptColor(a.SelectedColor, b.SelectedColor, t),
		IconColor:                   lerpOptColor(a.IconColor, b.IconColor, t),
		TextColor:                   lerpOptColor(a.TextColor, b.TextColor, t),
		TitleTextStyle:              painting.TextStyleLerp(a.TitleTextStyle, b.TitleTextStyle, t),
		SubtitleTextStyle:           painting.TextStyleLerp(a.SubtitleTextStyle, b.SubtitleTextStyle, t),
		LeadingAndTrailingTextStyle: painting.TextStyleLerp(a.LeadingAndTrailingTextStyle, b.LeadingAndTrailingTextStyle, t),
		ContentPadding:              painting.EdgeInsetsGeometryLerp(a.ContentPadding, b.ContentPadding, t),
		TileColor:                   lerpOptColor(a.TileColor, b.TileColor, t),
		SelectedTileColor:           lerpOptColor(a.SelectedTileColor, b.SelectedTileColor, t),
		HorizontalTitleGap:          lerpOptDouble(a.HorizontalTitleGap, b.HorizontalTitleGap, t),
		MinVerticalPadding:          lerpOptDouble(a.MinVerticalPadding, b.MinVerticalPadding, t),
		MinLeadingWidth:             lerpOptDouble(a.MinLeadingWidth, b.MinLeadingWidth, t),
		MinTileHeight:               lerpOptDouble(a.MinTileHeight, b.MinTileHeight, t),
		EnableFeedback:              lerpFirstOrSecond(a.EnableFeedback, b.EnableFeedback, t),
		MouseCursor:                 lerpFirstOrSecond(a.MouseCursor, b.MouseCursor, t),
		VisualDensity:               lerpFirstOrSecond(a.VisualDensity, b.VisualDensity, t),
		TitleAlignment:              lerpFirstOrSecond(a.TitleAlignment, b.TitleAlignment, t),
		ControlAffinity:             lerpFirstOrSecond(a.ControlAffinity, b.ControlAffinity, t),
		IsThreeLine:                 lerpFirstOrSecond(a.IsThreeLine, b.IsThreeLine, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/list_tile_theme.dart ListTileThemeData.==
func (d ListTileThemeData) Equal(o ListTileThemeData) bool {
	return d.Dense == o.Dense &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		d.Style == o.Style &&
		d.SelectedColor == o.SelectedColor &&
		d.IconColor == o.IconColor &&
		d.TextColor == o.TextColor &&
		textStyleEqual(d.TitleTextStyle, o.TitleTextStyle) &&
		textStyleEqual(d.SubtitleTextStyle, o.SubtitleTextStyle) &&
		textStyleEqual(d.LeadingAndTrailingTextStyle, o.LeadingAndTrailingTextStyle) &&
		foundation.ValuesEqual(d.ContentPadding, o.ContentPadding) &&
		d.TileColor == o.TileColor &&
		d.SelectedTileColor == o.SelectedTileColor &&
		d.HorizontalTitleGap == o.HorizontalTitleGap &&
		d.MinVerticalPadding == o.MinVerticalPadding &&
		d.MinLeadingWidth == o.MinLeadingWidth &&
		d.MinTileHeight == o.MinTileHeight &&
		d.EnableFeedback == o.EnableFeedback &&
		foundation.ValuesEqual(d.MouseCursor, o.MouseCursor) &&
		d.VisualDensity == o.VisualDensity &&
		d.TitleAlignment == o.TitleAlignment &&
		d.ControlAffinity == o.ControlAffinity &&
		d.IsThreeLine == o.IsThreeLine
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (ListTileThemeData) ToStringShort() string { return "ListTileThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d ListTileThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/list_tile_theme.dart ListTileThemeData.debugFillProperties
func (d ListTileThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("dense", optValue(d.Dense)))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("style", nilIfZero(d.Style)))
	p.Add(colorProp("selectedColor", d.SelectedColor))
	p.Add(colorProp("iconColor", d.IconColor))
	p.Add(colorProp("textColor", d.TextColor))
	p.Add(objectProp("titleTextStyle", d.TitleTextStyle))
	p.Add(objectProp("subtitleTextStyle", d.SubtitleTextStyle))
	p.Add(objectProp("leadingAndTrailingTextStyle", d.LeadingAndTrailingTextStyle))
	p.Add(objectProp("contentPadding", d.ContentPadding))
	p.Add(colorProp("tileColor", d.TileColor))
	p.Add(colorProp("selectedTileColor", d.SelectedTileColor))
	p.Add(doubleProp("horizontalTitleGap", d.HorizontalTitleGap))
	p.Add(doubleProp("minVerticalPadding", d.MinVerticalPadding))
	p.Add(doubleProp("minLeadingWidth", d.MinLeadingWidth))
	p.Add(doubleProp("minTileHeight", d.MinTileHeight))
	p.Add(objectProp("enableFeedback", optValue(d.EnableFeedback)))
	p.Add(objectProp("mouseCursor", propOrNil(d.MouseCursor)))
	p.Add(objectProp("visualDensity", optValue(d.VisualDensity)))
	p.Add(objectProp("titleAlignment", nilIfZero(d.TitleAlignment)))
	p.Add(objectProp("controlAffinity", nilIfZero(d.ControlAffinity)))
	p.Add(objectProp("isThreeLine", optValue(d.IsThreeLine)))
}
