//go:build darwin && !ios

package macos

// lifecycleWindows are the windows as the lifecycle sees them.
func (p *Platform) lifecycleWindows() []lifecycleWindow {
	out := make([]lifecycleWindow, 0, len(p.windows))
	for _, w := range p.sortedWindows() {
		out = append(out, lifecycleWindow{Visible: w.shown && !w.hidden && !w.closed, Minimized: w.miniaturized, Key: w.activated})
	}
	return out
}

// updateLifecycle tells the handler the lifecycle state when it changed (SPEC §29.8). It is called after
// the focus was reported, so that the framework has the focus of the view when it learns the state.
func (p *Platform) updateLifecycle() {
	h := p.handler
	if h == nil {
		return
	}
	if state, changed := p.lc.update(p.lifecycleWindows(), p.appHidden, p.ending); changed {
		h.OnLifecycle(state)
	}
}

// setAppHidden is applicationDidHide and applicationDidUnhide (Command+H, the Dock's Hide).
func (p *Platform) setAppHidden(hidden bool) {
	p.appHidden = hidden
	p.updateLifecycle()
}
