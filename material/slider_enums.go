// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/widgets"
	"reflect"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// ShowValueIndicator says when the value indicator of a Slider or a RangeSlider
// shows.
//
// Flutter: material/slider_theme.dart ShowValueIndicator
type ShowValueIndicator int

const (
	// ShowValueIndicatorUnspecified is the zero value: the default, which is
	// onlyForDiscrete.
	ShowValueIndicatorUnspecified ShowValueIndicator = iota
	// ShowValueIndicatorOnlyForDiscrete shows the indicator while the thumb is
	// pressed, on sliders with divisions only.
	//
	// Flutter: material/slider_theme.dart ShowValueIndicator.onlyForDiscrete
	ShowValueIndicatorOnlyForDiscrete
	// ShowValueIndicatorOnlyForContinuous shows the indicator while the thumb is
	// pressed, on sliders without divisions only.
	//
	// Flutter: material/slider_theme.dart ShowValueIndicator.onlyForContinuous
	ShowValueIndicatorOnlyForContinuous
	// ShowValueIndicatorAlways shows the indicator while the thumb is pressed.
	//
	// Deprecated: Flutter deprecates it after v3.28; use ShowValueIndicatorOnDrag.
	//
	// Flutter: material/slider_theme.dart ShowValueIndicator.always
	ShowValueIndicatorAlways
	// ShowValueIndicatorOnDrag shows the indicator while the thumb is dragged.
	//
	// Flutter: material/slider_theme.dart ShowValueIndicator.onDrag
	ShowValueIndicatorOnDrag
	// ShowValueIndicatorAlwaysVisible shows the indicator always, and not only
	// while the thumb is dragged.
	//
	// Flutter: material/slider_theme.dart ShowValueIndicator.alwaysVisible
	ShowValueIndicatorAlwaysVisible
	// ShowValueIndicatorNever never shows the indicator.
	//
	// Flutter: material/slider_theme.dart ShowValueIndicator.never
	ShowValueIndicatorNever
)

// Name is the Dart name of the value.
func (s ShowValueIndicator) Name() string {
	return enumName(int(s), "onlyForDiscrete", "onlyForContinuous", "always", "onDrag", "alwaysVisible", "never")
}

// String is "ShowValueIndicator." and the name.
func (s ShowValueIndicator) String() string { return "ShowValueIndicator." + s.Name() }

// Thumb is one of the two thumbs of a RangeSlider.
//
// Flutter: material/slider_theme.dart Thumb
type Thumb int

const (
	// ThumbUnspecified is the zero value: no thumb.
	ThumbUnspecified Thumb = iota
	// ThumbStart is the thumb for the start of the range.
	//
	// Flutter: material/slider_theme.dart Thumb.start
	ThumbStart
	// ThumbEnd is the thumb for the end of the range.
	//
	// Flutter: material/slider_theme.dart Thumb.end
	ThumbEnd
)

// Name is the Dart name of the value.
func (t Thumb) Name() string { return enumName(int(t), "start", "end") }

// String is "Thumb." and the name.
func (t Thumb) String() string { return "Thumb." + t.Name() }

// SemanticFormatterCallback formats the value of a slider for the semantics.
//
// Flutter: material/slider_theme.dart SemanticFormatterCallback
type SemanticFormatterCallback func(value float64) string

// sameFunc says whether two functions are the same one (Dart compares functions by
// identity).
func sameFunc[T any](a, b T) bool {
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if va.IsNil() || vb.IsNil() {
		return va.IsNil() && vb.IsNil()
	}
	return va.Pointer() == vb.Pointer()
}

// funcOrNil is the function for the diagnostics, or nil when it is nil.
func funcOrNil[T any](f T) any {
	if reflect.ValueOf(f).IsNil() {
		return nil
	}
	return f
}

// sliderOverlayColorOf is the overlay color of the theme for the states: the
// property when there is one, the color otherwise.
func (d SliderThemeData) overlayColorFor(states widgets.WidgetStates) foundation.Opt[ui.Color] {
	if d.StateOverlayColor != nil {
		return d.StateOverlayColor.Resolve(states)
	}
	return d.OverlayColor
}

// lerpSliderOverlayColor interpolates the overlay colors of two themes: a color
// that depends on the states counts as the color it has for no state.
//
// Flutter: material/slider_theme.dart SliderThemeData.lerp
func lerpSliderOverlayColor(a, b SliderThemeData, t float64) foundation.Opt[ui.Color] {
	return lerpOptColor(a.overlayColorFor(0), b.overlayColorFor(0), t)
}

// NewSliderThemeDataFromPrimaryColors creates the theme from the primary colors of
// a theme, with the paddle value indicator and the round shapes.
//
// Flutter: material/slider_theme.dart SliderThemeData.fromPrimaryColors
func NewSliderThemeDataFromPrimaryColors(primaryColor, primaryColorDark, primaryColorLight ui.Color, valueIndicatorTextStyle *painting.TextStyle) SliderThemeData {
	const (
		activeTrackAlpha                  = 0xff
		inactiveTrackAlpha                = 0x3d // 24% opacity
		secondaryActiveTrackAlpha         = 0x8a // 54% opacity
		disabledActiveTrackAlpha          = 0x52 // 32% opacity
		disabledInactiveTrackAlpha        = 0x1f // 12% opacity
		disabledSecondaryActiveTrackAlpha = 0x1f // 12% opacity
		activeTickMarkAlpha               = 0x8a // 54% opacity
		inactiveTickMarkAlpha             = 0x8a // 54% opacity
		disabledActiveTickMarkAlpha       = 0x1f // 12% opacity
		disabledInactiveTickMarkAlpha     = 0x1f // 12% opacity
		thumbAlpha                        = 0xff
		disabledThumbAlpha                = 0x52 // 32% opacity
		overlayAlpha                      = 0x1f // 12% opacity
		valueIndicatorAlpha               = 0xff
	)
	some := foundation.Some[ui.Color]
	return NewSliderThemeData(SliderThemeDataOpts{
		TrackHeight:                       foundation.Some(2.0),
		ActiveTrackColor:                  some(primaryColor.WithAlpha(activeTrackAlpha)),
		InactiveTrackColor:                some(primaryColor.WithAlpha(inactiveTrackAlpha)),
		SecondaryActiveTrackColor:         some(primaryColor.WithAlpha(secondaryActiveTrackAlpha)),
		DisabledActiveTrackColor:          some(primaryColorDark.WithAlpha(disabledActiveTrackAlpha)),
		DisabledInactiveTrackColor:        some(primaryColorDark.WithAlpha(disabledInactiveTrackAlpha)),
		DisabledSecondaryActiveTrackColor: some(primaryColorDark.WithAlpha(disabledSecondaryActiveTrackAlpha)),
		ActiveTickMarkColor:               some(primaryColorLight.WithAlpha(activeTickMarkAlpha)),
		InactiveTickMarkColor:             some(primaryColor.WithAlpha(inactiveTickMarkAlpha)),
		DisabledActiveTickMarkColor:       some(primaryColorLight.WithAlpha(disabledActiveTickMarkAlpha)),
		DisabledInactiveTickMarkColor:     some(primaryColorDark.WithAlpha(disabledInactiveTickMarkAlpha)),
		ThumbColor:                        some(primaryColor.WithAlpha(thumbAlpha)),
		OverlappingShapeStrokeColor:       some(Colors.White),
		DisabledThumbColor:                some(primaryColorDark.WithAlpha(disabledThumbAlpha)),
		OverlayColor:                      some(primaryColor.WithAlpha(overlayAlpha)),
		ValueIndicatorColor:               some(primaryColor.WithAlpha(valueIndicatorAlpha)),
		ValueIndicatorStrokeColor:         some(primaryColor.WithAlpha(valueIndicatorAlpha)),
		OverlayShape:                      RoundSliderOverlayShape{},
		TickMarkShape:                     RoundSliderTickMarkShape{},
		ThumbShape:                        RoundSliderThumbShape{},
		TrackShape:                        RoundedRectSliderTrackShape{},
		ValueIndicatorShape:               PaddleSliderValueIndicatorShape{},
		RangeTickMarkShape:                RoundRangeSliderTickMarkShape{},
		RangeThumbShape:                   RoundRangeSliderThumbShape{},
		RangeTrackShape:                   RoundedRectRangeSliderTrackShape{},
		RangeValueIndicatorShape:          PaddleRangeSliderValueIndicatorShape{},
		ValueIndicatorTextStyle:           valueIndicatorTextStyle,
		ShowValueIndicator:                ShowValueIndicatorOnlyForDiscrete,
	})
}

// SliderInteraction is the interaction that changes the value of a Slider.
//
// Flutter: material/slider.dart SliderInteraction
type SliderInteraction int

const (
	// SliderInteractionUnspecified is the zero value: the default, which is
	// tapAndSlide.
	SliderInteractionUnspecified SliderInteraction = iota
	// SliderInteractionTapAndSlide allows the user to tap anywhere on the track to
	// move the thumb and to drag it.
	//
	// Flutter: material/slider.dart SliderInteraction.tapAndSlide
	SliderInteractionTapAndSlide
	// SliderInteractionTapOnly allows the user to tap anywhere on the track to move
	// the thumb, but not to drag it.
	//
	// Flutter: material/slider.dart SliderInteraction.tapOnly
	SliderInteractionTapOnly
	// SliderInteractionSlideOnly allows the user to drag the thumb, but not to tap
	// on the track.
	//
	// Flutter: material/slider.dart SliderInteraction.slideOnly
	SliderInteractionSlideOnly
	// SliderInteractionSlideThumb allows the user to drag the thumb only when the
	// press starts on its overlay.
	//
	// Flutter: material/slider.dart SliderInteraction.slideThumb
	SliderInteractionSlideThumb
)

// Name is the Dart name of the value.
func (s SliderInteraction) Name() string {
	return enumName(int(s), "tapAndSlide", "tapOnly", "slideOnly", "slideThumb")
}

// String is "SliderInteraction." and the name.
func (s SliderInteraction) String() string { return "SliderInteraction." + s.Name() }
