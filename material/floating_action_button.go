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

// FloatingActionButtonOpts are the optional parameters of
// NewFloatingActionButton.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.new
type FloatingActionButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Tooltip is the text that describes the action of the button; it is the
	// semantics tooltip of the button.
	//
	// TODO(step 13.3): show it in a Tooltip.
	Tooltip string
	// ForegroundColor is the default icon and text color; the default is the
	// on-primary-container color of the color scheme.
	ForegroundColor foundation.Opt[ui.Color]
	// BackgroundColor is the color of the button; the default is the
	// primary-container color of the color scheme.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the button at rest; the default is 6.
	Elevation foundation.Opt[float64]
	// Shape is the shape of the button; the default is a rounded rectangle with
	// a radius of 16.
	Shape painting.ShapeBorder
}

// FloatingActionButton is a circular icon button that hovers over content to
// promote a primary action in the application. This is the minimum of the
// regular Material 3 button that the counter example needs.
//
// TODO(step 13.3): the small, large and extended types, the theme, the
// elevations of the focus, hover, highlight and disabled states, the hero tag,
// the tooltip, the focus node and the Material 2 defaults.
//
// Flutter: material/floating_action_button.dart FloatingActionButton
type FloatingActionButton struct {
	widgets.StatelessWidget
	onPressed foundation.VoidCallback
	child     widgets.Widget
	o         FloatingActionButtonOpts
}

// NewFloatingActionButton creates a regular floating action button. A nil
// onPressed disables the button. The child is usually an Icon.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.new
func NewFloatingActionButton(onPressed foundation.VoidCallback, child widgets.Widget, opts ...FloatingActionButtonOpts) *FloatingActionButton {
	o := firstOpts(opts)
	return &FloatingActionButton{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed: onPressed, child: child, o: o}
}

// OnPressed is the callback that is called when the button is tapped, or nil
// when the button is disabled.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.onPressed
func (w *FloatingActionButton) OnPressed() foundation.VoidCallback { return w.onPressed }

// Child is the widget below this widget in the tree.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.child
func (w *FloatingActionButton) Child() widgets.Widget { return w.child }

// Tooltip is the text that describes the action of the button.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.tooltip
func (w *FloatingActionButton) Tooltip() string { return w.o.Tooltip }

// Build builds the button as Flutter's RawMaterialButton does: a constrained
// Material with an InkWell.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.build
func (w *FloatingActionButton) Build(bc widgets.BuildContext) widgets.Widget {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	foregroundColor := w.o.ForegroundColor.OrElse(colors.OnPrimaryContainer())
	backgroundColor := w.o.BackgroundColor.OrElse(colors.PrimaryContainer())
	splashColor := colors.OnPrimaryContainer().WithOpacity(0.1)
	hoverColor := colors.OnPrimaryContainer().WithOpacity(0.08)
	focusColor := colors.OnPrimaryContainer().WithOpacity(0.1)
	elevation := w.o.Elevation.OrElse(6.0)
	var shape painting.ShapeBorder = painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusCircular(16)})
	if w.o.Shape != nil {
		shape = w.o.Shape
	}
	const iconSize = 24.0
	var child widgets.Widget
	if w.child != nil {
		child = widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(iconSize)}), w.child)
	}

	var onTap func()
	if w.onPressed != nil {
		onTap = func() { w.onPressed() }
	}
	var result widgets.Widget = widgets.NewConstrainedBox(
		rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(56.0), Height: foundation.Some(56.0)}),
		NewMaterial(
			NewInkWell(
				widgets.NewIconThemeMerge(
					widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(foregroundColor)}),
					widgets.NewCenter(child, widgets.CenterOpts{WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)}),
				),
				InkWellOpts{
					OnTap:          onTap,
					CustomBorder:   shape,
					SplashColor:    foundation.Some(splashColor),
					HoverColor:     foundation.Some(hoverColor),
					FocusColor:     foundation.Some(focusColor),
					EnableFeedback: foundation.Some(true),
				},
			),
			MaterialOpts{
				Elevation: elevation,
				Color:     foundation.Some(backgroundColor),
				Shape:     shape,
				TextStyle: theme.TextTheme.LabelLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(foregroundColor)}),
			},
		),
	)
	if w.o.Tooltip != "" {
		result = widgets.NewSemantics(result, widgets.SemanticsOpts{Tooltip: foundation.Some(w.o.Tooltip)})
	}
	return widgets.NewMergeSemantics(result)
}
