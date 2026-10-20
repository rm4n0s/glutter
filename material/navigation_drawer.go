// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// NavigationDrawerOpts are the optional parameters of NewNavigationDrawer.
//
// Flutter: material/navigation_drawer.dart NavigationDrawer.new
type NavigationDrawerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Header is the widget above the destinations.
	Header widgets.Widget
	// Footer is the widget below the destinations.
	Footer widgets.Widget
	// BackgroundColor is the color of the drawer.
	BackgroundColor foundation.Opt[ui.Color]
	// ShadowColor is the color of the shadow below the drawer.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the drawer.
	SurfaceTintColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the drawer.
	Elevation foundation.Opt[float64]
	// IndicatorColor is the color of the indicator of the selected destination.
	IndicatorColor foundation.Opt[ui.Color]
	// IndicatorShape is the shape of the indicator.
	IndicatorShape painting.ShapeBorder
	// OnDestinationSelected is called when a destination is tapped, with its index
	// among the destinations.
	OnDestinationSelected func(index int)
	// SelectedIndex is the index of the selected destination, among the
	// destinations; none for no selection. The default is 0.
	SelectedIndex foundation.Opt[int]
	// NoSelectedIndex says that no destination is selected (Flutter's
	// selectedIndex: null), which the zero value of SelectedIndex cannot say.
	NoSelectedIndex bool
	// TilePadding is the padding around each destination; the default is 12 at
	// each side.
	TilePadding painting.EdgeInsetsGeometry
}

// NavigationDrawer is a Material Design navigation drawer: a Drawer with
// destinations, headlines and dividers.
//
// Flutter: material/navigation_drawer.dart NavigationDrawer
type NavigationDrawer struct {
	widgets.StatelessWidget
	children []widgets.Widget
	o        NavigationDrawerOpts
}

// NewNavigationDrawer creates a drawer; the NavigationDrawerDestination widgets
// among the children are the destinations, in order.
//
// Flutter: material/navigation_drawer.dart NavigationDrawer.new
func NewNavigationDrawer(children []widgets.Widget, opts ...NavigationDrawerOpts) *NavigationDrawer {
	o := firstOpts(opts)
	if o.TilePadding == nil {
		o.TilePadding = painting.NewEdgeInsetsSymmetric(0, 12.0)
	}
	return &NavigationDrawer{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), children: children, o: o}
}

// Children are the destinations and the other widgets of the drawer.
//
// Flutter: material/navigation_drawer.dart NavigationDrawer.children
func (w *NavigationDrawer) Children() []widgets.Widget { return w.children }

// SelectedIndex is the index of the selected destination, if any.
//
// Flutter: material/navigation_drawer.dart NavigationDrawer.selectedIndex
func (w *NavigationDrawer) SelectedIndex() foundation.Opt[int] {
	if w.o.NoSelectedIndex {
		return foundation.None[int]()
	}
	return foundation.Some(w.o.SelectedIndex.OrElse(0))
}

// Build builds the drawer.
//
// Flutter: material/navigation_drawer.dart NavigationDrawer.build
func (w *NavigationDrawer) Build(bc widgets.BuildContext) widgets.Widget {
	totalNumberOfDestinations := 0
	for _, c := range w.children {
		if _, ok := c.(*NavigationDrawerDestination); ok {
			totalNumberOfDestinations++
		}
	}
	selectedIndex := w.SelectedIndex()

	destinationIndex := 0
	wrapChild := func(child widgets.Widget, index int) widgets.Widget {
		return newSelectableAnimatedBuilder(selectedIndex == foundation.Some(index), 500*time.Millisecond, false,
			func(_ widgets.BuildContext, a animation.Animation[float64]) widgets.Widget {
				return &navigationDrawerDestinationInfo{
					InheritedWidget:           widgets.NewInheritedWidgetBase(child),
					index:                     index,
					totalNumberOfDestinations: totalNumberOfDestinations,
					selectedAnimation:         a,
					indicatorColor:            w.o.IndicatorColor,
					indicatorShape:            w.o.IndicatorShape,
					tilePadding:               w.o.TilePadding,
					onTap: func() {
						if w.o.OnDestinationSelected != nil {
							w.o.OnDestinationSelected(index)
						}
					},
				}
			})
	}

	wrappedChildren := make([]widgets.Widget, len(w.children))
	for i, child := range w.children {
		if _, ok := child.(*NavigationDrawerDestination); ok {
			wrappedChildren[i] = wrapChild(child, destinationIndex)
			destinationIndex++
		} else {
			wrappedChildren[i] = child
		}
	}
	navigationDrawerTheme := NavigationDrawerThemeOf(bc)

	column := make([]widgets.Widget, 0, 3)
	if w.o.Header != nil {
		column = append(column, w.o.Header)
	}
	column = append(column, widgets.NewExpanded(NewMaterial(widgets.NewListView(wrappedChildren), MaterialOpts{Type: MaterialTypeTransparency})))
	if w.o.Footer != nil {
		column = append(column, w.o.Footer)
	}
	return NewDrawer(DrawerOpts{
		BackgroundColor:  firstOpt(w.o.BackgroundColor, navigationDrawerTheme.BackgroundColor),
		ShadowColor:      firstOpt(w.o.ShadowColor, navigationDrawerTheme.ShadowColor),
		SurfaceTintColor: firstOpt(w.o.SurfaceTintColor, navigationDrawerTheme.SurfaceTintColor),
		Elevation:        firstOpt(w.o.Elevation, navigationDrawerTheme.Elevation),
		Child:            widgets.NewSafeArea(widgets.NewColumn(column), widgets.SafeAreaOpts{Bottom: foundation.Some(false)}),
	})
}

// NavigationDrawerDestinationOpts are the optional parameters of
// NewNavigationDrawerDestination.
//
// Flutter: material/navigation_drawer.dart NavigationDrawerDestination.new
type NavigationDrawerDestinationOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// BackgroundColor is the color painted behind the destination.
	BackgroundColor foundation.Opt[ui.Color]
	// SelectedIcon is the icon that shows while the destination is selected; the
	// default is the icon.
	SelectedIcon widgets.Widget
	// NoEnabled says that the destination is disabled (Flutter's enabled: false).
	NoEnabled bool
}

// NavigationDrawerDestination is a destination of a NavigationDrawer: an icon
// and a label in a row.
//
// Flutter: material/navigation_drawer.dart NavigationDrawerDestination
type NavigationDrawerDestination struct {
	widgets.StatelessWidget
	icon  widgets.Widget
	label widgets.Widget
	o     NavigationDrawerDestinationOpts
}

// NewNavigationDrawerDestination creates a destination.
//
// Flutter: material/navigation_drawer.dart NavigationDrawerDestination.new
func NewNavigationDrawerDestination(icon, label widgets.Widget, opts ...NavigationDrawerDestinationOpts) *NavigationDrawerDestination {
	o := firstOpts(opts)
	return &NavigationDrawerDestination{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), icon: icon, label: label, o: o}
}

// Icon is the icon of the destination.
//
// Flutter: material/navigation_drawer.dart NavigationDrawerDestination.icon
func (w *NavigationDrawerDestination) Icon() widgets.Widget { return w.icon }

// Label is the label of the destination.
//
// Flutter: material/navigation_drawer.dart NavigationDrawerDestination.label
func (w *NavigationDrawerDestination) Label() widgets.Widget { return w.label }

// Enabled is whether the destination can be tapped.
//
// Flutter: material/navigation_drawer.dart NavigationDrawerDestination.enabled
func (w *NavigationDrawerDestination) Enabled() bool { return !w.o.NoEnabled }

// Build builds the destination.
//
// Flutter: material/navigation_drawer.dart NavigationDrawerDestination.build
func (w *NavigationDrawerDestination) Build(bc widgets.BuildContext) widgets.Widget {
	navigationDrawerTheme := NavigationDrawerThemeOf(bc)
	defaults := navigationDrawerDefaultsM3(ThemeOf(bc))
	a := navigationDrawerDestinationInfoOf(bc).selectedAnimation
	enabled := w.Enabled()
	stateOr := func(enabledState widgets.WidgetStates) widgets.WidgetStates {
		if enabled {
			return enabledState
		}
		return disabledState
	}

	return newNavigationDrawerDestinationBuilder(w.o.BackgroundColor, enabled,
		func(widgets.BuildContext) widgets.Widget {
			selectedIcon := w.o.SelectedIcon
			if selectedIcon == nil {
				selectedIcon = w.icon
			}
			if animation.IsForwardOrCompleted(a) {
				return widgets.NewIconThemeMerge(resolveIconThemeOr(navigationDrawerTheme.IconTheme, defaults.IconTheme, stateOr(selectedState)), selectedIcon)
			}
			return widgets.NewIconThemeMerge(resolveIconThemeOr(navigationDrawerTheme.IconTheme, defaults.IconTheme, stateOr(unselectedState)), w.icon)
		},
		func(widgets.BuildContext) widgets.Widget {
			state := stateOr(unselectedState)
			if animation.IsForwardOrCompleted(a) {
				state = stateOr(selectedState)
			}
			return widgets.NewDefaultTextStyle(
				firstTextStyle(resolveTextStyle(navigationDrawerTheme.LabelTextStyle, state), resolveTextStyle(defaults.LabelTextStyle, state)),
				w.label)
		})
}

// navigationDrawerDestinationBuilder builds the ink, the indicator and the row of
// a destination.
//
// Flutter: material/navigation_drawer.dart _NavigationDestinationBuilder
type navigationDrawerDestinationBuilder struct {
	widgets.StatelessWidget
	buildIcon       widgets.WidgetBuilder
	buildLabel      widgets.WidgetBuilder
	enabled         bool
	backgroundColor foundation.Opt[ui.Color]
}

func newNavigationDrawerDestinationBuilder(backgroundColor foundation.Opt[ui.Color], enabled bool, buildIcon, buildLabel widgets.WidgetBuilder) *navigationDrawerDestinationBuilder {
	return &navigationDrawerDestinationBuilder{StatelessWidget: widgets.NewStatelessWidgetBase(), buildIcon: buildIcon, buildLabel: buildLabel,
		enabled: enabled, backgroundColor: backgroundColor}
}

func (w *navigationDrawerDestinationBuilder) Build(bc widgets.BuildContext) widgets.Widget {
	info := navigationDrawerDestinationInfoOf(bc)
	navigationDrawerTheme := NavigationDrawerThemeOf(bc)
	defaults := navigationDrawerDefaultsM3(ThemeOf(bc))

	indicatorShape := info.indicatorShape
	if indicatorShape == nil {
		indicatorShape = navigationDrawerTheme.IndicatorShape
	}
	if indicatorShape == nil {
		indicatorShape = defaults.IndicatorShape
	}
	indicatorSize := firstOpt(navigationDrawerTheme.IndicatorSize, defaults.IndicatorSize).Value()
	var onTap func()
	if w.enabled {
		onTap = info.onTap
	}
	inkWell := NewInkWell(
		widgets.NewStack([]widgets.Widget{
			NewNavigationIndicator(info.selectedAnimation, NavigationIndicatorOpts{
				Color: firstOpt(info.indicatorColor, navigationDrawerTheme.IndicatorColor, defaults.IndicatorColor), Shape: indicatorShape,
				Width: foundation.Some(indicatorSize.Width), Height: foundation.Some(indicatorSize.Height)}),
			widgets.NewRow([]widgets.Widget{
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(16.0)}),
				w.buildIcon(bc),
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(12.0)}),
				w.buildLabel(bc),
			}),
		}, widgets.StackOpts{Alignment: painting.AlignmentCenter}),
		InkWellOpts{HighlightColor: foundation.Some(Colors.Transparent), OnTap: onTap, CustomBorder: indicatorShape})

	destination := widgets.NewPadding(info.tilePadding,
		newNavigationDrawerDestinationSemantics(
			widgets.NewSizedBox(inkWell, widgets.SizedBoxOpts{Height: foundation.Some(navigationDrawerTheme.TileHeight.OrElse(defaults.TileHeight.Value()))})))
	if w.backgroundColor.IsSome() {
		return NewInk(destination, InkOpts{Color: w.backgroundColor})
	}
	return destination
}

// navigationDrawerDestinationSemantics gives a destination the semantics of a
// selectable item with its position in the drawer.
//
// Flutter: material/navigation_drawer.dart _NavigationDestinationSemantics
type navigationDrawerDestinationSemantics struct {
	widgets.StatelessWidget
	child widgets.Widget
}

func newNavigationDrawerDestinationSemantics(child widgets.Widget) *navigationDrawerDestinationSemantics {
	return &navigationDrawerDestinationSemantics{StatelessWidget: widgets.NewStatelessWidgetBase(), child: child}
}

func (w *navigationDrawerDestinationSemantics) Build(bc widgets.BuildContext) widgets.Widget {
	localizations := MaterialLocalizationsOf(bc)
	info := navigationDrawerDestinationInfoOf(bc)
	// The status builder makes sure that the semantics update to "selected" when
	// the animation status changes.
	return newStatusTransitionBuilder(info.selectedAnimation, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
		return widgets.NewSemantics(child, widgets.SemanticsOpts{
			Selected: foundation.Some(animation.IsForwardOrCompleted(info.selectedAnimation)), Container: true})
	}, widgets.NewStack([]widgets.Widget{
		w.child,
		widgets.NewSemantics(nil, widgets.SemanticsOpts{Label: foundation.Some(localizations.TabLabel(info.index+1, info.totalNumberOfDestinations))}),
	}, widgets.StackOpts{Alignment: painting.AlignmentCenter}))
}

// navigationDrawerDestinationInfo is what a NavigationDrawer tells its
// destinations.
//
// Flutter: material/navigation_drawer.dart _NavigationDrawerDestinationInfo
type navigationDrawerDestinationInfo struct {
	widgets.InheritedWidget
	index                     int
	totalNumberOfDestinations int
	selectedAnimation         animation.Animation[float64]
	indicatorColor            foundation.Opt[ui.Color]
	indicatorShape            painting.ShapeBorder
	onTap                     func()
	tilePadding               painting.EdgeInsetsGeometry
}

func navigationDrawerDestinationInfoOf(bc widgets.BuildContext) *navigationDrawerDestinationInfo {
	result, ok := widgets.DependOn[*navigationDrawerDestinationInfo](bc)
	if !ok {
		panic(errors.New("NavigationDrawerDestinationInfoMissing", "Navigation destinations need a navigationDrawerDestinationInfo parent, which is usually provided by NavigationDrawer.",
			"widget", foundation.RuntimeTypeName(bc.Widget())))
	}
	return result
}

// UpdateShouldNotify is always true: Flutter compares the onTap closure, which
// is a new one on every build.
func (w *navigationDrawerDestinationInfo) UpdateShouldNotify(widgets.Widget) bool { return true }

// navigationDrawerDefaultsM3 are the Material 3 defaults.
//
// Flutter: material/navigation_drawer.dart _NavigationDrawerDefaultsM3
func navigationDrawerDefaultsM3(theme *ThemeData) NavigationDrawerThemeData {
	colors, textTheme := theme.ColorScheme, theme.TextTheme
	stateColor := func(states widgets.WidgetStates) ui.Color {
		switch {
		case states.Has(widgets.WidgetStateDisabled):
			return colors.OnSurfaceVariant().WithOpacity(0.38)
		case states.Has(widgets.WidgetStateSelected):
			return colors.OnSecondaryContainer()
		}
		return colors.OnSurfaceVariant()
	}
	return NavigationDrawerThemeData{
		Elevation: foundation.Some(1.0), TileHeight: foundation.Some(56.0),
		IndicatorShape: painting.NewStadiumBorder(), IndicatorSize: foundation.Some(ui.Size{Width: 336.0, Height: 56.0}),
		BackgroundColor: foundation.Some(colors.SurfaceContainerLow()), SurfaceTintColor: foundation.Some(Colors.Transparent),
		ShadowColor: foundation.Some(Colors.Transparent), IndicatorColor: foundation.Some(colors.SecondaryContainer()),
		IconTheme: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) widgets.IconThemeData {
			return widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(24.0), Color: foundation.Some(stateColor(states))})
		}),
		LabelTextStyle: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) *painting.TextStyle {
			return textTheme.LabelLarge.Apply(painting.TextStyleApply{Color: foundation.Some(stateColor(states))})
		}),
	}
}
