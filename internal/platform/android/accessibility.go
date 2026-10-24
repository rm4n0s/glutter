package android

import (
	"context"
	"sync"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// The operations of the activity's command call for accessibility (back.go has 14 and 15).
const (
	// opSemantics: S is a semantics update, see accessibility_wire.go. The Java side copies it and
	// applies it on the main thread.
	opSemantics = 16
	// opA11yForce: A is 1 to make the Java side behave as if a screen reader were on, 0 to stop
	// (debug builds only, GLUTTER_A11Y; the emulator images have no TalkBack).
	opA11yForce = 17
	// opA11yReset: the Java side drops its copy of the tree (the bridge went off, or the view went).
	opA11yReset = 18
)

// A11yEnv is the environment variable (an intent extra) that forces the accessibility bridge on in
// debug builds, so that the emulator images without TalkBack can test it (SPEC §24.13). It is read
// only in debug builds (foundation.DebugMode).
//
// Flutter: no counterpart
const A11yEnv = "GLUTTER_A11Y"

// a11yState is what the accessibility manager of Android says (Native.a11yState).
type a11yState struct {
	// enabled is AccessibilityManager.isEnabled: an accessibility service runs.
	enabled bool
	// touch is AccessibilityManager.isTouchExplorationEnabled.
	touch bool
	// navigation is Flutter's accessibleNavigation: touch exploration is on, or a service asked for
	// the nodes (AccessibilityBridge.setAccessibleNavigation).
	navigation bool
}

// on says whether the framework should make the semantics tree: a service runs or touch
// exploration is on.
func (s a11yState) on() bool { return s.enabled || s.touch }

// accessibility is the platform.Accessibility of Android: it follows the accessibility manager
// through the Java side, turns the framework's semantics tree on and off with it, sends each
// update to the main thread (GlutterAccessibility keeps the tree there and answers the system),
// and gives the framework the actions that come back. Its methods are for the UI thread, except
// Close.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityBridge.java AccessibilityBridge
type accessibility struct {
	p *Platform

	// Everything below belongs to the UI thread, but closed.
	host platform.AccessibilityHost
	// forced is whether a debug build forces the bridge on.
	forced bool
	// on is whether the host was told that a screen reader runs.
	on bool
	// navigation is the accessibleNavigation the framework was told last.
	navigation bool
	// custom are the custom actions the framework defined, by id: a node refers to them by id.
	custom map[int]ui.SemanticsCustomActionUpdate
	// title and active are the window's, as SetWindow gave them.
	title  string
	active bool

	mu     sync.Mutex
	closed bool
}

var _ platform.Accessibility = (*accessibility)(nil)

func newAccessibility(p *Platform) *accessibility {
	return &accessibility{p: p, custom: map[int]ui.SemanticsCustomActionUpdate{}}
}

// isClosed reports whether Close was called; it is safe from any goroutine.
func (a *accessibility) isClosed() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.closed
}

// services is the platform.Services of Android: the accessibility bridge, and nothing else yet
// (the desktop services come in step 25.12).
//
// Flutter: no counterpart
type services struct{ a *accessibility }

var _ platform.Services = services{}

// Accessibility implements platform.Services.
func (s services) Accessibility() platform.Accessibility { return s.a }

// Start gives the bridge the host, and tells it at once whether a screen reader runs. The bridge
// stops when ctx ends or Close is called. Unlike the desktops, Android has no service to wait
// for: the Java side reports the accessibility manager's state (Native.a11yState) from the main
// thread, and the bridge acts on it.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityBridge.java AccessibilityBridge.AccessibilityBridge
func (a *accessibility) Start(ctx context.Context, host platform.AccessibilityHost) {
	if a.isClosed() || a.host != nil {
		return
	}
	a.host = host
	context.AfterFunc(ctx, a.Close)
	a.sync()
}

// sync tells the host when a screen reader started or stopped. When it stopped, the Java side drops
// its tree as well.
func (a *accessibility) sync() {
	if a.host == nil || a.isClosed() {
		return
	}
	want := a.forced || a.p.b.a11ySnapshot().on()
	if want == a.on {
		return
	}
	a.on = want
	a.host.ScreenReaderChanged(want)
	if !want {
		a.p.host.Command(command{Op: opA11yReset})
	}
}

// stateChanged is a change of the accessibility manager's state (Native.a11yState): the bridge goes
// on or off, and the framework hears of accessibleNavigation.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityBridge.java AccessibilityBridge.accessibilityStateChangeListener
func (a *accessibility) stateChanged() {
	a.sync()
	if nav := a.p.b.a11ySnapshot().navigation; nav != a.navigation {
		a.navigation = nav
		a.p.settingsChanged()
	}
}

// force makes the Java side behave as if a screen reader were on (debug builds, GLUTTER_A11Y).
func (a *accessibility) force() {
	a.forced = true
	a.p.host.Command(command{Op: opA11yForce, A: 1})
	a.sync()
}

// resend is for a new activity in a living process, which knows nothing of the tree: what the
// activity keeps is asked for again, and the framework makes the tree again, by turning
// the semantics off and on.
func (a *accessibility) resend() {
	if a.forced {
		a.p.host.Command(command{Op: opA11yForce, A: 1})
	}
	if a.on && a.host != nil && !a.isClosed() {
		a.on = false
		a.host.ScreenReaderChanged(false)
		a.sync()
	}
}

// UpdateSemantics sends the changes of the tree of the view to the main thread. While the bridge is
// off nothing is sent (the framework makes no tree then; an update that was on its way is stale),
// and the one view of Android is the only one.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityBridge.java AccessibilityBridge.updateSemantics
func (a *accessibility) UpdateSemantics(view ui.ViewID, update *ui.SemanticsUpdate) {
	if update == nil || view != ui.ImplicitViewID || !a.on || a.isClosed() {
		return
	}
	for _, c := range update.CustomActions {
		a.custom[c.ID] = c
	}
	if len(update.Nodes) == 0 {
		return
	}
	nodes := make([]androidNode, len(update.Nodes))
	for i, n := range update.Nodes {
		nodes[i] = newAndroidNode(n, a.custom)
	}
	a.p.host.Command(command{Op: opSemantics, S: string(encodeSemantics(nodes))})
}

// SetWindow keeps the title and the focus of the window. Android names a screen by the route that
// the tree says (the pane title of the view, set by the Java side), not by the title of the window,
// so they are not used.
//
// Flutter: no counterpart
func (a *accessibility) SetWindow(_ ui.ViewID, title string, active bool) {
	a.title, a.active = title, active
}

// RemoveView drops the tree of the view that closed.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityBridge.java AccessibilityBridge.reset
func (a *accessibility) RemoveView(view ui.ViewID) {
	if view != ui.ImplicitViewID || a.isClosed() {
		return
	}
	clear(a.custom)
	a.p.host.Command(command{Op: opA11yReset})
}

// Close stops the bridge. It is safe from any goroutine and may be called twice.
//
// Flutter: shell/platform/android/io/flutter/view/AccessibilityBridge.java AccessibilityBridge.release
func (a *accessibility) Close() {
	a.mu.Lock()
	a.closed = true
	a.mu.Unlock()
}

// perform is an action of the Java side (Native.a11yAction): it goes to the framework while the
// bridge is on. A request that cannot be understood is logged and dropped.
//
// Flutter: shell/platform/android/io/flutter/embedding/engine/systemchannels/AccessibilityChannel.java AccessibilityChannel.dispatchSemanticsAction
func (a *accessibility) perform(nodeID, action int, args []byte) {
	if a.host == nil || !a.on || a.isClosed() {
		return
	}
	ev, err := decodeAction(nodeID, action, args)
	if err != nil {
		a.p.log.Error("an accessibility action was not understood", "err", err)
		return
	}
	a.host.PerformAction(ev)
}
