// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// NavigationRailLabelType defines the behavior of the labels of a
// NavigationRail.
//
// Flutter: material/navigation_rail.dart NavigationRailLabelType
type NavigationRailLabelType int

const (
	// NavigationRailLabelTypeUnspecified is the zero value: the type of the theme,
	// or none.
	NavigationRailLabelTypeUnspecified NavigationRailLabelType = iota
	// NavigationRailLabelTypeNone shows no labels.
	//
	// Flutter: material/navigation_rail.dart NavigationRailLabelType.none
	NavigationRailLabelTypeNone
	// NavigationRailLabelTypeSelected shows only the label of the selected
	// destination.
	//
	// Flutter: material/navigation_rail.dart NavigationRailLabelType.selected
	NavigationRailLabelTypeSelected
	// NavigationRailLabelTypeAll shows all the labels.
	//
	// Flutter: material/navigation_rail.dart NavigationRailLabelType.all
	NavigationRailLabelTypeAll
)

// Name is the name of the value, as Dart's Enum.name.
func (t NavigationRailLabelType) Name() string { return enumName(int(t), "none", "selected", "all") }

// String is "NavigationRailLabelType." and the name.
func (t NavigationRailLabelType) String() string { return "NavigationRailLabelType." + t.Name() }

const (
	// Flutter: material/navigation_rail.dart _kCircularIndicatorDiameter
	railCircularIndicatorDiameter = 56.0
	// Flutter: material/navigation_rail.dart _kIndicatorHeight
	railIndicatorHeight = 32.0
	// There don't appear to be tokens for these values, but they are shown in the
	// spec.
	//
	// Flutter: material/navigation_rail.dart _horizontalDestinationPadding
	railHorizontalDestinationPadding = 8.0
	// Flutter: material/navigation_rail.dart _verticalDestinationPaddingNoLabel
	railVerticalDestinationPaddingNoLabel = 24.0
	// Flutter: material/navigation_rail.dart _verticalDestinationPaddingWithLabel
	railVerticalDestinationPaddingWithLabel = 16.0
	// Flutter: material/navigation_rail.dart _verticalIconLabelSpacingM3
	railVerticalIconLabelSpacingM3 = 4.0
	// Flutter: material/navigation_rail.dart _verticalDestinationSpacingM3
	railVerticalDestinationSpacingM3 = 12.0
	// Flutter: material/navigation_rail.dart _horizontalDestinationSpacingM3
	railHorizontalDestinationSpacingM3 = 12.0
)

// Flutter: material/navigation_rail.dart _verticalSpacer
func railVerticalSpacer() widgets.Widget {
	return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(8.0)})
}

// NavigationRailDestinationOpts are the optional parameters of
// NewNavigationRailDestination.
//
// Flutter: material/navigation_rail.dart NavigationRailDestination.new
type NavigationRailDestinationOpts struct {
	// SelectedIcon is the icon that shows while the destination is selected; the
	// default is the icon.
	SelectedIcon widgets.Widget
	// IndicatorColor is the color of the indicator of this destination.
	IndicatorColor foundation.Opt[ui.Color]
	// IndicatorShape is the shape of the indicator of this destination.
	IndicatorShape painting.ShapeBorder
	// Padding is the padding around the destination.
	Padding painting.EdgeInsetsGeometry
	// Disabled says that the destination cannot be selected.
	Disabled bool
}

// NavigationRailDestination defines the appearance of a button in a
// NavigationRail.
//
// Flutter: material/navigation_rail.dart NavigationRailDestination
type NavigationRailDestination struct {
	// Icon is the icon of the destination.
	//
	// Flutter: material/navigation_rail.dart NavigationRailDestination.icon
	Icon widgets.Widget
	// SelectedIcon is the icon that shows while the destination is selected.
	//
	// Flutter: material/navigation_rail.dart NavigationRailDestination.selectedIcon
	SelectedIcon widgets.Widget
	// IndicatorColor is the color of the indicator of this destination.
	//
	// Flutter: material/navigation_rail.dart NavigationRailDestination.indicatorColor
	IndicatorColor foundation.Opt[ui.Color]
	// IndicatorShape is the shape of the indicator of this destination.
	//
	// Flutter: material/navigation_rail.dart NavigationRailDestination.indicatorShape
	IndicatorShape painting.ShapeBorder
	// Label is the label of the destination.
	//
	// Flutter: material/navigation_rail.dart NavigationRailDestination.label
	Label widgets.Widget
	// Padding is the padding around the destination.
	//
	// Flutter: material/navigation_rail.dart NavigationRailDestination.padding
	Padding painting.EdgeInsetsGeometry
	// Disabled says that the destination cannot be selected.
	//
	// Flutter: material/navigation_rail.dart NavigationRailDestination.disabled
	Disabled bool
}

// NewNavigationRailDestination creates a destination.
//
// Flutter: material/navigation_rail.dart NavigationRailDestination.new
func NewNavigationRailDestination(icon, label widgets.Widget, opts ...NavigationRailDestinationOpts) NavigationRailDestination {
	o := firstOpts(opts)
	selected := o.SelectedIcon
	if selected == nil {
		selected = icon
	}
	return NavigationRailDestination{Icon: icon, SelectedIcon: selected, IndicatorColor: o.IndicatorColor, IndicatorShape: o.IndicatorShape,
		Label: label, Padding: o.Padding, Disabled: o.Disabled}
}

// NavigationRailOpts are the optional parameters of NewNavigationRail.
//
// Flutter: material/navigation_rail.dart NavigationRail.new
type NavigationRailOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BackgroundColor is the color of the rail.
	BackgroundColor foundation.Opt[ui.Color]
	// Extended says that the rail is wide and shows the labels next to the icons.
	Extended bool
	// Leading is the widget at the top of the rail, such as a floating action
	// button.
	Leading widgets.Widget
	// Trailing is the widget below the destinations.
	Trailing widgets.Widget
	// OnDestinationSelected is called when a destination is tapped, with its index.
	OnDestinationSelected func(index int)
	// Elevation is the z-coordinate of the rail; it must be positive.
	Elevation foundation.Opt[float64]
	// GroupAlignment is the vertical alignment of the destinations, from -1 (top)
	// to 1 (bottom).
	GroupAlignment foundation.Opt[float64]
	// LabelType is how the labels are shown.
	LabelType NavigationRailLabelType
	// UnselectedLabelTextStyle is the style of the labels of the destinations that
	// are not selected.
	UnselectedLabelTextStyle *painting.TextStyle
	// SelectedLabelTextStyle is the style of the label of the selected destination.
	SelectedLabelTextStyle *painting.TextStyle
	// UnselectedIconTheme is the theme of the icons of the destinations that are
	// not selected.
	UnselectedIconTheme foundation.Opt[widgets.IconThemeData]
	// SelectedIconTheme is the theme of the icon of the selected destination.
	SelectedIconTheme foundation.Opt[widgets.IconThemeData]
	// MinWidth is the width of the rail when it is not extended; it must be positive.
	MinWidth foundation.Opt[float64]
	// MinExtendedWidth is the width of the rail when it is extended; it must be
	// positive.
	MinExtendedWidth foundation.Opt[float64]
	// UseIndicator says whether the selected destination shows an indicator.
	UseIndicator foundation.Opt[bool]
	// IndicatorColor is the color of the indicator.
	IndicatorColor foundation.Opt[ui.Color]
	// IndicatorShape is the shape of the indicator.
	IndicatorShape painting.ShapeBorder
	// NoLeadingAtTop puts the leading widget in the group of the destinations
	// instead of above it (Flutter's leadingAtTop: false).
	NoLeadingAtTop bool
	// TrailingAtBottom puts the trailing widget at the bottom of the rail instead
	// of below the destinations.
	TrailingAtBottom bool
	// Scrollable makes the destinations scroll when they do not fit.
	Scrollable bool
	// MainAxisAlignment is how the destinations are placed vertically; it
	// overrides the group alignment when given.
	MainAxisAlignment foundation.Opt[rendering.MainAxisAlignment]
}

// NavigationRail is a Material Design widget that is meant to be displayed at the
// left or right of an app to navigate between a small number of views, typically
// between three and five.
//
// Flutter: material/navigation_rail.dart NavigationRail
type NavigationRail struct {
	widgets.StatefulWidget
	destinations  []NavigationRailDestination
	selectedIndex foundation.Opt[int]
	o             NavigationRailOpts
}

// NewNavigationRail creates a rail; selectedIndex is none for no selection. In
// debug builds it panics with the tag NavigationRailSelectedIndex,
// NavigationRailElevation, NavigationRailMinWidth, NavigationRailMinExtendedWidth,
// NavigationRailMinWidths or NavigationRailExtendedLabels when the arguments are
// not valid.
//
// Flutter: material/navigation_rail.dart NavigationRail.new
func NewNavigationRail(destinations []NavigationRailDestination, selectedIndex foundation.Opt[int], opts ...NavigationRailOpts) *NavigationRail {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if i, ok := selectedIndex.Get(); ok && (i < 0 || i >= len(destinations)) {
			panic(errors.New(tagNavigationRailSelectedIndex, "the selected index of a NavigationRail must be null or one of its destinations", "selectedIndex", i, "destinations", len(destinations)))
		}
		if v, ok := o.Elevation.Get(); ok && v <= 0 {
			panic(errors.New(tagNavigationRailElevation, "the elevation of a NavigationRail must be greater than zero", "elevation", v))
		}
		if v, ok := o.MinWidth.Get(); ok && v <= 0 {
			panic(errors.New(tagNavigationRailMinWidth, "the minimum width of a NavigationRail must be greater than zero", "minWidth", v))
		}
		if v, ok := o.MinExtendedWidth.Get(); ok && v <= 0 {
			panic(errors.New(tagNavigationRailMinExtendedWidth, "the minimum extended width of a NavigationRail must be greater than zero", "minExtendedWidth", v))
		}
		if o.MinWidth.IsSome() && o.MinExtendedWidth.IsSome() && o.MinExtendedWidth.Value() < o.MinWidth.Value() {
			panic(errors.New(tagNavigationRailMinWidths, "the minimum extended width of a NavigationRail must not be smaller than its minimum width",
				"minWidth", o.MinWidth.Value(), "minExtendedWidth", o.MinExtendedWidth.Value()))
		}
		if o.Extended && o.LabelType != NavigationRailLabelTypeUnspecified && o.LabelType != NavigationRailLabelTypeNone {
			panic(errors.New(tagNavigationRailExtendedLabels, "an extended NavigationRail can only have the label type none", "labelType", o.LabelType.Name()))
		}
	}
	return &NavigationRail{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}),
		destinations: destinations, selectedIndex: selectedIndex, o: o}
}

// Destinations are the destinations of the rail.
//
// Flutter: material/navigation_rail.dart NavigationRail.destinations
func (w *NavigationRail) Destinations() []NavigationRailDestination { return w.destinations }

// SelectedIndex is the index of the selected destination, if any.
//
// Flutter: material/navigation_rail.dart NavigationRail.selectedIndex
func (w *NavigationRail) SelectedIndex() foundation.Opt[int] { return w.selectedIndex }

// Extended is whether the rail is extended.
//
// Flutter: material/navigation_rail.dart NavigationRail.extended
func (w *NavigationRail) Extended() bool { return w.o.Extended }

// LeadingAtTop is whether the leading widget is above the group of destinations.
//
// Flutter: material/navigation_rail.dart NavigationRail.leadingAtTop
func (w *NavigationRail) LeadingAtTop() bool { return !w.o.NoLeadingAtTop }

// navigationRailExtendedAnimation is what a rail tells its destinations.
//
// Flutter: material/navigation_rail.dart _ExtendedNavigationRailAnimation
type navigationRailExtendedAnimation struct {
	widgets.InheritedWidget
	animation animation.Animation[float64]
}

func (w *navigationRailExtendedAnimation) UpdateShouldNotify(old widgets.Widget) bool {
	return w.animation != old.(*navigationRailExtendedAnimation).animation
}

// NavigationRailExtendedAnimation is the animation that controls the transition
// of the rail between narrow and extended, for the destinations and the leading
// and trailing widgets of the rail. The context depends on it.
//
// Flutter: material/navigation_rail.dart NavigationRail.extendedAnimation
func NavigationRailExtendedAnimation(bc widgets.BuildContext) animation.Animation[float64] {
	a, ok := widgets.DependOn[*navigationRailExtendedAnimation](bc)
	if !ok {
		panic(errors.New(tagNavigationRailExtendedAnimationMissing, "NavigationRailExtendedAnimation was called with a context that does not contain a NavigationRail.",
			"widget", foundation.RuntimeTypeName(bc.Widget())))
	}
	return a.animation
}

// CreateState creates the state.
//
// Flutter: material/navigation_rail.dart NavigationRail.createState
func (w *NavigationRail) CreateState() widgets.State { return &navigationRailState{} }

type navigationRailState struct {
	widgets.StateOf[*NavigationRail]
	widgets.TickerProvider
	destinationControllers []*animation.AnimationController
	extendedController     *animation.AnimationController
	extendedAnimation      *animation.CurvedAnimation
}

func (s *navigationRailState) InitState() { s.initControllers() }

func (s *navigationRailState) Dispose() { s.disposeControllers() }

func (s *navigationRailState) DidUpdateWidget(old *NavigationRail) {
	w := s.Widget()
	if w.o.Extended != old.o.Extended {
		if w.o.Extended {
			s.extendedController.Forward()
		} else {
			s.extendedController.Reverse()
		}
	}

	// No animated segue if the length of the items list changes.
	if len(w.destinations) != len(old.destinations) {
		s.resetState()
		return
	}

	if w.selectedIndex != old.selectedIndex {
		if i, ok := old.selectedIndex.Get(); ok {
			s.destinationControllers[i].Reverse()
		}
		if i, ok := w.selectedIndex.Get(); ok {
			s.destinationControllers[i].Forward()
		}
	}
}

func (s *navigationRailState) disposeControllers() {
	for _, c := range s.destinationControllers {
		c.Dispose()
	}
	s.extendedController.Dispose()
	s.extendedAnimation.Dispose()
}

func (s *navigationRailState) initControllers() {
	w := s.Widget()
	s.destinationControllers = make([]*animation.AnimationController, len(w.destinations))
	for i := range s.destinationControllers {
		c := animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(ThemeAnimationDuration)})
		c.AddListener(s.rebuild)
		s.destinationControllers[i] = c
	}
	if i, ok := w.selectedIndex.Get(); ok {
		s.destinationControllers[i].SetValue(1.0)
	}
	value := 0.0
	if w.o.Extended {
		value = 1.0
	}
	s.extendedController = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: foundation.Some(ThemeAnimationDuration), Value: foundation.Some(value)})
	s.extendedAnimation = animation.NewCurvedAnimation(s.extendedController, animation.Curves.EaseInOut)
	s.extendedController.AddListener(s.rebuild)
}

func (s *navigationRailState) resetState() {
	s.disposeControllers()
	s.initControllers()
}

// rebuild rebuilds when any of the controllers tick, that is when the items are
// animating.
func (s *navigationRailState) rebuild() { s.SetState(func() {}) }

func navigationRailDefaults(bc widgets.BuildContext) NavigationRailThemeData {
	theme := ThemeOf(bc)
	if theme.UseMaterial3 {
		return navigationRailDefaultsM3(theme)
	}
	return navigationRailDefaultsM2(theme)
}

func (s *navigationRailState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	navigationRailTheme := NavigationRailThemeOf(bc)
	theme := ThemeOf(bc)
	defaults := navigationRailDefaults(bc)
	localizations := MaterialLocalizationsOf(bc)

	backgroundColor := firstOpt(o.BackgroundColor, navigationRailTheme.BackgroundColor, defaults.BackgroundColor).Value()
	elevation := firstOpt(o.Elevation, navigationRailTheme.Elevation, defaults.Elevation).Value()
	minWidth := firstOpt(o.MinWidth, navigationRailTheme.MinWidth, defaults.MinWidth).Value()
	minExtendedWidth := firstOpt(o.MinExtendedWidth, navigationRailTheme.MinExtendedWidth, defaults.MinExtendedWidth).Value()
	unselectedLabelTextStyle := firstTextStyle(o.UnselectedLabelTextStyle, navigationRailTheme.UnselectedLabelTextStyle, defaults.UnselectedLabelTextStyle)
	selectedLabelTextStyle := firstTextStyle(o.SelectedLabelTextStyle, navigationRailTheme.SelectedLabelTextStyle, defaults.SelectedLabelTextStyle)
	unselectedIconTheme := firstOpt(o.UnselectedIconTheme, navigationRailTheme.UnselectedIconTheme, defaults.UnselectedIconTheme).Value()
	selectedIconTheme := firstOpt(o.SelectedIconTheme, navigationRailTheme.SelectedIconTheme, defaults.SelectedIconTheme).Value()
	groupAlignment := firstOpt(o.GroupAlignment, navigationRailTheme.GroupAlignment, defaults.GroupAlignment).Value()
	labelType := o.LabelType
	if labelType == NavigationRailLabelTypeUnspecified {
		labelType = navigationRailTheme.LabelType
	}
	if labelType == NavigationRailLabelTypeUnspecified {
		labelType = defaults.LabelType
	}
	useIndicator := firstOpt(o.UseIndicator, navigationRailTheme.UseIndicator, defaults.UseIndicator).Value()
	indicatorColor := firstOpt(o.IndicatorColor, navigationRailTheme.IndicatorColor, defaults.IndicatorColor)
	indicatorShape := o.IndicatorShape
	if indicatorShape == nil {
		indicatorShape = navigationRailTheme.IndicatorShape
	}
	if indicatorShape == nil {
		indicatorShape = defaults.IndicatorShape
	}

	// For backwards compatibility, in M2 the opacity of the unselected icons needs
	// to be set to the default if it isn't in the given theme. This can be removed
	// when Material 3 is the default.
	effectiveUnselectedIconTheme := unselectedIconTheme
	if !theme.UseMaterial3 {
		effectiveUnselectedIconTheme = unselectedIconTheme.CopyWith(widgets.IconThemeDataOpts{
			Opacity: firstOpt(unselectedIconTheme.Opacity(), defaults.UnselectedIconTheme.Value().Opacity())})
	}

	isRTLDirection := widgets.DirectionalityOf(bc) == ui.TextDirectionRTL

	var group []widgets.Widget
	if !w.LeadingAtTop() && o.Leading != nil {
		group = append(group, o.Leading, railVerticalSpacer())
	}
	for i, d := range w.destinations {
		selected := w.selectedIndex == foundation.Some(i)
		icon, labelStyle, iconTheme := d.Icon, unselectedLabelTextStyle, effectiveUnselectedIconTheme
		if selected {
			icon, labelStyle, iconTheme = d.SelectedIcon, selectedLabelTextStyle, selectedIconTheme
		}
		var destIndicatorColor foundation.Opt[ui.Color]
		var destIndicatorShape painting.ShapeBorder
		if useIndicator {
			destIndicatorColor, destIndicatorShape = indicatorColor, indicatorShape
		}
		i := i
		group = append(group, newRailDestination(railDestinationOpts{
			minWidth: minWidth, minExtendedWidth: minExtendedWidth, extendedTransitionAnimation: s.extendedAnimation, selected: selected,
			icon: icon, label: d.Label, destinationAnimation: s.destinationControllers[i], labelType: labelType, iconTheme: iconTheme,
			labelTextStyle: labelStyle, padding: d.Padding, useIndicator: useIndicator, indicatorColor: destIndicatorColor, indicatorShape: destIndicatorShape,
			onTap: func() {
				if o.OnDestinationSelected != nil {
					o.OnDestinationSelected(i)
				}
			},
			indexLabel: localizations.TabLabel(i+1, len(w.destinations)), disabled: d.Disabled,
		}))
	}
	if !o.TrailingAtBottom && o.Trailing != nil {
		group = append(group, o.Trailing)
	}
	columnOpts := widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin}
	if m, ok := o.MainAxisAlignment.Get(); ok {
		columnOpts = widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMax, MainAxisAlignment: m}
	}
	var mainGroup widgets.Widget = widgets.NewColumn(group, columnOpts)
	if o.Scrollable {
		mainGroup = widgets.NewSingleChildScrollView(mainGroup)
	}

	column := []widgets.Widget{railVerticalSpacer()}
	if w.LeadingAtTop() && o.Leading != nil {
		column = append(column, o.Leading, railVerticalSpacer())
	}
	column = append(column, widgets.NewFlexible(widgets.NewAlign(mainGroup, widgets.AlignOpts{Alignment: painting.Alignment{X: 0, Y: groupAlignment}})))
	if o.TrailingAtBottom && o.Trailing != nil {
		column = append(column, o.Trailing)
	}
	return widgets.NewSemantics(
		&navigationRailExtendedAnimation{
			InheritedWidget: widgets.NewInheritedWidgetBase(
				widgets.NewSemantics(
					NewMaterial(
						widgets.NewSafeArea(widgets.NewColumn(column),
							widgets.SafeAreaOpts{Right: foundation.Some(isRTLDirection), Left: foundation.Some(!isRTLDirection)}),
						MaterialOpts{Elevation: elevation, Color: foundation.Some(backgroundColor)}),
					widgets.SemanticsOpts{ExplicitChildNodes: true})),
			animation: s.extendedAnimation,
		},
		widgets.SemanticsOpts{Container: true})
}

type railDestinationOpts struct {
	minWidth, minExtendedWidth  float64
	icon, label                 widgets.Widget
	destinationAnimation        animation.Animation[float64]
	extendedTransitionAnimation animation.Animation[float64]
	labelType                   NavigationRailLabelType
	selected                    bool
	iconTheme                   widgets.IconThemeData
	labelTextStyle              *painting.TextStyle
	onTap                       func()
	indexLabel                  string
	padding                     painting.EdgeInsetsGeometry
	useIndicator                bool
	indicatorColor              foundation.Opt[ui.Color]
	indicatorShape              painting.ShapeBorder
	disabled                    bool
}

// railDestination is the button of a destination of a rail.
//
// Flutter: material/navigation_rail.dart _RailDestination
type railDestination struct {
	widgets.StatefulWidget
	o railDestinationOpts
}

func newRailDestination(o railDestinationOpts) *railDestination {
	return &railDestination{StatefulWidget: widgets.NewStatefulWidgetBase(), o: o}
}

func (w *railDestination) CreateState() widgets.State { return &railDestinationState{} }

type railDestinationState struct {
	widgets.StateOf[*railDestination]
	positionAnimation *animation.CurvedAnimation
}

func (s *railDestinationState) InitState() { s.setPositionAnimation() }

func (s *railDestinationState) DidUpdateWidget(old *railDestination) {
	if s.Widget().o.destinationAnimation != old.o.destinationAnimation {
		s.positionAnimation.Dispose()
		s.setPositionAnimation()
	}
}

func (s *railDestinationState) setPositionAnimation() {
	s.positionAnimation = animation.NewCurvedAnimation(animation.NewReverseAnimation(s.Widget().o.destinationAnimation),
		animation.Curves.EaseInOut, animation.Curves.EaseInOut.Flipped())
}

func (s *railDestinationState) Dispose() { s.positionAnimation.Dispose() }

func (s *railDestinationState) Build(bc widgets.BuildContext) widgets.Widget {
	o := s.Widget().o
	theme := ThemeOf(bc)
	textDirection := widgets.DirectionalityOf(bc)
	material3 := theme.UseMaterial3
	var padding painting.EdgeInsetsGeometry = painting.EdgeInsetsZero
	if o.padding != nil {
		padding = o.padding
	}
	destinationPadding := padding.Resolve(textDirection)
	var indicatorOffset ui.Offset
	applyXOffset := false

	iconTheme := o.iconTheme
	labelTextStyle := o.labelTextStyle
	if o.disabled {
		disabledColor := foundation.Some(theme.ColorScheme.OnSurface().WithOpacity(0.38))
		iconTheme = iconTheme.CopyWith(widgets.IconThemeDataOpts{Color: disabledColor})
		labelTextStyle = labelTextStyle.CopyWith(painting.TextStyleCopy{Color: disabledColor})
	}
	themedIcon := widgets.NewIconTheme(iconTheme, o.icon)
	styledLabel := widgets.NewDefaultTextStyle(labelTextStyle, o.label)

	var content widgets.Widget

	// The indicator height is fixed and equal to railIndicatorHeight. When the icon
	// height is larger than the indicator height the indicator vertical offset is
	// used to vertically center the indicator.
	isLargeIconSize := o.iconTheme.Size().IsSome() && o.iconTheme.Size().Value() > railIndicatorHeight
	indicatorVerticalOffset := 0.0
	if isLargeIconSize {
		indicatorVerticalOffset = (o.iconTheme.Size().Value() - railIndicatorHeight) / 2
	}

	addIndicator := func(isCircular bool, child widgets.Widget) widgets.Widget {
		return newRailAddIndicator(o.useIndicator, isCircular, o.indicatorColor, o.indicatorShape, o.destinationAnimation, child)
	}

	switch o.labelType {
	case NavigationRailLabelTypeNone:
		// Split the destination spacing across the top and bottom to keep the icon
		// centered.
		spacing := func() []widgets.Widget {
			if material3 {
				return []widgets.Widget{widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(railVerticalDestinationSpacingM3 / 2)})}
			}
			return nil
		}
		indicatorOffset = ui.Offset{Dx: o.minWidth/2 + destinationPadding.Left,
			Dy: railVerticalDestinationSpacingM3/2 + destinationPadding.Top + indicatorVerticalOffset}
		sizedHeight := foundation.Some(o.minWidth)
		if material3 {
			sizedHeight = foundation.None[float64]()
		}
		iconPart := widgets.NewColumn(append(append(spacing(),
			widgets.NewSizedBox(widgets.NewCenter(addIndicator(!material3, themedIcon)),
				widgets.SizedBoxOpts{Width: foundation.Some(o.minWidth), Height: sizedHeight})),
			spacing()...))
		if o.extendedTransitionAnimation.Value() == 0 {
			content = widgets.NewPadding(padding, widgets.NewStack([]widgets.Widget{
				iconPart,
				// For semantics when label is not showing,
				widgets.NewSizedBoxShrink(widgets.NewVisibilityMaintain(o.label, widgets.VisibilityOpts{Visible: foundation.Some(false)})),
			}))
		} else {
			labelFadeAnimation := animation.Drive[float64](o.extendedTransitionAnimation, animation.NewCurveTween(animation.NewInterval(0.0, 0.25)))
			applyXOffset = true
			content = widgets.NewPadding(padding,
				widgets.NewConstrainedBox(
					rendering.BoxConstraints{MinWidth: ui.LerpDouble(o.minWidth, o.minExtendedWidth, o.extendedTransitionAnimation.Value()), MaxWidth: math.Inf(1), MaxHeight: math.Inf(1)},
					widgets.NewClipRect(widgets.NewRow([]widgets.Widget{
						iconPart,
						widgets.NewFlexible(widgets.NewAlign(
							widgets.NewFadeTransition(labelFadeAnimation, styledLabel, widgets.FadeTransitionOpts{AlwaysIncludeSemantics: true}),
							widgets.AlignOpts{HeightFactor: foundation.Some(1.0), WidthFactor: foundation.Some(o.extendedTransitionAnimation.Value()),
								Alignment: painting.AlignmentDirectionalCenterStart})),
						widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(railHorizontalDestinationPadding * o.extendedTransitionAnimation.Value())}),
					}, widgets.RowOpts{MainAxisSize: rendering.MainAxisSizeMin}))))
		}

	case NavigationRailLabelTypeSelected:
		appearingAnimationValue := 1 - s.positionAnimation.Value()
		verticalPadding := ui.LerpDouble(railVerticalDestinationPaddingNoLabel, railVerticalDestinationPaddingWithLabel, appearingAnimationValue)
		interval := animation.NewInterval(0.75, 1.0)
		if o.selected {
			interval = animation.NewInterval(0.25, 0.75)
		}
		labelFadeAnimation := animation.Drive[float64](o.destinationAnimation, animation.NewCurveTween(interval))
		minHeight := o.minWidth
		topSpacingHeight, labelSpacingHeight, bottomSpacingHeight := verticalPadding, 0.0, verticalPadding
		if material3 {
			minHeight = 0
			topSpacingHeight = 0
			labelSpacingHeight = ui.LerpDouble(0, railVerticalIconLabelSpacingM3, appearingAnimationValue)
			bottomSpacingHeight = railVerticalDestinationSpacingM3
		}
		indicatorHorizontalPadding := destinationPadding.Left/2 - destinationPadding.Right/2
		indicatorVerticalPadding := destinationPadding.Top
		indicatorOffset = ui.Offset{Dx: o.minWidth/2 + indicatorHorizontalPadding, Dy: indicatorVerticalPadding + indicatorVerticalOffset}
		if o.minWidth < navigationRailDefaultsM2(theme).MinWidth.Value() {
			indicatorOffset = ui.Offset{Dx: o.minWidth/2 + railHorizontalDestinationSpacingM3, Dy: indicatorVerticalPadding + indicatorVerticalOffset}
		}
		contentPadding := o.padding
		if contentPadding == nil {
			contentPadding = painting.NewEdgeInsetsSymmetric(0, railHorizontalDestinationPadding)
		}
		content = widgets.NewConstrainedBox(
			rendering.BoxConstraints{MinWidth: o.minWidth, MaxWidth: math.Inf(1), MinHeight: minHeight, MaxHeight: math.Inf(1)},
			widgets.NewPadding(contentPadding, widgets.NewClipRect(widgets.NewColumn([]widgets.Widget{
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(topSpacingHeight)}),
				addIndicator(false, themedIcon),
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(labelSpacingHeight)}),
				widgets.NewAlign(
					widgets.NewFadeTransition(labelFadeAnimation, styledLabel, widgets.FadeTransitionOpts{AlwaysIncludeSemantics: true}),
					widgets.AlignOpts{Alignment: painting.AlignmentTopCenter, HeightFactor: foundation.Some(appearingAnimationValue), WidthFactor: foundation.Some(1.0)}),
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(bottomSpacingHeight)}),
			}, widgets.ColumnOpts{MainAxisSize: rendering.MainAxisSizeMin, MainAxisAlignment: rendering.MainAxisAlignmentCenter}))))

	case NavigationRailLabelTypeAll:
		minHeight := o.minWidth
		topSpacingHeight, labelSpacingHeight, bottomSpacingHeight := railVerticalDestinationPaddingWithLabel, 0.0, railVerticalDestinationPaddingWithLabel
		if material3 {
			minHeight = 0
			topSpacingHeight = 0
			labelSpacingHeight = railVerticalIconLabelSpacingM3
			bottomSpacingHeight = railVerticalDestinationSpacingM3
		}
		indicatorHorizontalPadding := destinationPadding.Left/2 - destinationPadding.Right/2
		indicatorVerticalPadding := destinationPadding.Top
		indicatorOffset = ui.Offset{Dx: o.minWidth/2 + indicatorHorizontalPadding, Dy: indicatorVerticalPadding + indicatorVerticalOffset}
		if o.minWidth < navigationRailDefaultsM2(theme).MinWidth.Value() {
			indicatorOffset = ui.Offset{Dx: o.minWidth/2 + railHorizontalDestinationSpacingM3, Dy: indicatorVerticalPadding + indicatorVerticalOffset}
		}
		contentPadding := o.padding
		if contentPadding == nil {
			contentPadding = painting.NewEdgeInsetsSymmetric(0, railHorizontalDestinationPadding)
		}
		content = widgets.NewConstrainedBox(
			rendering.BoxConstraints{MinWidth: o.minWidth, MaxWidth: math.Inf(1), MinHeight: minHeight, MaxHeight: math.Inf(1)},
			widgets.NewPadding(contentPadding, widgets.NewColumn([]widgets.Widget{
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(topSpacingHeight)}),
				addIndicator(false, themedIcon),
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(labelSpacingHeight)}),
				styledLabel,
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(bottomSpacingHeight)}),
			})))
	}

	colors := theme.ColorScheme
	primaryColorAlphaModified := colors.Primary().Alpha() < 255
	effectiveSplashColor := colors.Primary().WithOpacity(0.12)
	effectiveHoverColor := colors.Primary().WithOpacity(0.04)
	if primaryColorAlphaModified {
		effectiveSplashColor, effectiveHoverColor = colors.Primary(), colors.Primary()
	}
	var onTap func()
	if !o.disabled {
		onTap = o.onTap
	}
	inkOpts := InkResponseOpts{
		OnTap: onTap, ContainedInkWell: true, HighlightShape: painting.BoxShapeRectangle,
		SplashColor: foundation.Some(effectiveSplashColor), HoverColor: foundation.Some(effectiveHoverColor),
	}
	if material3 {
		inkOpts.CustomBorder = o.indicatorShape
		boxWidthOf := func(referenceBox rendering.Box) float64 { return referenceBox.Size().Width }
		inkOpts.GetRectCallback = func(referenceBox rendering.Box) RectCallback {
			boxWidth := boxWidthOf(referenceBox)
			indicatorHorizontalCenter := boxWidth / 2
			if applyXOffset {
				indicatorHorizontalCenter = indicatorOffset.Dx
			}
			if textDirection == ui.TextDirectionRTL {
				indicatorHorizontalCenter = boxWidth - indicatorHorizontalCenter
			}
			return func() ui.Rect {
				return ui.NewRectFromLTWH(indicatorHorizontalCenter-railCircularIndicatorDiameter/2, indicatorOffset.Dy,
					railCircularIndicatorDiameter, railIndicatorHeight)
			}
		}
	} else {
		inkOpts.BorderRadius = foundation.Some(painting.NewBorderRadiusAll(ui.NewRadiusCircular(o.minWidth / 2.0)))
	}
	return widgets.NewSemantics(
		widgets.NewStack([]widgets.Widget{
			NewMaterial(NewInkResponse(content, inkOpts), MaterialOpts{Type: MaterialTypeTransparency}),
			widgets.NewSemantics(nil, widgets.SemanticsOpts{Label: foundation.Some(o.indexLabel)}),
		}),
		widgets.SemanticsOpts{Container: true, Selected: foundation.Some(o.selected)})
}

// railAddIndicator puts the indicator of the selected destination behind its
// icon.
//
// Flutter: material/navigation_rail.dart _AddIndicator
type railAddIndicator struct {
	widgets.StatelessWidget
	addIndicator, isCircular bool
	indicatorColor           foundation.Opt[ui.Color]
	indicatorShape           painting.ShapeBorder
	indicatorAnimation       animation.Animation[float64]
	child                    widgets.Widget
}

func newRailAddIndicator(addIndicator, isCircular bool, color foundation.Opt[ui.Color], shape painting.ShapeBorder,
	a animation.Animation[float64], child widgets.Widget) *railAddIndicator {
	return &railAddIndicator{StatelessWidget: widgets.NewStatelessWidgetBase(), addIndicator: addIndicator, isCircular: isCircular,
		indicatorColor: color, indicatorShape: shape, indicatorAnimation: a, child: child}
}

func (w *railAddIndicator) Build(widgets.BuildContext) widgets.Widget {
	if !w.addIndicator {
		return w.child
	}
	var indicator widgets.Widget
	if w.isCircular {
		indicator = NewNavigationIndicator(w.indicatorAnimation, NavigationIndicatorOpts{
			Height: foundation.Some(railCircularIndicatorDiameter), Width: foundation.Some(railCircularIndicatorDiameter),
			BorderRadius: foundation.Some(painting.NewBorderRadiusAll(ui.NewRadiusCircular(railCircularIndicatorDiameter / 2))), Color: w.indicatorColor})
	} else {
		indicator = NewNavigationIndicator(w.indicatorAnimation, NavigationIndicatorOpts{
			Width: foundation.Some(railCircularIndicatorDiameter), Shape: w.indicatorShape, Color: w.indicatorColor})
	}
	return widgets.NewStack([]widgets.Widget{indicator, w.child}, widgets.StackOpts{Alignment: painting.AlignmentCenter})
}

// navigationRailDefaultsM2 are the hand coded defaults based on Material Design 2.
//
// Flutter: material/navigation_rail.dart _NavigationRailDefaultsM2
func navigationRailDefaultsM2(theme *ThemeData) NavigationRailThemeData {
	colors := theme.ColorScheme
	return NavigationRailThemeData{
		Elevation: foundation.Some(0.0), GroupAlignment: foundation.Some(-1.0), LabelType: NavigationRailLabelTypeNone,
		UseIndicator: foundation.Some(false), MinWidth: foundation.Some(72.0), MinExtendedWidth: foundation.Some(256.0),
		BackgroundColor:          foundation.Some(colors.Surface()),
		UnselectedLabelTextStyle: theme.TextTheme.BodyLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurface().WithOpacity(0.64))}),
		SelectedLabelTextStyle:   theme.TextTheme.BodyLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.Primary())}),
		UnselectedIconTheme: foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{
			Size: foundation.Some(24.0), Color: foundation.Some(colors.OnSurface()), Opacity: foundation.Some(0.64)})),
		SelectedIconTheme: foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{
			Size: foundation.Some(24.0), Color: foundation.Some(colors.Primary()), Opacity: foundation.Some(1.0)})),
	}
}

// navigationRailDefaultsM3 are the Material 3 defaults.
//
// Flutter: material/navigation_rail.dart _NavigationRailDefaultsM3
func navigationRailDefaultsM3(theme *ThemeData) NavigationRailThemeData {
	colors := theme.ColorScheme
	return NavigationRailThemeData{
		Elevation: foundation.Some(0.0), GroupAlignment: foundation.Some(-1.0), LabelType: NavigationRailLabelTypeNone,
		UseIndicator: foundation.Some(true), MinWidth: foundation.Some(80.0), MinExtendedWidth: foundation.Some(256.0),
		BackgroundColor:          foundation.Some(colors.Surface()),
		UnselectedLabelTextStyle: theme.TextTheme.LabelMedium.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurface())}),
		SelectedLabelTextStyle:   theme.TextTheme.LabelMedium.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurface())}),
		UnselectedIconTheme: foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{
			Size: foundation.Some(24.0), Color: foundation.Some(colors.OnSurfaceVariant())})),
		SelectedIconTheme: foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{
			Size: foundation.Some(24.0), Color: foundation.Some(colors.OnSecondaryContainer())})),
		IndicatorColor: foundation.Some(colors.SecondaryContainer()),
		IndicatorShape: painting.NewStadiumBorder(),
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagNavigationRailElevation                = "NavigationRailElevation"
	tagNavigationRailExtendedAnimationMissing = "NavigationRailExtendedAnimationMissing"
	tagNavigationRailExtendedLabels           = "NavigationRailExtendedLabels"
	tagNavigationRailMinExtendedWidth         = "NavigationRailMinExtendedWidth"
	tagNavigationRailMinWidth                 = "NavigationRailMinWidth"
	tagNavigationRailMinWidths                = "NavigationRailMinWidths"
	tagNavigationRailSelectedIndex            = "NavigationRailSelectedIndex"
)
