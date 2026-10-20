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

// The widgets in this file stand in for TextButton, IconButton and Divider, which
// come with steps 13.3 and 13.9. They have the Material 3 look of the real ones
// at rest, but none of their states, themes or options. Each use is marked, and
// the real widgets replace them when they arrive.

// tempTextButton stands in for a TextButton with a stadium shape.
//
// TODO(step 13.3): TextButton.
type tempTextButton struct {
	widgets.StatelessWidget
	label           string
	onPressed       func()
	foreground      widgets.WidgetStateProperty[ui.Color]
	background      widgets.WidgetStateProperty[ui.Color]
	horizontalInset float64
}

func (w *tempTextButton) Build(bc widgets.BuildContext) widgets.Widget {
	theme := ThemeOf(bc)
	states := widgets.NewWidgetStates()
	if w.onPressed == nil {
		states = widgets.NewWidgetStates(widgets.WidgetStateDisabled)
	}
	fg := theme.ColorScheme.Primary()
	if w.foreground != nil {
		fg = w.foreground.Resolve(states)
	}
	bg := foundation.None[ui.Color]()
	if w.background != nil {
		if c := w.background.Resolve(states); c.A != 0 {
			bg = foundation.Some(c)
		}
	}
	shape := painting.NewStadiumBorder()
	var onTap func()
	if w.onPressed != nil {
		onTap = w.onPressed
	}
	materialType := MaterialTypeButton
	if bg.IsNone() {
		materialType = MaterialTypeTransparency
	}
	return NewMaterial(
		NewInkWell(
			widgets.NewConstrainedBox(
				rendering.BoxConstraints{MinWidth: 64, MaxWidth: 1e9, MinHeight: 40, MaxHeight: 1e9},
				widgets.NewPadding(painting.NewEdgeInsetsSymmetric(w.horizontalInset, 0),
					widgets.NewCenter(widgets.NewText(w.label, widgets.TextOpts{Style: theme.TextTheme.LabelLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(fg)})}),
						widgets.CenterOpts{WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)}))),
			InkWellOpts{
				OnTap:          onTap,
				CustomBorder:   shape,
				HoverColor:     foundation.Some(fg.WithOpacity(0.08)),
				FocusColor:     foundation.Some(fg.WithOpacity(0.1)),
				HighlightColor: foundation.Some(fg.WithOpacity(0.1)),
				SplashColor:    foundation.Some(fg.WithOpacity(0.1)),
			}),
		MaterialOpts{Type: materialType, Color: bg, Shape: shape, ShadowColor: foundation.Some(Colors.Transparent), SurfaceTintColor: foundation.Some(Colors.Transparent)},
	)
}

// tempIconButton stands in for a standard IconButton: a 48 by 48 round target
// around the icon.
//
// TODO(step 13.3): IconButton.
type tempIconButton struct {
	widgets.StatelessWidget
	// child, when it is not nil, is the content instead of the icon: it takes the
	// color of the icon theme of its context.
	child     widgets.Widget
	icon      widgets.IconData
	size      float64
	color     foundation.Opt[ui.Color]
	onPressed func()
	tooltip   string
}

func (w *tempIconButton) Build(bc widgets.BuildContext) widgets.Widget {
	color := w.color.OrElse(ThemeOf(bc).ColorScheme.OnSurfaceVariant())
	content := w.child
	if content == nil {
		content = widgets.NewIcon(w.icon, widgets.IconOpts{Size: foundation.Some(w.size), Color: foundation.Some(color)})
	}
	return widgets.NewSemantics(
		NewMaterial(
			NewInkWell(
				widgets.NewSizedBox(widgets.NewCenter(content),
					widgets.SizedBoxOpts{Width: foundation.Some(48.0), Height: foundation.Some(48.0)}),
				InkWellOpts{OnTap: w.onPressed, CustomBorder: painting.NewCircleBorder(), HoverColor: foundation.Some(color.WithOpacity(0.08)),
					FocusColor: foundation.Some(color.WithOpacity(0.1)), HighlightColor: foundation.Some(color.WithOpacity(0.1)), SplashColor: foundation.Some(color.WithOpacity(0.1))}),
			MaterialOpts{Type: MaterialTypeTransparency}),
		widgets.SemanticsOpts{Button: foundation.Some(true), Tooltip: foundation.Some(w.tooltip)},
	)
}

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
