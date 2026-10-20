// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The functions in this file stand in for Divider and Tooltip, which come with
// step 13.9. Each use is marked, and the real widgets replace them when they
// arrive.

// tempHairlineDivider stands in for Divider(height: 0, color: color): a hairline
// along the bottom of a zero height box.
//
// TODO(step 13.9): Divider.
func tempHairlineDivider(color foundation.Opt[ui.Color]) widgets.Widget {
	side := painting.NewBorderSide(painting.BorderSideOpts{Color: color, Width: foundation.Some(0.0)})
	return widgets.NewSizedBox(
		widgets.NewCenter(widgets.NewContainer(nil, widgets.ContainerOpts{
			Height:     foundation.Some(0.0),
			Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{Border: painting.Border{Bottom: side}}),
		})),
		widgets.SizedBoxOpts{Height: foundation.Some(0.0)})
}

// tempTooltip stands in for a Tooltip: it only gives the child the tooltip
// semantics.
//
// TODO(step 13.9): Tooltip.
func tempTooltip(message string, child widgets.Widget) widgets.Widget {
	return widgets.NewSemantics(child, widgets.SemanticsOpts{Tooltip: foundation.Some(message)})
}

// tempScrollbar stands in for the Material Scrollbar that wraps the scrollable
// of a menu: a RawScrollbar with the thickness, the radius, the margin, the
// fading and the colors of the Material 3 scrollbar at rest, without the
// colors for the states of hover and drag.
//
// TODO(step 13.10): Scrollbar.
func tempScrollbar(bc widgets.BuildContext, thumbVisibility bool, child widgets.Widget) widgets.Widget {
	theme := ThemeOf(bc)
	thumb := theme.ColorScheme.OnSurface().WithOpacity(0.1)
	if theme.Brightness() == ui.BrightnessDark {
		thumb = theme.HighlightColor.WithOpacity(1.0)
	}
	return widgets.NewRawScrollbar(child, widgets.RawScrollbarOpts{
		ThumbVisibility: foundation.Some(thumbVisibility),
		Thickness:       foundation.Some(8.0),
		Radius:          foundation.Some(ui.NewRadiusCircular(8.0)),
		CrossAxisMargin: 2.0,
		MinThumbLength:  foundation.Some(48.0),
		FadeDuration:    foundation.Some(300 * time.Millisecond),
		TimeToFade:      foundation.Some(600 * time.Millisecond),
		ThumbColor:      foundation.Some(thumb),
	})
}

// tempDividerOpts are the parameters of tempDivider.
type tempDividerOpts struct {
	height                       float64
	thickness, indent, endIndent foundation.Opt[float64]
	radius                       painting.BorderRadiusGeometry
	color                        foundation.Opt[ui.Color]
}

// tempDivider stands in for Divider: a line in the middle of a box, with the
// Material 3 colors and thickness of the theme.
//
// TODO(step 13.9): Divider.
func tempDivider(bc widgets.BuildContext, o tempDividerOpts) widgets.Widget {
	theme := ThemeOf(bc)
	thickness := o.thickness.OrElse(0.0)
	color := theme.DividerColor
	if theme.UseMaterial3 {
		thickness = o.thickness.OrElse(1.0)
		color = theme.ColorScheme.OutlineVariant()
	}
	if c, ok := o.color.Get(); ok {
		color = c
	}
	return widgets.NewSizedBox(
		widgets.NewCenter(widgets.NewContainer(nil, widgets.ContainerOpts{
			Height: foundation.Some(thickness),
			Margin: painting.EdgeInsetsDirectional{Start: o.indent.OrElse(0), End: o.endIndent.OrElse(0)},
			Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
				BorderRadius: o.radius,
				Border:       painting.Border{Bottom: painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(color), Width: foundation.Some(thickness)})},
			}),
		})),
		widgets.SizedBoxOpts{Height: foundation.Some(o.height)})
}

// tempListTileOpts are the parameters of tempListTile.
type tempListTileOpts struct {
	enabled    bool
	titleStyle *painting.TextStyle
	leading    widgets.Widget
	title      widgets.Widget
}

// tempListTile stands in for a ListTile with no content padding, a leading
// widget and a title: the leading widget in a box 40 wide, 16 of space, and the
// title, at least 56 high.
//
// TODO(step 13.9): ListTile.
func tempListTile(_ widgets.BuildContext, o tempListTileOpts) widgets.Widget {
	title := widgets.Widget(o.title)
	if title != nil && o.titleStyle != nil {
		title = widgets.NewDefaultTextStyleMerge(title, widgets.DefaultTextStyleMergeOpts{Style: o.titleStyle})
	}
	return widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinHeight: foundation.Some(56.0)}),
		widgets.NewRow([]widgets.Widget{
			widgets.NewSizedBox(widgets.NewAlign(o.leading, widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart}), widgets.SizedBoxOpts{Width: foundation.Some(40.0)}),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(16.0)}),
			widgets.NewExpanded(widgets.NewAlign(title, widgets.AlignOpts{Alignment: painting.AlignmentDirectionalCenterStart})),
		}, widgets.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentCenter}))
}
