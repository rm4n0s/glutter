package uia

import "sync"

// Enable is the on/off state of the bridge (SPEC §25.11). The bridge starts off. The first WM_GETOBJECT
// that asks for the UI Automation provider of a window (the object ID UiaRootObjectId) turns it on, as
// Flutter's embedder does with its first WM_GETOBJECT: the engine then makes the semantics tree. It stays
// on for the life of the process, because a client that goes away cannot be told apart cheaply (events
// are raised only while UiaClientsAreListening). Close turns it off for good.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::GetNativeViewAccessible
type Enable struct {
	mu      sync.Mutex
	started bool
	on      bool
	closed  bool
}

// Started tells the state that the bridge has a host to ask for the tree: an ask before that is
// refused, so the window answers nothing and the client asks again.
//
// Flutter: no counterpart
func (e *Enable) Started() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.started = true
}

// Ask handles a WM_GETOBJECT with the object ID. It says whether the window answers with the provider
// (answer) and whether this ask turned the bridge on (turnedOn), in which case the caller tells the
// engine. An object ID other than UiaRootObjectId (the MSAA clients' OBJID_CLIENT, say) is not answered
// here: Windows' proxy answers it.
//
// Flutter: shell/platform/windows/flutter_windows_view.cc FlutterWindowsView::GetNativeViewAccessible
func (e *Enable) Ask(objectID int32) (answer, turnedOn bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if objectID != RootObjectID || !e.started || e.closed {
		return false, false
	}
	if !e.on {
		e.on = true
		return true, true
	}
	return true, false
}

// On says whether the bridge is on.
//
// Flutter: no counterpart
func (e *Enable) On() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.on && !e.closed
}

// Close turns the bridge off for good. It says whether it was on, so that the caller tells the engine
// to stop making the tree.
//
// Flutter: no counterpart
func (e *Enable) Close() (wasOn bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	wasOn = e.on && !e.closed
	e.closed = true
	e.on = false
	return wasOn
}
