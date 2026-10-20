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
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// defaultHeroTag is the hero tag of the floating action buttons that have no
// other.
//
// Flutter: material/floating_action_button.dart _DefaultHeroTag
type defaultHeroTag struct{ _ byte }

func (*defaultHeroTag) String() string { return "<default FloatingActionButton tag>" }

var fabDefaultHeroTag = &defaultHeroTag{}

// floatingActionButtonType is the kind of a FloatingActionButton.
//
// Flutter: material/floating_action_button.dart _FloatingActionButtonType
type floatingActionButtonType int

const (
	fabRegular floatingActionButtonType = iota
	fabSmall
	fabLarge
	fabExtended
)

// FloatingActionButtonOpts are the optional parameters of NewFloatingActionButton,
// NewFloatingActionButtonSmall and NewFloatingActionButtonLarge.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.new
type FloatingActionButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Tooltip is the text that describes the action of the button.
	Tooltip string
	// ForegroundColor is the default icon and text color; the default is the
	// on-primary-container color of the color scheme in Material 3.
	ForegroundColor foundation.Opt[ui.Color]
	// BackgroundColor is the color of the button; the default is the
	// primary-container color of the color scheme in Material 3.
	BackgroundColor foundation.Opt[ui.Color]
	// FocusColor is the color of the ink well when the button has the focus.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the color of the ink well when a pointer hovers over the button.
	HoverColor foundation.Opt[ui.Color]
	// SplashColor is the splash color of the ink well.
	SplashColor foundation.Opt[ui.Color]
	// HeroTag is the tag of the Hero that wraps the button; the default is a
	// tag that all the buttons share. Use NoHero for no hero.
	HeroTag any
	// NoHero says that the button is not wrapped in a Hero (Flutter's heroTag: null).
	NoHero bool
	// Elevation is the z-coordinate of the button at rest; the default is 6.
	Elevation foundation.Opt[float64]
	// FocusElevation is the z-coordinate when the button has the focus; the default is 6.
	FocusElevation foundation.Opt[float64]
	// HoverElevation is the z-coordinate when a pointer hovers over the button; the default is 8.
	HoverElevation foundation.Opt[float64]
	// HighlightElevation is the z-coordinate while the button is pressed; the
	// default is 6 in Material 3 and 12 in Material 2.
	HighlightElevation foundation.Opt[float64]
	// DisabledElevation is the z-coordinate when the button is disabled; the
	// default is the elevation.
	DisabledElevation foundation.Opt[float64]
	// MouseCursor is the cursor over the button; it may be a WidgetStateMouseCursor.
	MouseCursor services.MouseCursor
	// Mini says that the regular button is the small size (Flutter's mini: true).
	Mini bool
	// Shape is the shape of the button; the default is a rounded rectangle in
	// Material 3 and a circle in Material 2.
	Shape painting.ShapeBorder
	// ClipBehavior is how the content is clipped.
	ClipBehavior ui.Clip
	// FocusNode is the focus node of the button.
	FocusNode *widgets.FocusNode
	// Autofocus says that the button focuses itself when it is built.
	Autofocus bool
	// MaterialTapTargetSize configures the minimum size of the tap target; the
	// default is the one of the theme.
	MaterialTapTargetSize MaterialTapTargetSize
	// EnableFeedback says whether detected gestures give acoustic and haptic
	// feedback; the default is true.
	EnableFeedback foundation.Opt[bool]
}

// FloatingActionButtonExtendedOpts are the optional parameters of
// NewFloatingActionButtonExtended.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.extended
type FloatingActionButtonExtendedOpts struct {
	FloatingActionButtonOpts
	// Icon is the icon before the label.
	Icon widgets.Widget
	// NoExtended says that the button shows only the icon (Flutter's isExtended: false).
	NoExtended bool
	// ExtendedIconLabelSpacing is the spacing between the icon and the label.
	ExtendedIconLabelSpacing foundation.Opt[float64]
	// ExtendedPadding is the padding of the content of the extended button.
	ExtendedPadding painting.EdgeInsetsGeometry
	// ExtendedTextStyle is the style of the label of the extended button.
	ExtendedTextStyle *painting.TextStyle
}

// FloatingActionButton is a Material Design floating action button: a circular
// (in Material 2) or rounded icon button that hovers over content to promote a
// primary action in the application. It comes in regular, small, large and
// extended sizes.
//
// Flutter: material/floating_action_button.dart FloatingActionButton
type FloatingActionButton struct {
	widgets.StatelessWidget
	onPressed     func()
	child         widgets.Widget
	extendedLabel widgets.Widget
	o             FloatingActionButtonExtendedOpts
	typ           floatingActionButtonType
}

func newFloatingActionButton(onPressed func(), child widgets.Widget, typ floatingActionButtonType, o FloatingActionButtonExtendedOpts) *FloatingActionButton {
	if foundation.DebugMode {
		for _, e := range []struct {
			name string
			v    foundation.Opt[float64]
		}{{"elevation", o.Elevation}, {"focusElevation", o.FocusElevation}, {"hoverElevation", o.HoverElevation},
			{"highlightElevation", o.HighlightElevation}, {"disabledElevation", o.DisabledElevation}} {
			if v, ok := e.v.Get(); ok && !(v >= 0.0) {
				panic(errors.New("FloatingActionButtonElevationNegative", "an elevation of a FloatingActionButton must not be negative", "elevation", e.name, "value", v))
			}
		}
	}
	return &FloatingActionButton{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed: onPressed, child: child, o: o, typ: typ}
}

// NewFloatingActionButton creates a regular floating action button, or a small
// one with Mini. A nil onPressed disables the button. The child is usually an
// Icon.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.new
func NewFloatingActionButton(onPressed func(), child widgets.Widget, opts ...FloatingActionButtonOpts) *FloatingActionButton {
	o := firstOpts(opts)
	typ := fabRegular
	if o.Mini {
		typ = fabSmall
	}
	return newFloatingActionButton(onPressed, child, typ, FloatingActionButtonExtendedOpts{FloatingActionButtonOpts: o, NoExtended: true})
}

// NewFloatingActionButtonSmall creates a small floating action button: 40 by 40.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.small
func NewFloatingActionButtonSmall(onPressed func(), child widgets.Widget, opts ...FloatingActionButtonOpts) *FloatingActionButton {
	o := firstOpts(opts)
	o.Mini = true
	return newFloatingActionButton(onPressed, child, fabSmall, FloatingActionButtonExtendedOpts{FloatingActionButtonOpts: o, NoExtended: true})
}

// NewFloatingActionButtonLarge creates a large floating action button: 96 by 96.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.large
func NewFloatingActionButtonLarge(onPressed func(), child widgets.Widget, opts ...FloatingActionButtonOpts) *FloatingActionButton {
	o := firstOpts(opts)
	o.Mini = false
	return newFloatingActionButton(onPressed, child, fabLarge, FloatingActionButtonExtendedOpts{FloatingActionButtonOpts: o, NoExtended: true})
}

// NewFloatingActionButtonExtended creates an extended floating action button: a
// stadium with a label and an optional icon.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.extended
func NewFloatingActionButtonExtended(onPressed func(), label widgets.Widget, opts ...FloatingActionButtonExtendedOpts) *FloatingActionButton {
	o := firstOpts(opts)
	o.Mini = false
	b := newFloatingActionButton(onPressed, o.Icon, fabExtended, o)
	b.extendedLabel = label
	return b
}

// OnPressed is the callback that is called when the button is tapped, or nil
// when the button is disabled.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.onPressed
func (w *FloatingActionButton) OnPressed() func() { return w.onPressed }

// Child is the widget below this widget in the tree: the icon.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.child
func (w *FloatingActionButton) Child() widgets.Widget { return w.child }

// Tooltip is the text that describes the action of the button.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.tooltip
func (w *FloatingActionButton) Tooltip() string { return w.o.Tooltip }

// Mini says whether the button is small.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.mini
func (w *FloatingActionButton) Mini() bool { return w.typ == fabSmall }

// IsExtended says whether the extended button shows its label.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.isExtended
func (w *FloatingActionButton) IsExtended() bool { return w.typ == fabExtended && !w.o.NoExtended }

// fabDefaults is the default values of the floating action buttons for the theme.
//
// Flutter: material/floating_action_button.dart _FABDefaultsM2
func fabDefaults(bc widgets.BuildContext, typ floatingActionButtonType, hasChild bool) FloatingActionButtonThemeData {
	theme := ThemeOf(bc)
	colors := theme.ColorScheme
	d := FloatingActionButtonThemeData{
		Elevation:               foundation.Some(6.0),
		FocusElevation:          foundation.Some(6.0),
		HoverElevation:          foundation.Some(8.0),
		HighlightElevation:      foundation.Some(12.0),
		EnableFeedback:          foundation.Some(true),
		SizeConstraints:         foundation.Some(fabTightConstraints(56.0, 56.0)),
		SmallSizeConstraints:    foundation.Some(fabTightConstraints(40.0, 40.0)),
		LargeSizeConstraints:    foundation.Some(fabTightConstraints(96.0, 96.0)),
		ExtendedSizeConstraints: foundation.Some(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Height: foundation.Some(48.0)})),

		ExtendedIconLabelSpacing: foundation.Some(8.0),
	}
	start := 20.0
	if hasChild && typ == fabExtended {
		start = 16.0
	}
	d.ExtendedPadding = painting.EdgeInsetsDirectional{Start: start, End: 20.0}
	if !theme.UseMaterial3 {
		d.ForegroundColor = foundation.Some(colors.OnSecondary())
		d.BackgroundColor = foundation.Some(colors.Secondary())
		d.FocusColor = foundation.Some(theme.FocusColor)
		d.HoverColor = foundation.Some(theme.HoverColor)
		d.SplashColor = foundation.Some(theme.SplashColor)
		if typ == fabExtended {
			d.Shape = painting.NewStadiumBorder()
		} else {
			d.Shape = painting.NewCircleBorder()
		}
		d.IconSize = foundation.Some(24.0)
		if typ == fabLarge {
			d.IconSize = foundation.Some(36.0)
		}
		d.ExtendedTextStyle = theme.TextTheme.LabelLarge.CopyWith(painting.TextStyleCopy{LetterSpacing: foundation.Some(1.2)})
		return d
	}
	// Material 3 (_FABDefaultsM3).
	d.HighlightElevation = foundation.Some(6.0)
	d.ExtendedSizeConstraints = foundation.Some(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Height: foundation.Some(56.0)}))
	d.ForegroundColor = foundation.Some(colors.OnPrimaryContainer())
	d.BackgroundColor = foundation.Some(colors.PrimaryContainer())
	d.SplashColor = foundation.Some(colors.OnPrimaryContainer().WithOpacity(0.1))
	d.FocusColor = foundation.Some(colors.OnPrimaryContainer().WithOpacity(0.1))
	d.HoverColor = foundation.Some(colors.OnPrimaryContainer().WithOpacity(0.08))
	radius := 16.0
	switch typ {
	case fabSmall:
		radius = 12.0
	case fabLarge:
		radius = 28.0
	}
	d.Shape = painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(radius))})
	d.IconSize = foundation.Some(24.0)
	if typ == fabLarge {
		d.IconSize = foundation.Some(36.0)
	}
	d.ExtendedTextStyle = theme.TextTheme.LabelLarge
	return d
}

func fabTightConstraints(width, height float64) rendering.BoxConstraints {
	return rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(width), Height: foundation.Some(height)})
}

// Build builds the button as a RawMaterialButton, with the tooltip and the hero.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.build
func (w *FloatingActionButton) Build(bc widgets.BuildContext) widgets.Widget {
	theme := ThemeOf(bc)
	fabTheme := FloatingActionButtonThemeOf(bc)
	defaults := fabDefaults(bc, w.typ, w.child != nil)
	o := w.o

	foregroundColor := firstOpt(o.ForegroundColor, fabTheme.ForegroundColor, defaults.ForegroundColor).Value()
	backgroundColor := firstOpt(o.BackgroundColor, fabTheme.BackgroundColor, defaults.BackgroundColor).Value()
	focusColor := firstOpt(o.FocusColor, fabTheme.FocusColor, defaults.FocusColor).Value()
	hoverColor := firstOpt(o.HoverColor, fabTheme.HoverColor, defaults.HoverColor).Value()
	splashColor := firstOpt(o.SplashColor, fabTheme.SplashColor, defaults.SplashColor).Value()
	elevation := firstOpt(o.Elevation, fabTheme.Elevation, defaults.Elevation).Value()
	focusElevation := firstOpt(o.FocusElevation, fabTheme.FocusElevation, defaults.FocusElevation).Value()
	hoverElevation := firstOpt(o.HoverElevation, fabTheme.HoverElevation, defaults.HoverElevation).Value()
	disabledElevation := firstOpt(o.DisabledElevation, fabTheme.DisabledElevation, defaults.DisabledElevation, foundation.Some(elevation)).Value()
	highlightElevation := firstOpt(o.HighlightElevation, fabTheme.HighlightElevation, defaults.HighlightElevation).Value()
	materialTapTargetSize := o.MaterialTapTargetSize
	if materialTapTargetSize == MaterialTapTargetSizeUnspecified {
		materialTapTargetSize = theme.MaterialTapTargetSize
	}
	enableFeedback := firstOpt(o.EnableFeedback, fabTheme.EnableFeedback, defaults.EnableFeedback).Value()
	iconSize := firstOpt(fabTheme.IconSize, defaults.IconSize).Value()
	extendedTextStyle := firstTextStyle(o.ExtendedTextStyle, fabTheme.ExtendedTextStyle, defaults.ExtendedTextStyle).
		CopyWith(painting.TextStyleCopy{Color: foundation.Some(foregroundColor)})
	var shape painting.ShapeBorder
	for _, s := range []painting.ShapeBorder{o.Shape, fabTheme.Shape, defaults.Shape} {
		if s != nil {
			shape = s
			break
		}
	}

	var sizeConstraints rendering.BoxConstraints
	var resolvedChild widgets.Widget
	if w.child != nil {
		resolvedChild = widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(iconSize)}), w.child)
	}
	switch w.typ {
	case fabRegular:
		sizeConstraints = firstOpt(fabTheme.SizeConstraints, defaults.SizeConstraints).Value()
	case fabSmall:
		sizeConstraints = firstOpt(fabTheme.SmallSizeConstraints, defaults.SmallSizeConstraints).Value()
	case fabLarge:
		sizeConstraints = firstOpt(fabTheme.LargeSizeConstraints, defaults.LargeSizeConstraints).Value()
	case fabExtended:
		sizeConstraints = firstOpt(fabTheme.ExtendedSizeConstraints, defaults.ExtendedSizeConstraints).Value()
		iconLabelSpacing := firstOpt(o.ExtendedIconLabelSpacing, fabTheme.ExtendedIconLabelSpacing, foundation.Some(8.0)).Value()
		var padding painting.EdgeInsetsGeometry
		for _, p := range []painting.EdgeInsetsGeometry{o.ExtendedPadding, fabTheme.ExtendedPadding, defaults.ExtendedPadding} {
			if p != nil {
				padding = p
				break
			}
		}
		var row []widgets.Widget
		if w.child != nil {
			row = append(row, w.child)
		}
		if w.child != nil && w.IsExtended() {
			row = append(row, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(iconLabelSpacing)}))
		}
		if w.IsExtended() {
			row = append(row, w.extendedLabel)
		}
		resolvedChild = newChildOverflowBox(widgets.NewPadding(padding, widgets.NewRow(row, widgets.RowOpts{MainAxisSize: rendering.MainAxisSizeMin})))
	}

	var result widgets.Widget = NewRawMaterialButton(w.onPressed, resolvedChild, RawMaterialButtonOpts{
		MouseCursor: widgets.WidgetStateMouseCursorResolveWith(func(states widgets.WidgetStates) services.MouseCursor {
			if c := widgets.WidgetStateResolveAs[services.MouseCursor](o.MouseCursor, states); c != nil {
				return c
			}
			if fabTheme.MouseCursor != nil {
				if c := fabTheme.MouseCursor.Resolve(states); c != nil {
					return c
				}
			}
			return widgets.WidgetStateMouseCursorAdaptiveClickable.Resolve(states)
		}, "WidgetStateMouseCursor(FloatActionButton)"),
		Elevation:             foundation.Some(elevation),
		FocusElevation:        foundation.Some(focusElevation),
		HoverElevation:        foundation.Some(hoverElevation),
		HighlightElevation:    foundation.Some(highlightElevation),
		DisabledElevation:     foundation.Some(disabledElevation),
		Constraints:           foundation.Some(sizeConstraints),
		MaterialTapTargetSize: materialTapTargetSize,
		FillColor:             foundation.Some(backgroundColor),
		FocusColor:            foundation.Some(focusColor),
		HoverColor:            foundation.Some(hoverColor),
		SplashColor:           foundation.Some(splashColor),
		TextStyle:             extendedTextStyle,
		Shape:                 shape,
		ClipBehavior:          o.ClipBehavior,
		FocusNode:             o.FocusNode,
		Autofocus:             o.Autofocus,
		NoEnableFeedback:      !enableFeedback,
	})

	if o.Tooltip != "" {
		result = tempTooltip(o.Tooltip, result)
	}
	if !o.NoHero {
		var tag any = fabDefaultHeroTag
		if o.HeroTag != nil {
			tag = o.HeroTag
		}
		result = widgets.NewHero(tag, result)
	}
	return widgets.NewMergeSemantics(result)
}

// DebugFillProperties adds the properties of the button.
//
// Flutter: material/floating_action_button.dart FloatingActionButton.debugFillProperties
func (w *FloatingActionButton) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	var pressed any
	if w.onPressed != nil {
		pressed = true
	}
	properties.Add(foundation.NewObjectFlagProperty("onPressed", pressed, foundation.ObjectFlagPropertyOpts{IfNull: "disabled"}))
	properties.Add(foundation.NewStringProperty("tooltip", foundation.Some(w.o.Tooltip)))
	properties.Add(colorProp("foregroundColor", w.o.ForegroundColor))
	properties.Add(colorProp("backgroundColor", w.o.BackgroundColor))
	properties.Add(colorProp("focusColor", w.o.FocusColor))
	properties.Add(colorProp("hoverColor", w.o.HoverColor))
	properties.Add(colorProp("splashColor", w.o.SplashColor))
	properties.Add(doubleProp("elevation", w.o.Elevation))
	properties.Add(doubleProp("focusElevation", w.o.FocusElevation))
	properties.Add(doubleProp("hoverElevation", w.o.HoverElevation))
	properties.Add(doubleProp("highlightElevation", w.o.HighlightElevation))
	properties.Add(doubleProp("disabledElevation", w.o.DisabledElevation))
	properties.Add(objectProp("shape", w.o.Shape))
	properties.Add(objectProp("focusNode", nilIfZero(w.o.FocusNode)))
	properties.Add(foundation.NewFlagProperty("isExtended", foundation.Some(w.IsExtended()), foundation.FlagPropertyOpts{IfTrue: "extended"}))
}

// childOverflowBox lets its child have the size it wants, and overflow the
// constraints: the extended button is as wide as its content.
//
// Flutter: material/floating_action_button.dart _ChildOverflowBox
type childOverflowBox struct {
	widgets.SingleChildRenderObjectWidget
}

func newChildOverflowBox(child widgets.Widget) *childOverflowBox {
	return &childOverflowBox{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child)}
}

func (w *childOverflowBox) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	r := &renderChildOverflowBox{}
	rendering.InitRenderAligningShiftedBox(&r.RenderAligningShiftedBox, r, nil, widgets.DirectionalityOf(bc), nil)
	return r
}

func (w *childOverflowBox) UpdateRenderObject(bc widgets.BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*renderChildOverflowBox).SetTextDirection(widgets.DirectionalityOf(bc))
}

// Flutter: material/floating_action_button.dart _RenderChildOverflowBox
type renderChildOverflowBox struct {
	rendering.RenderAligningShiftedBox
}

func (r *renderChildOverflowBox) ComputeMinIntrinsicWidth(float64) float64  { return 0.0 }
func (r *renderChildOverflowBox) ComputeMinIntrinsicHeight(float64) float64 { return 0.0 }

func (r *renderChildOverflowBox) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	if child := r.Child(); child != nil {
		childSize := child.GetDryLayout(rendering.BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)})
		return ui.Size{
			Width:  math.Max(constraints.MinWidth, math.Min(constraints.MaxWidth, childSize.Width)),
			Height: math.Max(constraints.MinHeight, math.Min(constraints.MaxHeight, childSize.Height)),
		}
	}
	return constraints.Biggest()
}

func (r *renderChildOverflowBox) PerformLayout() {
	constraints := r.BoxConstraints()
	if child := r.Child(); child != nil {
		child.Layout(rendering.BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)}, rendering.LayoutOpts{ParentUsesSize: true})
		r.SetSize(ui.Size{
			Width:  math.Max(constraints.MinWidth, math.Min(constraints.MaxWidth, child.Size().Width)),
			Height: math.Max(constraints.MinHeight, math.Min(constraints.MaxHeight, child.Size().Height)),
		})
		r.AlignChild()
	} else {
		r.SetSize(constraints.Biggest())
	}
}
