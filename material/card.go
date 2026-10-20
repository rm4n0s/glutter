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

// cardVariant is the look of a card.
//
// Flutter: material/card.dart _CardVariant
type cardVariant int

const (
	cardVariantElevated cardVariant = iota
	cardVariantFilled
	cardVariantOutlined
)

// CardOpts are the optional parameters of NewCard, NewCardFilled and
// NewCardOutlined.
//
// Flutter: material/card.dart Card.new
type CardOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Color is the card's background color.
	Color foundation.Opt[ui.Color]
	// ShadowColor is the color of the card's shadow.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the color of the surface tint overlay (Material 3).
	SurfaceTintColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the card.
	Elevation foundation.Opt[float64]
	// Shape is the shape of the card's Material.
	Shape painting.ShapeBorder
	// NoBorderOnForeground paints the border behind the child: Flutter's
	// borderOnForeground is false.
	NoBorderOnForeground bool
	// Margin is the empty space around the card.
	Margin painting.EdgeInsetsGeometry
	// ClipBehavior is how the card's content is clipped.
	ClipBehavior ui.Clip
	// NoSemanticContainer makes the card's children separate nodes in the
	// semantics tree: Flutter's semanticContainer is false.
	NoSemanticContainer bool
}

// Card is a Material Design card: a panel with slightly rounded corners and an
// elevation shadow.
//
// Flutter: material/card.dart Card
type Card struct {
	widgets.StatelessWidget
	child   widgets.Widget
	o       CardOpts
	variant cardVariant
}

func newCard(child widgets.Widget, v cardVariant, opts []CardOpts) *Card {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Elevation.OrElse(0) < 0 {
		panic(errors.New("CardElevationNegative", "a card's elevation must not be negative", "elevation", o.Elevation.Value()))
	}
	return &Card{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child, o: o, variant: v}
}

// NewCard creates an elevated card. The tag of its failure is
// CardElevationNegative.
//
// Flutter: material/card.dart Card.new
func NewCard(child widgets.Widget, opts ...CardOpts) *Card {
	return newCard(child, cardVariantElevated, opts)
}

// NewCardFilled creates a filled card, with a filled surface and no elevation.
//
// Flutter: material/card.dart Card.filled
func NewCardFilled(child widgets.Widget, opts ...CardOpts) *Card {
	return newCard(child, cardVariantFilled, opts)
}

// NewCardOutlined creates an outlined card, with an outline and no elevation.
//
// Flutter: material/card.dart Card.outlined
func NewCardOutlined(child widgets.Widget, opts ...CardOpts) *Card {
	return newCard(child, cardVariantOutlined, opts)
}

// cardDefaults are the defaults of a card: the values that its theme does not
// give.
//
// Flutter: material/card.dart _CardDefaultsM2, _CardDefaultsM3, _FilledCardDefaultsM3, _OutlinedCardDefaultsM3
type cardDefaults struct {
	color            ui.Color
	shadowColor      ui.Color
	surfaceTintColor foundation.Opt[ui.Color]
	elevation        float64
	shape            painting.ShapeBorder
}

func cardDefaultsOf(theme *ThemeData, v cardVariant) cardDefaults {
	if !theme.UseMaterial3 {
		return cardDefaults{
			color: theme.CardColor, shadowColor: theme.ShadowColor, elevation: 1,
			shape: painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(4))}),
		}
	}
	colors := theme.ColorScheme
	shape := painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(12))}
	d := cardDefaults{shadowColor: colors.Shadow(), surfaceTintColor: foundation.Some(Colors.Transparent)}
	switch v {
	case cardVariantFilled:
		d.color = colors.SurfaceContainerHighest()
	case cardVariantOutlined:
		d.color = colors.Surface()
		shape.Side = painting.NewBorderSide(painting.BorderSideOpts{Color: foundation.Some(colors.OutlineVariant())})
	default:
		d.color, d.elevation = colors.SurfaceContainerLow(), 1
	}
	d.shape = painting.NewRoundedRectangleBorder(shape)
	return d
}

// Build builds the card.
//
// Flutter: material/card.dart Card.build
func (w *Card) Build(bc widgets.BuildContext) widgets.Widget {
	o := w.o
	cardTheme := CardThemeOf(bc)
	defaults := cardDefaultsOf(ThemeOf(bc), w.variant)
	margin := firstNonNil(o.Margin, cardTheme.Margin, painting.EdgeInsetsGeometry(painting.NewEdgeInsetsAll(4)))
	return widgets.NewSemantics(widgets.NewPadding(margin, NewMaterial(
		widgets.NewSemantics(w.child, widgets.SemanticsOpts{ExplicitChildNodes: o.NoSemanticContainer}),
		MaterialOpts{
			Type:                 MaterialTypeCard,
			Color:                firstOpt(o.Color, cardTheme.Color, foundation.Some(defaults.color)),
			ShadowColor:          firstOpt(o.ShadowColor, cardTheme.ShadowColor, foundation.Some(defaults.shadowColor)),
			SurfaceTintColor:     firstOpt(o.SurfaceTintColor, cardTheme.SurfaceTintColor, defaults.surfaceTintColor),
			Elevation:            firstOpt(o.Elevation, cardTheme.Elevation, foundation.Some(defaults.elevation)).Value(),
			Shape:                firstNonNil(o.Shape, cardTheme.Shape, defaults.shape),
			NoBorderOnForeground: o.NoBorderOnForeground,
			ClipBehavior:         firstNonZero(o.ClipBehavior, cardTheme.ClipBehavior, ui.ClipNone),
		})), widgets.SemanticsOpts{Container: !o.NoSemanticContainer})
}
