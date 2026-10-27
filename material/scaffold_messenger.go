// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/widgets"
)

// completer is Dart's Completer for a value that is delivered once, as a
// channel: the framework touches it on the UI thread only.
type completer[T any] struct {
	ch   chan T
	done bool
}

func newCompleter[T any]() *completer[T] { return &completer[T]{ch: make(chan T, 1)} }

// complete delivers the value to the channel; a completed completer ignores it.
func (c *completer[T]) complete(v T) {
	if c.done {
		return
	}
	c.done = true
	c.ch <- v
	close(c.ch)
}

// ScaffoldFeatureController is the interface for controlling a feature of a
// Scaffold, such as a SnackBar or a MaterialBanner. It is returned by
// ScaffoldMessengerState.ShowSnackBar and ShowMaterialBanner.
//
// Flutter: material/scaffold.dart ScaffoldFeatureController
type ScaffoldFeatureController[T widgets.Widget, U any] struct {
	widget    T
	completer *completer[U]
	close     func()
	setState  func(func())
}

// Widget is the widget that the feature shows.
//
// Flutter: material/scaffold.dart ScaffoldFeatureController._widget
func (c *ScaffoldFeatureController[T, U]) Widget() T { return c.widget }

// Closed is a channel that receives the reason when the feature is closed. It is
// Flutter's Future of the controller.
//
// Flutter: material/scaffold.dart ScaffoldFeatureController.closed
func (c *ScaffoldFeatureController[T, U]) Closed() <-chan U { return c.completer.ch }

// Close removes the feature (with the hide reason; the animation runs).
//
// Flutter: material/scaffold.dart ScaffoldFeatureController.close
func (c *ScaffoldFeatureController[T, U]) Close() { c.close() }

// SetState marks the feature as needing a rebuild, when it has a builder: the
// snack bar and the banner have none, so it does nothing for them.
//
// Flutter: material/scaffold.dart ScaffoldFeatureController.setState
func (c *ScaffoldFeatureController[T, U]) SetState(fn func()) {
	if c.setState != nil {
		c.setState(fn)
		return
	}
	fn()
}

// ScaffoldMessengerOpts are the optional parameters of NewScaffoldMessenger.
//
// Flutter: material/scaffold.dart ScaffoldMessenger.new
type ScaffoldMessengerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
}

// ScaffoldMessenger manages SnackBars and MaterialBanners for descendant
// Scaffolds. MaterialApp creates one by default.
//
// Flutter: material/scaffold.dart ScaffoldMessenger
type ScaffoldMessenger struct {
	widgets.StatefulWidget
	child widgets.Widget
}

// NewScaffoldMessenger creates the widget.
//
// Flutter: material/scaffold.dart ScaffoldMessenger.new
func NewScaffoldMessenger(child widgets.Widget, opts ...ScaffoldMessengerOpts) *ScaffoldMessenger {
	o := firstOpts(opts)
	return &ScaffoldMessenger{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), child: child}
}

// Child is the widget below this widget in the tree.
//
// Flutter: material/scaffold.dart ScaffoldMessenger.child
func (w *ScaffoldMessenger) Child() widgets.Widget { return w.child }

// CreateState creates the state.
//
// Flutter: material/scaffold.dart ScaffoldMessenger.createState
func (w *ScaffoldMessenger) CreateState() widgets.State { return &ScaffoldMessengerState{} }

// ScaffoldMessengerOf returns the state of the closest ScaffoldMessenger above
// the context, which the context depends on. It panics with the tag
// ScaffoldMessengerMissing in debug builds when there is none.
//
// Flutter: material/scaffold.dart ScaffoldMessenger.of
func ScaffoldMessengerOf(bc widgets.BuildContext) *ScaffoldMessengerState {
	scope, ok := widgets.DependOn[*scaffoldMessengerScope](bc)
	if !ok {
		panic(errors.New(tagScaffoldMessengerMissing, "No ScaffoldMessenger widget found. "+
			foundation.RuntimeTypeName(bc.Widget())+" widgets require a ScaffoldMessenger widget ancestor. MaterialApp makes one.",
			"widget", foundation.RuntimeTypeName(bc.Widget())))
	}
	return scope.state
}

// ScaffoldMessengerMaybeOf is like ScaffoldMessengerOf but returns nil when there
// is no ScaffoldMessenger.
//
// Flutter: material/scaffold.dart ScaffoldMessenger.maybeOf
func ScaffoldMessengerMaybeOf(bc widgets.BuildContext) *ScaffoldMessengerState {
	if scope, ok := widgets.DependOn[*scaffoldMessengerScope](bc); ok {
		return scope.state
	}
	return nil
}

// ShowSnackBarOpts are the optional parameters of ShowSnackBar.
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.showSnackBar
type ShowSnackBarOpts struct {
	// SnackBarAnimationStyle customizes the animation of the snack bar: its
	// duration and reverse duration.
	SnackBarAnimationStyle *animation.AnimationStyle
}

// ScaffoldMessengerState is the state of a ScaffoldMessenger. It shows the
// snack bars and banners in the root Scaffold of the scaffolds below it.
//
// Flutter: material/scaffold.dart ScaffoldMessengerState
type ScaffoldMessengerState struct {
	widgets.StateOf[*ScaffoldMessenger]
	widgets.TickerProvider

	scaffolds []*ScaffoldState

	materialBanners          []*ScaffoldFeatureController[*MaterialBanner, MaterialBannerClosedReason]
	materialBannerController *animation.AnimationController

	snackBars          []*ScaffoldFeatureController[*SnackBar, SnackBarClosedReason]
	snackBarController *animation.AnimationController
	snackBarTimer      *gestures.Timer
	// statusHandles remove the status listeners of the controllers.
	accessibleNavigation bool
}

// DidChangeDependencies reads the accessibility setting.
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.didChangeDependencies
func (s *ScaffoldMessengerState) DidChangeDependencies() {
	s.accessibleNavigation = widgets.MediaQueryAccessibleNavigationOf(s.Context())
}

func (s *ScaffoldMessengerState) register(scaffold *ScaffoldState) {
	// Flutter's _scaffolds is a Set: a scaffold whose dependencies change registers again, and
	// the one unregister at its disposal must remove it.
	if !slices.Contains(s.scaffolds, scaffold) {
		s.scaffolds = append(s.scaffolds, scaffold)
	}
	if s.isRoot(scaffold) {
		if len(s.snackBars) > 0 {
			scaffold.updateSnackBar()
		}
		if len(s.materialBanners) > 0 {
			scaffold.updateMaterialBanner()
		}
	}
}

func (s *ScaffoldMessengerState) unregister(scaffold *ScaffoldState) {
	if i := slices.Index(s.scaffolds, scaffold); i >= 0 {
		s.scaffolds = slices.Delete(s.scaffolds, i, i+1)
	}
}

func (s *ScaffoldMessengerState) updateScaffolds() {
	for _, scaffold := range slices.Clone(s.scaffolds) {
		if s.isRoot(scaffold) {
			scaffold.updateSnackBar()
			scaffold.updateMaterialBanner()
		}
	}
}

// isRoot says whether the scaffold is not nested in another scaffold that this
// messenger knows. Nested Scaffolds are handled by the ScaffoldMessenger by only
// presenting a MaterialBanner or SnackBar in the root Scaffold of the nested set.
func (s *ScaffoldMessengerState) isRoot(scaffold *ScaffoldState) bool {
	parent, ok := widgets.FindAncestorStateOfType[*ScaffoldState](scaffold.Context())
	return !ok || !slices.Contains(s.scaffolds, parent)
}

// ShowSnackBar shows a SnackBar across all registered Scaffolds. Scaffolds
// register to receive snack bars from their closest ScaffoldMessenger ancestor.
// If there are several registered scaffolds, a snack bar is shown by the root
// Scaffold of the nested set. In debug builds it panics with the tag
// ShowSnackBarNoScaffold when there is no Scaffold below the messenger.
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.showSnackBar
func (s *ScaffoldMessengerState) ShowSnackBar(snackBar *SnackBar, opts ...ShowSnackBarOpts) *ScaffoldFeatureController[*SnackBar, SnackBarClosedReason] {
	o := firstOpts(opts)
	if foundation.DebugMode && len(s.scaffolds) == 0 {
		panic(errors.New(tagShowSnackBarNoScaffold, "ScaffoldMessenger.showSnackBar was called, but there are currently no descendant Scaffolds to present to."))
	}
	var duration, reverseDuration foundation.Opt[time.Duration]
	if st := o.SnackBarAnimationStyle; st != nil {
		duration, reverseDuration = st.Duration, st.ReverseDuration
		s.didUpdateAnimationStyle(st)
	}
	if s.snackBarController == nil {
		s.snackBarController = SnackBarCreateAnimationController(s, duration, reverseDuration)
		s.snackBarController.AddStatusListener(s.handleSnackBarStatusChanged)
	}
	if len(s.snackBars) == 0 {
		s.snackBarController.Forward()
	}
	controller := &ScaffoldFeatureController[*SnackBar, SnackBarClosedReason]{
		// We provide a fallback key so that if back-to-back snackbars happen to
		// match in structure, material ink splashes and highlights don't survive
		// from one to the next.
		widget:    snackBar.WithAnimation(s.snackBarController, foundation.NewUniqueKey()),
		completer: newCompleter[SnackBarClosedReason](),
		close:     func() { s.HideCurrentSnackBar() },
	}
	s.SetState(func() { s.snackBars = append(s.snackBars, controller) })
	s.updateScaffolds()
	return controller
}

func (s *ScaffoldMessengerState) didUpdateAnimationStyle(st *animation.AnimationStyle) {
	if s.snackBarController != nil && (s.snackBarController.Duration != st.Duration || s.snackBarController.ReverseDuration != st.ReverseDuration) {
		s.snackBarController.Dispose()
		s.snackBarController = nil
	}
}

func (s *ScaffoldMessengerState) handleSnackBarStatusChanged(status animation.AnimationStatus) {
	switch status {
	case animation.AnimationStatusDismissed:
		s.SetState(func() { s.snackBars = s.snackBars[1:] })
		s.updateScaffolds()
		if len(s.snackBars) > 0 {
			s.snackBarController.Forward()
		}
	case animation.AnimationStatusCompleted:
		// build will create a new timer if necessary to dismiss the snackBar.
		s.SetState(func() {})
		s.updateScaffolds()
	}
}

// RemoveCurrentSnackBar removes the current SnackBar (if any) immediately from
// registered Scaffolds, with the reason (remove by default).
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.removeCurrentSnackBar
func (s *ScaffoldMessengerState) RemoveCurrentSnackBar(reason ...SnackBarClosedReason) {
	r := SnackBarClosedReasonRemove
	if len(reason) > 0 {
		r = reason[0]
	}
	if len(s.snackBars) == 0 {
		return
	}
	s.snackBars[0].completer.complete(r)
	if s.snackBarTimer != nil {
		s.snackBarTimer.Cancel()
		s.snackBarTimer = nil
	}
	// This will trigger the animation's status callback.
	s.snackBarController.SetValue(0.0)
}

// HideCurrentSnackBar removes the current SnackBar by running its normal exit
// animation, with the reason (hide by default).
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.hideCurrentSnackBar
func (s *ScaffoldMessengerState) HideCurrentSnackBar(reason ...SnackBarClosedReason) {
	r := SnackBarClosedReasonHide
	if len(reason) > 0 {
		r = reason[0]
	}
	if len(s.snackBars) == 0 || s.snackBarController.Status().IsDismissed() {
		return
	}
	completer := s.snackBars[0].completer
	if s.accessibleNavigation {
		s.snackBarController.SetValue(0.0)
		completer.complete(r)
	} else {
		s.snackBarController.Reverse().Then(func() { completer.complete(r) })
	}
	if s.snackBarTimer != nil {
		s.snackBarTimer.Cancel()
		s.snackBarTimer = nil
	}
}

// ClearSnackBars removes all the SnackBars currently in queue by clearing the
// queue and running normal exit animation on the current SnackBar.
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.clearSnackBars
func (s *ScaffoldMessengerState) ClearSnackBars() {
	if len(s.snackBars) == 0 || s.snackBarController.Status().IsDismissed() {
		return
	}
	current := s.snackBars[0]
	s.snackBars = []*ScaffoldFeatureController[*SnackBar, SnackBarClosedReason]{current}
	s.HideCurrentSnackBar()
}

// ShowMaterialBanner shows a MaterialBanner across all registered Scaffolds. In
// debug builds it panics with the tag ShowMaterialBannerNoScaffold when there is
// no Scaffold below the messenger.
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.showMaterialBanner
func (s *ScaffoldMessengerState) ShowMaterialBanner(materialBanner *MaterialBanner) *ScaffoldFeatureController[*MaterialBanner, MaterialBannerClosedReason] {
	if foundation.DebugMode && len(s.scaffolds) == 0 {
		panic(errors.New(tagShowMaterialBannerNoScaffold, "ScaffoldMessenger.showMaterialBanner was called, but there are currently no descendant Scaffolds to present to."))
	}
	if s.materialBannerController == nil {
		s.materialBannerController = MaterialBannerCreateAnimationController(s)
		s.materialBannerController.AddStatusListener(s.handleMaterialBannerStatusChanged)
	}
	if len(s.materialBanners) == 0 {
		s.materialBannerController.Forward()
	}
	controller := &ScaffoldFeatureController[*MaterialBanner, MaterialBannerClosedReason]{
		// We provide a fallback key so that if back-to-back material banners happen
		// to match in structure, material ink splashes and highlights don't survive
		// from one to the next.
		widget:    materialBanner.WithAnimation(s.materialBannerController, foundation.NewUniqueKey()),
		completer: newCompleter[MaterialBannerClosedReason](),
		close:     func() { s.HideCurrentMaterialBanner() },
	}
	s.SetState(func() { s.materialBanners = append(s.materialBanners, controller) })
	s.updateScaffolds()
	return controller
}

func (s *ScaffoldMessengerState) handleMaterialBannerStatusChanged(status animation.AnimationStatus) {
	switch status {
	case animation.AnimationStatusDismissed:
		s.SetState(func() { s.materialBanners = s.materialBanners[1:] })
		s.updateScaffolds()
		if len(s.materialBanners) > 0 {
			s.materialBannerController.Forward()
		}
	case animation.AnimationStatusCompleted:
		s.updateScaffolds()
	}
}

// RemoveCurrentMaterialBanner removes the current MaterialBanner (if any)
// immediately from registered Scaffolds, with the reason (remove by default).
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.removeCurrentMaterialBanner
func (s *ScaffoldMessengerState) RemoveCurrentMaterialBanner(reason ...MaterialBannerClosedReason) {
	r := MaterialBannerClosedReasonRemove
	if len(reason) > 0 {
		r = reason[0]
	}
	if len(s.materialBanners) == 0 {
		return
	}
	s.materialBanners[0].completer.complete(r)
	// This will trigger the animation's status callback.
	s.materialBannerController.SetValue(0.0)
}

// HideCurrentMaterialBanner removes the current MaterialBanner by running its
// normal exit animation, with the reason (hide by default).
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.hideCurrentMaterialBanner
func (s *ScaffoldMessengerState) HideCurrentMaterialBanner(reason ...MaterialBannerClosedReason) {
	r := MaterialBannerClosedReasonHide
	if len(reason) > 0 {
		r = reason[0]
	}
	if len(s.materialBanners) == 0 || s.materialBannerController.Status().IsDismissed() {
		return
	}
	completer := s.materialBanners[0].completer
	if s.accessibleNavigation {
		s.materialBannerController.SetValue(0.0)
		completer.complete(r)
	} else {
		s.materialBannerController.Reverse().Then(func() { completer.complete(r) })
	}
}

// ClearMaterialBanners removes all the MaterialBanners currently in queue by
// clearing the queue and running normal exit animation on the current
// MaterialBanner.
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.clearMaterialBanners
func (s *ScaffoldMessengerState) ClearMaterialBanners() {
	if len(s.materialBanners) == 0 || s.materialBannerController.Status().IsDismissed() {
		return
	}
	current := s.materialBanners[0]
	s.materialBanners = []*ScaffoldFeatureController[*MaterialBanner, MaterialBannerClosedReason]{current}
	s.HideCurrentMaterialBanner()
}

// Build starts the timer of the current snack bar, when it is fully shown, and
// provides the messenger to its descendants.
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.build
func (s *ScaffoldMessengerState) Build(bc widgets.BuildContext) widgets.Widget {
	widgets.DebugCheckHasMediaQuery(bc)
	s.accessibleNavigation = widgets.MediaQueryAccessibleNavigationOf(bc)
	if len(s.snackBars) > 0 {
		route := widgets.ModalRouteOf(bc)
		if route == nil || route.IsCurrent() {
			if s.snackBarController.Status().IsCompleted() && s.snackBarTimer == nil {
				snackBar := s.snackBars[0].widget
				s.snackBarTimer = widgets.WidgetsBindingOf(bc).Gestures().NewTimer(snackBar.Duration(), func() {
					// Look up MediaQuery again in case the setting changed.
					if snackBar.Persist() {
						return
					}
					s.HideCurrentSnackBar(SnackBarClosedReasonTimeout)
				})
			}
		}
	}
	return &scaffoldMessengerScope{InheritedWidget: widgets.NewInheritedWidgetBase(s.Widget().child), state: s}
}

// Dispose releases the controllers and the timer.
//
// Flutter: material/scaffold.dart ScaffoldMessengerState.dispose
func (s *ScaffoldMessengerState) Dispose() {
	if s.materialBannerController != nil {
		s.materialBannerController.Dispose()
	}
	if s.snackBarController != nil {
		s.snackBarController.Dispose()
	}
	if s.snackBarTimer != nil {
		s.snackBarTimer.Cancel()
		s.snackBarTimer = nil
	}
}

// scaffoldMessengerScope provides the state of a ScaffoldMessenger.
//
// Flutter: material/scaffold.dart _ScaffoldMessengerScope
type scaffoldMessengerScope struct {
	widgets.InheritedWidget
	state *ScaffoldMessengerState
}

func (w *scaffoldMessengerScope) UpdateShouldNotify(old widgets.Widget) bool {
	return w.state != old.(*scaffoldMessengerScope).state
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagScaffoldMessengerMissing     = "ScaffoldMessengerMissing"
	tagShowMaterialBannerNoScaffold = "ShowMaterialBannerNoScaffold"
	tagShowSnackBarNoScaffold       = "ShowSnackBarNoScaffold"
)
