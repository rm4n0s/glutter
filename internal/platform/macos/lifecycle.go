package macos

import "github.com/rm4n0s/glutter/ui"

// This file is the plain-Go part of the lifecycle of the app (SPEC §29.8): the platform reports the
// state itself, from what its windows and the app are, as Flutter's macOS embedder reports it from the
// notifications of NSApplication (FlutterAppLifecycleDelegate).

// lifecycleWindow is what the lifecycle looks at in a window.
type lifecycleWindow struct {
	// Visible is whether the window was shown after its first frame and the app did not hide it
	// (SetHidden).
	Visible bool
	// Minimized is whether the window is minimized to the Dock.
	Minimized bool
	// Key is whether the window is the key window of the app, with the keyboard focus.
	Key bool
}

// lifecycleOf is the lifecycle state:
//   - Detached when the app is ending (the loop is stopping);
//   - Hidden when the app is hidden (Command+H) or every window is minimized or hidden (and so with no
//     window yet);
//   - Resumed when a window is visible, not minimized and the key window;
//   - Inactive when a window is visible and none is the key window (another app is in front).
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterAppDelegate.mm FlutterAppDelegate applicationDidHide
func lifecycleOf(windows []lifecycleWindow, appHidden, ended bool) ui.AppLifecycleState {
	if ended {
		return ui.AppLifecycleStateDetached
	}
	if appHidden {
		return ui.AppLifecycleStateHidden
	}
	state := ui.AppLifecycleStateHidden
	for _, w := range windows {
		if !w.Visible || w.Minimized {
			continue
		}
		if w.Key {
			return ui.AppLifecycleStateResumed
		}
		state = ui.AppLifecycleStateInactive
	}
	return state
}

// lifecycleTracker remembers the state that was reported, so that only a change is reported again.
type lifecycleTracker struct {
	last     ui.AppLifecycleState
	reported bool
}

// update gives the state to report: changed is false when it is the one reported last. A first state
// that is Hidden is not reported: an app with no window shown yet has not left its start state, and the
// first window brings Resumed or Inactive.
func (t *lifecycleTracker) update(windows []lifecycleWindow, appHidden, ended bool) (state ui.AppLifecycleState, changed bool) {
	state = lifecycleOf(windows, appHidden, ended)
	if t.reported && state == t.last {
		return state, false
	}
	if !t.reported && state == ui.AppLifecycleStateHidden {
		return state, false
	}
	t.last, t.reported = state, true
	return state, true
}

// The four-character codes of the quit Apple event (AE/AppleEvents.h, AERegistry.h): the event's class and ID,
// the attribute of its reason, and the reasons that the system gives when it ends the session.
const (
	aeCoreEventClass  = 'a'<<24 | 'e'<<16 | 'v'<<8 | 't' // kCoreEventClass
	aeQuitApplication = 'q'<<24 | 'u'<<16 | 'i'<<8 | 't' // kAEQuitApplication
	aeQuitReason      = 'w'<<24 | 'h'<<16 | 'y'<<8 | '?' // keyAEQuitReason
	aeErrorNumber     = 'e'<<24 | 'r'<<16 | 'r'<<8 | 'n' // keyErrorNumber
	aeLogOut          = 'l'<<24 | 'o'<<16 | 'g'<<8 | 'o' // kAELogOut
	aeReallyLogOut    = 'r'<<24 | 'l'<<16 | 'g'<<8 | 'o' // kAEReallyLogOut
	aeShowRestart     = 'r'<<24 | 'r'<<16 | 's'<<8 | 't' // kAEShowRestartDialog
	aeRestart         = 'r'<<24 | 'e'<<16 | 's'<<8 | 't' // kAERestart
	aeShowShutdown    = 'r'<<24 | 's'<<16 | 'd'<<8 | 'n' // kAEShowShutdownDialog
	aeShutDown        = 's'<<24 | 'h'<<16 | 'u'<<8 | 't' // kAEShutDown
	aeUserCanceledErr = -128                             // userCanceledErr: the reply of an app that refuses to quit
	aeDescriptorNull  = 'n'<<24 | 'u'<<16 | 'l'<<8 | 'l' // typeNull: a reply that nobody waits for
)

// quitReasonOf is the reason of a quit Apple event whose keyAEQuitReason is code (0 when it has none): the
// system's logout, restart and shutdown, and the questions before them, end the session; any other quit (the
// Dock's Quit, a script) is the app's Quit.
func quitReasonOf(code uint32) ui.AppExitReason {
	switch code {
	case aeLogOut, aeReallyLogOut, aeShowRestart, aeRestart, aeShowShutdown, aeShutDown:
		return ui.AppExitReasonSessionEnd
	}
	return ui.AppExitReasonQuit
}

// openItem is one URL of application:openURLs: as Go reads it: the path of a file URL, or the text of any other.
type openItem struct {
	// File says that the URL is a file URL, and Text its path.
	File bool
	// Text is the path of the file, or the URL as it is written.
	Text string
}

// openRequestOf is the request that the items make: the files and the URLs, each in their order. An item
// without text (a file URL with no path) is left out.
func openRequestOf(items []openItem) ui.OpenRequest {
	var r ui.OpenRequest
	for _, it := range items {
		switch {
		case it.Text == "":
		case it.File:
			r.Files = append(r.Files, it.Text)
		default:
			r.URLs = append(r.URLs, it.Text)
		}
	}
	return r
}
