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
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// maxTitleTextScaleFactor is the biggest text scale factor of the title.
//
// Flutter: material/app_bar.dart _kMaxTitleTextScaleFactor
const maxTitleTextScaleFactor = 1.34

// appBarLeadingWidth is the width of the leading widget, so that the leading
// button is square.
//
// Flutter: material/app_bar.dart _kLeadingWidth
const appBarLeadingWidth = ToolbarHeight

// AppBarOpts are the optional parameters of NewAppBar.
//
// Flutter: material/app_bar.dart AppBar.new
type AppBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Leading is the widget at the start of the toolbar, before the title.
	Leading widgets.Widget
	// NoAutomaticallyImplyLeading says that the app bar does not try to make a
	// leading widget (a drawer button or a back button) when Leading is nil
	// (Flutter's automaticallyImplyLeading: false).
	NoAutomaticallyImplyLeading bool
	// Title is the primary widget displayed in the app bar, usually a Text.
	Title widgets.Widget
	// Actions are the widgets displayed after the title, usually icon buttons.
	Actions []widgets.Widget
	// NoAutomaticallyImplyActions says that the app bar does not make an end
	// drawer button (Flutter's automaticallyImplyActions: false).
	NoAutomaticallyImplyActions bool
	// FlexibleSpace is stacked behind the toolbar and the bottom widget; its
	// height is the same as the app bar's.
	FlexibleSpace widgets.Widget
	// Bottom is the widget at the bottom of the app bar, usually a TabBar.
	Bottom widgets.PreferredSizeWidget
	// Elevation is the z-coordinate of the app bar; it must not be negative. The
	// default is 0 (Material 3) or 4 (Material 2).
	Elevation foundation.Opt[float64]
	// ScrolledUnderElevation is the elevation when content is scrolled under
	// the app bar.
	ScrolledUnderElevation foundation.Opt[float64]
	// NotificationPredicate decides which scroll notifications make the app bar
	// scrolled under; the default is widgets.DefaultScrollNotificationPredicate.
	NotificationPredicate widgets.ScrollNotificationPredicate
	// ShadowColor is the color of the shadow below the app bar.
	ShadowColor foundation.Opt[ui.Color]
	// SurfaceTintColor is the color of the surface tint overlay.
	SurfaceTintColor foundation.Opt[ui.Color]
	// Shape is the shape of the app bar's Material.
	Shape painting.ShapeBorder
	// BackgroundColor is the fill color; the default is the surface color of the
	// color scheme.
	BackgroundColor foundation.Opt[ui.Color]
	// ForegroundColor is the default color of the title and the icons; the
	// default is the on-surface color of the color scheme.
	ForegroundColor foundation.Opt[ui.Color]
	// IconTheme is the color, opacity and size of the icons of the app bar.
	IconTheme foundation.Opt[widgets.IconThemeData]
	// ActionsIconTheme is the color, opacity and size of the icons of the
	// actions.
	ActionsIconTheme foundation.Opt[widgets.IconThemeData]
	// NoPrimary says that the app bar is not at the top of the screen, so it does
	// not pad itself with the top padding of the media query (Flutter's
	// primary: false).
	NoPrimary bool
	// CenterTitle says whether the title is centered; the default is false on
	// every platform but iOS and macOS, where it is true for fewer than two
	// actions.
	CenterTitle foundation.Opt[bool]
	// ExcludeHeaderSemantics says that the title does not get the header
	// semantics.
	ExcludeHeaderSemantics bool
	// TitleSpacing is the spacing around the title; the default is
	// widgets.NavigationToolbarMiddleSpacing.
	TitleSpacing foundation.Opt[float64]
	// ToolbarOpacity is how opaque the toolbar part of the app bar is; the
	// default is 1.
	ToolbarOpacity foundation.Opt[float64]
	// BottomOpacity is how opaque the bottom part of the app bar is; the
	// default is 1.
	BottomOpacity foundation.Opt[float64]
	// ToolbarHeight is the height of the toolbar; the default is ToolbarHeight
	// (56).
	ToolbarHeight foundation.Opt[float64]
	// LeadingWidth is the width of the leading widget; the default is 56.
	LeadingWidth foundation.Opt[float64]
	// ToolbarTextStyle is the default text style of the toolbar.
	ToolbarTextStyle *painting.TextStyle
	// TitleTextStyle is the default text style of the title.
	TitleTextStyle *painting.TextStyle
	// SystemOverlayStyle is the style of the system overlays (the status bar).
	SystemOverlayStyle foundation.Opt[services.SystemUiOverlayStyle]
	// ForceMaterialTransparency makes the Material of the app bar transparent.
	ForceMaterialTransparency bool
	// NoUseDefaultSemanticsOrder says that the semantics order of the flexible
	// space and the toolbar is not set (Flutter's useDefaultSemanticsOrder:
	// false).
	NoUseDefaultSemanticsOrder bool
	// ClipBehavior is how the toolbar is clipped; the default is ClipHardEdge.
	ClipBehavior ui.Clip
	// ActionsPadding is the padding around the actions.
	ActionsPadding painting.EdgeInsetsGeometry
	// AnimateColor says that the color of the Material animates when it changes.
	AnimateColor bool
}

// AppBar is a Material Design app bar, consisting of a toolbar and potentially
// other widgets, such as a TabBar and a FlexibleSpaceBar. It is the Scaffold's
// top row.
//
// Flutter: material/app_bar.dart AppBar
type AppBar struct {
	widgets.StatefulWidget
	o AppBarOpts
}

// NewAppBar creates an app bar. In debug builds it panics with the tag
// AppBarElevationNegative when the elevation is negative.
//
// Flutter: material/app_bar.dart AppBar.new
func NewAppBar(opts ...AppBarOpts) *AppBar {
	o := firstOpts(opts)
	if foundation.DebugMode && o.Elevation.OrElse(0) < 0 {
		panic(errors.New("AppBarElevationNegative", "the elevation of an AppBar must not be negative", "elevation", o.Elevation.Value()))
	}
	if o.NotificationPredicate == nil {
		o.NotificationPredicate = widgets.DefaultScrollNotificationPredicate
	}
	return &AppBar{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o: o}
}

// Leading is the widget at the start of the toolbar.
//
// Flutter: material/app_bar.dart AppBar.leading
func (w *AppBar) Leading() widgets.Widget { return w.o.Leading }

// AutomaticallyImplyLeading is whether the app bar tries to make a leading
// widget.
//
// Flutter: material/app_bar.dart AppBar.automaticallyImplyLeading
func (w *AppBar) AutomaticallyImplyLeading() bool { return !w.o.NoAutomaticallyImplyLeading }

// Title is the primary widget displayed in the app bar.
//
// Flutter: material/app_bar.dart AppBar.title
func (w *AppBar) Title() widgets.Widget { return w.o.Title }

// Actions are the widgets displayed after the title.
//
// Flutter: material/app_bar.dart AppBar.actions
func (w *AppBar) Actions() []widgets.Widget { return w.o.Actions }

// FlexibleSpace is stacked behind the toolbar and the bottom widget.
//
// Flutter: material/app_bar.dart AppBar.flexibleSpace
func (w *AppBar) FlexibleSpace() widgets.Widget { return w.o.FlexibleSpace }

// Bottom is the widget at the bottom of the app bar.
//
// Flutter: material/app_bar.dart AppBar.bottom
func (w *AppBar) Bottom() widgets.PreferredSizeWidget { return w.o.Bottom }

// Elevation is the elevation, when it was set.
//
// Flutter: material/app_bar.dart AppBar.elevation
func (w *AppBar) Elevation() foundation.Opt[float64] { return w.o.Elevation }

// Primary is whether the app bar is at the top of the screen.
//
// Flutter: material/app_bar.dart AppBar.primary
func (w *AppBar) Primary() bool { return !w.o.NoPrimary }

// ToolbarHeight is the height of the toolbar, when it was set.
//
// Flutter: material/app_bar.dart AppBar.toolbarHeight
func (w *AppBar) ToolbarHeight() foundation.Opt[float64] { return w.o.ToolbarHeight }

// PreferredSize is the size of the app bar: infinitely wide and as high as the
// toolbar (ToolbarHeight unless set) plus the bottom widget. The theme's toolbar
// height is not part of it; see AppBarPreferredHeightFor.
//
// Flutter: material/app_bar.dart AppBar.preferredSize
func (w *AppBar) PreferredSize() ui.Size {
	bottom := 0.0
	if w.o.Bottom != nil {
		bottom = w.o.Bottom.PreferredSize().Height
	}
	return ui.Size{Width: math.Inf(1), Height: w.o.ToolbarHeight.OrElse(ToolbarHeight) + bottom}
}

// AppBarPreferredHeightFor returns the height that a widget with the preferred
// size takes. It is the same as PreferredSize().Height, unless the widget is an
// AppBar whose ToolbarHeight is nil and AppBarThemeOf(bc).ToolbarHeight is set;
// in that case the toolbar height is the theme's.
//
// Flutter: material/app_bar.dart AppBar.preferredHeightFor
func AppBarPreferredHeightFor(bc widgets.BuildContext, w widgets.PreferredSizeWidget) float64 {
	if bar, ok := w.(*AppBar); ok && bar.o.ToolbarHeight.IsNone() {
		bottom := 0.0
		if bar.o.Bottom != nil {
			bottom = bar.o.Bottom.PreferredSize().Height
		}
		return AppBarThemeOf(bc).ToolbarHeight.OrElse(ToolbarHeight) + bottom
	}
	return w.PreferredSize().Height
}

func (w *AppBar) effectiveCenterTitle(theme *ThemeData, appBarTheme AppBarThemeData) bool {
	if c, ok := w.o.CenterTitle.Get(); ok {
		return c
	}
	if c, ok := appBarTheme.CenterTitle.Get(); ok {
		return c
	}
	switch theme.Platform {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		return len(w.o.Actions) < 2
	}
	return false
}

// CreateState creates the state.
//
// Flutter: material/app_bar.dart AppBar.createState
func (w *AppBar) CreateState() widgets.State { return &appBarState{} }

type appBarState struct {
	widgets.StateOf[*AppBar]
	observer      *widgets.ScrollNotificationObserverState
	handle        widgets.ScrollNotificationListenerHandle
	scrolledUnder bool
}

// DidChangeDependencies listens to the scroll notifications, unless a drawer
// of the scaffold is open.
//
// Flutter: material/app_bar.dart _AppBarState.didChangeDependencies
func (s *appBarState) DidChangeDependencies() {
	if s.observer != nil {
		s.observer.RemoveListener(s.handle)
		s.observer = nil
	}
	if scaffold := ScaffoldMaybeOf(s.Context()); scaffold != nil && (scaffold.IsDrawerOpen() || scaffold.IsEndDrawerOpen()) {
		return
	}
	s.observer = widgets.ScrollNotificationObserverMaybeOf(s.Context())
	if s.observer != nil {
		s.handle = s.observer.AddListener(s.handleScrollNotification)
	}
}

// Dispose stops listening.
//
// Flutter: material/app_bar.dart _AppBarState.dispose
func (s *appBarState) Dispose() {
	if s.observer != nil {
		s.observer.RemoveListener(s.handle)
		s.observer = nil
	}
}

func (s *appBarState) handleScrollNotification(notification widgets.ScrollNotification) {
	update, ok := notification.(*widgets.ScrollUpdateNotification)
	if !ok || !s.Widget().o.NotificationPredicate(notification) {
		return
	}
	old := s.scrolledUnder
	metrics := update.Metrics()
	switch metrics.AxisDirection() {
	case painting.AxisDirectionUp:
		// Scroll view is reversed.
		s.scrolledUnder = metrics.ExtentAfter() > 0
	case painting.AxisDirectionDown:
		s.scrolledUnder = metrics.ExtentBefore() > 0
	}
	// For horizontal scroll views the app bar is not scrolled under: the flag
	// stays as it is.
	if s.scrolledUnder != old {
		s.SetState(func() {
			// The state that changed is in scrolledUnder.
		})
	}
}

// systemOverlayStyleForBrightness is the overlay style for an app bar of the
// brightness.
//
// Flutter: material/app_bar.dart _AppBarState._systemOverlayStyleForBrightness
func systemOverlayStyleForBrightness(brightness ui.Brightness, backgroundColor foundation.Opt[ui.Color]) services.SystemUiOverlayStyle {
	style := services.SystemUiOverlayStyleDark
	if brightness == ui.BrightnessDark {
		style = services.SystemUiOverlayStyleLight
	}
	return services.SystemUiOverlayStyle{
		StatusBarColor:                  backgroundColor,
		StatusBarBrightness:             style.StatusBarBrightness,
		StatusBarIconBrightness:         style.StatusBarIconBrightness,
		SystemStatusBarContrastEnforced: style.SystemStatusBarContrastEnforced,
	}
}

// appBarDefaults returns the defaults of the app bar of the theme: the
// generated Material 3 ones or the hand coded Material 2 ones.
func appBarDefaults(theme *ThemeData) AppBarThemeData {
	if theme.UseMaterial3 {
		return appBarDefaultsM3(theme)
	}
	return appBarDefaultsM2(theme)
}

// appBarDefaultsM3 are the Material 3 defaults of the app bar (generated from
// the Material token database in Flutter).
//
// Flutter: material/app_bar.dart _AppBarDefaultsM3
func appBarDefaultsM3(theme *ThemeData) AppBarThemeData {
	colors := theme.ColorScheme
	return AppBarThemeData{
		Elevation:              foundation.Some(0.0),
		ScrolledUnderElevation: foundation.Some(3.0),
		TitleSpacing:           foundation.Some(widgets.NavigationToolbarMiddleSpacing),
		ToolbarHeight:          foundation.Some(64.0),
		BackgroundColor:        foundation.Some(colors.Surface()),
		ForegroundColor:        foundation.Some(colors.OnSurface()),
		ShadowColor:            foundation.Some(Colors.Transparent),
		SurfaceTintColor:       foundation.Some(Colors.Transparent),
		IconTheme:              foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(colors.OnSurface()), Size: foundation.Some(24.0)})),
		ActionsIconTheme:       foundation.Some(widgets.NewIconThemeData(widgets.IconThemeDataOpts{Color: foundation.Some(colors.OnSurfaceVariant()), Size: foundation.Some(24.0)})),
		ToolbarTextStyle:       theme.TextTheme.BodyMedium,
		TitleTextStyle:         theme.TextTheme.TitleLarge,
		// TODO(Craftplacer): Consider using EdgeInsets.only(right: 8.0) instead of
		// EdgeInsets.zero for Material 3 in the future,
		// https://github.com/flutter/flutter/issues/155747
		ActionsPadding: painting.EdgeInsetsZero,
	}
}

// appBarDefaultsM2 are the hand coded defaults of the app bar for Material 2.
//
// Flutter: material/app_bar.dart _AppBarDefaultsM2
func appBarDefaultsM2(theme *ThemeData) AppBarThemeData {
	colors := theme.ColorScheme
	background, foreground := colors.Primary(), colors.OnPrimary()
	if colors.Brightness() == ui.BrightnessDark {
		background, foreground = colors.Surface(), colors.OnSurface()
	}
	return AppBarThemeData{
		Elevation:        foundation.Some(4.0),
		ShadowColor:      foundation.Some(ui.NewColor(0xFF000000)),
		TitleSpacing:     foundation.Some(widgets.NavigationToolbarMiddleSpacing),
		ToolbarHeight:    foundation.Some(ToolbarHeight),
		BackgroundColor:  foundation.Some(background),
		ForegroundColor:  foundation.Some(foreground),
		IconTheme:        foundation.Some(theme.IconTheme),
		ToolbarTextStyle: theme.TextTheme.BodyMedium,
		TitleTextStyle:   theme.TextTheme.TitleLarge,
		ActionsPadding:   painting.EdgeInsetsZero,
	}
}

// resolveAppBarColor is the first color that is set of the widget's, the
// theme's and the default.
func firstColor(colors ...foundation.Opt[ui.Color]) ui.Color {
	for _, c := range colors {
		if v, ok := c.Get(); ok {
			return v
		}
	}
	return ui.Color{}
}

// Build builds the app bar.
//
// Flutter: material/app_bar.dart _AppBarState.build
func (s *appBarState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	if !o.NoPrimary {
		widgets.DebugCheckHasMediaQuery(bc)
	}
	theme := ThemeOf(bc)
	appBarTheme := AppBarThemeOf(bc)
	defaults := appBarDefaults(theme)
	iconButtonTheme := IconButtonThemeOf(bc)
	scaffold := ScaffoldMaybeOf(bc)
	parentRoute := widgets.ModalRouteOf(bc)
	hasDrawer := scaffold != nil && scaffold.HasDrawer()
	hasEndDrawer := scaffold != nil && scaffold.HasEndDrawer()
	useCloseButton := parentRoute != nil && parentRoute.FullscreenDialog()
	settings, hasSettings := widgets.DependOn[*FlexibleSpaceBarSettings](bc)
	scrolledUnder := s.scrolledUnder
	if hasSettings && settings.isScrolledUnder.IsSome() {
		scrolledUnder = settings.isScrolledUnder.Value()
	}

	toolbarHeight := o.ToolbarHeight.OrElse(appBarTheme.ToolbarHeight.OrElse(ToolbarHeight))

	backgroundColor := firstColor(o.BackgroundColor, appBarTheme.BackgroundColor, defaults.BackgroundColor)
	scrolledUnderBackground := firstColor(o.BackgroundColor, appBarTheme.BackgroundColor, foundation.Some(theme.ColorScheme.SurfaceContainer()))
	effectiveBackgroundColor := backgroundColor
	if scrolledUnder {
		effectiveBackgroundColor = scrolledUnderBackground
	}

	foregroundColor := firstColor(o.ForegroundColor, appBarTheme.ForegroundColor, defaults.ForegroundColor)

	elevation := o.Elevation.OrElse(appBarTheme.Elevation.OrElse(defaults.Elevation.Value()))
	effectiveElevation := elevation
	if scrolledUnder {
		effectiveElevation = o.ScrolledUnderElevation.OrElse(appBarTheme.ScrolledUnderElevation.OrElse(
			defaults.ScrolledUnderElevation.OrElse(elevation)))
	}

	overallIconTheme := pickIconTheme(o.IconTheme, appBarTheme.IconTheme, foundation.Some(defaults.IconTheme.Value().CopyWith(widgets.IconThemeDataOpts{Color: foundation.Some(foregroundColor)})))
	actionForegroundColor := o.ForegroundColor
	if actionForegroundColor.IsNone() {
		actionForegroundColor = appBarTheme.ForegroundColor
	}
	var actionsIconTheme widgets.IconThemeData
	switch {
	case o.ActionsIconTheme.IsSome():
		actionsIconTheme = o.ActionsIconTheme.Value()
	case appBarTheme.ActionsIconTheme.IsSome():
		actionsIconTheme = appBarTheme.ActionsIconTheme.Value()
	case o.IconTheme.IsSome():
		actionsIconTheme = o.IconTheme.Value()
	case appBarTheme.IconTheme.IsSome():
		actionsIconTheme = appBarTheme.IconTheme.Value()
	case defaults.ActionsIconTheme.IsSome():
		actionsIconTheme = defaults.ActionsIconTheme.Value().CopyWith(widgets.IconThemeDataOpts{Color: actionForegroundColor})
	default:
		actionsIconTheme = overallIconTheme
	}

	actionsPadding := o.ActionsPadding
	if actionsPadding == nil {
		actionsPadding = appBarTheme.ActionsPadding
	}
	if actionsPadding == nil {
		actionsPadding = defaults.ActionsPadding
	}

	toolbarTextStyle := o.ToolbarTextStyle
	if toolbarTextStyle == nil {
		toolbarTextStyle = appBarTheme.ToolbarTextStyle
	}
	if toolbarTextStyle == nil && defaults.ToolbarTextStyle != nil {
		toolbarTextStyle = defaults.ToolbarTextStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(foregroundColor)})
	}
	titleTextStyle := o.TitleTextStyle
	if titleTextStyle == nil {
		titleTextStyle = appBarTheme.TitleTextStyle
	}
	if titleTextStyle == nil && defaults.TitleTextStyle != nil {
		titleTextStyle = defaults.TitleTextStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(foregroundColor)})
	}

	toolbarOpacity := o.ToolbarOpacity.OrElse(1.0)
	if toolbarOpacity != 1.0 {
		opacity := animation.NewInterval(0.25, 1.0, animation.Curves.FastOutSlowIn).Transform(toolbarOpacity)
		if titleTextStyle != nil && titleTextStyle.Color.IsSome() {
			titleTextStyle = titleTextStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(titleTextStyle.Color.Value().WithOpacity(opacity))})
		}
		if toolbarTextStyle != nil && toolbarTextStyle.Color.IsSome() {
			toolbarTextStyle = toolbarTextStyle.CopyWith(painting.TextStyleCopy{Color: foundation.Some(toolbarTextStyle.Color.Value().WithOpacity(opacity))})
		}
		overallIconTheme = overallIconTheme.CopyWith(widgets.IconThemeDataOpts{Opacity: foundation.Some(opacity * overallIconTheme.Opacity().OrElse(1.0))})
		actionsIconTheme = actionsIconTheme.CopyWith(widgets.IconThemeDataOpts{Opacity: foundation.Some(opacity * actionsIconTheme.Opacity().OrElse(1.0))})
	}

	leading := o.Leading
	if leading == nil && !o.NoAutomaticallyImplyLeading {
		switch {
		case hasDrawer:
			leading = NewDrawerButton(ActionButtonOpts{Style: IconButtonStyleFrom(IconButtonStyleFromOpts{IconSize: foundation.Some(overallIconTheme.Size().OrElse(24))})})
		case parentRoute != nil && parentRoute.ImpliesAppBarDismissal():
			if useCloseButton {
				leading = NewCloseButton()
			} else {
				leading = NewBackButton()
			}
		}
	}
	if leading != nil {
		if theme.UseMaterial3 {
			var effectiveIconButtonTheme IconButtonThemeData
			if defaults.IconTheme.IsSome() && overallIconTheme.Equal(defaults.IconTheme.Value()) {
				effectiveIconButtonTheme = iconButtonTheme
			} else {
				leadingIconButtonStyle := IconButtonStyleFrom(IconButtonStyleFromOpts{ForegroundColor: overallIconTheme.Color(), IconSize: overallIconTheme.Size()})
				effectiveIconButtonTheme = IconButtonThemeData{Style: copyIconButtonThemeStyle(iconButtonTheme.Style, leadingIconButtonStyle)}
			}
			if isIconButtonWidget(leading) {
				leading = widgets.NewCenter(leading)
			}
			leading = NewIconButtonTheme(effectiveIconButtonTheme, leading)
		}
		leading = widgets.NewConstrainedBox(
			rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(o.LeadingWidth.OrElse(appBarTheme.LeadingWidth.OrElse(appBarLeadingWidth)))}),
			leading)
	}

	title := o.Title
	if title != nil {
		title = newAppBarTitleBox(title)
		if !o.ExcludeHeaderSemantics {
			title = widgets.NewSemantics(title, widgets.SemanticsOpts{NamesRoute: namesRoute(), Header: foundation.Some(true)})
		}
		style := titleTextStyle
		title = widgets.NewDefaultTextStyle(style, title, widgets.DefaultTextStyleOpts{
			SoftWrap: foundation.Some(false),
			Overflow: painting.TextOverflowEllipsis,
		})
		// Set maximum text scale factor to maxTitleTextScaleFactor for the title to
		// keep the visual hierarchy the same even with larger font sizes. To opt out,
		// wrap the title widget in a MediaQuery widget with a different TextScaler.
		title = widgets.NewMediaQueryWithClampedTextScaling(title, widgets.MediaQueryClampedTextScalingOpts{MaxScaleFactor: maxTitleTextScaleFactor})
	}

	var actions widgets.Widget
	if len(o.Actions) > 0 {
		cross := rendering.CrossAxisAlignmentStretch
		if theme.UseMaterial3 {
			cross = rendering.CrossAxisAlignmentCenter
		}
		actions = widgets.NewPadding(actionsPadding, widgets.NewRow(o.Actions, widgets.RowOpts{
			MainAxisSize:       rendering.MainAxisSizeMin,
			CrossAxisAlignment: cross,
		}))
	}
	if actions == nil && hasEndDrawer && !o.NoAutomaticallyImplyActions {
		actions = NewEndDrawerButton(ActionButtonOpts{Style: IconButtonStyleFrom(IconButtonStyleFromOpts{IconSize: foundation.Some(overallIconTheme.Size().OrElse(24))})})
	}
	if actions != nil {
		var effectiveActionsIconButtonTheme IconButtonThemeData
		if defaults.ActionsIconTheme.IsSome() && actionsIconTheme.Equal(defaults.ActionsIconTheme.Value()) {
			effectiveActionsIconButtonTheme = iconButtonTheme
		} else {
			actionsIconButtonStyle := IconButtonStyleFrom(IconButtonStyleFromOpts{ForegroundColor: actionsIconTheme.Color(), IconSize: actionsIconTheme.Size()})
			effectiveActionsIconButtonTheme = IconButtonThemeData{Style: copyIconButtonThemeStyle(iconButtonTheme.Style, actionsIconButtonStyle)}
		}
		actions = NewIconButtonTheme(effectiveActionsIconButtonTheme, widgets.NewIconThemeMerge(actionsIconTheme, actions))
	}

	toolbar := widgets.NewNavigationToolbar(widgets.NavigationToolbarOpts{
		Leading:        leading,
		Middle:         title,
		Trailing:       actions,
		NoCenterMiddle: !w.effectiveCenterTitle(theme, appBarTheme),
		MiddleSpacing:  foundation.Some(o.TitleSpacing.OrElse(appBarTheme.TitleSpacing.OrElse(widgets.NavigationToolbarMiddleSpacing))),
	})

	clip := o.ClipBehavior
	if clip == ui.ClipUnspecified {
		clip = ui.ClipHardEdge
	}
	// If the toolbar is allocated less than toolbarHeight make it appear to
	// scroll upwards within its shrinking container.
	var appBar widgets.Widget = widgets.NewClipRect(
		widgets.NewCustomSingleChildLayout(
			&toolbarContainerLayout{SingleChildLayoutDelegateBase: rendering.NewSingleChildLayoutDelegateBase(nil), toolbarHeight: toolbarHeight},
			widgets.NewIconThemeMerge(overallIconTheme, widgets.NewDefaultTextStyle(toolbarTextStyle, toolbar)),
		),
		widgets.ClipRectOpts{ClipBehavior: clip},
	)
	if o.Bottom != nil {
		var bottom widgets.Widget = o.Bottom
		if bo := o.BottomOpacity.OrElse(1.0); bo != 1.0 {
			bottom = widgets.NewOpacity(animation.NewInterval(0.25, 1.0, animation.Curves.FastOutSlowIn).Transform(bo), o.Bottom)
		}
		appBar = widgets.NewColumn([]widgets.Widget{
			widgets.NewFlexible(widgets.NewConstrainedBox(rendering.BoxConstraints{MaxWidth: math.Inf(1), MaxHeight: toolbarHeight}, appBar)),
			bottom,
		}, widgets.ColumnOpts{MainAxisAlignment: rendering.MainAxisAlignmentSpaceBetween})
	}

	// The padding applies to the toolbar and the tab bar, not to the flexible
	// space.
	if !o.NoPrimary {
		appBar = widgets.NewSafeArea(appBar, widgets.SafeAreaOpts{Bottom: foundation.Some(false)})
	}

	appBar = widgets.NewAlign(appBar, widgets.AlignOpts{Alignment: painting.AlignmentTopCenter})

	if o.FlexibleSpace != nil {
		var flexKey, toolbarKey semantics.SemanticsSortKey
		if !o.NoUseDefaultSemanticsOrder {
			flexKey, toolbarKey = semantics.NewOrdinalSortKey(1.0), semantics.NewOrdinalSortKey(0.0)
		}
		appBar = widgets.NewStack([]widgets.Widget{
			widgets.NewSemantics(o.FlexibleSpace, widgets.SemanticsOpts{SortKey: flexKey, ExplicitChildNodes: true}),
			// Creates a material widget to prevent the flexibleSpace from
			// obscuring the ink splashes produced by appBar children.
			widgets.NewSemantics(NewMaterial(appBar, MaterialOpts{Type: MaterialTypeTransparency}), widgets.SemanticsOpts{SortKey: toolbarKey, ExplicitChildNodes: true}),
		}, widgets.StackOpts{Fit: rendering.StackFitPassthrough})
	}

	overlayStyle, ok := o.SystemOverlayStyle.Get()
	if !ok {
		overlayStyle, ok = appBarTheme.SystemOverlayStyle.Get()
	}
	if !ok {
		var statusColor foundation.Opt[ui.Color]
		if theme.UseMaterial3 {
			// Make the status bar transparent for M3 so the elevation overlay color is
			// picked up by the status bar.
			statusColor = foundation.Some(ui.NewColor(0x00000000))
		}
		overlayStyle = systemOverlayStyleForBrightness(EstimateBrightnessForColor(effectiveBackgroundColor), statusColor)
	}

	materialColor := backgroundColor
	if theme.UseMaterial3 {
		materialColor = effectiveBackgroundColor
	}
	materialType := MaterialTypeCanvas
	if o.ForceMaterialTransparency {
		materialType = MaterialTypeTransparency
	}
	surfaceTint := o.SurfaceTintColor
	if surfaceTint.IsNone() {
		surfaceTint = appBarTheme.SurfaceTintColor
	}
	if surfaceTint.IsNone() && theme.UseMaterial3 {
		surfaceTint = foundation.Some(theme.ColorScheme.SurfaceTint())
	}
	shape := o.Shape
	if shape == nil {
		shape = appBarTheme.Shape
	}
	if shape == nil {
		shape = defaults.Shape
	}
	shadow := o.ShadowColor
	if shadow.IsNone() {
		shadow = appBarTheme.ShadowColor
	}
	if shadow.IsNone() {
		shadow = defaults.ShadowColor
	}

	return widgets.NewSemantics(
		widgets.NewAnnotatedRegion(overlayStyle,
			NewMaterial(
				widgets.NewSemantics(appBar, widgets.SemanticsOpts{ExplicitChildNodes: true}),
				MaterialOpts{
					Color:            foundation.Some(materialColor),
					Elevation:        effectiveElevation,
					Type:             materialType,
					ShadowColor:      shadow,
					SurfaceTintColor: surfaceTint,
					Shape:            shape,
					AnimateColor:     o.AnimateColor,
				},
			)),
		widgets.SemanticsOpts{Container: true},
	)
}

// pickIconTheme is the first icon theme that is set.
func pickIconTheme(themes ...foundation.Opt[widgets.IconThemeData]) widgets.IconThemeData {
	for _, t := range themes {
		if v, ok := t.Get(); ok {
			return v
		}
	}
	return widgets.IconThemeData{}
}

// namesRoute is the namesRoute semantics of the title: true on the platforms
// other than iOS and macOS.
func namesRoute() foundation.Opt[bool] {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
		return foundation.None[bool]()
	}
	return foundation.Some(true)
}

// toolbarContainerLayout lays the toolbar out at the bottom of the space of
// the app bar ("Bottom justify the toolbarHeight child which may overflow the
// top").
//
// Flutter: material/app_bar.dart _ToolbarContainerLayout
type toolbarContainerLayout struct {
	rendering.SingleChildLayoutDelegateBase
	toolbarHeight float64
}

func (d *toolbarContainerLayout) GetConstraintsForChild(constraints rendering.BoxConstraints) rendering.BoxConstraints {
	return constraints.Tighten(rendering.TightenOpts{Height: foundation.Some(d.toolbarHeight)})
}

func (d *toolbarContainerLayout) GetSize(constraints rendering.BoxConstraints) ui.Size {
	return ui.Size{Width: constraints.MaxWidth, Height: d.toolbarHeight}
}

func (d *toolbarContainerLayout) GetPositionForChild(size, childSize ui.Size) ui.Offset {
	return ui.Offset{Dx: 0, Dy: size.Height - childSize.Height}
}

func (d *toolbarContainerLayout) ShouldRelayout(old rendering.SingleChildLayoutDelegate) bool {
	return d.toolbarHeight != old.(*toolbarContainerLayout).toolbarHeight
}

// appBarTitleBox lays out the title of an AppBar with unconstrained height,
// vertically centers it within its (NavigationToolbar) parent, and allows the
// parent to constrain the title's actual height.
//
// Flutter: material/app_bar.dart _AppBarTitleBox
type appBarTitleBox struct {
	widgets.SingleChildRenderObjectWidget
}

func newAppBarTitleBox(child widgets.Widget) *appBarTitleBox {
	return &appBarTitleBox{widgets.NewSingleChildRenderObjectWidgetBase(child)}
}

func (w *appBarTitleBox) CreateRenderObject(bc widgets.BuildContext) rendering.RenderObject {
	return newRenderAppBarTitleBox(widgets.DirectionalityOf(bc))
}

func (w *appBarTitleBox) UpdateRenderObject(bc widgets.BuildContext, renderObject rendering.RenderObject) {
	renderObject.(*renderAppBarTitleBox).SetTextDirection(widgets.DirectionalityOf(bc))
}

// renderAppBarTitleBox is the render object of appBarTitleBox.
//
// Flutter: material/app_bar.dart _RenderAppBarTitleBox
type renderAppBarTitleBox struct {
	rendering.RenderAligningShiftedBox
}

func newRenderAppBarTitleBox(textDirection ui.TextDirection) *renderAppBarTitleBox {
	r := &renderAppBarTitleBox{}
	rendering.InitRenderAligningShiftedBox(&r.RenderAligningShiftedBox, r, painting.AlignmentCenter, textDirection, nil)
	return r
}

func (r *renderAppBarTitleBox) ComputeDryLayout(constraints rendering.BoxConstraints) ui.Size {
	inner := constraints.CopyWith(rendering.BoxConstraintsCopy{MaxHeight: foundation.Some(math.Inf(1))})
	childSize := r.Child().GetDryLayout(inner)
	return constraints.Constrain(childSize)
}

func (r *renderAppBarTitleBox) ComputeDryBaseline(constraints rendering.BoxConstraints, baseline ui.TextBaseline) foundation.Opt[float64] {
	child := r.Child()
	if child == nil {
		return foundation.None[float64]()
	}
	inner := constraints.CopyWith(rendering.BoxConstraintsCopy{MaxHeight: foundation.Some(math.Inf(1))})
	result, ok := child.GetDryBaseline(inner, baseline).Get()
	if !ok {
		return foundation.None[float64]()
	}
	childSize := child.GetDryLayout(inner)
	mine := r.GetDryLayout(constraints)
	diff := ui.Offset{Dx: mine.Width - childSize.Width, Dy: mine.Height - childSize.Height}
	return foundation.Some(result + r.ResolvedAlignment().AlongOffset(diff).Dy)
}

func (r *renderAppBarTitleBox) PerformLayout() {
	constraints := r.BoxConstraints()
	inner := constraints.CopyWith(rendering.BoxConstraintsCopy{MaxHeight: foundation.Some(math.Inf(1))})
	r.Child().Layout(inner, rendering.LayoutOpts{ParentUsesSize: true})
	r.SetSize(constraints.Constrain(r.Child().Size()))
	r.AlignChild()
}

// copyIconButtonThemeStyle is the style of the theme with the foreground color,
// overlay color and icon size of the app bar: nil when the theme has no style.
func copyIconButtonThemeStyle(themeStyle, appBarStyle *ButtonStyle) *ButtonStyle {
	if themeStyle == nil {
		return nil
	}
	return themeStyle.CopyWith(ButtonStyleOpts{ForegroundColor: appBarStyle.ForegroundColor, OverlayColor: appBarStyle.OverlayColor, IconSize: appBarStyle.IconSize})
}

// isIconButtonWidget says whether the widget is an IconButton, or one of the
// buttons that Flutter derives from it (the back, close and drawer buttons).
func isIconButtonWidget(w widgets.Widget) bool {
	switch w.(type) {
	case *IconButton, *BackButton, *CloseButton, *DrawerButton, *EndDrawerButton:
		return true
	}
	return false
}
