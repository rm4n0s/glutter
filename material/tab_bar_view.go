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
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TabBarViewOpts are the optional parameters of NewTabBarView.
//
// Flutter: material/tabs.dart TabBarView.new
type TabBarViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Controller coordinates the tab selection with the TabBar; the default is the
	// one of the closest DefaultTabController.
	Controller *TabController
	// Physics is how the pages respond to the user.
	Physics widgets.ScrollPhysics
	// DragStartBehavior is where a drag starts.
	DragStartBehavior gestures.DragStartBehavior
	// ViewportFraction is the fraction of the viewport that each page occupies; the
	// default is 1.
	ViewportFraction foundation.Opt[float64]
	// ClipBehavior is how the view clips its content; the default is
	// ClipHardEdge.
	ClipBehavior ui.Clip
}

// TabBarView is a page view that displays the widget which corresponds to the
// currently selected tab. It is typically used in conjunction with a TabBar.
//
// Flutter: material/tabs.dart TabBarView
type TabBarView struct {
	widgets.StatefulWidget
	children []widgets.Widget
	o        TabBarViewOpts
}

// NewTabBarView creates a view of the pages, one for each tab.
//
// Flutter: material/tabs.dart TabBarView.new
func NewTabBarView(children []widgets.Widget, opts ...TabBarViewOpts) *TabBarView {
	o := firstOpts(opts)
	return &TabBarView{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), children: children, o: o}
}

// Children are the pages.
//
// Flutter: material/tabs.dart TabBarView.children
func (w *TabBarView) Children() []widgets.Widget { return w.children }

// CreateState creates the state.
//
// Flutter: material/tabs.dart TabBarView.createState
func (w *TabBarView) CreateState() widgets.State { return &tabBarViewState{} }

type tabBarViewState struct {
	widgets.StateOf[*TabBarView]
	controller                *TabController
	pageController            *widgets.PageController
	childrenWithKey           []widgets.Widget
	currentIndex              int
	warpUnderwayCount         int
	scrollUnderwayCount       int
	hasScheduledChildrenCheck bool
	animationHandle           foundation.ListenerHandle
}

// controllerIsValid is false when the TabBarView is rebuilt with a new tab
// controller and the caller disposed the old one: its animation is nil then and
// must not be accessed.
func (s *tabBarViewState) controllerIsValid() bool {
	return s.controller != nil && s.controller.Animation() != nil
}

func (s *tabBarViewState) updateTabController() {
	newController := s.Widget().o.Controller
	if newController == nil {
		newController = DefaultTabControllerMaybeOf(s.Context())
	}
	if foundation.DebugMode && newController == nil {
		panic(errors.New("TabBarViewNoController", "No TabController for TabBarView. When creating a TabBarView, you must either provide an explicit TabController using the \"controller\" property, or you must ensure that there is a DefaultTabController above the TabBarView. In this case, there was neither an explicit controller nor a default controller."))
	}

	if newController == s.controller {
		return
	}

	if s.controllerIsValid() {
		s.controller.Animation().RemoveListener(s.animationHandle)
	}
	s.controller = newController
	if s.controller != nil {
		s.animationHandle = s.controller.Animation().AddListener(s.handleTabControllerAnimationTick)
	}
}

func (s *tabBarViewState) jumpToPage(page int) {
	s.warpUnderwayCount++
	s.pageController.JumpToPage(page)
	s.warpUnderwayCount--
}

// animateToPage animates to the page; done is called when the animation ends.
func (s *tabBarViewState) animateToPage(page int, duration time.Duration, curve animation.Curve, done func()) {
	s.warpUnderwayCount++
	s.pageController.AnimateToPage(page, duration, curve).WhenCompleteOrCancel(func() {
		s.warpUnderwayCount--
		if done != nil {
			done()
		}
	})
}

func (s *tabBarViewState) InitState() { s.updateChildren() }

func (s *tabBarViewState) DidChangeDependencies() {
	s.updateTabController()
	s.currentIndex = s.controller.Index()
	if s.pageController == nil {
		s.pageController = widgets.NewPageController(widgets.PageControllerOpts{
			InitialPage: s.currentIndex, ViewportFraction: s.Widget().o.ViewportFraction})
	} else {
		s.pageController.JumpToPage(s.currentIndex)
	}
}

func (s *tabBarViewState) DidUpdateWidget(old *TabBarView) {
	w := s.Widget()
	if w.o.Controller != old.o.Controller {
		s.updateTabController()
		s.currentIndex = s.controller.Index()
		s.jumpToPage(s.currentIndex)
	}
	if w.o.ViewportFraction != old.o.ViewportFraction {
		if s.pageController != nil {
			s.pageController.Dispose()
		}
		s.pageController = widgets.NewPageController(widgets.PageControllerOpts{
			InitialPage: s.currentIndex, ViewportFraction: w.o.ViewportFraction})
	}
	// While a warp is under way, we stop updating the tab page contents.
	if !sameWidgets(w.children, old.children) && s.warpUnderwayCount == 0 {
		s.updateChildren()
	}
}

func sameWidgets(a, b []widgets.Widget) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *tabBarViewState) Dispose() {
	if s.controllerIsValid() {
		s.controller.Animation().RemoveListener(s.animationHandle)
	}
	s.controller = nil
	if s.pageController != nil {
		s.pageController.Dispose()
	}
	// We don't own the controller Animation, so it's not disposed here.
}

func (s *tabBarViewState) updateChildren() {
	children := make([]widgets.Widget, len(s.Widget().children))
	for i, child := range s.Widget().children {
		children[i] = widgets.NewSemantics(child, widgets.SemanticsOpts{Role: foundation.Some(ui.SemanticsRoleTabPanel)})
	}
	s.childrenWithKey = widgets.KeyedSubtreeWrapList(children, 0)
}

func (s *tabBarViewState) handleTabControllerAnimationTick() {
	if s.scrollUnderwayCount > 0 || !s.controller.IndexIsChanging() {
		return
	} // This widget is driving the controller's animation.

	if s.controller.Index() != s.currentIndex {
		s.currentIndex = s.controller.Index()
		s.warpToCurrentIndex()
	}
}

func (s *tabBarViewState) warpToCurrentIndex() {
	if !s.Mounted() || s.pageController.Page() == foundation.Some(float64(s.currentIndex)) {
		return
	}

	adjacentDestination := math.Abs(float64(s.currentIndex-s.controller.PreviousIndex())) == 1
	if adjacentDestination {
		s.warpToAdjacentTab(s.controller.AnimationDuration())
	} else {
		s.warpToNonAdjacentTab(s.controller.AnimationDuration())
	}
}

func (s *tabBarViewState) refreshChildren() {
	if s.Mounted() {
		s.SetState(func() { s.updateChildren() })
	}
}

func (s *tabBarViewState) warpToAdjacentTab(duration time.Duration) {
	if duration == 0 {
		s.jumpToPage(s.currentIndex)
		s.refreshChildren()
		return
	}
	s.animateToPage(s.currentIndex, duration, animation.Curves.Ease, s.refreshChildren)
}

func (s *tabBarViewState) warpToNonAdjacentTab(duration time.Duration) {
	previousIndex := s.controller.PreviousIndex()

	// initialPage defines which page is shown when starting the animation. This
	// page is adjacent to the destination page.
	initialPage := s.currentIndex + 1
	if s.currentIndex > previousIndex {
		initialPage = s.currentIndex - 1
	}

	s.SetState(func() {
		// Needed for RenderSliverMultiBoxAdaptor.move and kept alive children.
		s.childrenWithKey = append([]widgets.Widget(nil), s.childrenWithKey...)
		s.childrenWithKey[initialPage], s.childrenWithKey[previousIndex] = s.childrenWithKey[previousIndex], s.childrenWithKey[initialPage]
	})

	// Make a first jump to the adjacent page.
	s.jumpToPage(initialPage)

	// Jump or animate to the destination page.
	if duration == 0 {
		s.jumpToPage(s.currentIndex)
		s.refreshChildren()
		return
	}
	s.animateToPage(s.currentIndex, duration, animation.Curves.Ease, s.refreshChildren)
}

func (s *tabBarViewState) syncControllerOffset() {
	s.controller.SetOffset(ui.ClampDouble(s.pageController.Page().Value()-float64(s.controller.Index()), -1.0, 1.0))
}

// handleScrollNotification is called when the PageView scrolls.
func (s *tabBarViewState) handleScrollNotification(notification widgets.ScrollNotification) bool {
	if s.warpUnderwayCount > 0 || s.scrollUnderwayCount > 0 {
		return false
	}

	if notification.Depth() != 0 {
		return false
	}

	if !s.controllerIsValid() {
		return false
	}

	s.scrollUnderwayCount++
	page := s.pageController.Page().Value()
	switch notification.(type) {
	case *widgets.ScrollUpdateNotification:
		if !s.controller.IndexIsChanging() {
			pageChanged := math.Abs(page-float64(s.controller.Index())) > 1.0
			if pageChanged {
				s.controller.SetIndex(int(math.Round(page)))
				s.currentIndex = s.controller.Index()
			}
			s.syncControllerOffset()
		}
	case *widgets.ScrollEndNotification:
		s.controller.SetIndex(int(math.Round(page)))
		s.currentIndex = s.controller.Index()
		if !s.controller.IndexIsChanging() {
			s.syncControllerOffset()
		}
	}
	s.scrollUnderwayCount--

	return false
}

func (s *tabBarViewState) scheduleCheckHasValidChildrenCount() {
	if s.hasScheduledChildrenCheck {
		return
	}
	widgets.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) {
		s.hasScheduledChildrenCheck = false
		if !s.Mounted() {
			return
		}
		if n := len(s.Widget().children); s.controller.Length() != n {
			panic(errors.New("TabBarViewChildrenCount", fmt.Sprintf("Controller's length property (%d) does not match the number of children (%d) present in TabBarView's children property.", s.controller.Length(), n),
				"controllerLength", s.controller.Length(), "children", n))
		}
	})
	s.hasScheduledChildrenCheck = true
}

func (s *tabBarViewState) Build(widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	if foundation.DebugMode {
		s.scheduleCheckHasValidChildrenCount()
	}

	var physics widgets.ScrollPhysics
	if w.o.Physics == nil {
		physics = widgets.NewPageScrollPhysics(widgets.NewClampingScrollPhysics(nil))
	} else {
		physics = widgets.NewPageScrollPhysics(w.o.Physics)
	}
	return widgets.NewNotificationListener(
		widgets.NewPageView(s.childrenWithKey, widgets.PageViewOpts{
			DragStartBehavior: w.o.DragStartBehavior, ClipBehavior: w.o.ClipBehavior, Controller: s.pageController, Physics: physics}),
		s.handleScrollNotification)
}
