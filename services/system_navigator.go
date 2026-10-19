package services

import (
	"net/url"

	"github.com/rm4n0s/glutter/foundation"
)

// SystemNavigator controls the navigation of the system around the app: popping
// the last route closes the app, and a router tells the system about its route
// information. The binding makes one, and sets OnPop to end the app.
//
// Deviation: Flutter sends these calls to the platform through the
// flutter/platform and flutter/navigation channels; here they are the hooks of
// the struct, which the binding wires. The desktop has no browser history, so
// the history hooks are for a host that has one, or a test (SPEC §12.10).
//
// Flutter: services/system_navigator.dart SystemNavigator
type SystemNavigator struct {
	// OnPop is called by Pop; the binding sets it to end the app.
	OnPop func(animated foundation.Opt[bool])
	// OnSelectSingleEntryHistory is called by SelectSingleEntryHistory.
	OnSelectSingleEntryHistory func()
	// OnSelectMultiEntryHistory is called by SelectMultiEntryHistory.
	OnSelectMultiEntryHistory func()
	// OnRouteInformationUpdated is called by RouteInformationUpdated.
	OnRouteInformationUpdated func(uri *url.URL, state any, replace bool)
}

// SetFrameworkHandlesBack tells the system whether the framework handles the back
// gesture. It matters only on Android; here it does nothing.
//
// Flutter: services/system_navigator.dart SystemNavigator.setFrameworkHandlesBack
func (n *SystemNavigator) SetFrameworkHandlesBack(bool) {}

// Pop tells the system to pop the top route, which closes the app when it is the
// last one. Animated says whether the system animates it (iOS only).
//
// Flutter: services/system_navigator.dart SystemNavigator.pop
func (n *SystemNavigator) Pop(animated ...bool) {
	if n == nil || n.OnPop == nil {
		return
	}
	a := foundation.None[bool]()
	if len(animated) > 0 {
		a = foundation.Some(animated[0])
	}
	n.OnPop(a)
}

// SelectSingleEntryHistory selects the single-entry history mode: the system
// history has one entry, and the system back pops the app.
//
// Flutter: services/system_navigator.dart SystemNavigator.selectSingleEntryHistory
func (n *SystemNavigator) SelectSingleEntryHistory() {
	if n != nil && n.OnSelectSingleEntryHistory != nil {
		n.OnSelectSingleEntryHistory()
	}
}

// SelectMultiEntryHistory selects the multiple-entry history mode: the system
// history has an entry for each route the router reports.
//
// Flutter: services/system_navigator.dart SystemNavigator.selectMultiEntryHistory
func (n *SystemNavigator) SelectMultiEntryHistory() {
	if n != nil && n.OnSelectMultiEntryHistory != nil {
		n.OnSelectMultiEntryHistory()
	}
}

// RouteInformationUpdated tells the system the route information of the app. With
// replace the system replaces its current history entry, and otherwise it adds
// one.
//
// Deviation: Flutter's deprecated location parameter is not ported.
//
// Flutter: services/system_navigator.dart SystemNavigator.routeInformationUpdated
func (n *SystemNavigator) RouteInformationUpdated(uri *url.URL, state any, replace bool) {
	if n != nil && n.OnRouteInformationUpdated != nil {
		n.OnRouteInformationUpdated(uri, state, replace)
	}
}
