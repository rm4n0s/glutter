// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// RawMaterialButtonOpts are the optional parameters of NewRawMaterialButton.
// The elevations, the constraints, the shape and the animation duration have the
// defaults of Flutter when they are unset (Opt) or nil.
//
// Flutter: material/button.dart RawMaterialButton.new
type RawMaterialButtonOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnLongPress is called when the button is long-pressed.
	OnLongPress func()
	// OnHighlightChanged is called by the ink well when a highlight is added or removed.
	OnHighlightChanged func(value bool)
	// MouseCursor is the cursor over the button; it may be a WidgetStateMouseCursor.
	MouseCursor services.MouseCursor
	// TextStyle is the style of the text of the button; its color may be a
	// WidgetStateColor.
	TextStyle *painting.TextStyle
	// FillColor is the color of the button's Material.
	FillColor foundation.Opt[ui.Color]
	// FocusColor is the color of the ink well when the button has the focus.
	FocusColor foundation.Opt[ui.Color]
	// HoverColor is the color of the ink well when a pointer hovers over the button.
	HoverColor foundation.Opt[ui.Color]
	// HighlightColor is the highlight color of the ink well.
	HighlightColor foundation.Opt[ui.Color]
	// SplashColor is the splash color of the ink well.
	SplashColor foundation.Opt[ui.Color]
	// Elevation is the elevation at rest; the default is 2.
	Elevation foundation.Opt[float64]
	// FocusElevation is the elevation when the button has the focus; the default is 4.
	FocusElevation foundation.Opt[float64]
	// HoverElevation is the elevation when a pointer hovers over it; the default is 4.
	HoverElevation foundation.Opt[float64]
	// HighlightElevation is the elevation while the button is pressed; the default is 8.
	HighlightElevation foundation.Opt[float64]
	// DisabledElevation is the elevation when the button is disabled; the default is 0.
	DisabledElevation foundation.Opt[float64]
	// Padding is the padding of the child; the default is zero.
	Padding painting.EdgeInsetsGeometry
	// VisualDensity defines how compact the layout is; the default is standard.
	VisualDensity foundation.Opt[VisualDensity]
	// Constraints are the size constraints of the button; the default is a
	// minimum of 88 by 36.
	Constraints foundation.Opt[rendering.BoxConstraints]
	// Shape is the shape of the button; the default is a rounded rectangle
	// without radius.
	Shape painting.ShapeBorder
	// AnimationDuration is the duration of the changes of the shape and the
	// elevation; the default is ThemeChangeDuration.
	AnimationDuration foundation.Opt[time.Duration]
	// ClipBehavior is how the content is clipped.
	ClipBehavior ui.Clip
	// FocusNode is the focus node of the button.
	FocusNode *widgets.FocusNode
	// Autofocus says that the button focuses itself when it is built.
	Autofocus bool
	// MaterialTapTargetSize is the size of the tap target; the default is padded.
	MaterialTapTargetSize MaterialTapTargetSize
	// NoEnableFeedback says that the gestures give no acoustic and haptic
	// feedback (Flutter's enableFeedback false).
	NoEnableFeedback bool
}

// RawMaterialButton is the basic Material button: a Material with an ink well
// and the elevations of its states. It is what the floating action button is
// made from.
//
// Flutter: material/button.dart RawMaterialButton
type RawMaterialButton struct {
	widgets.StatefulWidget
	onPressed func()
	child     widgets.Widget
	o         RawMaterialButtonOpts
}

// NewRawMaterialButton creates the button. A nil onPressed (without
// OnLongPress) disables it.
//
// Flutter: material/button.dart RawMaterialButton.new
func NewRawMaterialButton(onPressed func(), child widgets.Widget, opts ...RawMaterialButtonOpts) *RawMaterialButton {
	o := firstOpts(opts)
	if foundation.DebugMode {
		for _, e := range []struct {
			name string
			v    foundation.Opt[float64]
		}{{"elevation", o.Elevation}, {"focusElevation", o.FocusElevation}, {"hoverElevation", o.HoverElevation},
			{"highlightElevation", o.HighlightElevation}, {"disabledElevation", o.DisabledElevation}} {
			if v, ok := e.v.Get(); ok && !(v >= 0.0) {
				panic(errors.New(tagRawMaterialButtonElevationNegative, "an elevation of a RawMaterialButton must not be negative", "elevation", e.name, "value", v))
			}
		}
	}
	return &RawMaterialButton{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), onPressed: onPressed, child: child, o: o}
}

// OnPressed is called when the button is tapped; nil (without OnLongPress)
// disables the button.
//
// Flutter: material/button.dart RawMaterialButton.onPressed
func (w *RawMaterialButton) OnPressed() func() { return w.onPressed }

// Child is the widget in the button.
//
// Flutter: material/button.dart RawMaterialButton.child
func (w *RawMaterialButton) Child() widgets.Widget { return w.child }

// Enabled says whether the button can be pressed.
//
// Flutter: material/button.dart RawMaterialButton.enabled
func (w *RawMaterialButton) Enabled() bool { return w.onPressed != nil || w.o.OnLongPress != nil }

// CreateState creates the state.
//
// Flutter: material/button.dart RawMaterialButton.createState
func (w *RawMaterialButton) CreateState() widgets.State { return &rawMaterialButtonState{} }

type rawMaterialButtonState struct {
	widgets.StateOf[*RawMaterialButton]
	states widgets.WidgetStates
}

// setMaterialState adds or removes the state, rebuilding when it changed.
//
// Flutter: material/material_state_mixin.dart MaterialStateMixin.setMaterialState
func (s *rawMaterialButtonState) setMaterialState(state widgets.WidgetState, isSet bool) {
	next := s.states.Without(state)
	if isSet {
		next = s.states.With(state)
	}
	if next != s.states {
		s.SetState(func() { s.states = next })
	}
}

// updateMaterialState gives a callback for the state that sets it, and calls
// onChanged after.
//
// Flutter: material/material_state_mixin.dart MaterialStateMixin.updateMaterialState
func (s *rawMaterialButtonState) updateMaterialState(state widgets.WidgetState, onChanged func(bool)) func(bool) {
	return func(value bool) {
		s.setMaterialState(state, value)
		if onChanged != nil {
			onChanged(value)
		}
	}
}

func (s *rawMaterialButtonState) InitState() {
	s.states = s.states.With(widgets.WidgetStateDisabled)
	if s.Widget().Enabled() {
		s.states = s.states.Without(widgets.WidgetStateDisabled)
	}
}

func (s *rawMaterialButtonState) DidUpdateWidget(old *RawMaterialButton) {
	s.setMaterialState(widgets.WidgetStateDisabled, !s.Widget().Enabled())
	if s.states.Has(widgets.WidgetStateDisabled) && s.states.Has(widgets.WidgetStatePressed) {
		s.states = s.states.Without(widgets.WidgetStatePressed)
	}
}

func (s *rawMaterialButtonState) effectiveElevation() float64 {
	o := s.Widget().o
	switch {
	case s.states.Has(widgets.WidgetStateDisabled):
		return o.DisabledElevation.OrElse(0.0)
	case s.states.Has(widgets.WidgetStatePressed):
		return o.HighlightElevation.OrElse(8.0)
	case s.states.Has(widgets.WidgetStateHovered):
		return o.HoverElevation.OrElse(4.0)
	case s.states.Has(widgets.WidgetStateFocused):
		return o.FocusElevation.OrElse(4.0)
	}
	return o.Elevation.OrElse(2.0)
}

func (s *rawMaterialButtonState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	theme := ThemeOf(bc)

	var effectiveTextColor foundation.Opt[ui.Color]
	if o.TextStyle != nil {
		effectiveTextColor = o.TextStyle.Color
	}
	var effectiveShape painting.ShapeBorder = painting.NewRoundedRectangleBorder()
	if o.Shape != nil {
		effectiveShape = o.Shape
	}
	visualDensity := o.VisualDensity.OrElse(VisualDensityStandard())
	densityAdjustment := visualDensity.BaseSizeAdjustment()
	effectiveConstraints := visualDensity.EffectiveConstraints(o.Constraints.OrElse(
		rendering.BoxConstraints{MinWidth: 88.0, MinHeight: 36.0, MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)}))
	var mouseCursor services.MouseCursor = widgets.WidgetStateMouseCursorAdaptiveClickable
	if o.MouseCursor != nil {
		mouseCursor = o.MouseCursor
	}
	effectiveMouseCursor := widgets.WidgetStateResolveAs[services.MouseCursor](mouseCursor, s.states)
	var basePadding painting.EdgeInsetsGeometry = painting.EdgeInsetsZero
	if o.Padding != nil {
		basePadding = o.Padding
	}
	padding := basePadding.Add(painting.NewEdgeInsetsFromLTRB(densityAdjustment.Dx, densityAdjustment.Dy, densityAdjustment.Dx, densityAdjustment.Dy)).
		Clamp(painting.EdgeInsetsZero, painting.EdgeInsetsGeometryInfinity)

	materialType := MaterialTypeButton
	if o.FillColor.IsNone() {
		materialType = MaterialTypeTransparency
	}
	var shadowColor foundation.Opt[ui.Color]
	if theme.UseMaterial3 {
		shadowColor = foundation.Some(theme.ShadowColor)
	}
	var textStyle *painting.TextStyle
	if o.TextStyle != nil {
		textStyle = o.TextStyle.CopyWith(painting.TextStyleCopy{Color: effectiveTextColor})
	}
	result := widgets.NewConstrainedBox(effectiveConstraints,
		NewMaterial(
			NewInkWell(
				widgets.NewIconThemeMerge(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: effectiveTextColor}),
					widgets.NewPadding(padding, widgets.NewCenter(w.child, widgets.CenterOpts{WidthFactor: foundation.Some(1.0), HeightFactor: foundation.Some(1.0)}))),
				InkWellOpts{
					FocusNode:          o.FocusNode,
					CanRequestFocus:    foundation.Some(w.Enabled()),
					OnFocusChange:      s.updateMaterialState(widgets.WidgetStateFocused, nil),
					Autofocus:          o.Autofocus,
					OnHighlightChanged: s.updateMaterialState(widgets.WidgetStatePressed, o.OnHighlightChanged),
					SplashColor:        o.SplashColor,
					HighlightColor:     o.HighlightColor,
					FocusColor:         o.FocusColor,
					HoverColor:         o.HoverColor,
					OnHover:            s.updateMaterialState(widgets.WidgetStateHovered, nil),
					OnTap:              w.onPressed,
					OnLongPress:        o.OnLongPress,
					EnableFeedback:     foundation.Some(!o.NoEnableFeedback),
					CustomBorder:       effectiveShape,
					MouseCursor:        effectiveMouseCursor,
				}),
			MaterialOpts{
				Elevation:         s.effectiveElevation(),
				TextStyle:         textStyle,
				Shape:             effectiveShape,
				Color:             o.FillColor,
				ShadowColor:       shadowColor,
				Type:              materialType,
				AnimationDuration: o.AnimationDuration.OrElse(ThemeChangeDuration),
				ClipBehavior:      o.ClipBehavior,
			}))

	minSize := ui.Size{}
	if o.MaterialTapTargetSize != MaterialTapTargetSizeShrinkWrap {
		minSize = ui.Size{Width: MinInteractiveDimension + densityAdjustment.Dx, Height: MinInteractiveDimension + densityAdjustment.Dy}
	}
	return widgets.NewSemantics(newInputPadding(minSize, result),
		widgets.SemanticsOpts{Container: true, Button: foundation.Some(true), Enabled: foundation.Some(w.Enabled())})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagRawMaterialButtonElevationNegative = "RawMaterialButtonElevationNegative"
)
