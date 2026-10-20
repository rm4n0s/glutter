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

// BottomSheetThemeData defines default property values for descendant BottomSheet widgets. Every
// field that is not set leaves the choice to the widget's own defaults.
//
// Flutter: material/bottom_sheet_theme.dart BottomSheetThemeData
//
//glutter:copywith value
type BottomSheetThemeData struct {
	// BackgroundColor is the color of the bottom sheet.
	BackgroundColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the bottom sheet (Material 3).
	SurfaceTintColor foundation.Opt[ui.Color]
	// Elevation is the elevation of the bottom sheet.
	Elevation foundation.Opt[float64]
	// ModalBackgroundColor is the color of a modal bottom sheet.
	ModalBackgroundColor foundation.Opt[ui.Color]
	// ModalBarrierColor is the color of the barrier of a modal bottom sheet.
	ModalBarrierColor foundation.Opt[ui.Color]
	// ShadowColor is the color of the shadow below the bottom sheet.
	ShadowColor foundation.Opt[ui.Color]
	// ModalElevation is the elevation of a modal bottom sheet.
	ModalElevation foundation.Opt[float64]
	// Shape is the shape of the bottom sheet.
	Shape painting.ShapeBorder
	// ShowDragHandle says whether a bottom sheet shows a drag handle.
	ShowDragHandle foundation.Opt[bool]
	// DragHandleColor is the color of the drag handle.
	DragHandleColor widgets.WidgetStateProperty[ui.Color]
	// DragHandleSize is the size of the drag handle.
	DragHandleSize foundation.Opt[ui.Size]
	// ClipBehavior is the clip of the bottom sheet's Material.
	ClipBehavior ui.Clip
	// Constraints are the constraints of the bottom sheet.
	Constraints foundation.Opt[rendering.BoxConstraints]
}

// BottomSheetThemeDataOpts are the parameters of NewBottomSheetThemeData: the fields of the data.
//
// Flutter: material/bottom_sheet_theme.dart BottomSheetThemeData.new
type BottomSheetThemeDataOpts = BottomSheetThemeData

// NewBottomSheetThemeData creates the data; every field is optional.
//
// Flutter: material/bottom_sheet_theme.dart BottomSheetThemeData.new
func NewBottomSheetThemeData(opts ...BottomSheetThemeDataOpts) BottomSheetThemeData {
	return firstOpts(opts)
}

// BottomSheetThemeDataLerp linearly interpolates between two themes.
//
// Flutter: material/bottom_sheet_theme.dart BottomSheetThemeData.lerp
func BottomSheetThemeDataLerp(a, b BottomSheetThemeData, t float64) BottomSheetThemeData {
	return BottomSheetThemeData{
		BackgroundColor:      lerpOptColor(a.BackgroundColor, b.BackgroundColor, t),
		SurfaceTintColor:     lerpOptColor(a.SurfaceTintColor, b.SurfaceTintColor, t),
		Elevation:            lerpOptDouble(a.Elevation, b.Elevation, t),
		ModalBackgroundColor: lerpOptColor(a.ModalBackgroundColor, b.ModalBackgroundColor, t),
		ModalBarrierColor:    lerpOptColor(a.ModalBarrierColor, b.ModalBarrierColor, t),
		ShadowColor:          lerpOptColor(a.ShadowColor, b.ShadowColor, t),
		ModalElevation:       lerpOptDouble(a.ModalElevation, b.ModalElevation, t),
		Shape:                painting.ShapeBorderLerp(a.Shape, b.Shape, t),
		ShowDragHandle:       lerpFirstOrSecond(a.ShowDragHandle, b.ShowDragHandle, t),
		DragHandleColor:      lerpStateColor(a.DragHandleColor, b.DragHandleColor, t),
		DragHandleSize:       lerpOptSize(a.DragHandleSize, b.DragHandleSize, t),
		ClipBehavior:         lerpFirstOrSecond(a.ClipBehavior, b.ClipBehavior, t),
		Constraints:          lerpOptConstraints(a.Constraints, b.Constraints, t),
	}
}

// Equal reports whether the two themes have equal fields.
//
// Flutter: material/bottom_sheet_theme.dart BottomSheetThemeData.==
func (d BottomSheetThemeData) Equal(o BottomSheetThemeData) bool {
	return d.BackgroundColor == o.BackgroundColor &&
		d.SurfaceTintColor == o.SurfaceTintColor &&
		d.Elevation == o.Elevation &&
		d.ModalBackgroundColor == o.ModalBackgroundColor &&
		d.ModalBarrierColor == o.ModalBarrierColor &&
		d.ShadowColor == o.ShadowColor &&
		d.ModalElevation == o.ModalElevation &&
		foundation.ValuesEqual(d.Shape, o.Shape) &&
		d.ShowDragHandle == o.ShowDragHandle &&
		foundation.ValuesEqual(d.DragHandleColor, o.DragHandleColor) &&
		d.DragHandleSize == o.DragHandleSize &&
		d.ClipBehavior == o.ClipBehavior &&
		d.Constraints == o.Constraints
}

// ToStringShort returns the name of the type.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func (BottomSheetThemeData) ToStringShort() string { return "BottomSheetThemeData" }

// String returns the description of the theme.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func (d BottomSheetThemeData) String() string { return foundation.DiagnosticableString(d) }

// DebugFillProperties adds the properties of the theme.
//
// Flutter: material/bottom_sheet_theme.dart BottomSheetThemeData.debugFillProperties
func (d BottomSheetThemeData) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	p.Add(colorProp("backgroundColor", d.BackgroundColor))
	p.Add(colorProp("surfaceTintColor", d.SurfaceTintColor))
	p.Add(doubleProp("elevation", d.Elevation))
	p.Add(colorProp("modalBackgroundColor", d.ModalBackgroundColor))
	p.Add(colorProp("modalBarrierColor", d.ModalBarrierColor))
	p.Add(colorProp("shadowColor", d.ShadowColor))
	p.Add(doubleProp("modalElevation", d.ModalElevation))
	p.Add(objectProp("shape", d.Shape))
	p.Add(objectProp("showDragHandle", optValue(d.ShowDragHandle)))
	p.Add(objectProp("dragHandleColor", propOrNil(d.DragHandleColor)))
	p.Add(objectProp("dragHandleSize", optValue(d.DragHandleSize)))
	p.Add(objectProp("clipBehavior", nilIfZero(d.ClipBehavior)))
	p.Add(objectProp("constraints", optValue(d.Constraints)))
}
