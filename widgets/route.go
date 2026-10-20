// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"fmt"
	"sync"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/scheduler"
)

// RoutePopDisposition describes how a route handles a request to pop it.
//
// Flutter: widgets/navigator.dart RoutePopDisposition
type RoutePopDisposition int

const (
	// RoutePopDispositionPop means the route should be popped.
	//
	// Flutter: widgets/navigator.dart RoutePopDisposition.pop
	RoutePopDispositionPop RoutePopDisposition = iota
	// RoutePopDispositionDoNotPop means the route must not be popped.
	//
	// Flutter: widgets/navigator.dart RoutePopDisposition.doNotPop
	RoutePopDispositionDoNotPop
	// RoutePopDispositionBubble means the route does not decide, so the request
	// goes to the parent: for instance, the navigator of an enclosing app.
	//
	// Flutter: widgets/navigator.dart RoutePopDisposition.bubble
	RoutePopDispositionBubble
)

// String names the disposition.
func (d RoutePopDisposition) String() string {
	switch d {
	case RoutePopDispositionPop:
		return "RoutePopDisposition.pop"
	case RoutePopDispositionDoNotPop:
		return "RoutePopDisposition.doNotPop"
	case RoutePopDispositionBubble:
		return "RoutePopDisposition.bubble"
	}
	return fmt.Sprintf("RoutePopDisposition(%d)", int(d))
}

// RouteFactory creates a route for the given settings.
//
// Flutter: widgets/navigator.dart RouteFactory
type RouteFactory func(settings RouteSettings) Route

// RouteListFactory creates the list of routes that the navigator starts with.
//
// Flutter: widgets/navigator.dart RouteListFactory
type RouteListFactory func(navigator *NavigatorState, initialRoute string) []Route

// PopPageCallback is called when a page-based route is popped; it returns whether
// the pop succeeded, and a pop that did not succeed leaves the route in place. It
// is the deprecated way of handling pops, replaced by DidRemovePageCallback.
//
// Flutter: widgets/navigator.dart PopPageCallback
type PopPageCallback func(route Route, result any) bool

// DidRemovePageCallback is called after the route of a page was removed from the
// navigator by an imperative call such as Pop, so that the page can be removed
// from Navigator.pages.
//
// Flutter: widgets/navigator.dart DidRemovePageCallback
type DidRemovePageCallback func(page Page)

// RoutePredicate decides for a route; it is used to say where to stop popping or
// removing routes.
//
// Flutter: widgets/navigator.dart RoutePredicate
type RoutePredicate func(route Route) bool

// PopInvokedWithResultCallback is called after a route pop was handled, or
// blocked. DidPop is whether the route was popped, and result is the result of
// the pop (nil if blocked).
//
// Flutter: widgets/routes.dart PopInvokedWithResultCallback
type PopInvokedWithResultCallback func(didPop bool, result any)

// RouteSettings describes a route: its name and the arguments it was pushed
// with. Page implements it as well.
//
// Flutter: widgets/navigator.dart RouteSettings
type RouteSettings interface {
	// Name is the name of the route, for instance "/settings"; empty if none.
	//
	// Flutter: widgets/navigator.dart RouteSettings.name
	Name() string
	// Arguments is the arguments the route was pushed with, or nil.
	//
	// Flutter: widgets/navigator.dart RouteSettings.arguments
	Arguments() any
}

// RouteSettingsOpts are the optional parameters of NewRouteSettings.
//
// Flutter: widgets/navigator.dart RouteSettings.new
type RouteSettingsOpts struct {
	// Name is the name of the route; empty means none.
	Name string
	// Arguments are passed to the route.
	Arguments any
}

// RouteSettingsBase is the implementation of RouteSettings that Page
// implementations embed, and that NewRouteSettings returns.
//
// Flutter: widgets/navigator.dart RouteSettings
type RouteSettingsBase struct {
	name      string
	arguments any
}

// NewRouteSettings creates route settings.
//
// Flutter: widgets/navigator.dart RouteSettings.new
func NewRouteSettings(opts ...RouteSettingsOpts) *RouteSettingsBase {
	o := firstOpts(opts)
	return &RouteSettingsBase{o.Name, o.Arguments}
}

// Name is the name of the route, empty if none.
//
// Flutter: widgets/navigator.dart RouteSettings.name
func (s *RouteSettingsBase) Name() string { return s.name }

// Arguments are the arguments of the route, or nil.
//
// Flutter: widgets/navigator.dart RouteSettings.arguments
func (s *RouteSettingsBase) Arguments() any { return s.arguments }

// String describes the settings: the name in quotes ("none" if empty) and the
// arguments.
//
// Flutter: widgets/navigator.dart RouteSettings.toString
func (s *RouteSettingsBase) String() string {
	return describeRouteSettings("RouteSettings", s.name, s.arguments)
}

func describeRouteSettings(typeName, name string, arguments any) string {
	n := "none"
	if name != "" {
		n = `"` + name + `"`
	}
	if arguments == nil {
		return fmt.Sprintf("%s(%s, null)", typeName, n)
	}
	return fmt.Sprintf("%s(%s, %v)", typeName, n, arguments)
}

// Page is an immutable description of a route: the Navigator creates a Route
// from it and updates the route when the page is replaced by one that CanUpdate.
// A route created from a page has the page as its settings.
//
// Flutter: widgets/navigator.dart Page
type Page interface {
	RouteSettings
	// Key is the key of the page; pages with equal keys are the same page.
	//
	// Flutter: widgets/navigator.dart Page.key
	Key() foundation.Key
	// RestorationId is the id for state restoration of the page, or empty.
	//
	// Flutter: widgets/navigator.dart Page.restorationId
	RestorationId() string
	// CanPop is whether the route created from the page can be popped.
	//
	// Flutter: widgets/navigator.dart Page.canPop
	CanPop() bool
	// OnPopInvoked is called after the pop of the route was handled or blocked.
	//
	// Flutter: widgets/navigator.dart Page.onPopInvoked
	OnPopInvoked(didPop bool, result any)
	// CanUpdate is whether the route of the other page can be updated to this
	// page: the pages have the same type and key.
	//
	// Flutter: widgets/navigator.dart Page.canUpdate
	CanUpdate(other Page) bool
	// CreateRoute creates the route of the page; its settings must be the page.
	//
	// Flutter: widgets/navigator.dart Page.createRoute
	CreateRoute(bc BuildContext) Route
}

// PageOpts are the optional parameters of a page, for pages that embed PageBase.
//
// Flutter: widgets/navigator.dart Page.new
type PageOpts struct {
	// Key identifies the page.
	Key foundation.Key
	// Name of the route.
	Name string
	// Arguments of the route.
	Arguments any
	// RestorationId is the id for state restoration.
	RestorationId string
	// CanPop is whether the route can be popped; the default is true.
	CanPop foundation.Opt[bool]
	// OnPopInvoked is called after the pop was handled or blocked.
	OnPopInvoked PopInvokedWithResultCallback
}

// PageBase is the base of the Page implementations: embed it, call InitPageBase
// and implement CreateRoute.
//
// Flutter: widgets/navigator.dart Page
type PageBase struct {
	RouteSettingsBase
	self          Page
	key           foundation.Key
	restorationID string
	canPop        bool
	onPopInvoked  PopInvokedWithResultCallback
}

// InitPageBase sets up the embedded base of a page whose outer value is self.
//
// Flutter: widgets/navigator.dart Page.new
func InitPageBase(b *PageBase, self Page, opts ...PageOpts) {
	o := firstOpts(opts)
	*b = PageBase{
		RouteSettingsBase: RouteSettingsBase{o.Name, o.Arguments},
		self:              self,
		key:               o.Key,
		restorationID:     o.RestorationId,
		canPop:            o.CanPop.OrElse(true),
		onPopInvoked:      o.OnPopInvoked,
	}
}

// Key is the key of the page, or nil.
//
// Flutter: widgets/navigator.dart Page.key
func (p *PageBase) Key() foundation.Key { return p.key }

// RestorationId is the id for state restoration, or empty.
//
// Flutter: widgets/navigator.dart Page.restorationId
func (p *PageBase) RestorationId() string { return p.restorationID }

// CanPop is whether the route created from the page can be popped.
//
// Flutter: widgets/navigator.dart Page.canPop
func (p *PageBase) CanPop() bool { return p.canPop }

// OnPopInvoked calls the callback of the page, if any.
//
// Flutter: widgets/navigator.dart Page.onPopInvoked
func (p *PageBase) OnPopInvoked(didPop bool, result any) {
	if p.onPopInvoked != nil {
		p.onPopInvoked(didPop, result)
	}
}

// CanUpdate is whether the other page has the same type and key.
//
// Flutter: widgets/navigator.dart Page.canUpdate
func (p *PageBase) CanUpdate(other Page) bool {
	return foundation.RuntimeTypeName(other) == foundation.RuntimeTypeName(p.self) && other.Key() == p.key
}

// completer is a one-shot value that any number of goroutines can wait for: the
// Completer of dart:async. Each Chan call returns a channel that gets the value
// and is closed; callbacks registered with Then run when the value is set (at
// once if it already is).
//
// Flutter: no counterpart (dart:async Completer)
type completer struct {
	mu        sync.Mutex
	done      bool
	value     any
	waiters   []chan any
	callbacks []func(any)
}

func (c *completer) IsCompleted() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.done
}

func (c *completer) Complete(value any) {
	c.mu.Lock()
	if c.done {
		c.mu.Unlock()
		return
	}
	c.done, c.value = true, value
	waiters, callbacks := c.waiters, c.callbacks
	c.waiters, c.callbacks = nil, nil
	c.mu.Unlock()
	for _, w := range waiters {
		w <- value
		close(w)
	}
	for _, cb := range callbacks {
		cb(value)
	}
}

// Chan returns a channel that gets the value, then is closed.
func (c *completer) Chan() <-chan any {
	ch := make(chan any, 1)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.done {
		ch <- c.value
		close(ch)
		return ch
	}
	c.waiters = append(c.waiters, ch)
	return ch
}

// Then calls fn with the value once it is set.
func (c *completer) Then(fn func(any)) {
	c.mu.Lock()
	if !c.done {
		c.callbacks = append(c.callbacks, fn)
		c.mu.Unlock()
		return
	}
	v := c.value
	c.mu.Unlock()
	fn(v)
}

// NavigatorObserver watches the changes of a Navigator: a route pushed, popped,
// removed or replaced, the top route changed, and a user gesture started or
// stopped. Embed NavigatorObserverBase and override the methods of interest.
//
// Flutter: widgets/navigator.dart NavigatorObserver
type NavigatorObserver interface {
	// Navigator is the navigator the observer is observing, or nil.
	//
	// Flutter: widgets/navigator.dart NavigatorObserver.navigator
	Navigator() *NavigatorState
	setNavigator(navigator *NavigatorState)
	// DidPush is called when a route was pushed onto the navigator.
	//
	// Flutter: widgets/navigator.dart NavigatorObserver.didPush
	DidPush(route, previousRoute Route)
	// DidPop is called when a route was popped off the navigator.
	//
	// Flutter: widgets/navigator.dart NavigatorObserver.didPop
	DidPop(route, previousRoute Route)
	// DidRemove is called when a route was removed from the navigator.
	//
	// Flutter: widgets/navigator.dart NavigatorObserver.didRemove
	DidRemove(route, previousRoute Route)
	// DidReplace is called when a route was replaced in the navigator.
	//
	// Flutter: widgets/navigator.dart NavigatorObserver.didReplace
	DidReplace(newRoute, oldRoute Route)
	// DidChangeTop is called when the top route of the navigator changed.
	//
	// Flutter: widgets/navigator.dart NavigatorObserver.didChangeTop
	DidChangeTop(topRoute, previousTopRoute Route)
	// DidStartUserGesture is called when the user starts a gesture that may pop
	// the route, for instance a swipe to go back.
	//
	// Flutter: widgets/navigator.dart NavigatorObserver.didStartUserGesture
	DidStartUserGesture(route, previousRoute Route)
	// DidStopUserGesture is called when the user stopped the gesture.
	//
	// Flutter: widgets/navigator.dart NavigatorObserver.didStopUserGesture
	DidStopUserGesture()
}

// NavigatorObserverBase implements NavigatorObserver with methods that do
// nothing. Embed it.
//
// Flutter: widgets/navigator.dart NavigatorObserver
type NavigatorObserverBase struct{ navigator *NavigatorState }

// Navigator is the navigator that the observer observes, or nil.
//
// Flutter: widgets/navigator.dart NavigatorObserver.navigator
func (o *NavigatorObserverBase) Navigator() *NavigatorState { return o.navigator }

func (o *NavigatorObserverBase) setNavigator(n *NavigatorState) { o.navigator = n }

// DidPush does nothing.
//
// Flutter: widgets/navigator.dart NavigatorObserver.didPush
func (o *NavigatorObserverBase) DidPush(route, previousRoute Route) {}

// DidPop does nothing.
//
// Flutter: widgets/navigator.dart NavigatorObserver.didPop
func (o *NavigatorObserverBase) DidPop(route, previousRoute Route) {}

// DidRemove does nothing.
//
// Flutter: widgets/navigator.dart NavigatorObserver.didRemove
func (o *NavigatorObserverBase) DidRemove(route, previousRoute Route) {}

// DidReplace does nothing.
//
// Flutter: widgets/navigator.dart NavigatorObserver.didReplace
func (o *NavigatorObserverBase) DidReplace(newRoute, oldRoute Route) {}

// DidChangeTop does nothing.
//
// Flutter: widgets/navigator.dart NavigatorObserver.didChangeTop
func (o *NavigatorObserverBase) DidChangeTop(topRoute, previousTopRoute Route) {}

// DidStartUserGesture does nothing.
//
// Flutter: widgets/navigator.dart NavigatorObserver.didStartUserGesture
func (o *NavigatorObserverBase) DidStartUserGesture(route, previousRoute Route) {}

// DidStopUserGesture does nothing.
//
// Flutter: widgets/navigator.dart NavigatorObserver.didStopUserGesture
func (o *NavigatorObserverBase) DidStopUserGesture() {}

// NavigationNotification tells the ancestors whether the navigator or a pop
// scope below can handle a pop.
//
// Flutter: widgets/navigator.dart NavigationNotification
type NavigationNotification struct {
	// CanHandlePop is whether the originator can handle a pop: a PopScope that
	// blocks pops, or a navigator that has routes to pop.
	//
	// Flutter: widgets/navigator.dart NavigationNotification.canHandlePop
	CanHandlePop bool
}

// String describes the notification.
//
// Flutter: widgets/navigator.dart NavigationNotification.toString
func (n NavigationNotification) String() string {
	return fmt.Sprintf("NavigationNotification canHandlePop: %v", n.CanHandlePop)
}

// Route is an entry in the history of a Navigator: a screen or a dialog. The
// result of a route is passed to the channel that Push returned, and the
// results are any values.
//
// Implement a route by embedding RouteBase (InitRouteBase), or one of the
// route bases of this package (OverlayRouteBase, TransitionRouteBase,
// ModalRouteBase, PageRouteBase, PopupRouteBase).
//
// Flutter: widgets/navigator.dart Route
type Route interface {
	// AsRoute is the base of the route.
	AsRoute() *RouteBase
	// Navigator is the navigator the route is installed in, or nil.
	//
	// Flutter: widgets/navigator.dart Route.navigator
	Navigator() *NavigatorState
	// Settings are the settings of the route.
	//
	// Flutter: widgets/navigator.dart Route.settings
	Settings() RouteSettings
	// RequestFocus is whether the route requests the focus when it becomes the
	// top one.
	//
	// Flutter: widgets/navigator.dart Route.requestFocus
	RequestFocus() bool
	// OverlayEntries are the entries the route puts in the overlay of the
	// navigator.
	//
	// Flutter: widgets/navigator.dart Route.overlayEntries
	OverlayEntries() []*OverlayEntry
	// Install is called when the route is inserted into the navigator.
	//
	// Flutter: widgets/navigator.dart Route.install
	Install()
	// DidPush is called after Install when the route is pushed; the future
	// completes when the transition in finished.
	//
	// Flutter: widgets/navigator.dart Route.didPush
	DidPush() *scheduler.TickerFuture
	// DidAdd is called after Install when the route was added to the navigator
	// without a transition.
	//
	// Flutter: widgets/navigator.dart Route.didAdd
	DidAdd()
	// DidReplace is called after Install when the route replaces another one.
	//
	// Flutter: widgets/navigator.dart Route.didReplace
	DidReplace(oldRoute Route)
	// PopDisposition says how the route handles a request to pop it.
	//
	// Flutter: widgets/navigator.dart Route.popDisposition
	PopDisposition() RoutePopDisposition
	// OnPopInvokedWithResult is called after a pop of the route was handled
	// (didPop true) or blocked (false).
	//
	// Flutter: widgets/navigator.dart Route.onPopInvokedWithResult
	OnPopInvokedWithResult(didPop bool, result any)
	// WillHandlePopInternally is whether a pop is handled inside the route
	// (local history entries) instead of removing it.
	//
	// Flutter: widgets/navigator.dart Route.willHandlePopInternally
	WillHandlePopInternally() bool
	// CurrentResult is the value that the route completes with when it is popped
	// without a result.
	//
	// Flutter: widgets/navigator.dart Route.currentResult
	CurrentResult() any
	// Popped returns a channel that gets the result of the route when it is
	// popped, then is closed. Each call returns a new channel.
	//
	// Flutter: widgets/navigator.dart Route.popped
	Popped() <-chan any
	// DidPop is called when the route is popped; it returns false if the route
	// handled the pop itself (a local history entry was removed).
	//
	// Flutter: widgets/navigator.dart Route.didPop
	DidPop(result any) bool
	// DidComplete is called when the route is popped or removed to complete the
	// future of Popped.
	//
	// Flutter: widgets/navigator.dart Route.didComplete
	DidComplete(result any)
	// DidPopNext is called when the route above this one was popped, and this one
	// is the top route now.
	//
	// Flutter: widgets/navigator.dart Route.didPopNext
	DidPopNext(nextRoute Route)
	// DidChangeNext is called when the route above this one changed.
	//
	// Flutter: widgets/navigator.dart Route.didChangeNext
	DidChangeNext(nextRoute Route)
	// DidChangePrevious is called when the route below this one changed.
	//
	// Flutter: widgets/navigator.dart Route.didChangePrevious
	DidChangePrevious(previousRoute Route)
	// ChangedInternalState is called when the state of the route changed.
	//
	// Flutter: widgets/navigator.dart Route.changedInternalState
	ChangedInternalState()
	// ChangedExternalState is called when the navigator's dependencies changed.
	//
	// Flutter: widgets/navigator.dart Route.changedExternalState
	ChangedExternalState()
	// Dispose is called when the route is no longer needed.
	//
	// Flutter: widgets/navigator.dart Route.dispose
	Dispose()
	// IsCurrent is whether the route is the top one of the navigator.
	//
	// Flutter: widgets/navigator.dart Route.isCurrent
	IsCurrent() bool
	// IsFirst is whether the route is the bottom one of the navigator.
	//
	// Flutter: widgets/navigator.dart Route.isFirst
	IsFirst() bool
	// IsActive is whether the route is on the navigator and not popped or
	// removed.
	//
	// Flutter: widgets/navigator.dart Route.isActive
	IsActive() bool
	// HasActiveRouteBelow is whether there is an active route below this one.
	//
	// Flutter: widgets/navigator.dart Route.hasActiveRouteBelow
	HasActiveRouteBelow() bool
	// String describes the route.
	String() string
}

// RouteBase implements the parts of Route that every route shares. Embed it
// (or a base that embeds it) and call InitRouteBase with the outer value.
//
// Flutter: widgets/navigator.dart Route
type RouteBase struct {
	self             Route
	settings         RouteSettings
	requestFocus     foundation.Opt[bool]
	navigator        *NavigatorState
	popCompleter     completer
	disposeCompleter completer
}

// RouteOpts are the optional parameters of the route bases.
//
// Flutter: widgets/navigator.dart Route.new
type RouteOpts struct {
	// Settings are the settings of the route; the default has no name and no
	// arguments.
	Settings RouteSettings
	// RequestFocus is whether the route requests the focus when it becomes the
	// top one; the default is the RequestFocus of the navigator.
	RequestFocus foundation.Opt[bool]
}

// InitRouteBase sets the outer value of the route.
//
// Flutter: widgets/navigator.dart Route.new
func InitRouteBase(b *RouteBase, self Route, opts RouteOpts) {
	b.self = self
	b.settings = opts.Settings
	if b.settings == nil {
		b.settings = &RouteSettingsBase{}
	}
	b.requestFocus = opts.RequestFocus
}

// AsRoute is the base of the route.
func (b *RouteBase) AsRoute() *RouteBase { return b }

// RequestFocus is whether the route requests focus when it becomes the top one.
// It defaults to the RequestFocus of the navigator, or false without one.
//
// Flutter: widgets/navigator.dart Route.requestFocus
func (b *RouteBase) RequestFocus() bool {
	if v, ok := b.requestFocus.Get(); ok {
		return v
	}
	if b.navigator != nil {
		return b.navigator.Widget().RequestFocus()
	}
	return false
}

// Navigator is the navigator the route is installed in, or nil.
//
// Flutter: widgets/navigator.dart Route.navigator
func (b *RouteBase) Navigator() *NavigatorState { return b.navigator }

func (b *RouteBase) installed() bool { return b.navigator != nil }

func (b *RouteBase) isInstalledIn(state *NavigatorState) bool { return b.navigator == state }

// Settings are the settings of the route.
//
// Flutter: widgets/navigator.dart Route.settings
func (b *RouteBase) Settings() RouteSettings { return b.settings }

func (b *RouteBase) isPageBased() bool {
	_, ok := b.settings.(Page)
	return ok
}

// updateSettings replaces the settings of the route: the page of a page-based
// route changed. An installed route is told that its internal state changed.
//
// Flutter: widgets/navigator.dart Route._updateSettings
func (b *RouteBase) updateSettings(newSettings RouteSettings) {
	if b.settings != newSettings {
		b.settings = newSettings
		if b.installed() {
			b.self.ChangedInternalState()
		}
	}
}

// OverlayEntries is empty; overlay routes have entries.
//
// Flutter: widgets/navigator.dart Route.overlayEntries
func (b *RouteBase) OverlayEntries() []*OverlayEntry { return nil }

// Install does nothing.
//
// Flutter: widgets/navigator.dart Route.install
func (b *RouteBase) Install() {}

// DidPush returns a future that is complete, and asks for focus on the
// navigator when it requests focus.
//
// Flutter: widgets/navigator.dart Route.didPush
func (b *RouteBase) DidPush() *scheduler.TickerFuture {
	f := scheduler.NewTickerFutureComplete()
	if b.RequestFocus() {
		b.requestNavigatorFocus()
	}
	return f
}

// requestNavigatorFocus requests the focus for the scope that encloses the focus
// node of the navigator, once the pending work of the frame is done (Dart runs the
// callback of a future in a microtask). The navigator can be gone by then, if the
// route was disposed in the same frame.
func (b *RouteBase) requestNavigatorFocus() {
	navigator := b.navigator
	if navigator == nil {
		return
	}
	request := func() {
		if b.navigator == nil {
			return
		}
		if scope := b.navigator.FocusNode().EnclosingScope(); scope != nil {
			scope.RequestFocus()
		}
	}
	if binding := MaybeWidgetsBindingOf(navigator.Context()); binding != nil {
		binding.Post(request)
		return
	}
	request()
}

// DidAdd asks for focus on the navigator when the route requests focus.
//
// Flutter: widgets/navigator.dart Route.didAdd
func (b *RouteBase) DidAdd() {
	if b.RequestFocus() {
		b.requestNavigatorFocus()
	}
}

// DidReplace does nothing.
//
// Flutter: widgets/navigator.dart Route.didReplace
func (b *RouteBase) DidReplace(oldRoute Route) {}

// PopDisposition is how the route handles a pop: a route of a page with
// canPop false is not popped, the first route bubbles up, and every other
// route is popped.
//
// Flutter: widgets/navigator.dart Route.popDisposition
func (b *RouteBase) PopDisposition() RoutePopDisposition {
	if page, ok := b.settings.(Page); ok && !page.CanPop() {
		return RoutePopDispositionDoNotPop
	}
	if b.self.IsFirst() {
		return RoutePopDispositionBubble
	}
	return RoutePopDispositionPop
}

// OnPopInvokedWithResult tells the page, if the route was made from one.
//
// Flutter: widgets/navigator.dart Route.onPopInvokedWithResult
func (b *RouteBase) OnPopInvokedWithResult(didPop bool, result any) {
	if page, ok := b.settings.(Page); ok {
		page.OnPopInvoked(didPop, result)
	}
}

// WillHandlePopInternally is false.
//
// Flutter: widgets/navigator.dart Route.willHandlePopInternally
func (b *RouteBase) WillHandlePopInternally() bool { return false }

// CurrentResult is nil.
//
// Flutter: widgets/navigator.dart Route.currentResult
func (b *RouteBase) CurrentResult() any { return nil }

// Popped returns a channel that gets the result of the route when it is popped
// or removed.
//
// Flutter: widgets/navigator.dart Route.popped
func (b *RouteBase) Popped() <-chan any { return b.popCompleter.Chan() }

// DidPop completes the route and returns true.
//
// Flutter: widgets/navigator.dart Route.didPop
func (b *RouteBase) DidPop(result any) bool {
	b.self.DidComplete(result)
	return true
}

// DidComplete completes the channel of Popped with the result, or with the
// current result when there is none.
//
// Flutter: widgets/navigator.dart Route.didComplete
func (b *RouteBase) DidComplete(result any) {
	if result == nil {
		result = b.self.CurrentResult()
	}
	b.popCompleter.Complete(result)
}

// DidPopNext does nothing.
//
// Flutter: widgets/navigator.dart Route.didPopNext
func (b *RouteBase) DidPopNext(nextRoute Route) {}

// DidChangeNext does nothing.
//
// Flutter: widgets/navigator.dart Route.didChangeNext
func (b *RouteBase) DidChangeNext(nextRoute Route) {}

// DidChangePrevious does nothing.
//
// Flutter: widgets/navigator.dart Route.didChangePrevious
func (b *RouteBase) DidChangePrevious(previousRoute Route) {}

// ChangedInternalState does nothing.
//
// Flutter: widgets/navigator.dart Route.changedInternalState
func (b *RouteBase) ChangedInternalState() {}

// ChangedExternalState does nothing.
//
// Flutter: widgets/navigator.dart Route.changedExternalState
func (b *RouteBase) ChangedExternalState() {}

// Dispose lets go of the navigator and completes the dispose future.
//
// Flutter: widgets/navigator.dart Route.dispose
func (b *RouteBase) Dispose() {
	b.navigator = nil
	b.disposeCompleter.Complete(nil)
}

// IsCurrent is whether the route is the top present route of the navigator.
//
// Flutter: widgets/navigator.dart Route.isCurrent
func (b *RouteBase) IsCurrent() bool {
	if !b.installed() {
		return false
	}
	current := b.navigator.lastRouteEntryWhere(routeEntryIsPresent)
	if current == nil {
		return false
	}
	return current.route == b.self
}

// IsFirst is whether the route is the first present route of the navigator.
//
// Flutter: widgets/navigator.dart Route.isFirst
func (b *RouteBase) IsFirst() bool {
	if !b.installed() {
		return false
	}
	first := b.navigator.firstRouteEntryWhere(routeEntryIsPresent)
	if first == nil {
		return false
	}
	return first.route == b.self
}

// HasActiveRouteBelow is whether there is a present route below this one.
//
// Flutter: widgets/navigator.dart Route.hasActiveRouteBelow
func (b *RouteBase) HasActiveRouteBelow() bool {
	if !b.installed() {
		return false
	}
	for _, entry := range b.navigator.history.entries {
		if entry.route == b.self {
			return false
		}
		if entry.isPresent() {
			return true
		}
	}
	return false
}

// IsActive is whether the route is in the history and not popped or removed.
//
// Flutter: widgets/navigator.dart Route.isActive
func (b *RouteBase) IsActive() bool {
	if b.navigator == nil {
		return false
	}
	entry := b.navigator.firstRouteEntryWhere(routeEntryIsRoute(b.self))
	return entry != nil && entry.isPresent()
}

// String describes the route.
func (b *RouteBase) String() string {
	return foundation.RuntimeTypeName(b.self) + "(" + b.settings.Name() + ")"
}

func describeContext(bc BuildContext) string {
	if e, ok := bc.(Element); ok {
		return foundation.DescribeIdentity(e)
	}
	return foundation.DescribeIdentity(bc)
}

// binding is the widgets binding of the navigator the route is installed in, or
// nil.
func (b *RouteBase) binding() *WidgetsBinding {
	if b.navigator == nil || !b.navigator.Mounted() {
		return nil
	}
	return MaybeWidgetsBindingOf(b.navigator.Context())
}
