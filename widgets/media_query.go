// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// Orientation is whether a size is taller than it is wide.
//
// Flutter: widgets/media_query.dart Orientation
type Orientation int

// The values of Orientation follow Flutter's declaration order.
const (
	// OrientationUnspecified is the zero value.
	//
	// Flutter: no counterpart
	OrientationUnspecified Orientation = iota
	// OrientationPortrait is a height at least as big as the width.
	//
	// Flutter: widgets/media_query.dart Orientation.portrait
	OrientationPortrait
	// OrientationLandscape is a width bigger than the height.
	//
	// Flutter: widgets/media_query.dart Orientation.landscape
	OrientationLandscape
)

// String returns the Flutter name of the value, such as "landscape".
//
// Flutter: widgets/media_query.dart Orientation
func (o Orientation) String() string {
	switch o {
	case OrientationPortrait:
		return "portrait"
	case OrientationLandscape:
		return "landscape"
	}
	return fmt.Sprintf("Orientation(%d)", int(o))
}

// NavigationMode describes the navigation mode to be set by a MediaQuery
// widget: traditional (touch and pointer) or directional (arrow keys move the
// focus).
//
// Flutter: widgets/media_query.dart NavigationMode
type NavigationMode int

// The values of NavigationMode follow Flutter's declaration order.
const (
	// NavigationModeUnspecified is the zero value; it counts as traditional.
	//
	// Flutter: no counterpart
	NavigationModeUnspecified NavigationMode = iota
	// NavigationModeTraditional uses a pointer or touch; the default.
	//
	// Flutter: widgets/media_query.dart NavigationMode.traditional
	NavigationModeTraditional
	// NavigationModeDirectional uses directional input to move the focus.
	//
	// Flutter: widgets/media_query.dart NavigationMode.directional
	NavigationModeDirectional
)

// String returns the Flutter name of the value, such as "traditional".
//
// Flutter: widgets/media_query.dart NavigationMode
func (m NavigationMode) String() string {
	switch m {
	case NavigationModeUnspecified, NavigationModeTraditional:
		return "traditional"
	case NavigationModeDirectional:
		return "directional"
	}
	return fmt.Sprintf("NavigationMode(%d)", int(m))
}

// mediaQueryAspect is a part of a MediaQueryData that a widget can depend on.
//
// Flutter: widgets/media_query.dart _MediaQueryAspect
type mediaQueryAspect int

const (
	aspectSize mediaQueryAspect = iota
	aspectWidth
	aspectHeight
	aspectOrientation
	aspectDevicePixelRatio
	aspectTextScaler
	aspectPlatformBrightness
	aspectPadding
	aspectViewInsets
	aspectSystemGestureInsets
	aspectViewPadding
	aspectAlwaysUse24HourFormat
	aspectAccessibleNavigation
	aspectInvertColors
	aspectHighContrast
	aspectOnOffSwitchLabels
	aspectDisableAnimations
	aspectBoldText
	aspectNavigationMode
	aspectGestureSettings
	aspectSupportsShowingSystemContextMenu
)

// MediaQueryData holds the information about a piece of media, such as a
// window: its size, the pixel density, the text scale, the insets the system
// covers and the accessibility settings. A MediaQuery widget makes it available
// to the widgets below. Make one with NewMediaQueryData or
// NewMediaQueryDataFromView; it is a value, copy it with CopyWith.
//
// Deviation: the fields Glutter has no source for are left out: display
// features and corner radii, supportsAnnounce, and the text style overrides
// (SPEC §12.10). AlwaysUse24HourFormat comes from the parent data only, and is
// false for a data made from the view.
//
// Flutter: widgets/media_query.dart MediaQueryData
type MediaQueryData struct {
	// Size is the size of the media in logical pixels (not physical pixels).
	//
	// Flutter: widgets/media_query.dart MediaQueryData.size
	Size ui.Size
	// DevicePixelRatio is the number of device pixels for each logical pixel.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.devicePixelRatio
	DevicePixelRatio float64
	// TextScaler is the font scaling strategy to use for the text in the app;
	// never nil.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.textScaler
	TextScaler painting.TextScaler
	// PlatformBrightness is the brightness of the system's color scheme.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.platformBrightness
	PlatformBrightness ui.Brightness
	// ViewInsets are the parts of the display that are completely obscured by
	// system UI, such as an on-screen keyboard.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.viewInsets
	ViewInsets painting.EdgeInsets
	// Padding are the parts of the display that are partially obscured by system
	// UI, and that SafeArea avoids; the view padding minus the view insets.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.padding
	Padding painting.EdgeInsets
	// ViewPadding are the parts of the display that are partially obscured by
	// system UI, whether or not the insets cover them.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.viewPadding
	ViewPadding painting.EdgeInsets
	// SystemGestureInsets are the areas along the edges where the system
	// consumes certain input events.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.systemGestureInsets
	SystemGestureInsets painting.EdgeInsets
	// AlwaysUse24HourFormat is whether to use 24-hour format when formatting
	// time.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.alwaysUse24HourFormat
	AlwaysUse24HourFormat bool
	// AccessibleNavigation is whether the user is using an accessibility
	// service, such as a screen reader.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.accessibleNavigation
	AccessibleNavigation bool
	// InvertColors is whether the platform is inverting the colors.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.invertColors
	InvertColors bool
	// HighContrast is whether the user requested a high contrast.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.highContrast
	HighContrast bool
	// OnOffSwitchLabels is whether the user requested labels on switches.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.onOffSwitchLabels
	OnOffSwitchLabels bool
	// DisableAnimations is whether the platform is requesting that animations be
	// disabled or reduced as much as possible.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.disableAnimations
	DisableAnimations bool
	// BoldText is whether the user requested bold text.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.boldText
	BoldText bool
	// NavigationMode is the navigation mode of the app.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.navigationMode
	NavigationMode NavigationMode
	// GestureSettings are the gesture thresholds of the device.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.gestureSettings
	GestureSettings gestures.DeviceGestureSettings
	// SupportsShowingSystemContextMenu is whether the platform can show the
	// system context menu.
	//
	// Flutter: widgets/media_query.dart MediaQueryData.supportsShowingSystemContextMenu
	SupportsShowingSystemContextMenu bool
}

// MediaQueryDataOpts are the optional parameters of NewMediaQueryData. The zero
// value of each field is Flutter's default: no size, a pixel ratio of 1, no
// text scaling, the light brightness, no insets, a touch slop of 18 and the
// traditional navigation mode.
//
// Flutter: widgets/media_query.dart MediaQueryData.new
type MediaQueryDataOpts struct {
	// Size is the size in logical pixels.
	Size ui.Size
	// DevicePixelRatio is the number of device pixels per logical pixel; 0 means 1.
	DevicePixelRatio float64
	// TextScaler is the text scaling; nil means no scaling.
	TextScaler painting.TextScaler
	// PlatformBrightness defaults to the light brightness.
	PlatformBrightness ui.Brightness
	// Padding is the partially obscured area.
	Padding painting.EdgeInsets
	// ViewInsets is the completely obscured area.
	ViewInsets painting.EdgeInsets
	// SystemGestureInsets is the area where the system takes the gestures.
	SystemGestureInsets painting.EdgeInsets
	// ViewPadding is the partially obscured area, ignoring the view insets.
	ViewPadding painting.EdgeInsets
	// AlwaysUse24HourFormat says to use the 24-hour format.
	AlwaysUse24HourFormat bool
	// AccessibleNavigation says that an accessibility service is in use.
	AccessibleNavigation bool
	// InvertColors says that the platform inverts the colors.
	InvertColors bool
	// HighContrast says that the user asked for a high contrast.
	HighContrast bool
	// OnOffSwitchLabels says that the user asked for labels on switches.
	OnOffSwitchLabels bool
	// DisableAnimations says that animations should be off or reduced.
	DisableAnimations bool
	// BoldText says that the user asked for bold text.
	BoldText bool
	// NavigationMode defaults to the traditional mode.
	NavigationMode NavigationMode
	// GestureSettings defaults to a touch slop of gestures.TouchSlop.
	GestureSettings foundation.Opt[gestures.DeviceGestureSettings]
	// SupportsShowingSystemContextMenu says that the platform can show the
	// system context menu.
	SupportsShowingSystemContextMenu bool
}

// NewMediaQueryData creates the data from the options.
//
// Flutter: widgets/media_query.dart MediaQueryData.new
func NewMediaQueryData(opts ...MediaQueryDataOpts) MediaQueryData {
	o := firstOpts(opts)
	d := MediaQueryData{
		Size: o.Size, DevicePixelRatio: o.DevicePixelRatio, TextScaler: o.TextScaler, PlatformBrightness: o.PlatformBrightness,
		Padding: o.Padding, ViewInsets: o.ViewInsets, SystemGestureInsets: o.SystemGestureInsets, ViewPadding: o.ViewPadding,
		AlwaysUse24HourFormat: o.AlwaysUse24HourFormat, AccessibleNavigation: o.AccessibleNavigation, InvertColors: o.InvertColors,
		HighContrast: o.HighContrast, OnOffSwitchLabels: o.OnOffSwitchLabels, DisableAnimations: o.DisableAnimations,
		BoldText: o.BoldText, NavigationMode: o.NavigationMode, SupportsShowingSystemContextMenu: o.SupportsShowingSystemContextMenu,
	}
	if d.DevicePixelRatio == 0 {
		d.DevicePixelRatio = 1
	}
	if d.TextScaler == nil {
		d.TextScaler = painting.TextScalerNoScaling
	}
	if d.PlatformBrightness == ui.BrightnessUnspecified {
		d.PlatformBrightness = ui.BrightnessLight
	}
	if d.NavigationMode == NavigationModeUnspecified {
		d.NavigationMode = NavigationModeTraditional
	}
	d.GestureSettings = o.GestureSettings.OrElse(gestures.NewDeviceGestureSettings(foundation.Some(gestures.TouchSlop)))
	return d
}

// NewMediaQueryDataFromView creates the data from the view and the dispatcher
// it belongs to. The data of a MediaQuery above (platformData, which may be
// nil) overrides the settings of the platform, as it does for a MediaQuery
// made with MediaQueryFromView: the brightness, the text scaler, the
// accessibility settings, the 24-hour format, the navigation mode and the
// system context menu support.
//
// Flutter: widgets/media_query.dart MediaQueryData.fromView
func NewMediaQueryDataFromView(view *ui.View, platformData *MediaQueryData) MediaQueryData {
	d := view.PlatformDispatcher()
	ratio := view.DevicePixelRatio()
	size := view.PhysicalSize()
	features := d.AccessibilityFeatures()
	data := MediaQueryData{
		Size:                 ui.Size{Width: size.Width / ratio, Height: size.Height / ratio},
		DevicePixelRatio:     ratio,
		TextScaler:           painting.NewTextScalerLinear(d.TextScaleFactor()),
		PlatformBrightness:   d.PlatformBrightness(),
		Padding:              painting.NewEdgeInsetsFromViewPadding(view.Padding(), ratio),
		ViewPadding:          painting.NewEdgeInsetsFromViewPadding(view.ViewPadding(), ratio),
		ViewInsets:           painting.NewEdgeInsetsFromViewPadding(view.ViewInsets(), ratio),
		SystemGestureInsets:  painting.NewEdgeInsetsFromViewPadding(view.SystemGestureInsets(), ratio),
		AccessibleNavigation: features.AccessibleNavigation, InvertColors: features.InvertColors,
		DisableAnimations: features.DisableAnimations, BoldText: features.BoldText, HighContrast: features.HighContrast,
		OnOffSwitchLabels: features.OnOffSwitchLabels,
		NavigationMode:    NavigationModeTraditional,
		GestureSettings:   gestures.NewDeviceGestureSettingsFromView(view),
	}
	if p := platformData; p != nil {
		data.TextScaler = p.TextScaler
		data.PlatformBrightness = p.PlatformBrightness
		data.AccessibleNavigation = p.AccessibleNavigation
		data.InvertColors = p.InvertColors
		data.DisableAnimations = p.DisableAnimations
		data.BoldText = p.BoldText
		data.HighContrast = p.HighContrast
		data.OnOffSwitchLabels = p.OnOffSwitchLabels
		data.AlwaysUse24HourFormat = p.AlwaysUse24HourFormat
		data.NavigationMode = p.NavigationMode
		data.SupportsShowingSystemContextMenu = p.SupportsShowingSystemContextMenu
	}
	return data
}

// Orientation is whether the size is wider than it is tall.
//
// Flutter: widgets/media_query.dart MediaQueryData.orientation
func (d MediaQueryData) Orientation() Orientation {
	if d.Size.Width > d.Size.Height {
		return OrientationLandscape
	}
	return OrientationPortrait
}

// MediaQueryDataCopyOpts are the fields CopyWith replaces; a field left None
// keeps the value of the data.
//
// Flutter: widgets/media_query.dart MediaQueryData.copyWith
type MediaQueryDataCopyOpts struct {
	Size                             foundation.Opt[ui.Size]
	DevicePixelRatio                 foundation.Opt[float64]
	TextScaler                       foundation.Opt[painting.TextScaler]
	PlatformBrightness               foundation.Opt[ui.Brightness]
	Padding                          foundation.Opt[painting.EdgeInsets]
	ViewPadding                      foundation.Opt[painting.EdgeInsets]
	ViewInsets                       foundation.Opt[painting.EdgeInsets]
	SystemGestureInsets              foundation.Opt[painting.EdgeInsets]
	AlwaysUse24HourFormat            foundation.Opt[bool]
	HighContrast                     foundation.Opt[bool]
	OnOffSwitchLabels                foundation.Opt[bool]
	DisableAnimations                foundation.Opt[bool]
	InvertColors                     foundation.Opt[bool]
	AccessibleNavigation             foundation.Opt[bool]
	BoldText                         foundation.Opt[bool]
	NavigationMode                   foundation.Opt[NavigationMode]
	GestureSettings                  foundation.Opt[gestures.DeviceGestureSettings]
	SupportsShowingSystemContextMenu foundation.Opt[bool]
}

// CopyWith returns a copy with the fields in the options replaced.
//
// Flutter: widgets/media_query.dart MediaQueryData.copyWith
func (d MediaQueryData) CopyWith(o MediaQueryDataCopyOpts) MediaQueryData {
	if v, ok := o.Size.Get(); ok {
		d.Size = v
	}
	if v, ok := o.DevicePixelRatio.Get(); ok {
		d.DevicePixelRatio = v
	}
	if v, ok := o.TextScaler.Get(); ok {
		d.TextScaler = v
	}
	if v, ok := o.PlatformBrightness.Get(); ok {
		d.PlatformBrightness = v
	}
	if v, ok := o.Padding.Get(); ok {
		d.Padding = v
	}
	if v, ok := o.ViewPadding.Get(); ok {
		d.ViewPadding = v
	}
	if v, ok := o.ViewInsets.Get(); ok {
		d.ViewInsets = v
	}
	if v, ok := o.SystemGestureInsets.Get(); ok {
		d.SystemGestureInsets = v
	}
	if v, ok := o.AlwaysUse24HourFormat.Get(); ok {
		d.AlwaysUse24HourFormat = v
	}
	if v, ok := o.HighContrast.Get(); ok {
		d.HighContrast = v
	}
	if v, ok := o.OnOffSwitchLabels.Get(); ok {
		d.OnOffSwitchLabels = v
	}
	if v, ok := o.DisableAnimations.Get(); ok {
		d.DisableAnimations = v
	}
	if v, ok := o.InvertColors.Get(); ok {
		d.InvertColors = v
	}
	if v, ok := o.AccessibleNavigation.Get(); ok {
		d.AccessibleNavigation = v
	}
	if v, ok := o.BoldText.Get(); ok {
		d.BoldText = v
	}
	if v, ok := o.NavigationMode.Get(); ok {
		d.NavigationMode = v
	}
	if v, ok := o.GestureSettings.Get(); ok {
		d.GestureSettings = v
	}
	if v, ok := o.SupportsShowingSystemContextMenu.Get(); ok {
		d.SupportsShowingSystemContextMenu = v
	}
	return d
}

// RemoveSides says which sides of the insets a Remove method removes.
//
// Flutter: widgets/media_query.dart MediaQueryData.removePadding
type RemoveSides struct {
	// Left removes the left side.
	Left bool
	// Top removes the top side.
	Top bool
	// Right removes the right side.
	Right bool
	// Bottom removes the bottom side.
	Bottom bool
}

func (s RemoveSides) any() bool { return s.Left || s.Top || s.Right || s.Bottom }

// zeroSides sets the sides to zero.
func zeroSides(e painting.EdgeInsets, s RemoveSides) painting.EdgeInsets {
	if s.Left {
		e.Left = 0
	}
	if s.Top {
		e.Top = 0
	}
	if s.Right {
		e.Right = 0
	}
	if s.Bottom {
		e.Bottom = 0
	}
	return e
}

// shrinkSides reduces the sides of e by the sides of by, not below zero.
func shrinkSides(e, by painting.EdgeInsets, s RemoveSides) painting.EdgeInsets {
	if s.Left {
		e.Left = math.Max(0, e.Left-by.Left)
	}
	if s.Top {
		e.Top = math.Max(0, e.Top-by.Top)
	}
	if s.Right {
		e.Right = math.Max(0, e.Right-by.Right)
	}
	if s.Bottom {
		e.Bottom = math.Max(0, e.Bottom-by.Bottom)
	}
	return e
}

// RemovePadding returns a copy with the sides of the padding removed (set to
// zero) and the same amount taken from the view padding.
//
// Flutter: widgets/media_query.dart MediaQueryData.removePadding
func (d MediaQueryData) RemovePadding(sides RemoveSides) MediaQueryData {
	if !sides.any() {
		return d
	}
	return d.CopyWith(MediaQueryDataCopyOpts{
		Padding:     foundation.Some(zeroSides(d.Padding, sides)),
		ViewPadding: foundation.Some(shrinkSides(d.ViewPadding, d.Padding, sides)),
	})
}

// RemoveViewInsets returns a copy with the sides of the view insets removed and
// the same amount taken from the view padding.
//
// Flutter: widgets/media_query.dart MediaQueryData.removeViewInsets
func (d MediaQueryData) RemoveViewInsets(sides RemoveSides) MediaQueryData {
	if !sides.any() {
		return d
	}
	return d.CopyWith(MediaQueryDataCopyOpts{
		ViewPadding: foundation.Some(shrinkSides(d.ViewPadding, d.ViewInsets, sides)),
		ViewInsets:  foundation.Some(zeroSides(d.ViewInsets, sides)),
	})
}

// RemoveViewPadding returns a copy with the sides of the padding and of the view
// padding removed.
//
// Flutter: widgets/media_query.dart MediaQueryData.removeViewPadding
func (d MediaQueryData) RemoveViewPadding(sides RemoveSides) MediaQueryData {
	if !sides.any() {
		return d
	}
	return d.CopyWith(MediaQueryDataCopyOpts{
		Padding:     foundation.Some(zeroSides(d.Padding, sides)),
		ViewPadding: foundation.Some(zeroSides(d.ViewPadding, sides)),
	})
}

// Equal is whether the two data are the same, field by field. The text scalers
// are equal when they scale the same way.
//
// Flutter: widgets/media_query.dart MediaQueryData.operator ==
func (d MediaQueryData) Equal(o MediaQueryData) bool {
	return d.Size == o.Size && d.DevicePixelRatio == o.DevicePixelRatio &&
		painting.TextScalerEqual(d.TextScaler, o.TextScaler) && d.PlatformBrightness == o.PlatformBrightness &&
		d.Padding == o.Padding && d.ViewPadding == o.ViewPadding && d.ViewInsets == o.ViewInsets &&
		d.SystemGestureInsets == o.SystemGestureInsets && d.AlwaysUse24HourFormat == o.AlwaysUse24HourFormat &&
		d.HighContrast == o.HighContrast && d.OnOffSwitchLabels == o.OnOffSwitchLabels &&
		d.DisableAnimations == o.DisableAnimations && d.InvertColors == o.InvertColors &&
		d.AccessibleNavigation == o.AccessibleNavigation && d.BoldText == o.BoldText &&
		d.NavigationMode == o.NavigationMode && d.GestureSettings == o.GestureSettings &&
		d.SupportsShowingSystemContextMenu == o.SupportsShowingSystemContextMenu
}

// String describes the data, such as
// "MediaQueryData(size: Size(800.0, 600.0), devicePixelRatio: 1.0, ...)".
//
// Flutter: widgets/media_query.dart MediaQueryData.toString
func (d MediaQueryData) String() string {
	ratio := fmt.Sprintf("%.1f", d.DevicePixelRatio)
	scaler := fmt.Sprint(d.TextScaler)
	props := []string{
		"size: " + d.Size.String(),
		"devicePixelRatio: " + ratio,
		"textScaler: " + scaler,
		"platformBrightness: " + brightnessName(d.PlatformBrightness),
		"padding: " + d.Padding.String(),
		"viewPadding: " + d.ViewPadding.String(),
		"viewInsets: " + d.ViewInsets.String(),
		"systemGestureInsets: " + d.SystemGestureInsets.String(),
		"alwaysUse24HourFormat: " + fmt.Sprint(d.AlwaysUse24HourFormat),
		"accessibleNavigation: " + fmt.Sprint(d.AccessibleNavigation),
		"highContrast: " + fmt.Sprint(d.HighContrast),
		"onOffSwitchLabels: " + fmt.Sprint(d.OnOffSwitchLabels),
		"disableAnimations: " + fmt.Sprint(d.DisableAnimations),
		"invertColors: " + fmt.Sprint(d.InvertColors),
		"boldText: " + fmt.Sprint(d.BoldText),
		"navigationMode: " + d.NavigationMode.String(),
		"gestureSettings: " + d.GestureSettings.String(),
		"supportsShowingSystemContextMenu: " + fmt.Sprint(d.SupportsShowingSystemContextMenu),
	}
	return "MediaQueryData(" + strings.Join(props, ", ") + ")"
}

func brightnessName(b ui.Brightness) string {
	switch b {
	case ui.BrightnessDark:
		return "dark"
	case ui.BrightnessLight:
		return "light"
	}
	return "unspecified"
}

// MediaQuery establishes a subtree in which media queries resolve to the given
// data. Widgets below depend on the whole data (MediaQueryOf) or on one aspect
// of it (MediaQuerySizeOf and the like), and are rebuilt only when what they
// depend on changed.
//
// Flutter: widgets/media_query.dart MediaQuery
type MediaQuery struct {
	InheritedModel
	data MediaQueryData
}

// NewMediaQuery creates a MediaQuery that gives the data to the widgets below.
//
// Flutter: widgets/media_query.dart MediaQuery.new
func NewMediaQuery(data MediaQueryData, child Widget, opts ...WidgetOpts) *MediaQuery {
	return &MediaQuery{NewInheritedModelBase(child, opts...), data}
}

// NewMediaQueryRemovePadding creates a MediaQuery whose data is the one of the
// context with the sides of the padding removed. It depends on the whole data
// of the MediaQuery above.
//
// Flutter: widgets/media_query.dart MediaQuery.removePadding
func NewMediaQueryRemovePadding(bc BuildContext, sides RemoveSides, child Widget, opts ...WidgetOpts) *MediaQuery {
	return NewMediaQuery(MediaQueryOf(bc).RemovePadding(sides), child, opts...)
}

// NewMediaQueryRemoveViewInsets creates a MediaQuery whose data is the one of
// the context with the sides of the view insets removed.
//
// Flutter: widgets/media_query.dart MediaQuery.removeViewInsets
func NewMediaQueryRemoveViewInsets(bc BuildContext, sides RemoveSides, child Widget, opts ...WidgetOpts) *MediaQuery {
	return NewMediaQuery(MediaQueryOf(bc).RemoveViewInsets(sides), child, opts...)
}

// NewMediaQueryRemoveViewPadding creates a MediaQuery whose data is the one of
// the context with the sides of the view padding removed.
//
// Flutter: widgets/media_query.dart MediaQuery.removeViewPadding
func NewMediaQueryRemoveViewPadding(bc BuildContext, sides RemoveSides, child Widget, opts ...WidgetOpts) *MediaQuery {
	return NewMediaQuery(MediaQueryOf(bc).RemoveViewPadding(sides), child, opts...)
}

// Data is the data the widget gives.
//
// Flutter: widgets/media_query.dart MediaQuery.data
func (w *MediaQuery) Data() MediaQueryData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: widgets/media_query.dart MediaQuery.updateShouldNotify
func (w *MediaQuery) UpdateShouldNotify(old Widget) bool {
	return !w.data.Equal(old.(*MediaQuery).data)
}

// UpdateShouldNotifyDependent is whether one of the aspects the dependent asked
// for changed.
//
// Flutter: widgets/media_query.dart MediaQuery.updateShouldNotifyDependent
func (w *MediaQuery) UpdateShouldNotifyDependent(old Widget, dependencies *AspectSet) bool {
	o, n := old.(*MediaQuery).data, w.data
	return slices.ContainsFunc(dependencies.Items(), func(dep any) bool {
		aspect, ok := dep.(mediaQueryAspect)
		if !ok {
			return false
		}
		switch aspect {
		case aspectSize:
			return n.Size != o.Size
		case aspectWidth:
			return n.Size.Width != o.Size.Width
		case aspectHeight:
			return n.Size.Height != o.Size.Height
		case aspectOrientation:
			return n.Orientation() != o.Orientation()
		case aspectDevicePixelRatio:
			return n.DevicePixelRatio != o.DevicePixelRatio
		case aspectTextScaler:
			return !painting.TextScalerEqual(n.TextScaler, o.TextScaler)
		case aspectPlatformBrightness:
			return n.PlatformBrightness != o.PlatformBrightness
		case aspectPadding:
			return n.Padding != o.Padding
		case aspectViewInsets:
			return n.ViewInsets != o.ViewInsets
		case aspectViewPadding:
			return n.ViewPadding != o.ViewPadding
		case aspectInvertColors:
			return n.InvertColors != o.InvertColors
		case aspectHighContrast:
			return n.HighContrast != o.HighContrast
		case aspectOnOffSwitchLabels:
			return n.OnOffSwitchLabels != o.OnOffSwitchLabels
		case aspectDisableAnimations:
			return n.DisableAnimations != o.DisableAnimations
		case aspectBoldText:
			return n.BoldText != o.BoldText
		case aspectNavigationMode:
			return n.NavigationMode != o.NavigationMode
		case aspectGestureSettings:
			return n.GestureSettings != o.GestureSettings
		case aspectSystemGestureInsets:
			return n.SystemGestureInsets != o.SystemGestureInsets
		case aspectAccessibleNavigation:
			return n.AccessibleNavigation != o.AccessibleNavigation
		case aspectAlwaysUse24HourFormat:
			return n.AlwaysUse24HourFormat != o.AlwaysUse24HourFormat
		case aspectSupportsShowingSystemContextMenu:
			return n.SupportsShowingSystemContextMenu != o.SupportsShowingSystemContextMenu
		}
		return false
	})
}

// DebugFillProperties adds the data.
//
// Flutter: widgets/media_query.dart MediaQuery.debugFillProperties
func (w *MediaQuery) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(foundation.NewStringProperty("data", foundation.Some(w.data.String()), foundation.StringPropertyOpts{ShowName: foundation.Some(false)}))
}

// DebugCheckHasMediaQuery panics with the tag MediaQueryMissing when there is
// no MediaQuery above the context (or at it). It returns true so it can be used
// in a condition.
//
// Flutter: widgets/debug.dart debugCheckHasMediaQuery
func DebugCheckHasMediaQuery(bc BuildContext) bool {
	if _, ok := bc.Widget().(*MediaQuery); ok {
		return true
	}
	if GetElementForInheritedWidget[*MediaQuery](bc) != nil {
		return true
	}
	panic(errors.New("MediaQueryMissing",
		"no MediaQuery widget ancestor found; the widget needs a MediaQuery above it to read the media; MediaQuery is introduced by View (the app's root, made by RunApp) and by WidgetsApp, so this usually means the widget was built outside of them",
		"widget", bc.(Element).String()))
}

func mediaQueryOf(bc BuildContext, aspect any) MediaQueryData {
	DebugCheckHasMediaQuery(bc)
	m, _ := DependOnModel[*MediaQuery](bc, aspect)
	return m.data
}

func mediaQueryMaybeOf(bc BuildContext, aspect any) (MediaQueryData, bool) {
	m, ok := DependOnModel[*MediaQuery](bc, aspect)
	if !ok {
		return MediaQueryData{}, false
	}
	return m.data, true
}

// MediaQueryOf is the data of the closest MediaQuery above the context; the
// context depends on all of it, so prefer the functions for one aspect. It
// panics with the tag MediaQueryMissing when there is none.
//
// Flutter: widgets/media_query.dart MediaQuery.of
func MediaQueryOf(bc BuildContext) MediaQueryData { return mediaQueryOf(bc, nil) }

// MediaQueryMaybeOf is MediaQueryOf that returns false when there is no
// MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeOf
func MediaQueryMaybeOf(bc BuildContext) (MediaQueryData, bool) { return mediaQueryMaybeOf(bc, nil) }

// MediaQuerySizeOf is the size of the media; the context depends on the size
// only.
//
// Flutter: widgets/media_query.dart MediaQuery.sizeOf
func MediaQuerySizeOf(bc BuildContext) ui.Size { return mediaQueryOf(bc, aspectSize).Size }

// MediaQueryMaybeSizeOf is MediaQuerySizeOf that returns false when there is no
// MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeSizeOf
func MediaQueryMaybeSizeOf(bc BuildContext) (ui.Size, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectSize)
	return d.Size, ok
}

// MediaQueryWidthOf is the width of the media; the context depends on the width
// only.
//
// Flutter: widgets/media_query.dart MediaQuery.widthOf
func MediaQueryWidthOf(bc BuildContext) float64 { return mediaQueryOf(bc, aspectWidth).Size.Width }

// MediaQueryMaybeWidthOf is MediaQueryWidthOf that returns false when there is
// no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeWidthOf
func MediaQueryMaybeWidthOf(bc BuildContext) (float64, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectWidth)
	return d.Size.Width, ok
}

// MediaQueryHeightOf is the height of the media; the context depends on the
// height only.
//
// Flutter: widgets/media_query.dart MediaQuery.heightOf
func MediaQueryHeightOf(bc BuildContext) float64 { return mediaQueryOf(bc, aspectHeight).Size.Height }

// MediaQueryMaybeHeightOf is MediaQueryHeightOf that returns false when there
// is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeHeightOf
func MediaQueryMaybeHeightOf(bc BuildContext) (float64, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectHeight)
	return d.Size.Height, ok
}

// MediaQueryOrientationOf is the orientation of the media.
//
// Flutter: widgets/media_query.dart MediaQuery.orientationOf
func MediaQueryOrientationOf(bc BuildContext) Orientation {
	return mediaQueryOf(bc, aspectOrientation).Orientation()
}

// MediaQueryMaybeOrientationOf is MediaQueryOrientationOf that returns false
// when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeOrientationOf
func MediaQueryMaybeOrientationOf(bc BuildContext) (Orientation, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectOrientation)
	return d.Orientation(), ok
}

// MediaQueryDevicePixelRatioOf is the number of device pixels per logical pixel.
//
// Flutter: widgets/media_query.dart MediaQuery.devicePixelRatioOf
func MediaQueryDevicePixelRatioOf(bc BuildContext) float64 {
	return mediaQueryOf(bc, aspectDevicePixelRatio).DevicePixelRatio
}

// MediaQueryMaybeDevicePixelRatioOf is MediaQueryDevicePixelRatioOf that returns
// false when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeDevicePixelRatioOf
func MediaQueryMaybeDevicePixelRatioOf(bc BuildContext) (float64, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectDevicePixelRatio)
	return d.DevicePixelRatio, ok
}

// MediaQueryTextScalerOf is the text scaler of the media, or no scaling when
// there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.textScalerOf
func MediaQueryTextScalerOf(bc BuildContext) painting.TextScaler {
	if d, ok := mediaQueryMaybeOf(bc, aspectTextScaler); ok {
		return d.TextScaler
	}
	return painting.TextScalerNoScaling
}

// MediaQueryMaybeTextScalerOf is the text scaler of the media; false when there
// is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeTextScalerOf
func MediaQueryMaybeTextScalerOf(bc BuildContext) (painting.TextScaler, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectTextScaler)
	return d.TextScaler, ok
}

// MediaQueryPlatformBrightnessOf is the brightness of the system, or light when
// there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.platformBrightnessOf
func MediaQueryPlatformBrightnessOf(bc BuildContext) ui.Brightness {
	if d, ok := mediaQueryMaybeOf(bc, aspectPlatformBrightness); ok {
		return d.PlatformBrightness
	}
	return ui.BrightnessLight
}

// MediaQueryMaybePlatformBrightnessOf is the brightness of the system; false
// when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybePlatformBrightnessOf
func MediaQueryMaybePlatformBrightnessOf(bc BuildContext) (ui.Brightness, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectPlatformBrightness)
	return d.PlatformBrightness, ok
}

// MediaQueryPaddingOf is the padding of the media.
//
// Flutter: widgets/media_query.dart MediaQuery.paddingOf
func MediaQueryPaddingOf(bc BuildContext) painting.EdgeInsets {
	return mediaQueryOf(bc, aspectPadding).Padding
}

// MediaQueryMaybePaddingOf is MediaQueryPaddingOf that returns false when there
// is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybePaddingOf
func MediaQueryMaybePaddingOf(bc BuildContext) (painting.EdgeInsets, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectPadding)
	return d.Padding, ok
}

// MediaQueryViewInsetsOf is the view insets of the media.
//
// Flutter: widgets/media_query.dart MediaQuery.viewInsetsOf
func MediaQueryViewInsetsOf(bc BuildContext) painting.EdgeInsets {
	return mediaQueryOf(bc, aspectViewInsets).ViewInsets
}

// MediaQueryMaybeViewInsetsOf is MediaQueryViewInsetsOf that returns false when
// there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeViewInsetsOf
func MediaQueryMaybeViewInsetsOf(bc BuildContext) (painting.EdgeInsets, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectViewInsets)
	return d.ViewInsets, ok
}

// MediaQuerySystemGestureInsetsOf is the system gesture insets of the media.
//
// Flutter: widgets/media_query.dart MediaQuery.systemGestureInsetsOf
func MediaQuerySystemGestureInsetsOf(bc BuildContext) painting.EdgeInsets {
	return mediaQueryOf(bc, aspectSystemGestureInsets).SystemGestureInsets
}

// MediaQueryMaybeSystemGestureInsetsOf is MediaQuerySystemGestureInsetsOf that
// returns false when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeSystemGestureInsetsOf
func MediaQueryMaybeSystemGestureInsetsOf(bc BuildContext) (painting.EdgeInsets, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectSystemGestureInsets)
	return d.SystemGestureInsets, ok
}

// MediaQueryViewPaddingOf is the view padding of the media.
//
// Flutter: widgets/media_query.dart MediaQuery.viewPaddingOf
func MediaQueryViewPaddingOf(bc BuildContext) painting.EdgeInsets {
	return mediaQueryOf(bc, aspectViewPadding).ViewPadding
}

// MediaQueryMaybeViewPaddingOf is MediaQueryViewPaddingOf that returns false
// when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeViewPaddingOf
func MediaQueryMaybeViewPaddingOf(bc BuildContext) (painting.EdgeInsets, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectViewPadding)
	return d.ViewPadding, ok
}

// MediaQueryAlwaysUse24HourFormatOf is whether to use the 24-hour format.
//
// Flutter: widgets/media_query.dart MediaQuery.alwaysUse24HourFormatOf
func MediaQueryAlwaysUse24HourFormatOf(bc BuildContext) bool {
	return mediaQueryOf(bc, aspectAlwaysUse24HourFormat).AlwaysUse24HourFormat
}

// MediaQueryMaybeAlwaysUse24HourFormatOf is MediaQueryAlwaysUse24HourFormatOf
// that returns false when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeAlwaysUse24HourFormatOf
func MediaQueryMaybeAlwaysUse24HourFormatOf(bc BuildContext) (value, ok bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectAlwaysUse24HourFormat)
	return d.AlwaysUse24HourFormat, ok
}

// MediaQueryAccessibleNavigationOf is whether an accessibility service is in
// use.
//
// Flutter: widgets/media_query.dart MediaQuery.accessibleNavigationOf
func MediaQueryAccessibleNavigationOf(bc BuildContext) bool {
	return mediaQueryOf(bc, aspectAccessibleNavigation).AccessibleNavigation
}

// MediaQueryMaybeAccessibleNavigationOf is MediaQueryAccessibleNavigationOf that
// returns false in ok when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeAccessibleNavigationOf
func MediaQueryMaybeAccessibleNavigationOf(bc BuildContext) (value, ok bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectAccessibleNavigation)
	return d.AccessibleNavigation, ok
}

// MediaQueryInvertColorsOf is whether the platform inverts the colors.
//
// Flutter: widgets/media_query.dart MediaQuery.invertColorsOf
func MediaQueryInvertColorsOf(bc BuildContext) bool {
	return mediaQueryOf(bc, aspectInvertColors).InvertColors
}

// MediaQueryMaybeInvertColorsOf is MediaQueryInvertColorsOf that returns false
// in ok when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeInvertColorsOf
func MediaQueryMaybeInvertColorsOf(bc BuildContext) (value, ok bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectInvertColors)
	return d.InvertColors, ok
}

// MediaQueryHighContrastOf is whether the user asked for a high contrast; false
// when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.highContrastOf
func MediaQueryHighContrastOf(bc BuildContext) bool {
	d, ok := mediaQueryMaybeOf(bc, aspectHighContrast)
	return ok && d.HighContrast
}

// MediaQueryMaybeHighContrastOf is the high contrast setting; false in ok when
// there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeHighContrastOf
func MediaQueryMaybeHighContrastOf(bc BuildContext) (value, ok bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectHighContrast)
	return d.HighContrast, ok
}

// MediaQueryOnOffSwitchLabelsOf is whether the user asked for labels on
// switches; false when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.onOffSwitchLabelsOf
func MediaQueryOnOffSwitchLabelsOf(bc BuildContext) bool {
	d, ok := mediaQueryMaybeOf(bc, aspectOnOffSwitchLabels)
	return ok && d.OnOffSwitchLabels
}

// MediaQueryMaybeOnOffSwitchLabelsOf is the switch labels setting; false in ok
// when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeOnOffSwitchLabelsOf
func MediaQueryMaybeOnOffSwitchLabelsOf(bc BuildContext) (value, ok bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectOnOffSwitchLabels)
	return d.OnOffSwitchLabels, ok
}

// MediaQueryDisableAnimationsOf is whether animations should be off or reduced.
//
// Flutter: widgets/media_query.dart MediaQuery.disableAnimationsOf
func MediaQueryDisableAnimationsOf(bc BuildContext) bool {
	return mediaQueryOf(bc, aspectDisableAnimations).DisableAnimations
}

// MediaQueryMaybeDisableAnimationsOf is MediaQueryDisableAnimationsOf that
// returns false in ok when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeDisableAnimationsOf
func MediaQueryMaybeDisableAnimationsOf(bc BuildContext) (value, ok bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectDisableAnimations)
	return d.DisableAnimations, ok
}

// MediaQueryBoldTextOf is whether the user asked for bold text; false when there
// is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.boldTextOf
func MediaQueryBoldTextOf(bc BuildContext) bool {
	d, ok := mediaQueryMaybeOf(bc, aspectBoldText)
	return ok && d.BoldText
}

// MediaQueryMaybeBoldTextOf is the bold text setting; false in ok when there is
// no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeBoldTextOf
func MediaQueryMaybeBoldTextOf(bc BuildContext) (value, ok bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectBoldText)
	return d.BoldText, ok
}

// MediaQueryNavigationModeOf is the navigation mode of the app.
//
// Flutter: widgets/media_query.dart MediaQuery.navigationModeOf
func MediaQueryNavigationModeOf(bc BuildContext) NavigationMode {
	return mediaQueryOf(bc, aspectNavigationMode).NavigationMode
}

// MediaQueryMaybeNavigationModeOf is MediaQueryNavigationModeOf that returns
// false when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeNavigationModeOf
func MediaQueryMaybeNavigationModeOf(bc BuildContext) (NavigationMode, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectNavigationMode)
	return d.NavigationMode, ok
}

// MediaQueryGestureSettingsOf is the gesture settings of the device.
//
// Flutter: widgets/media_query.dart MediaQuery.gestureSettingsOf
func MediaQueryGestureSettingsOf(bc BuildContext) gestures.DeviceGestureSettings {
	return mediaQueryOf(bc, aspectGestureSettings).GestureSettings
}

// MediaQueryMaybeGestureSettingsOf is MediaQueryGestureSettingsOf that returns
// false when there is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeGestureSettingsOf
func MediaQueryMaybeGestureSettingsOf(bc BuildContext) (gestures.DeviceGestureSettings, bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectGestureSettings)
	return d.GestureSettings, ok
}

// MediaQuerySupportsShowingSystemContextMenu is whether the platform can show
// the system context menu.
//
// Flutter: widgets/media_query.dart MediaQuery.supportsShowingSystemContextMenu
func MediaQuerySupportsShowingSystemContextMenu(bc BuildContext) bool {
	return mediaQueryOf(bc, aspectSupportsShowingSystemContextMenu).SupportsShowingSystemContextMenu
}

// MediaQueryMaybeSupportsShowingSystemContextMenu is
// MediaQuerySupportsShowingSystemContextMenu that returns false in ok when there
// is no MediaQuery.
//
// Flutter: widgets/media_query.dart MediaQuery.maybeSupportsShowingSystemContextMenu
func MediaQueryMaybeSupportsShowingSystemContextMenu(bc BuildContext) (value, ok bool) {
	d, ok := mediaQueryMaybeOf(bc, aspectSupportsShowingSystemContextMenu)
	return d.SupportsShowingSystemContextMenu, ok
}

// buildFunc is a stateless widget made from a function, used where Flutter uses
// a Builder.
type buildFunc struct {
	StatelessWidget
	build func(bc BuildContext) Widget
}

func newBuildFunc(build func(BuildContext) Widget, opts ...WidgetOpts) *buildFunc {
	return &buildFunc{NewStatelessWidgetBase(opts...), build}
}

func (w *buildFunc) Build(bc BuildContext) Widget { return w.build(bc) }

// NewMediaQueryWithNoTextScaling creates a MediaQuery that is like the one above
// but does not scale the text.
//
// Flutter: widgets/media_query.dart MediaQuery.withNoTextScaling
func NewMediaQueryWithNoTextScaling(child Widget, opts ...WidgetOpts) Widget {
	return newBuildFunc(func(bc BuildContext) Widget {
		DebugCheckHasMediaQuery(bc)
		return NewMediaQuery(MediaQueryOf(bc).CopyWith(MediaQueryDataCopyOpts{
			TextScaler: foundation.Some(painting.TextScalerNoScaling)}), child)
	}, opts...)
}

// MediaQueryClampedTextScalingOpts are the optional parameters of
// NewMediaQueryWithClampedTextScaling.
//
// Flutter: widgets/media_query.dart MediaQuery.withClampedTextScaling
type MediaQueryClampedTextScalingOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// MinScaleFactor is the smallest scale factor; the default is 0.
	MinScaleFactor float64
	// MaxScaleFactor is the biggest scale factor; 0 means no limit.
	MaxScaleFactor float64
}

// NewMediaQueryWithClampedTextScaling creates a MediaQuery that is like the one
// above but limits the text scaling to the factors in the options.
//
// Flutter: widgets/media_query.dart MediaQuery.withClampedTextScaling
func NewMediaQueryWithClampedTextScaling(child Widget, opts ...MediaQueryClampedTextScalingOpts) Widget {
	o := firstOpts(opts)
	maxScale := o.MaxScaleFactor
	if maxScale == 0 {
		maxScale = math.Inf(1)
	}
	if foundation.DebugMode && (maxScale < o.MinScaleFactor || math.IsNaN(maxScale) || math.IsInf(o.MinScaleFactor, 0) || o.MinScaleFactor < 0) {
		panic(errors.New("MediaQueryClampInvalid", "the text scale factor limits of MediaQuery.withClampedTextScaling must satisfy 0 <= min <= max",
			"min", o.MinScaleFactor, "max", maxScale))
	}
	return newBuildFunc(func(bc BuildContext) Widget {
		DebugCheckHasMediaQuery(bc)
		data := MediaQueryOf(bc)
		return NewMediaQuery(data.CopyWith(MediaQueryDataCopyOpts{TextScaler: foundation.Some(
			data.TextScaler.Clamp(painting.TextScalerClampOpts{MinScaleFactor: o.MinScaleFactor, MaxScaleFactor: foundation.Some(maxScale)}))}), child)
	}, WidgetOpts{Key: o.Key})
}

// NewMediaQueryFromView creates a widget that makes a MediaQuery from the
// view and keeps it up to date as the metrics and the platform settings of the
// view change. View uses it. If there is a MediaQuery above, its settings
// (brightness, text scaler, accessibility) win over the platform's.
//
// Flutter: widgets/media_query.dart MediaQuery.fromView
func NewMediaQueryFromView(view *ui.View, child Widget, opts ...WidgetOpts) Widget {
	return &mediaQueryFromView{StatefulWidget: NewStatefulWidgetBase(opts...), view: view, child: child}
}

// mediaQueryFromView is the widget behind MediaQuery.fromView.
//
// Flutter: widgets/media_query.dart _MediaQueryFromView
type mediaQueryFromView struct {
	StatefulWidget
	view  *ui.View
	child Widget
}

func (w *mediaQueryFromView) CreateState() State { return &mediaQueryFromViewState{} }

type mediaQueryFromViewState struct {
	StateOf[*mediaQueryFromView]
	WidgetsBindingObserverBase
	parentData *MediaQueryData
	data       *MediaQueryData
}

func (s *mediaQueryFromViewState) InitState() {
	WidgetsBindingOf(s.Context()).AddObserver(s)
}

func (s *mediaQueryFromViewState) Dispose() {
	if b := MaybeWidgetsBindingOf(s.Context()); b != nil {
		b.RemoveObserver(s)
	}
}

func (s *mediaQueryFromViewState) DidChangeDependencies() {
	s.updateParentData()
	s.updateData(true)
}

func (s *mediaQueryFromViewState) DidUpdateWidget(old *mediaQueryFromView) {
	if s.data == nil || old.view != s.Widget().view {
		s.updateData(true)
	}
}

func (s *mediaQueryFromViewState) updateParentData() {
	s.parentData = nil
	if d, ok := MediaQueryMaybeOf(s.Context()); ok {
		s.parentData = &d
	}
	s.data = nil // updateData must be called again after changing the parent data
}

// updateData makes the data from the view. During a build (inBuild) the new
// data is stored directly, because the build follows; otherwise SetState marks
// the element dirty.
func (s *mediaQueryFromViewState) updateData(inBuild bool) {
	newData := NewMediaQueryDataFromView(s.Widget().view, s.parentData)
	if s.data != nil && newData.Equal(*s.data) {
		return
	}
	if inBuild {
		s.data = &newData
		return
	}
	s.SetState(func() { s.data = &newData })
}

func (s *mediaQueryFromViewState) DidChangeAccessibilityFeatures() {
	if s.parentData == nil {
		s.updateData(false)
	}
}

func (s *mediaQueryFromViewState) DidChangeMetrics() { s.updateData(false) }

func (s *mediaQueryFromViewState) DidChangeTextScaleFactor() {
	if s.parentData == nil {
		s.updateData(false)
	}
}

func (s *mediaQueryFromViewState) DidChangePlatformBrightness() {
	if s.parentData == nil {
		s.updateData(false)
	}
}

func (s *mediaQueryFromViewState) Build(bc BuildContext) Widget {
	effective := *s.data
	if foundation.DebugMode && s.parentData == nil {
		if o := WidgetsBindingOf(bc).DebugOptions().BrightnessOverride; o != ui.BrightnessUnspecified && effective.PlatformBrightness != o {
			effective = effective.CopyWith(MediaQueryDataCopyOpts{PlatformBrightness: foundation.Some(o)})
		}
	}
	return NewMediaQuery(effective, s.Widget().child)
}
