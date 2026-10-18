package ui

// AppLifecycleState states that an application can be in once it is running.
// The zero value, AppLifecycleStateUnspecified, means no state is known yet.
//
// Flutter: ui/platform_dispatcher.dart AppLifecycleState
type AppLifecycleState int

// The values of AppLifecycleState follow Flutter's declaration order.
const (
	// AppLifecycleStateUnspecified is the zero value: no state is known yet.
	//
	// Flutter: no counterpart
	AppLifecycleStateUnspecified AppLifecycleState = iota
	// AppLifecycleStateDetached means the application is still hosted by the
	// engine but detached from any views, or is shutting down.
	//
	// Flutter: ui/platform_dispatcher.dart AppLifecycleState.detached
	AppLifecycleStateDetached
	// AppLifecycleStateResumed means the application is visible and responding
	// to user input.
	//
	// Flutter: ui/platform_dispatcher.dart AppLifecycleState.resumed
	AppLifecycleStateResumed
	// AppLifecycleStateInactive means the application is in an inactive state
	// and is not receiving user input.
	//
	// Flutter: ui/platform_dispatcher.dart AppLifecycleState.inactive
	AppLifecycleStateInactive
	// AppLifecycleStateHidden means all views of the application are hidden.
	//
	// Flutter: ui/platform_dispatcher.dart AppLifecycleState.hidden
	AppLifecycleStateHidden
	// AppLifecycleStatePaused means the application is not currently visible to
	// the user and not responding to user input. Desktop platforms do not use
	// it.
	//
	// Flutter: ui/platform_dispatcher.dart AppLifecycleState.paused
	AppLifecycleStatePaused
)

var appLifecycleStateNames = [...]string{
	"AppLifecycleStateUnspecified",
	"AppLifecycleStateDetached",
	"AppLifecycleStateResumed",
	"AppLifecycleStateInactive",
	"AppLifecycleStateHidden",
	"AppLifecycleStatePaused",
}

// String returns the name of the state, such as "AppLifecycleStateResumed".
//
// Flutter: ui/platform_dispatcher.dart AppLifecycleState
func (s AppLifecycleState) String() string {
	return enumName(appLifecycleStateNames[:], int(s), "AppLifecycleState")
}
