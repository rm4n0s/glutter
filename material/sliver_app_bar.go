// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// sliverAppVariant is the size of a SliverAppBar.
//
// Flutter: material/app_bar.dart _SliverAppVariant
type sliverAppVariant int

const (
	sliverAppVariantSmall sliverAppVariant = iota
	sliverAppVariantMedium
	sliverAppVariantLarge
)

// scrollUnderFlexibleConfig has the sizes and styles of the medium and large app
// bars.
//
// Flutter: material/app_bar.dart _ScrollUnderFlexibleConfig
type scrollUnderFlexibleConfig struct {
	collapsedHeight, expandedHeight float64
	expandedTextStyle               *painting.TextStyle
	expandedTitlePadding            painting.EdgeInsets
}

// Flutter: material/app_bar.dart _MediumScrollUnderFlexibleConfig
const (
	mediumScrollUnderCollapsedHeight = 64.0
	mediumScrollUnderExpandedHeight  = 112.0
)

// Flutter: material/app_bar.dart _LargeScrollUnderFlexibleConfig
const (
	largeScrollUnderCollapsedHeight = 64.0
	largeScrollUnderExpandedHeight  = 152.0
)

// Flutter: material/app_bar.dart _MediumScrollUnderFlexibleConfig
func mediumScrollUnderFlexibleConfig(bc widgets.BuildContext) scrollUnderFlexibleConfig {
	theme := ThemeOf(bc)
	return scrollUnderFlexibleConfig{
		collapsedHeight: mediumScrollUnderCollapsedHeight, expandedHeight: mediumScrollUnderExpandedHeight,
		expandedTextStyle:    applyColorOrNil(theme.TextTheme.HeadlineSmall, theme.ColorScheme.OnSurface()),
		expandedTitlePadding: painting.NewEdgeInsetsFromLTRB(16, 0, 16, 20),
	}
}

// Flutter: material/app_bar.dart _LargeScrollUnderFlexibleConfig
func largeScrollUnderFlexibleConfig(bc widgets.BuildContext) scrollUnderFlexibleConfig {
	theme := ThemeOf(bc)
	return scrollUnderFlexibleConfig{
		collapsedHeight: largeScrollUnderCollapsedHeight, expandedHeight: largeScrollUnderExpandedHeight,
		expandedTextStyle:    applyColorOrNil(theme.TextTheme.HeadlineMedium, theme.ColorScheme.OnSurface()),
		expandedTitlePadding: painting.NewEdgeInsetsFromLTRB(16, 0, 16, 28),
	}
}

func applyColorOrNil(style *painting.TextStyle, color ui.Color) *painting.TextStyle {
	if style == nil {
		return nil
	}
	return style.Apply(painting.TextStyleApply{Color: foundation.Some(color)})
}

// SliverAppBarOpts are the optional parameters of NewSliverAppBar,
// NewSliverAppBarMedium and NewSliverAppBarLarge.
//
// Flutter: material/app_bar.dart SliverAppBar.new
type SliverAppBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Leading is the widget at the start of the toolbar, before the title.
	Leading widgets.Widget
	// NoAutomaticallyImplyLeading says that the app bar does not try to make a
	// leading widget (Flutter's automaticallyImplyLeading: false).
	NoAutomaticallyImplyLeading bool
	// Title is the primary widget of the toolbar.
	Title widgets.Widget
	// Actions are the widgets at the end of the toolbar.
	Actions []widgets.Widget
	// NoAutomaticallyImplyActions says that the app bar does not add the end
	// drawer button by itself (Flutter's automaticallyImplyActions: false).
	NoAutomaticallyImplyActions bool
	// FlexibleSpace is stacked behind the toolbar and the bottom widget; its
	// height is the same as the app bar's.
	FlexibleSpace widgets.Widget
	// Bottom is the widget below the toolbar.
	Bottom widgets.PreferredSizeWidget
	// Elevation is the z-coordinate of the app bar when it is scrolled under.
	Elevation foundation.Opt[float64]
	// ScrolledUnderElevation is the elevation when content is scrolled under.
	ScrolledUnderElevation foundation.Opt[float64]
	// ShadowColor is the color of the shadow below the app bar.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the color of the surface tint overlay (Material 3).
	SurfaceTintColor foundation.Opt[ui.Color]
	// ForceElevated says whether to show the shadow even when the content is not
	// scrolled under.
	ForceElevated bool
	// BackgroundColor is the fill color of the app bar.
	BackgroundColor foundation.Opt[ui.Color]
	// ForegroundColor is the default color of the text and icons.
	ForegroundColor foundation.Opt[ui.Color]
	// IconTheme is the color, opacity and size of the icons of the toolbar.
	IconTheme foundation.Opt[widgets.IconThemeData]
	// ActionsIconTheme is the icon theme of the actions.
	ActionsIconTheme foundation.Opt[widgets.IconThemeData]
	// NoPrimary says that the app bar is not at the top of the screen, so it does
	// not make room for the status bar (Flutter's primary: false).
	NoPrimary bool
	// CenterTitle says whether the title is centered.
	CenterTitle foundation.Opt[bool]
	// ExcludeHeaderSemantics says whether the flexible space is excluded from the
	// header semantics.
	ExcludeHeaderSemantics bool
	// TitleSpacing is the spacing around the title on the sides with other widgets.
	TitleSpacing foundation.Opt[float64]
	// CollapsedHeight is the height when the app bar is collapsed.
	CollapsedHeight foundation.Opt[float64]
	// ExpandedHeight is the height when the app bar is fully expanded.
	ExpandedHeight foundation.Opt[float64]
	// Floating says whether the app bar becomes visible as soon as the user
	// scrolls towards it.
	Floating bool
	// Pinned says whether the app bar stays visible at the start of the scroll
	// view; the default is false for the small app bar and true for the others.
	Pinned foundation.Opt[bool]
	// Snap says that a floating app bar snaps to be fully visible or hidden
	// (only for floating app bars).
	Snap bool
	// Stretch says whether the app bar stretches when the user overscrolls.
	Stretch bool
	// StretchTriggerOffset is the overscroll offset that triggers OnStretchTrigger;
	// the default is 100.
	StretchTriggerOffset foundation.Opt[float64]
	// OnStretchTrigger is called when the overscroll reaches the offset.
	OnStretchTrigger func()
	// Shape is the shape of the Material of the app bar.
	Shape painting.ShapeBorder
	// ToolbarHeight is the height of the toolbar; the default is ToolbarHeight (56)
	// and, for the medium and large app bars, 64.
	ToolbarHeight foundation.Opt[float64]
	// LeadingWidth is the width of the leading widget.
	LeadingWidth foundation.Opt[float64]
	// ToolbarTextStyle is the default text style of the toolbar widgets.
	ToolbarTextStyle *painting.TextStyle
	// TitleTextStyle is the default text style of the title.
	TitleTextStyle *painting.TextStyle
	// SystemOverlayStyle is the style of the system status bar.
	SystemOverlayStyle foundation.Opt[services.SystemUiOverlayStyle]
	// ForceMaterialTransparency says that the Material is transparent.
	ForceMaterialTransparency bool
	// NoUseDefaultSemanticsOrder says that the semantics order of the app bar is
	// not the default (Flutter's useDefaultSemanticsOrder: false).
	NoUseDefaultSemanticsOrder bool
	// ClipBehavior is the clip of the Material of the app bar.
	ClipBehavior ui.Clip
	// ActionsPadding is the padding around the actions.
	ActionsPadding painting.EdgeInsetsGeometry
}

// SliverAppBar is a Material Design app bar that integrates with a CustomScrollView:
// it can grow, shrink, float and stay pinned.
//
// Deviation: the small app bar's Pinned, Floating and Snap fields are in the
// options; the medium and large constructors differ by their defaults (SPEC
// §12.10).
//
// Flutter: material/app_bar.dart SliverAppBar
type SliverAppBar struct {
	widgets.StatefulWidget
	o       SliverAppBarOpts
	variant sliverAppVariant
}

func newSliverAppBar(variant sliverAppVariant, opts []SliverAppBarOpts) *SliverAppBar {
	o := firstOpts(opts)
	defaultToolbarHeight := ToolbarHeight
	switch variant {
	case sliverAppVariantMedium:
		defaultToolbarHeight = mediumScrollUnderCollapsedHeight
		o.Pinned = foundation.Some(o.Pinned.OrElse(true))
	case sliverAppVariantLarge:
		defaultToolbarHeight = largeScrollUnderCollapsedHeight
		o.Pinned = foundation.Some(o.Pinned.OrElse(true))
	}
	o.ToolbarHeight = foundation.Some(o.ToolbarHeight.OrElse(defaultToolbarHeight))
	if foundation.DebugMode {
		if !o.Floating && o.Snap {
			panic(errors.New("SliverAppBarSnapNotFloating", "The \"snap\" argument only makes sense for floating app bars."))
		}
		if !(o.StretchTriggerOffset.OrElse(100.0) > 0.0) {
			panic(errors.New("SliverAppBarStretchTriggerOffset", "stretchTriggerOffset must be greater than zero",
				"stretchTriggerOffset", o.StretchTriggerOffset.Value()))
		}
		if o.CollapsedHeight.IsSome() && o.CollapsedHeight.Value() < o.ToolbarHeight.Value() {
			panic(errors.New("SliverAppBarCollapsedHeight", "The \"collapsedHeight\" argument has to be larger than or equal to [toolbarHeight].",
				"collapsedHeight", o.CollapsedHeight.Value(), "toolbarHeight", o.ToolbarHeight.Value()))
		}
	}
	return &SliverAppBar{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o, variant: variant}
}

// NewSliverAppBar creates a standard app bar. In debug builds it panics with the
// tags SliverAppBarSnapNotFloating, SliverAppBarStretchTriggerOffset and
// SliverAppBarCollapsedHeight on invalid options.
//
// Flutter: material/app_bar.dart SliverAppBar.new
func NewSliverAppBar(opts ...SliverAppBarOpts) *SliverAppBar {
	return newSliverAppBar(sliverAppVariantSmall, opts)
}

// NewSliverAppBarMedium creates an app bar with a medium-size expanded title that
// shrinks to the toolbar title while scrolling. It is pinned by default.
//
// Flutter: material/app_bar.dart SliverAppBar.medium
func NewSliverAppBarMedium(opts ...SliverAppBarOpts) *SliverAppBar {
	return newSliverAppBar(sliverAppVariantMedium, opts)
}

// NewSliverAppBarLarge creates an app bar with a large expanded title that
// shrinks to the toolbar title while scrolling. It is pinned by default.
//
// Flutter: material/app_bar.dart SliverAppBar.large
func NewSliverAppBarLarge(opts ...SliverAppBarOpts) *SliverAppBar {
	return newSliverAppBar(sliverAppVariantLarge, opts)
}

// Title is the primary widget of the toolbar.
//
// Flutter: material/app_bar.dart SliverAppBar.title
func (w *SliverAppBar) Title() widgets.Widget { return w.o.Title }

// Pinned is whether the app bar stays visible at the start of the scroll view.
//
// Flutter: material/app_bar.dart SliverAppBar.pinned
func (w *SliverAppBar) Pinned() bool { return w.o.Pinned.OrElse(false) }

// Floating is whether the app bar becomes visible as soon as the user scrolls
// towards it.
//
// Flutter: material/app_bar.dart SliverAppBar.floating
func (w *SliverAppBar) Floating() bool { return w.o.Floating }

// Snap is whether a floating app bar snaps.
//
// Flutter: material/app_bar.dart SliverAppBar.snap
func (w *SliverAppBar) Snap() bool { return w.o.Snap }

// ToolbarHeight is the height of the toolbar.
//
// Flutter: material/app_bar.dart SliverAppBar.toolbarHeight
func (w *SliverAppBar) ToolbarHeight() float64 { return w.o.ToolbarHeight.Value() }

// CreateState creates the state.
//
// Flutter: material/app_bar.dart SliverAppBar.createState
func (w *SliverAppBar) CreateState() widgets.State { return &sliverAppBarState{} }

// sliverAppBarState is only stateful because it owns the TickerProvider used by
// the snap animation of the floating app bar.
//
// Flutter: material/app_bar.dart _SliverAppBarState
type sliverAppBarState struct {
	widgets.StateOf[*SliverAppBar]
	widgets.TickerProvider
	snapConfiguration         *rendering.FloatingHeaderSnapConfiguration
	stretchConfiguration      *rendering.OverScrollHeaderStretchConfiguration
	showOnScreenConfiguration *rendering.PersistentHeaderShowOnScreenConfiguration
}

func (s *sliverAppBarState) updateSnapConfiguration() {
	w := s.Widget()
	if w.o.Snap && w.o.Floating {
		s.snapConfiguration = rendering.NewFloatingHeaderSnapConfiguration(rendering.FloatingHeaderSnapConfigurationOpts{
			Curve: animation.Curves.EaseOut, Duration: foundation.Some(200 * time.Millisecond)})
	} else {
		s.snapConfiguration = nil
	}
	if w.o.Floating && w.o.Snap {
		s.showOnScreenConfiguration = rendering.NewPersistentHeaderShowOnScreenConfiguration(
			rendering.PersistentHeaderShowOnScreenConfigurationOpts{MinShowOnScreenExtent: foundation.Some(math.Inf(1))})
	} else {
		s.showOnScreenConfiguration = nil
	}
}

func (s *sliverAppBarState) updateStretchConfiguration() {
	w := s.Widget()
	if w.o.Stretch {
		s.stretchConfiguration = &rendering.OverScrollHeaderStretchConfiguration{
			StretchTriggerOffset: w.o.StretchTriggerOffset.OrElse(100.0), OnStretchTrigger: w.o.OnStretchTrigger}
	} else {
		s.stretchConfiguration = nil
	}
}

func (s *sliverAppBarState) InitState() {
	s.updateSnapConfiguration()
	s.updateStretchConfiguration()
}

func (s *sliverAppBarState) DidUpdateWidget(old *SliverAppBar) {
	w := s.Widget()
	if w.o.Snap != old.o.Snap || w.o.Floating != old.o.Floating {
		s.updateSnapConfiguration()
	}
	if w.o.Stretch != old.o.Stretch {
		s.updateStretchConfiguration()
	}
}

func (s *sliverAppBarState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	if !o.NoPrimary {
		widgets.DebugCheckHasMediaQuery(bc)
	}
	var bottomHeight float64
	if o.Bottom != nil {
		bottomHeight = o.Bottom.PreferredSize().Height
	}
	topPadding := 0.0
	if !o.NoPrimary {
		topPadding = widgets.MediaQueryPaddingOf(bc).Top
	}
	toolbarHeight := o.ToolbarHeight.Value()
	var collapsedHeight float64
	if w.Pinned() && o.Floating && o.Bottom != nil {
		collapsedHeight = o.CollapsedHeight.OrElse(0.0) + bottomHeight + topPadding
	} else {
		collapsedHeight = o.CollapsedHeight.OrElse(toolbarHeight) + bottomHeight + topPadding
	}
	expandedHeight := o.ExpandedHeight
	effectiveCollapsedHeight := collapsedHeight
	flexibleSpace := o.FlexibleSpace
	switch w.variant {
	case sliverAppVariantMedium, sliverAppVariantLarge:
		builder := mediumScrollUnderFlexibleConfig
		config := scrollUnderFlexibleConfig{collapsedHeight: mediumScrollUnderCollapsedHeight, expandedHeight: mediumScrollUnderExpandedHeight}
		if w.variant == sliverAppVariantLarge {
			builder = largeScrollUnderFlexibleConfig
			config = scrollUnderFlexibleConfig{collapsedHeight: largeScrollUnderCollapsedHeight, expandedHeight: largeScrollUnderExpandedHeight}
		}
		expandedHeight = foundation.Some(o.ExpandedHeight.OrElse(config.expandedHeight + bottomHeight))
		effectiveCollapsedHeight = o.CollapsedHeight.OrElse(topPadding + config.collapsedHeight + bottomHeight)
		if flexibleSpace == nil {
			flexibleSpace = &scrollUnderFlexibleSpace{
				StatelessWidget: widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}),
				title:           o.Title, foregroundColor: o.ForegroundColor, configBuilder: builder,
				titleTextStyle: o.TitleTextStyle, bottomHeight: bottomHeight,
			}
		}
	}

	return widgets.NewMediaQueryRemovePadding(bc, widgets.RemoveSides{Bottom: true},
		widgets.NewSliverPersistentHeader(&sliverAppBarDelegate{
			vsync: s, leading: o.Leading, automaticallyImplyLeading: !o.NoAutomaticallyImplyLeading, title: o.Title,
			actions: o.Actions, automaticallyImplyActions: !o.NoAutomaticallyImplyActions, flexibleSpace: flexibleSpace,
			bottom: o.Bottom, bottomHeight: bottomHeight, elevation: o.Elevation, scrolledUnderElevation: o.ScrolledUnderElevation,
			shadowColor: o.ShadowColor, surfaceTintColor: o.SurfaceTintColor, forceElevated: o.ForceElevated,
			backgroundColor: o.BackgroundColor, foregroundColor: o.ForegroundColor, iconTheme: o.IconTheme,
			actionsIconTheme: o.ActionsIconTheme, primary: !o.NoPrimary, centerTitle: o.CenterTitle,
			excludeHeaderSemantics: o.ExcludeHeaderSemantics, titleSpacing: o.TitleSpacing, expandedHeight: expandedHeight,
			collapsedHeight: effectiveCollapsedHeight, topPadding: topPadding, floating: o.Floating, pinned: w.Pinned(),
			shape: o.Shape, snapConfiguration: s.snapConfiguration, stretchConfiguration: s.stretchConfiguration,
			showOnScreenConfiguration: s.showOnScreenConfiguration, toolbarHeight: o.ToolbarHeight, leadingWidth: o.LeadingWidth,
			toolbarTextStyle: o.ToolbarTextStyle, titleTextStyle: o.TitleTextStyle, systemOverlayStyle: o.SystemOverlayStyle,
			forceMaterialTransparency: o.ForceMaterialTransparency, useDefaultSemanticsOrder: !o.NoUseDefaultSemanticsOrder,
			clipBehavior: o.ClipBehavior, variant: w.variant, accessibleNavigation: widgets.MediaQueryAccessibleNavigationOf(bc),
			actionsPadding: o.ActionsPadding,
		}, widgets.SliverPersistentHeaderOpts{Floating: o.Floating, Pinned: w.Pinned()}))
}

// sliverAppBarDelegate builds the AppBar of a SliverAppBar for each scroll offset.
//
// Flutter: material/app_bar.dart _SliverAppBarDelegate
type sliverAppBarDelegate struct {
	widgets.SliverPersistentHeaderDelegateBase
	leading                   widgets.Widget
	automaticallyImplyLeading bool
	title                     widgets.Widget
	actions                   []widgets.Widget
	automaticallyImplyActions bool
	flexibleSpace             widgets.Widget
	bottom                    widgets.PreferredSizeWidget
	bottomHeight              float64
	elevation                 foundation.Opt[float64]
	scrolledUnderElevation    foundation.Opt[float64]
	shadowColor               foundation.Opt[ui.Color]
	surfaceTintColor          foundation.Opt[ui.Color]
	forceElevated             bool
	backgroundColor           foundation.Opt[ui.Color]
	foregroundColor           foundation.Opt[ui.Color]
	iconTheme                 foundation.Opt[widgets.IconThemeData]
	actionsIconTheme          foundation.Opt[widgets.IconThemeData]
	primary                   bool
	centerTitle               foundation.Opt[bool]
	excludeHeaderSemantics    bool
	titleSpacing              foundation.Opt[float64]
	expandedHeight            foundation.Opt[float64]
	collapsedHeight           float64
	topPadding                float64
	floating                  bool
	pinned                    bool
	shape                     painting.ShapeBorder
	toolbarHeight             foundation.Opt[float64]
	leadingWidth              foundation.Opt[float64]
	toolbarTextStyle          *painting.TextStyle
	titleTextStyle            *painting.TextStyle
	systemOverlayStyle        foundation.Opt[services.SystemUiOverlayStyle]
	forceMaterialTransparency bool
	useDefaultSemanticsOrder  bool
	clipBehavior              ui.Clip
	variant                   sliverAppVariant
	accessibleNavigation      bool
	actionsPadding            painting.EdgeInsetsGeometry
	vsync                     scheduler.TickerProvider
	snapConfiguration         *rendering.FloatingHeaderSnapConfiguration
	stretchConfiguration      *rendering.OverScrollHeaderStretchConfiguration
	showOnScreenConfiguration *rendering.PersistentHeaderShowOnScreenConfiguration
}

func (d *sliverAppBarDelegate) MinExtent() float64 { return d.collapsedHeight }

func (d *sliverAppBarDelegate) MaxExtent() float64 {
	return math.Max(d.topPadding+d.expandedHeight.OrElse(d.toolbarHeight.OrElse(ToolbarHeight)+d.bottomHeight), d.MinExtent())
}

func (d *sliverAppBarDelegate) Vsync() scheduler.TickerProvider { return d.vsync }

func (d *sliverAppBarDelegate) SnapConfiguration() *rendering.FloatingHeaderSnapConfiguration {
	return d.snapConfiguration
}

func (d *sliverAppBarDelegate) StretchConfiguration() *rendering.OverScrollHeaderStretchConfiguration {
	return d.stretchConfiguration
}

func (d *sliverAppBarDelegate) ShowOnScreenConfiguration() *rendering.PersistentHeaderShowOnScreenConfiguration {
	return d.showOnScreenConfiguration
}

func (d *sliverAppBarDelegate) Build(bc widgets.BuildContext, shrinkOffset float64, overlapsContent bool) widgets.Widget {
	minExtent, maxExtent := d.MinExtent(), d.MaxExtent()
	toolbarHeight := d.toolbarHeight.OrElse(ToolbarHeight)
	visibleMainHeight := maxExtent - shrinkOffset - d.topPadding
	extraToolbarHeight := math.Max(minExtent-d.bottomHeight-d.topPadding-toolbarHeight, 0.0)
	visibleToolbarHeight := visibleMainHeight - d.bottomHeight - extraToolbarHeight

	isScrolledUnder := overlapsContent || d.forceElevated || (d.pinned && shrinkOffset > maxExtent-minExtent)
	isPinnedWithOpacityFade := d.pinned && d.floating && d.bottom != nil && extraToolbarHeight == 0.0
	toolbarOpacity := 1.0
	if !d.accessibleNavigation && (!d.pinned || isPinnedWithOpacityFade) {
		toolbarOpacity = ui.ClampDouble(visibleToolbarHeight/toolbarHeight, 0.0, 1.0)
	}
	effectiveTitle := d.title
	if d.variant != sliverAppVariantSmall {
		opacity := 0.0
		if isScrolledUnder {
			opacity = 1.0
		}
		effectiveTitle = widgets.NewAnimatedOpacity(opacity, 500*time.Millisecond, d.title,
			widgets.AnimatedOpacityOpts{Curve: animation.NewCubic(0.2, 0.0, 0.0, 1.0)})
	}

	flexibleSpace := d.flexibleSpace
	if d.title == nil && d.flexibleSpace != nil && !d.excludeHeaderSemantics {
		flexibleSpace = widgets.NewSemantics(d.flexibleSpace, widgets.SemanticsOpts{Header: foundation.Some(true)})
	}
	elevation := foundation.Some(0.0)
	if isScrolledUnder {
		elevation = d.elevation
	}
	bottomOpacity := 1.0
	if !d.pinned {
		bottomOpacity = ui.ClampDouble(visibleMainHeight/d.bottomHeight, 0.0, 1.0)
	}
	return FlexibleSpaceBarCreateSettings(math.Max(minExtent, maxExtent-shrinkOffset),
		NewAppBar(AppBarOpts{
			ClipBehavior: d.clipBehavior, Leading: d.leading, NoAutomaticallyImplyLeading: !d.automaticallyImplyLeading,
			Title: effectiveTitle, Actions: d.actions, NoAutomaticallyImplyActions: !d.automaticallyImplyActions,
			FlexibleSpace: flexibleSpace, Bottom: d.bottom, Elevation: elevation, ScrolledUnderElevation: d.scrolledUnderElevation,
			ShadowColor: d.shadowColor, SurfaceTintColor: d.surfaceTintColor, BackgroundColor: d.backgroundColor,
			ForegroundColor: d.foregroundColor, IconTheme: d.iconTheme, ActionsIconTheme: d.actionsIconTheme,
			NoPrimary: !d.primary, CenterTitle: d.centerTitle, ExcludeHeaderSemantics: d.excludeHeaderSemantics,
			TitleSpacing: d.titleSpacing, Shape: d.shape, ToolbarOpacity: foundation.Some(toolbarOpacity),
			BottomOpacity: foundation.Some(bottomOpacity), ToolbarHeight: d.toolbarHeight, LeadingWidth: d.leadingWidth,
			ToolbarTextStyle: d.toolbarTextStyle, TitleTextStyle: d.titleTextStyle, SystemOverlayStyle: d.systemOverlayStyle,
			ForceMaterialTransparency: d.forceMaterialTransparency, NoUseDefaultSemanticsOrder: !d.useDefaultSemanticsOrder,
			ActionsPadding: d.actionsPadding,
		}),
		FlexibleSpaceBarSettingsOpts{
			MinExtent: foundation.Some(minExtent), MaxExtent: foundation.Some(maxExtent), ToolbarOpacity: foundation.Some(toolbarOpacity),
			IsScrolledUnder: foundation.Some(isScrolledUnder), HasLeading: foundation.Some(d.leading != nil || d.automaticallyImplyLeading),
		})
}

func (d *sliverAppBarDelegate) ShouldRebuild(old widgets.SliverPersistentHeaderDelegate) bool {
	o, ok := old.(*sliverAppBarDelegate)
	if !ok {
		return true
	}
	return d.leading != o.leading || d.automaticallyImplyLeading != o.automaticallyImplyLeading || d.title != o.title ||
		!slices.Equal(d.actions, o.actions) || d.automaticallyImplyActions != o.automaticallyImplyActions ||
		d.flexibleSpace != o.flexibleSpace || d.bottom != o.bottom || d.bottomHeight != o.bottomHeight ||
		d.elevation != o.elevation || d.shadowColor != o.shadowColor || d.backgroundColor != o.backgroundColor ||
		d.foregroundColor != o.foregroundColor || !iconThemeOptEqual(d.iconTheme, o.iconTheme) ||
		!iconThemeOptEqual(d.actionsIconTheme, o.actionsIconTheme) || d.primary != o.primary || d.centerTitle != o.centerTitle ||
		d.titleSpacing != o.titleSpacing || d.expandedHeight != o.expandedHeight || d.topPadding != o.topPadding ||
		d.pinned != o.pinned || d.floating != o.floating || d.vsync != o.vsync || d.snapConfiguration != o.snapConfiguration ||
		d.stretchConfiguration != o.stretchConfiguration || d.showOnScreenConfiguration != o.showOnScreenConfiguration ||
		d.forceElevated != o.forceElevated || d.toolbarHeight != o.toolbarHeight || d.leadingWidth != o.leadingWidth ||
		!textStyleEqual(d.toolbarTextStyle, o.toolbarTextStyle) || !textStyleEqual(d.titleTextStyle, o.titleTextStyle) ||
		d.systemOverlayStyle != o.systemOverlayStyle || d.forceMaterialTransparency != o.forceMaterialTransparency ||
		d.useDefaultSemanticsOrder != o.useDefaultSemanticsOrder || d.accessibleNavigation != o.accessibleNavigation ||
		d.actionsPadding != o.actionsPadding
}

// scrollUnderFlexibleSpace is the flexible space of the medium and large app
// bars: the expanded title, which scrolls under the toolbar.
//
// Flutter: material/app_bar.dart _ScrollUnderFlexibleSpace
type scrollUnderFlexibleSpace struct {
	widgets.StatelessWidget
	title           widgets.Widget
	foregroundColor foundation.Opt[ui.Color]
	configBuilder   func(bc widgets.BuildContext) scrollUnderFlexibleConfig
	titleTextStyle  *painting.TextStyle
	bottomHeight    float64
}

func (w *scrollUnderFlexibleSpace) Build(bc widgets.BuildContext) widgets.Widget {
	appBarTheme := AppBarThemeOf(bc)
	defaults := appBarDefaults(ThemeOf(bc))
	settings, ok := widgets.DependOn[*FlexibleSpaceBarSettings](bc)
	if !ok {
		panic(errors.New("ScrollUnderFlexibleSpaceNoSettings", "a medium or large SliverAppBar needs FlexibleSpaceBarSettings above its flexible space"))
	}
	config := w.configBuilder(bc)

	expandedTextStyle := w.titleTextStyle
	if expandedTextStyle == nil {
		expandedTextStyle = appBarTheme.TitleTextStyle
	}
	if expandedTextStyle == nil && config.expandedTextStyle != nil {
		expandedTextStyle = config.expandedTextStyle.CopyWith(painting.TextStyleCopy{
			Color: foundation.Some(firstColor(w.foregroundColor, appBarTheme.ForegroundColor, defaults.ForegroundColor))})
	}

	var expandedTitle widgets.Widget
	switch {
	case w.title == nil:
	case expandedTextStyle == nil:
		expandedTitle = w.title
	default:
		expandedTitle = widgets.NewDefaultTextStyle(expandedTextStyle, w.title)
	}

	resolvedTitlePadding := config.expandedTitlePadding
	expandedTitlePadding := resolvedTitlePadding
	if w.bottomHeight > 0 {
		expandedTitlePadding = resolvedTitlePadding.CopyWith(painting.EdgeInsetsCopy{Bottom: foundation.Some(0.0)})
	}

	// Set the maximum text scale factor of the title to keep the visual hierarchy
	// the same even with larger font sizes. To opt out, wrap the title widget in a
	// MediaQuery with a different TextScaler.
	children := []widgets.Widget{
		widgets.NewPadding(painting.EdgeInsets{Top: settings.MinExtent() - w.bottomHeight}, nil),
		widgets.NewFlexible(widgets.NewClipRect(newExpandedTitleWithPadding(expandedTitlePadding, settings.MaxExtent()-settings.MinExtent(), expandedTitle))),
	}
	if w.bottomHeight > 0 {
		// Reserve space for AppBar.bottom, which is a sibling of this widget, on the
		// parent Stack.
		children = append(children, widgets.NewPadding(painting.EdgeInsets{Bottom: w.bottomHeight}, nil))
	}
	// This column assumes the full height of the parent Stack.
	return widgets.NewMediaQueryWithClampedTextScaling(widgets.NewColumn(children),
		widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: maxTitleTextScaleFactor})
}

// expandedTitleWithPadding bottom-start aligns its child (the expanded title
// widget), and insets the child according to the padding. It gives the child an
// infinite max height constraint, and attempts to limit the child's bounding box
// (without the padding) vertically to within [0, maxExtent], to make sure the
// child is visible when the AppBar is fully expanded.
//
// Flutter: material/app_bar.dart _ExpandedTitleWithPadding
type expandedTitleWithPadding struct {
	widgets.SingleChildRenderObjectWidget
	padding   painting.EdgeInsets
	maxExtent float64
}

func newExpandedTitleWithPadding(padding painting.EdgeInsets, maxExtent float64, child widgets.Widget) *expandedTitleWithPadding {
	return &expandedTitleWithPadding{SingleChildRenderObjectWidget: widgets.NewSingleChildRenderObjectWidgetBase(child), padding: padding, maxExtent: maxExtent}
}

func (w *expandedTitleWithPadding) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	return newRenderExpandedTitleBox(w.padding, painting.AlignmentDirectionalBottomStart.Resolve(widgets.DirectionalityOf(bc)), w.maxExtent)
}

func (w *expandedTitleWithPadding) UpdateRenderObject(bc widgets.BuildContext, renderObject rendering.RenderObject) {
	r := renderObject.(*renderExpandedTitleBox)
	r.setPadding(w.padding)
	r.setTitleAlignment(painting.AlignmentDirectionalBottomStart.Resolve(widgets.DirectionalityOf(bc)))
	r.setMaxExtent(w.maxExtent)
}

// renderExpandedTitleBox is the render object of expandedTitleWithPadding.
//
// Flutter: material/app_bar.dart _RenderExpandedTitleBox
type renderExpandedTitleBox struct {
	rendering.RenderShiftedBox
	padding        painting.EdgeInsets
	titleAlignment painting.Alignment
	maxExtent      float64
}

func newRenderExpandedTitleBox(padding painting.EdgeInsets, titleAlignment painting.Alignment, maxExtent float64) *renderExpandedTitleBox {
	r := &renderExpandedTitleBox{padding: padding, titleAlignment: titleAlignment, maxExtent: maxExtent}
	rendering.InitRenderShiftedBox(&r.RenderShiftedBox, r, nil)
	return r
}

func (r *renderExpandedTitleBox) setPadding(value painting.EdgeInsets) {
	if r.padding == value {
		return
	}
	r.padding = value
	r.MarkNeedsLayout()
}

func (r *renderExpandedTitleBox) setTitleAlignment(value painting.Alignment) {
	if r.titleAlignment == value {
		return
	}
	r.titleAlignment = value
	r.MarkNeedsLayout()
}

func (r *renderExpandedTitleBox) setMaxExtent(value float64) {
	if r.maxExtent == value {
		return
	}
	r.maxExtent = value
	r.MarkNeedsLayout()
}

func (r *renderExpandedTitleBox) hasChild() bool { return r.Child() != nil }

func (r *renderExpandedTitleBox) ComputeMaxIntrinsicHeight(width float64) float64 {
	if !r.hasChild() {
		return 0
	}
	return r.Child().GetMaxIntrinsicHeight(math.Max(0, width-r.padding.Horizontal())) + r.padding.Vertical()
}

func (r *renderExpandedTitleBox) ComputeMaxIntrinsicWidth(height float64) float64 {
	if !r.hasChild() {
		return 0
	}
	return r.Child().GetMaxIntrinsicWidth(math.Inf(1)) + r.padding.Horizontal()
}

func (r *renderExpandedTitleBox) ComputeMinIntrinsicHeight(width float64) float64 {
	if !r.hasChild() {
		return 0
	}
	return r.Child().GetMinIntrinsicHeight(math.Max(0, width-r.padding.Horizontal())) + r.padding.Vertical()
}

func (r *renderExpandedTitleBox) ComputeMinIntrinsicWidth(height float64) float64 {
	if !r.hasChild() {
		return 0
	}
	return r.Child().GetMinIntrinsicWidth(math.Inf(1)) + r.padding.Horizontal()
}

func (r *renderExpandedTitleBox) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	if !r.hasChild() {
		return ui.Size{}
	}
	return constraints.Biggest()
}

func (r *renderExpandedTitleBox) childOffsetFromSize(childSize, size ui.Size) ui.Offset {
	// yAdjustment is the minimum additional y offset to shift the child in the
	// visible vertical space when the AppBar is fully expanded. The goal is to
	// prevent the expanded title from being clipped when the expanded title widget
	// plus the bottom padding is too tall to fit in the flexible space (the top
	// padding is ignored since the expanded title is bottom-aligned).
	yAdjustment := ui.ClampDouble(childSize.Height+r.padding.Bottom-r.maxExtent, 0, r.padding.Bottom)
	offsetX := (r.titleAlignment.X+1)/2*(size.Width-r.padding.Horizontal()-childSize.Width) + r.padding.Left
	offsetY := size.Height - childSize.Height - r.padding.Bottom + yAdjustment
	return ui.Offset{Dx: offsetX, Dy: offsetY}
}

func (r *renderExpandedTitleBox) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	if !r.hasChild() {
		return foundation.None[float64]()
	}
	child := r.Child()
	childConstraints := constraints.WidthConstraints().Deflate(r.padding)
	b, ok := child.GetDryBaseline(childConstraints, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	return foundation.Some(b + r.childOffsetFromSize(child.GetDryLayout(childConstraints), r.GetDryLayout(constraints)).Dy)
}

func (r *renderExpandedTitleBox) PerformLayout() {
	constraints := r.BoxConstraints()
	if !r.hasChild() {
		r.SetSize(constraints.Smallest())
		return
	}
	child := r.Child()
	r.SetSize(constraints.Biggest())
	child.Layout(constraints.WidthConstraints().Deflate(r.padding), rendering.LayoutOpts{ParentUsesSize: true})
	rendering.BoxParentDataOf(child).Offset = r.childOffsetFromSize(child.Size(), r.Size())
}
