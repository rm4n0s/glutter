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
	"time"
)

// TooltipThemeData defines default property values for descendant Tooltip widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/tooltip_theme.dart TooltipThemeData
//
//glutter:copywith value
type TooltipThemeData struct {
	// Constraints are the size constraints of the tooltip.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Padding is the padding between the message and the edge of the tooltip.
	Padding painting.EdgeInsetsGeometry
	// Margin is the empty space around the tooltip.
	Margin painting.EdgeInsetsGeometry
	// VerticalOffset is the vertical gap between the widget and the tooltip.
	VerticalOffset foundation.Opt[float64]
	// PreferBelow is whether the tooltip is below the widget when there is room for it.
	PreferBelow foundation.Opt[bool]
	// ExcludeFromSemantics is whether the tooltip's message is left out of the semantics tree.
	ExcludeFromSemantics foundation.Opt[bool]
	// Decoration is the decoration of the tooltip.
	Decoration painting.Decoration
	// TextStyle is the style of the message.
	TextStyle *painting.TextStyle
	// TextAlign is the alignment of the message.
	TextAlign ui.TextAlign
	// WaitDuration is how long a pointer hovers before the tooltip shows. Flutter's lerp leaves it out.
	WaitDuration foundation.Opt[time.Duration]
	// ShowDuration is how long the tooltip stays after a touch trigger. Flutter's lerp leaves it out.
	ShowDuration foundation.Opt[time.Duration]
	// ExitDuration is how long the tooltip stays after the pointer leaves. Flutter's lerp leaves it out.
	ExitDuration foundation.Opt[time.Duration]
	// TriggerMode is the gesture that shows the tooltip. Flutter's lerp leaves it out.
	TriggerMode widgets.TooltipTriggerMode
	// EnableFeedback is whether the tooltip gives feedback to a gesture that shows it. Flutter's lerp leaves it out.
	EnableFeedback foundation.Opt[bool]
}

// TooltipThemeDataOpts are the parameters of NewTooltipThemeData: the fields of the data.
//
// Flutter: material/tooltip_theme.dart TooltipThemeData.new
type TooltipThemeDataOpts = TooltipThemeData

// NewTooltipThemeData creates the data; every field is optional.
//
// Flutter: material/tooltip_theme.dart TooltipThemeData.new
func NewTooltipThemeData(opts ...TooltipThemeDataOpts) TooltipThemeData {
	return firstOpts(opts)
}

// TooltipThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/tooltip_theme.dart TooltipThemeData.lerp
func TooltipThemeDataLerp(a, b TooltipThemeData, t float64) TooltipThemeData {
	return TooltipThemeData{
		Constraints:          lerpOptConstraints(a.Constraints, b.Constraints, t),
		Padding:              painting.EdgeInsetsGeometryLerp(a.Padding, b.Padding, t),
		Margin:               painting.EdgeInsetsGeometryLerp(a.Margin, b.Margin, t),
		VerticalOffset:       lerpOptDouble(a.VerticalOffset, b.VerticalOffset, t),
		PreferBelow:          lerpFirstOrSecond(a.PreferBelow, b.PreferBelow, t),
		ExcludeFromSemantics: lerpFirstOrSecond(a.ExcludeFromSemantics, b.ExcludeFromSemantics, t),
		Decoration:           painting.DecorationLerp(a.Decoration, b.Decoration, t),
		TextStyle:            painting.TextStyleLerp(a.TextStyle, b.TextStyle, t),
		TextAlign:            lerpFirstOrSecond(a.TextAlign, b.TextAlign, t),
		WaitDuration:         foundation.None[time.Duration](),
		ShowDuration:         foundation.None[time.Duration](),
		ExitDuration:         foundation.None[time.Duration](),
		TriggerMode:          widgets.TooltipTriggerModeUnspecified,
		EnableFeedback:       foundation.None[bool](),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/tooltip_theme.dart TooltipThemeData.==
func (d TooltipThemeData) Equal(o TooltipThemeData) bool {
	return d.Constraints == o.Constraints &&
		foundation.ValuesEqual(d.Padding, o.Padding) &&
		foundation.ValuesEqual(d.Margin, o.Margin) &&
		d.VerticalOffset == o.VerticalOffset &&
		d.PreferBelow == o.PreferBelow &&
		d.ExcludeFromSemantics == o.ExcludeFromSemantics &&
		foundation.ValuesEqual(d.Decoration, o.Decoration) &&
		textStyleEqual(d.TextStyle, o.TextStyle) &&
		d.TextAlign == o.TextAlign &&
		d.WaitDuration == o.WaitDuration &&
		d.ShowDuration == o.ShowDuration &&
		d.ExitDuration == o.ExitDuration &&
		d.TriggerMode == o.TriggerMode &&
		d.EnableFeedback == o.EnableFeedback
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (TooltipThemeData) ToStringShort() string { return "TooltipThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d TooltipThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/tooltip_theme.dart TooltipThemeData.debugFillProperties
func (d TooltipThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(objectProp("constraints", optValue(d.Constraints)))
	p.Add(objectProp("padding", d.Padding))
	p.Add(objectProp("margin", d.Margin))
	p.Add(doubleProp("verticalOffset", d.VerticalOffset))
	p.Add(objectProp("preferBelow", optValue(d.PreferBelow)))
	p.Add(objectProp("excludeFromSemantics", optValue(d.ExcludeFromSemantics)))
	p.Add(objectProp("decoration", d.Decoration))
	p.Add(objectProp("textStyle", d.TextStyle))
	p.Add(objectProp("textAlign", nilIfZero(d.TextAlign)))
	p.Add(objectProp("waitDuration", optValue(d.WaitDuration)))
	p.Add(objectProp("showDuration", optValue(d.ShowDuration)))
	p.Add(objectProp("exitDuration", optValue(d.ExitDuration)))
	p.Add(objectProp("triggerMode", nilIfZero(d.TriggerMode)))
	p.Add(objectProp("enableFeedback", optValue(d.EnableFeedback)))
}

// TooltipTheme is an inherited theme that overrides the TooltipThemeData of the theme for the
// widgets below it.
//
// Flutter: material/tooltip_theme.dart TooltipTheme
type TooltipTheme struct {
	widgets.InheritedTheme
	data TooltipThemeData
}

// NewTooltipTheme creates the theme.
//
// Flutter: material/tooltip_theme.dart TooltipTheme.new
func NewTooltipTheme(data TooltipThemeData, child widgets.Widget, opts ...widgets.WidgetOpts) *TooltipTheme {
	return &TooltipTheme{InheritedTheme: widgets.NewInheritedThemeBase(child, opts...), data: data}
}

// Data is the data of the theme.
//
// Flutter: material/tooltip_theme.dart TooltipTheme.data
func (w *TooltipTheme) Data() TooltipThemeData { return w.data }

// UpdateShouldNotify is whether the data changed.
//
// Flutter: material/tooltip_theme.dart TooltipTheme.updateShouldNotify
func (w *TooltipTheme) UpdateShouldNotify(old widgets.Widget) bool {
	return !w.data.Equal(old.(*TooltipTheme).data)
}

// Wrap makes a TooltipTheme with the same data around the child.
//
// Flutter: material/tooltip_theme.dart TooltipTheme.wrap
func (w *TooltipTheme) Wrap(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
	return NewTooltipTheme(w.data, child)
}

// TooltipThemeOf returns the data of the closest TooltipTheme above the context, or the
// tooltipTheme of the closest Theme. The context depends on it.
//
// Flutter: material/tooltip_theme.dart TooltipTheme.of
func TooltipThemeOf(bc widgets.BuildContext) TooltipThemeData {
	if t, ok := widgets.DependOn[*TooltipTheme](bc); ok {
		return t.data
	}
	return ThemeOf(bc).TooltipTheme
}
