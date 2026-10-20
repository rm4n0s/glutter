// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"math"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
	"github.com/rm4n0s/glutter/widgets"
)

// TabControllerOpts are the optional parameters of NewTabController.
//
// Flutter: material/tab_controller.dart TabController.new
type TabControllerOpts struct {
	// InitialIndex is the index of the selected tab at first; the default is 0.
	InitialIndex int
	// AnimationDuration is the duration of the animation of a tab change; the
	// default is TabScrollDuration.
	AnimationDuration foundation.Opt[time.Duration]
}

// TabController coordinates tab selection between a TabBar and a TabBarView. The
// index is the selected tab. The animation drives from the previously selected
// tab to the newly selected one, and is also driven by a drag of the TabBarView.
//
// Flutter: material/tab_controller.dart TabController
type TabController struct {
	foundation.ChangeNotifier
	animationController  *animation.AnimationController
	animationDuration    time.Duration
	length               int
	index                int
	previousIndex        int
	indexIsChangingCount int
}

// NewTabController creates a controller for length tabs, whose animation gets its
// ticks from vsync. In debug builds it panics with the tag TabControllerLength or
// TabControllerInitialIndex when the arguments are not valid.
//
// Flutter: material/tab_controller.dart TabController.new
func NewTabController(length int, vsync scheduler.TickerProvider, opts ...TabControllerOpts) *TabController {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if length < 0 {
			panic(errors.New("TabControllerLength", "the length of a TabController must not be negative", "length", length))
		}
		if o.InitialIndex < 0 || (length != 0 && o.InitialIndex >= length) {
			panic(errors.New("TabControllerInitialIndex", "the initial index of a TabController must be one of its tabs",
				"initialIndex", o.InitialIndex, "length", length))
		}
	}
	return &TabController{
		index: o.InitialIndex, previousIndex: o.InitialIndex, length: length,
		animationDuration: o.AnimationDuration.OrElse(TabScrollDuration),
		animationController: animation.NewAnimationControllerUnbounded(vsync, animation.AnimationControllerOpts{
			Value: foundation.Some(float64(o.InitialIndex))}),
	}
}

// copyWithAndDispose is _copyWithAndDispose: it makes a new controller that
// takes over the animation controller of this one, which is disposed of.
func (c *TabController) copyWithAndDispose(index foundation.Opt[int], length foundation.Opt[int], previousIndex foundation.Opt[int],
	animationDuration foundation.Opt[time.Duration]) *TabController {
	if i, ok := index.Get(); ok {
		c.animationController.SetValue(float64(i))
	}
	result := &TabController{
		index: index.OrElse(c.index), length: length.OrElse(c.length), animationController: c.animationController,
		previousIndex: previousIndex.OrElse(c.previousIndex), animationDuration: animationDuration.OrElse(c.animationDuration),
	}
	c.animationController = nil
	c.Dispose()
	return result
}

// Animation is the animation that goes from the previously selected tab to the
// newly selected one, or nil when the controller is disposed. Its value is the
// position in tabs: it is between two indices while the tabs change or the
// TabBarView is dragged.
//
// Flutter: material/tab_controller.dart TabController.animation
func (c *TabController) Animation() animation.Animation[float64] {
	if c.animationController == nil {
		return nil
	}
	return c.animationController.View()
}

// AnimationDuration is the duration of the animation of a tab change.
//
// Flutter: material/tab_controller.dart TabController.animationDuration
func (c *TabController) AnimationDuration() time.Duration { return c.animationDuration }

// Length is the total number of tabs.
//
// Flutter: material/tab_controller.dart TabController.length
func (c *TabController) Length() int { return c.length }

func (c *TabController) changeIndex(value int, duration foundation.Opt[time.Duration], curve animation.Curve) {
	if foundation.DebugMode && (value < 0 || (value >= c.length && c.length != 0)) {
		panic(errors.New("TabControllerIndexRange", "the index of a TabController must be one of its tabs", "index", value, "length", c.length))
	}
	if value == c.index || c.length < 2 {
		return
	}
	c.previousIndex = c.index
	c.index = value
	if d, ok := duration.Get(); ok && d > 0 {
		c.indexIsChangingCount++
		c.NotifyListeners() // Because the value of indexIsChanging may have changed.
		controller := c.animationController
		controller.AnimateTo(float64(c.index), animation.AnimateOpts{Duration: duration, Curve: curve}).WhenCompleteOrCancel(func() {
			if c.animationController != nil {
				// Don't notify if we've been disposed.
				c.indexIsChangingCount--
				c.NotifyListeners()
			}
		})
	} else {
		c.indexIsChangingCount++
		c.animationController.SetValue(float64(c.index))
		c.indexIsChangingCount--
		c.NotifyListeners()
	}
}

// Index is the index of the currently selected tab. Changing it changes the
// selection at once, without an animation.
//
// Flutter: material/tab_controller.dart TabController.index
func (c *TabController) Index() int { return c.index }

// SetIndex selects the tab at once, without an animation.
//
// Flutter: material/tab_controller.dart TabController.index
func (c *TabController) SetIndex(value int) {
	c.changeIndex(value, foundation.None[time.Duration](), nil)
}

// PreviousIndex is the index of the previously selected tab.
//
// Flutter: material/tab_controller.dart TabController.previousIndex
func (c *TabController) PreviousIndex() int { return c.previousIndex }

// IndexIsChanging is whether the index is changing as a result of the AnimateTo
// call.
//
// Flutter: material/tab_controller.dart TabController.indexIsChanging
func (c *TabController) IndexIsChanging() bool { return c.indexIsChangingCount != 0 }

// AnimateToOpts are the optional parameters of TabController.AnimateTo.
//
// Flutter: material/tab_controller.dart TabController.animateTo
type AnimateToOpts struct {
	// Duration is the duration of the animation; the default is the animation
	// duration of the controller.
	Duration foundation.Opt[time.Duration]
	// Curve is the curve of the animation; the default is Curves.Ease.
	Curve animation.Curve
}

// AnimateTo changes the selected tab with an animation.
//
// Flutter: material/tab_controller.dart TabController.animateTo
func (c *TabController) AnimateTo(value int, opts ...AnimateToOpts) {
	o := firstOpts(opts)
	curve := o.Curve
	if curve == nil {
		curve = animation.Curves.Ease
	}
	c.changeIndex(value, foundation.Some(o.Duration.OrElse(c.animationDuration)), curve)
}

// Offset is the difference between the animation value and the index, from -1 to
// 1: it is how far the TabBarView is dragged from the selected tab.
//
// Flutter: material/tab_controller.dart TabController.offset
func (c *TabController) Offset() float64 { return c.animationController.Value() - float64(c.index) }

// SetOffset moves the animation away from the selected tab. In debug builds it
// panics with the tag TabControllerOffset when the offset is outside -1 to 1 or
// the index is changing.
//
// Flutter: material/tab_controller.dart TabController.offset
func (c *TabController) SetOffset(value float64) {
	if foundation.DebugMode && (value < -1.0 || value > 1.0 || c.IndexIsChanging()) {
		panic(errors.New("TabControllerOffset", "the offset of a TabController must be from -1 to 1, and the index must not be changing",
			"offset", value, "indexIsChanging", c.IndexIsChanging()))
	}
	if value == c.Offset() {
		return
	}
	c.animationController.SetValue(value + float64(c.index))
}

// Dispose releases the animation controller.
//
// Flutter: material/tab_controller.dart TabController.dispose
func (c *TabController) Dispose() {
	if c.animationController != nil {
		c.animationController.Dispose()
		c.animationController = nil
	}
	c.ChangeNotifier.Dispose()
}

// tabControllerScope is what a DefaultTabController tells its descendants.
//
// Flutter: material/tab_controller.dart _TabControllerScope
type tabControllerScope struct {
	widgets.InheritedWidget
	controller *TabController
	enabled    bool
}

func (w *tabControllerScope) UpdateShouldNotify(old widgets.Widget) bool {
	o := old.(*tabControllerScope)
	return w.enabled != o.enabled || w.controller != o.controller
}

// DefaultTabControllerOpts are the optional parameters of NewDefaultTabController.
//
// Flutter: material/tab_controller.dart DefaultTabController.new
type DefaultTabControllerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// InitialIndex is the index of the selected tab at first.
	InitialIndex int
	// AnimationDuration is the duration of the animation of a tab change.
	AnimationDuration foundation.Opt[time.Duration]
}

// DefaultTabController is the TabController for descendant widgets that do not
// specify one explicitly.
//
// Flutter: material/tab_controller.dart DefaultTabController
type DefaultTabController struct {
	widgets.StatefulWidget
	length int
	child  widgets.Widget
	o      DefaultTabControllerOpts
}

// NewDefaultTabController creates the widget. In debug builds it panics with the
// tag TabControllerLength or TabControllerInitialIndex when the arguments are not
// valid.
//
// Flutter: material/tab_controller.dart DefaultTabController.new
func NewDefaultTabController(length int, child widgets.Widget, opts ...DefaultTabControllerOpts) *DefaultTabController {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if length < 0 {
			panic(errors.New("TabControllerLength", "the length of a DefaultTabController must not be negative", "length", length))
		}
		if length != 0 && (o.InitialIndex < 0 || o.InitialIndex >= length) {
			panic(errors.New("TabControllerInitialIndex", "the initial index of a DefaultTabController must be one of its tabs",
				"initialIndex", o.InitialIndex, "length", length))
		}
	}
	return &DefaultTabController{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), length: length, child: child, o: o}
}

// Length is the total number of tabs.
//
// Flutter: material/tab_controller.dart DefaultTabController.length
func (w *DefaultTabController) Length() int { return w.length }

// Child is the widget below this widget in the tree.
//
// Flutter: material/tab_controller.dart DefaultTabController.child
func (w *DefaultTabController) Child() widgets.Widget { return w.child }

// DefaultTabControllerMaybeOf is the TabController of the closest
// DefaultTabController above the context, or nil. The context depends on it.
//
// Flutter: material/tab_controller.dart DefaultTabController.maybeOf
func DefaultTabControllerMaybeOf(bc widgets.BuildContext) *TabController {
	if scope, ok := widgets.DependOn[*tabControllerScope](bc); ok {
		return scope.controller
	}
	return nil
}

// DefaultTabControllerOf is the TabController of the closest DefaultTabController
// above the context. In debug builds it panics with the tag
// DefaultTabControllerMissing when there is none.
//
// Flutter: material/tab_controller.dart DefaultTabController.of
func DefaultTabControllerOf(bc widgets.BuildContext) *TabController {
	controller := DefaultTabControllerMaybeOf(bc)
	if controller == nil && foundation.DebugMode {
		panic(errors.New("DefaultTabControllerMissing", "DefaultTabControllerOf() was called with a context that does not contain a DefaultTabController widget.",
			"widget", foundation.RuntimeTypeName(bc.Widget())))
	}
	return controller
}

// CreateState creates the state.
//
// Flutter: material/tab_controller.dart DefaultTabController.createState
func (w *DefaultTabController) CreateState() widgets.State { return &defaultTabControllerState{} }

type defaultTabControllerState struct {
	widgets.StateOf[*DefaultTabController]
	widgets.SingleTickerProvider
	controller *TabController
}

func (s *defaultTabControllerState) InitState() {
	w := s.Widget()
	s.controller = NewTabController(w.length, s, TabControllerOpts{InitialIndex: w.o.InitialIndex, AnimationDuration: w.o.AnimationDuration})
}

func (s *defaultTabControllerState) Dispose() { s.controller.Dispose() }

func (s *defaultTabControllerState) Build(bc widgets.BuildContext) widgets.Widget {
	return &tabControllerScope{
		InheritedWidget: widgets.NewInheritedWidgetBase(s.Widget().child),
		controller:      s.controller,
		enabled:         widgets.TickerModeValuesOf(bc).Enabled,
	}
}

func (s *defaultTabControllerState) DidUpdateWidget(old *DefaultTabController) {
	w := s.Widget()
	if old.length != w.length {
		// If the length is shortened while the last tab is selected, we should
		// automatically update the index of the controller to be the new last tab.
		newIndex := foundation.None[int]()
		previousIndex := s.controller.PreviousIndex()
		if s.controller.Index() >= w.length {
			newIndex = foundation.Some(int(math.Max(0, float64(w.length-1))))
			previousIndex = s.controller.Index()
		}
		s.controller = s.controller.copyWithAndDispose(newIndex, foundation.Some(w.length), foundation.Some(previousIndex), w.o.AnimationDuration)
	}

	if old.o.AnimationDuration != w.o.AnimationDuration {
		s.controller = s.controller.copyWithAndDispose(foundation.Some(s.controller.Index()), foundation.Some(w.length),
			foundation.Some(s.controller.PreviousIndex()), w.o.AnimationDuration)
	}
}
