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
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// NavigationDestinationLabelBehavior defines how the labels of the destinations
// of a NavigationBar or a NavigationDrawer are shown.
//
// Flutter: material/navigation_bar.dart NavigationDestinationLabelBehavior
type NavigationDestinationLabelBehavior int

const (
	// NavigationDestinationLabelBehaviorUnspecified is the zero value: the
	// behavior of the theme, or always show.
	NavigationDestinationLabelBehaviorUnspecified NavigationDestinationLabelBehavior = iota
	// NavigationDestinationLabelBehaviorAlwaysShow shows all the labels.
	//
	// Flutter: material/navigation_bar.dart NavigationDestinationLabelBehavior.alwaysShow
	NavigationDestinationLabelBehaviorAlwaysShow
	// NavigationDestinationLabelBehaviorAlwaysHide hides all the labels.
	//
	// Flutter: material/navigation_bar.dart NavigationDestinationLabelBehavior.alwaysHide
	NavigationDestinationLabelBehaviorAlwaysHide
	// NavigationDestinationLabelBehaviorOnlyShowSelected shows only the label of
	// the selected destination.
	//
	// Flutter: material/navigation_bar.dart NavigationDestinationLabelBehavior.onlyShowSelected
	NavigationDestinationLabelBehaviorOnlyShowSelected
)

// Name is the name of the value, as Dart's Enum.name.
func (b NavigationDestinationLabelBehavior) Name() string {
	return enumName(int(b), "alwaysShow", "alwaysHide", "onlyShowSelected")
}

// String is "NavigationDestinationLabelBehavior." and the name.
func (b NavigationDestinationLabelBehavior) String() string {
	return "NavigationDestinationLabelBehavior." + b.Name()
}

const (
	// Flutter: material/navigation_bar.dart _kIndicatorHeight
	navigationIndicatorHeight = 32.0
	// Flutter: material/navigation_bar.dart _kIndicatorWidth
	navigationIndicatorWidth = 64.0
	// Flutter: material/navigation_bar.dart _kMaxLabelTextScaleFactor
	navigationMaxLabelTextScaleFactor = 1.3
)

// NavigationBarOpts are the optional parameters of NewNavigationBar.
//
// Flutter: material/navigation_bar.dart NavigationBar.new
type NavigationBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// AnimationDuration is the duration of the transitions of the destinations
	// when the selection changes; the default is 500 ms.
	AnimationDuration foundation.Opt[time.Duration]
	// SelectedIndex is the index of the selected destination.
	SelectedIndex int
	// OnDestinationSelected is called when a destination is tapped, with its index.
	OnDestinationSelected func(index int)
	// BackgroundColor is the color of the bar.
	BackgroundColor foundation.Opt[ui.Color]
	// Elevation is the z-coordinate of the bar.
	Elevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow below the bar.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the surface tint of the bar.
	SurfaceTintColor foundation.Opt[ui.Color]
	// IndicatorColor is the color of the indicator of the selected destination.
	IndicatorColor foundation.Opt[ui.Color]
	// IndicatorShape is the shape of the indicator.
	IndicatorShape painting.ShapeBorder
	// Height is the height of the bar.
	Height foundation.Opt[float64]
	// LabelBehavior is how the labels of the destinations are shown.
	LabelBehavior NavigationDestinationLabelBehavior
	// OverlayColor is the color of the highlight and the splash, in each state.
	OverlayColor widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	// LabelTextStyle is the style of the labels, in each state.
	LabelTextStyle widgets.WidgetStateProperty[*painting.TextStyle]
	// LabelPadding is the padding around the labels.
	LabelPadding painting.EdgeInsetsGeometry
	// MaintainBottomViewPadding says whether the bar keeps the bottom view padding
	// when the keyboard is shown.
	MaintainBottomViewPadding bool
}

// NavigationBar is a Material Design bottom navigation bar: it offers a
// persistent and convenient way to switch between the primary destinations of an
// app.
//
// Flutter: material/navigation_bar.dart NavigationBar
type NavigationBar struct {
	widgets.StatelessWidget
	destinations []widgets.Widget
	o            NavigationBarOpts
}

// NewNavigationBar creates a bar for two or more destinations, usually
// NavigationDestination widgets. In debug builds it panics with the tag
// NavigationBarDestinations when there are fewer than two and with
// NavigationBarSelectedIndex when the selected index is outside the destinations.
//
// Flutter: material/navigation_bar.dart NavigationBar.new
func NewNavigationBar(destinations []widgets.Widget, opts ...NavigationBarOpts) *NavigationBar {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if len(destinations) < 2 {
			panic(errors.New("NavigationBarDestinations", "a NavigationBar needs at least two destinations", "destinations", len(destinations)))
		}
		if o.SelectedIndex < 0 || o.SelectedIndex >= len(destinations) {
			panic(errors.New("NavigationBarSelectedIndex", "the selected index of a NavigationBar must be one of its destinations",
				"selectedIndex", o.SelectedIndex, "destinations", len(destinations)))
		}
	}
	return &NavigationBar{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), destinations: destinations, o: o}
}

// Destinations are the destinations of the bar.
//
// Flutter: material/navigation_bar.dart NavigationBar.destinations
func (w *NavigationBar) Destinations() []widgets.Widget { return w.destinations }

// SelectedIndex is the index of the selected destination.
//
// Flutter: material/navigation_bar.dart NavigationBar.selectedIndex
func (w *NavigationBar) SelectedIndex() int { return w.o.SelectedIndex }

func (w *NavigationBar) handleTap(index int) func() {
	if w.o.OnDestinationSelected != nil {
		return func() { w.o.OnDestinationSelected(index) }
	}
	return func() {}
}

// Build builds the bar.
//
// Flutter: material/navigation_bar.dart NavigationBar.build
func (w *NavigationBar) Build(bc widgets.BuildContext) widgets.Widget {
	defaults := navigationBarDefaults(bc)
	navigationBarTheme := NavigationBarThemeOf(bc)
	effectiveHeight := firstOpt(w.o.Height, navigationBarTheme.Height, defaults.Height).Value()
	effectiveLabelBehavior := w.o.LabelBehavior
	if effectiveLabelBehavior == NavigationDestinationLabelBehaviorUnspecified {
		effectiveLabelBehavior = navigationBarTheme.LabelBehavior
	}
	if effectiveLabelBehavior == NavigationDestinationLabelBehaviorUnspecified {
		effectiveLabelBehavior = defaults.LabelBehavior
	}

	duration := w.o.AnimationDuration.OrElse(500 * time.Millisecond)
	children := make([]widgets.Widget, len(w.destinations))
	for i, destination := range w.destinations {
		isSelected := i == w.o.SelectedIndex
		children[i] = widgets.NewExpanded(
			widgets.NewMergeSemantics(
				widgets.NewSemantics(
					newSelectableAnimatedBuilder(isSelected, duration, false, func(_ widgets.BuildContext, a animation.Animation[float64]) widgets.Widget {
						return &navigationDestinationInfo{
							InheritedWidget:           widgets.NewInheritedWidgetBase(destination),
							index:                     i,
							selectedIndex:             w.o.SelectedIndex,
							totalNumberOfDestinations: len(w.destinations),
							selectedAnimation:         a,
							labelBehavior:             effectiveLabelBehavior,
							indicatorColor:            w.o.IndicatorColor,
							indicatorShape:            w.o.IndicatorShape,
							overlayColor:              w.o.OverlayColor,
							onTap:                     w.handleTap(i),
							labelTextStyle:            w.o.LabelTextStyle,
							labelPadding:              w.o.LabelPadding,
						}
					}),
					widgets.SemanticsOpts{Role: foundation.Some(ui.SemanticsRoleTab), Selected: foundation.Some(isSelected)})))
	}

	return NewMaterial(
		widgets.NewSafeArea(
			widgets.NewSemantics(
				widgets.NewSizedBox(widgets.NewRow(children), widgets.SizedBoxOpts{Height: foundation.Some(effectiveHeight)}),
				widgets.SemanticsOpts{Role: foundation.Some(ui.SemanticsRoleTabBar), ExplicitChildNodes: true, Container: true}),
			widgets.SafeAreaOpts{MaintainBottomViewPadding: w.o.MaintainBottomViewPadding}),
		MaterialOpts{
			Color:            foundation.Some(firstOpt(w.o.BackgroundColor, navigationBarTheme.BackgroundColor, defaults.BackgroundColor).Value()),
			Elevation:        firstOpt(w.o.Elevation, navigationBarTheme.Elevation, defaults.Elevation).Value(),
			ShadowColor:      firstOpt(w.o.ShadowColor, navigationBarTheme.ShadowColor, defaults.ShadowColor),
			SurfaceTintColor: firstOpt(w.o.SurfaceTintColor, navigationBarTheme.SurfaceTintColor, defaults.SurfaceTintColor),
		})
}

// NavigationDestinationOpts are the optional parameters of
// NewNavigationDestination.
//
// Flutter: material/navigation_bar.dart NavigationDestination.new
type NavigationDestinationOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// SelectedIcon is the icon that shows while the destination is selected; the
	// default is the icon.
	SelectedIcon widgets.Widget
	// Tooltip is the text of the tooltip; the default is the label.
	Tooltip string
	// NoEnabled says that the destination is disabled and cannot be tapped
	// (Flutter's enabled: false).
	NoEnabled bool
}

// NavigationDestination is a destination of a NavigationBar: an icon with a
// label below it.
//
// Flutter: material/navigation_bar.dart NavigationDestination
type NavigationDestination struct {
	widgets.StatelessWidget
	icon  widgets.Widget
	label string
	o     NavigationDestinationOpts
}

// NewNavigationDestination creates a destination.
//
// Flutter: material/navigation_bar.dart NavigationDestination.new
func NewNavigationDestination(icon widgets.Widget, label string, opts ...NavigationDestinationOpts) *NavigationDestination {
	o := firstOpts(opts)
	return &NavigationDestination{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), icon: icon, label: label, o: o}
}

// Icon is the icon of the destination.
//
// Flutter: material/navigation_bar.dart NavigationDestination.icon
func (w *NavigationDestination) Icon() widgets.Widget { return w.icon }

// Label is the text label of the destination.
//
// Flutter: material/navigation_bar.dart NavigationDestination.label
func (w *NavigationDestination) Label() string { return w.label }

// Enabled is whether the destination can be tapped.
//
// Flutter: material/navigation_bar.dart NavigationDestination.enabled
func (w *NavigationDestination) Enabled() bool { return !w.o.NoEnabled }

var (
	selectedState   = widgets.NewWidgetStates(widgets.WidgetStateSelected)
	unselectedState = widgets.NewWidgetStates()
	disabledState   = widgets.NewWidgetStates(widgets.WidgetStateDisabled)
)

// resolveIconTheme resolves a state property of icon themes: a missing property
// and an empty result are Flutter's null.
func resolveIconTheme(p widgets.WidgetStateProperty[widgets.IconThemeData], states widgets.WidgetStates) (widgets.IconThemeData, bool) {
	if p == nil {
		return widgets.IconThemeData{}, false
	}
	v := p.Resolve(states)
	if foundation.ValuesEqual(v, widgets.IconThemeData{}) {
		return widgets.IconThemeData{}, false
	}
	return v, true
}

func resolveIconThemeOr(theme, defaults widgets.WidgetStateProperty[widgets.IconThemeData], states widgets.WidgetStates) widgets.IconThemeData {
	if v, ok := resolveIconTheme(theme, states); ok {
		return v
	}
	v, _ := resolveIconTheme(defaults, states)
	return v
}

// resolveTextStyle resolves a state property of text styles, where nil is
// Flutter's null.
func resolveTextStyle(p widgets.WidgetStateProperty[*painting.TextStyle], states widgets.WidgetStates) *painting.TextStyle {
	if p == nil {
		return nil
	}
	return p.Resolve(states)
}

func firstTextStyle(styles ...*painting.TextStyle) *painting.TextStyle {
	for _, s := range styles {
		if s != nil {
			return s
		}
	}
	return nil
}

// Build builds the destination.
//
// Flutter: material/navigation_bar.dart NavigationDestination.build
func (w *NavigationDestination) Build(bc widgets.BuildContext) widgets.Widget {
	info := navigationDestinationInfoOf(bc)
	navigationBarTheme := NavigationBarThemeOf(bc)
	defaults := navigationBarDefaults(bc)
	a := info.selectedAnimation
	enabled := w.Enabled()

	return newNavigationDestinationBuilder(w.label, w.o.Tooltip, enabled,
		func(widgets.BuildContext) widgets.Widget {
			selectedIconTheme := resolveIconThemeOr(navigationBarTheme.IconTheme, defaults.IconTheme, selectedState)
			unselectedIconTheme := resolveIconThemeOr(navigationBarTheme.IconTheme, defaults.IconTheme, unselectedState)
			disabledIconTheme := resolveIconThemeOr(navigationBarTheme.IconTheme, defaults.IconTheme, disabledState)

			selectedIcon := w.o.SelectedIcon
			if selectedIcon == nil {
				selectedIcon = w.icon
			}
			enabledTheme, enabledUnselectedTheme := selectedIconTheme, unselectedIconTheme
			if !enabled {
				enabledTheme, enabledUnselectedTheme = disabledIconTheme, disabledIconTheme
			}
			selectedIconWidget := widgets.NewIconThemeMerge(enabledTheme, selectedIcon)
			unselectedIconWidget := widgets.NewIconThemeMerge(enabledUnselectedTheme, w.icon)

			indicatorShape := info.indicatorShape
			if indicatorShape == nil {
				indicatorShape = navigationBarTheme.IndicatorShape
			}
			if indicatorShape == nil {
				indicatorShape = defaults.IndicatorShape
			}
			return widgets.NewStack([]widgets.Widget{
				NewNavigationIndicator(a, NavigationIndicatorOpts{
					Color: firstOpt(info.indicatorColor, navigationBarTheme.IndicatorColor, defaults.IndicatorColor), Shape: indicatorShape}),
				newStatusTransitionBuilder(a, func(widgets.BuildContext, widgets.Widget) widgets.Widget {
					if animation.IsForwardOrCompleted(a) {
						return selectedIconWidget
					}
					return unselectedIconWidget
				}, nil),
			}, widgets.StackOpts{Alignment: painting.AlignmentCenter})
		},
		func(widgets.BuildContext) widgets.Widget {
			selectedStyle := firstTextStyle(resolveTextStyle(info.labelTextStyle, selectedState),
				resolveTextStyle(navigationBarTheme.LabelTextStyle, selectedState), resolveTextStyle(defaults.LabelTextStyle, selectedState))
			unselectedStyle := firstTextStyle(resolveTextStyle(info.labelTextStyle, unselectedState),
				resolveTextStyle(navigationBarTheme.LabelTextStyle, unselectedState), resolveTextStyle(defaults.LabelTextStyle, unselectedState))
			disabledStyle := firstTextStyle(resolveTextStyle(info.labelTextStyle, disabledState),
				resolveTextStyle(navigationBarTheme.LabelTextStyle, disabledState), resolveTextStyle(defaults.LabelTextStyle, disabledState))
			labelPadding := info.labelPadding
			if labelPadding == nil {
				labelPadding = navigationBarTheme.LabelPadding
			}
			if labelPadding == nil {
				labelPadding = defaults.LabelPadding
			}

			textStyle := disabledStyle
			if enabled {
				if animation.IsForwardOrCompleted(a) {
					textStyle = selectedStyle
				} else {
					textStyle = unselectedStyle
				}
			}
			return widgets.NewPadding(labelPadding,
				// Set maximum text scale factor to navigationMaxLabelTextScaleFactor for
				// the label to keep the visual hierarchy the same even with larger font
				// sizes. To opt out, wrap the label widget in a MediaQuery widget with a
				// different TextScaler.
				widgets.NewMediaQueryWithClampedTextScaling(
					widgets.NewText(w.label, widgets.TextOpts{Style: textStyle}),
					widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: navigationMaxLabelTextScaleFactor}))
		})
}

// navigationDestinationBuilder builds the ink, the tooltip and the semantics
// that are the same for the destinations of the bar and of the drawer.
//
// Flutter: material/navigation_bar.dart _NavigationDestinationBuilder
type navigationDestinationBuilder struct {
	widgets.StatefulWidget
	buildIcon  widgets.WidgetBuilder
	buildLabel widgets.WidgetBuilder
	label      string
	tooltip    string
	enabled    bool
}

func newNavigationDestinationBuilder(label, tooltip string, enabled bool, buildIcon, buildLabel widgets.WidgetBuilder) *navigationDestinationBuilder {
	return &navigationDestinationBuilder{StatefulWidget: widgets.NewStatefulWidgetBase(), buildIcon: buildIcon, buildLabel: buildLabel,
		label: label, tooltip: tooltip, enabled: enabled}
}

func (w *navigationDestinationBuilder) CreateState() widgets.State {
	return &navigationDestinationBuilderState{iconKey: widgets.NewGlobalKey[widgets.State]()}
}

type navigationDestinationBuilderState struct {
	widgets.StateOf[*navigationDestinationBuilder]
	iconKey *widgets.GlobalKey[widgets.State]
}

func (s *navigationDestinationBuilderState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	info := navigationDestinationInfoOf(bc)
	navigationBarTheme := NavigationBarThemeOf(bc)
	defaults := navigationBarDefaults(bc)

	customBorder := info.indicatorShape
	if customBorder == nil {
		customBorder = navigationBarTheme.IndicatorShape
	}
	if customBorder == nil {
		customBorder = defaults.IndicatorShape
	}
	overlayColor := info.overlayColor
	if overlayColor == nil {
		overlayColor = navigationBarTheme.OverlayColor
	}
	var onTap func()
	if w.enabled {
		onTap = info.onTap
	}
	message := w.tooltip
	if message == "" {
		message = w.label
	}
	return newNavigationBarDestinationSemantics(w.enabled,
		NewTooltip(NewInkResponse(
			widgets.NewRow([]widgets.Widget{
				widgets.NewExpanded(newNavigationBarDestinationLayout(w.buildIcon(bc), s.iconKey, w.buildLabel(bc))),
			}),
			InkResponseOpts{
				OnTap: onTap, ContainedInkWell: true, HighlightColor: foundation.Some(Colors.Transparent),
				CustomBorder: customBorder, OverlayColor: overlayColor,
				GetRectCallback: func(referenceBox rendering.Box) RectCallback {
					return func() ui.Rect {
						iconBox := s.iconKey.CurrentContext().FindRenderObject().(rendering.Box)
						iconRect := iconBox.LocalToGlobal(ui.OffsetZero, nil).And(iconBox.Size())
						return referenceBox.GlobalToLocal(iconRect.TopLeft(), nil).And(iconBox.Size())
					}
				},
			}), TooltipOpts{Message: message}))
}

// navigationDestinationInfo is what a NavigationBar tells its destinations.
//
// Flutter: material/navigation_bar.dart _NavigationDestinationInfo
type navigationDestinationInfo struct {
	widgets.InheritedWidget
	index                     int
	selectedIndex             int
	totalNumberOfDestinations int
	selectedAnimation         animation.Animation[float64]
	labelBehavior             NavigationDestinationLabelBehavior
	indicatorColor            foundation.Opt[ui.Color]
	indicatorShape            painting.ShapeBorder
	overlayColor              widgets.WidgetStateProperty[foundation.Opt[ui.Color]]
	onTap                     func()
	labelTextStyle            widgets.WidgetStateProperty[*painting.TextStyle]
	labelPadding              painting.EdgeInsetsGeometry
}

func navigationDestinationInfoOf(bc widgets.BuildContext) *navigationDestinationInfo {
	result, ok := widgets.DependOn[*navigationDestinationInfo](bc)
	if !ok {
		panic(errors.New("NavigationDestinationInfoMissing", "Navigation destinations need a navigationDestinationInfo parent, which is usually provided by NavigationBar.",
			"widget", foundation.RuntimeTypeName(bc.Widget())))
	}
	return result
}

// UpdateShouldNotify is always true: Flutter compares the onTap closure, which
// is a new one on every build.
func (w *navigationDestinationInfo) UpdateShouldNotify(widgets.Widget) bool { return true }

// NavigationIndicatorOpts are the optional parameters of NewNavigationIndicator.
//
// Flutter: material/navigation_bar.dart NavigationIndicator.new
type NavigationIndicatorOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Color is the fill of the indicator; the default is the secondary color of
	// the theme.
	Color foundation.Opt[ui.Color]
	// Width is the width of the indicator; the default is 64.
	Width foundation.Opt[float64]
	// Height is the height of the indicator; the default is 32.
	Height foundation.Opt[float64]
	// BorderRadius is the radius of the indicator when it has no shape; the
	// default is 16.
	BorderRadius foundation.Opt[painting.BorderRadius]
	// Shape is the shape of the indicator; the default is a rounded rectangle.
	Shape painting.ShapeBorder
}

// NavigationIndicator is the pill-shaped indicator of the selected destination of
// a navigation bar or a navigation rail: it scales in and fades while its
// animation runs.
//
// Flutter: material/navigation_bar.dart NavigationIndicator
type NavigationIndicator struct {
	widgets.StatelessWidget
	animation animation.Animation[float64]
	o         NavigationIndicatorOpts
}

// NewNavigationIndicator creates the indicator.
//
// Flutter: material/navigation_bar.dart NavigationIndicator.new
func NewNavigationIndicator(a animation.Animation[float64], opts ...NavigationIndicatorOpts) *NavigationIndicator {
	o := firstOpts(opts)
	return &NavigationIndicator{StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{Key: o.Key}), animation: a, o: o}
}

// Build builds the indicator.
//
// Flutter: material/navigation_bar.dart NavigationIndicator.build
func (w *NavigationIndicator) Build(bc widgets.BuildContext) widgets.Widget {
	a := w.animation
	return widgets.NewAnimatedBuilder(a, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
		// The scale should be 0 when the animation is unselected, as soon as the
		// animation starts, the scale jumps to 40%, and then animates to 100% along
		// a curve.
		scale := 0.0
		if !animation.IsDismissed(a) {
			scale = animation.NewTween(0.4, 1.0).Transform(animation.NewCurveTween(animation.Curves.EaseInOutCubicEmphasized).Transform(a.Value()))
		}
		return widgets.NewTransform(ui.NewMatrix4Diagonal3Values(float32(scale), 1.0, 1.0), child,
			widgets.TransformOpts{Alignment: painting.AlignmentCenter})
	},
		// Fade should be a 100ms animation whenever the parent animation changes
		// direction.
		newStatusTransitionBuilder(a, func(widgets.BuildContext, widgets.Widget) widgets.Widget {
			return newSelectableAnimatedBuilder(animation.IsForwardOrCompleted(a), 100*time.Millisecond, true,
				func(bc widgets.BuildContext, fadeAnimation animation.Animation[float64]) widgets.Widget {
					shape := w.o.Shape
					if shape == nil {
						shape = painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{
							BorderRadius: w.o.BorderRadius.OrElse(painting.NewBorderRadiusAll(ui.NewRadiusCircular(16)))})
					}
					return widgets.NewFadeTransition(fadeAnimation,
						NewInk(nil, InkOpts{
							Width: foundation.Some(w.o.Width.OrElse(navigationIndicatorWidth)), Height: foundation.Some(w.o.Height.OrElse(navigationIndicatorHeight)),
							Decoration: painting.NewShapeDecoration(shape, painting.ShapeDecorationOpts{
								Color: foundation.Some(w.o.Color.OrElse(ThemeOf(bc).ColorScheme.Secondary()))})}))
				})
		}, nil))
}

// navigationBarDestinationLayout lays the icon and the label of a destination
// out.
//
// Flutter: material/navigation_bar.dart _NavigationBarDestinationLayout
type navigationBarDestinationLayout struct {
	widgets.StatelessWidget
	icon    widgets.Widget
	iconKey *widgets.GlobalKey[widgets.State]
	label   widgets.Widget
}

func newNavigationBarDestinationLayout(icon widgets.Widget, iconKey *widgets.GlobalKey[widgets.State], label widgets.Widget) *navigationBarDestinationLayout {
	return &navigationBarDestinationLayout{StatelessWidget: widgets.NewStatelessWidgetBase(), icon: icon, iconKey: iconKey, label: label}
}

const (
	navigationDestinationIconID  = 1
	navigationDestinationLabelID = 2
)

func (w *navigationBarDestinationLayout) Build(widgets.BuildContext) widgets.Widget {
	return newDestinationLayoutAnimationBuilder(func(_ widgets.BuildContext, a animation.Animation[float64]) widgets.Widget {
		return widgets.NewCustomMultiChildLayout(
			&navigationDestinationLayoutDelegate{MultiChildLayoutDelegateBase: rendering.NewMultiChildLayoutDelegateBase(a), animation: a},
			[]widgets.Widget{
				// The key is used by the ink response to query the icon position.
				widgets.NewLayoutId(navigationDestinationIconID, widgets.NewKeyedSubtree(w.icon, widgets.WidgetOpts{Key: w.iconKey})),
				widgets.NewLayoutId(navigationDestinationLabelID, widgets.NewFadeTransition(a, w.label, widgets.FadeTransitionOpts{AlwaysIncludeSemantics: true})),
			})
	})
}

// destinationLayoutAnimationBuilder gives the layout the animation that matches
// the label behavior.
//
// Flutter: material/navigation_bar.dart _DestinationLayoutAnimationBuilder
type destinationLayoutAnimationBuilder struct {
	widgets.StatelessWidget
	builder func(widgets.BuildContext, animation.Animation[float64]) widgets.Widget
}

func newDestinationLayoutAnimationBuilder(builder func(widgets.BuildContext, animation.Animation[float64]) widgets.Widget) *destinationLayoutAnimationBuilder {
	return &destinationLayoutAnimationBuilder{StatelessWidget: widgets.NewStatelessWidgetBase(), builder: builder}
}

func (w *destinationLayoutAnimationBuilder) Build(bc widgets.BuildContext) widgets.Widget {
	info := navigationDestinationInfoOf(bc)
	switch info.labelBehavior {
	case NavigationDestinationLabelBehaviorAlwaysHide:
		return w.builder(bc, animation.AlwaysDismissed)
	case NavigationDestinationLabelBehaviorOnlyShowSelected:
		return newCurvedAnimationBuilder(info.selectedAnimation, animation.Curves.EaseInOutCubicEmphasized,
			animation.Curves.EaseInOutCubicEmphasized.Flipped(), w.builder)
	default:
		return w.builder(bc, animation.AlwaysComplete)
	}
}

// navigationBarDestinationSemantics gives a destination the semantics of a
// button with its position in the bar.
//
// Flutter: material/navigation_bar.dart _NavigationBarDestinationSemantics
type navigationBarDestinationSemantics struct {
	widgets.StatelessWidget
	enabled bool
	child   widgets.Widget
}

func newNavigationBarDestinationSemantics(enabled bool, child widgets.Widget) *navigationBarDestinationSemantics {
	return &navigationBarDestinationSemantics{StatelessWidget: widgets.NewStatelessWidgetBase(), enabled: enabled, child: child}
}

func (w *navigationBarDestinationSemantics) Build(bc widgets.BuildContext) widgets.Widget {
	localizations := MaterialLocalizationsOf(bc)
	info := navigationDestinationInfoOf(bc)
	// The status builder makes sure that the semantics update to "selected" when
	// the animation status changes.
	return newStatusTransitionBuilder(info.selectedAnimation, func(_ widgets.BuildContext, child widgets.Widget) widgets.Widget {
		return widgets.NewSemantics(child, widgets.SemanticsOpts{Enabled: foundation.Some(w.enabled), Button: foundation.Some(true)})
	}, widgets.NewStack([]widgets.Widget{
		w.child,
		widgets.NewSemantics(nil, widgets.SemanticsOpts{Label: foundation.Some(localizations.TabLabel(info.index+1, info.totalNumberOfDestinations))}),
	}, widgets.StackOpts{Alignment: painting.AlignmentCenter}))
}

// navigationDestinationLayoutDelegate positions the icon and the label: the icon
// is centered when the label is hidden, and the pair is centered when it shows.
//
// Flutter: material/navigation_bar.dart _NavigationDestinationLayoutDelegate
type navigationDestinationLayoutDelegate struct {
	rendering.MultiChildLayoutDelegateBase
	animation animation.Animation[float64]
}

func (d *navigationDestinationLayoutDelegate) PerformLayout(size ui.Size) {
	halfWidth := func(s ui.Size) float64 { return s.Width / 2 }
	halfHeight := func(s ui.Size) float64 { return s.Height / 2 }

	iconSize := d.LayoutChild(navigationDestinationIconID, rendering.NewBoxConstraintsLoose(size))
	labelSize := d.LayoutChild(navigationDestinationLabelID, rendering.NewBoxConstraintsLoose(size))

	yPositionOffset := animation.NewTween(
		// When unselected, the icon is centered vertically.
		halfHeight(iconSize),
		// When selected, the icon and label are centered vertically.
		halfHeight(iconSize)+halfHeight(labelSize)).Transform(d.animation.Value())
	iconYPosition := halfHeight(size) - yPositionOffset

	// Position the icon, centered horizontally.
	d.PositionChild(navigationDestinationIconID, ui.Offset{Dx: halfWidth(size) - halfWidth(iconSize), Dy: iconYPosition})
	// Position the label, centered horizontally and directly below the icon.
	d.PositionChild(navigationDestinationLabelID, ui.Offset{Dx: halfWidth(size) - halfWidth(labelSize), Dy: iconYPosition + iconSize.Height})
}

func (d *navigationDestinationLayoutDelegate) ShouldRelayout(old rendering.MultiChildLayoutDelegate) bool {
	return old.(*navigationDestinationLayoutDelegate).animation != d.animation
}

// statusTransitionBuilder rebuilds when the status of its animation changes.
//
// Flutter: material/navigation_bar.dart _StatusTransitionWidgetBuilder
type statusTransitionBuilder struct {
	widgets.StatefulWidget
	animation animation.Animation[float64]
	builder   widgets.TransitionBuilder
	child     widgets.Widget
}

func newStatusTransitionBuilder(a animation.Animation[float64], builder widgets.TransitionBuilder, child widgets.Widget) *statusTransitionBuilder {
	return &statusTransitionBuilder{StatefulWidget: widgets.NewStatefulWidgetBase(), animation: a, builder: builder, child: child}
}

func (w *statusTransitionBuilder) CreateState() widgets.State { return &statusTransitionBuilderState{} }

type statusTransitionBuilderState struct {
	widgets.StateOf[*statusTransitionBuilder]
	handle animation.StatusListenerHandle
}

func (s *statusTransitionBuilderState) InitState() {
	s.handle = s.Widget().animation.AddStatusListener(s.animationStatusChanged)
}

func (s *statusTransitionBuilderState) DidUpdateWidget(old *statusTransitionBuilder) {
	if s.Widget().animation != old.animation {
		old.animation.RemoveStatusListener(s.handle)
		s.handle = s.Widget().animation.AddStatusListener(s.animationStatusChanged)
	}
}

func (s *statusTransitionBuilderState) Dispose() {
	s.Widget().animation.RemoveStatusListener(s.handle)
}

func (s *statusTransitionBuilderState) animationStatusChanged(animation.AnimationStatus) {
	s.SetState(func() {})
}

func (s *statusTransitionBuilderState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	return w.builder(bc, w.child)
}

// selectableAnimatedBuilder builds widgets that need to be animated from 0
// (unselected) to 1.0 (selected): it manages an AnimationController and passes
// it down to the builder. When isSelected changes while the widget is animating,
// the animation is reversed until it is 0 or 1 again, unless alwaysDoFullAnimation
// is true: then the animation resets to 0 or 1 before it begins, so that the full
// animation is done.
//
// Flutter: material/navigation_bar.dart _SelectableAnimatedBuilder
type selectableAnimatedBuilder struct {
	widgets.StatefulWidget
	isSelected            bool
	duration              time.Duration
	alwaysDoFullAnimation bool
	builder               func(widgets.BuildContext, animation.Animation[float64]) widgets.Widget
}

func newSelectableAnimatedBuilder(isSelected bool, duration time.Duration, alwaysDoFullAnimation bool,
	builder func(widgets.BuildContext, animation.Animation[float64]) widgets.Widget) *selectableAnimatedBuilder {
	return &selectableAnimatedBuilder{StatefulWidget: widgets.NewStatefulWidgetBase(), isSelected: isSelected, duration: duration,
		alwaysDoFullAnimation: alwaysDoFullAnimation, builder: builder}
}

func (w *selectableAnimatedBuilder) CreateState() widgets.State {
	return &selectableAnimatedBuilderState{}
}

type selectableAnimatedBuilderState struct {
	widgets.StateOf[*selectableAnimatedBuilder]
	widgets.SingleTickerProvider
	controller *animation.AnimationController
}

func (s *selectableAnimatedBuilderState) InitState() {
	w := s.Widget()
	value := 0.0
	if w.isSelected {
		value = 1.0
	}
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(w.duration), Value: foundation.Some(value)})
}

func (s *selectableAnimatedBuilderState) DidUpdateWidget(old *selectableAnimatedBuilder) {
	w := s.Widget()
	if old.duration != w.duration {
		s.controller.Duration = foundation.Some(w.duration)
	}
	if old.isSelected != w.isSelected {
		var from []float64
		if w.isSelected {
			if w.alwaysDoFullAnimation {
				from = []float64{0}
			}
			s.controller.Forward(from...)
		} else {
			if w.alwaysDoFullAnimation {
				from = []float64{1}
			}
			s.controller.Reverse(from...)
		}
	}
}

func (s *selectableAnimatedBuilderState) Dispose() { s.controller.Dispose() }

func (s *selectableAnimatedBuilderState) Build(bc widgets.BuildContext) widgets.Widget {
	return s.Widget().builder(bc, s.controller)
}

// curvedAnimationBuilder applies a curve to an animation, a different one for
// each direction that the animation was started in.
//
// Flutter: material/navigation_bar.dart _CurvedAnimationBuilder
type curvedAnimationBuilder struct {
	widgets.StatefulWidget
	animation    animation.Animation[float64]
	curve        animation.Curve
	reverseCurve animation.Curve
	builder      func(widgets.BuildContext, animation.Animation[float64]) widgets.Widget
}

func newCurvedAnimationBuilder(a animation.Animation[float64], curve, reverseCurve animation.Curve,
	builder func(widgets.BuildContext, animation.Animation[float64]) widgets.Widget) *curvedAnimationBuilder {
	return &curvedAnimationBuilder{StatefulWidget: widgets.NewStatefulWidgetBase(), animation: a, curve: curve, reverseCurve: reverseCurve, builder: builder}
}

func (w *curvedAnimationBuilder) CreateState() widgets.State { return &curvedAnimationBuilderState{} }

type curvedAnimationBuilderState struct {
	widgets.StateOf[*curvedAnimationBuilder]
	animationDirection animation.AnimationStatus
	preservedDirection foundation.Opt[animation.AnimationStatus]
	handle             animation.StatusListenerHandle
}

func (s *curvedAnimationBuilderState) InitState() {
	a := s.Widget().animation
	s.animationDirection = a.Status()
	s.updateStatus(a.Status())
	s.handle = a.AddStatusListener(s.updateStatus)
}

func (s *curvedAnimationBuilderState) Dispose() {
	s.Widget().animation.RemoveStatusListener(s.handle)
}

// updateStatus keeps track of the current animation status, as well as the
// "preserved direction" when the animation changes direction mid animation. The
// preserved direction is reset when the animation finishes in either direction.
func (s *curvedAnimationBuilderState) updateStatus(status animation.AnimationStatus) {
	if s.animationDirection != status {
		s.SetState(func() { s.animationDirection = status })
	}
	switch status {
	case animation.AnimationStatusForward, animation.AnimationStatusReverse:
		if s.preservedDirection.IsNone() {
			s.SetState(func() { s.preservedDirection = foundation.Some(status) })
		}
	case animation.AnimationStatusCompleted, animation.AnimationStatusDismissed:
		s.SetState(func() { s.preservedDirection = foundation.None[animation.AnimationStatus]() })
	}
}

func (s *curvedAnimationBuilderState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	shouldUseForwardCurve := s.preservedDirection.OrElse(s.animationDirection) != animation.AnimationStatusReverse
	curve := w.reverseCurve
	if shouldUseForwardCurve {
		curve = w.curve
	}
	return w.builder(bc, animation.NewCurveTween(curve).Animate(w.animation))
}

func navigationBarDefaults(bc widgets.BuildContext) NavigationBarThemeData {
	theme := ThemeOf(bc)
	if theme.UseMaterial3 {
		return navigationBarDefaultsM3(theme)
	}
	return navigationBarDefaultsM2(theme)
}

// navigationBarDefaultsM2 are the hand coded defaults based on Material Design 2.
//
// Flutter: material/navigation_bar.dart _NavigationBarDefaultsM2
func navigationBarDefaultsM2(theme *ThemeData) NavigationBarThemeData {
	colors := theme.ColorScheme
	return NavigationBarThemeData{
		Height:         foundation.Some(80.0),
		Elevation:      foundation.Some(0.0),
		IndicatorShape: painting.NewRoundedRectangleBorder(painting.RoundedRectangleBorderOpts{BorderRadius: painting.NewBorderRadiusAll(ui.NewRadiusCircular(16))}),
		LabelBehavior:  NavigationDestinationLabelBehaviorAlwaysShow,
		// With Material 2, the NavigationBar uses an overlay blend for the default
		// color regardless of light/dark mode.
		BackgroundColor: foundation.Some(ElevationOverlayColorWithOverlay(colors.Surface(), colors.OnSurface(), 3.0)),
		IconTheme: widgets.NewWidgetStatePropertyAll(widgets.NewIconThemeData(widgets.IconThemeDataOpts{
			Size: foundation.Some(24.0), Color: foundation.Some(colors.OnSurface())})),
		IndicatorColor: foundation.Some(colors.Secondary().WithOpacity(0.24)),
		LabelTextStyle: widgets.NewWidgetStatePropertyAll(theme.TextTheme.LabelSmall.CopyWith(painting.TextStyleCopy{Color: foundation.Some(colors.OnSurface())})),
		LabelPadding:   painting.NewEdgeInsetsFromLTRB(0, 4, 0, 0),
	}
}

// navigationBarDefaultsM3 are the Material 3 defaults.
//
// Flutter: material/navigation_bar.dart _NavigationBarDefaultsM3
func navigationBarDefaultsM3(theme *ThemeData) NavigationBarThemeData {
	colors, textTheme := theme.ColorScheme, theme.TextTheme
	return NavigationBarThemeData{
		Height:           foundation.Some(80.0),
		Elevation:        foundation.Some(3.0),
		LabelBehavior:    NavigationDestinationLabelBehaviorAlwaysShow,
		BackgroundColor:  foundation.Some(colors.SurfaceContainer()),
		ShadowColor:      foundation.Some(Colors.Transparent),
		SurfaceTintColor: foundation.Some(Colors.Transparent),
		IconTheme: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) widgets.IconThemeData {
			color := colors.OnSurfaceVariant()
			switch {
			case states.Has(widgets.WidgetStateDisabled):
				color = colors.OnSurfaceVariant().WithOpacity(0.38)
			case states.Has(widgets.WidgetStateSelected):
				color = colors.OnSecondaryContainer()
			}
			return widgets.NewIconThemeData(widgets.IconThemeDataOpts{Size: foundation.Some(24.0), Color: foundation.Some(color)})
		}),
		IndicatorColor: foundation.Some(colors.SecondaryContainer()),
		IndicatorShape: painting.NewStadiumBorder(),
		LabelTextStyle: widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) *painting.TextStyle {
			color := colors.OnSurfaceVariant()
			switch {
			case states.Has(widgets.WidgetStateDisabled):
				color = colors.OnSurfaceVariant().WithOpacity(0.38)
			case states.Has(widgets.WidgetStateSelected):
				color = colors.OnSurface()
			}
			return textTheme.LabelMedium.Apply(painting.TextStyleApply{Color: foundation.Some(color)})
		}),
		LabelPadding: painting.NewEdgeInsetsFromLTRB(0, 4, 0, 0),
	}
}
