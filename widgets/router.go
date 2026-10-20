package widgets

import (
	"net/url"
	"slices"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// RouteInformation is the unit of the route information that the platform, a
// route information provider and the router pass around: a URI and some state that
// the platform stores in the history with it. Compare the pointers of two values
// with == to know whether they are the same one.
//
// Flutter: widgets/router.dart RouteInformation
type RouteInformation struct {
	uri   *url.URL
	state any
}

// RouteInformationOpts are the parameters of NewRouteInformation.
//
// Flutter: widgets/router.dart RouteInformation.new
type RouteInformationOpts struct {
	// URI is the location; it is required.
	URI *url.URL
	// State is the state that the platform saves with the entry of the history.
	State any
}

// NewRouteInformation creates the route information. The URI is required: in debug
// builds a nil one panics with the tag RouteInformationURIMissing.
//
// Deviation: Flutter's deprecated location parameter, a string, is not ported.
//
// Flutter: widgets/router.dart RouteInformation.new
func NewRouteInformation(options ...RouteInformationOpts) *RouteInformation {
	opts := firstOpts(options)
	if foundation.DebugMode && opts.URI == nil {
		panic(errors.New("RouteInformationURIMissing", "a RouteInformation needs a URI"))
	}
	return &RouteInformation{uri: opts.URI, state: opts.State}
}

// URI is the location of the route information.
//
// Flutter: widgets/router.dart RouteInformation.uri
func (r *RouteInformation) URI() *url.URL { return r.uri }

// State is the state that the platform saves with the entry of the history.
//
// Flutter: widgets/router.dart RouteInformation.state
func (r *RouteInformation) State() any { return r.state }

// RouterConfig is the set of objects that configure a Router: the objects of
// WidgetsApp.router and MaterialApp.router.
//
// Flutter: widgets/router.dart RouterConfig
type RouterConfig[T any] struct {
	// RouteInformationProvider provides the route information of the platform;
	// when it is set the parser must be too.
	RouteInformationProvider RouteInformationProvider
	// RouteInformationParser parses the route information into a configuration.
	RouteInformationParser RouteInformationParser[T]
	// RouterDelegate builds the widgets of the router for the configuration; it is
	// required.
	RouterDelegate RouterDelegate[T]
	// BackButtonDispatcher tells the router that the platform wants to pop.
	BackButtonDispatcher BackButtonDispatcher
}

// RouterOpts are the optional parameters of NewRouter.
//
// Flutter: widgets/router.dart Router.new
type RouterOpts[T any] struct {
	// Key is the key of the widget.
	Key foundation.Key
	// RouteInformationProvider provides the route information of the platform;
	// when it is set RouteInformationParser must be too.
	RouteInformationProvider RouteInformationProvider
	// RouteInformationParser parses the route information into a configuration.
	RouteInformationParser RouteInformationParser[T]
	// BackButtonDispatcher tells the router that the platform wants to pop.
	BackButtonDispatcher BackButtonDispatcher
}

// RouteInformationReportingType is how the router reports its route information
// to the route information provider.
//
// Flutter: widgets/router.dart RouteInformationReportingType
type RouteInformationReportingType int

const (
	// RouteInformationReportingTypeUnspecified is the same as
	// RouteInformationReportingTypeNone.
	RouteInformationReportingTypeUnspecified RouteInformationReportingType = iota
	// RouteInformationReportingTypeNone lets the provider decide whether the
	// report replaces the entry of the history or adds one.
	//
	// Flutter: widgets/router.dart RouteInformationReportingType.none
	RouteInformationReportingTypeNone
	// RouteInformationReportingTypeNeglect makes the provider replace the entry of
	// the history.
	//
	// Flutter: widgets/router.dart RouteInformationReportingType.neglect
	RouteInformationReportingTypeNeglect
	// RouteInformationReportingTypeNavigate makes the provider add an entry to the
	// history.
	//
	// Flutter: widgets/router.dart RouteInformationReportingType.navigate
	RouteInformationReportingTypeNavigate
)

// Router is the widget that reports the route information of the app to the
// platform and builds the widgets of the app from it, through its delegate. It
// keeps a route information provider, a parser, a delegate and a back button
// dispatcher together.
//
// Flutter: widgets/router.dart Router
type Router[T any] struct {
	StatefulWidget
	delegate RouterDelegate[T]
	o        RouterOpts[T]
}

// NewRouter creates a router. In debug builds a route information provider without
// a parser panics with the tag RouterParserMissing.
//
// Flutter: widgets/router.dart Router.new
func NewRouter[T any](routerDelegate RouterDelegate[T], opts ...RouterOpts[T]) *Router[T] {
	o := firstOpts(opts)
	if foundation.DebugMode && o.RouteInformationProvider != nil && o.RouteInformationParser == nil {
		panic(errors.New("RouterParserMissing", "a routeInformationParser must be provided when a routeInformationProvider is specified"))
	}
	return &Router[T]{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), routerDelegate, o}
}

// NewRouterWithConfig creates a router from the configuration.
//
// Flutter: widgets/router.dart Router.withConfig
func NewRouterWithConfig[T any](config RouterConfig[T], key ...foundation.Key) *Router[T] {
	o := RouterOpts[T]{
		RouteInformationProvider: config.RouteInformationProvider,
		RouteInformationParser:   config.RouteInformationParser,
		BackButtonDispatcher:     config.BackButtonDispatcher,
	}
	if len(key) > 0 {
		o.Key = key[0]
	}
	return NewRouter(config.RouterDelegate, o)
}

// RouteInformationProvider is the provider of the router.
//
// Flutter: widgets/router.dart Router.routeInformationProvider
func (w *Router[T]) RouteInformationProvider() RouteInformationProvider {
	return w.o.RouteInformationProvider
}

// RouteInformationParser is the parser of the router.
//
// Flutter: widgets/router.dart Router.routeInformationParser
func (w *Router[T]) RouteInformationParser() RouteInformationParser[T] {
	return w.o.RouteInformationParser
}

// RouterDelegate is the delegate of the router.
//
// Flutter: widgets/router.dart Router.routerDelegate
func (w *Router[T]) RouterDelegate() RouterDelegate[T] { return w.delegate }

// BackButtonDispatcher is the dispatcher of the router.
//
// Flutter: widgets/router.dart Router.backButtonDispatcher
func (w *Router[T]) BackButtonDispatcher() BackButtonDispatcher { return w.o.BackButtonDispatcher }

// CreateState creates the state.
//
// Flutter: widgets/router.dart Router.createState
func (w *Router[T]) CreateState() State { return &routerState[T]{} }

// routerStateAccess is what the scope of a router knows about the state of the
// router, whatever its type.
type routerStateAccess interface {
	routerWidget() Widget
	setStateWithExplicitReportStatus(status RouteInformationReportingType, fn func())
}

func findRouterScope(bc BuildContext) *routerScope {
	scope, _ := DependOn[*routerScope](bc)
	return scope
}

// RouterOf returns the router above the context. In debug builds it panics with
// the tag RouterMissing when there is none, or RouterTypeMismatch when the router
// has another configuration type.
//
// Flutter: widgets/router.dart Router.of
func RouterOf[T any](bc BuildContext) *Router[T] {
	scope := findRouterScope(bc)
	if foundation.DebugMode && scope == nil {
		panic(errors.New("RouterMissing",
			"Router operation requested with a context that does not include a Router; the context used to retrieve the Router must be that of a widget that is a descendant of a Router widget",
			"context", describeContext(bc)))
	}
	router, ok := scope.routerState.routerWidget().(*Router[T])
	if !ok {
		panic(errors.New("RouterTypeMismatch", "the Router above the context has another configuration type",
			"router", foundation.RuntimeTypeName(scope.routerState.routerWidget())))
	}
	return router
}

// RouterMaybeOf returns the router above the context, or nil when there is none.
//
// Flutter: widgets/router.dart Router.maybeOf
func RouterMaybeOf[T any](bc BuildContext) *Router[T] {
	scope := findRouterScope(bc)
	if scope == nil {
		return nil
	}
	router, _ := scope.routerState.routerWidget().(*Router[T])
	return router
}

func routerScopeElement(bc BuildContext) *routerScope {
	el := GetElementForInheritedWidget[*routerScope](bc)
	if el == nil {
		panic(errors.New("RouterScopeMissing", "Router.navigate and Router.neglect need a context below a Router", "context", describeContext(bc)))
	}
	return el.AsElement().widget.(*routerScope)
}

// RouterNavigate calls the callback, which changes the state of the app, and makes
// the router report the new route information as a navigation: the provider adds
// an entry to the history. The callback runs at once.
//
// Flutter: widgets/router.dart Router.navigate
func RouterNavigate(bc BuildContext, callback func()) {
	routerScopeElement(bc).routerState.setStateWithExplicitReportStatus(RouteInformationReportingTypeNavigate, callback)
}

// RouterNeglect calls the callback, which changes the state of the app, and makes
// the router report the new route information as a replacement: the provider
// replaces the current entry of the history. The callback runs at once.
//
// Flutter: widgets/router.dart Router.neglect
func RouterNeglect(bc BuildContext, callback func()) {
	routerScopeElement(bc).routerState.setStateWithExplicitReportStatus(RouteInformationReportingTypeNeglect, callback)
}

// sameObject is whether two values are the same one: it compares them with ==
// and treats values that cannot be compared as different.
func sameObject(a, b any) (same bool) {
	defer func() {
		if recover() != nil {
			same = false
		}
	}()
	return a == b
}

// routerTransaction identifies one chain of work of the router (a parse of route
// information, a pop): a later one makes the earlier ones stale.
type routerTransaction struct{ _ byte }

// routerState is the state of a Router.
//
// Flutter: widgets/router.dart _RouterState
type routerState[T any] struct {
	StateOf[*Router[T]]

	currentRouterTransaction *routerTransaction
	currentIntentionToReport foundation.Opt[RouteInformationReportingType]
	routeInformation         *RouteInformation
	routeParsePending        bool

	providerHandle foundation.ListenerHandle
	delegateHandle foundation.ListenerHandle
	backHandle     BackButtonCallbackHandle
	listening      routerListening

	routeInformationReportingTaskScheduled bool
	historyInitialized                     bool
}

// routerListening remembers what the state listens to, so that it can stop when
// the widget changes or goes away.
type routerListening struct {
	provider   RouteInformationProvider
	dispatcher BackButtonDispatcher
	delegate   foundation.Listenable
}

func (s *routerState[T]) routerWidget() Widget { return s.Widget() }

func (s *routerState[T]) binding() *WidgetsBinding { return WidgetsBindingOf(s.Context()) }

func (s *routerState[T]) post(fn func()) { s.binding().Post(fn) }

func (s *routerState[T]) listen() {
	w := s.Widget()
	if w.o.RouteInformationProvider != nil {
		s.providerHandle = w.o.RouteInformationProvider.AddListener(s.handleRouteInformationProviderNotification)
	}
	if w.o.BackButtonDispatcher != nil {
		s.backHandle = w.o.BackButtonDispatcher.AddCallback(s.handleBackButtonDispatcherNotification)
	}
	s.delegateHandle = w.delegate.AddListener(s.handleRouterDelegateNotification)
	s.listening = routerListening{w.o.RouteInformationProvider, w.o.BackButtonDispatcher, w.delegate}
}

func (s *routerState[T]) stopListening() {
	if s.listening.provider != nil {
		s.listening.provider.RemoveListener(s.providerHandle)
	}
	if s.listening.dispatcher != nil {
		s.listening.dispatcher.RemoveCallback(s.backHandle)
	}
	if s.listening.delegate != nil {
		s.listening.delegate.RemoveListener(s.delegateHandle)
	}
	s.listening = routerListening{}
}

// InitState starts listening to the provider, the dispatcher and the delegate.
//
// Flutter: widgets/router.dart _RouterState.initState
func (s *routerState[T]) InitState() { s.listen() }

// restoreState processes the initial route information of the provider. Flutter
// does it in RestorationMixin.didChangeDependencies; this port has no state
// restoration, so it runs the first time DidChangeDependencies is called.
//
// Flutter: widgets/router.dart _RouterState.restoreState
func (s *routerState[T]) restoreState() {
	if provider := s.Widget().o.RouteInformationProvider; provider != nil {
		s.processRouteInformation(provider.Value(), func() func(T) *Future[struct{}] {
			return s.Widget().delegate.SetInitialRoutePath
		})
	}
}

func (s *routerState[T]) scheduleRouteInformationReportingTask() {
	if s.routeInformationReportingTaskScheduled || s.Widget().o.RouteInformationProvider == nil {
		return
	}
	s.routeInformationReportingTaskScheduled = true
	WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(s.reportRouteInformation)
}

func (s *routerState[T]) reportRouteInformation(_ time.Duration) {
	if !s.Mounted() {
		return
	}
	s.routeInformationReportingTaskScheduled = false
	if s.routeInformation != nil {
		s.Widget().o.RouteInformationProvider.RouterReportsNewRouteInformation(s.routeInformation, s.currentIntentionToReport.OrElse(RouteInformationReportingTypeNone))
	}
	s.currentIntentionToReport = foundation.Some(RouteInformationReportingTypeNone)
}

func (s *routerState[T]) retrieveNewRouteInformation() *RouteInformation {
	w := s.Widget()
	configuration, ok := w.delegate.CurrentConfiguration().Get()
	if !ok || w.o.RouteInformationParser == nil {
		return nil
	}
	return w.o.RouteInformationParser.RestoreRouteInformation(configuration)
}

func (s *routerState[T]) setStateWithExplicitReportStatus(status RouteInformationReportingType, fn func()) {
	if foundation.DebugMode {
		if current, ok := s.currentIntentionToReport.Get(); ok && current != RouteInformationReportingTypeNone && current != status {
			if report := s.binding().ErrorReporter(); report != nil {
				report(errors.New("RouterReportAmbiguous",
					"both Router.navigate and Router.neglect have been called in this build cycle, and the Router cannot decide whether to report the route information; please make sure only one of them is called within the same build cycle"))
			}
		}
	}
	s.currentIntentionToReport = foundation.Some(status)
	s.scheduleRouteInformationReportingTask()
	fn()
}

func (s *routerState[T]) maybeNeedToReportRouteInformation() {
	s.routeInformation = s.retrieveNewRouteInformation()
	if !s.currentIntentionToReport.IsSome() {
		s.currentIntentionToReport = foundation.Some(RouteInformationReportingTypeNone)
	}
	s.scheduleRouteInformationReportingTask()
}

// DidChangeDependencies parses the route information of the provider the first
// time, and lets the parser look up dependencies again later.
//
// Flutter: widgets/router.dart _RouterState.didChangeDependencies
func (s *routerState[T]) DidChangeDependencies() {
	s.routeParsePending = true
	if !s.historyInitialized {
		s.historyInitialized = true
		s.restoreState()
	}
	// The restore may have parsed the route information already.
	var current *RouteInformation
	if s.routeInformation != nil {
		current = s.routeInformation
	} else if provider := s.Widget().o.RouteInformationProvider; provider != nil {
		current = provider.Value()
	}
	if current != nil && s.routeParsePending {
		s.processRouteInformation(current, func() func(T) *Future[struct{}] {
			return s.Widget().delegate.SetNewRoutePath
		})
	}
	s.routeParsePending = false
	s.maybeNeedToReportRouteInformation()
}

// DidUpdateWidget moves the listeners to the new objects of the widget.
//
// Flutter: widgets/router.dart _RouterState.didUpdateWidget
func (s *routerState[T]) DidUpdateWidget(old *Router[T]) {
	w := s.Widget()
	if !sameObject(w.o.RouteInformationProvider, old.o.RouteInformationProvider) ||
		!sameObject(w.o.BackButtonDispatcher, old.o.BackButtonDispatcher) ||
		!sameObject(w.o.RouteInformationParser, old.o.RouteInformationParser) ||
		!sameObject(w.delegate, old.delegate) {
		s.currentRouterTransaction = &routerTransaction{}
	}
	if !sameObject(w.o.RouteInformationProvider, old.o.RouteInformationProvider) {
		if s.listening.provider != nil {
			s.listening.provider.RemoveListener(s.providerHandle)
			s.listening.provider = nil
		}
		if w.o.RouteInformationProvider != nil {
			s.providerHandle = w.o.RouteInformationProvider.AddListener(s.handleRouteInformationProviderNotification)
			s.listening.provider = w.o.RouteInformationProvider
		}
		var oldValue, newValue *RouteInformation
		if old.o.RouteInformationProvider != nil {
			oldValue = old.o.RouteInformationProvider.Value()
		}
		if w.o.RouteInformationProvider != nil {
			newValue = w.o.RouteInformationProvider.Value()
		}
		if oldValue != newValue {
			s.handleRouteInformationProviderNotification()
		}
	}
	if !sameObject(w.o.BackButtonDispatcher, old.o.BackButtonDispatcher) {
		if s.listening.dispatcher != nil {
			s.listening.dispatcher.RemoveCallback(s.backHandle)
			s.listening.dispatcher = nil
		}
		if w.o.BackButtonDispatcher != nil {
			s.backHandle = w.o.BackButtonDispatcher.AddCallback(s.handleBackButtonDispatcherNotification)
			s.listening.dispatcher = w.o.BackButtonDispatcher
		}
	}
	if !sameObject(w.delegate, old.delegate) {
		if s.listening.delegate != nil {
			s.listening.delegate.RemoveListener(s.delegateHandle)
		}
		s.delegateHandle = w.delegate.AddListener(s.handleRouterDelegateNotification)
		s.listening.delegate = w.delegate
		s.maybeNeedToReportRouteInformation()
	}
}

// Dispose stops listening.
//
// Flutter: widgets/router.dart _RouterState.dispose
func (s *routerState[T]) Dispose() {
	s.stopListening()
	s.currentRouterTransaction = nil
}

// processRouteInformation parses the information and gives the result to the
// delegate, through the setter that the getter returns.
//
// Flutter: widgets/router.dart _RouterState._processRouteInformation
func (s *routerState[T]) processRouteInformation(information *RouteInformation, delegateRouteSetter func() func(T) *Future[struct{}]) {
	s.routeParsePending = false
	s.currentRouterTransaction = &routerTransaction{}
	transaction := s.currentRouterTransaction
	parser := s.Widget().o.RouteInformationParser
	parser.ParseRouteInformationWithDependencies(information, s.Context()).Then(s.post, s.processParsedRouteInformation(transaction, delegateRouteSetter))
}

// processParsedRouteInformation makes the callback that sets the parsed
// configuration in the delegate, unless a newer transaction started.
//
// Flutter: widgets/router.dart _RouterState._processParsedRouteInformation
func (s *routerState[T]) processParsedRouteInformation(transaction *routerTransaction, delegateRouteSetter func() func(T) *Future[struct{}]) func(T) {
	return func(data T) {
		if s.currentRouterTransaction != transaction {
			return
		}
		delegateRouteSetter()(data).Then(s.post, func(struct{}) {
			if s.currentRouterTransaction == transaction {
				s.rebuild()
			}
		})
	}
}

func (s *routerState[T]) handleRouteInformationProviderNotification() {
	s.routeParsePending = true
	s.processRouteInformation(s.Widget().o.RouteInformationProvider.Value(), func() func(T) *Future[struct{}] {
		return s.Widget().delegate.SetNewRoutePath
	})
}

// handleBackButtonDispatcherNotification asks the delegate to pop and rebuilds.
// It returns true when the pop was handled, or when another source rebuilt the
// router in the meantime, to prevent bubbling.
//
// Flutter: widgets/router.dart _RouterState._handleBackButtonDispatcherNotification
func (s *routerState[T]) handleBackButtonDispatcherNotification() bool {
	s.currentRouterTransaction = &routerTransaction{}
	transaction := s.currentRouterTransaction
	popped := s.Widget().delegate.PopRoute()
	if transaction != s.currentRouterTransaction {
		// A rebuild was triggered from a different source. Returns true to prevent
		// bubbling.
		return true
	}
	s.rebuild()
	return popped
}

func (s *routerState[T]) rebuild() {
	s.SetState(func() { /* the router delegate is ready to rebuild */ })
	s.maybeNeedToReportRouteInformation()
}

func (s *routerState[T]) handleRouterDelegateNotification() {
	s.SetState(func() { /* the router delegate wants to rebuild */ })
	s.maybeNeedToReportRouteInformation()
}

// Build builds the scope and the delegate's widgets below it.
//
// Flutter: widgets/router.dart _RouterState.build
func (s *routerState[T]) Build(BuildContext) Widget {
	w := s.Widget()
	var parser any
	if w.o.RouteInformationParser != nil {
		parser = w.o.RouteInformationParser
	}
	var provider RouteInformationProvider
	if w.o.RouteInformationProvider != nil {
		provider = w.o.RouteInformationProvider
	}
	return &routerScope{
		InheritedWidget:          NewInheritedWidgetBase(NewBuilder(w.delegate.Build)),
		routeInformationProvider: provider,
		backButtonDispatcher:     w.o.BackButtonDispatcher,
		routeInformationParser:   parser,
		routerDelegate:           w.delegate,
		routerState:              s,
	}
}

// routerScope tells the widgets below a router about it.
//
// Flutter: widgets/router.dart _RouterScope
type routerScope struct {
	InheritedWidget
	routeInformationProvider RouteInformationProvider
	backButtonDispatcher     BackButtonDispatcher
	routeInformationParser   any
	routerDelegate           any
	routerState              routerStateAccess
}

func (w *routerScope) UpdateShouldNotify(old Widget) bool {
	o := old.(*routerScope)
	return !sameObject(w.routeInformationProvider, o.routeInformationProvider) ||
		!sameObject(w.backButtonDispatcher, o.backButtonDispatcher) ||
		!sameObject(w.routeInformationParser, o.routeInformationParser) ||
		!sameObject(w.routerDelegate, o.routerDelegate) ||
		!sameObject(w.routerState, o.routerState)
}

// BackButtonCallback is a function of the back button dispatchers: it returns true
// when it handled the back button.
//
// Flutter: widgets/router.dart BackButtonDispatcher
type BackButtonCallback func() bool

// BackButtonCallbackHandle identifies a callback that was added to a dispatcher,
// to remove it.
//
// Flutter: no counterpart
type BackButtonCallbackHandle struct{ id uint64 }

type backButtonCallbackEntry struct {
	id uint64
	fn BackButtonCallback
}

// BackButtonDispatcher tells the router that the platform's back button was
// pressed. A dispatcher has at most one callback of its own, and child
// dispatchers that take priority over it.
//
// Flutter: widgets/router.dart BackButtonDispatcher
type BackButtonDispatcher interface {
	// HasCallbacks is whether the dispatcher or one of its children has a callback.
	//
	// Flutter: widgets/router.dart BackButtonDispatcher.hasCallbacks
	HasCallbacks() bool
	// AddCallback adds the callback that handles the back button; a dispatcher
	// supports one callback at a time.
	//
	// Flutter: widgets/router.dart BackButtonDispatcher.addCallback
	AddCallback(callback BackButtonCallback) BackButtonCallbackHandle
	// RemoveCallback removes the callback that AddCallback returned the handle of.
	//
	// Flutter: widgets/router.dart BackButtonDispatcher.removeCallback
	RemoveCallback(handle BackButtonCallbackHandle)
	// InvokeCallback calls the children, the last one that took priority first,
	// then the callback of the dispatcher, and returns what the first that handled
	// the button returned, or defaultValue.
	//
	// Flutter: widgets/router.dart BackButtonDispatcher.invokeCallback
	InvokeCallback(defaultValue bool) bool
	// CreateChildBackButtonDispatcher creates a child of this dispatcher.
	//
	// Flutter: widgets/router.dart BackButtonDispatcher.createChildBackButtonDispatcher
	CreateChildBackButtonDispatcher() *ChildBackButtonDispatcher
	// TakePriority makes this dispatcher the first one to be asked.
	//
	// Flutter: widgets/router.dart BackButtonDispatcher.takePriority
	TakePriority()
	// DeferTo makes the child the first one to be asked.
	//
	// Flutter: widgets/router.dart BackButtonDispatcher.deferTo
	DeferTo(child *ChildBackButtonDispatcher)
	// Forget removes the child from the ones that are asked.
	//
	// Flutter: widgets/router.dart BackButtonDispatcher.forget
	Forget(child *ChildBackButtonDispatcher)
	backButtonBase() *BackButtonDispatcherBase
}

// BackButtonDispatcherBase implements BackButtonDispatcher: embed it and call
// InitBackButtonDispatcherBase.
//
// Flutter: widgets/router.dart BackButtonDispatcher
type BackButtonDispatcherBase struct {
	self     BackButtonDispatcher
	callback []backButtonCallbackEntry
	nextID   uint64
	children []*ChildBackButtonDispatcher
	report   func() foundation.ErrorReporter
}

// InitBackButtonDispatcherBase sets the outer value of the dispatcher. The report
// function returns the reporter of the errors of the callbacks; it may be nil.
//
// Flutter: widgets/router.dart BackButtonDispatcher
func InitBackButtonDispatcherBase(b *BackButtonDispatcherBase, self BackButtonDispatcher, report func() foundation.ErrorReporter) {
	b.self = self
	b.report = report
}

func (b *BackButtonDispatcherBase) backButtonBase() *BackButtonDispatcherBase { return b }

// HasCallbacks is whether the dispatcher has a callback or children.
//
// Flutter: widgets/router.dart BackButtonDispatcher.hasCallbacks
func (b *BackButtonDispatcherBase) HasCallbacks() bool {
	return len(b.callback) > 0 || len(b.children) > 0
}

// AddCallback adds the callback.
//
// Flutter: widgets/router.dart BackButtonDispatcher.addCallback
func (b *BackButtonDispatcherBase) AddCallback(callback BackButtonCallback) BackButtonCallbackHandle {
	b.nextID++
	b.callback = append(b.callback, backButtonCallbackEntry{b.nextID, callback})
	return BackButtonCallbackHandle{b.nextID}
}

// RemoveCallback removes the callback of the handle.
//
// Flutter: widgets/router.dart BackButtonDispatcher.removeCallback
func (b *BackButtonDispatcherBase) RemoveCallback(handle BackButtonCallbackHandle) {
	b.callback = slices.DeleteFunc(b.callback, func(e backButtonCallbackEntry) bool { return e.id == handle.id })
}

// invokeOwnCallback calls the callback of the dispatcher itself; two callbacks are
// an error that is reported, with the default as the answer.
func (b *BackButtonDispatcherBase) invokeOwnCallback(defaultValue bool) bool {
	if len(b.callback) == 0 {
		return defaultValue
	}
	var report foundation.ErrorReporter
	if b.report != nil {
		report = b.report()
	}
	result := defaultValue
	foundation.Guard(report, foundation.CallbackPanicked, "while invoking the callback for "+foundation.RuntimeTypeName(b.self), nil, func() {
		if len(b.callback) != 1 {
			panic(errors.New("BackButtonDispatcherCallbacksInvalid", "a back button dispatcher must have exactly one callback to invoke it",
				"callbacks", len(b.callback)))
		}
		result = b.callback[0].fn()
	})
	return result
}

// InvokeCallback asks the children, the last one first, then the callback.
//
// Flutter: widgets/router.dart BackButtonDispatcher.invokeCallback
func (b *BackButtonDispatcherBase) InvokeCallback(defaultValue bool) bool {
	if len(b.children) > 0 {
		children := slices.Clone(b.children)
		for childIndex := len(children) - 1; childIndex >= 0; childIndex-- {
			// If a child handles the callback, returns its result.
			if result := children[childIndex].NotifiedByParent(defaultValue); result {
				return true
			}
		}
		// If none of the children handles the callback, the parent does.
		return b.invokeOwnCallback(defaultValue)
	}
	return b.invokeOwnCallback(defaultValue)
}

// CreateChildBackButtonDispatcher creates a child dispatcher.
//
// Flutter: widgets/router.dart BackButtonDispatcher.createChildBackButtonDispatcher
func (b *BackButtonDispatcherBase) CreateChildBackButtonDispatcher() *ChildBackButtonDispatcher {
	return NewChildBackButtonDispatcher(b.self)
}

// TakePriority removes the children, so that this dispatcher is asked first.
//
// Flutter: widgets/router.dart BackButtonDispatcher.takePriority
func (b *BackButtonDispatcherBase) TakePriority() { b.children = nil }

// DeferTo makes the child the first to be asked. In debug builds a dispatcher
// without callbacks panics with the tag BackButtonDispatcherEmpty.
//
// Flutter: widgets/router.dart BackButtonDispatcher.deferTo
func (b *BackButtonDispatcherBase) DeferTo(child *ChildBackButtonDispatcher) {
	if foundation.DebugMode && !b.self.HasCallbacks() {
		panic(errors.New("BackButtonDispatcherEmpty", "a back button dispatcher that defers to a child must have callbacks"))
	}
	b.children = slices.DeleteFunc(b.children, func(c *ChildBackButtonDispatcher) bool { return c == child })
	b.children = append(b.children, child)
}

// Forget removes the child.
//
// Flutter: widgets/router.dart BackButtonDispatcher.forget
func (b *BackButtonDispatcherBase) Forget(child *ChildBackButtonDispatcher) {
	b.children = slices.DeleteFunc(b.children, func(c *ChildBackButtonDispatcher) bool { return c == child })
}

// RootBackButtonDispatcher listens to the system for the back button, through the
// widgets binding, and tells the callbacks. It is the dispatcher of a Router that
// is not nested in another.
//
// Flutter: widgets/router.dart RootBackButtonDispatcher
type RootBackButtonDispatcher struct {
	BackButtonDispatcherBase
	WidgetsBindingObserverBase
	binding *WidgetsBinding
}

// NewRootBackButtonDispatcher creates a dispatcher that registers with the binding
// while it has callbacks.
//
// Deviation: Flutter reads WidgetsBinding.instance; Glutter has no singletons, so
// the binding is a parameter (SPEC §12.10).
//
// Flutter: widgets/router.dart RootBackButtonDispatcher.new
func NewRootBackButtonDispatcher(binding *WidgetsBinding) *RootBackButtonDispatcher {
	d := &RootBackButtonDispatcher{binding: binding}
	InitBackButtonDispatcherBase(&d.BackButtonDispatcherBase, d, binding.ErrorReporter)
	return d
}

// AddCallback adds the callback; the first one registers the dispatcher with the
// binding.
//
// Flutter: widgets/router.dart RootBackButtonDispatcher.addCallback
func (d *RootBackButtonDispatcher) AddCallback(callback BackButtonCallback) BackButtonCallbackHandle {
	if !d.HasCallbacks() {
		d.binding.AddObserver(d)
	}
	return d.BackButtonDispatcherBase.AddCallback(callback)
}

// RemoveCallback removes the callback; the last one unregisters the dispatcher.
//
// Flutter: widgets/router.dart RootBackButtonDispatcher.removeCallback
func (d *RootBackButtonDispatcher) RemoveCallback(handle BackButtonCallbackHandle) {
	d.BackButtonDispatcherBase.RemoveCallback(handle)
	if !d.HasCallbacks() {
		d.binding.RemoveObserver(d)
	}
}

// DidPopRoute invokes the callbacks, with false as the default.
//
// Flutter: widgets/router.dart RootBackButtonDispatcher.didPopRoute
func (d *RootBackButtonDispatcher) DidPopRoute() bool { return d.InvokeCallback(false) }

// ChildBackButtonDispatcher is a dispatcher below another one, for a Router nested
// in the router of the parent: the parent asks it first once it took priority.
//
// Flutter: widgets/router.dart ChildBackButtonDispatcher
type ChildBackButtonDispatcher struct {
	BackButtonDispatcherBase
	parent BackButtonDispatcher
}

// NewChildBackButtonDispatcher creates a child of the parent.
//
// Flutter: widgets/router.dart ChildBackButtonDispatcher.new
func NewChildBackButtonDispatcher(parent BackButtonDispatcher) *ChildBackButtonDispatcher {
	d := &ChildBackButtonDispatcher{parent: parent}
	InitBackButtonDispatcherBase(&d.BackButtonDispatcherBase, d, func() foundation.ErrorReporter {
		return parent.backButtonBase().errorReporter()
	})
	return d
}

func (b *BackButtonDispatcherBase) errorReporter() foundation.ErrorReporter {
	if b.report == nil {
		return nil
	}
	return b.report()
}

// Parent is the dispatcher that created this one.
//
// Flutter: widgets/router.dart ChildBackButtonDispatcher.parent
func (d *ChildBackButtonDispatcher) Parent() BackButtonDispatcher { return d.parent }

// NotifiedByParent is called when the parent was asked and has not handled the
// button; the child invokes its callbacks.
//
// Flutter: widgets/router.dart ChildBackButtonDispatcher.notifiedByParent
func (d *ChildBackButtonDispatcher) NotifiedByParent(defaultValue bool) bool {
	return d.self.InvokeCallback(defaultValue)
}

// TakePriority makes the parent ask this dispatcher first, and this dispatcher ask
// its own callback before its children.
//
// Flutter: widgets/router.dart ChildBackButtonDispatcher.takePriority
func (d *ChildBackButtonDispatcher) TakePriority() {
	d.parent.DeferTo(d)
	d.BackButtonDispatcherBase.TakePriority()
}

// DeferTo makes the parent defer to this dispatcher, and this one to the child.
//
// Flutter: widgets/router.dart ChildBackButtonDispatcher.deferTo
func (d *ChildBackButtonDispatcher) DeferTo(child *ChildBackButtonDispatcher) {
	if foundation.DebugMode && !d.self.HasCallbacks() {
		panic(errors.New("ChildBackButtonDispatcherEmpty", "a child back button dispatcher that defers to a child must have callbacks"))
	}
	d.parent.DeferTo(d)
	d.BackButtonDispatcherBase.DeferTo(child)
}

// RemoveCallback removes the callback; without callbacks the parent forgets the
// child.
//
// Flutter: widgets/router.dart ChildBackButtonDispatcher.removeCallback
func (d *ChildBackButtonDispatcher) RemoveCallback(handle BackButtonCallbackHandle) {
	d.BackButtonDispatcherBase.RemoveCallback(handle)
	if !d.self.HasCallbacks() {
		d.parent.Forget(d)
	}
}

// BackButtonListener registers a callback with the back button dispatcher of the
// router above it, and takes priority over it while it is in the tree.
//
// Flutter: widgets/router.dart BackButtonListener
type BackButtonListener struct {
	StatefulWidget
	child               Widget
	onBackButtonPressed BackButtonCallback
}

// NewBackButtonListener creates the widget; onBackButtonPressed returns true when
// it handled the button.
//
// Flutter: widgets/router.dart BackButtonListener.new
func NewBackButtonListener(onBackButtonPressed BackButtonCallback, child Widget, opts ...WidgetOpts) *BackButtonListener {
	return &BackButtonListener{NewStatefulWidgetBase(opts...), child, onBackButtonPressed}
}

// Child is the widget below.
//
// Flutter: widgets/router.dart BackButtonListener.child
func (w *BackButtonListener) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/router.dart BackButtonListener.createState
func (w *BackButtonListener) CreateState() State { return &backButtonListenerState{} }

type backButtonListenerState struct {
	StateOf[*BackButtonListener]
	dispatcher *ChildBackButtonDispatcher
	handle     BackButtonCallbackHandle
}

func (s *backButtonListenerState) DidChangeDependencies() {
	if s.dispatcher != nil {
		s.dispatcher.RemoveCallback(s.handle)
	}
	scope := findRouterScope(s.Context())
	var root BackButtonDispatcher
	if scope != nil {
		root = scope.backButtonDispatcher
	}
	if foundation.DebugMode && root == nil {
		panic(errors.New("BackButtonDispatcherMissing", "the parent router must have a backButtonDispatcher to use this widget"))
	}
	s.dispatcher = root.CreateChildBackButtonDispatcher()
	s.handle = s.dispatcher.AddCallback(s.Widget().onBackButtonPressed)
	s.dispatcher.TakePriority()
}

func (s *backButtonListenerState) DidUpdateWidget(old *BackButtonListener) {
	// Functions cannot be compared, so the callback is replaced every time, as it
	// is when the widget gets another closure in Flutter.
	if s.dispatcher != nil {
		s.dispatcher.RemoveCallback(s.handle)
		s.handle = s.dispatcher.AddCallback(s.Widget().onBackButtonPressed)
		s.dispatcher.TakePriority()
	}
}

func (s *backButtonListenerState) Dispose() {
	if s.dispatcher != nil {
		s.dispatcher.RemoveCallback(s.handle)
	}
}

func (s *backButtonListenerState) Build(BuildContext) Widget { return s.Widget().child }

// RouteInformationParser converts the route information of the platform to a
// configuration of the app, and back.
//
// Flutter: widgets/router.dart RouteInformationParser
type RouteInformationParser[T any] interface {
	// ParseRouteInformation converts the route information to a configuration.
	//
	// Flutter: widgets/router.dart RouteInformationParser.parseRouteInformation
	ParseRouteInformation(routeInformation *RouteInformation) *Future[T]
	// ParseRouteInformationWithDependencies is like ParseRouteInformation but gets
	// the context of the router, to look up dependencies; when a dependency
	// changes, the router parses again.
	//
	// Flutter: widgets/router.dart RouteInformationParser.parseRouteInformationWithDependencies
	ParseRouteInformationWithDependencies(routeInformation *RouteInformation, bc BuildContext) *Future[T]
	// RestoreRouteInformation converts the configuration to route information, or
	// returns nil to not report it to the platform.
	//
	// Flutter: widgets/router.dart RouteInformationParser.restoreRouteInformation
	RestoreRouteInformation(configuration T) *RouteInformation
}

// RouteInformationParserBase implements the defaults of RouteInformationParser:
// embed it, call InitRouteInformationParserBase, and implement
// ParseRouteInformation or ParseRouteInformationWithDependencies.
//
// Flutter: widgets/router.dart RouteInformationParser
type RouteInformationParserBase[T any] struct {
	self RouteInformationParser[T]
}

// InitRouteInformationParserBase sets the outer value of the parser.
//
// Flutter: widgets/router.dart RouteInformationParser.new
func InitRouteInformationParserBase[T any](b *RouteInformationParserBase[T], self RouteInformationParser[T]) {
	b.self = self
}

// ParseRouteInformation panics: one of the two parse methods must be implemented.
//
// Flutter: widgets/router.dart RouteInformationParser.parseRouteInformation
func (b *RouteInformationParserBase[T]) ParseRouteInformation(*RouteInformation) *Future[T] {
	panic(errors.New("RouteInformationParserUnimplemented",
		"one of ParseRouteInformation or ParseRouteInformationWithDependencies must be implemented"))
}

// ParseRouteInformationWithDependencies calls ParseRouteInformation.
//
// Flutter: widgets/router.dart RouteInformationParser.parseRouteInformationWithDependencies
func (b *RouteInformationParserBase[T]) ParseRouteInformationWithDependencies(routeInformation *RouteInformation, _ BuildContext) *Future[T] {
	return b.self.ParseRouteInformation(routeInformation)
}

// RestoreRouteInformation returns nil: nothing is reported.
//
// Flutter: widgets/router.dart RouteInformationParser.restoreRouteInformation
func (b *RouteInformationParserBase[T]) RestoreRouteInformation(T) *RouteInformation { return nil }

// RouterDelegate builds the widgets of a Router for a configuration of the app,
// and tells the router to rebuild (it is a Listenable) when the configuration
// changes.
//
// Flutter: widgets/router.dart RouterDelegate
type RouterDelegate[T any] interface {
	foundation.Listenable
	// SetInitialRoutePath is called when the router is created and the platform
	// gave an initial route; the default calls SetNewRoutePath.
	//
	// Flutter: widgets/router.dart RouterDelegate.setInitialRoutePath
	SetInitialRoutePath(configuration T) *Future[struct{}]
	// SetRestoredRoutePath is called when the router restores its state; the
	// default calls SetNewRoutePath.
	//
	// Flutter: widgets/router.dart RouterDelegate.setRestoredRoutePath
	SetRestoredRoutePath(configuration T) *Future[struct{}]
	// SetNewRoutePath is called when the platform pushes a new route and the
	// parser made the configuration of it; the future completes when the delegate
	// is ready to build.
	//
	// Flutter: widgets/router.dart RouterDelegate.setNewRoutePath
	SetNewRoutePath(configuration T) *Future[struct{}]
	// PopRoute is called when the back button is pressed; it returns true when it
	// handled it.
	//
	// Deviation: Flutter's popRoute returns a Future<bool>; Glutter's is synchronous
	// because Navigator.maybePop is (SPEC §12.10).
	//
	// Flutter: widgets/router.dart RouterDelegate.popRoute
	PopRoute() bool
	// CurrentConfiguration is the configuration of the app, which the parser
	// converts to route information for the platform, or none.
	//
	// Flutter: widgets/router.dart RouterDelegate.currentConfiguration
	CurrentConfiguration() foundation.Opt[T]
	// Build builds the widgets of the router.
	//
	// Flutter: widgets/router.dart RouterDelegate.build
	Build(bc BuildContext) Widget
}

// RouterDelegateBase implements the defaults of RouterDelegate and its Listenable:
// embed it, call InitRouterDelegateBase, and implement SetNewRoutePath, PopRoute
// and Build.
//
// Flutter: widgets/router.dart RouterDelegate
type RouterDelegateBase[T any] struct {
	foundation.ChangeNotifier
	self RouterDelegate[T]
}

// InitRouterDelegateBase sets the outer value of the delegate.
//
// Flutter: widgets/router.dart RouterDelegate.new
func InitRouterDelegateBase[T any](b *RouterDelegateBase[T], self RouterDelegate[T]) { b.self = self }

// SetInitialRoutePath calls SetNewRoutePath.
//
// Flutter: widgets/router.dart RouterDelegate.setInitialRoutePath
func (b *RouterDelegateBase[T]) SetInitialRoutePath(configuration T) *Future[struct{}] {
	return b.self.SetNewRoutePath(configuration)
}

// SetRestoredRoutePath calls SetNewRoutePath.
//
// Flutter: widgets/router.dart RouterDelegate.setRestoredRoutePath
func (b *RouterDelegateBase[T]) SetRestoredRoutePath(configuration T) *Future[struct{}] {
	return b.self.SetNewRoutePath(configuration)
}

// CurrentConfiguration is none.
//
// Flutter: widgets/router.dart RouterDelegate.currentConfiguration
func (b *RouterDelegateBase[T]) CurrentConfiguration() foundation.Opt[T] { return foundation.Opt[T]{} }

// RouteInformationProvider provides the route information of the platform to the
// router, and gets what the router reports back.
//
// Flutter: widgets/router.dart RouteInformationProvider
type RouteInformationProvider interface {
	foundation.ValueListenable[*RouteInformation]
	// RouterReportsNewRouteInformation is called when the router reports its route
	// information, for the provider to tell the platform.
	//
	// Flutter: widgets/router.dart RouteInformationProvider.routerReportsNewRouteInformation
	RouterReportsNewRouteInformation(routeInformation *RouteInformation, reportingType RouteInformationReportingType)
}

// RouteInformationProviderBase implements RouterReportsNewRouteInformation as a
// function that does nothing; embed it in a provider.
//
// Flutter: widgets/router.dart RouteInformationProvider
type RouteInformationProviderBase struct{}

// RouterReportsNewRouteInformation does nothing.
//
// Flutter: widgets/router.dart RouteInformationProvider.routerReportsNewRouteInformation
func (RouteInformationProviderBase) RouterReportsNewRouteInformation(*RouteInformation, RouteInformationReportingType) {
}

// PlatformRouteInformationProvider is the route information provider that listens
// to the routes that the platform pushes and keeps the route information that the
// router reports. There is no platform history on the desktop, so it keeps the
// entries in memory.
//
// Flutter: widgets/router.dart PlatformRouteInformationProvider
type PlatformRouteInformationProvider struct {
	foundation.ChangeNotifier
	WidgetsBindingObserverBase
	RouteInformationProviderBase
	binding       *WidgetsBinding
	value         *RouteInformation
	valueInEngine *RouteInformation
	observing     bool
}

// NewPlatformRouteInformationProvider creates the provider with the route
// information that the app starts with.
//
// Deviation: Flutter reads WidgetsBinding.instance; Glutter has no singletons, so
// the binding is a parameter (SPEC §12.10).
//
// Flutter: widgets/router.dart PlatformRouteInformationProvider.new
func NewPlatformRouteInformationProvider(binding *WidgetsBinding, initialRouteInformation *RouteInformation) *PlatformRouteInformationProvider {
	defaultRoute, _ := url.Parse(NavigatorDefaultRouteName)
	return &PlatformRouteInformationProvider{
		binding:       binding,
		value:         initialRouteInformation,
		valueInEngine: NewRouteInformation(RouteInformationOpts{URI: defaultRoute}),
	}
}

// routeURIsEqual is whether two URIs have the same path, fragment and query
// parameters, in any order.
func routeURIsEqual(a, b *url.URL) bool {
	if a.Path != b.Path || a.Fragment != b.Fragment {
		return false
	}
	qa, qb := a.Query(), b.Query()
	if len(qa) != len(qb) {
		return false
	}
	for key, va := range qa {
		vb, ok := qb[key]
		if !ok || len(va) != len(vb) {
			return false
		}
		sa, sb := slices.Clone(va), slices.Clone(vb)
		slices.Sort(sa)
		slices.Sort(sb)
		if !slices.Equal(sa, sb) {
			return false
		}
	}
	return true
}

// RouterReportsNewRouteInformation tells the platform about the route information
// that the router reports, as a replacement of the current entry of the history
// for a neglect report, or for a report without intention whose location is the
// same as the one the platform has, and as a new entry otherwise.
//
// It calls SelectMultiEntryHistory and RouteInformationUpdated of the binding's
// SystemNavigator.
//
// Flutter: widgets/router.dart PlatformRouteInformationProvider.routerReportsNewRouteInformation
func (p *PlatformRouteInformationProvider) RouterReportsNewRouteInformation(routeInformation *RouteInformation, reportingType RouteInformationReportingType) {
	replace := reportingType == RouteInformationReportingTypeNeglect ||
		(reportingType != RouteInformationReportingTypeNavigate && routeURIsEqual(p.valueInEngine.URI(), routeInformation.URI()))
	navigator := p.binding.SystemNavigator()
	navigator.SelectMultiEntryHistory()
	navigator.RouteInformationUpdated(routeInformation.URI(), routeInformation.State(), replace)
	p.value = routeInformation
	p.valueInEngine = routeInformation
}

// Value is the route information that the platform or the router reported last.
//
// Flutter: widgets/router.dart PlatformRouteInformationProvider.value
func (p *PlatformRouteInformationProvider) Value() *RouteInformation { return p.value }

// AddListener adds the listener; the first one registers the provider with the
// binding.
//
// Flutter: widgets/router.dart PlatformRouteInformationProvider.addListener
func (p *PlatformRouteInformationProvider) AddListener(listener foundation.VoidCallback) foundation.ListenerHandle {
	if !p.HasListeners() {
		p.binding.AddObserver(p)
		p.observing = true
	}
	return p.ChangeNotifier.AddListener(listener)
}

// RemoveListener removes the listener; without listeners the provider unregisters
// from the binding.
//
// Flutter: widgets/router.dart PlatformRouteInformationProvider.removeListener
func (p *PlatformRouteInformationProvider) RemoveListener(handle foundation.ListenerHandle) {
	p.ChangeNotifier.RemoveListener(handle)
	if !p.HasListeners() && p.observing {
		p.binding.RemoveObserver(p)
		p.observing = false
	}
}

// Dispose unregisters from the binding and disposes the notifier.
//
// Flutter: widgets/router.dart PlatformRouteInformationProvider.dispose
func (p *PlatformRouteInformationProvider) Dispose() {
	if p.observing {
		p.binding.RemoveObserver(p)
		p.observing = false
	}
	p.ChangeNotifier.Dispose()
}

// DidPushRouteInformation is called when the platform pushes a route: the value
// changes, and the listeners (the router) are told.
//
// Flutter: widgets/router.dart PlatformRouteInformationProvider.didPushRouteInformation
func (p *PlatformRouteInformationProvider) DidPushRouteInformation(routeInformation *RouteInformation) bool {
	if foundation.DebugMode && !p.HasListeners() {
		panic(errors.New("RouteInformationProviderNoListeners", "the platform pushed route information to a provider that has no listeners"))
	}
	if p.value == routeInformation {
		return true
	}
	p.value = routeInformation
	p.valueInEngine = routeInformation
	p.NotifyListeners()
	return true
}

// PopNavigatorRouterDelegatePopRoute is the PopRoute of a router delegate that has
// a Navigator: it asks the navigator to pop, and returns false (let the platform
// handle it) when there is no navigator.
//
// Deviation: Flutter's PopNavigatorRouterDelegateMixin is a mixin that reads the
// navigatorKey getter; a delegate calls this function with its key.
//
// Flutter: widgets/router.dart PopNavigatorRouterDelegateMixin.popRoute
func PopNavigatorRouterDelegatePopRoute(navigatorKey *GlobalKey[*NavigatorState]) bool {
	if navigatorKey == nil {
		return false
	}
	navigator, ok := navigatorKey.CurrentState()
	if !ok {
		return false
	}
	return navigator.MaybePop()
}
