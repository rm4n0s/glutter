// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package ui

import (
	"strconv"
	"strings"
)

// Brightness describes the contrast of a theme or color palette. The zero
// value, BrightnessUnspecified, means the platform does not say.
//
// Flutter: ui/platform_dispatcher.dart Brightness
type Brightness int

// The values of Brightness follow Flutter's declaration order.
const (
	// BrightnessUnspecified is the zero value.
	//
	// Flutter: no counterpart
	BrightnessUnspecified Brightness = iota
	// BrightnessDark is a dark appearance.
	//
	// Flutter: ui/platform_dispatcher.dart Brightness.dark
	BrightnessDark
	// BrightnessLight is a light appearance.
	//
	// Flutter: ui/platform_dispatcher.dart Brightness.light
	BrightnessLight
)

var brightnessNames = [...]string{"BrightnessUnspecified", "BrightnessDark", "BrightnessLight"}

// String returns the name of the value, such as "BrightnessDark".
//
// Flutter: ui/platform_dispatcher.dart Brightness
func (b Brightness) String() string { return enumName(brightnessNames[:], int(b), "Brightness") }

// AccessibilityFeatures are the platform's accessibility settings that apply
// to the whole app.
//
// Flutter: ui/window.dart AccessibilityFeatures
type AccessibilityFeatures struct {
	// AccessibleNavigation is whether an assistive technology, such as a
	// screen reader, is in use.
	//
	// Flutter: ui/window.dart AccessibilityFeatures.accessibleNavigation
	AccessibleNavigation bool
	// InvertColors is whether the platform inverts the colors of the screen.
	//
	// Flutter: ui/window.dart AccessibilityFeatures.invertColors
	InvertColors bool
	// DisableAnimations is whether the user asked to turn animations off or
	// reduce them.
	//
	// Flutter: ui/window.dart AccessibilityFeatures.disableAnimations
	DisableAnimations bool
	// BoldText is whether the user asked for bold text.
	//
	// Flutter: ui/window.dart AccessibilityFeatures.boldText
	BoldText bool
	// ReduceMotion is whether the user asked to reduce motion.
	//
	// Flutter: ui/window.dart AccessibilityFeatures.reduceMotion
	ReduceMotion bool
	// HighContrast is whether the user asked for more contrast.
	//
	// Flutter: ui/window.dart AccessibilityFeatures.highContrast
	HighContrast bool
	// OnOffSwitchLabels is whether the user asked for labels on switches.
	//
	// Flutter: ui/window.dart AccessibilityFeatures.onOffSwitchLabels
	OnOffSwitchLabels bool
}

// String describes the features that are on, such as
// "AccessibilityFeatures[disableAnimations]".
//
// Flutter: ui/window.dart AccessibilityFeatures.toString
func (f AccessibilityFeatures) String() string {
	var on []string
	for _, e := range []struct {
		name string
		on   bool
	}{
		{"accessibleNavigation", f.AccessibleNavigation},
		{"invertColors", f.InvertColors},
		{"disableAnimations", f.DisableAnimations},
		{"boldText", f.BoldText},
		{"reduceMotion", f.ReduceMotion},
		{"highContrast", f.HighContrast},
		{"onOffSwitchLabels", f.OnOffSwitchLabels},
	} {
		if e.on {
			on = append(on, e.name)
		}
	}
	return "AccessibilityFeatures[" + strings.Join(on, ", ") + "]"
}

// ViewPadding is a set of four distances, in physical pixels, from the edges
// of a view: the areas that the system covers (or that the app should avoid).
//
// Flutter: ui/window.dart ViewPadding
type ViewPadding struct {
	// Left is the distance from the left edge.
	Left float64
	// Top is the distance from the top edge.
	Top float64
	// Right is the distance from the right edge.
	Right float64
	// Bottom is the distance from the bottom edge.
	Bottom float64
}

// ViewPaddingZero is a padding of zero on every side.
//
// Flutter: ui/window.dart ViewPadding.zero
var ViewPaddingZero = ViewPadding{}

// ViewConstraints are the limits, in physical pixels, on the size of a view.
//
// Flutter: ui/window.dart ViewConstraints
type ViewConstraints struct {
	// MinWidth is the smallest width the view can have.
	MinWidth float64
	// MaxWidth is the largest width the view can have.
	MaxWidth float64
	// MinHeight is the smallest height the view can have.
	MinHeight float64
	// MaxHeight is the largest height the view can have.
	MaxHeight float64
}

// NewViewConstraintsTight creates constraints that allow only the given size.
//
// Flutter: ui/window.dart ViewConstraints.tight
func NewViewConstraintsTight(size Size) ViewConstraints {
	return ViewConstraints{MinWidth: size.Width, MaxWidth: size.Width, MinHeight: size.Height, MaxHeight: size.Height}
}

// Display is a physical display (a monitor) that a view can be on.
//
// Flutter: ui/window.dart Display
type Display struct {
	// ID identifies the display.
	ID int
	// DevicePixelRatio is the number of physical pixels per logical pixel on
	// the display.
	DevicePixelRatio float64
	// Size is the size of the display in physical pixels.
	Size Size
	// RefreshRate is the refresh rate in frames per second.
	RefreshRate float64
}

// String describes the display, such as "Display(id: 1, size: 1920.0x1080.0)".
//
// Flutter: ui/window.dart Display.toString
func (d Display) String() string {
	return "Display(id: " + strconv.Itoa(d.ID) + ", size: " + toStringAsFixed1(d.Size.Width) + "x" +
		toStringAsFixed1(d.Size.Height) + ", dpr: " + toStringAsFixed1(d.DevicePixelRatio) + ", refreshRate: " +
		toStringAsFixed1(d.RefreshRate) + ")"
}
