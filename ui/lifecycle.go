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

// AppExitType is how an app is asked to exit: the app can refuse a cancelable
// exit, and cannot refuse a required one.
//
// Flutter: ui/platform_dispatcher.dart ui.AppExitType
type AppExitType int

// The values of AppExitType follow Flutter's declaration order.
const (
	// AppExitTypeUnspecified is the zero value.
	//
	// Flutter: no counterpart
	AppExitTypeUnspecified AppExitType = iota
	// AppExitTypeCancelable is an exit that the app can cancel, such as the user
	// closing the last window.
	//
	// Flutter: ui/platform_dispatcher.dart AppExitType.cancelable
	AppExitTypeCancelable
	// AppExitTypeRequired is an exit that the app cannot refuse, such as the
	// system shutting down.
	//
	// Flutter: ui/platform_dispatcher.dart AppExitType.required
	AppExitTypeRequired
)

var appExitTypeNames = [...]string{"AppExitTypeUnspecified", "AppExitTypeCancelable", "AppExitTypeRequired"}

// String returns the name of the value, such as "AppExitTypeCancelable".
//
// Flutter: ui/platform_dispatcher.dart AppExitType
func (t AppExitType) String() string { return enumName(appExitTypeNames[:], int(t), "AppExitType") }

// AppExitResponse is the answer of an app to a request to exit.
//
// Flutter: ui/platform_dispatcher.dart ui.AppExitResponse
type AppExitResponse int

// The values of AppExitResponse follow Flutter's declaration order.
const (
	// AppExitResponseUnspecified is the zero value; it counts as Exit.
	//
	// Flutter: no counterpart
	AppExitResponseUnspecified AppExitResponse = iota
	// AppExitResponseExit lets the app exit.
	//
	// Flutter: ui/platform_dispatcher.dart AppExitResponse.exit
	AppExitResponseExit
	// AppExitResponseCancel keeps the app running.
	//
	// Flutter: ui/platform_dispatcher.dart AppExitResponse.cancel
	AppExitResponseCancel
)

var appExitResponseNames = [...]string{"AppExitResponseUnspecified", "AppExitResponseExit", "AppExitResponseCancel"}

// String returns the name of the value, such as "AppExitResponseCancel".
//
// Flutter: ui/platform_dispatcher.dart AppExitResponse
func (r AppExitResponse) String() string {
	return enumName(appExitResponseNames[:], int(r), "AppExitResponse")
}
