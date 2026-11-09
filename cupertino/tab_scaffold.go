package cupertino

import (
	"fmt"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// CupertinoTabController manages the state of a CupertinoTabScaffold: the index of the tab that shows.
// Setting the index notifies the listeners (the scaffold rebuilds and switches tabs).
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabController
type CupertinoTabController struct {
	foundation.ChangeNotifier
	index      int
	isDisposed bool
}

// CupertinoTabControllerOpts are the optional parameters of NewCupertinoTabController.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabController.new
type CupertinoTabControllerOpts struct {
	// InitialIndex is the index of the tab that shows first; the default is 0.
	InitialIndex int
}

// NewCupertinoTabController creates a controller. In debug builds it panics with the tag
// CupertinoTabControllerIndexNegative when the initial index is negative.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabController.new
func NewCupertinoTabController(opts ...CupertinoTabControllerOpts) *CupertinoTabController {
	o := firstOf(opts)
	if foundation.DebugMode && o.InitialIndex < 0 {
		panic(errors.New(tagCupertinoTabControllerIndexNegative, "the initial index must not be negative", "index", o.InitialIndex))
	}
	return &CupertinoTabController{index: o.InitialIndex}
}

// Index is the index of the tab that shows.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabController.index
func (c *CupertinoTabController) Index() int { return c.index }

// SetIndex changes the tab that shows, and notifies the listeners when it is another one. In debug builds it
// panics with the tag CupertinoTabControllerIndexNegative when the index is negative.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabController.index
func (c *CupertinoTabController) SetIndex(value int) {
	if foundation.DebugMode && value < 0 {
		panic(errors.New(tagCupertinoTabControllerIndexNegative, "the index must not be negative", "index", value))
	}
	if c.index == value {
		return
	}
	c.index = value
	c.NotifyListeners()
}

// Dispose releases the listeners. The controller is not used after it.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabController.dispose
func (c *CupertinoTabController) Dispose() {
	c.ChangeNotifier.Dispose()
	c.isDisposed = true
}

// CupertinoTabScaffoldOpts are the optional parameters of NewCupertinoTabScaffold.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabScaffold.new
type CupertinoTabScaffoldOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller controls the tab that shows; without one the scaffold has its own, which starts at the
	// current index of the tab bar. When there is one, the current index of the tab bar is ignored.
	Controller *CupertinoTabController
	// BackgroundColor is the color behind the tabs; the default is the scaffold background color of the theme.
	BackgroundColor ui.ColorValue
	// NotResizeToAvoidBottomInset says that the content does not shrink for the keyboard (Flutter's
	// resizeToAvoidBottomInset is true by default).
	NotResizeToAvoidBottomInset bool
	// Deviation: no RestorationId (state restoration is a non-goal, SPEC §2).
}

// CupertinoTabScaffold is the layout of an iOS app with tabs: a tab bar at the bottom and the content of the
// tab that is active above it. The tabs that were built stay in the tree (their state is kept), and the tab
// builder is called for a tab the first time it is active. A tab bar that is not opaque lets the content draw
// under it.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabScaffold
type CupertinoTabScaffold struct {
	widgets.StatefulWidget
	tabBar     *CupertinoTabBar
	tabBuilder widgets.IndexedWidgetBuilder
	o          CupertinoTabScaffoldOpts
}

// NewCupertinoTabScaffold creates a scaffold with the tab bar and the builder of the tabs. In debug builds it
// panics with the tag CupertinoTabScaffoldControllerIndexOutOfRange when the index of the controller is not a
// tab of the bar.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabScaffold.new
func NewCupertinoTabScaffold(tabBar *CupertinoTabBar, tabBuilder widgets.IndexedWidgetBuilder, opts ...CupertinoTabScaffoldOpts) *CupertinoTabScaffold {
	o := firstOf(opts)
	if foundation.DebugMode && o.Controller != nil && o.Controller.index >= len(tabBar.items) {
		panic(errors.New(tagCupertinoTabScaffoldControllerIndexOutOfRange,
			fmt.Sprintf("The CupertinoTabController's current index %d is out of bounds for the tab bar with %d tabs", o.Controller.index, len(tabBar.items)),
			"index", o.Controller.index, "tabs", len(tabBar.items)))
	}
	return &CupertinoTabScaffold{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), tabBar, tabBuilder, o}
}

// TabBar is the tab bar at the bottom.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabScaffold.tabBar
func (w *CupertinoTabScaffold) TabBar() *CupertinoTabBar { return w.tabBar }

// Controller is the controller that the widget was given, or nil.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabScaffold.controller
func (w *CupertinoTabScaffold) Controller() *CupertinoTabController { return w.o.Controller }

// TabBuilder builds the content of a tab.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabScaffold.tabBuilder
func (w *CupertinoTabScaffold) TabBuilder() widgets.IndexedWidgetBuilder { return w.tabBuilder }

// BackgroundColor is the color behind the tabs, or nil for the theme's.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabScaffold.backgroundColor
func (w *CupertinoTabScaffold) BackgroundColor() ui.ColorValue { return w.o.BackgroundColor }

// ResizeToAvoidBottomInset is whether the content shrinks for the keyboard.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabScaffold.resizeToAvoidBottomInset
func (w *CupertinoTabScaffold) ResizeToAvoidBottomInset() bool {
	return !w.o.NotResizeToAvoidBottomInset
}

// CreateState creates the state.
//
// Flutter: cupertino/tab_scaffold.dart CupertinoTabScaffold.createState
func (w *CupertinoTabScaffold) CreateState() widgets.State { return &cupertinoTabScaffoldState{} }

type cupertinoTabScaffoldState struct {
	widgets.StateOf[*CupertinoTabScaffold]
	internalController *CupertinoTabController
	// listened is the controller that the state listens to, and its handle.
	listened       *CupertinoTabController
	listenerHandle foundation.ListenerHandle
}

func (s *cupertinoTabScaffoldState) controller() *CupertinoTabController {
	if c := s.Widget().o.Controller; c != nil {
		return c
	}
	return s.internalController
}

func (s *cupertinoTabScaffoldState) InitState() { s.updateTabController(nil) }

// updateTabController makes the internal controller when there is none from the widget, and moves the
// listener to the controller in use.
//
// Flutter: cupertino/tab_scaffold.dart _CupertinoTabScaffoldState._updateTabController
func (s *cupertinoTabScaffoldState) updateTabController(oldWidgetController *CupertinoTabController) {
	w := s.Widget()
	if w.o.Controller == nil && s.internalController == nil {
		// No widget-provided controller: create an internal controller.
		s.internalController = NewCupertinoTabController(CupertinoTabControllerOpts{InitialIndex: w.tabBar.CurrentIndex()})
	}
	if w.o.Controller != nil && s.internalController != nil {
		// Use the widget-provided controller.
		s.internalController.Dispose()
		s.internalController = nil
	}
	s.moveListener(oldWidgetController)
}

func (s *cupertinoTabScaffoldState) moveListener(old *CupertinoTabController) {
	current := s.controller()
	if s.listened == current {
		return
	}
	if s.listened != nil && !s.listened.isDisposed {
		s.listened.RemoveListener(s.listenerHandle)
	}
	s.listened = current
	s.listenerHandle = current.AddListener(s.onCurrentIndexChange)
}

func (s *cupertinoTabScaffoldState) onCurrentIndexChange() {
	c := s.controller()
	if foundation.DebugMode && (c.index < 0 || c.index >= len(s.Widget().tabBar.items)) {
		panic(errors.New(tagCupertinoTabScaffoldControllerIndexOutOfRange,
			fmt.Sprintf("The CupertinoTabScaffold's current index %d is out of bounds for the tab bar with %d tabs", c.index, len(s.Widget().tabBar.items)),
			"index", c.index, "tabs", len(s.Widget().tabBar.items)))
	}
	// The value of the index has already been updated at this point: rebuild with it.
	s.SetState(func() {})
}

func (s *cupertinoTabScaffoldState) DidUpdateWidget(old *CupertinoTabScaffold) {
	w := s.Widget()
	switch {
	case w.o.Controller != old.o.Controller:
		s.updateTabController(old.o.Controller)
	case s.controller().index >= len(w.tabBar.items):
		// If a new tab bar with less than (index + 1) items is provided, clamp the current index.
		s.controller().SetIndex(len(w.tabBar.items) - 1)
	}
}

func (s *cupertinoTabScaffoldState) Dispose() {
	if s.listened != nil && !s.listened.isDisposed {
		s.listened.RemoveListener(s.listenerHandle)
	}
	if s.internalController != nil {
		s.internalController.Dispose()
	}
}

func (s *cupertinoTabScaffoldState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	existingMediaQuery := widgets.MediaQueryOf(bc)
	newMediaQuery := existingMediaQuery
	controller := s.controller()

	var content widgets.Widget = newTabSwitchingView(controller.index, len(w.tabBar.items), w.tabBuilder)
	contentPadding := painting.EdgeInsets{}

	if w.ResizeToAvoidBottomInset() {
		// Remove the view inset and add it back as a padding in the inner content.
		newMediaQuery = newMediaQuery.RemoveViewInsets(widgets.RemoveSides{Bottom: true})
		contentPadding = painting.EdgeInsets{Bottom: existingMediaQuery.ViewInsets.Bottom}
	}

	// Only pad the content with the height of the tab bar if the tab isn't already entirely obstructed by a
	// keyboard or other view insets. Don't double pad.
	if !w.ResizeToAvoidBottomInset() || w.tabBar.PreferredSize().Height > existingMediaQuery.ViewInsets.Bottom {
		// The preferred size of the bar is used instead of its real size (a partial layout is not done).
		bottomPadding := w.tabBar.PreferredSize().Height + existingMediaQuery.Padding.Bottom
		// If tab bar opaque, directly stop the main content higher. If translucent, let main content draw
		// behind the tab bar but hint the obstructed area.
		if w.tabBar.Opaque(bc) {
			contentPadding = painting.EdgeInsets{Bottom: bottomPadding}
			newMediaQuery = newMediaQuery.RemovePadding(widgets.RemoveSides{Bottom: true})
		} else {
			newMediaQuery = newMediaQuery.CopyWith(widgets.MediaQueryDataCopyOpts{
				Padding: foundation.Some(newMediaQuery.Padding.CopyWith(painting.EdgeInsetsCopy{Bottom: foundation.Some(bottomPadding)}))})
		}
	}

	content = widgets.NewMediaQuery(newMediaQuery, widgets.NewPadding(contentPadding, content))

	backgroundColor := CupertinoThemeOf(bc).ScaffoldBackgroundColor().EffectiveColor()
	if c := CupertinoDynamicColorMaybeResolve(w.o.BackgroundColor, bc); c.IsSome() {
		backgroundColor = c.Value()
	}
	return widgets.NewDecoratedBox(
		painting.NewBoxDecoration(painting.BoxDecorationOpts{Color: foundation.Some(backgroundColor)}),
		widgets.NewStack([]widgets.Widget{
			// The main content being at the bottom is added to the stack first.
			content,
			widgets.NewMediaQueryWithNoTextScaling(widgets.NewAlign(
				// Override the tab bar's current index to the current tab and hook in our own listener to update
				// the controller's index on top of a possibly user provided callback.
				w.tabBar.CopyWith(CupertinoTabBarCopyOpts{
					CurrentIndex: foundation.Some(controller.index),
					OnTap: func(newIndex int) {
						controller.SetIndex(newIndex)
						// Chain the user's original callback.
						if w.tabBar.OnTap() != nil {
							w.tabBar.OnTap()(newIndex)
						}
					},
				}), widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})),
		}))
}

// tabSwitchingView is the stack of the tabs of a CupertinoTabScaffold: only the active one is visible, the
// others are kept offstage, with their tickers and heroes off.
//
// Flutter: cupertino/tab_scaffold.dart _TabSwitchingView
type tabSwitchingView struct {
	widgets.StatefulWidget
	currentTabIndex int
	tabCount        int
	tabBuilder      widgets.IndexedWidgetBuilder
}

func newTabSwitchingView(currentTabIndex, tabCount int, tabBuilder widgets.IndexedWidgetBuilder) *tabSwitchingView {
	if foundation.DebugMode && tabCount <= 0 {
		panic(errors.New(tagCupertinoTabScaffoldTabCountInvalid, "the number of tabs must be positive", "tabs", tabCount))
	}
	return &tabSwitchingView{widgets.NewStatefulWidgetBase(widgets.WidgetOpts{}), currentTabIndex, tabCount, tabBuilder}
}

func (w *tabSwitchingView) CreateState() widgets.State { return &tabSwitchingViewState{} }

type tabSwitchingViewState struct {
	widgets.StateOf[*tabSwitchingView]
	shouldBuildTab []bool
	tabFocusNodes  []*widgets.FocusScopeNode
	// discardedNodes are the focus nodes no longer needed. Nothing else can be trusted to have stopped
	// listening to them before the widget is disposed, so they are disposed then.
	discardedNodes []*widgets.FocusScopeNode
}

func (s *tabSwitchingViewState) InitState() {
	s.shouldBuildTab = make([]bool, s.Widget().tabCount)
}

func (s *tabSwitchingViewState) DidChangeDependencies() { s.focusActiveTab() }

func (s *tabSwitchingViewState) DidUpdateWidget(*tabSwitchingView) {
	// Only partially invalidate the tabs cache. The only possible change is either new tabs appended to the
	// list, or some trailing tabs removed; with anything else some tabs may lose their state.
	lengthDiff := s.Widget().tabCount - len(s.shouldBuildTab)
	if lengthDiff > 0 {
		s.shouldBuildTab = append(s.shouldBuildTab, make([]bool, lengthDiff)...)
	} else if lengthDiff < 0 {
		s.shouldBuildTab = s.shouldBuildTab[:s.Widget().tabCount]
	}
	s.focusActiveTab()
}

// focusActiveTab focuses the active tab if the FocusScope above it has focus already. If not, it marks it as
// the preferred focus of that scope.
func (s *tabSwitchingViewState) focusActiveTab() {
	w := s.Widget()
	if len(s.tabFocusNodes) != w.tabCount {
		if len(s.tabFocusNodes) > w.tabCount {
			s.discardedNodes = append(s.discardedNodes, s.tabFocusNodes[w.tabCount:]...)
			s.tabFocusNodes = s.tabFocusNodes[:w.tabCount]
		} else {
			for len(s.tabFocusNodes) < w.tabCount {
				s.tabFocusNodes = append(s.tabFocusNodes, widgets.NewFocusScopeNode(widgets.FocusScopeNodeOpts{
					DebugLabel: fmt.Sprintf("CupertinoTabScaffold Tab %d", len(s.tabFocusNodes))}))
			}
		}
	}
	widgets.FocusScopeOf(s.Context()).SetFirstFocus(s.tabFocusNodes[w.currentTabIndex])
}

func (s *tabSwitchingViewState) Dispose() {
	for _, n := range s.tabFocusNodes {
		n.Dispose()
	}
	for _, n := range s.discardedNodes {
		n.Dispose()
	}
}

func (s *tabSwitchingViewState) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	children := make([]widgets.Widget, w.tabCount)
	for index := range children {
		index := index
		active := index == w.currentTabIndex
		s.shouldBuildTab[index] = active || s.shouldBuildTab[index]
		children[index] = widgets.NewHeroMode(
			widgets.NewOffstage(
				widgets.NewTickerMode(active,
					widgets.NewFocusScope(
						widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
							if s.shouldBuildTab[index] {
								return w.tabBuilder(bc, index)
							}
							return widgets.NewSizedBoxShrink(nil)
						}),
						widgets.FocusScopeOpts{Node: s.tabFocusNodes[index]})),
				widgets.OffstageOpts{Offstage: foundation.Some(!active)}),
			widgets.HeroModeOpts{Enabled: foundation.Some(active)})
	}
	return widgets.NewStack(children, widgets.StackOpts{Fit: rendering.StackFitExpand})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCupertinoTabControllerIndexNegative           = "CupertinoTabControllerIndexNegative"
	tagCupertinoTabScaffoldControllerIndexOutOfRange = "CupertinoTabScaffoldControllerIndexOutOfRange"
	tagCupertinoTabScaffoldTabCountInvalid           = "CupertinoTabScaffoldTabCountInvalid"
)
