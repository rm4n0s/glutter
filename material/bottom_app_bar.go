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
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// BottomAppBarOpts are the optional parameters of NewBottomAppBar.
//
// Flutter: material/bottom_app_bar.dart BottomAppBar.new
type BottomAppBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Color is the background color; the default comes from the theme.
	Color foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of this bar; the default comes from the theme.
	Elevation foundation.Opt[float64]
	// Shape is the notched shape of the bar, which makes room for the floating
	// action button of the scaffold; the default comes from the theme.
	Shape painting.NotchedShape
	// ClipBehavior is whether the content is clipped and how; the default is
	// ClipNone.
	ClipBehavior ui.Clip
	// NotchMargin is the margin between the notch and the floating action button;
	// the default is 4.
	NotchMargin foundation.Opt[float64]
	// Child is the widget in the bar.
	Child widgets.Widget
	// Padding is the padding of the child; the default comes from the theme.
	Padding painting.EdgeInsetsGeometry
	// SurfaceTintColor is the color of the surface tint overlay (Material 3).
	SurfaceTintColor foundation.Opt[ui.Color]
	// ShadowColor is the color of the shadow; the default comes from the theme.
	ShadowColor foundation.Opt[ui.Color]
	// Height is the height of the bar, without the safe area; the default comes
	// from the theme.
	Height foundation.Opt[float64]
}

// BottomAppBar is a container that is typically used with Scaffold's
// bottomNavigationBar, and can have a notch for the floating action button.
//
// Flutter: material/bottom_app_bar.dart BottomAppBar
type BottomAppBar struct {
	widgets.StatefulWidget
	o BottomAppBarOpts
}

// NewBottomAppBar creates the bar. In debug builds it panics with the tag
// BottomAppBarElevationNegative when the elevation is negative.
//
// Flutter: material/bottom_app_bar.dart BottomAppBar.new
func NewBottomAppBar(opts ...BottomAppBarOpts) *BottomAppBar {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Elevation.OrElse(0) < 0 {
		panic(errors.New("BottomAppBarElevationNegative", "the elevation of a BottomAppBar must not be negative", "elevation", o.Elevation.Value()))
	}
	if o.ClipBehavior == ui.ClipUnspecified {
		o.ClipBehavior = ui.ClipNone
	}
	return &BottomAppBar{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Child is the widget in the bar.
//
// Flutter: material/bottom_app_bar.dart BottomAppBar.child
func (w *BottomAppBar) Child() widgets.Widget { return w.o.Child }

// NotchMargin is the margin between the notch and the floating action button.
//
// Flutter: material/bottom_app_bar.dart BottomAppBar.notchMargin
func (w *BottomAppBar) NotchMargin() float64 { return w.o.NotchMargin.OrElse(4.0) }

// CreateState creates the state.
//
// Flutter: material/bottom_app_bar.dart BottomAppBar.createState
func (w *BottomAppBar) CreateState() widgets.State { return &bottomAppBarState{} }

type bottomAppBarState struct {
	widgets.StateOf[*BottomAppBar]
	geometryListenable foundation.ValueListenable[ScaffoldGeometry]
	materialKey        *widgets.GlobalKey[widgets.State]
}

func (s *bottomAppBarState) InitState() { s.materialKey = widgets.NewGlobalKey[widgets.State]() }

func (s *bottomAppBarState) DidChangeDependencies() {
	s.geometryListenable = ScaffoldGeometryOf(s.Context())
}

func (s *bottomAppBarState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	theme := ThemeOf(bc)
	isMaterial3 := theme.UseMaterial3
	babTheme := BottomAppBarThemeOf(bc)
	var defaults BottomAppBarThemeData
	var defaultColor ui.Color
	var defaultSurfaceTint ui.Color
	var defaultShadow ui.Color
	if isMaterial3 {
		defaults = NewBottomAppBarThemeData(BottomAppBarThemeDataOpts{
			Elevation: foundation.Some(3.0), Height: foundation.Some(80.0),
			Shape: painting.AutomaticNotchedShape{Host: painting.NewRoundedRectangleBorder()}})
		defaultColor = theme.ColorScheme.SurfaceContainer()
		defaultSurfaceTint, defaultShadow = Colors.Transparent, Colors.Transparent
	} else {
		defaults = NewBottomAppBarThemeData(BottomAppBarThemeDataOpts{Elevation: foundation.Some(8.0)})
		if ThemeBrightnessOf(bc) == ui.BrightnessDark {
			defaultColor = Colors.Grey.Shade800()
		} else {
			defaultColor = Colors.White
		}
		defaultSurfaceTint, defaultShadow = theme.ColorScheme.SurfaceTint(), ui.NewColor(0xFF000000)
	}

	hasFab := ScaffoldOf(bc).HasFloatingActionButton()
	notchedShape := w.o.Shape
	if notchedShape == nil {
		notchedShape = babTheme.Shape
	}
	if notchedShape == nil {
		notchedShape = defaults.Shape
	}
	var clipper rendering.CustomClipper[*ui.Path]
	if notchedShape != nil && hasFab {
		clipper = &bottomAppBarClipper{
			CustomClipperBase: rendering.NewCustomClipperBase[*ui.Path](s.geometryListenable),
			geometry:          s.geometryListenable, shape: notchedShape, materialKey: s.materialKey, notchMargin: w.NotchMargin(),
		}
	} else {
		clipper = rendering.NewShapeBorderClipper(painting.NewRoundedRectangleBorder())
	}
	elevation := w.o.Elevation.OrElse(babTheme.Elevation.OrElse(defaults.Elevation.Value()))
	height := w.o.Height
	if height.IsNone() {
		height = babTheme.Height
	}
	if height.IsNone() {
		height = defaults.Height
	}
	color := w.o.Color.OrElse(babTheme.Color.OrElse(defaultColor))
	surfaceTintColor := w.o.SurfaceTintColor.OrElse(babTheme.SurfaceTintColor.OrElse(defaultSurfaceTint))
	var effectiveColor ui.Color
	if isMaterial3 {
		effectiveColor = ElevationOverlayApplySurfaceTint(color, &surfaceTintColor, elevation)
	} else {
		effectiveColor = ElevationOverlayApplyOverlay(bc, color, elevation)
	}
	shadowColor := w.o.ShadowColor.OrElse(babTheme.ShadowColor.OrElse(defaultShadow))

	padding := w.o.Padding
	if padding == nil {
		padding = babTheme.Padding
	}
	if padding == nil {
		if isMaterial3 {
			padding = painting.NewEdgeInsetsSymmetric(12.0, 16.0)
		} else {
			padding = painting.EdgeInsetsZero
		}
	}
	child := widgets.NewSizedBox(widgets.NewPadding(padding, w.o.Child), widgets.SizedBoxOpts{Height: height})
	material := NewMaterial(widgets.NewSafeArea(child), MaterialOpts{Key: s.materialKey, Type: MaterialTypeTransparency})
	return widgets.NewPhysicalShape(clipper, effectiveColor, material, widgets.PhysicalShapeOpts{
		Elevation: elevation, ShadowColor: foundation.Some(shadowColor), ClipBehavior: w.o.ClipBehavior})
}

// bottomAppBarClipper cuts the notch of the shape for the floating action button.
//
// Flutter: material/bottom_app_bar.dart _BottomAppBarClipper
type bottomAppBarClipper struct {
	rendering.CustomClipperBase[*ui.Path]
	geometry    foundation.ValueListenable[ScaffoldGeometry]
	shape       painting.NotchedShape
	materialKey *widgets.GlobalKey[widgets.State]
	notchMargin float64
}

// bottomNavigationBarTop is the top of the BottomAppBar in global coordinates:
// the scaffold's geometry when its bottomNavigationBar was specified, otherwise
// computed from the Material of the bar.
func (c *bottomAppBarClipper) bottomNavigationBarTop() float64 {
	if top, ok := c.geometry.Value().BottomNavigationBarTop.Get(); ok {
		return top
	}
	if bc := c.materialKey.CurrentContext(); bc != nil {
		if box, ok := bc.FindRenderObject().(rendering.Box); ok {
			return box.LocalToGlobal(ui.OffsetZero, nil).Dy
		}
	}
	return 0
}

func (c *bottomAppBarClipper) GetClip(size ui.Size) *ui.Path {
	// button is the floating action button's bounding rectangle in the coordinate
	// system whose origin is at the appBar's top left corner, if there is one.
	var button foundation.Opt[ui.Rect]
	if area, ok := c.geometry.Value().FloatingActionButtonArea.Get(); ok {
		button = foundation.Some(area.Translate(0.0, c.bottomNavigationBarTop()*-1.0).Inflate(c.notchMargin))
	}
	return c.shape.GetOuterPath(ui.OffsetZero.And(size), button)
}

func (c *bottomAppBarClipper) ShouldReclip(old rendering.CustomClipper[*ui.Path]) bool {
	o, ok := old.(*bottomAppBarClipper)
	return !ok || o.geometry != c.geometry || o.shape != c.shape || o.notchMargin != c.notchMargin
}
