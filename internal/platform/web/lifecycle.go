package web

import "github.com/rm4n0s/glutter/ui"

// LifecycleOf is the application lifecycle state of a page: Hidden when the page is not
// visible (document.visibilityState, a background tab or a minimized window), Resumed
// when it is visible and has the focus, and Inactive when it is visible without it
// (SPEC §13, §23.3).
//
// Flutter: lib/web_ui/lib/src/engine/platform_dispatcher.dart EnginePlatformDispatcher.sendAppLifecycleState
func LifecycleOf(visible, focused bool) ui.AppLifecycleState {
	switch {
	case !visible:
		return ui.AppLifecycleStateHidden
	case focused:
		return ui.AppLifecycleStateResumed
	}
	return ui.AppLifecycleStateInactive
}
