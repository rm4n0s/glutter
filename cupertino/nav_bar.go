package cupertino

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

// NavigationBarBottomMode is how the bottom of a navigation bar (a search field, or the widget given as
// Bottom) behaves when the page scrolls.
//
// Flutter: cupertino/nav_bar.dart NavigationBarBottomMode
type NavigationBarBottomMode int

const (
	// NavigationBarBottomModeUnspecified is the zero value: the mode that the bar chooses (automatic).
	NavigationBarBottomModeUnspecified NavigationBarBottomMode = iota
	// NavigationBarBottomModeAutomatic makes the bottom collapse as the page scrolls, before the large title.
	//
	// Flutter: cupertino/nav_bar.dart NavigationBarBottomMode.automatic
	NavigationBarBottomModeAutomatic
	// NavigationBarBottomModeAlways keeps the bottom visible when the page scrolls.
	//
	// Flutter: cupertino/nav_bar.dart NavigationBarBottomMode.always
	NavigationBarBottomModeAlways
)

// CupertinoNavigationBarOpts are the optional parameters of NewCupertinoNavigationBar and of
// NewCupertinoNavigationBarLarge.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar.new
type CupertinoNavigationBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// LargeTitle is the large title of a bar that NewCupertinoNavigationBarLarge makes; nil takes the title
	// of the route. A bar made by NewCupertinoNavigationBar has none.
	LargeTitle widgets.Widget
	// Leading is the widget at the start of the bar. Without it, the bar shows a back button when the route
	// can be popped (and a Cancel button in a full screen dialog).
	Leading widgets.Widget
	// NotAutomaticallyImplyLeading says that the bar does not make a leading widget from the route (Flutter's
	// automaticallyImplyLeading is true by default).
	NotAutomaticallyImplyLeading bool
	// NotAutomaticallyImplyMiddle says that the bar does not take its middle from the title of the route
	// (Flutter's automaticallyImplyMiddle, and automaticallyImplyTitle for the large bar, are true by default).
	NotAutomaticallyImplyMiddle bool
	// PreviousPageTitle is the label of the back button; the default is the title of the previous route.
	PreviousPageTitle foundation.Opt[string]
	// Middle is the widget in the middle of the bar, which a bar made by NewCupertinoNavigationBarLarge does not
	// have.
	Middle widgets.Widget
	// Trailing is the widget at the end of the bar.
	Trailing widgets.Widget
	// Border is the border at the bottom of the bar; the default is a hairline.
	// Deviation: the colors of a border that is given are used as they are (a painting.BorderSide has a
	// plain color).
	Border foundation.Opt[painting.Border]
	// NoBorder says that the bar has no border (Flutter's border: null).
	NoBorder bool
	// BackgroundColor is the color of the bar; the default is the bar background color of the theme.
	BackgroundColor ui.ColorValue
	// NotAutomaticBackgroundVisibility says that the background shows always: by default, in a page scaffold,
	// it shows when the content scrolls under the bar (Flutter's automaticBackgroundVisibility is true).
	NotAutomaticBackgroundVisibility bool
	// NotEnableBackgroundFilterBlur says that the bar does not blur what is behind it (Flutter's
	// enableBackgroundFilterBlur is true).
	NotEnableBackgroundFilterBlur bool
	// Brightness is the brightness of the status bar icons; the default follows the luminance of the background.
	Brightness ui.Brightness
	// Padding is the padding of the content; the default is 16 at the edges.
	Padding *painting.EdgeInsetsDirectional
	// NotTransitionBetweenRoutes says that the bar does not fly between routes (Flutter's
	// transitionBetweenRoutes is true).
	NotTransitionBetweenRoutes bool
	// HeroTag is the tag of the hero of the flight between routes; nil is the default tag of the navigator.
	HeroTag any
	// Bottom is the widget under the bar.
	Bottom widgets.PreferredSizeWidget
}

// CupertinoNavigationBar is an iOS styled navigation bar, to put in the navigationBar of a
// CupertinoPageScaffold: a leading widget or back button, a middle (the title) and a trailing widget, with a
// translucent background that blurs what is under it, and a flight to the next bar when the route changes.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar
type CupertinoNavigationBar struct {
	widgets.StatefulWidget
	o CupertinoNavigationBarOpts
}

// NewCupertinoNavigationBar creates a bar with a middle. In debug builds it panics with the tag
// NavigationBarHeroTagWithoutTransition when a hero tag is given to a bar that does not transition.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar.new
func NewCupertinoNavigationBar(opts ...CupertinoNavigationBarOpts) *CupertinoNavigationBar {
	o := firstOf(opts)
	o.LargeTitle = nil
	if foundation.DebugMode && !o.NotTransitionBetweenRoutes && o.HeroTag != nil {
		panic(errors.New(tagNavBarHeroTagWithoutTransition, "Cannot specify a heroTag override if this navigation bar does not transition due to transitionBetweenRoutes = false."))
	}
	return &CupertinoNavigationBar{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o}
}

// NewCupertinoNavigationBarLarge creates a bar with a large title (the title of the route when none is
// given), which does not scroll; use CupertinoSliverNavigationBar for one that does. In debug builds it
// panics with the tag NavigationBarHeroTagWithoutTransition like NewCupertinoNavigationBar.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar.large
func NewCupertinoNavigationBarLarge(opts ...CupertinoNavigationBarOpts) *CupertinoNavigationBar {
	o := firstOf(opts)
	o.Middle = nil
	if foundation.DebugMode && !o.NotTransitionBetweenRoutes && o.HeroTag != nil {
		panic(errors.New(tagNavBarHeroTagWithoutTransition, "Cannot specify a heroTag override if this navigation bar does not transition due to transitionBetweenRoutes = false."))
	}
	return &CupertinoNavigationBar{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o}
}

// large is whether the bar is the large one (it is known by its large title being set, as in Flutter, or by
// the large constructor: a large bar whose title comes from the route has none given).
func (w *CupertinoNavigationBar) isLarge() bool { return w.o.Middle == nil && w.o.LargeTitle != nil }

// LargeTitle is the large title, or nil.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar.largeTitle
func (w *CupertinoNavigationBar) LargeTitle() widgets.Widget { return w.o.LargeTitle }

// Leading is the leading widget, or nil.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar.leading
func (w *CupertinoNavigationBar) Leading() widgets.Widget { return w.o.Leading }

// Middle is the middle widget, or nil.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar.middle
func (w *CupertinoNavigationBar) Middle() widgets.Widget { return w.o.Middle }

// Trailing is the trailing widget, or nil.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar.trailing
func (w *CupertinoNavigationBar) Trailing() widgets.Widget { return w.o.Trailing }

// BackgroundColor is the color of the bar, or nil for the theme's.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar.backgroundColor
func (w *CupertinoNavigationBar) BackgroundColor() ui.ColorValue { return w.o.BackgroundColor }

// ShouldFullyObstruct is whether the background of the bar is opaque: the content then starts under the bar.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar.shouldFullyObstruct
func (w *CupertinoNavigationBar) ShouldFullyObstruct(bc widgets.BuildContext) bool {
	backgroundColor := CupertinoThemeOf(bc).BarBackgroundColor().EffectiveColor()
	if c := CupertinoDynamicColorMaybeResolve(w.o.BackgroundColor, bc); c.IsSome() {
		backgroundColor = c.Value()
	}
	return backgroundColor.Alpha() == 0xFF
}

// PreferredSize is the height of the bar: the persistent height, and the large title and the bottom when it
// has them.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar.preferredSize
func (w *CupertinoNavigationBar) PreferredSize() ui.Size {
	bottomHeight := 0.0
	if w.o.Bottom != nil {
		bottomHeight = w.o.Bottom.PreferredSize().Height
	}
	effectiveLargeHeight := 0.0
	if w.o.LargeTitle != nil {
		effectiveLargeHeight = navBarLargeTitleHeightExtension
	}
	return ui.Size{Width: infinity, Height: navBarPersistentHeight + bottomHeight + effectiveLargeHeight}
}

// CreateState creates the state.
//
// Flutter: cupertino/nav_bar.dart CupertinoNavigationBar.createState
func (w *CupertinoNavigationBar) CreateState() widgets.State { return &cupertinoNavigationBarState{} }

type cupertinoNavigationBarState struct {
	widgets.StateOf[*CupertinoNavigationBar]
	keys                       *navBarComponentsKeys
	scrollNotificationObserver *widgets.ScrollNotificationObserverState
	scrollObserverHandle       widgets.ScrollNotificationListenerHandle
	scrollAnimationValue       float64
}

func (s *cupertinoNavigationBarState) InitState() { s.keys = newNavBarComponentsKeys() }

func (s *cupertinoNavigationBarState) DidChangeDependencies() {
	s.removeScrollListener()
	s.scrollNotificationObserver = widgets.ScrollNotificationObserverMaybeOf(s.Context())
	if s.scrollNotificationObserver != nil {
		s.scrollObserverHandle = s.scrollNotificationObserver.AddListener(s.handleScrollNotification)
	}
}

func (s *cupertinoNavigationBarState) removeScrollListener() {
	if s.scrollNotificationObserver != nil {
		s.scrollNotificationObserver.RemoveListener(s.scrollObserverHandle)
		s.scrollNotificationObserver = nil
	}
}

func (s *cupertinoNavigationBarState) Dispose() { s.removeScrollListener() }

func (s *cupertinoNavigationBarState) handleScrollNotification(notification widgets.ScrollNotification) {
	if _, ok := notification.(*widgets.ScrollUpdateNotification); !ok || notification.Depth() != 0 {
		return
	}
	metrics := notification.Metrics()
	oldScrollAnimationValue := s.scrollAnimationValue
	scrollExtent := 0.0
	switch metrics.AxisDirection() {
	case painting.AxisDirectionUp:
		scrollExtent = metrics.ExtentAfter()
	case painting.AxisDirectionDown:
		scrollExtent = metrics.ExtentBefore()
	}
	switch {
	case scrollExtent >= 0 && scrollExtent < navBarScrollUnderAnimationExtent:
		s.SetState(func() { s.scrollAnimationValue = clampDouble(scrollExtent/navBarScrollUnderAnimationExtent, 0, 1) })
	case scrollExtent > navBarScrollUnderAnimationExtent && oldScrollAnimationValue != 1.0:
		s.SetState(func() { s.scrollAnimationValue = 1.0 })
	case scrollExtent <= 0 && oldScrollAnimationValue != 0.0:
		s.SetState(func() { s.scrollAnimationValue = 0.0 })
	}
}

// barBackground is the background color of a bar that has the options: the one given resolved, or the one of
// the theme.
func barBackground(bc widgets.BuildContext, given ui.ColorValue) ui.Color {
	if c := CupertinoDynamicColorMaybeResolve(given, bc); c.IsSome() {
		return c.Value()
	}
	return CupertinoThemeOf(bc).BarBackgroundColor().EffectiveColor()
}

// effectiveBackgroundAndBorder are the color and the border of a bar that scrolled by t of its animation: when
// the bar is in a page scaffold with an automatic background it starts as the color of the page, with no border.
func effectiveBackgroundAndBorder(bc widgets.BuildContext, automaticBackgroundVisibility bool, border foundation.Opt[painting.Border],
	backgroundColor ui.Color, t float64) (ui.Color, foundation.Opt[painting.Border]) {
	parentBg := CupertinoPageScaffoldBackgroundColorMaybeOf(bc)
	automatic := automaticBackgroundVisibility && parentBg.IsSome()
	var effectiveBorder foundation.Opt[painting.Border]
	if b, ok := border.Get(); ok {
		initial := b
		if automatic {
			initial = navBarTransparentBorder()
		}
		effectiveBorder = foundation.Some(*painting.BorderLerp(&initial, &b, t))
	}
	effectiveBackground := backgroundColor
	if automatic {
		p := parentBg.Value()
		if lerped := ui.ColorLerp(&p, &backgroundColor, t); lerped != nil {
			effectiveBackground = *lerped
		}
	}
	return effectiveBackground, effectiveBorder
}

func (s *cupertinoNavigationBarState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	backgroundColor := barBackground(bc, w.o.BackgroundColor)
	border := effectiveNavBarBorder(w.o.Border, w.o.NoBorder)
	effectiveBackground, effectiveBorder := effectiveBackgroundAndBorder(bc, !w.o.NotAutomaticBackgroundVisibility, border, backgroundColor, s.scrollAnimationValue)

	bottomHeight := 0.0
	if w.o.Bottom != nil {
		bottomHeight = w.o.Bottom.PreferredSize().Height
	}
	persistentHeight := navBarPersistentHeight + bottomHeight + widgets.MediaQueryPaddingOf(bc).Top
	largeHeight := persistentHeight + navBarLargeTitleHeightExtension

	large := w.o.LargeTitle != nil
	components := newNavBarComponents(bc, navBarComponentsOpts{
		keys: s.keys, route: widgets.ModalRouteOf(bc),
		userLeading: w.o.Leading, automaticallyImplyLeading: !w.o.NotAutomaticallyImplyLeading,
		automaticallyImplyTitle: !w.o.NotAutomaticallyImplyMiddle, previousPageTitle: w.o.PreviousPageTitle,
		userMiddle: w.o.Middle, userTrailing: w.o.Trailing, padding: w.o.Padding,
		userLargeTitle: w.o.LargeTitle, userBottom: bottomWidget(w.o.Bottom),
		large: large, staticBar: true, // This one does not scroll
	})

	var middleVisible *bool
	if w.o.LargeTitle == nil {
		t := true
		middleVisible = &t
	}
	var navBar widgets.Widget = newPersistentNavigationBar(components, w.o.Padding, middleVisible)

	textTheme := CupertinoThemeOf(bc).TextTheme()
	if large {
		children := []widgets.Widget{
			navBar,
			widgets.NewExpanded(widgets.NewPadding(painting.EdgeInsetsDirectional{Start: navBarEdgePadding, Bottom: navBarBottomPadding},
				widgets.NewSemantics(widgets.NewDefaultTextStyle(textTheme.NavLargeTitleTextStyle(),
					newLargeTitle(keyedWidget(components.largeTitle), navBarLargeTitleHeightExtension),
					widgets.DefaultTextStyleOpts{MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis}),
					widgets.SemanticsOpts{Header: foundation.Some(true)}))),
		}
		if w.o.Bottom != nil {
			children = append(children, widgets.NewSizedBox(components.navBarBottom, widgets.SizedBoxOpts{Height: foundation.Some(bottomHeight)}))
		}
		navBar = widgets.NewConstrainedBox(rendering.BoxConstraints{MaxWidth: infinity, MaxHeight: largeHeight}, widgets.NewColumn(children))
	} else {
		children := []widgets.Widget{navBar}
		if w.o.Bottom != nil {
			children = append(children, widgets.NewSizedBox(components.navBarBottom, widgets.SizedBoxOpts{Height: foundation.Some(bottomHeight)}))
		}
		navBar = widgets.NewConstrainedBox(rendering.BoxConstraints{MaxWidth: infinity, MaxHeight: persistentHeight}, widgets.NewColumn(children))
	}

	navBar = wrapWithBackground(effectiveBorder, effectiveBackground, w.o.Brightness,
		widgets.NewDefaultTextStyle(textTheme.TextStyle(), navBar), true, !w.o.NotEnableBackgroundFilterBlur)

	if w.o.NotTransitionBetweenRoutes || !isTransitionable(bc) {
		return navBar
	}
	return widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		var tag any = w.o.HeroTag
		if tag == nil {
			tag = navBarHeroTag{widgets.NavigatorOf(bc)}
		}
		theme := CupertinoThemeOf(bc).TextTheme()
		return widgets.NewHero(tag, newTransitionableNavigationBar(transitionableNavigationBarOpts{
			componentsKeys: s.keys, backgroundColor: foundation.Some(effectiveBackground),
			backButtonTextStyle: theme.NavActionTextStyle(), titleTextStyle: theme.NavTitleTextStyle(),
			largeTitleTextStyle: theme.NavLargeTitleTextStyle(), border: effectiveBorder,
			hasUserMiddle: w.o.Middle != nil, largeExpanded: w.o.LargeTitle != nil, searchable: false,
			automaticBackgroundVisibility: !w.o.NotAutomaticBackgroundVisibility, child: navBar,
		}), widgets.HeroOpts{CreateRectTween: linearTranslateWithLargestRectSizeTween, PlaceholderBuilder: navBarHeroLaunchPadBuilder,
			FlightShuttleBuilder: navBarHeroFlightShuttleBuilder, TransitionOnUserGestures: true})
	})
}

// bottomWidget is the widget of a bottom, or nil (a nil interface and not a typed nil).
func bottomWidget(b widgets.PreferredSizeWidget) widgets.Widget {
	if b == nil {
		return nil
	}
	return b
}

// CupertinoSliverNavigationBarOpts are the optional parameters of NewCupertinoSliverNavigationBar and of
// NewCupertinoSliverNavigationBarSearch.
//
// Flutter: cupertino/nav_bar.dart CupertinoSliverNavigationBar.new
type CupertinoSliverNavigationBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// LargeTitle is the large title; nil takes the title of the route.
	LargeTitle widgets.Widget
	// Leading is the widget at the start of the bar.
	Leading widgets.Widget
	// NotAutomaticallyImplyLeading says that the bar does not make a leading widget from the route.
	NotAutomaticallyImplyLeading bool
	// NotAutomaticallyImplyTitle says that the bar does not take its title from the route; it then needs a
	// LargeTitle.
	NotAutomaticallyImplyTitle bool
	// NotAlwaysShowMiddle says that the middle shows only when the large title is collapsed (Flutter's
	// alwaysShowMiddle is true).
	NotAlwaysShowMiddle bool
	// PreviousPageTitle is the label of the back button.
	PreviousPageTitle foundation.Opt[string]
	// Middle is the widget in the middle of the bar; in a landscape view it is the large title by default.
	Middle widgets.Widget
	// Trailing is the widget at the end of the bar.
	Trailing widgets.Widget
	// Border is the border at the bottom of the bar; the default is a hairline.
	Border foundation.Opt[painting.Border]
	// NoBorder says that the bar has no border.
	NoBorder bool
	// BackgroundColor is the color of the bar; the default is the bar background color of the theme.
	BackgroundColor ui.ColorValue
	// NotAutomaticBackgroundVisibility says that the background shows always.
	NotAutomaticBackgroundVisibility bool
	// NotEnableBackgroundFilterBlur says that the bar does not blur what is behind it.
	NotEnableBackgroundFilterBlur bool
	// Brightness is the brightness of the status bar icons.
	Brightness ui.Brightness
	// Padding is the padding of the content.
	Padding *painting.EdgeInsetsDirectional
	// NotTransitionBetweenRoutes says that the bar does not fly between routes.
	NotTransitionBetweenRoutes bool
	// HeroTag is the tag of the hero of the flight; nil is the default tag of the navigator.
	HeroTag any
	// Stretch says that the bar stretches in an overscroll.
	Stretch bool
	// Bottom is a widget under the large title (not for the search bar, which has the search field).
	Bottom widgets.PreferredSizeWidget
	// BottomMode is how the bottom behaves when the page scrolls; the default is automatic.
	BottomMode NavigationBarBottomMode
	// OnSearchableBottomTap is called with whether the search is going to be active, when the search
	// field of the search bar is tapped (or its Cancel button).
	OnSearchableBottomTap func(active bool)
}

// CupertinoSliverNavigationBar is an iOS styled navigation bar with a large title that collapses into the
// middle as the page scrolls, for a CustomScrollView; its search form makes a search field under the title that
// takes over the bar when it is tapped.
//
// Flutter: cupertino/nav_bar.dart CupertinoSliverNavigationBar
type CupertinoSliverNavigationBar struct {
	widgets.StatefulWidget
	o           CupertinoSliverNavigationBarOpts
	searchField widgets.Widget
	searchable  bool
}

// NewCupertinoSliverNavigationBar creates a sliver bar. In debug builds it panics with the tag
// NavigationBarLargeTitleRequired when it has no large title and does not take it from the route, and with
// NavigationBarBottomModeWithoutBottom when it has a bottom mode and no bottom.
//
// Flutter: cupertino/nav_bar.dart CupertinoSliverNavigationBar.new
func NewCupertinoSliverNavigationBar(opts ...CupertinoSliverNavigationBarOpts) *CupertinoSliverNavigationBar {
	o := firstOf(opts)
	if foundation.DebugMode {
		if o.NotAutomaticallyImplyTitle && o.LargeTitle == nil {
			panic(errors.New(tagNavBarLargeTitleRequired, "No largeTitle has been provided but automaticallyImplyTitle is also false. Either provide a largeTitle or set automaticallyImplyTitle to true."))
		}
		if o.BottomMode != NavigationBarBottomModeUnspecified && o.Bottom == nil {
			panic(errors.New(tagNavBarBottomModeWithoutBottom, "A bottomMode was provided without a corresponding bottom."))
		}
	}
	o.OnSearchableBottomTap = nil
	return &CupertinoSliverNavigationBar{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o, nil, false}
}

// NewCupertinoSliverNavigationBarSearch creates a sliver bar with the search field under the large title.
// The field is the widget that is given (typically a CupertinoSearchTextField). In debug builds it panics
// with the tag NavigationBarLargeTitleRequired when it has no large title and does not take it from the route.
//
// Flutter: cupertino/nav_bar.dart CupertinoSliverNavigationBar.search
func NewCupertinoSliverNavigationBarSearch(searchField widgets.Widget, opts ...CupertinoSliverNavigationBarOpts) *CupertinoSliverNavigationBar {
	o := firstOf(opts)
	if foundation.DebugMode && o.NotAutomaticallyImplyTitle && o.LargeTitle == nil {
		panic(errors.New(tagNavBarLargeTitleRequired, "No largeTitle has been provided but automaticallyImplyTitle is also false. Either provide a largeTitle or set automaticallyImplyTitle to true."))
	}
	o.Bottom = nil
	if o.BottomMode == NavigationBarBottomModeUnspecified {
		o.BottomMode = NavigationBarBottomModeAutomatic
	}
	return &CupertinoSliverNavigationBar{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), o, searchField, true}
}

// Opaque is whether the background color that was given is opaque.
//
// Flutter: cupertino/nav_bar.dart CupertinoSliverNavigationBar.opaque
func (w *CupertinoSliverNavigationBar) Opaque() bool {
	return w.o.BackgroundColor != nil && w.o.BackgroundColor.EffectiveColor().Alpha() == 0xFF
}

// LargeTitle is the large title, or nil.
//
// Flutter: cupertino/nav_bar.dart CupertinoSliverNavigationBar.largeTitle
func (w *CupertinoSliverNavigationBar) LargeTitle() widgets.Widget { return w.o.LargeTitle }

// CreateState creates the state.
//
// Flutter: cupertino/nav_bar.dart CupertinoSliverNavigationBar.createState
func (w *CupertinoSliverNavigationBar) CreateState() widgets.State {
	return &cupertinoSliverNavigationBarState{}
}

type cupertinoSliverNavigationBarState struct {
	widgets.StateOf[*CupertinoSliverNavigationBar]
	widgets.TickerProvider
	keys                      *navBarComponentsKeys
	scrollableState           *widgets.ScrollableState
	scrollingHandle           foundation.ListenerHandle
	effectiveMiddle           widgets.Widget
	animationController       *animation.AnimationController
	searchAnimation           *animation.CurvedAnimation
	persistentHeightAnimation animation.Animation[float64]
	largeTitleHeightAnimation animation.Animation[float64]
	scaledSearchFieldHeight   float64
	scaledLargeTitleHeight    float64
	searchIsActive            bool
	isPortrait                bool
	statusHandle              animation.StatusListenerHandle
}

func (s *cupertinoSliverNavigationBarState) InitState() {
	s.keys = newNavBarComponentsKeys()
	s.isPortrait = true
	s.animationController = animation.NewAnimationController(s, animation.AnimationControllerOpts{Duration: foundation.Some(navBarSearchDuration)})
	s.searchAnimation = animation.NewCurvedAnimation(s.animationController, navBarSearchCurve)
}

func (s *cupertinoSliverNavigationBarState) DidUpdateWidget(old *CupertinoSliverNavigationBar) {
	if s.Widget().o.Middle != old.o.Middle {
		s.updateEffectiveMiddle()
	}
}

func (s *cupertinoSliverNavigationBarState) DidChangeDependencies() {
	s.isPortrait = widgets.MediaQueryOrientationOf(s.Context()) == widgets.OrientationPortrait
	s.updateEffectiveMiddle()
	s.computeScaledHeights()
	s.setupSearchableAnimation()
	s.removeScrollingListener()
	s.scrollableState = widgets.ScrollableMaybeOf(s.Context())
	if s.scrollableState != nil {
		s.scrollingHandle = s.scrollableState.Position().IsScrollingNotifier().AddListener(s.handleScrollChange)
	}
}

func (s *cupertinoSliverNavigationBarState) removeScrollingListener() {
	if s.scrollableState != nil && s.scrollableState.Position() != nil {
		s.scrollableState.Position().IsScrollingNotifier().RemoveListener(s.scrollingHandle)
	}
}

func (s *cupertinoSliverNavigationBarState) Dispose() {
	s.removeScrollingListener()
	s.searchAnimation.Dispose()
	s.animationController.Dispose()
}

func (s *cupertinoSliverNavigationBarState) bottomHeight() float64 {
	w := s.Widget()
	switch {
	case w.searchable:
		return s.scaledSearchFieldHeight + navBarBottomPadding
	case w.o.Bottom != nil:
		return w.o.Bottom.PreferredSize().Height
	}
	return 0.0
}

func (s *cupertinoSliverNavigationBarState) updateEffectiveMiddle() {
	w := s.Widget()
	s.effectiveMiddle = w.o.Middle
	if s.effectiveMiddle == nil && !s.isPortrait {
		s.effectiveMiddle = w.o.LargeTitle
	}
}

func (s *cupertinoSliverNavigationBarState) computeScaledHeights() {
	textScaler := widgets.MediaQueryTextScalerOf(s.Context())
	s.scaledSearchFieldHeight = navBarSearchFieldHeight * dampScaleFactor(textScaler.Scale(navBarSearchFieldHeight), navBarSearchFieldHeight, navBarMaxScaleFactor)
	s.scaledLargeTitleHeight = 0.0
	if s.isPortrait {
		s.scaledLargeTitleHeight = navBarLargeTitleHeightExtension * dampScaleFactor(
			textScaler.Scale(navBarLargeTitleHeightExtension), navBarLargeTitleHeightExtension, navBarLargeTitleScaleDampingRatio)
	}
}

func (s *cupertinoSliverNavigationBarState) setupSearchableAnimation() {
	if s.persistentHeightAnimation != nil {
		s.persistentHeightAnimation.RemoveStatusListener(s.statusHandle)
	}
	s.persistentHeightAnimation = animation.Drive[float64](s.animationController, animation.NewTween(navBarPersistentHeight, 0.0))
	s.statusHandle = s.persistentHeightAnimation.AddStatusListener(s.handleSearchFieldStatusChanged)
	s.largeTitleHeightAnimation = animation.Drive[float64](s.animationController, animation.NewTween(s.scaledLargeTitleHeight, 0.0))
}

// handleScrollChange snaps the bar to a collapsed or an expanded state when the scrolling stops half way.
//
// Flutter: cupertino/nav_bar.dart _CupertinoSliverNavigationBarState._handleScrollChange
func (s *cupertinoSliverNavigationBarState) handleScrollChange() {
	w := s.Widget()
	var position widgets.ScrollPosition
	if s.scrollableState != nil {
		position = s.scrollableState.Position()
	}
	if position == nil || !position.HasPixels() || position.Pixels() <= 0.0 {
		return
	}
	var target foundation.Opt[float64]
	bottomScrollOffset := s.bottomHeight()
	if w.o.BottomMode == NavigationBarBottomModeAlways {
		bottomScrollOffset = 0.0
	}
	canScrollBottom := (w.searchable || w.o.Bottom != nil) && bottomScrollOffset > 0.0
	if canScrollBottom && position.Pixels() < bottomScrollOffset {
		if position.Pixels() > bottomScrollOffset/2 {
			target = foundation.Some(bottomScrollOffset)
		} else {
			target = foundation.Some(0.0)
		}
	} else if position.Pixels() > bottomScrollOffset && position.Pixels() < bottomScrollOffset+s.scaledLargeTitleHeight {
		if position.Pixels() > bottomScrollOffset+(s.scaledLargeTitleHeight/2) {
			target = foundation.Some(bottomScrollOffset + s.scaledLargeTitleHeight)
		} else {
			target = foundation.Some(bottomScrollOffset)
		}
	}
	if t, ok := target.Get(); ok && t <= position.MaxScrollExtent() {
		position.AnimateTo(t, navBarSnapDuration, animation.Curves.FastEaseInToSlowEaseOut)
	}
}

func (s *cupertinoSliverNavigationBarState) handleSearchFieldStatusChanged(status animation.AnimationStatus) {
	s.SetState(func() {
		switch status {
		case animation.AnimationStatusForward:
			s.searchIsActive = true
		case animation.AnimationStatusReverse:
			s.searchIsActive = false
		}
	})
}

func (s *cupertinoSliverNavigationBarState) onSearchFieldTap() {
	if cb := s.Widget().o.OnSearchableBottomTap; cb != nil {
		cb(!s.searchIsActive)
	}
	s.animationController.Toggle()
}

func (s *cupertinoSliverNavigationBarState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var userLeading, userTrailing, userBottom widgets.Widget
	if w.o.Leading != nil {
		userLeading = widgets.NewVisibility(w.o.Leading, widgets.VisibilityOpts{Visible: foundation.Some(!s.searchIsActive)})
	}
	if w.o.Trailing != nil {
		userTrailing = widgets.NewVisibility(w.o.Trailing, widgets.VisibilityOpts{Visible: foundation.Some(!s.searchIsActive)})
	}
	userMiddle := s.effectiveMiddle
	if s.animationController.IsAnimating() {
		userMiddle = widgets.NewText("")
	}
	switch {
	case w.searchable && s.searchIsActive:
		userBottom = newActiveSearchableBottom(s.animationController, s.persistentHeightAnimation, w.searchField, s.scaledSearchFieldHeight, s.onSearchFieldTap)
	case w.searchable:
		userBottom = newInactiveSearchableBottom(s.animationController, s.persistentHeightAnimation, w.searchField, s.scaledSearchFieldHeight, s.onSearchFieldTap)
	default:
		userBottom = bottomWidget(w.o.Bottom)
	}
	if userBottom == nil {
		userBottom = widgets.NewSizedBoxShrink(nil)
	}
	components := newNavBarComponents(bc, navBarComponentsOpts{
		keys: s.keys, route: widgets.ModalRouteOf(bc),
		userLeading: userLeading, automaticallyImplyLeading: !w.o.NotAutomaticallyImplyLeading,
		automaticallyImplyTitle: !w.o.NotAutomaticallyImplyTitle, previousPageTitle: w.o.PreviousPageTitle,
		userMiddle: userMiddle, userTrailing: userTrailing, userLargeTitle: w.o.LargeTitle, userBottom: userBottom,
		padding: w.o.Padding, large: s.isPortrait, staticBar: false, // This one scrolls.
	})

	return widgets.NewMediaQueryWithNoTextScaling(widgets.NewAnimatedBuilder(s.searchAnimation,
		func(bc widgets.BuildContext, _ widgets.Widget) widgets.Widget {
			bottomMode := w.o.BottomMode
			if bottomMode == NavigationBarBottomModeUnspecified {
				bottomMode = NavigationBarBottomModeAutomatic
			}
			if s.searchIsActive {
				bottomMode = NavigationBarBottomModeAlways
			}
			var stretch *rendering.OverScrollHeaderStretchConfiguration
			if w.o.Stretch && !s.searchIsActive {
				stretch = rendering.NewOverScrollHeaderStretchConfiguration()
			}
			return widgets.NewSliverPersistentHeader(&largeTitleNavigationBarSliverDelegate{
				keys: s.keys, components: components, userMiddle: s.effectiveMiddle,
				backgroundColor:               barBackground(bc, w.o.BackgroundColor),
				automaticBackgroundVisibility: !w.o.NotAutomaticBackgroundVisibility,
				brightness:                    w.o.Brightness, border: effectiveNavBarBorder(w.o.Border, w.o.NoBorder), padding: w.o.Padding,
				actionsForegroundColor:  CupertinoThemeOf(bc).PrimaryColor().EffectiveColor(),
				transitionBetweenRoutes: !w.o.NotTransitionBetweenRoutes, heroTag: w.o.HeroTag,
				persistentHeight:     s.persistentHeightAnimation.Value() + widgets.MediaQueryPaddingOf(bc).Top,
				largeTitleHeight:     s.largeTitleHeightAnimation.Value(),
				alwaysShowMiddle:     !w.o.NotAlwaysShowMiddle && s.effectiveMiddle != nil,
				stretchConfiguration: stretch, enableBackgroundFilterBlur: !w.o.NotEnableBackgroundFilterBlur,
				bottomMode: bottomMode, bottomHeight: s.bottomHeight(), controller: s.animationController, searchable: w.searchable,
			}, widgets.SliverPersistentHeaderOpts{Pinned: true}) // iOS navigation bars are always pinned.
		}, nil))
}

// navBarSnapDuration is the duration of the animation that snaps the bar to its collapsed or expanded state.
const navBarSnapDuration = 300 * time.Millisecond

// largeTitleNavigationBarSliverDelegate builds the bar of a CupertinoSliverNavigationBar for the shrink offset
// of its sliver.
//
// Flutter: cupertino/nav_bar.dart _LargeTitleNavigationBarSliverDelegate
type largeTitleNavigationBarSliverDelegate struct {
	widgets.SliverPersistentHeaderDelegateBase
	keys                          *navBarComponentsKeys
	components                    *navBarComponents
	userMiddle                    widgets.Widget
	backgroundColor               ui.Color
	automaticBackgroundVisibility bool
	brightness                    ui.Brightness
	border                        foundation.Opt[painting.Border]
	padding                       *painting.EdgeInsetsDirectional
	actionsForegroundColor        ui.Color
	transitionBetweenRoutes       bool
	heroTag                       any
	persistentHeight              float64
	largeTitleHeight              float64
	alwaysShowMiddle              bool
	enableBackgroundFilterBlur    bool
	bottomMode                    NavigationBarBottomMode
	bottomHeight                  float64
	controller                    *animation.AnimationController
	searchable                    bool
	stretchConfiguration          *rendering.OverScrollHeaderStretchConfiguration
}

func (d *largeTitleNavigationBarSliverDelegate) StretchConfiguration() *rendering.OverScrollHeaderStretchConfiguration {
	return d.stretchConfiguration
}

func (d *largeTitleNavigationBarSliverDelegate) MinExtent() float64 {
	extra := 0.0
	if d.bottomMode == NavigationBarBottomModeAlways {
		extra = d.bottomHeight
	}
	return d.persistentHeight + extra
}

func (d *largeTitleNavigationBarSliverDelegate) MaxExtent() float64 {
	return d.persistentHeight + d.largeTitleHeight + d.bottomHeight
}

func (d *largeTitleNavigationBarSliverDelegate) Build(bc widgets.BuildContext, shrinkOffset float64, overlapsContent bool) widgets.Widget {
	largeTitleThreshold := d.MaxExtent() - d.MinExtent() - navBarShowLargeTitleThreshold
	showLargeTitle := shrinkOffset < largeTitleThreshold

	bottomShrinkFactor := 0.0
	if d.bottomHeight != 0 {
		bottomShrinkFactor = clampDouble(shrinkOffset/d.bottomHeight, 0, 1)
	} else if shrinkOffset > 0 {
		bottomShrinkFactor = 1
	}
	shrinkAnimationValue := clampDouble((shrinkOffset-largeTitleThreshold-navBarScrollUnderAnimationExtent)/navBarScrollUnderAnimationExtent, 0, 1)

	var middleVisible *bool
	if !d.alwaysShowMiddle {
		v := !showLargeTitle
		middleVisible = &v
	}
	persistent := newPersistentNavigationBar(d.components, d.padding, middleVisible)

	effectiveBackground, effectiveBorder := effectiveBackgroundAndBorder(bc, d.automaticBackgroundVisibility, d.border, d.backgroundColor, shrinkAnimationValue)

	textTheme := CupertinoThemeOf(bc).TextTheme()
	largeTitleOpacity := 0.0
	if showLargeTitle && !d.controller.Status().IsForwardOrCompleted() {
		largeTitleOpacity = 1.0
	}
	bottomBottom := 0.0
	if d.bottomMode == NavigationBarBottomModeAutomatic {
		bottomBottom = d.bottomHeight * (1.0 - bottomShrinkFactor)
	}
	stackChildren := []widgets.Widget{
		widgets.NewPositioned(widgets.NewClipRect(widgets.NewPadding(painting.EdgeInsetsDirectional{Start: navBarEdgePadding, Bottom: navBarBottomPadding},
			widgets.NewSafeArea(widgets.NewAnimatedOpacity(largeTitleOpacity, navBarTitleFadeDuration,
				widgets.NewSemantics(widgets.NewDefaultTextStyle(textTheme.NavLargeTitleTextStyle(),
					newLargeTitle(keyedWidget(d.components.largeTitle), d.largeTitleHeight),
					widgets.DefaultTextStyleOpts{MaxLines: foundation.Some(1), Overflow: painting.TextOverflowEllipsis}),
					widgets.SemanticsOpts{Header: foundation.Some(true)})),
				widgets.SafeAreaOpts{Top: foundation.Some(false), Bottom: foundation.Some(false)}))),
			widgets.PositionedOpts{Top: foundation.Some(d.persistentHeight), Left: foundation.Some(0.0), Right: foundation.Some(0.0), Bottom: foundation.Some(bottomBottom)}),
		widgets.NewPositioned(persistent, widgets.PositionedOpts{Left: foundation.Some(0.0), Right: foundation.Some(0.0), Top: foundation.Some(0.0)}),
	}
	if d.bottomMode == NavigationBarBottomModeAutomatic {
		stackChildren = append(stackChildren, widgets.NewPositioned(
			widgets.NewSizedBox(widgets.NewClipRect(d.components.navBarBottom), widgets.SizedBoxOpts{Height: foundation.Some(d.bottomHeight * (1.0 - bottomShrinkFactor))}),
			widgets.PositionedOpts{Left: foundation.Some(0.0), Right: foundation.Some(0.0), Bottom: foundation.Some(0.0)}))
	}
	columnChildren := []widgets.Widget{widgets.NewExpanded(widgets.NewStack(stackChildren))}
	if d.bottomMode == NavigationBarBottomModeAlways {
		columnChildren = append(columnChildren, widgets.NewSizedBox(d.components.navBarBottom, widgets.SizedBoxOpts{Height: foundation.Some(d.bottomHeight)}))
	}
	navBar := wrapWithBackground(effectiveBorder, effectiveBackground, d.brightness,
		widgets.NewDefaultTextStyle(textTheme.TextStyle(), widgets.NewColumn(columnChildren)), true, d.enableBackgroundFilterBlur)

	if !d.transitionBetweenRoutes || !isTransitionable(bc) {
		return navBar
	}
	tag := d.heroTag
	if tag == nil {
		tag = navBarHeroTag{widgets.NavigatorOf(bc)}
	}
	return widgets.NewHero(tag, newTransitionableNavigationBar(transitionableNavigationBarOpts{
		componentsKeys: d.keys, backgroundColor: foundation.Some(effectiveBackground),
		backButtonTextStyle: textTheme.NavActionTextStyle(), titleTextStyle: textTheme.NavTitleTextStyle(),
		largeTitleTextStyle: textTheme.NavLargeTitleTextStyle(), border: effectiveBorder,
		hasUserMiddle: d.userMiddle != nil && (d.alwaysShowMiddle || !showLargeTitle), largeExpanded: showLargeTitle,
		searchable: d.searchable, automaticBackgroundVisibility: d.automaticBackgroundVisibility, child: navBar,
	}), widgets.HeroOpts{CreateRectTween: linearTranslateWithLargestRectSizeTween, PlaceholderBuilder: navBarHeroLaunchPadBuilder,
		FlightShuttleBuilder: navBarHeroFlightShuttleBuilder, TransitionOnUserGestures: true})
}

func (d *largeTitleNavigationBarSliverDelegate) ShouldRebuild(old widgets.SliverPersistentHeaderDelegate) bool {
	o, ok := old.(*largeTitleNavigationBarSliverDelegate)
	if !ok {
		return true
	}
	return d.components != o.components || d.userMiddle != o.userMiddle || d.backgroundColor != o.backgroundColor ||
		d.automaticBackgroundVisibility != o.automaticBackgroundVisibility || d.border != o.border || d.padding != o.padding ||
		d.actionsForegroundColor != o.actionsForegroundColor || d.transitionBetweenRoutes != o.transitionBetweenRoutes ||
		d.persistentHeight != o.persistentHeight || d.largeTitleHeight != o.largeTitleHeight || d.alwaysShowMiddle != o.alwaysShowMiddle ||
		d.heroTag != o.heroTag || d.enableBackgroundFilterBlur != o.enableBackgroundFilterBlur || d.bottomMode != o.bottomMode ||
		d.bottomHeight != o.bottomHeight || d.controller != o.controller || d.searchable != o.searchable
}

// inactiveSearchableBottom is the bottom of the search bar when the search is not active: the search field,
// that does not take input and that a tap activates, with the Cancel button growing in beside it.
//
// Flutter: cupertino/nav_bar.dart _InactiveSearchableBottom
type inactiveSearchableBottom struct {
	widgets.StatelessWidget
	animationController *animation.AnimationController
	searchField         widgets.Widget
	animation           animation.Animation[float64]
	searchFieldHeight   float64
	onSearchFieldTap    func()
}

func newInactiveSearchableBottom(c *animation.AnimationController, a animation.Animation[float64], searchField widgets.Widget, height float64, onTap func()) *inactiveSearchableBottom {
	return &inactiveSearchableBottom{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), c, searchField, a, height, onTap}
}

func (w *inactiveSearchableBottom) Build(widgets.BuildContext) widgets.Widget {
	child := widgets.NewGestureDetector(widgets.NewAbsorbPointer(widgets.NewFocusableActionDetector(
		widgets.NewPadding(painting.EdgeInsetsDirectional{Start: navBarEdgePadding, End: navBarEdgePadding, Bottom: navBarBottomPadding},
			widgets.NewSizedBox(w.searchField, widgets.SizedBoxOpts{Height: foundation.Some(w.searchFieldHeight)})),
		widgets.FocusableActionDetectorOpts{DescendantsAreNotFocusable: true})), widgets.GestureDetectorOpts{OnTap: w.onSearchFieldTap})
	return widgets.NewAnimatedBuilder(w.animation, func(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
		return widgets.NewLayoutBuilder(func(bc widgets.BuildContext, constraints rendering.BoxConstraints) widgets.Widget {
			return widgets.NewRow([]widgets.Widget{
				widgets.NewSizedBox(child, widgets.SizedBoxOpts{Width: foundation.Some(constraints.MaxWidth - navBarSearchFieldCancelButtonWidth*w.animationController.Value())}),
				widgets.NewSizedBox(widgets.NewPadding(painting.EdgeInsets{Bottom: navBarBottomPadding}, newCancelButton(0.4, func() {})),
					widgets.SizedBoxOpts{Width: foundation.Some(w.animationController.Value() * navBarSearchFieldCancelButtonWidth)}),
			})
		})
	}, child)
}

// activeSearchableBottom is the bottom of the search bar when the search is active: the search field and the
// Cancel button.
//
// Flutter: cupertino/nav_bar.dart _ActiveSearchableBottom
type activeSearchableBottom struct {
	widgets.StatelessWidget
	animationController *animation.AnimationController
	searchField         widgets.Widget
	animation           animation.Animation[float64]
	searchFieldHeight   float64
	onSearchFieldTap    func()
}

func newActiveSearchableBottom(c *animation.AnimationController, a animation.Animation[float64], searchField widgets.Widget, height float64, onTap func()) *activeSearchableBottom {
	return &activeSearchableBottom{widgets.NewStatelessWidgetBase(widgets.WidgetOpts{}), c, searchField, a, height, onTap}
}

func (w *activeSearchableBottom) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewPadding(painting.EdgeInsetsDirectional{Start: navBarEdgePadding, Bottom: navBarBottomPadding},
		widgets.NewRow([]widgets.Widget{
			widgets.NewExpanded(widgets.NewSizedBox(orShrink(w.searchField), widgets.SizedBoxOpts{Height: foundation.Some(w.searchFieldHeight)})),
			widgets.NewAnimatedBuilder(w.animation, func(bc widgets.BuildContext, child widgets.Widget) widgets.Widget {
				return widgets.NewSizedBox(child, widgets.SizedBoxOpts{Width: foundation.Some(w.animationController.Value() * navBarSearchFieldCancelButtonWidth)})
			}, widgets.NewFadeTransition(animation.Drive[float64](w.animationController, animation.NewTween(0.0, 1.0)), newCancelButton(1.0, w.onSearchFieldTap))),
		}, widgets.RowOpts{Spacing: 12.0})) // Eyeballed on an iPhone 15 simulator running iOS 17.5.
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagNavBarHeroTagWithoutTransition = "NavigationBarHeroTagWithoutTransition"
	tagNavBarLargeTitleRequired       = "NavigationBarLargeTitleRequired"
	tagNavBarBottomModeWithoutBottom  = "NavigationBarBottomModeWithoutBottom"
)

// keyedWidget is the keyed subtree as a widget, or nil (not a nil pointer in an interface) when there is none.
func keyedWidget(k *widgets.KeyedSubtree) widgets.Widget {
	if k == nil {
		return nil
	}
	return k
}
