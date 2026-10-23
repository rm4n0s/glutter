// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CircleAvatarOpts are the optional parameters of NewCircleAvatar.
//
// Flutter: material/circle_avatar.dart CircleAvatar.new
type CircleAvatarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Child is the widget below this one in the tree, typically a Text with the
	// initials of the person.
	Child widgets.Widget
	// BackgroundColor is the color with which to fill the circle.
	BackgroundColor foundation.Opt[ui.Color]
	// BackgroundImage is the image of the circle, under the child.
	BackgroundImage painting.ImageProvider
	// ForegroundImage is the image of the circle, over the child.
	ForegroundImage painting.ImageProvider
	// OnBackgroundImageError is called when BackgroundImage fails to load.
	OnBackgroundImageError painting.ImageErrorListener
	// OnForegroundImageError is called when ForegroundImage fails to load.
	OnForegroundImageError painting.ImageErrorListener
	// ForegroundColor is the default color for text and icons in the child.
	ForegroundColor foundation.Opt[ui.Color]
	// Radius is the size of the avatar, as the radius of the circle; it cannot be
	// given with MinRadius or MaxRadius.
	Radius foundation.Opt[float64]
	// MinRadius is the minimum size of the avatar.
	MinRadius foundation.Opt[float64]
	// MaxRadius is the maximum size of the avatar.
	MaxRadius foundation.Opt[float64]
}

// CircleAvatar is a circle that represents a user.
//
// Flutter: material/circle_avatar.dart CircleAvatar
type CircleAvatar struct {
	widgets.StatelessWidget
	o CircleAvatarOpts
}

const (
	circleAvatarDefaultRadius    = 20.0
	circleAvatarDefaultMinRadius = 0.0
)

// NewCircleAvatar creates an avatar. In debug builds it panics with the tag
// CircleAvatarRadiusConflict when Radius is given together with MinRadius or
// MaxRadius, and with CircleAvatarImageErrorWithoutImage when an error callback
// has no image.
//
// Flutter: material/circle_avatar.dart CircleAvatar.new
func NewCircleAvatar(opts ...CircleAvatarOpts) *CircleAvatar {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if o.Radius.IsSome() && (o.MinRadius.IsSome() || o.MaxRadius.IsSome()) {
			panic(errors.New(tagCircleAvatarRadiusConflict, "the radius of a CircleAvatar cannot be given with its minRadius or maxRadius"))
		}
		if (o.BackgroundImage == nil && o.OnBackgroundImageError != nil) || (o.ForegroundImage == nil && o.OnForegroundImageError != nil) {
			panic(errors.New(tagCircleAvatarImageErrorWithoutImage, "an image error callback of a CircleAvatar needs its image"))
		}
	}
	return &CircleAvatar{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Child is the widget below this one in the tree.
//
// Flutter: material/circle_avatar.dart CircleAvatar.child
func (w *CircleAvatar) Child() widgets.Widget { return w.o.Child }

func (w *CircleAvatar) noRadius() bool {
	return w.o.Radius.IsNone() && w.o.MinRadius.IsNone() && w.o.MaxRadius.IsNone()
}

func (w *CircleAvatar) minDiameter() float64 {
	if w.noRadius() {
		return circleAvatarDefaultRadius * 2.0
	}
	if r, ok := w.o.Radius.Get(); ok {
		return 2.0 * r
	}
	return 2.0 * w.o.MinRadius.OrElse(circleAvatarDefaultMinRadius)
}

func (w *CircleAvatar) maxDiameter() float64 {
	if w.noRadius() {
		return circleAvatarDefaultRadius * 2.0
	}
	if r, ok := w.o.Radius.Get(); ok {
		return 2.0 * r
	}
	return 2.0 * w.o.MaxRadius.OrElse(math.Inf(1))
}

// Build builds the avatar.
//
// Flutter: material/circle_avatar.dart CircleAvatar.build
func (w *CircleAvatar) Build(bc widgets.BuildContext) widgets.Widget {
	widgets.DebugCheckHasMediaQuery(bc)
	theme := ThemeOf(bc)
	var effectiveForeground foundation.Opt[ui.Color]
	if c, ok := w.o.ForegroundColor.Get(); ok {
		effectiveForeground = foundation.Some(c)
	} else if theme.UseMaterial3 {
		effectiveForeground = foundation.Some(theme.ColorScheme.OnPrimaryContainer())
	}
	effectiveTextStyle := theme.PrimaryTextTheme.TitleMedium
	if theme.UseMaterial3 {
		effectiveTextStyle = theme.TextTheme.TitleMedium
	}
	textStyle := effectiveTextStyle.CopyWith(painting.TextStyleCopy{Color: effectiveForeground})
	var effectiveBackground ui.Color
	if c, ok := w.o.BackgroundColor.Get(); ok {
		effectiveBackground = c
	} else if theme.UseMaterial3 {
		effectiveBackground = theme.ColorScheme.PrimaryContainer()
	} else if EstimateBrightnessForColor(textStyle.Color.Value()) == ui.BrightnessDark {
		effectiveBackground = theme.PrimaryColorLight
	} else {
		effectiveBackground = theme.PrimaryColorDark
	}
	if w.o.BackgroundColor.IsSome() || theme.UseMaterial3 {
		if effectiveForeground.IsNone() {
			light := theme.PrimaryColorDark
			if EstimateBrightnessForColor(w.o.BackgroundColor.Value()) == ui.BrightnessDark {
				light = theme.PrimaryColorLight
			}
			textStyle = textStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(light)})
		}
	}

	decoration := painting.BoxDecorationOpts{Color: foundation.Some(effectiveBackground), Shape: painting.BoxShapeCircle}
	if w.o.BackgroundImage != nil {
		decoration.Image = painting.NewDecorationImage(w.o.BackgroundImage,
			painting.DecorationImageOpts{OnError: w.o.OnBackgroundImageError, Fit: painting.BoxFitCover})
	}
	var foreground painting.Decoration
	if w.o.ForegroundImage != nil {
		foreground = painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Image: painting.NewDecorationImage(w.o.ForegroundImage,
				painting.DecorationImageOpts{OnError: w.o.OnForegroundImageError, Fit: painting.BoxFitCover}),
			Shape: painting.BoxShapeCircle,
		})
	}
	var child widgets.Widget
	if w.o.Child != nil {
		child = widgets.NewCenter(widgets.NewMediaQueryWithNoTextScaling(
			widgets.NewIconTheme(theme.IconTheme.CopyWith(widgets.IconThemeDataOpts{Color: textStyle.Color}),
				widgets.NewDefaultTextStyle(textStyle, w.o.Child))))
	}
	minD, maxD := w.minDiameter(), w.maxDiameter()
	return widgets.NewAnimatedContainer(ThemeChangeDuration, child, widgets.AnimatedContainerOpts{
		Constraints: foundation.Some(rendering.BoxConstraints{MinWidth: minD, MaxWidth: maxD, MinHeight: minD, MaxHeight: maxD}),
		Decoration:  painting.NewBoxDecoration(decoration), ForegroundDecoration: foreground,
	})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCircleAvatarImageErrorWithoutImage = "CircleAvatarImageErrorWithoutImage"
	tagCircleAvatarRadiusConflict         = "CircleAvatarRadiusConflict"
)
