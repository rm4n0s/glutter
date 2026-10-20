// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"reflect"
	"slices"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// ScrollViewKeyboardDismissBehavior is how a scroll view reacts to the keyboard
// when the user drags it. Linux has no on-screen keyboard to dismiss, so it is
// only stored in the behavior.
//
// Flutter: widgets/scroll_view.dart ScrollViewKeyboardDismissBehavior
type ScrollViewKeyboardDismissBehavior int

const (
	// ScrollViewKeyboardDismissBehaviorUnspecified is the zero value: the
	// behavior's default, which is manual.
	//
	// Flutter: no counterpart
	ScrollViewKeyboardDismissBehaviorUnspecified ScrollViewKeyboardDismissBehavior = iota
	// ScrollViewKeyboardDismissBehaviorManual does not dismiss the keyboard on
	// its own.
	//
	// Flutter: widgets/scroll_view.dart ScrollViewKeyboardDismissBehavior.manual
	ScrollViewKeyboardDismissBehaviorManual
	// ScrollViewKeyboardDismissBehaviorOnDrag dismisses the keyboard when the
	// user starts to drag the scroll view.
	//
	// Flutter: widgets/scroll_view.dart ScrollViewKeyboardDismissBehavior.onDrag
	ScrollViewKeyboardDismissBehaviorOnDrag
)

// String is the name with the enum, such as "ScrollViewKeyboardDismissBehavior.manual".
//
// Flutter: widgets/scroll_view.dart ScrollViewKeyboardDismissBehavior
func (b ScrollViewKeyboardDismissBehavior) String() string {
	switch b {
	case ScrollViewKeyboardDismissBehaviorManual:
		return "ScrollViewKeyboardDismissBehavior.manual"
	case ScrollViewKeyboardDismissBehaviorOnDrag:
		return "ScrollViewKeyboardDismissBehavior.onDrag"
	}
	return "ScrollViewKeyboardDismissBehavior.unspecified"
}

// touchLikeDeviceTypes are the pointer kinds that can drag a scrollable by
// default: not a mouse.
var touchLikeDeviceTypes = []ui.PointerDeviceKind{
	ui.PointerDeviceKindTouch,
	ui.PointerDeviceKindStylus,
	ui.PointerDeviceKindInvertedStylus,
	ui.PointerDeviceKindTrackpad,
	// The VoiceControl accessibility tool on iOS sends a pointer of unknown kind.
	ui.PointerDeviceKindUnknown,
}

// wrapKeyboardDismiss makes a drag of the scrollable unfocus the primary focus,
// when the effective keyboard dismiss behavior is onDrag. The behavior is the one
// given, then the one of the scroll behavior (if any), then the one of the
// ScrollConfiguration above.
func wrapKeyboardDismiss(bc BuildContext, behavior ScrollViewKeyboardDismissBehavior, scrollBehavior ScrollBehavior, scrollable Widget) Widget {
	effective := behavior
	if effective == ScrollViewKeyboardDismissBehaviorUnspecified {
		if scrollBehavior != nil {
			effective = scrollBehavior.GetKeyboardDismissBehavior(bc)
		} else {
			effective = ScrollConfigurationOf(bc).GetKeyboardDismissBehavior(bc)
		}
	}
	if effective != ScrollViewKeyboardDismissBehaviorOnDrag {
		return scrollable
	}
	return NewNotificationListener(scrollable, func(notification *ScrollUpdateNotification) bool {
		currentScope := FocusScopeOf(bc)
		if notification.DragDetails.IsSome() && !currentScope.HasPrimaryFocus() && currentScope.HasFocus() {
			if primary := bc.Owner().FocusManager().PrimaryFocus(); primary != nil {
				primary.Unfocus()
			}
		}
		return false
	})
}

// ScrollBehavior describes how a scrollable behaves on a platform: which
// pointers drag it, its physics, its scrollbar and its overscroll indicator.
// ScrollConfiguration gives it to the scrollables below it.
//
// Implement it by embedding ScrollBehaviorBase (InitScrollBehaviorBase) and
// overriding what differs.
//
// Deviation: buildOverscrollIndicator returns the child on every platform
// because the Android indicators are not ported (SPEC §12.10).
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior
type ScrollBehavior interface {
	// CopyWith returns a behavior that wraps this one, with the given values
	// changed.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.copyWith
	CopyWith(changes ...ScrollBehaviorCopy) ScrollBehavior
	// GetPlatform is the platform the scrollable behaves like; the default is
	// foundation.DefaultTargetPlatform.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.getPlatform
	GetPlatform(bc BuildContext) foundation.TargetPlatform
	// DragDevices are the kinds of pointer that can drag the scrollable.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.dragDevices
	DragDevices() []ui.PointerDeviceKind
	// GetMultitouchDragStrategy is how a drag follows several fingers.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.getMultitouchDragStrategy
	GetMultitouchDragStrategy(bc BuildContext) gestures.MultitouchDragStrategy
	// PointerAxisModifiers are the keys that make a mouse wheel scroll the other
	// axis while one of them is held; the default is the two Shift keys.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.pointerAxisModifiers
	PointerAxisModifiers() []services.LogicalKeyboardKey
	// BuildScrollbar wraps the child in a scrollbar; the default is a
	// RawScrollbar on Linux, macOS and Windows.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.buildScrollbar
	BuildScrollbar(bc BuildContext, child Widget, details ScrollableDetails) Widget
	// BuildOverscrollIndicator wraps the child in an overscroll indicator; the
	// default returns the child.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.buildOverscrollIndicator
	BuildOverscrollIndicator(bc BuildContext, child Widget, details ScrollableDetails) Widget
	// VelocityTrackerBuilder makes the velocity tracker of a pointer.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.velocityTrackerBuilder
	VelocityTrackerBuilder(bc BuildContext) gestures.GestureVelocityTrackerBuilder
	// GetScrollPhysics are the default physics of the scrollables.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.getScrollPhysics
	GetScrollPhysics(bc BuildContext) ScrollPhysics
	// ShouldNotify is whether the scrollables that depend on the configuration
	// should rebuild when this behavior replaces the old one of the same type.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.shouldNotify
	ShouldNotify(oldDelegate ScrollBehavior) bool
	// GetKeyboardDismissBehavior is how scroll views react to a drag with the
	// keyboard open.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.getKeyboardDismissBehavior
	GetKeyboardDismissBehavior(bc BuildContext) ScrollViewKeyboardDismissBehavior
	// String is the name of the type of the behavior.
	//
	// Flutter: widgets/scroll_configuration.dart ScrollBehavior.toString
	String() string
}

// ScrollBehaviorCopy are the values that ScrollBehavior.CopyWith changes.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.copyWith
type ScrollBehaviorCopy struct {
	// Scrollbars is whether to build a scrollbar; the default is true.
	Scrollbars foundation.Opt[bool]
	// Overscroll is whether to build an overscroll indicator; the default is
	// true.
	Overscroll foundation.Opt[bool]
	// DragDevices replaces the kinds of pointer that drag; nil keeps them.
	DragDevices []ui.PointerDeviceKind
	// MultitouchDragStrategy replaces the strategy; the Unspecified value keeps
	// it.
	MultitouchDragStrategy gestures.MultitouchDragStrategy
	// PointerAxisModifiers replaces the keys that flip the axis of the wheel; nil
	// keeps them.
	PointerAxisModifiers []services.LogicalKeyboardKey
	// Physics replaces the physics; nil keeps them.
	Physics ScrollPhysics
	// Platform replaces the platform; the Unspecified value keeps it.
	Platform foundation.TargetPlatform
	// KeyboardDismissBehavior replaces the behavior; the Unspecified value keeps
	// it.
	KeyboardDismissBehavior ScrollViewKeyboardDismissBehavior
}

// ScrollBehaviorBase implements ScrollBehavior with Flutter's defaults for the
// platform. Embed it and call InitScrollBehaviorBase with the outer value so
// that the methods call the outer type's overrides.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior
type ScrollBehaviorBase struct {
	self ScrollBehavior
}

// InitScrollBehaviorBase sets the outer value.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.new
func InitScrollBehaviorBase(b *ScrollBehaviorBase, self ScrollBehavior) { b.self = self }

// plainScrollBehavior is the ScrollBehavior class itself.
type plainScrollBehavior struct{ ScrollBehaviorBase }

// NewScrollBehavior creates the default behavior.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.new
func NewScrollBehavior() ScrollBehavior {
	b := &plainScrollBehavior{}
	InitScrollBehaviorBase(&b.ScrollBehaviorBase, b)
	return b
}

func (b *plainScrollBehavior) String() string { return "ScrollBehavior" }

// CopyWith returns a behavior that wraps this one, with the given values
// changed.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.copyWith
func (b *ScrollBehaviorBase) CopyWith(changes ...ScrollBehaviorCopy) ScrollBehavior {
	c := firstOpts(changes)
	return newWrappedScrollBehavior(b.self, wrappedScrollBehaviorOpts{
		scrollbars: c.Scrollbars.OrElse(true), overscroll: c.Overscroll.OrElse(true),
		dragDevices: c.DragDevices, multitouchDragStrategy: c.MultitouchDragStrategy,
		pointerAxisModifiers: c.PointerAxisModifiers,
		physics:              c.Physics, platform: c.Platform, keyboardDismissBehavior: c.KeyboardDismissBehavior,
	})
}

// GetPlatform is foundation.DefaultTargetPlatform.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.getPlatform
func (b *ScrollBehaviorBase) GetPlatform(BuildContext) foundation.TargetPlatform {
	return foundation.DefaultTargetPlatform()
}

// DragDevices are the kinds of pointer other than a mouse.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.dragDevices
func (b *ScrollBehaviorBase) DragDevices() []ui.PointerDeviceKind {
	return slices.Clone(touchLikeDeviceTypes)
}

// PointerAxisModifiers are the two Shift keys.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.pointerAxisModifiers
func (b *ScrollBehaviorBase) PointerAxisModifiers() []services.LogicalKeyboardKey {
	return []services.LogicalKeyboardKey{services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftRight}
}

// GetMultitouchDragStrategy is the average of the boundary pointers on macOS and
// iOS, and the latest pointer elsewhere.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.getMultitouchDragStrategy
func (b *ScrollBehaviorBase) GetMultitouchDragStrategy(bc BuildContext) gestures.MultitouchDragStrategy {
	switch b.self.GetPlatform(bc) {
	case foundation.TargetPlatformMacOS, foundation.TargetPlatformIOS:
		return gestures.MultitouchDragStrategyAverageBoundaryPointers
	}
	return gestures.MultitouchDragStrategyLatestPointer
}

// BuildScrollbar is a RawScrollbar on Linux, macOS and Windows, and the child
// on the other platforms.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.buildScrollbar
func (b *ScrollBehaviorBase) BuildScrollbar(bc BuildContext, child Widget, details ScrollableDetails) Widget {
	switch b.self.GetPlatform(bc) {
	case foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows:
		if foundation.DebugMode && isNil(details.Controller) {
			panic(scrollbarControllerMissingError())
		}
		return NewRawScrollbar(child, RawScrollbarOpts{Controller: details.Controller})
	}
	return child
}

// BuildOverscrollIndicator is the child.
//
// Deviation: Flutter builds a GlowingOverscrollIndicator on Android and
// Fuchsia; the Android indicators are not ported (SPEC §12.10).
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.buildOverscrollIndicator
func (b *ScrollBehaviorBase) BuildOverscrollIndicator(_ BuildContext, child Widget, _ ScrollableDetails) Widget {
	return child
}

// VelocityTrackerBuilder makes the iOS tracker on iOS, the macOS tracker on
// macOS and the plain one elsewhere.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.velocityTrackerBuilder
func (b *ScrollBehaviorBase) VelocityTrackerBuilder(bc BuildContext) gestures.GestureVelocityTrackerBuilder {
	var clock gestures.SamplingClock
	if binding := MaybeWidgetsBindingOf(bc); binding != nil {
		clock = binding.Gestures().Clock()
	}
	switch b.self.GetPlatform(bc) {
	case foundation.TargetPlatformIOS:
		return func(event gestures.PointerEvent) *gestures.VelocityTracker {
			return gestures.NewIOSScrollViewFlingVelocityTracker(clock, event.Kind())
		}
	case foundation.TargetPlatformMacOS:
		return func(event gestures.PointerEvent) *gestures.VelocityTracker {
			return gestures.NewMacOSScrollViewFlingVelocityTracker(clock, event.Kind())
		}
	}
	return func(event gestures.PointerEvent) *gestures.VelocityTracker {
		return gestures.NewVelocityTrackerWithKind(clock, event.Kind())
	}
}

var (
	bouncingPhysics        ScrollPhysics = NewBouncingScrollPhysics(BouncingScrollPhysicsOpts{Parent: NewRangeMaintainingScrollPhysics(nil)})
	bouncingDesktopPhysics ScrollPhysics = NewBouncingScrollPhysics(BouncingScrollPhysicsOpts{
		DecelerationRate: ScrollDecelerationRateFast, Parent: NewRangeMaintainingScrollPhysics(nil)})
	clampingPhysics ScrollPhysics = NewClampingScrollPhysics(NewRangeMaintainingScrollPhysics(nil))
)

// GetScrollPhysics are bouncing physics on iOS, fast bouncing physics on macOS
// and clamping physics elsewhere, each over RangeMaintainingScrollPhysics.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.getScrollPhysics
func (b *ScrollBehaviorBase) GetScrollPhysics(bc BuildContext) ScrollPhysics {
	switch b.self.GetPlatform(bc) {
	case foundation.TargetPlatformIOS:
		return bouncingPhysics
	case foundation.TargetPlatformMacOS:
		return bouncingDesktopPhysics
	}
	return clampingPhysics
}

// ShouldNotify is false.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.shouldNotify
func (b *ScrollBehaviorBase) ShouldNotify(ScrollBehavior) bool { return false }

// GetKeyboardDismissBehavior is manual.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.getKeyboardDismissBehavior
func (b *ScrollBehaviorBase) GetKeyboardDismissBehavior(BuildContext) ScrollViewKeyboardDismissBehavior {
	return ScrollViewKeyboardDismissBehaviorManual
}

// String is the name of the type of the behavior.
//
// Flutter: widgets/scroll_configuration.dart ScrollBehavior.toString
func (b *ScrollBehaviorBase) String() string { return foundation.RuntimeTypeName(b.self) }

type wrappedScrollBehaviorOpts struct {
	scrollbars, overscroll  bool
	dragDevices             []ui.PointerDeviceKind
	multitouchDragStrategy  gestures.MultitouchDragStrategy
	pointerAxisModifiers    []services.LogicalKeyboardKey
	physics                 ScrollPhysics
	platform                foundation.TargetPlatform
	keyboardDismissBehavior ScrollViewKeyboardDismissBehavior
}

// wrappedScrollBehavior is the behavior ScrollBehavior.copyWith returns: it
// replaces some of the values of the one it wraps.
//
// Flutter: widgets/scroll_configuration.dart _WrappedScrollBehavior
type wrappedScrollBehavior struct {
	delegate ScrollBehavior
	wrappedScrollBehaviorOpts
}

func newWrappedScrollBehavior(delegate ScrollBehavior, o wrappedScrollBehaviorOpts) *wrappedScrollBehavior {
	return &wrappedScrollBehavior{delegate, o}
}

func (w *wrappedScrollBehavior) DragDevices() []ui.PointerDeviceKind {
	if w.dragDevices != nil {
		return w.dragDevices
	}
	return w.delegate.DragDevices()
}

func (w *wrappedScrollBehavior) PointerAxisModifiers() []services.LogicalKeyboardKey {
	if w.pointerAxisModifiers != nil {
		return w.pointerAxisModifiers
	}
	return w.delegate.PointerAxisModifiers()
}

func (w *wrappedScrollBehavior) GetMultitouchDragStrategy(bc BuildContext) gestures.MultitouchDragStrategy {
	if w.multitouchDragStrategy != gestures.MultitouchDragStrategyUnspecified {
		return w.multitouchDragStrategy
	}
	return w.delegate.GetMultitouchDragStrategy(bc)
}

func (w *wrappedScrollBehavior) BuildOverscrollIndicator(bc BuildContext, child Widget, details ScrollableDetails) Widget {
	if w.overscroll {
		return w.delegate.BuildOverscrollIndicator(bc, child, details)
	}
	return child
}

func (w *wrappedScrollBehavior) BuildScrollbar(bc BuildContext, child Widget, details ScrollableDetails) Widget {
	if w.scrollbars {
		return w.delegate.BuildScrollbar(bc, child, details)
	}
	return child
}

func (w *wrappedScrollBehavior) CopyWith(changes ...ScrollBehaviorCopy) ScrollBehavior {
	c := firstOpts(changes)
	dragDevices := c.DragDevices
	if dragDevices == nil {
		dragDevices = w.DragDevices()
	}
	multitouch := c.MultitouchDragStrategy
	if multitouch == gestures.MultitouchDragStrategyUnspecified {
		multitouch = w.multitouchDragStrategy
	}
	modifiers := c.PointerAxisModifiers
	if modifiers == nil {
		modifiers = w.PointerAxisModifiers()
	}
	physics := c.Physics
	if isNil(physics) {
		physics = w.physics
	}
	platform := c.Platform
	if platform == foundation.TargetPlatformUnspecified {
		platform = w.platform
	}
	dismiss := c.KeyboardDismissBehavior
	if dismiss == ScrollViewKeyboardDismissBehaviorUnspecified {
		dismiss = w.keyboardDismissBehavior
	}
	return w.delegate.CopyWith(ScrollBehaviorCopy{
		Scrollbars: foundation.Some(c.Scrollbars.OrElse(w.scrollbars)), Overscroll: foundation.Some(c.Overscroll.OrElse(w.overscroll)),
		DragDevices: dragDevices, MultitouchDragStrategy: multitouch, PointerAxisModifiers: modifiers, Physics: physics, Platform: platform, KeyboardDismissBehavior: dismiss,
	})
}

func (w *wrappedScrollBehavior) GetPlatform(bc BuildContext) foundation.TargetPlatform {
	if w.platform != foundation.TargetPlatformUnspecified {
		return w.platform
	}
	return w.delegate.GetPlatform(bc)
}

func (w *wrappedScrollBehavior) GetScrollPhysics(bc BuildContext) ScrollPhysics {
	if !isNil(w.physics) {
		return w.physics
	}
	return w.delegate.GetScrollPhysics(bc)
}

func (w *wrappedScrollBehavior) GetKeyboardDismissBehavior(bc BuildContext) ScrollViewKeyboardDismissBehavior {
	if w.keyboardDismissBehavior != ScrollViewKeyboardDismissBehaviorUnspecified {
		return w.keyboardDismissBehavior
	}
	return w.delegate.GetKeyboardDismissBehavior(bc)
}

func (w *wrappedScrollBehavior) ShouldNotify(oldDelegate ScrollBehavior) bool {
	old, ok := oldDelegate.(*wrappedScrollBehavior)
	if !ok {
		return true
	}
	return reflect.TypeOf(old.delegate) != reflect.TypeOf(w.delegate) ||
		old.scrollbars != w.scrollbars ||
		old.overscroll != w.overscroll ||
		!sameKinds(old.DragDevices(), w.DragDevices()) ||
		old.multitouchDragStrategy != w.multitouchDragStrategy ||
		!sameKeys(old.PointerAxisModifiers(), w.PointerAxisModifiers()) ||
		old.physics != w.physics ||
		old.platform != w.platform ||
		w.delegate.ShouldNotify(old.delegate)
}

func sameKinds(a, b []ui.PointerDeviceKind) bool {
	if len(a) != len(b) {
		return false
	}
	for _, k := range a {
		if !slices.Contains(b, k) {
			return false
		}
	}
	return true
}

func sameKeys(a, b []services.LogicalKeyboardKey) bool {
	if len(a) != len(b) {
		return false
	}
	for _, k := range a {
		if !slices.Contains(b, k) {
			return false
		}
	}
	return true
}

func (w *wrappedScrollBehavior) VelocityTrackerBuilder(bc BuildContext) gestures.GestureVelocityTrackerBuilder {
	return w.delegate.VelocityTrackerBuilder(bc)
}

func (w *wrappedScrollBehavior) String() string { return "_WrappedScrollBehavior" }

// ScrollConfiguration controls how Scrollable widgets behave in a subtree: it
// gives them a ScrollBehavior.
//
// Flutter: widgets/scroll_configuration.dart ScrollConfiguration
type ScrollConfiguration struct {
	InheritedWidget
	behavior ScrollBehavior
}

// NewScrollConfiguration creates the widget.
//
// Flutter: widgets/scroll_configuration.dart ScrollConfiguration.new
func NewScrollConfiguration(behavior ScrollBehavior, child Widget, opts ...WidgetOpts) *ScrollConfiguration {
	return &ScrollConfiguration{NewInheritedWidgetBase(child, opts...), behavior}
}

// Behavior is the scroll behavior for the subtree.
//
// Flutter: widgets/scroll_configuration.dart ScrollConfiguration.behavior
func (w *ScrollConfiguration) Behavior() ScrollBehavior { return w.behavior }

var defaultScrollBehavior = NewScrollBehavior()

// ScrollConfigurationOf is the behavior of the closest ScrollConfiguration above
// the context, or the default one. The context depends on it.
//
// Flutter: widgets/scroll_configuration.dart ScrollConfiguration.of
func ScrollConfigurationOf(bc BuildContext) ScrollBehavior {
	if w, ok := DependOn[*ScrollConfiguration](bc); ok {
		return w.behavior
	}
	return defaultScrollBehavior
}

// UpdateShouldNotify is whether the behavior is of another type, or notifies.
//
// Flutter: widgets/scroll_configuration.dart ScrollConfiguration.updateShouldNotify
func (w *ScrollConfiguration) UpdateShouldNotify(old Widget) bool {
	o := old.(*ScrollConfiguration)
	return reflect.TypeOf(w.behavior) != reflect.TypeOf(o.behavior) ||
		(w.behavior != o.behavior && w.behavior.ShouldNotify(o.behavior))
}
