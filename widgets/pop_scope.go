package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
)

// PopScopeOpts are the optional parameters of NewPopScope.
//
// Flutter: widgets/pop_scope.dart PopScope.new
type PopScopeOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// CanPop is whether the route can be popped; the default is true.
	CanPop foundation.Opt[bool]
	// OnPopInvokedWithResult is called after a pop was handled (didPop true) or
	// blocked (false).
	OnPopInvokedWithResult PopInvokedWithResultCallback
}

// PopScope manages whether the route it is in can be popped, and is told about
// the pops: a back gesture, the Navigator's MaybePop, or the system back.
//
// Flutter: widgets/pop_scope.dart PopScope
type PopScope struct {
	StatefulWidget
	child                  Widget
	canPop                 bool
	onPopInvokedWithResult PopInvokedWithResultCallback
}

// NewPopScope creates the widget.
//
// Flutter: widgets/pop_scope.dart PopScope.new
func NewPopScope(child Widget, opts ...PopScopeOpts) *PopScope {
	o := firstOpts(opts)
	return &PopScope{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child, o.CanPop.OrElse(true), o.OnPopInvokedWithResult}
}

// Child is the widget below.
//
// Flutter: widgets/pop_scope.dart PopScope.child
func (w *PopScope) Child() Widget { return w.child }

// CanPop is whether the route can be popped.
//
// Flutter: widgets/pop_scope.dart PopScope.canPop
func (w *PopScope) CanPop() bool { return w.canPop }

// CreateState creates the state.
//
// Flutter: widgets/pop_scope.dart PopScope.createState
func (w *PopScope) CreateState() State { return &popScopeState{} }

type popScopeState struct {
	StateOf[*PopScope]
	route       ModalRoute
	canPopValue *foundation.ValueNotifier[bool]
}

// OnPopInvokedWithResult calls the callback of the widget.
func (s *popScopeState) OnPopInvokedWithResult(didPop bool, result any) {
	if cb := s.Widget().onPopInvokedWithResult; cb != nil {
		cb(didPop, result)
	}
}

func (s *popScopeState) CanPopNotifier() foundation.ValueListenable[bool] { return s.canPopValue }

func (s *popScopeState) InitState() {
	s.canPopValue = foundation.NewValueNotifier(s.Widget().canPop)
}

func (s *popScopeState) DidChangeDependencies() {
	nextRoute := ModalRouteOf(s.Context())
	if nextRoute != s.route {
		if s.route != nil {
			s.route.UnregisterPopEntry(s)
		}
		s.route = nextRoute
		if s.route != nil {
			s.route.RegisterPopEntry(s)
		}
	}
}

func (s *popScopeState) DidUpdateWidget(old *PopScope) {
	s.canPopValue.SetValue(s.Widget().canPop)
}

func (s *popScopeState) Dispose() {
	if s.route != nil {
		s.route.UnregisterPopEntry(s)
	}
	s.canPopValue.Dispose()
}

func (s *popScopeState) Build(bc BuildContext) Widget { return s.Widget().child }

// PopResultCallback is called with the result of a pop.
//
// Flutter: widgets/navigator_pop_handler.dart PopResultCallback
type PopResultCallback func(result any)

// NavigatorPopHandlerOpts are the optional parameters of NewNavigatorPopHandler.
//
// Flutter: widgets/navigator_pop_handler.dart NavigatorPopHandler.new
type NavigatorPopHandlerOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// OnPopWithResult is called when a pop is invoked but not handled by the
	// navigator below.
	OnPopWithResult PopResultCallback
	// Disabled makes the handler let pops through; it is enabled by default.
	Disabled bool
}

// NavigatorPopHandler enables the handling of system back gestures for a nested
// Navigator: the pops go to the nested one first, and to onPop when it can't
// handle them.
//
// Flutter: widgets/navigator_pop_handler.dart NavigatorPopHandler
type NavigatorPopHandler struct {
	StatefulWidget
	child           Widget
	enabled         bool
	onPopWithResult PopResultCallback
}

// NewNavigatorPopHandler creates the widget.
//
// Flutter: widgets/navigator_pop_handler.dart NavigatorPopHandler.new
func NewNavigatorPopHandler(child Widget, opts ...NavigatorPopHandlerOpts) *NavigatorPopHandler {
	o := firstOpts(opts)
	return &NavigatorPopHandler{NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), child, !o.Disabled, o.OnPopWithResult}
}

// Child is the widget below.
//
// Flutter: widgets/navigator_pop_handler.dart NavigatorPopHandler.child
func (w *NavigatorPopHandler) Child() Widget { return w.child }

// Enabled is whether the handler takes the pops.
//
// Flutter: widgets/navigator_pop_handler.dart NavigatorPopHandler.enabled
func (w *NavigatorPopHandler) Enabled() bool { return w.enabled }

// CreateState creates the state.
//
// Flutter: widgets/navigator_pop_handler.dart NavigatorPopHandler.createState
func (w *NavigatorPopHandler) CreateState() State { return &navigatorPopHandlerState{canPop: true} }

type navigatorPopHandlerState struct {
	StateOf[*NavigatorPopHandler]
	canPop bool
}

func (s *navigatorPopHandlerState) Build(bc BuildContext) Widget {
	w := s.Widget()
	// When the widget subtree indicates it can handle a pop, popping here is
	// disabled, so that it can be handled manually in canPop.
	return NewPopScope(
		NewNotificationListener(func(notification NavigationNotification) bool {
			// If the subtree can't handle a pop, canPop is true so that the
			// PopScope lets the navigator higher in the tree handle the pop.
			nextCanPop := !notification.CanHandlePop
			if nextCanPop != s.canPop {
				s.SetState(func() { s.canPop = nextCanPop })
			}
			return false
		}, w.child),
		PopScopeOpts{
			CanPop: foundation.Some(!w.enabled || s.canPop),
			OnPopInvokedWithResult: func(didPop bool, result any) {
				if didPop {
					return
				}
				if w.onPopWithResult != nil {
					w.onPopWithResult(result)
				}
			},
		},
	)
}

var _ = errors.New
