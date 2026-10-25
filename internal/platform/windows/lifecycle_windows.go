//go:build windows

package windows

// lifecycleChanged notes that a window was shown, hidden, minimized, restored, activated or
// destroyed. The state is worked out in the next turn of the loop, once, however many windows
// changed: switching from one window to another deactivates one before it activates the other, and
// the app must not look inactive in between (Flutter does the same with a posted task).
//
// Flutter: shell/platform/windows/windows_lifecycle_manager.cc WindowsLifecycleManager::OnWindowStateEvent
func (p *Platform) lifecycleChanged() {
	if p.lifePending || p.closed {
		return
	}
	p.lifePending = true
	p.Post(p.updateLifecycle)
}

// updateLifecycle reports the state of the app when it differs from the one reported last.
//
// Flutter: shell/platform/windows/windows_lifecycle_manager.cc WindowsLifecycleManager::UpdateState
func (p *Platform) updateLifecycle() {
	p.lifePending = false
	ws := make([]lifecycleWindow, 0, len(p.windows))
	for _, w := range p.sortedWindows() {
		ws = append(ws, lifecycleWindow{
			Visible: w.shown && !w.hidden && !w.closed, Minimized: w.state.minimized, Active: w.activated,
		})
	}
	state, changed := p.life.update(ws, p.ended)
	if changed && p.handler != nil {
		p.handler.OnLifecycle(state)
	}
}

// onQueryEndSession answers WM_QUERYENDSESSION: the app does not hold the shutdown or the log-off.
//
// Flutter: no counterpart (Flutter's embedder does not handle the message)
func (p *Platform) onQueryEndSession() uintptr { return 1 }

// onEndSession handles WM_ENDSESSION. When the session ends (wParam is TRUE) the app is Detached
// at once and Run ends: Windows allows an app a few seconds, so nothing waits for a turn of the
// loop. The session cannot be cancelled by the app, so this is not an exit request. When the end
// was cancelled (wParam FALSE) nothing happens.
//
// Flutter: no counterpart (Flutter's embedder does not handle the message)
func (p *Platform) onEndSession(ending bool) {
	if !ending || p.ended {
		return
	}
	p.ended = true
	p.updateLifecycle()
	p.quit = true
}

// endLifecycle reports Detached when Run ends, for the last window that was destroyed or the
// context that was cancelled.
func (p *Platform) endLifecycle() {
	if p.ended {
		return
	}
	p.ended = true
	p.updateLifecycle()
}

// sessionEnding is whether the wParam of WM_ENDSESSION says the session is ending.
func sessionEnding(wparam uintptr) bool { return wparam != 0 }
