package widgets

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// AppLifecycleListenerOpts are the optional parameters of NewAppLifecycleListener.
// Each callback is called for the matching transition of the app lifecycle
// state; a nil one is skipped.
//
// Flutter: widgets/app_lifecycle_listener.dart AppLifecycleListener.new
type AppLifecycleListenerOpts struct {
	// OnResume is called when the app is resumed (focused and visible).
	OnResume func()
	// OnInactive is called when the app goes from resumed to inactive (visible,
	// not focused).
	OnInactive func()
	// OnHide is called when the app goes from inactive to hidden.
	OnHide func()
	// OnShow is called when the app goes from hidden to inactive.
	OnShow func()
	// OnPause is called when the app goes from hidden to paused. Desktop
	// platforms do not use it.
	OnPause func()
	// OnRestart is called when the app goes from paused to hidden.
	OnRestart func()
	// OnDetach is called when the app is detached from the platform: it is
	// about to end.
	OnDetach func()
	// OnExitRequested is called when the app is asked to exit; the answer Cancel
	// keeps the app running (SPEC §8.7 "Closing"). Nil lets the app exit.
	OnExitRequested func() ui.AppExitResponse
	// OnStateChange is called with the new state after each change of the
	// lifecycle state, after the callback of the transition.
	OnStateChange func(state ui.AppLifecycleState)
}

// AppLifecycleListener listens to the changes of the app lifecycle state and to
// the requests to exit, and calls a callback for each. Create one with
// NewAppLifecycleListener and call Dispose when done.
//
// Flutter: widgets/app_lifecycle_listener.dart AppLifecycleListener
type AppLifecycleListener struct {
	WidgetsBindingObserverBase
	binding  *WidgetsBinding
	opts     AppLifecycleListenerOpts
	state    ui.AppLifecycleState
	disposed bool
}

// NewAppLifecycleListener creates a listener and registers it with the binding.
// It starts from the lifecycle state the scheduler binding has.
//
// Flutter: widgets/app_lifecycle_listener.dart AppLifecycleListener.new
func NewAppLifecycleListener(binding *WidgetsBinding, opts ...AppLifecycleListenerOpts) *AppLifecycleListener {
	l := &AppLifecycleListener{binding: binding, opts: firstOpts(opts), state: binding.scheduler.LifecycleState()}
	binding.AddObserver(l)
	return l
}

// Binding is the binding the listener is registered with.
//
// Flutter: widgets/app_lifecycle_listener.dart AppLifecycleListener.binding
func (l *AppLifecycleListener) Binding() *WidgetsBinding { return l.binding }

// Dispose removes the listener from the binding. Using the listener after this
// panics in debug builds with the tag AppLifecycleListenerDisposed.
//
// Flutter: widgets/app_lifecycle_listener.dart AppLifecycleListener.dispose
func (l *AppLifecycleListener) Dispose() {
	l.debugAssertNotDisposed()
	l.binding.RemoveObserver(l)
	l.disposed = true
}

func (l *AppLifecycleListener) debugAssertNotDisposed() {
	if foundation.DebugMode && l.disposed {
		panic(errors.New(tagAppLifecycleListenerDisposed,
			"an AppLifecycleListener was used after being disposed; once you have called Dispose on an AppLifecycleListener, it can no longer be used"))
	}
}

// DidRequestAppExit asks OnExitRequested, or lets the app exit.
//
// Flutter: widgets/app_lifecycle_listener.dart AppLifecycleListener.didRequestAppExit
func (l *AppLifecycleListener) DidRequestAppExit() ui.AppExitResponse {
	l.debugAssertNotDisposed()
	if l.opts.OnExitRequested == nil {
		return ui.AppExitResponseExit
	}
	return l.opts.OnExitRequested()
}

// DidChangeAppLifecycleState calls the callback of the transition from the
// previous state to the new one. In debug builds a transition that the
// lifecycle does not allow panics with the tag AppLifecycleInvalidTransition.
//
// Flutter: widgets/app_lifecycle_listener.dart AppLifecycleListener.didChangeAppLifecycleState
func (l *AppLifecycleListener) DidChangeAppLifecycleState(state ui.AppLifecycleState) {
	l.debugAssertNotDisposed()
	previous := l.state
	if state == previous {
		return
	}
	l.state = state
	none := ui.AppLifecycleStateUnspecified
	check := func(ok bool) {
		if foundation.DebugMode && !ok {
			panic(errors.New(tagAppLifecycleInvalidTransition, "invalid state transition from "+previous.String()+" to "+state.String(),
				"from", previous.String(), "to", state.String()))
		}
	}
	call := func(fn func()) {
		if fn != nil {
			fn()
		}
	}
	switch state {
	case ui.AppLifecycleStateResumed:
		check(previous == none || previous == ui.AppLifecycleStateInactive || previous == ui.AppLifecycleStateDetached)
		call(l.opts.OnResume)
	case ui.AppLifecycleStateInactive:
		check(previous == none || previous == ui.AppLifecycleStateHidden || previous == ui.AppLifecycleStateResumed)
		switch previous {
		case ui.AppLifecycleStateHidden:
			call(l.opts.OnShow)
		case none, ui.AppLifecycleStateResumed:
			call(l.opts.OnInactive)
		}
	case ui.AppLifecycleStateHidden:
		check(previous == none || previous == ui.AppLifecycleStatePaused || previous == ui.AppLifecycleStateInactive)
		switch previous {
		case ui.AppLifecycleStatePaused:
			call(l.opts.OnRestart)
		case none, ui.AppLifecycleStateInactive:
			call(l.opts.OnHide)
		}
	case ui.AppLifecycleStatePaused:
		check(previous == none || previous == ui.AppLifecycleStateHidden)
		if previous == none || previous == ui.AppLifecycleStateHidden {
			call(l.opts.OnPause)
		}
	case ui.AppLifecycleStateDetached:
		check(previous == none || previous == ui.AppLifecycleStatePaused)
		call(l.opts.OnDetach)
	}
	if l.opts.OnStateChange != nil {
		l.opts.OnStateChange(state)
	}
}

// DebugFillProperties adds the binding and the callbacks that are set.
//
// Flutter: widgets/app_lifecycle_listener.dart AppLifecycleListener.debugFillProperties
func (l *AppLifecycleListener) DebugFillProperties(p *foundation.DiagnosticPropertiesBuilder) {
	flag := func(name string, set bool) {
		p.Add(foundation.NewFlagProperty(name, foundation.Some(set), foundation.FlagPropertyOpts{IfTrue: name}))
	}
	flag("onStateChange", l.opts.OnStateChange != nil)
	flag("onInactive", l.opts.OnInactive != nil)
	flag("onResume", l.opts.OnResume != nil)
	flag("onHide", l.opts.OnHide != nil)
	flag("onShow", l.opts.OnShow != nil)
	flag("onPause", l.opts.OnPause != nil)
	flag("onRestart", l.opts.OnRestart != nil)
	flag("onExitRequested", l.opts.OnExitRequested != nil)
	flag("onDetach", l.opts.OnDetach != nil)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagAppLifecycleInvalidTransition = "AppLifecycleInvalidTransition"
	tagAppLifecycleListenerDisposed  = "AppLifecycleListenerDisposed"
)
