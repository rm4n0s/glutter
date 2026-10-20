// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/widgets"
)

// ListTileTheme is an inherited theme that overrides the ListTileThemeData of
// the theme for the list tiles below it.
//
// Deviation: Flutter's constructor also takes the fields one by one (legacy);
// here the theme takes the data only (SPEC §12.10).
//
// Flutter: material/list_tile_theme.dart ListTileTheme
type ListTileTheme struct {
	widgets.InheritedTheme
	data ListTileThemeData
}

// NewListTileTheme creates the theme.
//
// Flutter: material/list_tile_theme.dart ListTileTheme.new
func NewListTileTheme(data ListTileThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *ListTileTheme {
	return &ListTileTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/list_tile_theme.dart ListTileTheme.data
func (w *ListTileTheme) Data() ListTileThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/list_tile_theme.dart ListTileTheme.updateShouldNotify
func (w *ListTileTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*ListTileTheme).data)
}

// Wrap makes a theme with the data that a list tile reads, around the child.
//
// Flutter: material/list_tile_theme.dart ListTileTheme.wrap
func (w *ListTileTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	d := w.data
	return NewListTileTheme(ListTileThemeData{
		Dense: d.Dense, Shape: d.Shape, Style: d.Style, SelectedColor: d.SelectedColor, IconColor: d.IconColor, TextColor: d.TextColor,
		ContentPadding: d.ContentPadding, TileColor: d.TileColor, SelectedTileColor: d.SelectedTileColor, EnableFeedback: d.EnableFeedback,
		HorizontalTitleGap: d.HorizontalTitleGap, MinVerticalPadding: d.MinVerticalPadding, MinLeadingWidth: d.MinLeadingWidth,
		IsThreeLine: d.IsThreeLine,
	}, child)
}

// ListTileThemeOf returns the data of the closest ListTileTheme above the
// context, or the ListTileTheme of the closest Theme. The context depends on it.
//
// Flutter: material/list_tile_theme.dart ListTileTheme.of
func ListTileThemeOf(bc widgets.BuildContext) ListTileThemeData {
	if t, ok := widgets.DependOn[*ListTileTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).ListTileTheme
}

// ListTileThemeMerge is a ListTileTheme whose data is the one of the closest
// theme, with the fields that are set in the overrides replaced.
//
// Flutter: material/list_tile_theme.dart ListTileTheme.merge
func ListTileThemeMerge(overrides ListTileThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) widgets.Widget {
	return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		p := ListTileThemeOf(bc)
		o := overrides
		return NewListTileTheme(ListTileThemeData{
			Dense: firstOpt(o.Dense, p.Dense), Shape: firstNonNil(o.Shape, p.Shape), Style: firstNonZero(o.Style, p.Style),
			SelectedColor: firstOpt(o.SelectedColor, p.SelectedColor), IconColor: firstOpt(o.IconColor, p.IconColor),
			TextColor: firstOpt(o.TextColor, p.TextColor), TitleTextStyle: firstNonNil(o.TitleTextStyle, p.TitleTextStyle),
			SubtitleTextStyle:           firstNonNil(o.SubtitleTextStyle, p.SubtitleTextStyle),
			LeadingAndTrailingTextStyle: firstNonNil(o.LeadingAndTrailingTextStyle, p.LeadingAndTrailingTextStyle),
			ContentPadding:              firstNonNil(o.ContentPadding, p.ContentPadding), TileColor: firstOpt(o.TileColor, p.TileColor),
			SelectedTileColor: firstOpt(o.SelectedTileColor, p.SelectedTileColor), EnableFeedback: firstOpt(o.EnableFeedback, p.EnableFeedback),
			HorizontalTitleGap: firstOpt(o.HorizontalTitleGap, p.HorizontalTitleGap), MinVerticalPadding: firstOpt(o.MinVerticalPadding, p.MinVerticalPadding),
			MinLeadingWidth: firstOpt(o.MinLeadingWidth, p.MinLeadingWidth), MinTileHeight: firstOpt(o.MinTileHeight, p.MinTileHeight),
			TitleAlignment: firstNonZero(o.TitleAlignment, p.TitleAlignment), MouseCursor: firstNonNil(o.MouseCursor, p.MouseCursor),
			VisualDensity: firstOpt(o.VisualDensity, p.VisualDensity), ControlAffinity: firstNonZero(o.ControlAffinity, p.ControlAffinity),
			IsThreeLine: firstOpt(o.IsThreeLine, p.IsThreeLine),
		}, child, opts...)
	})
}
