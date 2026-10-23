// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"fmt"
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

type tabBarState struct {
	widgets.StateOf[*TabBar]
	internalScrollController *TabBarScrollController
	controller               *TabController
	indicatorPainter         *indicatorPainter
	currentIndex             int
	tabStripWidth            float64
	tabKeys                  []*widgets.GlobalKey[widgets.State]
	labelPaddings            []painting.EdgeInsetsGeometry
	hasScheduledTabsCheck    bool
	animationHandle          foundation.ListenerHandle
	controllerHandle         foundation.ListenerHandle
}

func (s *tabBarState) InitState() {
	w := s.Widget()
	// If indicatorSize is TabIndicatorSize.label, tabKeys[i] is used to find the
	// width of tab widget i. See indicatorPainter.indicatorRect().
	s.tabKeys = make([]*widgets.GlobalKey[widgets.State], len(w.tabs))
	s.labelPaddings = make([]painting.EdgeInsetsGeometry, len(w.tabs))
	for i := range w.tabs {
		s.tabKeys[i] = widgets.NewGlobalKey[widgets.State]()
		s.labelPaddings[i] = painting.EdgeInsetsZero
	}
}

func (s *tabBarState) defaults() TabBarThemeData {
	w := s.Widget()
	theme := ThemeOf(s.Context())
	if theme.UseMaterial3 {
		if w.isPrimary {
			return tabsPrimaryDefaultsM3(theme, w.o.IsScrollable)
		}
		return tabsSecondaryDefaultsM3(theme, w.o.IsScrollable)
	}
	return tabsDefaultsM2(theme, w.o.IsScrollable)
}

func (s *tabBarState) effectiveScrollController() *TabBarScrollController {
	if c := s.Widget().o.ScrollController; c != nil {
		if s.internalScrollController != nil {
			s.internalScrollController.Dispose()
			s.internalScrollController = nil
		}
		return c
	}
	if s.internalScrollController == nil {
		s.internalScrollController = NewTabBarScrollController()
	}
	return s.internalScrollController
}

func (s *tabBarState) getIndicator(indicatorSize TabBarIndicatorSize) painting.Decoration {
	w := s.Widget()
	bc := s.Context()
	theme := ThemeOf(bc)
	tabBarTheme := TabBarThemeOf(bc)

	if w.o.Indicator != nil {
		return w.o.Indicator
	}
	if tabBarTheme.Indicator != nil {
		return tabBarTheme.Indicator
	}

	color := firstOpt(w.o.IndicatorColor, tabBarTheme.IndicatorColor, s.defaults().IndicatorColor).Value()
	// ThemeData tries to avoid this by having indicatorColor avoid being the
	// primaryColor. However, it's possible that the tab bar is on a Material that
	// isn't the primaryColor. In that case, if the indicator color ends up matching
	// the material's color, then this overrides it. When that happens, automatic
	// transitions of the theme will likely look ugly as the indicator color
	// suddenly snaps to white at one end, but it's not clear how to avoid that any
	// further.
	//
	// The material's color might be none (if it's a transparency). In that case
	// there's no good way for us to find out what the color is so we don't.
	if !w.o.NoAutomaticIndicatorColorAdjustment {
		if m, ok := MaterialMaybeOf(bc); ok && m.Color() == foundation.Some(color) {
			color = Colors.White
		}
	}

	effectiveIndicatorWeight := w.o.IndicatorWeight.Value()
	if theme.UseMaterial3 {
		if w.isPrimary {
			effectiveIndicatorWeight = math.Max(effectiveIndicatorWeight, tabsPrimaryIndicatorWeight(indicatorSize))
		} else {
			effectiveIndicatorWeight = math.Max(effectiveIndicatorWeight, tabsSecondaryIndicatorWeight)
		}
	}
	// Only Material 3 primary TabBar with label indicatorSize should be rounded.
	primaryWithLabelIndicator := indicatorSize == TabBarIndicatorSizeLabel && w.isPrimary
	effectiveBorderRadius := foundation.None[painting.BorderRadius]()
	if theme.UseMaterial3 && primaryWithLabelIndicator {
		effectiveBorderRadius = foundation.Some(painting.BorderRadius{
			TopLeft: ui.NewRadiusCircular(effectiveIndicatorWeight), TopRight: ui.NewRadiusCircular(effectiveIndicatorWeight)})
	}
	return NewUnderlineTabIndicator(UnderlineTabIndicatorOpts{
		BorderRadius: effectiveBorderRadius,
		BorderSide:   foundation.Some(painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(effectiveIndicatorWeight), Color: foundation.Some(color)})),
	})
}

// controllerIsValid is false when the TabBar is rebuilt with a new tab controller
// and the caller disposed the old one: its animation is nil then and must not be
// accessed.
func (s *tabBarState) controllerIsValid() bool {
	return s.controller != nil && s.controller.Animation() != nil
}

func (s *tabBarState) updateTabController() {
	newController := s.Widget().o.Controller
	if newController == nil {
		newController = DefaultTabControllerMaybeOf(s.Context())
	}
	if foundation.DebugMode && newController == nil {
		panic(errors.New(tagTabBarNoController, "No TabController for TabBar. When creating a TabBar, you must either provide an explicit TabController using the \"controller\" property, or you must ensure that there is a DefaultTabController above the TabBar. In this case, there was neither an explicit controller nor a default controller."))
	}

	if newController == s.controller {
		return
	}

	if s.controllerIsValid() {
		s.controller.Animation().RemoveListener(s.animationHandle)
		s.controller.RemoveListener(s.controllerHandle)
	}
	s.controller = newController
	if s.controller != nil {
		s.animationHandle = s.controller.Animation().AddListener(s.handleTabControllerAnimationTick)
		s.controllerHandle = s.controller.AddListener(s.handleTabControllerTick)
		s.currentIndex = s.controller.Index()
	}
}

func (s *tabBarState) updateScrollController(oldScrollController *TabBarScrollController) {
	w := s.Widget()
	if oldScrollController != w.o.ScrollController && oldScrollController != nil {
		oldScrollController.tabBarState = nil
	}
	if w.o.ScrollController != nil {
		if s.internalScrollController != nil {
			s.internalScrollController.tabBarState = nil
		}
		w.o.ScrollController.tabBarState = s
	} else {
		if s.internalScrollController == nil {
			s.internalScrollController = NewTabBarScrollController()
		}
		s.internalScrollController.tabBarState = s
	}
}

func (s *tabBarState) initIndicatorPainter() {
	w := s.Widget()
	bc := s.Context()
	theme := ThemeOf(bc)
	tabBarTheme := TabBarThemeOf(bc)
	defaults := s.defaults()
	indicatorSize := w.o.IndicatorSize
	if indicatorSize == TabBarIndicatorSizeUnspecified {
		indicatorSize = tabBarTheme.IndicatorSize
	}
	if indicatorSize == TabBarIndicatorSizeUnspecified {
		indicatorSize = defaults.IndicatorSize
	}

	oldPainter := s.indicatorPainter

	defaultTabIndicatorAnimation := TabIndicatorAnimationLinear
	if indicatorSize == TabBarIndicatorSizeLabel {
		defaultTabIndicatorAnimation = TabIndicatorAnimationElastic
	}
	indicatorAnimation := w.o.IndicatorAnimation
	if indicatorAnimation == TabIndicatorAnimationUnspecified {
		indicatorAnimation = tabBarTheme.IndicatorAnimation
	}
	if indicatorAnimation == TabIndicatorAnimationUnspecified {
		indicatorAnimation = defaultTabIndicatorAnimation
	}

	s.indicatorPainter = nil
	if s.controllerIsValid() {
		s.indicatorPainter = newIndicatorPainter(indicatorPainterOpts{
			controller:       s.controller,
			indicator:        s.getIndicator(indicatorSize),
			indicatorSize:    indicatorSize,
			indicatorPadding: w.o.IndicatorPadding,
			tabKeys:          s.tabKeys,
			// Passing old painter so that the constructor can copy some values from it.
			old:                oldPainter,
			labelPaddings:      s.labelPaddings,
			dividerColor:       firstOpt(w.o.DividerColor, tabBarTheme.DividerColor, defaults.DividerColor),
			dividerHeight:      firstOpt(w.o.DividerHeight, tabBarTheme.DividerHeight, defaults.DividerHeight),
			showDivider:        theme.UseMaterial3 && !w.o.IsScrollable,
			devicePixelRatio:   foundation.Some(widgets.MediaQueryDevicePixelRatioOf(bc)),
			indicatorAnimation: indicatorAnimation,
			textDirection:      widgets.DirectionalityOf(bc),
		})
	}

	if oldPainter != nil {
		oldPainter.dispose()
	}
}

func (s *tabBarState) DidChangeDependencies() {
	s.updateScrollController(s.Widget().o.ScrollController)
	s.updateTabController()
	s.initIndicatorPainter()
}

func (s *tabBarState) DidUpdateWidget(old *TabBar) {
	w := s.Widget()
	switch {
	case w.o.Controller != old.o.Controller || w.o.ScrollController != old.o.ScrollController:
		s.updateScrollController(old.o.ScrollController)
		s.updateTabController()
		s.initIndicatorPainter()

		// Adjust scroll position.
		if c := s.effectiveScrollController(); c.HasClients() {
			if position, ok := c.Position().(*tabBarScrollPosition); ok {
				position.markNeedsPixelsCorrection()
			}
		}
	case w.o.IndicatorColor != old.o.IndicatorColor || w.o.IndicatorWeight != old.o.IndicatorWeight ||
		w.o.IndicatorSize != old.o.IndicatorSize || !foundation.ValuesEqual(w.o.IndicatorPadding, old.o.IndicatorPadding) ||
		!foundation.ValuesEqual(w.o.Indicator, old.o.Indicator) || w.o.DividerColor != old.o.DividerColor ||
		w.o.DividerHeight != old.o.DividerHeight || w.o.IndicatorAnimation != old.o.IndicatorAnimation:
		s.initIndicatorPainter()
	}

	if len(w.tabs) > len(s.tabKeys) {
		for i := len(s.tabKeys); i < len(w.tabs); i++ {
			s.tabKeys = append(s.tabKeys, widgets.NewGlobalKey[widgets.State]())
			s.labelPaddings = append(s.labelPaddings, painting.EdgeInsetsZero)
		}
	} else if len(w.tabs) < len(s.tabKeys) {
		s.tabKeys = s.tabKeys[:len(w.tabs)]
		s.labelPaddings = s.labelPaddings[:len(w.tabs)]
	}
}

func (s *tabBarState) Dispose() {
	if s.indicatorPainter != nil {
		s.indicatorPainter.dispose()
	}
	if s.controllerIsValid() {
		s.controller.Animation().RemoveListener(s.animationHandle)
		s.controller.RemoveListener(s.controllerHandle)
	}
	s.controller = nil
	if s.internalScrollController != nil {
		s.internalScrollController.Dispose()
	}
	if c := s.Widget().o.ScrollController; c != nil {
		c.tabBarState = nil
	}
	// We don't own the controller Animation, so it's not disposed here.
}

func (s *tabBarState) maxTabIndex() int { return s.indicatorPainter.maxTabIndex() }

func (s *tabBarState) tabScrollOffset(index int, viewportWidth, minExtent, maxExtent float64) float64 {
	w := s.Widget()
	if !w.o.IsScrollable {
		return 0.0
	}
	tabCenter := s.indicatorPainter.centerOf(index)
	var paddingStart float64
	direction := widgets.DirectionalityOf(s.Context())
	if w.o.Padding != nil {
		if direction == ui.TextDirectionRTL {
			paddingStart = w.o.Padding.Resolve(direction).Right
		} else {
			paddingStart = w.o.Padding.Resolve(direction).Left
		}
	}
	if direction == ui.TextDirectionRTL {
		tabCenter = s.tabStripWidth - tabCenter
	}
	return ui.ClampDouble(tabCenter+paddingStart-viewportWidth/2.0, minExtent, maxExtent)
}

func (s *tabBarState) tabCenteredScrollOffset(index int) float64 {
	position := s.effectiveScrollController().Position()
	return s.tabScrollOffset(index, position.ViewportDimension(), position.MinScrollExtent(), position.MaxScrollExtent())
}

func (s *tabBarState) initialScrollOffset(viewportWidth, minExtent, maxExtent float64) float64 {
	return s.tabScrollOffset(s.currentIndex, viewportWidth, minExtent, maxExtent)
}

func (s *tabBarState) scrollToCurrentIndex() {
	offset := s.tabCenteredScrollOffset(s.currentIndex)
	s.effectiveScrollController().AnimateTo(offset, TabScrollDuration, animation.Curves.Ease)
}

func (s *tabBarState) scrollToControllerValue() {
	leading := foundation.None[float64]()
	if s.currentIndex > 0 {
		leading = foundation.Some(s.tabCenteredScrollOffset(s.currentIndex - 1))
	}
	middlePosition := s.tabCenteredScrollOffset(s.currentIndex)
	trailing := foundation.None[float64]()
	if s.currentIndex < s.maxTabIndex() {
		trailing = foundation.Some(s.tabCenteredScrollOffset(s.currentIndex + 1))
	}

	index := float64(s.controller.Index())
	value := s.controller.Animation().Value()
	var offset float64
	switch d := value - index; {
	case d == -1.0:
		offset = leading.OrElse(middlePosition)
	case d == 1.0:
		offset = trailing.OrElse(middlePosition)
	case d == 0:
		offset = middlePosition
	case d < 0:
		if l, ok := leading.Get(); ok {
			offset = ui.LerpDouble(middlePosition, l, index-value)
		} else {
			offset = middlePosition
		}
	default:
		if t, ok := trailing.Get(); ok {
			offset = ui.LerpDouble(middlePosition, t, value-index)
		} else {
			offset = middlePosition
		}
	}

	s.effectiveScrollController().JumpTo(offset)
}

func (s *tabBarState) handleTabControllerAnimationTick() {
	if !s.controller.IndexIsChanging() && s.Widget().o.IsScrollable {
		// Sync the TabBar's scroll position with the TabBarView's PageView.
		s.currentIndex = s.controller.Index()
		s.scrollToControllerValue()
	}
}

func (s *tabBarState) handleTabControllerTick() {
	if s.controller.Index() != s.currentIndex {
		s.currentIndex = s.controller.Index()
		if s.Widget().o.IsScrollable {
			s.scrollToCurrentIndex()
		}
	}
	// Rebuild the tabs after a (potentially animated) index change has completed.
	s.SetState(func() {})
}

// saveTabOffsets is called each time layout completes.
func (s *tabBarState) saveTabOffsets(tabOffsets []float64, textDirection ui.TextDirection, width float64) {
	s.tabStripWidth = width
	if s.indicatorPainter != nil {
		s.indicatorPainter.saveTabOffsets(tabOffsets, textDirection)
	}
}

func (s *tabBarState) handleTap(index int) {
	s.controller.AnimateTo(index)
	if cb := s.Widget().o.OnTap; cb != nil {
		cb(index)
	}
}

func (s *tabBarState) buildStyledTab(child widgets.Widget, isSelected bool, a animation.Animation[float64], defaults TabBarThemeData) widgets.Widget {
	w := s.Widget()
	return newTabStyle(a, isSelected, w.isPrimary, w.o.LabelColor, w.o.UnselectedLabelColor, w.o.LabelStyle, w.o.UnselectedLabelStyle, defaults, child)
}

func (s *tabBarState) scheduleCheckHasValidTabsCount() {
	if s.hasScheduledTabsCheck {
		return
	}
	widgets.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		s.hasScheduledTabsCheck = false
		if !s.Mounted() {
			return
		}
		if n := len(s.Widget().tabs); s.controller.Length() != n {
			panic(errors.New(tagTabBarTabsCount, fmt.Sprintf("Controller's length property (%d) does not match the number of tabs (%d) present in TabBar's tabs property.", s.controller.Length(), n),
				"controllerLength", s.controller.Length(), "tabs", n))
		}
	})
	s.hasScheduledTabsCheck = true
}

func (s *tabBarState) debugTabAlignmentIsValid(tabAlignment TabAlignment) {
	w := s.Widget()
	if w.o.IsScrollable && tabAlignment == TabAlignmentFill {
		panic(errors.New(tagTabBarAlignmentFillScrollable, tabAlignment.String()+" is only valid for non-scrollable tab bars.", "tabAlignment", tabAlignment.Name()))
	}
	if !w.o.IsScrollable && (tabAlignment == TabAlignmentStart || tabAlignment == TabAlignmentStartOffset) {
		panic(errors.New(tagTabBarAlignmentStartFixed, tabAlignment.String()+" is only valid for scrollable tab bars.", "tabAlignment", tabAlignment.Name()))
	}
}

func (s *tabBarState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	o := w.o
	DebugCheckHasMaterialLocalizations(bc)
	if foundation.DebugMode {
		s.scheduleCheckHasValidTabsCount()
	}
	theme := ThemeOf(bc)
	tabBarTheme := TabBarThemeOf(bc)
	defaults := s.defaults()
	effectiveTabAlignment := o.TabAlignment
	if effectiveTabAlignment == TabAlignmentUnspecified {
		effectiveTabAlignment = tabBarTheme.TabAlignment
	}
	if effectiveTabAlignment == TabAlignmentUnspecified {
		effectiveTabAlignment = defaults.TabAlignment
	}
	if foundation.DebugMode {
		s.debugTabAlignmentIsValid(effectiveTabAlignment)
	}

	localizations := MaterialLocalizationsOf(bc)
	if s.controller.Length() == 0 {
		return widgets.NewLimitedBox(
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(math.Inf(1)), Height: foundation.Some(tabHeight + o.IndicatorWeight.Value())}),
			widgets.LimitedBoxOpts{MaxWidth: foundation.Some(0.0)})
	}

	wrappedTabs := make([]widgets.Widget, len(w.tabs))
	for index, tab := range w.tabs {
		padding := o.LabelPadding
		if padding == nil {
			padding = tabBarTheme.LabelPadding
		}
		if padding == nil {
			padding = TabLabelPadding
		}
		const verticalAdjustment = (tabTextAndIconHeight - tabHeight) / 2.0

		if p, ok := tab.(widgets.PreferredSizeWidget); ok && p.PreferredSize().Height == tabHeight && w.TabHasTextAndIcon() {
			padding = padding.Add(painting.NewEdgeInsetsSymmetric(verticalAdjustment, 0))
		}
		s.labelPaddings[index] = padding

		wrappedTabs[index] = widgets.NewCenter(
			widgets.NewPadding(s.labelPaddings[index], widgets.NewKeyedSubtree(tab, widgets.WidgetOpts{Key: s.tabKeys[index]})),
			widgets.CenterOpts{HeightFactor: foundation.Some(1.0)})
	}

	// If the controller was provided by DefaultTabController and we're part of a
	// Hero (typically the AppBar), then we will not be able to find the controller
	// during a Hero transition.
	if s.controller != nil {
		previousIndex := s.controller.PreviousIndex()

		if s.controller.IndexIsChanging() {
			// The user tapped on a tab, the tab controller's animation is running.
			var a animation.Animation[float64] = &changeAnimation{controller: s.controller}
			wrappedTabs[s.currentIndex] = s.buildStyledTab(wrappedTabs[s.currentIndex], true, a, defaults)
			wrappedTabs[previousIndex] = s.buildStyledTab(wrappedTabs[previousIndex], false, a, defaults)
		} else {
			// The user is dragging the TabBarView's PageView left or right.
			tabIndex := s.currentIndex
			wrappedTabs[tabIndex] = s.buildStyledTab(wrappedTabs[tabIndex], true,
				&dragAnimation{changeAnimation: changeAnimation{controller: s.controller}, index: tabIndex}, defaults)
			if s.currentIndex > 0 {
				tabIndex := s.currentIndex - 1
				wrappedTabs[tabIndex] = s.buildStyledTab(wrappedTabs[tabIndex], false,
					animation.NewReverseAnimation(&dragAnimation{changeAnimation: changeAnimation{controller: s.controller}, index: tabIndex}), defaults)
			}
			if s.currentIndex < len(w.tabs)-1 {
				tabIndex := s.currentIndex + 1
				wrappedTabs[tabIndex] = s.buildStyledTab(wrappedTabs[tabIndex], false,
					animation.NewReverseAnimation(&dragAnimation{changeAnimation: changeAnimation{controller: s.controller}, index: tabIndex}), defaults)
			}
		}
	}

	// Add the tap handler to each tab. If the tab bar is not scrollable, then give
	// all of the tabs equal flexibility so that they each occupy the same share of
	// the tab bar's overall width.
	tabCount := len(w.tabs)
	for index := 0; index < tabCount; index++ {
		index := index
		states := unselectedState
		if index == s.currentIndex {
			states = selectedState
		}

		var effectiveMouseCursor services.MouseCursor
		if o.MouseCursor != nil {
			effectiveMouseCursor = widgets.WidgetStateResolveAs[services.MouseCursor](o.MouseCursor, states)
		}
		if effectiveMouseCursor == nil && tabBarTheme.MouseCursor != nil {
			effectiveMouseCursor = tabBarTheme.MouseCursor.Resolve(states)
		}
		if effectiveMouseCursor == nil {
			effectiveMouseCursor = widgets.WidgetStateMouseCursorClickable.Resolve(states)
		}

		overlayColor := o.OverlayColor
		if overlayColor == nil {
			overlayColor = tabBarTheme.OverlayColor
		}
		if overlayColor == nil {
			overlayColor = widgets.WidgetStatePropertyResolveWith(func(st widgets.WidgetStates) foundation.Opt[ui.Color] {
				effectiveStates := states
				for _, state := range st.Slice() {
					effectiveStates = effectiveStates.With(state)
				}
				if defaults.OverlayColor == nil {
					return foundation.None[ui.Color]()
				}
				return defaults.OverlayColor.Resolve(effectiveStates)
			})
		}
		splashFactory := o.SplashFactory
		if splashFactory == nil {
			splashFactory = tabBarTheme.SplashFactory
		}
		if splashFactory == nil {
			splashFactory = defaults.SplashFactory
		}
		wrappedTabs[index] = NewInkWell(
			widgets.NewPadding(painting.NewEdgeInsetsFromLTRB(0, 0, 0, o.IndicatorWeight.Value()),
				widgets.NewSemantics(
					// This has to be wrapped above the Stack to override any role set by the child.
					widgets.NewStack([]widgets.Widget{
						wrappedTabs[index],
						widgets.NewSemantics(nil, widgets.SemanticsOpts{
							Selected: foundation.Some(index == s.currentIndex),
							Label:    foundation.Some(localizations.TabLabel(index+1, tabCount))}),
					}),
					widgets.SemanticsOpts{Role: foundation.Some(ui.SemanticsRoleTab)})),
			InkWellOpts{
				MouseCursor: effectiveMouseCursor,
				OnTap:       func() { s.handleTap(index) },
				OnHover: func(value bool) {
					if o.OnHover != nil {
						o.OnHover(value, index)
					}
				},
				OnFocusChange: func(value bool) {
					if o.OnFocusChange != nil {
						o.OnFocusChange(value, index)
					}
				},
				EnableFeedback: foundation.Some(o.EnableFeedback.OrElse(true)),
				OverlayColor:   overlayColor,
				SplashFactory:  splashFactory,
				BorderRadius:   firstOpt(o.SplashBorderRadius, tabBarTheme.SplashBorderRadius, defaults.SplashBorderRadius),
			})
		wrappedTabs[index] = widgets.NewMergeSemantics(wrappedTabs[index])
		if !o.IsScrollable && effectiveTabAlignment == TabAlignmentFill {
			wrappedTabs[index] = widgets.NewExpanded(wrappedTabs[index])
		}
	}

	mainAxisSize := rendering.MainAxisSizeMin
	if effectiveTabAlignment == TabAlignmentFill {
		mainAxisSize = rendering.MainAxisSizeMax
	}
	var painter rendering.CustomPainter
	if s.indicatorPainter != nil {
		painter = s.indicatorPainter
	}
	var tabBar widgets.Widget = widgets.NewSemantics(
		widgets.NewCustomPaint(
			newTabStyle(animation.AlwaysDismissed, false, w.isPrimary, o.LabelColor, o.UnselectedLabelColor, o.LabelStyle, o.UnselectedLabelStyle, defaults,
				newTabLabelBar(wrappedTabs, mainAxisSize, s.saveTabOffsets)),
			widgets.CustomPaintOpts{Painter: painter}),
		widgets.SemanticsOpts{Role: foundation.Some(ui.SemanticsRoleTabBar), Container: true, ExplicitChildNodes: true})

	if o.IsScrollable {
		effectivePadding := o.Padding
		if effectiveTabAlignment == TabAlignmentStartOffset {
			var rest painting.EdgeInsetsGeometry = painting.EdgeInsetsZero
			if o.Padding != nil {
				rest = o.Padding
			}
			effectivePadding = painting.NewEdgeInsetsDirectionalFromSTEB(tabStartOffset, 0, 0, 0).Add(rest)
		}

		tabBar = widgets.NewScrollConfiguration(
			// The scrolling tabs should not show an overscroll indicator.
			widgets.ScrollConfigurationOf(bc).CopyWith(widgets.ScrollBehaviorCopy{Overscroll: foundation.Some(false)}),
			widgets.NewSingleChildScrollView(tabBar, widgets.SingleChildScrollViewOpts{
				DragStartBehavior: o.DragStartBehavior, ScrollDirection: painting.AxisHorizontal, Controller: s.effectiveScrollController(),
				Padding: effectivePadding, Physics: o.Physics}))
		if theme.UseMaterial3 {
			var effectiveAlignment painting.AlignmentGeometry = painting.AlignmentDirectionalCenterStart
			if effectiveTabAlignment == TabAlignmentCenter {
				effectiveAlignment = painting.AlignmentCenter
			}

			dividerColor := firstOpt(o.DividerColor, tabBarTheme.DividerColor, defaults.DividerColor).Value()
			dividerHeight := firstOpt(o.DividerHeight, tabBarTheme.DividerHeight, defaults.DividerHeight).Value()

			widthFactor := foundation.Some(1.0)
			if dividerHeight > 0 {
				widthFactor = foundation.None[float64]()
			}
			tabBar = widgets.NewAlign(tabBar, widgets.AlignOpts{HeightFactor: foundation.Some(1.0), WidthFactor: widthFactor, Alignment: effectiveAlignment})

			if dividerColor != Colors.Transparent && dividerHeight > 0 {
				tabBar = widgets.NewCustomPaint(tabBar, widgets.CustomPaintOpts{Painter: newDividerPainter(dividerColor, dividerHeight)})
			}
		}
	} else if o.Padding != nil {
		tabBar = widgets.NewPadding(o.Padding, tabBar)
	}

	data := widgets.MediaQueryOf(bc)
	textScaler := o.TextScaler
	if textScaler == nil {
		textScaler = tabBarTheme.TextScaler
	}
	if textScaler != nil {
		data = data.CopyWith(widgets.MediaQueryDataCopyOpts{TextScaler: foundation.Some(textScaler)})
	}
	return NewMaterial(widgets.NewMediaQuery(data, tabBar), MaterialOpts{Type: MaterialTypeTransparency})
}

var (
	// Flutter: material/tabs.dart _TabsDefaultsM2.iconMargin
	tabsM2IconMargin painting.EdgeInsetsGeometry = painting.NewEdgeInsetsFromLTRB(0, 0, 0, 10)
	// Flutter: material/tabs.dart _TabsPrimaryDefaultsM3.iconMargin
	tabsPrimaryIconMargin painting.EdgeInsetsGeometry = painting.NewEdgeInsetsFromLTRB(0, 0, 0, 2)
)

// tabsDefaultsM2 are the hand coded defaults based on Material Design 2.
//
// Flutter: material/tabs.dart _TabsDefaultsM2
func tabsDefaultsM2(theme *ThemeData, isScrollable bool) TabBarThemeData {
	colors := theme.ColorScheme
	isDark := theme.ColorScheme.Brightness() == ui.BrightnessDark
	primaryColor := Colors.Blue.Primary
	if isDark {
		primaryColor = Colors.Grey.Shade900()
	}
	indicatorColor := colors.Secondary()
	if indicatorColor == primaryColor {
		indicatorColor = Colors.White
	}
	alignment := TabAlignmentFill
	if isScrollable {
		alignment = TabAlignmentStart
	}
	return TabBarThemeData{
		IndicatorSize:        TabBarIndicatorSizeTab,
		IndicatorColor:       foundation.Some(indicatorColor),
		LabelColor:           theme.PrimaryTextTheme.BodyLarge.Color,
		LabelStyle:           theme.PrimaryTextTheme.BodyLarge,
		UnselectedLabelStyle: theme.PrimaryTextTheme.BodyLarge,
		SplashFactory:        theme.SplashFactory,
		TabAlignment:         alignment,
	}
}

func tabsM3Overlay(colors *ColorScheme, selectedPressed, selectedHovered, selectedFocused, pressed, hovered, focused ui.Color) widgets.WidgetStateProperty[foundation.Opt[ui.Color]] {
	_ = colors
	return widgets.WidgetStatePropertyResolveWith(func(states widgets.WidgetStates) foundation.Opt[ui.Color] {
		if states.Has(widgets.WidgetStateSelected) {
			switch {
			case states.Has(widgets.WidgetStatePressed):
				return foundation.Some(selectedPressed.WithOpacity(0.1))
			case states.Has(widgets.WidgetStateHovered):
				return foundation.Some(selectedHovered.WithOpacity(0.08))
			case states.Has(widgets.WidgetStateFocused):
				return foundation.Some(selectedFocused.WithOpacity(0.1))
			}
			return foundation.None[ui.Color]()
		}
		switch {
		case states.Has(widgets.WidgetStatePressed):
			return foundation.Some(pressed.WithOpacity(0.1))
		case states.Has(widgets.WidgetStateHovered):
			return foundation.Some(hovered.WithOpacity(0.08))
		case states.Has(widgets.WidgetStateFocused):
			return foundation.Some(focused.WithOpacity(0.1))
		}
		return foundation.None[ui.Color]()
	})
}

// tabsPrimaryDefaultsM3 are the Material 3 defaults of the primary tab bar.
//
// Flutter: material/tabs.dart _TabsPrimaryDefaultsM3
func tabsPrimaryDefaultsM3(theme *ThemeData, isScrollable bool) TabBarThemeData {
	colors, textTheme := theme.ColorScheme, theme.TextTheme
	alignment := TabAlignmentFill
	if isScrollable {
		alignment = TabAlignmentStartOffset
	}
	return TabBarThemeData{
		IndicatorSize: TabBarIndicatorSizeLabel,
		// These values come from Divider widget defaults. Token db deprecated
		// 'primary-navigation-tab.divider.color' and '.height' tokens.
		DividerColor: foundation.Some(colors.OutlineVariant()), DividerHeight: foundation.Some(1.0),
		IndicatorColor: foundation.Some(colors.Primary()), LabelColor: foundation.Some(colors.Primary()),
		LabelStyle: textTheme.TitleSmall, UnselectedLabelColor: foundation.Some(colors.OnSurfaceVariant()),
		UnselectedLabelStyle: textTheme.TitleSmall,
		OverlayColor:         tabsM3Overlay(colors, colors.Primary(), colors.Primary(), colors.Primary(), colors.Primary(), colors.OnSurface(), colors.OnSurface()),
		SplashFactory:        theme.SplashFactory, TabAlignment: alignment,
	}
}

// tabsPrimaryIndicatorWeight is the weight of the indicator of the primary tab bar.
//
// Flutter: material/tabs.dart _TabsPrimaryDefaultsM3.indicatorWeight
func tabsPrimaryIndicatorWeight(indicatorSize TabBarIndicatorSize) float64 {
	if indicatorSize == TabBarIndicatorSizeLabel {
		return 3.0
	}
	return 2.0
}

// Flutter: material/tabs.dart _TabsSecondaryDefaultsM3.indicatorWeight
const tabsSecondaryIndicatorWeight = 2.0

// tabsSecondaryDefaultsM3 are the Material 3 defaults of the secondary tab bar.
//
// Flutter: material/tabs.dart _TabsSecondaryDefaultsM3
func tabsSecondaryDefaultsM3(theme *ThemeData, isScrollable bool) TabBarThemeData {
	colors, textTheme := theme.ColorScheme, theme.TextTheme
	alignment := TabAlignmentFill
	if isScrollable {
		alignment = TabAlignmentStartOffset
	}
	return TabBarThemeData{
		IndicatorSize:  TabBarIndicatorSizeTab,
		DividerColor:   foundation.Some(colors.OutlineVariant()),
		DividerHeight:  foundation.Some(1.0),
		IndicatorColor: foundation.Some(colors.Primary()), LabelColor: foundation.Some(colors.OnSurface()),
		LabelStyle: textTheme.TitleSmall, UnselectedLabelColor: foundation.Some(colors.OnSurfaceVariant()),
		UnselectedLabelStyle: textTheme.TitleSmall,
		OverlayColor:         tabsM3Overlay(colors, colors.OnSurface(), colors.OnSurface(), colors.OnSurface(), colors.OnSurface(), colors.OnSurface(), colors.OnSurface()),
		SplashFactory:        theme.SplashFactory, TabAlignment: alignment,
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTabBarAlignmentFillScrollable = "TabBarAlignmentFillScrollable"
	tagTabBarAlignmentStartFixed     = "TabBarAlignmentStartFixed"
	tagTabBarNoController            = "TabBarNoController"
	tagTabBarTabsCount               = "TabBarTabsCount"
)
